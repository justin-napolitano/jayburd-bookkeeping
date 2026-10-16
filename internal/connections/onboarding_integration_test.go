//go:build integration

package connections

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

// creatorFixture creates a real book through setup_book, with no connection operator stage.
func creatorFixture(t *testing.T) (*Service, *pgx.Conn, string, Scope, *fakeProvider) {
	t.Helper()
	admin, db := sqlFixture(t)
	raw := proofRaw(map[string]any{"entity_name": "Fictional creator", "entity_kind": "personal", "book_name": "Self service", "starts_on": "2026-01-01", "ends_before": "2027-01-01", "functional_currency_version_id": currency, "accounts": []any{
		map[string]any{"code": "1000", "name": "Cash", "category": "asset", "normal_side": "debit", "monetary": true},
		map[string]any{"code": "3000", "name": "Equity", "category": "equity", "normal_side": "credit", "monetary": false},
	}})
	result := proofObject(t, proofCall(t, db, "synthetic_operator", "SELECT body::text FROM ledger.setup_book($1,'creator-book',gen_random_uuid(),$2)", actor, raw))
	scope := Scope{Actor: actor, Entity: result["entity_id"].(string), Book: result["book_id"].(string)}
	runtime := proofConnect(t, db, "synthetic_runtime")
	t.Cleanup(func() { runtime.Close(context.Background()) })
	_, _, provider, store := fixture()
	s, err := NewForBookCreators(runtime, provider, store, WorkerLogins{Sync: "synthetic_sync", Publisher: "synthetic_publisher"})
	if err != nil {
		t.Fatal(err)
	}
	return s, admin, db, scope, provider
}

// TestCreatorFirstUse proves a real API login reaches Link without operator book grants.
func TestCreatorFirstUse(t *testing.T) {
	s, admin, _, sc, provider := creatorFixture(t)
	ctx := context.Background()
	if _, err := s.List(ctx, sc); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Attempts(ctx, sc, 100, nil); err != nil {
		t.Fatal(err)
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM integration.connection_worker") != "0" || len(provider.calls) != 0 {
		t.Fatal("discovery provisioned or contacted provider")
	}
	in := BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}}
	if _, err := s.Begin(ctx, sc, "creator-begin", in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Begin(ctx, sc, "creator-begin", in); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"integration.connection_worker", "integration.sync_delivery_binding", "integration.connection_attempt"} {
		if proofValue(t, admin, "SELECT count(*)::text FROM "+table) != "1" {
			t.Fatal("provisioning or replay count", table)
		}
	}
	if count(provider.calls, "link") != 1 || proofValue(t, admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("duplicate handoff or financial mutation")
	}
}

// TestCreatorDenied proves creator evidence alone never overrides revocation or disabled state.
func TestCreatorDenied(t *testing.T) {
	for _, scenario := range []string{"foreign", "revoked", "disabled", "book_disabled", "worker_missing"} {
		t.Run(scenario, func(t *testing.T) {
			s, admin, _, sc, provider := creatorFixture(t)
			switch scenario {
			case "foreign":
				proofExec(t, admin, "INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$2,'post',NULL)", sc.Entity, reader)
				sc.Actor = reader
			case "revoked":
				proofExec(t, admin, "INSERT INTO ledger.membership VALUES($1,$2,'connect',clock_timestamp())", sc.Entity, actor)
			case "disabled":
				proofExec(t, admin, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", actor)
			case "book_disabled":
				proofExec(t, admin, "UPDATE ledger.book SET disabled_at=clock_timestamp() WHERE book_id=$1", sc.Book)
			case "worker_missing":
				s.creatorLogins.Sync = "missing_worker"
			}
			if _, err := s.List(context.Background(), sc); err == nil {
				t.Fatal("discovery authorized denied creator")
			}
			if _, err := s.Begin(context.Background(), sc, "denied", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}}); err == nil {
				t.Fatal("onboarding authorized denied creator")
			}
			if len(provider.calls) != 0 || proofValue(t, admin, "SELECT count(*)::text FROM integration.connection_worker") != "0" {
				t.Fatal("denial left effects")
			}
		})
	}
}

// TestCreatorRollback proves an invalid attempt rolls back its preceding book preparation.
func TestCreatorRollback(t *testing.T) {
	s, admin, _, sc, provider := creatorFixture(t)
	_, err := s.Begin(context.Background(), sc, "rollback", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{book}})
	if err == nil {
		t.Fatal("invalid new attempt accepted")
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM integration.connection_worker") != "0" || proofValue(t, admin, "SELECT count(*)::text FROM ledger.principal WHERE issuer='urn:jayburd:service:book-acquisition'") != "0" || len(provider.calls) != 0 {
		t.Fatal("failed attempt left grants or provider effects")
	}
}

// TestCreatorRuntimeBoundary proves the API can only invoke the guarded public wrappers.
func TestCreatorRuntimeBoundary(t *testing.T) {
	_, _, db, sc, _ := creatorFixture(t)
	_, err := proofInvoke(t, db, "synthetic_runtime", "SELECT integration.creator_connection_setup($1,$2,$3,'synthetic_sync','synthetic_publisher',true)::text", sc.Actor, sc.Entity, sc.Book)
	proofCode(t, err, "42501")
	if proofCall(t, db, "synthetic_runtime", "SELECT (pg_has_role(current_user,'bookkeeping_owner','MEMBER'))::text") != "false" {
		t.Fatal("API inherited owner authority")
	}
}

// TestCreatorConcurrent proves two retries create one principal, mandate and durable attempt.
func TestCreatorConcurrent(t *testing.T) {
	_, admin, db, sc, _ := creatorFixture(t)
	const query = "SELECT integration.creator_begin_connection($1,$2,$3,'concurrent',$4,'synthetic_sync','synthetic_publisher')::text"
	raw := proofRaw(BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	results := make(chan error, 2)
	var start sync.WaitGroup
	start.Add(1)
	for i := 0; i < 2; i++ {
		go func() {
			start.Wait()
			_, err := proofInvoke(t, db, "synthetic_runtime", query, sc.Actor, sc.Entity, sc.Book, raw)
			results <- err
		}()
	}
	start.Done()
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM integration.connection_attempt") != "1" || proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_delivery_binding") != "1" || proofValue(t, admin, "SELECT count(*)::text FROM ledger.principal WHERE issuer='urn:jayburd:service:book-acquisition'") != "1" {
		t.Fatal("concurrent provisioning duplicated state")
	}
}

// TestCreatorNoReactivation refuses to repair a mandate revoked after first registration.
func TestCreatorNoReactivation(t *testing.T) {
	for _, scenario := range []string{"grant", "worker", "delivery", "service"} {
		t.Run(scenario, func(t *testing.T) {
			s, admin, _, sc, provider := creatorFixture(t)
			in := BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}}
			if _, err := s.Begin(context.Background(), sc, "first", in); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "grant":
				proofExec(t, admin, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND principal_id=$2 AND permission='connect'", sc.Entity, sc.Actor)
			case "worker":
				proofExec(t, admin, "UPDATE integration.connection_worker SET enabled=false WHERE book_id=$1", sc.Book)
			case "delivery":
				proofExec(t, admin, "UPDATE integration.sync_delivery_binding SET enabled=false,binding_revision=binding_revision+1 WHERE book_id=$1", sc.Book)
			case "service":
				proofExec(t, admin, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=(SELECT service_principal FROM integration.connection_worker WHERE book_id=$1)", sc.Book)
			}
			before := len(provider.calls)
			if _, err := s.Begin(context.Background(), sc, "first", in); err == nil {
				t.Fatal("revoked configuration reactivated")
			}
			if len(provider.calls) != before {
				t.Fatal("provider contacted after revocation")
			}
		})
	}
}
