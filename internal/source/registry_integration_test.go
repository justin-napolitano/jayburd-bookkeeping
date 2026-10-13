//go:build integration

package source

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proof = flag.Bool("source-proof", false, "Use only the disposable internal registry database")

const entity = "00000000-0000-4000-8000-000000000002"
const book = "00000000-0000-4000-8000-000000000003"
const actor = "00000000-0000-4000-8000-000000000001"
const reader = "00000000-0000-4000-8000-000000000020"
const other = "00000000-0000-4000-8000-000000000021"
const account = "00000000-0000-4000-8000-000000000011"
const currency = "00000000-0000-4000-8000-000000000005"

// connect uses only the runner's private synthetic trust network, never an inherited DSN.
func connect(t testing.TB, db, user string) *pgx.Conn {
	t.Helper()
	if !*proof {
		t.Skip("use scripts/test-source-registry.sh")
	}
	c, e := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=5432 user=%s dbname=%s sslmode=disable connect_timeout=5", user, db))
	if e != nil {
		t.Fatal(e)
	}
	return c
}

// exec is owner-only fixture setup or inspection, never product privilege.
func exec(t testing.TB, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, e := c.Exec(context.Background(), q, args...); e != nil {
		t.Fatal(e)
	}
}

// value reads fixed aggregate proof facts with the fixture owner.
func value(t testing.TB, c *pgx.Conn, q string, args ...any) string {
	t.Helper()
	var s string
	if e := c.QueryRow(context.Background(), q, args...).Scan(&s); e != nil {
		t.Fatal(e)
	}
	return s
}

// requireCode compares SQLSTATE without relying on potentially sensitive diagnostics.
func requireCode(t testing.TB, e error, want string) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(e, &p) || p.Code != want {
		t.Fatalf("want %s got %v", want, e)
	}
}

// fixture installs only fictional ownership and independently chosen currency metadata.
func fixture(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	db := "source_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE source_template")
	c := connect(t, db, "postgres")
	t.Cleanup(func() { c.Close(context.Background()) })
	exec(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','approver'),($2,'synthetic','reader'),($3,'synthetic','other')`, actor, reader, other)
	exec(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional LLC','legal')`, entity)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$3,'read',NULL),($1,$4,'read',NULL)`, entity, actor, reader, other)
	exec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Synthetic book')`, entity, book)
	exec(t, c, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000006','JPY','392',0,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000007','KWD','414',3,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000008','XXX','999',NULL,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000009','EUR','978',2,'2020-01-01','2025-01-01','synthetic')`, currency)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'1000','Fictional cash','asset','debit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000012','2000','Fictional liability','liability','credit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000013','1001','Fictional nonmonetary','asset','debit',false,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000014','1002','Fictional contra','asset','credit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000015','1003','Fictional second deposit','asset','debit',true,'2026-01-01',NULL)`, entity, book, account)
	return c, db
}

// manifest is hand-authored profile input independent of SQL normalization.
func manifest() map[string]any {
	return map[string]any{"ledger_account_id": account, "currency_version_id": currency, "binding_date": "2026-01-05", "binding_evidence_ref": "00000000-0000-4000-8000-000000000030", "reader_principal_id": reader, "namespace": map[string]any{"profile": "jayburd-synthetic-statement-csv/1", "synthetic_dataset_id": "00000000-0000-4000-8000-000000000031", "source_account_ref": "SYNTH-CASH-01"}}
}

// raw encodes synthetic exact values without floating-point amounts.
func raw(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return string(b)
}

// invoke uses a real login, bounded transactions and full serialization retry.
func invoke(t testing.TB, db, user, q string, args ...any) (string, error) {
	t.Helper()
	c := connect(t, db, user)
	defer c.Close(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	var result string
	var err error
	for i := 0; i < 3; i++ {
		tx, e := c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			return "", e
		}
		_, e = tx.Exec(ctx, "SET LOCAL statement_timeout='5s'; SET LOCAL lock_timeout='1s'")
		if e == nil && user == "synthetic_operator" {
			_, e = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner")
		}
		if e == nil {
			e = tx.QueryRow(ctx, q, args...).Scan(&result)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		_ = tx.Rollback(context.Background())
		if e == nil {
			return result, nil
		}
		err = e
		var p *pgconn.PgError
		if !errors.As(e, &p) || (p.Code != "40001" && p.Code != "40P01") {
			break
		}
	}
	return "", err
}

// call requires a committed result for successful synthetic operations.
func call(t testing.TB, db, user, q string, args ...any) string {
	t.Helper()
	s, e := invoke(t, db, user, q, args...)
	if e != nil {
		t.Fatal(e)
	}
	return s
}

// enroll returns exact server-generated identity from the committed owner receipt.
func enroll(t testing.TB, db string) (string, string) {
	t.Helper()
	s := call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'enroll',$4)::text", actor, entity, book, raw(manifest()))
	var v map[string]string
	if e := json.Unmarshal([]byte(s), &v); e != nil {
		t.Fatal(e)
	}
	return v["source_account_id"], s
}

// read scopes every runtime retrieval through both permission domains.
func read(t testing.TB, db, id, principal string) (string, error) {
	return invoke(t, db, "synthetic_runtime", "SELECT banking.get_source($1,$2,$3,$4)::text", principal, entity, book, id)
}

// mutate routes only compiled owner operations; tests never grant runtime write access.
func mutate(t testing.TB, db, id, op, key string, v any) (string, error) {
	return invoke(t, db, "synthetic_operator", "SELECT banking."+op+"($1,$2,$3,$4,$5,$6)::text", actor, entity, book, id, key, raw(v))
}

// TestSourceRegistryEnrollment proves receipt equality, explicit identity and no financial effects.
func TestSourceRegistryEnrollment(t *testing.T) {
	c, db := fixture(t)
	before := value(t, c, "SELECT revision::text FROM ledger.book")
	id, result := enroll(t, db)
	again := call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'enroll',$4)::text", actor, entity, book, raw(manifest()))
	if again != result {
		t.Fatal("replay changed result")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.source_registry_receipt") != "1" {
		t.Fatal("duplicate audit")
	}
	v := manifest()
	v["binding_date"] = "2026-01-06"
	_, e := invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'enroll',$4)::text", actor, entity, book, raw(v))
	requireCode(t, e, "P2302")
	_, e = invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'new-key',$4)::text", actor, entity, book, raw(manifest()))
	requireCode(t, e, "P2302")
	got, e := read(t, db, id, reader)
	if e != nil {
		t.Fatal(e)
	}
	var view map[string]any
	_ = json.Unmarshal([]byte(got), &view)
	if view["currency"] != "USD" || view["binding_revision"] != "1" || view["source_revision"] != "0" || view["profile"] != "source-registry-v1" || strings.Contains(got, "synthetic_operator") || strings.Contains(got, "binding_evidence_ref") {
		t.Fatal("wrong read projection")
	}
	if value(t, c, "SELECT operator_login::text||':'||approved_by::text FROM integration.source_registry_receipt") != "synthetic_operator:"+actor {
		t.Fatal("attestation lost operator")
	}
	if value(t, c, "SELECT revision::text FROM ledger.book") != before || value(t, c, "SELECT (SELECT count(*) FROM ledger.journal_entry)::text||':'||(SELECT count(*) FROM integration.outbox_message)::text") != "0:0" {
		t.Fatal("registry changed books")
	}
	// Replaying enrollment does not restore a separately revoked source grant.
	_, e = mutate(t, db, id, "set_source_permission", "revoke", map[string]any{"expected_binding_revision": "1", "principal_id": reader, "enabled": false})
	if e != nil {
		t.Fatal(e)
	}
	call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'enroll',$4)::text", actor, entity, book, raw(manifest()))
	_, e = read(t, db, id, reader)
	requireCode(t, e, "P2301")
}

// TestSourceRegistryMapping rejects unsupported mappings atomically and preserves non-USD metadata.
func TestSourceRegistryMapping(t *testing.T) {
	cases := []struct{ name, key, value, code string }{{"liability", "ledger_account_id", "00000000-0000-4000-8000-000000000012", "P2303"}, {"nonmonetary", "ledger_account_id", "00000000-0000-4000-8000-000000000013", "P2303"}, {"contra", "ledger_account_id", "00000000-0000-4000-8000-000000000014", "P2303"}, {"unknownaccount", "ledger_account_id", other, "P2303"}, {"beforeactive", "binding_date", "2025-12-31", "P2303"}, {"baddate", "binding_date", "2026-02-30", "P2300"}, {"unknowncurrency", "currency_version_id", other, "P2303"}, {"nom inor", "currency_version_id", "00000000-0000-4000-8000-000000000008", "P2303"}, {"expiredcurrency", "currency_version_id", "00000000-0000-4000-8000-000000000009", "P2303"}}
	for _, tc := range cases {
		t.Run(strings.ReplaceAll(tc.name, " ", ""), func(t *testing.T) {
			c, db := fixture(t)
			v := manifest()
			v[tc.key] = tc.value
			_, e := invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'bad',$4)::text", actor, entity, book, raw(v))
			requireCode(t, e, tc.code)
			if value(t, c, "SELECT count(*)::text FROM banking.source_account") != "0" || value(t, c, "SELECT count(*)::text FROM integration.source_registry_receipt") != "0" {
				t.Fatal("partial enrollment")
			}
		})
	}
	for _, tc := range []struct {
		id, code string
		minor    int
	}{{"00000000-0000-4000-8000-000000000006", "JPY", 0}, {"00000000-0000-4000-8000-000000000007", "KWD", 3}} {
		t.Run(tc.code, func(t *testing.T) {
			c, db := fixture(t)
			v := manifest()
			v["currency_version_id"] = tc.id
			call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'foreign',$4)::text", actor, entity, book, raw(v))
			id := value(t, c, "SELECT source_account_id::text FROM banking.source_account")
			s, e := read(t, db, id, reader)
			if e != nil {
				t.Fatal(e)
			}
			var got struct {
				Currency string `json:"currency"`
				Minor    int    `json:"minor_units"`
			}
			_ = json.Unmarshal([]byte(s), &got)
			if got.Currency != tc.code || got.Minor != tc.minor {
				t.Fatal("currency metadata lost")
			}
		})
	}
}

// TestSourceRegistryAuthority checks current membership, source grant and scope before data release or replay.
func TestSourceRegistryAuthority(t *testing.T) {
	for _, condition := range []string{"missinggrant", "revokedgrant", "revokedmember", "disabledreader", "disabledentity", "disabledbook", "foreignsource", "foreignbook", "replaygrantee", "replayapprover"} {
		t.Run(condition, func(t *testing.T) {
			c, db := fixture(t)
			id, _ := enroll(t, db)
			who := reader
			switch condition {
			case "missinggrant":
				who = other
			case "revokedgrant":
				exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp()")
			case "revokedmember", "replaygrantee":
				exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1", reader)
			case "disabledreader":
				exec(t, c, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", reader)
			case "disabledentity":
				exec(t, c, "UPDATE ledger.entity SET disabled_at=clock_timestamp()")
			case "disabledbook":
				exec(t, c, "UPDATE ledger.book SET disabled_at=clock_timestamp()")
			case "foreignsource":
				id = other
			case "foreignbook":
				_, e := invoke(t, db, "synthetic_runtime", "SELECT banking.get_source($1,$2,$3,$4)::text", reader, entity, other, id)
				requireCode(t, e, "P2301")
				return
			case "replayapprover":
				exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1", actor)
			}
			if strings.HasPrefix(condition, "replay") {
				_, e := invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'enroll',$4)::text", actor, entity, book, raw(manifest()))
				requireCode(t, e, "P2301")
			} else {
				_, e := read(t, db, id, who)
				requireCode(t, e, "P2301")
			}
		})
	}
}

// TestSourceRegistryLifecycle proves CAS transitions, terminal retirement and preserved namespace identity.
func TestSourceRegistryLifecycle(t *testing.T) {
	c, db := fixture(t)
	id, _ := enroll(t, db)
	for i, state := range []string{"paused", "active", "retired"} {
		_, e := mutate(t, db, id, "set_source_state", state, map[string]any{"expected_binding_revision": fmt.Sprint(i + 1), "state": state})
		if e != nil {
			t.Fatal(e)
		}
		got, e := read(t, db, id, reader)
		if e != nil || !strings.Contains(got, `"state": "`+state+`"`) {
			t.Fatal("state not reviewable", e)
		}
	}
	_, e := mutate(t, db, id, "set_source_state", "reactivate", map[string]any{"expected_binding_revision": "4", "state": "active"})
	requireCode(t, e, "P2302")
	if value(t, c, "SELECT state FROM banking.source_namespace") != "retired" {
		t.Fatal("namespace not retired")
	}
	_, e = invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'reuse',$4)::text", actor, entity, book, raw(manifest()))
	requireCode(t, e, "P2302")
	for _, q := range []string{"UPDATE banking.source_account SET ledger_account_id='00000000-0000-4000-8000-000000000015'", "DELETE FROM banking.source_account", "UPDATE banking.source_namespace SET source_account_ref='REPLACED'", "DELETE FROM banking.source_namespace", "DELETE FROM integration.source_registry_receipt"} {
		_, e = c.Exec(context.Background(), q)
		var p *pgconn.PgError
		if !errors.As(e, &p) || (p.Code != "P2302" && p.Code != "P2108") {
			t.Fatal("immutable history changed", e)
		}
	}
}

// TestSourceRegistryNamespaces preserves independent adapter evidence and bounds per-source response size.
func TestSourceRegistryNamespaces(t *testing.T) {
	c, db := fixture(t)
	id, _ := enroll(t, db)
	v := manifest()
	v["ledger_account_id"] = "00000000-0000-4000-8000-000000000015"
	_, e := invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'duplicate-namespace',$4)::text", actor, entity, book, raw(v))
	requireCode(t, e, "P2302")
	for i := 1; i < 25; i++ {
		ns := manifest()["namespace"].(map[string]any)
		ns["source_account_ref"] = fmt.Sprintf("OTHER-%02d", i)
		_, e = mutate(t, db, id, "register_source_namespace", fmt.Sprint(i), map[string]any{"expected_binding_revision": fmt.Sprint(i), "namespace": ns})
		if e != nil {
			t.Fatal(e)
		}
	}
	ns := manifest()["namespace"].(map[string]any)
	ns["source_account_ref"] = "LIMIT"
	_, e = mutate(t, db, id, "register_source_namespace", "limit", map[string]any{"expected_binding_revision": "25", "namespace": ns})
	requireCode(t, e, "P2304")
	if value(t, c, "SELECT count(*)::text||':'||count(*) FILTER(WHERE overlap_review_required)::text FROM banking.source_namespace") != "25:24" {
		t.Fatal("namespace provenance/cap")
	}
	s, e := read(t, db, id, reader)
	if e != nil || len(s) > 65536 {
		t.Fatal("bounded read", e)
	}
}

// TestSourceRegistryPermission proves explicit revocation/regrant and revision conflicts without entity self-grants.
func TestSourceRegistryPermission(t *testing.T) {
	_, db := fixture(t)
	id, _ := enroll(t, db)
	_, e := mutate(t, db, id, "set_source_permission", "grant", map[string]any{"expected_binding_revision": "1", "principal_id": other, "enabled": true})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = read(t, db, id, other); e != nil {
		t.Fatal(e)
	}
	_, e = mutate(t, db, id, "set_source_permission", "stale", map[string]any{"expected_binding_revision": "1", "principal_id": other, "enabled": false})
	requireCode(t, e, "P2302")
	_, e = mutate(t, db, id, "set_source_permission", "revoke", map[string]any{"expected_binding_revision": "2", "principal_id": other, "enabled": false})
	if e != nil {
		t.Fatal(e)
	}
	_, e = read(t, db, id, other)
	requireCode(t, e, "P2301")
	_, e = mutate(t, db, id, "set_source_permission", "regrant", map[string]any{"expected_binding_revision": "3", "principal_id": other, "enabled": true})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = read(t, db, id, other); e != nil {
		t.Fatal(e)
	}
}

// TestSourceRegistryPrivileges uses real runtime sessions against all banking functions and owned relations.
func TestSourceRegistryPrivileges(t *testing.T) {
	c, db := fixture(t)
	id, _ := enroll(t, db)
	for _, role := range []string{"bookkeeping_api", "bookkeeping_observer", "bookkeeping_publisher"} {
		allowed := ""
		if role == "bookkeeping_api" {
			allowed = "get_source"
		}
		if value(t, c, `SELECT count(*)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='banking' AND has_function_privilege($1,p.oid,'EXECUTE')<>(p.proname=$2)`, role, allowed) != "0" {
			t.Fatal("function privileges")
		}
	}
	r := connect(t, db, "synthetic_runtime")
	defer r.Close(context.Background())
	for _, q := range []string{"SET ROLE bookkeeping_owner", "SELECT * FROM banking.source_account", "SELECT * FROM banking.source_namespace", "SELECT * FROM banking.source_permission", "SELECT * FROM integration.source_registry_receipt", "CREATE TABLE banking.invalid(id integer)", "SELECT banking.source_error('P2300')", "SELECT banking.register_source(NULL,NULL,NULL,'x','{}')"} {
		_, e := r.Exec(context.Background(), q)
		requireCode(t, e, "42501")
	}
	// An owner member must explicitly assume the owner role; ambient login does not become an operator API.
	_, e := invoke(t, db, "postgres", "SELECT banking.register_source($1,$2,$3,'root',$4)::text", actor, entity, book, raw(manifest()))
	requireCode(t, e, "42501")
	_, e = r.Exec(context.Background(), "SELECT banking.get_source($1,$2,$3,$4)", reader, entity, book, id)
	requireCode(t, e, "P2300")
}

// TestSourceRegistryMalformed rejects duplicate keys, unknown types and unreviewed adapter profiles atomically.
func TestSourceRegistryMalformed(t *testing.T) {
	c, db := fixture(t)
	for i, bad := range []string{`{}`, `null`, `{"x":1,"x":2}`, raw(manifest()) + `{}`, strings.Replace(raw(manifest()), `"namespace":`, `"unknown":`, 1), strings.Replace(raw(manifest()), `jayburd-synthetic-statement-csv/1`, `plaid`, 1), strings.Repeat(" ", 8193), strings.Replace(raw(manifest()), `"binding_date":"2026-01-05"`, `"binding_date":null`, 1), strings.Replace(raw(manifest()), `"binding_date":"2026-01-05"`, `"binding_date":1e1000000`, 1)} {
		_, e := invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,$4,$5)::text", actor, entity, book, fmt.Sprint(i), bad)
		var p *pgconn.PgError
		if !errors.As(e, &p) || (p.Code != "P2300" && p.Code != "P2303") {
			t.Fatal("malformed accepted", e)
		}
	}
	if value(t, c, "SELECT count(*)::text FROM banking.source_account") != "0" {
		t.Fatal("malformed state persisted")
	}
}

// TestSourceRegistryConcurrentReplay proves competing real sessions create one registry and receipt.
func TestSourceRegistryConcurrentReplay(t *testing.T) {
	c, db := fixture(t)
	var wg sync.WaitGroup
	results := make(chan string, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, e := invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'race',$4)::text", actor, entity, book, raw(manifest()))
			results <- s
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	if <-results != <-results || value(t, c, "SELECT count(*)::text FROM banking.source_account") != "1" || value(t, c, "SELECT count(*)::text FROM integration.source_registry_receipt") != "1" {
		t.Fatal("race duplicated identity")
	}
}

// TestSourceRegistryRevocationRace proves source grant revocation cannot pass a held authorized read.
func TestSourceRegistryRevocationRace(t *testing.T) {
	for _, first := range []string{"read", "revoke"} {
		t.Run(first, func(t *testing.T) {
			c, db := fixture(t)
			id, _ := enroll(t, db)
			r := connect(t, db, "synthetic_runtime")
			defer r.Close(context.Background())
			if first == "read" {
				tx, e := r.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
				if e != nil {
					t.Fatal(e)
				}
				defer tx.Rollback(context.Background())
				var s string
				if e = tx.QueryRow(context.Background(), "SELECT banking.get_source($1,$2,$3,$4)::text", reader, entity, book, id).Scan(&s); e != nil {
					t.Fatal(e)
				}
				done := make(chan error, 1)
				go func() {
					_, e := mutate(t, db, id, "set_source_permission", "race-revoke", map[string]any{"expected_binding_revision": "1", "principal_id": reader, "enabled": false})
					done <- e
				}()
				select {
				case e := <-done:
					t.Fatal("revocation bypassed read", e)
				case <-time.After(60 * time.Millisecond):
				}
				if e = tx.Commit(context.Background()); e != nil {
					t.Fatal(e)
				}
				if e = <-done; e != nil {
					t.Fatal(e)
				}
			} else {
				tx, e := c.Begin(context.Background())
				if e != nil {
					t.Fatal(e)
				}
				defer tx.Rollback(context.Background())
				_, e = tx.Exec(context.Background(), "UPDATE banking.source_permission SET revoked_at=clock_timestamp()")
				if e != nil {
					t.Fatal(e)
				}
				done := make(chan error, 1)
				go func() { _, e := read(t, db, id, reader); done <- e }()
				select {
				case e := <-done:
					t.Fatal("read bypassed revoke", e)
				case <-time.After(60 * time.Millisecond):
				}
				if e = tx.Commit(context.Background()); e != nil {
					t.Fatal(e)
				}
				requireCode(t, <-done, "P2301")
			}
			_, e := read(t, db, id, reader)
			requireCode(t, e, "P2301")
		})
	}
}

// TestSourceRegistryDormantRevoke removes latent grants without restoring disabled subject authority.
func TestSourceRegistryDormantRevoke(t *testing.T) {
	for _, mode := range []string{"disabled", "membership"} {
		t.Run(mode, func(t *testing.T) {
			c, db := fixture(t)
			id, _ := enroll(t, db)
			if mode == "disabled" {
				exec(t, c, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", reader)
			} else {
				exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1", reader)
			}
			body := map[string]any{"expected_binding_revision": "1", "principal_id": reader, "enabled": false}
			result, e := mutate(t, db, id, "set_source_permission", "dormant-revoke", body)
			if e != nil {
				t.Fatal(e)
			}
			replay, e := mutate(t, db, id, "set_source_permission", "dormant-revoke", body)
			if e != nil || replay != result {
				t.Fatal("revocation replay", e)
			}
			_, e = mutate(t, db, id, "set_source_permission", "early-regrant", map[string]any{"expected_binding_revision": "2", "principal_id": reader, "enabled": true})
			requireCode(t, e, "P2301")
			exec(t, c, "UPDATE ledger.principal SET disabled_at=NULL WHERE principal_id=$1", reader)
			exec(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE principal_id=$1", reader)
			_, e = read(t, db, id, reader)
			requireCode(t, e, "P2301")
			_, e = mutate(t, db, id, "set_source_permission", "explicit-regrant", map[string]any{"expected_binding_revision": "2", "principal_id": reader, "enabled": true})
			if e != nil {
				t.Fatal(e)
			}
			if _, e = read(t, db, id, reader); e != nil {
				t.Fatal(e)
			}
		})
	}
}

// TestSourceRegistryAtomicity injects a last-write audit failure after all source rows have been inserted.
func TestSourceRegistryAtomicity(t *testing.T) {
	c, db := fixture(t)
	before := value(t, c, "SELECT revision::text FROM ledger.book")
	exec(t, c, `CREATE FUNCTION integration.synthetic_receipt_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic failure'; END $$; CREATE TRIGGER synthetic_failure BEFORE INSERT ON integration.source_registry_receipt FOR EACH ROW EXECUTE FUNCTION integration.synthetic_receipt_failure()`)
	_, e := invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'atomic',$4)::text", actor, entity, book, raw(manifest()))
	requireCode(t, e, "P0001")
	if value(t, c, "SELECT (SELECT count(*) FROM banking.source_account)::text||':'||(SELECT count(*) FROM banking.source_namespace)::text||':'||(SELECT count(*) FROM banking.source_permission)::text||':'||(SELECT count(*) FROM integration.source_registry_receipt)::text") != "0:0:0:0" || value(t, c, "SELECT revision::text FROM ledger.book") != before {
		t.Fatal("audit failure committed prefix")
	}
	exec(t, c, "DROP TRIGGER synthetic_failure ON integration.source_registry_receipt")
	call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'atomic',$4)::text", actor, entity, book, raw(manifest()))
	if value(t, c, "SELECT count(*)::text FROM banking.source_account") != "1" {
		t.Fatal("retry did not converge")
	}
}

// TestSourceRegistryMixedScope proves actual foreign references and cross-book namespace reuse are rejected.
func TestSourceRegistryMixedScope(t *testing.T) {
	c, db := fixture(t)
	id, _ := enroll(t, db)
	exec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Second synthetic book')`, entity, other)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'1000','Other book cash','asset','debit',true,'2026-01-01',NULL)`, entity, other, other)
	v := manifest()
	v["ledger_account_id"] = other
	_, e := invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'foreign-mapping',$4)::text", actor, entity, book, raw(v))
	requireCode(t, e, "P2303")
	_, e = invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'namespace-conflict',$4)::text", actor, entity, other, raw(v))
	requireCode(t, e, "P2302")
	_, e = invoke(t, db, "synthetic_runtime", "SELECT banking.get_source($1,$2,$3,$4)::text", reader, entity, other, id)
	requireCode(t, e, "P2301")
	_, e = c.Exec(context.Background(), `INSERT INTO banking.source_permission VALUES($1,$2,$3,$4,'read',NULL)`, entity, other, id, reader)
	requireCode(t, e, "23503")
	exec(t, c, "UPDATE ledger.account SET active_to='2026-01-05' WHERE account_id=$1", other)
	v["namespace"].(map[string]any)["source_account_ref"] = "CLOSED-CASH"
	_, e = invoke(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'closed',$4)::text", actor, entity, other, raw(v))
	requireCode(t, e, "P2303")
}
