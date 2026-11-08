//go:build integration

package collectionlabels

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionactivation"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// activationIntent is independently authored metadata, not a user identity or
// provider-derived manifest. Its one source already exists in legacyFixture.
func activationIntent(w string, item map[string]any) map[string]any {
	return map[string]any{
		"profile": "private-sandbox-activation-v1", "project": "jayburd-bookkeeping-dev", "plan_id": evidence, "actor_id": actor,
		"workspace_id": w, "entity_id": entityID, "book_id": bookID, "sync_item_id": item["sync_item_id"], "expected_generation": "1",
		"idempotency_key": "reviewed-activation453", "evidence_ref": evidence, "expires_at": time.Now().Add(30 * time.Minute).UTC().Format("2006-01-02T15:04:05Z"),
		"sources": []map[string]string{{"source_account_id": sourceID, "display_name": "Reviewed fictional checking"}},
	}
}

func activationPlan(t *testing.T, v map[string]any) *collectionactivation.Plan {
	t.Helper()
	body := []byte(raw(v))
	sum := sha256.Sum256(body)
	envelope, _ := json.Marshal(struct {
		Hash string          `json:"sha256"`
		Plan json.RawMessage `json:"plan"`
	}{hex.EncodeToString(sum[:]), body})
	path := filepath.Join(t.TempDir(), "approved.json")
	if err := os.WriteFile(path, envelope, 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := collectionactivation.Load(path, time.Now())
	if err != nil {
		t.Fatal("synthetic approved plan rejected")
	}
	return plan
}

func activationPool(t *testing.T, db string) *pgxpool.Pool {
	t.Helper()
	p, e := pgxpool.New(context.Background(), "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5 pool_max_conns=2")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(p.Close)
	return p
}

func activationHandler(t *testing.T, db ledger.Beginner, v map[string]any) http.Handler {
	t.Helper()
	s, e := collectionactivation.NewService(db, activationPlan(t, v))
	if e != nil {
		t.Fatal(e)
	}
	return collectionactivation.Handler(labelHTTPAuth{}, s)
}

func activationRequest(t *testing.T, h http.Handler, token string, want int) string {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/v1/collection-activation/adopt", strings.NewReader(`{"plan_id":"`+evidence+`"}`))
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("activation wanted%d got%d: %s", want, w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("private response cacheable")
	}
	if strings.Contains(w.Body.String(), "legacy-item") || strings.Contains(w.Body.String(), "original-cursor") || strings.Contains(w.Body.String(), "Fictional original journal") {
		t.Fatal("protected source details escaped")
	}
	return w.Body.String()
}

// TestActivationHTTPPostgres proves the real HTTP/service/API-role path adopts
// exactly once, while source/cursor/credential/journal anchors remain untouched.
func TestActivationHTTPPostgres(t *testing.T) {
	owner, db, item := legacyFixture(t)
	w := provision(t, db, actor)
	before := legacyHash(t, owner)
	p := activationIntent(w, item)
	pool := activationPool(t, db)
	h := activationHandler(t, pool, p)
	activationRequest(t, h, "synthetic-label-other", 404)
	activationRequest(t, h, "synthetic-unenrolled", 401)
	first := activationRequest(t, h, "synthetic-label-owner", 200)
	if replay := activationRequest(t, h, "synthetic-label-owner", 200); replay != first {
		t.Fatal("exact replay changed")
	}
	if legacyHash(t, owner) != before {
		t.Fatal("immutable source/cursor/journal anchors changed")
	}
	if value(t, owner, "SELECT count(*)::text FROM integration.collection_receipt WHERE operation='adopt'") != "1" || value(t, owner, "SELECT state||':'||generation FROM integration.sync_item") != "paused:2" {
		t.Fatal("adoption not exactly once/fenced")
	}
	if value(t, owner, "SELECT has_function_privilege('synthetic_runtime','banking.provision_collection_owner(uuid,text,text)','EXECUTE')::text") != "false" {
		t.Fatal("runtime can selfprovision")
	}
	// A changed approved tuple with the same SQL key cannot replace prior intent.
	p["sources"] = []map[string]string{{"source_account_id": sourceID, "display_name": "Changed review"}}
	activationRequest(t, activationHandler(t, pool, p), "synthetic-label-owner", 409)
}

// TestActivationHTTPCurrentAuthority checks both first execution and receipt
// replay against present database authority, independently of auth test doubles.
func TestActivationHTTPCurrentAuthority(t *testing.T) {
	for _, stage := range []string{"before", "replay"} {
		for _, scope := range []string{"principal", "workspace", "book", "source"} {
			t.Run(stage+"_"+scope, func(t *testing.T) {
				owner, db, item := legacyFixture(t)
				w := provision(t, db, actor)
				pool := activationPool(t, db)
				h := activationHandler(t, pool, activationIntent(w, item))
				if stage == "replay" {
					activationRequest(t, h, "synthetic-label-owner", 200)
				}
				switch scope {
				case "principal":
					exec(t, owner, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", actor)
				case "workspace":
					_, e := call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,'revoke-activation',$2)", actor, raw(map[string]any{"expected_revision": "1", "capabilities": []string{}, "evidence_ref": evidence}))
					if e != nil {
						t.Fatal(e)
					}
				case "book":
					exec(t, owner, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND principal_id=$2 AND permission='read'", entityID, actor)
				case "source":
					exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3 AND principal_id=$4 AND capability='import'", entityID, bookID, sourceID, actor)
				}
				count := value(t, owner, "SELECT count(*)::text FROM integration.collection_receipt WHERE operation='adopt'")
				activationRequest(t, h, "synthetic-label-owner", 404)
				if value(t, owner, "SELECT count(*)::text FROM integration.collection_receipt WHERE operation='adopt'") != count {
					t.Fatal("denied authority added effect")
				}
			})
		}
	}
}

// TestActivationHTTPPreconditions proves generation, exact reviewed source set,
// pending connection/acquisition and late SQL failure leave no adoption effects.
func TestActivationHTTPPreconditions(t *testing.T) {
	for _, scenario := range []string{"generation", "incomplete", "connection", "staging", "late_failure"} {
		t.Run(scenario, func(t *testing.T) {
			owner, db, item := legacyFixture(t)
			w := provision(t, db, actor)
			p := activationIntent(w, item)
			wanted := 409
			switch scenario {
			case "generation":
				p["expected_generation"] = "2"
			case "incomplete":
				// A second real active stream is omitted from the reviewed plan.
				sid := "00000000-0000-4000-8000-000000000020"
				nid := "00000000-0000-4000-8000-000000000021"
				aid := "00000000-0000-4000-8000-000000000022"
				exec(t, owner, `INSERT INTO ledger.account VALUES($1,$2,$3,'1001','Fictional second checking','asset','debit',true,'2026-01-01',NULL)`, entityID, bookID, aid)
				exec(t, owner, `INSERT INTO banking.source_account(entity_id,book_id,source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by)VALUES($1,$2,$3,$4,$5,'2026-01-05',$6,$7)`, entityID, bookID, sid, aid, currency, evidence, actor)
				exec(t, owner, `INSERT INTO banking.source_namespace(entity_id,book_id,namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by)VALUES($1,$2,$3,$4,'jayburd-synthetic-statement-csv/1',$3,'FICTIONAL-02',false,$5)`, entityID, bookID, nid, sid, actor)
				exec(t, owner, `INSERT INTO integration.sync_stream(entity_id,book_id,sync_item_id,source_account_id,namespace_id,account_id,worker_login,service_principal)VALUES($1,$2,$3,$4,$5,'second-account','synthetic_sync',$6)`, entityID, bookID, item["sync_item_id"], sid, nid, other)
			case "connection":
				exec(t, owner, `INSERT INTO integration.connection_attempt(entity_id,book_id,actor_id,idempotency_key,mode,state,body,worker_revision,sync_item_id)VALUES($1,$2,$3,'pending-adoption','reconnect','link_pending','{}',1,$4)`, entityID, bookID, actor, item["sync_item_id"])
			case "staging":
				exec(t, owner, `INSERT INTO integration.sync_set(entity_id,book_id,stream_id,claim_key,item_generation,binding_revision,target_generation,fence)SELECT entity_id,book_id,stream_id,$1,1,1,1,1 FROM integration.sync_stream`, evidence)
			case "late_failure":
				// Fixture-only failure after the Item fence/source alias insert, before receipt.
				exec(t, owner, `CREATE FUNCTION public.synthetic_activation_reject()RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF NEW.operation='adopt'THEN RAISE EXCEPTION USING ERRCODE='P2804',MESSAGE='synthetic_adoption_late_failure';END IF;RETURN NEW;END$$;CREATE TRIGGER synthetic_activation_reject BEFORE INSERT ON integration.collection_receipt FOR EACH ROW EXECUTE FUNCTION public.synthetic_activation_reject()`)
			}
			before := legacyHash(t, owner)
			pool := activationPool(t, db)
			activationRequest(t, activationHandler(t, pool, p), "synthetic-label-owner", wanted)
			if legacyHash(t, owner) != before || value(t, owner, "SELECT count(*)::text FROM banking.collection_source") != "0" || value(t, owner, "SELECT count(*)::text FROM integration.collection_receipt WHERE operation='adopt'") != "0" || value(t, owner, "SELECT state||':'||generation FROM integration.sync_item") != "active:1" {
				t.Fatal("denied adoption changed history or authority")
			}
		})
	}
}

// activationLostDB loses one acknowledgement AFTER a real commit, never its
// result inside PostgreSQL. The following HTTP request must replay exact intent.
type activationLostDB struct {
	db   ledger.Beginner
	lose atomic.Bool
}

func (d *activationLostDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, e := d.db.BeginTx(ctx, o)
	if e != nil {
		return nil, e
	}
	return &activationLostTx{Tx: tx, d: d}, nil
}

type activationLostTx struct {
	pgx.Tx
	d *activationLostDB
}

func (t *activationLostTx) Commit(ctx context.Context) error {
	if e := t.Tx.Commit(ctx); e != nil {
		return e
	}
	if t.d.lose.CompareAndSwap(true, false) {
		return errors.New("synthetic lost commit acknowledgement")
	}
	return nil
}

func TestActivationHTTPLostReplyPostgres(t *testing.T) {
	owner, db, item := legacyFixture(t)
	w := provision(t, db, actor)
	before := legacyHash(t, owner)
	p := activationIntent(w, item)
	pool := activationPool(t, db)
	d := &activationLostDB{db: pool}
	d.lose.Store(true)
	h := activationHandler(t, d, p)
	activationRequest(t, h, "synthetic-label-owner", 503)
	if value(t, owner, "SELECT count(*)::text FROM integration.collection_receipt WHERE operation='adopt'") != "1" {
		t.Fatal("synthetic commit did not persist")
	}
	replay := activationRequest(t, h, "synthetic-label-owner", 200)
	if !strings.Contains(replay, `"item_generation":"2"`) || legacyHash(t, owner) != before {
		t.Fatal("unknown outcome replay corrupted anchors")
	}
	exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3 AND principal_id=$4 AND capability='read'", entityID, bookID, sourceID, actor)
	activationRequest(t, h, "synthetic-label-owner", 404)
	if value(t, owner, "SELECT count(*)::text FROM integration.collection_receipt WHERE operation='adopt'") != "1" {
		t.Fatal("replay added adoption")
	}
}

// TestActivationIdentityClaimRaces preserves the migration42 canonical invariant
// under both real waiting schedules while the new transport remains restricted.
func TestActivationIdentityClaimRaces(t *testing.T) {
	owner, db, _ := legacyFixture(t)
	w := provision(t, db, actor)
	legacySQL := `INSERT INTO integration.sync_item(entity_id,book_id,provider,environment,item_id,credential_ref,credential_version,consent_principal,consent_evidence_ref,consent_expires_at,synthetic)VALUES($1,$2,'plaid','sandbox',$3,$4,1,$5,$6,'2099-01-01',true)`
	manifest := func(item string) string {
		return raw(map[string]any{"provider": "plaid", "environment": "sandbox", "item_id": item, "credential_ref": credential, "credential_version": "1", "consent_evidence_ref": evidence, "consent_expires_at": "2099-01-01T00:00:00Z", "accounts": []map[string]any{{"account_id": "race-account", "display_name": "Fictional race account"}}})
	}
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	observer := connect(t, db, "postgres")
	defer observer.Close(context.Background())
	done := make(chan error, 1)
	legacy, e := owner.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer legacy.Rollback(context.Background())
	if _, e = legacy.Exec(context.Background(), legacySQL, entityID, bookID, "activation-race-legacy", credential, actor, evidence); e != nil {
		t.Fatal(e)
	}
	go func() {
		tx, er := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
		if er != nil {
			done <- er
			return
		}
		defer tx.Rollback(context.Background())
		var out []byte
		er = tx.QueryRow(context.Background(), "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "activation-race-legacy", manifest("activation-race-legacy")).Scan(&out)
		if er == nil {
			er = tx.Commit(context.Background())
		}
		_ = tx.Rollback(context.Background())
		done <- er
	}()
	waitBlocked(t, observer, runtime.PgConn().PID())
	if e = legacy.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	e = <-done
	var pg *pgconn.PgError
	if errors.As(e, &pg) && pg.Code == "40001" {
		_, e = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "activation-race-legacy", manifest("activation-race-legacy"))
	}
	code(t, e, "P2802")
	native, e := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer native.Rollback(context.Background())
	var out []byte
	if e = native.QueryRow(context.Background(), "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "activation-race-native", manifest("activation-race-native")).Scan(&out); e != nil {
		t.Fatal(e)
	}
	contender := connect(t, db, "postgres")
	defer contender.Close(context.Background())
	go func() {
		_, er := contender.Exec(context.Background(), legacySQL, entityID, bookID, "activation-race-native", credential, actor, evidence)
		done <- er
	}()
	waitBlocked(t, observer, contender.PgConn().PID())
	if e = native.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	code(t, <-done, "P2802")
	if value(t, owner, "SELECT count(*)::text FROM integration.collection_item WHERE item_id LIKE 'activation-race-%'") != "2" || value(t, owner, "SELECT count(*)::text FROM integration.sync_item WHERE item_id LIKE 'activation-race-%'") != "1" {
		t.Fatal("duplicate canonical authority survived")
	}
}
