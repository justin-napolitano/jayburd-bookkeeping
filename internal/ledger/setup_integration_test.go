//go:build integration

package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// setupCommand contains fictional setup data and no caller-controlled identities.
func setupCommand(key string) Command {
	return Command{Actor: actor, Key: key, Request: requestID(), Body: fmt.Sprintf(`{"entity_name":"Fictional Second LLC","entity_kind":"legal","book_name":"Development book","starts_on":"2026-01-01","ends_before":"2027-01-01","functional_currency_version_id":%q,"accounts":[{"code":"1000","name":"Cash","category":"asset","normal_side":"debit","monetary":true},{"code":"3000","name":"Capital","category":"equity","normal_side":"credit","monetary":false}]}`, usd)}
}

// setupResponse decodes only generated IDs and the revision returned to the caller.
type setupResponse struct {
	Entity   string `json:"entity_id"`
	Book     string `json:"book_id"`
	Policy   string `json:"policy_id"`
	Revision string `json:"book_revision"`
	Accounts []struct {
		ID string `json:"account_id"`
	} `json:"accounts"`
}

// decodeSetup checks the durable bundle has the IDs needed for a first posting.
func decodeSetup(t *testing.T, r Result) setupResponse {
	t.Helper()
	var v setupResponse
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatal(err)
	}
	if r.Status != 201 || v.Entity == "" || v.Book == "" || v.Policy == "" || len(v.Accounts) != 2 || v.Revision != "4" {
		t.Fatal("incomplete setup result")
	}
	return v
}

// TestControlledSetup exercises real SQL roles, replay and posting from returned IDs.
func TestControlledSetup(t *testing.T) {
	c, s, db := fixtureDB(t, "controlled_setup")
	ctx := context.Background()
	runtime := connection(t, db, "synthetic_runtime")
	defer runtime.Close(ctx)
	for _, q := range []string{"SELECT ledger.enroll_principal('https://securetoken.google.com/synthetic-project','new-user')", "SET ROLE bookkeeping_identity_operator", "SELECT * FROM ledger.principal", "INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES(gen_random_uuid(),'x','x')", "UPDATE ledger.setup_receipt SET request_id=gen_random_uuid()"} {
		if _, err := runtime.Exec(ctx, q); err == nil {
			t.Fatal("runtime crossed enrollment boundary")
		}
	}
	operator := connection(t, db, "synthetic_identity_operator")
	defer operator.Close(ctx)
	enrolled := scalar(t, operator, "SELECT ledger.enroll_principal('https://securetoken.google.com/synthetic-project','fictional-owner')::text")
	if scalar(t, operator, "SELECT ledger.enroll_principal('https://securetoken.google.com/synthetic-project','fictional-owner')::text") != enrolled {
		t.Fatal("enrollment replay changed identity")
	}
	if scalar(t, c, "SELECT count(*)::text FROM audit.identity_action") != "1" {
		t.Fatal("duplicate enrollment audit")
	}
	if scalar(t, c, "SELECT database_actor FROM audit.identity_action") != "synthetic_identity_operator" {
		t.Fatal("operator attribution lost")
	}
	for _, q := range []string{"SELECT * FROM ledger.principal", "SET ROLE bookkeeping_api", "SELECT ledger.resolve_principal('synthetic','poster')", "INSERT INTO ledger.entity(entity_id,name,kind) VALUES(gen_random_uuid(),'forbidden','legal')"} {
		if _, err := operator.Exec(ctx, q); err == nil {
			t.Fatal("operator crossed setup boundary")
		}
	}
	id, err := s.ResolvePrincipal(ctx, "https://securetoken.google.com/synthetic-project", "fictional-owner")
	if err != nil || id != enrolled {
		t.Fatal("enrollment not resolvable")
	}
	for _, pair := range [][2]string{{"https://securetoken.google.com/other-project", "fictional-owner"}, {"https://securetoken.google.com/synthetic-project", "unknown"}} {
		if id, err := s.ResolvePrincipal(ctx, pair[0], pair[1]); err != nil || id != "" {
			t.Fatal("identity scope leaked")
		}
	}
	cmd := setupCommand("initial")
	cmd.Actor = enrolled
	result, err := s.Setup(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	v := decodeSetup(t, result)
	replay, err := s.Setup(ctx, cmd)
	if err != nil || !replay.Replayed || string(replay.Body) != string(result.Body) {
		t.Fatal("setup replay changed")
	}
	conflict := change(cmd, func(v map[string]any) { v["book_name"] = "Another name" })
	_, err = s.Setup(ctx, conflict)
	requireFailure(t, err, "idempotency_conflict")
	post := command("first_real_boundary")
	post.Actor = enrolled
	post.Entity = v.Entity
	post.Book = v.Book
	post = change(post, func(j map[string]any) {
		j["expected_revision"] = v.Revision
		j["policy_id"] = v.Policy
		ls := j["lines"].([]any)
		ls[0].(map[string]any)["account_id"] = v.Accounts[0].ID
		ls[1].(map[string]any)["account_id"] = v.Accounts[1].ID
	})
	if _, err = s.Post(ctx, post); err != nil {
		t.Fatal("cannot post to new bundle", err)
	}
	// Existing fixture actor must not see the newly enrolled owner's entity.
	denied := post
	denied.Actor = actor
	denied.Body = "2026-01-31"
	_, err = s.TrialBalance(ctx, denied)
	requireFailure(t, err, "not_found")
	if scalar(t, c, "SELECT count(*)::text FROM ledger.membership WHERE principal_id=$1 AND permission NOT IN ('read','post')", enrolled) != "0" {
		t.Fatal("setup granted extra permissions")
	}
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE principal_id=$1 AND permission='post'", enrolled)
	_, err = s.Setup(ctx, cmd)
	requireFailure(t, err, "permission_denied")
	sql(t, c, "UPDATE ledger.principal SET disabled_at=now() WHERE principal_id=$1", enrolled)
	if id, err := s.ResolvePrincipal(ctx, "https://securetoken.google.com/synthetic-project", "fictional-owner"); err != nil || id != "" {
		t.Fatal("disabled mapping available")
	}
	_, err = s.Setup(ctx, cmd)
	requireFailure(t, err, "not_found")
	if _, err := operator.Exec(ctx, "SELECT ledger.enroll_principal('https://securetoken.google.com/synthetic-project','fictional-owner')"); err == nil {
		t.Fatal("enrollment reactivated disabled identity")
	}
	for _, table := range []string{"audit.identity_action", "ledger.setup_receipt"} {
		if _, err := c.Exec(ctx, "DELETE FROM "+table); err == nil {
			t.Fatal("audit history mutable")
		}
	}
}

// TestSetupInvalidRollback proves malformed bundles do not leave partial entities.
func TestSetupInvalidRollback(t *testing.T) {
	c, s, _ := fixtureDB(t, "setup_invalid")
	ctx := context.Background()
	for _, tc := range []struct {
		name, code string
		modify     func(map[string]any)
	}{
		{"actor_injection", "invalid_request", func(v map[string]any) { v["actor_id"] = actor }},
		{"bad_kind", "invalid_request", func(v map[string]any) { v["entity_kind"] = "company" }},
		{"null", "invalid_request", func(v map[string]any) { v["book_name"] = nil }},
		{"blank", "invalid_request", func(v map[string]any) { v["book_name"] = "  " }},
		{"date", "invalid_request", func(v map[string]any) { v["starts_on"] = "2026-02-30" }},
		{"period", "invalid_request", func(v map[string]any) { v["ends_before"] = "2025-01-01" }},
		{"currency", "unsupported_policy", func(v map[string]any) { v["functional_currency_version_id"] = eur }},
		{"account_category", "invalid_request", func(v map[string]any) { v["accounts"].([]any)[1].(map[string]any)["category"] = "invalid" }},
		{"duplicate_code", "invalid_request", func(v map[string]any) { v["accounts"].([]any)[1].(map[string]any)["code"] = "1000" }},
		{"boolean_coercion", "invalid_request", func(v map[string]any) { v["accounts"].([]any)[0].(map[string]any)["monetary"] = "true" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := change(setupCommand(tc.name), tc.modify)
			_, err := s.Setup(ctx, cmd)
			requireFailure(t, err, tc.code)
			if scalar(t, c, "SELECT count(*)::text FROM ledger.entity") != "1" || scalar(t, c, "SELECT count(*)::text FROM ledger.setup_receipt") != "0" {
				t.Fatal("partial setup persisted")
			}
		})
	}
	for _, raw := range []string{`{"entity_name":"a","entity_name":"b"}`, strings.Replace(setupCommand("dup").Body, `"code":"1000"`, `"code":"1000","code":"1001"`, 1)} {
		cmd := setupCommand("bad_json")
		cmd.Body = raw
		_, err := s.Setup(ctx, cmd)
		requireFailure(t, err, "invalid_request")
	}
	// Inject failure after the final receipt insert; the entire bundle must roll back.
	sql(t, c, `CREATE FUNCTION public.fail_setup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic failure'; END $$; CREATE TRIGGER fail AFTER INSERT ON ledger.setup_receipt FOR EACH ROW EXECUTE FUNCTION public.fail_setup()`)
	_, err := s.Setup(ctx, setupCommand("receipt_failure"))
	requireFailure(t, err, "internal_error")
	if scalar(t, c, "SELECT count(*)::text FROM ledger.entity") != "1" {
		t.Fatal("receipt failure leaked bundle")
	}
}

// TestSetupConcurrentReplay uses the principal lock to force overlapping requests.
func TestSetupConcurrentReplay(t *testing.T) {
	c, s, _ := fixtureDB(t, "setup_concurrent")
	ctx := context.Background()
	tx, err := c.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT 1 FROM ledger.principal WHERE principal_id=$1 FOR UPDATE", actor); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make([]Result, 2)
	errs := make([]error, 2)
	wg.Add(2)
	for i := range 2 {
		go func() { defer wg.Done(); results[i], errs[i] = s.Setup(ctx, setupCommand("shared")) }()
	}
	// Querying pg_stat_activity from the same admin session is safe while tx owns lock.
	waitSetupBlocked(t, c, 2)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if string(results[0].Body) != string(results[1].Body) || results[0].Replayed == results[1].Replayed {
		t.Fatal("concurrent setup duplicated")
	}
	if scalar(t, c, "SELECT count(*)::text FROM ledger.setup_receipt") != "1" {
		t.Fatal("duplicate receipt")
	}
}

// TestSetupHTTP proves setup transport reaches committed SQL with trusted identity.
func TestSetupHTTP(t *testing.T) {
	_, s, _ := fixtureDB(t, "setup_http")
	cmd := setupCommand("http")
	r := httptest.NewRequest("POST", "/v1/setup", strings.NewReader(cmd.Body))
	r.Header.Set("Authorization", "Bearer synthetic")
	r.Header.Set("Idempotency-Key", cmd.Key)
	w := httptest.NewRecorder()
	Handler(authStub{}, s).ServeHTTP(w, r)
	if w.Code != 201 || w.Header().Get("Idempotent-Replay") != "false" {
		t.Fatalf("setup transport failed: %d %s", w.Code, w.Body.String())
	}
	// Explicitly verify lower isolation cannot call setup directly.
	tx, err := s.DB.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(context.Background(), "SELECT * FROM ledger.setup_book($1,$2,$3,$4)", actor, "low", requestID(), cmd.Body); err == nil {
		t.Fatal("setup accepted unsafe isolation")
	}
}

// waitSetupBlocked observes actual database lock waits rather than assuming overlap.
func waitSetupBlocked(t *testing.T, c *pgx.Conn, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		sql(t, c, "SELECT pg_stat_clear_snapshot()")
		var count int
		if err := c.QueryRow(context.Background(), "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND usename='synthetic_runtime' AND cardinality(pg_blocking_pids(pid))>0").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("setup callers did not block")
}

// TestSetupCommitAcknowledgementLoss proves a committed bundle is recovered by
// its receipt after a lost reply, without creating another entity or book.
func TestSetupCommitAcknowledgementLoss(t *testing.T) {
	c, s, _ := fixtureDB(t, "setup_lost_ack")
	s.DB = &lostCommitDB{Beginner: s.DB}
	cmd := setupCommand("lost_setup")
	r, err := s.Setup(context.Background(), cmd)
	requireFailure(t, err, "temporarily_unavailable")
	if r.Status != 0 || len(r.Body) != 0 {
		t.Fatal("unacknowledged setup success escaped")
	}
	r, err = s.Setup(context.Background(), cmd)
	if err != nil || !r.Replayed {
		t.Fatal("setup receipt did not recover lost reply", err)
	}
	decodeSetup(t, r)
	if scalar(t, c, "SELECT count(*)::text FROM ledger.entity") != "2" || scalar(t, c, "SELECT count(*)::text FROM ledger.setup_receipt") != "1" {
		t.Fatal("lost reply duplicated setup")
	}
}

// TestSetupRevocationRace forces disabling to commit while setup waits on the
// principal row; whole-transaction retry must observe the disabled identity.
func TestSetupRevocationRace(t *testing.T) {
	c, s, _ := fixtureDB(t, "setup_revocation")
	ctx := context.Background()
	tx, err := c.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "UPDATE ledger.principal SET disabled_at=now() WHERE principal_id=$1", actor); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := s.Setup(ctx, setupCommand("revoked_setup")); done <- err }()
	waitSetupBlocked(t, c, 1)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	requireFailure(t, <-done, "not_found")
	if scalar(t, c, "SELECT count(*)::text FROM ledger.entity") != "1" {
		t.Fatal("disabled principal created entity")
	}
}
