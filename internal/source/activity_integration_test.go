//go:build integration

package source

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
	"github.com/justin-napolitano/jayburd-bookkeeping/tests/corpus/spending"
)

// activityUUID maps only fictional fixture names to deterministic test identities.
func activityUUID(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

// activityOracle adapts original fixture inputs, never calculating expected results.
// Owner inserts test evidence only; this does not prove a provider publication or managed identity flow.
func activityOracle(t *testing.T) (*pgx.Conn, string, map[string]string, map[string]string) {
	t.Helper()
	c, db := registryFixture(t, false)
	var input struct {
		CurrencyVersions []struct {
			ID, Code string
			Minor    int     `json:"minor_unit"`
			From     string  `json:"valid_from"`
			To       *string `json:"valid_to"`
		} `json:"currency_versions"`
		Scopes       []struct{ ID string } `json:"scopes"`
		Observations []struct {
			ID, Scope string
			Current   string `json:"current_revision"`
			Hold      *string
			Adapter   string `json:"adapter_profile"`
			Revisions []struct {
				Revision, Date, State, Direction, Amount string
				Unit                                     string  `json:"currency_version"`
				Lexeme                                   string  `json:"source_lexeme"`
				Merchant                                 *string `json:"merchant_name"`
			}
		} `json:"observations"`
	}
	if err := json.Unmarshal(spending.Observations, &input); err != nil {
		t.Fatal(err)
	}
	units := map[string]string{}
	sources := map[string]string{}
	namespaces := map[string]string{}
	scopeBooks := map[string]string{}
	for i, u := range input.CurrencyVersions {
		units[u.ID] = activityUUID(1000 + i)
		code := map[string]string{"USD": "840", "EUR": "978", "JPY": "392", "KWD": "414"}[u.Code]
		exec(t, c, `INSERT INTO reference.currency VALUES($1,$2,$3,$4,$5,$6,'fictional-oracle')`, units[u.ID], u.Code, code, u.Minor, u.From, u.To)
	}
	for i, scope := range input.Scopes {
		bb := book
		if scope.ID == "other-book" {
			bb = activityUUID(11000)
			exec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Fictional second book')`, entity, bb)
		}
		scopeBooks[scope.ID] = bb
		sid, nid, aid := activityUUID(2000+i), activityUUID(3000+i), activityUUID(4000+i)
		sources[scope.ID], namespaces[scope.ID] = sid, nid
		exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,$4,'Fictional oracle source','asset','debit',true,'2026-01-01',NULL)`, entity, bb, aid, fmt.Sprint(5000+i))
		exec(t, c, `INSERT INTO banking.source_account(entity_id,book_id,source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by) VALUES($1,$2,$3,$4,$5,'2026-01-01',$6,$7)`, entity, bb, sid, aid, units["usd-before"], activityUUID(6000+i), actor)
		exec(t, c, `INSERT INTO banking.source_namespace(entity_id,book_id,namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by) VALUES($1,$2,$3,$4,'jayburd-synthetic-statement-csv/1',$5,$6,false,$7)`, entity, bb, nid, sid, activityUUID(7000+i), fmt.Sprintf("SYNTH-%d", i), actor)
		principal := reader
		if scope.ID == "foreign" {
			principal = other
		}
		exec(t, c, `INSERT INTO banking.source_permission VALUES($1,$2,$3,$4,'read',NULL)`, entity, bb, sid, principal)
	}
	for i, o := range input.Observations {
		if o.Current != fmt.Sprint(len(o.Revisions)) {
			t.Fatal("oracle current pointer needs an explicit adapter")
		}
		oid, rid, did := activityUUID(8000+i), activityUUID(9000+i), activityUUID(10000+i)
		sid, nid := sources[o.Scope], namespaces[o.Scope]
		bb := scopeBooks[o.Scope]
		exec(t, c, "BEGIN")
		exec(t, c, `INSERT INTO banking.source_observation(entity_id,book_id,observation_id,source_account_id,namespace_id,external_entry_id) VALUES($1,$2,$3,$4,$5,$6)`, entity, bb, oid, sid, nid, o.ID)
		for j, r := range o.Revisions {
			if r.Revision != fmt.Sprint(j+1) {
				t.Fatal("oracle revisions are not sequential")
			}
			adapter := o.Adapter
			if adapter == "" {
				adapter = "jayburd-plaid-transactions/1"
			}
			var provider, lexeme, previous any
			if adapter != "jayburd-synthetic-statement-csv/1" {
				provider = raw(map[string]any{"merchant_name": r.Merchant})
				lexeme = r.Lexeme
			}
			if j > 0 {
				previous = j
			}
			exec(t, c, `INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields,previous_revision,adapter_profile,provider_amount_lexeme,provider_fields) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::text,$9::text::numeric,$10,'{}',$11,$12,$13,$14::jsonb)`, entity, bb, oid, j+1, r.State, r.Date, units[r.Unit], r.Direction, r.Amount, "Fictional "+o.ID, previous, adapter, lexeme, provider)
			if j > 0 {
				exec(t, c, `UPDATE banking.source_observation SET current_revision=current_revision+1 WHERE observation_id=$1`, oid)
			}
		}
		// Prove complete inventory includes pending/zero facts with no review projection.
		current := o.Revisions[len(o.Revisions)-1]
		if current.State == "pending" || current.Direction == "zero" {
			exec(t, c, "COMMIT")
			continue
		}
		state := "needs_review"
		if o.Hold != nil {
			if *o.Hold == "overlap" {
				state = "overlap_hold"
			} else {
				state = "correction_hold"
			}
		}
		exec(t, c, `INSERT INTO banking.review_item(entity_id,book_id,review_id,source_account_id,state) VALUES($1,$2,$3,$4,$5)`, entity, bb, rid, sid, state)
		exec(t, c, `INSERT INTO banking.review_decision(entity_id,book_id,decision_id,review_id,expected_review_revision,kind,actor_id,reason,body) VALUES($1,$2,$3,$4,0,'initialize',$5,'synthetic_import','{}')`, entity, bb, did, rid, actor)
		exec(t, c, `INSERT INTO banking.review_evidence VALUES($1,$2,$3,$4,$5,1)`, entity, bb, did, oid, rid)
		exec(t, c, `INSERT INTO banking.review_assignment VALUES($1,$2,$3,$4,$5)`, entity, bb, oid, rid, did)
		exec(t, c, "COMMIT")
	}
	return c, db, sources, units
}

// activityInput fixes the closed profile; tests independently change selectors and pins.
func activityInput(sid string) map[string]any {
	return map[string]any{"profile": "source-activity-v1", "source_account_id": sid, "starts_on": "2026-11-01", "through_date": "2026-11-30", "search": "", "kind": "summary", "currency_version_id": nil, "merchant": nil, "snapshot": nil, "cursor": nil, "limit": 20}
}

// activityRead invokes only the restricted read using a real synthetic runtime login.
func activityRead(t testing.TB, db string, q map[string]any) map[string]any {
	t.Helper()
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(q)))
}

// TestSpendingCurrenciesSeparate uses fixed expected quantities, never the fixture adapter's arithmetic.
func TestSpendingCurrenciesSeparate(t *testing.T) {
	c, db, sources, units := activityOracle(t)
	revision := value(t, c, "SELECT revision::text FROM ledger.book")
	q := activityInput(sources["checking"])
	p := activityRead(t, db, q)
	inv := p["inventory"].(map[string]any)
	for k, want := range map[string]string{"source_count": "21", "matching_count": "20", "eligible_count": "15", "pending_count": "1", "removed_count": "1", "zero_count": "1", "quality_hold_count": "2", "correction_attention_count": "1"} {
		if inv[k] != want {
			t.Fatal("inventory", k, inv[k])
		}
	}
	if inv["partial"] != true || p["bank_balance"] != nil || p["last_successful_sync"] != nil {
		t.Fatal("unknown/partial became zero or complete")
	}
	groups := p["units"].([]any)
	if len(groups) != 2 {
		t.Fatal("currency versions folded")
	}
	for _, v := range groups {
		u := v.(map[string]any)
		if u["currency_version_id"] == units["usd-before"] {
			if u["inflow"] != "2017.5" || u["outflow"] != "940" || u["net_inflow"] != "1077.5" || u["count"] != "14" {
				t.Fatal("independent checking totals", u)
			}
		} else if u["outflow"] != "2" {
			t.Fatal("new unit version")
		}
	}
	for scope, want := range map[string]string{"card": "85", "savings": "0", "yen-account": "1250", "kwd-account": "1.005", "eur-account": "20", "legacy": "60"} {
		out := activityRead(t, db, activityInput(sources[scope]))
		u := out["units"].([]any)[0].(map[string]any)
		if u["outflow"] != want {
			t.Fatal("original unit", scope, u)
		}
	}
	importNoFinance(t, c, revision)
}

// TestMerchantNormalizationVersion proves exact case-sensitive groups and unavailable metadata.
func TestMerchantNormalizationVersion(t *testing.T) {
	c, db, sources, units := activityOracle(t)
	q := activityInput(sources["checking"])
	p := activityRead(t, db, q)
	q["snapshot"], q["kind"], q["currency_version_id"], q["limit"] = p["snapshot"], "merchants", units["usd-before"], 2
	seen := map[string]bool{}
	pages := 0
	var lastCursor string
	groupIn, groupOut := new(big.Rat), new(big.Rat)
	for {
		if q["cursor"] != nil {
			lastCursor = q["cursor"].(string)
		}
		p = activityRead(t, db, q)
		for _, v := range p["merchant_groups"].([]any) {
			g := v.(map[string]any)
			key := fmt.Sprint(g["label"])
			if seen[key] {
				t.Fatal("duplicate group")
			}
			seen[key] = true
			for field, total := range map[string]*big.Rat{"inflow": groupIn, "outflow": groupOut} {
				amount, ok := new(big.Rat).SetString(g[field].(string))
				if !ok {
					t.Fatal("not exact group decimal")
				}
				total.Add(total, amount)
			}
			if key == "Fictional Cafe" && (g["inflow"] != "7.5" || g["outflow"] != "12.5" || g["net_outflow"] != "5") {
				t.Fatal("merchant exact totals")
			}
		}
		pages++
		q["cursor"] = p["next_cursor"]
		if q["cursor"] == nil {
			break
		}
		if pages > 4 {
			t.Fatal("unbounded group page")
		}
	}
	if len(seen) != 6 || !seen["FICTIONAL CAFE"] || !seen["Fictional Cafe"] || p["remaining_group_count"] != "0" {
		t.Fatal("case labels merged or lost", seen)
	}
	if groupIn.Cmp(big.NewRat(4035, 2)) != 0 || groupOut.Cmp(big.NewRat(940, 1)) != 0 {
		t.Fatal("complete merchant drill does not reconcile")
	}
	// An authorized end anchor has an empty remainder, never the original group count.
	decoded, err := hex.DecodeString(lastCursor)
	if err != nil {
		t.Fatal(err)
	}
	var end map[string]any
	if err = json.Unmarshal(decoded, &end); err != nil {
		t.Fatal(err)
	}
	end["anchor"] = p["group_count"]
	q["cursor"] = hex.EncodeToString([]byte(raw(end)))
	empty := activityRead(t, db, q)
	if len(empty["merchant_groups"].([]any)) != 0 || empty["remaining_group_count"] != "0" {
		t.Fatal("end anchor invented remainder")
	}
	q = activityInput(sources["legacy"])
	p = activityRead(t, db, q)
	q["snapshot"], q["kind"], q["currency_version_id"] = p["snapshot"], "merchants", units["usd-before"]
	p = activityRead(t, db, q)
	if p["merchant_groups"].([]any)[0].(map[string]any)["status"] != "unavailable" {
		t.Fatal("legacy metadata invented")
	}
	// Preserve a valid one-letter label; vertical-tab escaping must not strip the letter v.
	exec(t, c, "BEGIN")
	exec(t, c, `INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields,previous_revision,adapter_profile,provider_amount_lexeme,provider_fields) SELECT entity_id,book_id,observation_id,revision+1,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields,revision,adapter_profile,provider_amount_lexeme,'{"merchant_name":"v"}'::jsonb FROM banking.observation_revision WHERE observation_id=$1 AND revision=1`, activityUUID(8001))
	exec(t, c, `UPDATE banking.source_observation SET current_revision=current_revision+1 WHERE observation_id=$1`, activityUUID(8001))
	exec(t, c, `UPDATE banking.source_account SET source_revision=source_revision+1 WHERE source_account_id=$1`, sources["checking"])
	exec(t, c, "COMMIT")
	q = activityInput(sources["checking"])
	p = activityRead(t, db, q)
	q["kind"], q["currency_version_id"], q["snapshot"], q["merchant"] = "observations", units["usd-before"], p["snapshot"], map[string]any{"status": "attributed", "label": "v"}
	p = activityRead(t, db, q)
	if p["totals"].(map[string]any)["outflow"] != "12.5" || p["items"].([]any)[0].(map[string]any)["merchant"].(map[string]any)["label"] != "v" {
		t.Fatal("valid exact label treated as blank")
	}
}

// TestAggregateDrillEquality covers exact merchant selectors, source correction and stale/changed pins.
func TestAggregateDrillEquality(t *testing.T) {
	c, db, sources, units := activityOracle(t)
	q := activityInput(sources["checking"])
	p := activityRead(t, db, q)
	q["snapshot"], q["kind"], q["currency_version_id"], q["limit"] = p["snapshot"], "observations", units["usd-before"], 1
	q["merchant"] = map[string]any{"status": "attributed", "label": "Fictional Cafe"}
	first := activityRead(t, db, q)
	if first["totals"].(map[string]any)["net_inflow"] != "-5" {
		t.Fatal("drill aggregate")
	}
	q["cursor"] = first["next_cursor"]
	second := activityRead(t, db, q)
	if second["items"].([]any)[0].(map[string]any)["amount"] != "7.5" || second["next_cursor"] != nil {
		t.Fatal("exact merchant drill")
	}
	q["merchant"] = map[string]any{"status": "attributed", "label": "FICTIONAL CAFE"}
	_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(q))
	requireCode(t, err, "P2500")
	q["cursor"], q["merchant"], q["limit"] = nil, map[string]any{"status": "attributed", "label": "Fictional Corrected"}, 100
	corrected := activityRead(t, db, q)
	if corrected["totals"].(map[string]any)["outflow"] != "15" || corrected["items"].([]any)[0].(map[string]any)["observation_revision"] != "2" {
		t.Fatal("old source revision counted")
	}
	q["currency_version_id"] = units["kwd"]
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(q))
	requireCode(t, err, "P2505")
	q["currency_version_id"] = units["usd-before"]
	q["merchant"] = map[string]any{"status": "attributed", "label": "Fictional absent merchant"}
	empty := activityRead(t, db, q)
	if empty["totals"].(map[string]any)["count"] != "0" || len(empty["items"].([]any)) != 0 {
		t.Fatal("valid empty drill")
	}
	exec(t, c, `UPDATE banking.source_account SET source_revision=source_revision+1 WHERE source_account_id=$1`, sources["checking"])
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(q))
	requireCode(t, err, "P2503")
}

// TestAggregatePrincipalDeny validates current grants before pins, closed input, rollback and helper privileges.
func TestAggregatePrincipalDeny(t *testing.T) {
	c, db, sources, _ := activityOracle(t)
	revision := value(t, c, "SELECT revision::text FROM ledger.book")
	q := activityInput(sources["checking"])
	request := func(actorID string, v string) (string, error) {
		return invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", actorID, entity, book, v)
	}
	_, err := request(other, raw(q))
	requireCode(t, err, "P2501")
	q["source_account_id"] = sources["foreign"]
	_, err = request(reader, raw(q))
	requireCode(t, err, "P2501")
	q["source_account_id"] = sources["other-book"]
	_, err = request(reader, raw(q))
	requireCode(t, err, "P2501")
	q = activityInput(sources["checking"])
	for k, v := range map[string]any{"starts_on": "2026-02-30", "through_date": "2026-10-01", "search": strings.Repeat("x", 101), "limit": 21, "unknown": true, "profile": "other", "snapshot": map[string]any{}, "cursor": "zz"} {
		changed := activityInput(sources["checking"])
		changed[k] = v
		if k == "search" || k == "starts_on" || k == "through_date" {
			changed["cursor"] = nil
		}
		if _, err = request(reader, raw(changed)); err == nil {
			t.Fatal("malformed accepted", k)
		}
	}
	if _, err = request(reader, strings.TrimSuffix(raw(q), "}")+`,"search":"duplicate"}`); err == nil {
		t.Fatal("duplicate keys")
	}
	exec(t, c, `UPDATE banking.source_permission SET revoked_at=now() WHERE source_account_id=$1 AND principal_id=$2`, sources["checking"], reader)
	q["snapshot"] = map[string]any{"source_revision": "999", "binding_revision": "1", "review_revision_sum": "0", "filter_digest": strings.Repeat("0", 32)}
	_, err = request(reader, raw(q))
	requireCode(t, err, "P2501")
	if _, err = invoke(t, db, "synthetic_runtime", "SELECT count(*)::text FROM banking.activity_facts($1,$2,$3)", entity, book, sources["checking"]); err == nil {
		t.Fatal("helper directly executable")
	}
	for _, direction := range []string{"down", "up"} {
		sql, e := fs.ReadFile(migrations.Files, "000041_source_activity."+direction+".sql")
		if e != nil {
			t.Fatal(e)
		}
		exec(t, c, string(sql))
	}
	importNoFinance(t, c, revision)
}

// activityAuth is a synthetic transport adapter, never a managed identity acceptance claim.
type activityAuth struct{}

func (activityAuth) Authenticate(context.Context, string) (string, error) { return reader, nil }

// TestReadNeverPosts executes the real HTTP→Go→restricted SQL read with fictional identity.
func TestReadNeverPosts(t *testing.T) {
	c, db, sources, _ := activityOracle(t)
	revision := value(t, c, "SELECT revision::text FROM ledger.book")
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	h := ledger.Handler(activityAuth{}, &ledger.Service{DB: runtime})
	body := raw(activityInput(sources["checking"]))
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest("POST", "/v1/entities/"+entity+"/books/"+book+"/source-activity/query", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer synthetic")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), `"outflow": "940"`) && !strings.Contains(w.Body.String(), `"outflow":"940"`) {
			t.Fatal("real HTTP source read", w.Code, w.Body.String())
		}
	}
	importNoFinance(t, c, revision)
	if value(t, c, "SELECT count(*)::text FROM ledger.interactive_command") != "0" {
		t.Fatal("read registered/acknowledged a command")
	}
}

// TestWholeResultNotPage uses a separate 3000-row input recipe and fixed independent targets.
func TestWholeResultNotPage(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	for batch := 0; batch < 3; batch++ {
		rows := make([]map[string]any, 1000)
		for i := range rows {
			a := []string{"0.01", "0.02", "0.03"}[(batch*1000+i)%3]
			rows[i] = importRow(fmt.Sprintf("metric-%04d", batch*1000+i), a, i+2)
			rows[i]["canonical_amount"] = a
			rows[i]["booked_on"] = "2026-01-02"
			if batch*1000+i < 50 {
				rows[i]["booked_on"] = "2026-01-01"
			}
		}
		key := fmt.Sprintf("metric-volume-%d", batch)
		importFinish(t, db, key, importPublish(t, db, importRegister(t, db, sid, nid, key), importPublication(rows...)))
	}
	q := activityInput(sid)
	q["starts_on"], q["through_date"] = "2026-01-01", "2026-01-31"
	p := activityRead(t, db, q)
	if p["units"].([]any)[0].(map[string]any)["outflow"] != "60" {
		t.Fatal("page sum used as total")
	}
	q["kind"], q["currency_version_id"], q["snapshot"], q["limit"] = "observations", currency, p["snapshot"], 100
	revision := value(t, c, "SELECT revision::text FROM ledger.book")
	q["limit"] = 50
	visible := activityRead(t, db, q)
	sum := new(big.Rat)
	for _, v := range visible["items"].([]any) {
		a, ok := new(big.Rat).SetString(v.(map[string]any)["amount"].(string))
		if !ok {
			t.Fatal("not exact decimal")
		}
		sum.Add(sum, a)
	}
	if sum.Cmp(big.NewRat(99, 100)) != 0 {
		t.Fatal("independent first50 target", sum)
	}
	q["limit"] = 100
	seen := map[string]bool{}
	pages := 0
	totalRows := new(big.Rat)
	for {
		p = activityRead(t, db, q)
		if p["totals"].(map[string]any)["count"] != "3000" || p["totals"].(map[string]any)["outflow"] != "60" {
			t.Fatal("complete totals")
		}
		for _, v := range p["items"].([]any) {
			amount, ok := new(big.Rat).SetString(v.(map[string]any)["amount"].(string))
			if !ok {
				t.Fatal("not exact row decimal")
			}
			totalRows.Add(totalRows, amount)
			id := v.(map[string]any)["observation_id"].(string)
			if seen[id] {
				t.Fatal("duplicate")
			}
			seen[id] = true
		}
		pages++
		q["cursor"] = p["next_cursor"]
		if q["cursor"] == nil {
			break
		}
		if pages >= 30 {
			t.Fatal("missing end cursor")
		}
	}
	if len(seen) != 3000 || pages != 30 || totalRows.Cmp(big.NewRat(60, 1)) != 0 {
		t.Fatal("missing facts")
	}
	importNoFinance(t, c, revision)
}

// TestSourceRevisionSnapshot rejects changed selectors/anchors and mutable review pins.
func TestSourceRevisionSnapshot(t *testing.T) {
	c, db, sources, units := activityOracle(t)
	q := activityInput(sources["checking"])
	p := activityRead(t, db, q)
	q["snapshot"], q["kind"], q["currency_version_id"], q["limit"] = p["snapshot"], "observations", units["usd-before"], 1
	page := activityRead(t, db, q)
	q["cursor"] = page["next_cursor"]
	for k, v := range map[string]any{"search": "other", "starts_on": "2026-11-02", "through_date": "2026-11-29", "currency_version_id": units["usd-after"], "merchant": map[string]any{"status": "not_provided", "label": nil}} {
		changed := map[string]any{}
		for key, value := range q {
			changed[key] = value
		}
		changed[k] = v
		_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(changed))
		want := "P2500"
		if k == "search" || k == "starts_on" || k == "through_date" {
			want = "P2503"
		}
		requireCode(t, err, want)
	}
	decoded, err := hex.DecodeString(q["cursor"].(string))
	if err != nil {
		t.Fatal(err)
	}
	var cursor map[string]any
	if err = json.Unmarshal(decoded, &cursor); err != nil {
		t.Fatal(err)
	}
	cursor["anchor"] = activityUUID(999999)
	q["cursor"] = hex.EncodeToString([]byte(raw(cursor)))
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(q))
	requireCode(t, err, "P2500")
	q["cursor"] = nil
	exec(t, c, `UPDATE banking.review_item SET review_revision=review_revision+1,state='correction_hold' WHERE review_id=$1`, activityUUID(9001))
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(q))
	requireCode(t, err, "P2503")
}

// TestFreshnessUnknownStale keeps empty inventory, unavailable balance/sync and binding pins distinct.
func TestFreshnessUnknownStale(t *testing.T) {
	c, db, sid, _ := importSetup(t)
	q := activityInput(sid)
	p := activityRead(t, db, q)
	if p["inventory"].(map[string]any)["source_count"] != "0" || len(p["units"].([]any)) != 0 || p["available_from"] != nil || p["bank_balance"] != nil || p["last_successful_sync"] != nil || p["history_completeness"] != "unknown" {
		t.Fatal("unknown/empty became a bank balance")
	}
	q["snapshot"] = p["snapshot"]
	exec(t, c, `UPDATE banking.source_account SET binding_revision=binding_revision+1 WHERE source_account_id=$1`, sid)
	_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(q))
	requireCode(t, err, "P2503")
}
