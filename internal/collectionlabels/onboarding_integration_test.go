//go:build integration

package collectionlabels

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func onboarding47(t testing.TB, db, op string, args ...any) (map[string]any, error) {
	t.Helper()
	q := map[string]string{
		"catalog": "SELECT banking.get_collection_onboarding($1)", "preview": "SELECT banking.preview_collection_onboarding($1,$2)", "prepare": "SELECT banking.prepare_collection_onboarding($1,$2)", "pending": "SELECT banking.pending_collection_onboarding($1)", "transition": "SELECT banking.transition_collection_onboarding($1,$2,$3,$4)", "clear": "SELECT banking.clear_collection_onboarding($1,$2)",
	}[op]
	return call(t, db, "synthetic_runtime", "serializable", q, append([]any{actor}, args...)...)
}
func on47OK(t testing.TB, r map[string]any, e error) map[string]any {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func on47Preview(t testing.TB, db string, v any) map[string]any {
	t.Helper()
	r, e := onboarding47(t, db, "preview", raw(v))
	return on47OK(t, r, e)
}
func on47Prepare(t testing.TB, db, key string, p map[string]any) map[string]any {
	t.Helper()
	r, e := onboarding47(t, db, "prepare", raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": key}))
	return on47OK(t, r, e)
}
func on47Transition(t testing.TB, db string, c map[string]any, op string) map[string]any {
	t.Helper()
	r, e := onboarding47(t, db, "transition", c["command_id"], op, "{}")
	return on47OK(t, r, e)
}
func on47Pending(t testing.TB, db string) map[string]any {
	t.Helper()
	r, e := onboarding47(t, db, "pending")
	return on47OK(t, r, e)
}
func on47Setup(t testing.TB, db string) string {
	t.Helper()
	p := on47Preview(t, db, map[string]string{"operation": "setup"})
	c := on47Prepare(t, db, "setup", p)
	c = on47Transition(t, db, c, "execute")
	w := c["result"].(map[string]any)["workspace_id"].(string)
	on47Transition(t, db, c, "acknowledge")
	return w
}
func on47Candidate(t testing.TB, db string) map[string]any {
	t.Helper()
	r, e := onboarding47(t, db, "catalog")
	r = on47OK(t, r, e)
	a := r["candidates"].([]any)
	if len(a) != 1 {
		t.Fatal("wanted complete authorized one-Item catalog", len(a))
	}
	return a[0].(map[string]any)
}
func on47AdoptPreview(t testing.TB, db string) map[string]any {
	t.Helper()
	c := on47Candidate(t, db)
	return on47Preview(t, db, map[string]any{"operation": "adopt", "candidate_id": c["candidate_id"], "snapshot": c["snapshot"]})
}

func TestOnboarding47SetupLifecycle(t *testing.T) {
	owner, db := fixture(t)
	r, e := onboarding47(t, db, "catalog")
	r = on47OK(t, r, e)
	if r["can_setup"] != true || r["workspace"].(map[string]any)["state"] != "absent" || on47Pending(t, db)["state"] != "none" {
		t.Fatal("absent workspace not discoverable")
	}
	p := on47Preview(t, db, map[string]string{"operation": "setup"})
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_workspace") != "0" {
		t.Fatal("preview granted workspace")
	}
	c := on47Prepare(t, db, "setup-lost-reply", p)
	if c["prepare_key"] != "setup-lost-reply" || on47Pending(t, db)["state"] != "available" || value(t, owner, "SELECT count(*)::text FROM banking.collection_workspace") != "0" {
		t.Fatal("prepare granted or did not persist")
	}
	_, e = call(t, db, "synthetic_runtime", "serializable", "SELECT ledger.pending_command($1)", actor)
	code(t, e, "P2112")
	_, e = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.collection_recovery_status($1)", actor)
	code(t, e, "P2802")
	replay := on47Prepare(t, db, "setup-lost-reply", p)
	if replay["command_id"] != c["command_id"] {
		t.Fatal("prepare replay duplicated")
	}
	c = on47Transition(t, db, c, "execute")
	w := c["result"].(map[string]any)["workspace_id"].(string)
	c2 := on47Transition(t, db, c, "execute")
	if raw(c2) != raw(c) || value(t, owner, "SELECT count(*)::text FROM banking.collection_workspace") != "1" {
		t.Fatal("unknown execute replay duplicated")
	}
	c = on47Transition(t, db, c, "acknowledge")
	if c["acknowledged"] != true || raw(on47Transition(t, db, c, "acknowledge")) != raw(c) || on47Pending(t, db)["state"] != "none" {
		t.Fatal("exact ack recovery failed")
	}
	replay = on47Prepare(t, db, "setup-lost-reply", p)
	if replay["acknowledged"] != true || replay["command_id"] != c["command_id"] {
		t.Fatal("acknowledged prepare replay rejected")
	}
	r, e = onboarding47(t, db, "catalog")
	r = on47OK(t, r, e)
	if r["workspace"].(map[string]any)["workspace_id"] != w || r["can_setup"] != false {
		t.Fatal("setup result did not become ready")
	}
	_, e = onboarding47(t, db, "preview", `{"operation":"setup"}`)
	code(t, e, "P2802")
	if value(t, owner, "SELECT has_function_privilege('synthetic_runtime','banking.provision_collection_owner(uuid,text,text)','EXECUTE')::text") != "false" || value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("authority widened or journal created")
	}
}

func TestOnboarding47AdoptionAndFences(t *testing.T) {
	owner, db, _ := legacyFixture(t)
	w := on47Setup(t, db)
	before := legacyHash(t, owner)
	candidate := on47Candidate(t, db)
	encoded := raw(candidate)
	for _, bad := range []string{"legacy-item", "legacy-account", "original-cursor", "credential_ref"} {
		if contains47(encoded, bad) {
			t.Fatal("protected provider metadata escaped")
		}
	}
	p := on47AdoptPreview(t, db)
	c := on47Prepare(t, db, "adopt", p)
	if legacyHash(t, owner) != before {
		t.Fatal("inert adoption changed anchors")
	}
	c = on47Transition(t, db, c, "execute")
	legacyPage := page(t, db, w)
	if legacyPage["matching_count"] != "2" || len(legacyPage["items"].([]any)) != 2 || value(t, owner, "SELECT count(*)::text FROM banking.collection_observation") != "0" {
		t.Fatal("adopted original facts not readable through existing projection or copied to native storage")
	}
	for _, row := range legacyPage["items"].([]any) {
		r := row.(map[string]any)
		if r["money"].(map[string]any)["amount"] != "100" || r["observation_revision"] != "1" {
			t.Fatal("original units/revision changed")
		}
	}
	if value(t, owner, "SELECT state||':'||generation FROM integration.sync_item") != "paused:2" || legacyHash(t, owner) != before {
		t.Fatal("adoption changed history or failed fence")
	}
	if raw(on47Transition(t, db, c, "execute")) != raw(c) {
		t.Fatal("adoption replay changed receipt")
	}
	on47Transition(t, db, c, "acknowledge")
	r, e := onboarding47(t, db, "catalog")
	r = on47OK(t, r, e)
	if len(r["candidates"].([]any)) != 0 || r["workspace"].(map[string]any)["workspace_id"] != w {
		t.Fatal("adopted identity offered again")
	}
	// A distinct source generation drift stales execute, but cancellation stays recoverable.
	owner2, db2, _ := legacyFixture(t)
	on47Setup(t, db2)
	p = on47AdoptPreview(t, db2)
	c = on47Prepare(t, db2, "stale-generation", p)
	exec(t, owner2, "UPDATE integration.sync_item SET generation=generation+1")
	_, e = onboarding47(t, db2, "transition", c["command_id"], "execute", "{}")
	code(t, e, "P2803")
	on47Transition(t, db2, on47Transition(t, db2, c, "cancel"), "acknowledge")
}
func contains47(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestOnboarding47RevokedRecovery(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		for _, scope := range []string{"book", "read", "import"} {
			t.Run(fmt.Sprintf("%t_%s", terminal, scope), func(t *testing.T) {
				owner, db, _ := legacyFixture(t)
				w := on47Setup(t, db)
				c := on47Prepare(t, db, "adopt-revoke", on47AdoptPreview(t, db))
				if terminal {
					c = on47Transition(t, db, c, "execute")
				}
				if scope == "book" {
					exec(t, owner, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND principal_id=$2 AND permission='read'", entityID, actor)
				} else {
					exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3 AND principal_id=$4 AND capability=$5", entityID, bookID, sourceID, actor, scope)
				}
				pending := on47Pending(t, db)
				if pending["state"] != "inaccessible" || pending["command"] != nil || pending["recovery_id"] == nil {
					t.Fatal("revoked metadata leaked or work stranded")
				}
				rid := pending["recovery_id"]
				_, e := onboarding47(t, db, "transition", c["command_id"], "acknowledge", "{}")
				code(t, e, "P2801")
				r, e := onboarding47(t, db, "clear", raw(map[string]any{"recovery_id": rid}))
				r = on47OK(t, r, e)
				if r["cleared"] != true || on47Pending(t, db)["state"] != "none" {
					t.Fatal("inaccessible clear did not release")
				}
				// A new label claim must survive an old exact closure retry; wrong-engine
				// label clear cannot treat this onboarding closure as its own receipt.
				pp, e := preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{}))
				if e != nil {
					t.Fatal(e)
				}
				label, e := prepare(t, db, w, "new-label-claim", pp)
				if e != nil {
					t.Fatal(e)
				}
				r, e = onboarding47(t, db, "clear", raw(map[string]any{"recovery_id": rid}))
				on47OK(t, r, e)
				if on47Pending(t, db)["state"] != "blocked" {
					t.Fatal("old receipt released newer engine")
				}
				_, e = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.clear_collection_recovery($1,$2)", actor, raw(map[string]any{"recovery_id": rid}))
				code(t, e, "P2801")
				if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation WHERE label_command_id=$1", command(t, label)["command_id"]) != "1" {
					t.Fatal("new label reservation lost")
				}
			})
		}
	}
}

func TestOnboarding47ClosedAuthorityAndNoRestore(t *testing.T) {
	owner, db := fixture(t)
	p := on47Preview(t, db, map[string]string{"operation": "setup"})
	_, e := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.prepare_collection_onboarding($1,$2)", other, raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": "foreign"}))
	code(t, e, "P2801")
	for _, rawBody := range []string{`{"operation":"setup","actor":"foreign"}`, `{"operation":"setup","operation":"setup"}`, `{"Operation":"setup"}`, `{"operation":"adopt","candidate_id":"00000000-0000-4000-8000-000000000999","snapshot":"bad"}`} {
		_, e = onboarding47(t, db, "preview", rawBody)
		var pe *pgconn.PgError
		if !errors.As(e, &pe) || (pe.Code != "P2800" && pe.Code != "P2801") {
			t.Fatal("closed body accepted or unsafe error", e)
		}
	}
	w := on47Setup(t, db)
	_, e = call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,'read-only',$2)", actor, raw(map[string]any{"expected_revision": "1", "capabilities": []string{"read"}, "evidence_ref": evidence}))
	if e != nil {
		t.Fatal(e)
	}
	r, e := onboarding47(t, db, "catalog")
	r = on47OK(t, r, e)
	if r["can_setup"] != false || r["can_adopt"] != false || r["workspace"].(map[string]any)["state"] != "ready" {
		t.Fatal("read-only workspace was restorable")
	}
	_, e = onboarding47(t, db, "prepare", raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": "old"}))
	code(t, e, "P2801")
	if value(t, owner, "SELECT capabilities::text FROM banking.collection_workspace WHERE workspace_id=$1", w) != "{read}" {
		t.Fatal("capabilities elevated")
	}
	exec(t, owner, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", actor)
	_, e = onboarding47(t, db, "catalog")
	code(t, e, "P2801")
}

// TestOnboarding47PrimedWorkspace observes a real wait on the persistent fence;
// an older snapshot cannot report absent after a concurrent setup commit.
func TestOnboarding47PrimedWorkspace(t *testing.T) {
	owner, db := fixture(t)
	reader := connect(t, db, "synthetic_runtime")
	defer reader.Close(context.Background())
	tx, e := reader.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(context.Background())
	var n int
	if e = tx.QueryRow(context.Background(), "SELECT count(*) FROM pg_class").Scan(&n); e != nil {
		t.Fatal(e)
	}
	writer, e := owner.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer writer.Rollback(context.Background())
	// Fixture writer models an owner-provision path without changing the principal
	// row, isolating the persistent workspace-existence fence from principal locks.
	if _, e = writer.Exec(context.Background(), "INSERT INTO banking.collection_workspace(owner_principal_id,capabilities)VALUES($1,ARRAY['read'])", actor); e != nil {
		t.Fatal(e)
	}
	observer := connect(t, db, "postgres")
	defer observer.Close(context.Background())
	done := make(chan error, 1)
	go func() {
		var b []byte
		er := tx.QueryRow(context.Background(), "SELECT banking.get_collection_onboarding($1)", actor).Scan(&b)
		_ = tx.Rollback(context.Background())
		done <- er
	}()
	waitBlocked(t, observer, reader.PgConn().PID())
	if e = writer.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	code(t, <-done, "40001")
	r, e := onboarding47(t, db, "catalog")
	r = on47OK(t, r, e)
	if r["can_setup"] != false {
		t.Fatal("fresh discovery still absent")
	}
}

func TestOnboarding47SharedGateRaces(t *testing.T) {
	for _, winner := range []string{"onboarding", "book"} {
		t.Run(winner, func(t *testing.T) {
			owner, db := fixture(t)
			p := on47Preview(t, db, map[string]string{"operation": "setup"})
			praw := raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": "race-onboarding"})
			oldBody := raw(map[string]any{"operation": "setup", "entity_id": nil, "book_id": nil, "body": map[string]any{"entity_name": "Fictional race entity", "entity_kind": "personal", "book_name": "Fictional race book", "starts_on": "2026-01-01", "ends_before": "2027-01-01", "functional_currency_version_id": currency, "accounts": []map[string]any{{"code": "1000", "name": "Fictional cash", "category": "asset", "normal_side": "debit", "monetary": true}, {"code": "3000", "name": "Fictional equity", "category": "equity", "normal_side": "credit", "monetary": false}}}})
			onSQL := "SELECT banking.prepare_collection_onboarding($1,$2)"
			bookSQL := "SELECT ledger.register_command($1,'race-book',$2,$3)"
			first := connect(t, db, "synthetic_runtime")
			defer first.Close(context.Background())
			second := connect(t, db, "synthetic_runtime")
			defer second.Close(context.Background())
			tx, e := first.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(context.Background())
			var result []byte
			if winner == "onboarding" {
				e = tx.QueryRow(context.Background(), onSQL, actor, praw).Scan(&result)
			} else {
				e = tx.QueryRow(context.Background(), bookSQL, actor, evidence, oldBody).Scan(&result)
			}
			if e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			go func() {
				rt, er := second.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
				if er != nil {
					done <- er
					return
				}
				defer rt.Rollback(context.Background())
				var b []byte
				if winner == "onboarding" {
					er = rt.QueryRow(context.Background(), bookSQL, actor, evidence, oldBody).Scan(&b)
				} else {
					er = rt.QueryRow(context.Background(), onSQL, actor, praw).Scan(&b)
				}
				if er == nil {
					er = rt.Commit(context.Background())
				}
				_ = rt.Rollback(context.Background())
				done <- er
			}()
			waitBlocked(t, owner, second.PgConn().PID())
			if e = tx.Commit(context.Background()); e != nil {
				t.Fatal(e)
			}
			e = <-done
			var pe *pgconn.PgError
			if errors.As(e, &pe) && pe.Code == "40001" {
				if winner == "onboarding" {
					_, e = call(t, db, "synthetic_runtime", "serializable", bookSQL, actor, evidence, oldBody)
				} else {
					_, e = onboarding47(t, db, "prepare", praw)
				}
			}
			if winner == "onboarding" {
				code(t, e, "P2112")
			} else {
				code(t, e, "P2802")
			}
			if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation") != "1" {
				t.Fatal("multiple engines admitted")
			}
		})
	}
}

func TestOnboarding47HistoricalPrepareReplay(t *testing.T) {
	owner, db := fixture(t)
	w := on47Setup(t, db)
	// Independently authored already-committed historical intent models a reply
	// lost before its expiry; no immutable product row is edited to simulate time.
	var pid, cid string
	var result []byte
	if e := owner.QueryRow(context.Background(), `INSERT INTO banking.collection_onboarding_preview(principal_id,operation,snapshot,review,created_at,expires_at)VALUES($1,'setup',repeat('a',64),'{"capabilities":["read","acquire","classify"]}',clock_timestamp()-interval '20 minutes',clock_timestamp()-interval '5 minutes')RETURNING preview_id`, actor).Scan(&pid); e != nil {
		t.Fatal(e)
	}
	result = []byte(raw(map[string]any{"workspace_id": w, "workspace_revision": "1", "capabilities": []string{"read", "acquire", "classify"}}))
	if e := owner.QueryRow(context.Background(), `INSERT INTO banking.collection_onboarding_command(principal_id,preview_id,prepare_key,state,result,terminal_at)VALUES($1,$2,'historical','committed',$3,clock_timestamp()-interval '6 minutes')RETURNING command_id`, actor, pid, result).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	p := map[string]any{"preview_id": pid}
	c := on47Prepare(t, db, "historical", p)
	if c["state"] != "committed" || c["command_id"] != cid {
		t.Fatal("historical receipt rejected")
	}
	c = on47Transition(t, db, c, "acknowledge")
	if on47Prepare(t, db, "historical", p)["acknowledged"] != true {
		t.Fatal("acknowledged historical replay rejected")
	}
}

// TestOnboarding47CurrentItemReplayLocks primes before real authority writes,
// observes the wait and proves current Item/alias locks force whole retry rather
// than returning stale saved payloads through an unchanged principal/workspace.
func TestOnboarding47CurrentItemReplayLocks(t *testing.T) {
	for _, kind := range []string{"item", "source", "legacy_generation", "consent"} {
		t.Run(kind, func(t *testing.T) {
			owner, db, item := legacyFixture(t)
			w := on47Setup(t, db)
			p := on47AdoptPreview(t, db)
			c := on47Prepare(t, db, "current-replay", p)
			if kind != "legacy_generation" && kind != "consent" {
				c = on47Transition(t, db, c, "execute")
			}
			reader := connect(t, db, "synthetic_runtime")
			defer reader.Close(context.Background())
			rt, e := reader.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				t.Fatal(e)
			}
			defer rt.Rollback(context.Background())
			var n int
			if e = rt.QueryRow(context.Background(), "SELECT count(*) FROM pg_class").Scan(&n); e != nil {
				t.Fatal(e)
			}
			login := "synthetic_runtime"
			if kind == "legacy_generation" || kind == "consent" {
				login = "synthetic_operator"
			}
			writer := connect(t, db, login)
			defer writer.Close(context.Background())
			wt, e := writer.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				t.Fatal(e)
			}
			defer wt.Rollback(context.Background())
			var out []byte
			if kind == "legacy_generation" || kind == "consent" {
				if _, e = wt.Exec(context.Background(), "SET LOCAL ROLE bookkeeping_owner"); e != nil {
					t.Fatal(e)
				}
				if kind == "consent" {
					_, e = wt.Exec(context.Background(), "UPDATE integration.sync_item SET consent_principal=$1,generation=generation+1 WHERE sync_item_id=$2", other, item["sync_item_id"])
				} else {
					e = wt.QueryRow(context.Background(), "SELECT integration.set_sync_item_state($1,$2,$3,$4,'generation-race',$5)", actor, entityID, bookID, item["sync_item_id"], `{"expected_generation":"1","state":"paused"}`).Scan(&out)
				}
			} else {
				result := c["result"].(map[string]any)
				target := result["collection_item_id"]
				rev := "2"
				if kind == "source" {
					target = result["sources"].([]any)[0].(map[string]any)["source_account_id"]
					rev = "1"
				}
				e = wt.QueryRow(context.Background(), "SELECT banking.set_collection_state($1,$2,'revoke-race',$3)", actor, w, raw(map[string]any{"kind": kind, "target_id": target, "expected_revision": rev, "state": "revoked"})).Scan(&out)
			}
			if e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			go func() {
				var b []byte
				er := rt.QueryRow(context.Background(), "SELECT banking.prepare_collection_onboarding($1,$2)", actor, raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": "current-replay"})).Scan(&b)
				_ = rt.Rollback(context.Background())
				done <- er
			}()
			waitBlocked(t, owner, reader.PgConn().PID())
			if e = wt.Commit(context.Background()); e != nil {
				t.Fatal(e)
			}
			code(t, <-done, "40001")
			if kind == "legacy_generation" {
				on47Prepare(t, db, "current-replay", p)
				on47Transition(t, db, on47Transition(t, db, c, "cancel"), "acknowledge")
			} else {
				_, e = onboarding47(t, db, "prepare", raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": "current-replay"}))
				code(t, e, "P2801")
				if on47Pending(t, db)["state"] != "inaccessible" {
					t.Fatal("fresh current revocation not enforced")
				}
			}
		})
	}

}

func TestOnboarding47Privileges(t *testing.T) {
	owner, db := fixture(t)
	allowed := []string{"get_collection_onboarding(uuid)", "preview_collection_onboarding(uuid, text)", "prepare_collection_onboarding(uuid, text)", "pending_collection_onboarding(uuid)", "transition_collection_onboarding(uuid, uuid, text, text)", "clear_collection_onboarding(uuid, text)"}
	for _, role := range []string{"bookkeeping_api", "bookkeeping_sync", "bookkeeping_webhook", "bookkeeping_observer", "bookkeeping_publisher", "bookkeeping_verifier", "bookkeeping_identity_operator"} {
		want := []string{}
		if role == "bookkeeping_api" {
			want = allowed
		}
		if value(t, owner, `SELECT count(*)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='banking'AND p.proname LIKE '%collection_onboarding%'AND has_function_privilege($1,p.oid,'EXECUTE')<>((p.proname||'('||oidvectortypes(p.proargtypes)||')')=ANY($2::text[]))`, role, want) != "0" {
			t.Fatal("onboarding function inventory widened", role)
		}
	}
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	for _, q := range []string{"SET ROLE bookkeeping_owner", "SELECT * FROM banking.collection_onboarding_command", "SELECT * FROM banking.collection_onboarding_preview", "SELECT * FROM audit.collection_onboarding_action", "SELECT banking.collection_onboarding_candidate(NULL,NULL,true)", "SELECT ledger.onboarding_pending_base(NULL)", "SELECT banking.provision_collection_owner(NULL,'x','{}')"} {
		if _, e := runtime.Exec(context.Background(), q); e == nil {
			t.Fatal("runtime crossed onboarding helper/table/owner boundary")
		}
	}
}

func TestOnboarding47LabelGateRaces(t *testing.T) {
	for _, winner := range []string{"onboarding", "label"} {
		t.Run(winner, func(t *testing.T) {
			owner, db, _ := legacyFixture(t)
			w := on47Setup(t, db)
			op := on47AdoptPreview(t, db)
			lp, e := preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{}))
			if e != nil {
				t.Fatal(e)
			}
			onSQL := "SELECT banking.prepare_collection_onboarding($1,$2)"
			onBody := raw(map[string]any{"preview_id": op["preview_id"], "prepare_key": "race-onboard"})
			labelSQL := "SELECT banking.prepare_collection_label_command($1,$2,$3)"
			labelBody := raw(map[string]any{"preview_id": lp["preview_id"], "prepare_key": "race-label"})
			first := connect(t, db, "synthetic_runtime")
			defer first.Close(context.Background())
			second := connect(t, db, "synthetic_runtime")
			defer second.Close(context.Background())
			tx, e := first.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(context.Background())
			var out []byte
			if winner == "onboarding" {
				e = tx.QueryRow(context.Background(), onSQL, actor, onBody).Scan(&out)
			} else {
				e = tx.QueryRow(context.Background(), labelSQL, actor, w, labelBody).Scan(&out)
			}
			if e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			go func() {
				rt, er := second.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
				if er != nil {
					done <- er
					return
				}
				defer rt.Rollback(context.Background())
				var b []byte
				if winner == "onboarding" {
					er = rt.QueryRow(context.Background(), labelSQL, actor, w, labelBody).Scan(&b)
				} else {
					er = rt.QueryRow(context.Background(), onSQL, actor, onBody).Scan(&b)
				}
				if er == nil {
					er = rt.Commit(context.Background())
				}
				_ = rt.Rollback(context.Background())
				done <- er
			}()
			waitBlocked(t, owner, second.PgConn().PID())
			if e = tx.Commit(context.Background()); e != nil {
				t.Fatal(e)
			}
			e = <-done
			var pe *pgconn.PgError
			if errors.As(e, &pe) && pe.Code == "40001" {
				if winner == "onboarding" {
					_, e = call(t, db, "synthetic_runtime", "serializable", labelSQL, actor, w, labelBody)
				} else {
					_, e = onboarding47(t, db, "prepare", onBody)
				}
			}
			code(t, e, "P2802")
			if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation") != "1" {
				t.Fatal("independent label/onboarding gates admitted")
			}
			if winner == "onboarding" {
				for _, suffix := range []string{"", "_v2", "_v3"} {
					_, e = call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.pending_collection_label_command"+suffix+"($1,$2)", actor, w)
					code(t, e, "P2802")
				}
			} else {
				if on47Pending(t, db)["state"] != "blocked" {
					t.Fatal("label claim not globally blocked")
				}
			}
		})
	}
}

// Reject the terminal write after business mutation to prove the whole command
// and original history remain atomic under an actual restricted execution.
func TestOnboarding47AtomicFailure(t *testing.T) {
	for _, operation := range []string{"setup", "adopt"} {
		t.Run(operation, func(t *testing.T) {
			var owner *pgx.Conn
			var db string
			if operation == "setup" {
				owner, db = fixture(t)
			} else {
				owner, db, _ = legacyFixture(t)
				on47Setup(t, db)
			}
			var p map[string]any
			if operation == "setup" {
				p = on47Preview(t, db, map[string]string{"operation": "setup"})
			} else {
				p = on47AdoptPreview(t, db)
			}
			c := on47Prepare(t, db, "atomic", p)
			before := legacyHash(t, owner)
			exec(t, owner, `CREATE FUNCTION banking.synthetic_onboarding_abort() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF NEW.state='committed' THEN RAISE EXCEPTION USING ERRCODE='P2802',MESSAGE='synthetic_atomic_abort';END IF; RETURN NEW; END$$; CREATE TRIGGER synthetic_abort BEFORE UPDATE ON banking.collection_onboarding_command FOR EACH ROW EXECUTE FUNCTION banking.synthetic_onboarding_abort()`)
			_, e := onboarding47(t, db, "transition", c["command_id"], "execute", "{}")
			code(t, e, "P2802")
			if legacyHash(t, owner) != before || on47Pending(t, db)["command"].(map[string]any)["state"] != "prepared" {
				t.Fatal("late failure escaped atomic boundary")
			}
			if operation == "setup" && value(t, owner, "SELECT count(*)::text FROM banking.collection_workspace") != "0" {
				t.Fatal("failed command granted workspace")
			}
			if operation == "adopt" && value(t, owner, "SELECT count(*)::text FROM integration.collection_item WHERE workspace_id IS NOT NULL") != "0" {
				t.Fatal("failed command adopted Item")
			}
			exec(t, owner, "DROP TRIGGER synthetic_abort ON banking.collection_onboarding_command; DROP FUNCTION banking.synthetic_onboarding_abort()")
			on47Transition(t, db, on47Transition(t, db, c, "execute"), "acknowledge")
		})
	}
}

// A delivered command carries the exact recovery identity even if the subsequent
// acknowledgement reply is lost and current disclosure is later revoked.
func TestOnboarding47LostAcknowledgementRecovery(t *testing.T) {
	owner, db, _ := legacyFixture(t)
	w := on47Setup(t, db)
	p := on47AdoptPreview(t, db)
	c := on47Prepare(t, db, "lost-ack", p)
	rid := c["recovery_id"]
	if rid == nil || on47Pending(t, db)["recovery_id"] != rid {
		t.Fatal("command recovery identity not bound to current claim")
	}
	c = on47Transition(t, db, c, "execute")
	on47Transition(t, db, c, "acknowledge") // Commit succeeds; client discards the reply.
	exec(t, owner, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND principal_id=$2 AND permission='read'", entityID, actor)
	pp, e := preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{}))
	if e != nil {
		t.Fatal(e)
	}
	label, e := prepare(t, db, w, "after-lost-ack", pp)
	if e != nil {
		t.Fatal(e)
	}
	r, e := onboarding47(t, db, "clear", raw(map[string]any{"recovery_id": rid}))
	r = on47OK(t, r, e)
	if len(r) != 3 || r["recovery_id"] != rid || r["cleared"] != true || on47Pending(t, db)["state"] != "blocked" {
		t.Fatal("lost acknowledgement recovery leaked metadata or lost newer work")
	}
	if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation WHERE label_command_id=$1", command(t, label)["command_id"]) != "1" {
		t.Fatal("newer label claim released")
	}
}
