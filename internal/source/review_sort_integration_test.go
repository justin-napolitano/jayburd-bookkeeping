//go:build integration

package source

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

// TestSourceReviewSortedPages uses fixed independent expectations, real restricted logins and tiny pages.
func TestSourceReviewSortedPages(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	rows := []map[string]any{}
	for i, f := range []struct{ name, date, direction, amount string }{
		{"Alpha", "2026-01-03", "outflow", "10"},
		{"Bravo", "2026-01-01", "inflow", "2"},
		{"Charlie", "2026-01-02", "outflow", "2"},
		{"Delta", "2026-01-02", "inflow", "100"},
		{"Echo", "2026-01-03", "outflow", "10"},
	} {
		row := importRow(strings.ToLower(f.name), f.amount, i+2)
		row["booked_on"], row["description"], row["direction"], row["canonical_amount"] = f.date, f.name, f.direction, f.amount
		rows = append(rows, row)
	}
	batch := importRegister(t, db, sid, nid, "sort-fixture")
	importFinish(t, db, "sort-import", importPublish(t, db, batch, importPublication(rows...)))
	ids := map[string]string{}
	legacy, err := reviewList(t, db, sid, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range importObject(t, legacy)["items"].([]any) {
		r := x.(map[string]any)
		ids[r["description"].(string)] = r["review_id"].(string)
	}
	// A real reviewed draft supplies a distinct state without fabricating a missing draft.
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'prepare',NULL),($1,$2,'post',NULL)`, entity, reader)
	exec(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())`, entity, book, reviewPolicy, currency, actor)
	exec(t, c, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000061','2026-01-01','2026-02-01')`, entity, book)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'6000','Fictional supplies','expense','debit',false,'2026-01-01',NULL)`, entity, book, reviewExpense)
	if _, err := mutate(t, db, sid, "set_source_review_permission", "review-grant", map[string]any{"expected_binding_revision": "2", "principal_id": reader, "enabled": true}); err != nil {
		t.Fatal(err)
	}
	reviewDraft(t, db, ids["Alpha"], "sort-real-draft", reviewExpense)
	// Every tie resolves by UUID ascending in either direction, independent of input order.
	tie := func(names ...string) []string {
		sort.Slice(names, func(i, j int) bool { return ids[names[i]] < ids[names[j]] })
		return names
	}
	expectations := map[string]map[string][]string{
		"date":        {"asc": append(append([]string{"Bravo"}, tie("Charlie", "Delta")...), tie("Alpha", "Echo")...), "desc": append(append(tie("Alpha", "Echo"), tie("Charlie", "Delta")...), "Bravo")},
		"description": {"asc": {"Alpha", "Bravo", "Charlie", "Delta", "Echo"}, "desc": {"Echo", "Delta", "Charlie", "Bravo", "Alpha"}},
		"amount":      {"asc": append(append(tie("Bravo", "Charlie"), tie("Alpha", "Echo")...), "Delta"), "desc": append(append([]string{"Delta"}, tie("Alpha", "Echo")...), tie("Bravo", "Charlie")...)},
		"direction":   {"asc": append(tie("Bravo", "Delta"), tie("Alpha", "Charlie", "Echo")...), "desc": append(tie("Alpha", "Charlie", "Echo"), tie("Bravo", "Delta")...)},
		"status":      {"asc": append([]string{"Alpha"}, tie("Bravo", "Charlie", "Delta", "Echo")...), "desc": append(tie("Bravo", "Charlie", "Delta", "Echo"), "Alpha")},
	}
	request := func(key, order string, cursor any) (string, error) {
		return invoke(t, db, "synthetic_runtime", "SELECT banking.list_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(map[string]any{"source_account_id": sid, "state": nil, "limit": 2, "cursor": cursor, "sort": key, "order": order}))
	}
	var firstCursor string
	started := time.Now()
	for key, orders := range expectations {
		for order, want := range orders {
			t.Run(key+"/"+order, func(t *testing.T) {
				var cursor any
				got := []string{}
				seen := map[string]bool{}
				for page := 0; page < 4; page++ {
					result, err := request(key, order, cursor)
					if err != nil {
						t.Fatal(err)
					}
					p := importObject(t, result)
					if p["matching_count"] != "5" || p["sort"] != key || p["order"] != order {
						t.Fatal("sorted metadata")
					}
					for _, x := range p["items"].([]any) {
						r := x.(map[string]any)
						id := r["review_id"].(string)
						if seen[id] {
							t.Fatal("duplicated page row")
						}
						seen[id] = true
						got = append(got, r["description"].(string))
					}
					cursor = p["next_cursor"]
					if cursor == nil {
						break
					}
					encoded := cursor.(string)
					decoded, e := hex.DecodeString(encoded)
					if e != nil {
						t.Fatal(e)
					}
					var wire map[string]any
					if json.Unmarshal(decoded, &wire) != nil || len(wire) != 8 {
						t.Fatal("cursor shape")
					}
					for _, private := range []string{"description", "amount", "booked_date", "currency"} {
						if _, exists := wire[private]; exists {
							t.Fatal("private cursor field")
						}
					}
					if key == "date" && order == "desc" && firstCursor == "" {
						firstCursor = encoded
					}
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("want %v got %v", want, got)
				}
			})
		}
	}
	t.Logf("ten complete five-row/two-per-page orders: %s", time.Since(started))
	if firstCursor == "" {
		t.Fatal("missing cursor")
	}
	empty := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.list_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(map[string]any{"source_account_id": sid, "state": "posted", "limit": 2, "cursor": nil, "sort": "date", "order": "desc"})))
	if empty["matching_count"] != "0" || empty["start_index"] != "0" || empty["next_cursor"] != nil || len(empty["items"].([]any)) != 0 {
		t.Fatal("empty filtered page")
	}
	for _, state := range []any{"needs_review", "draft_ready"} {
		if _, err := invoke(t, db, "synthetic_runtime", "SELECT banking.list_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(map[string]any{"source_account_id": sid, "state": state, "limit": 2, "cursor": firstCursor, "sort": "date", "order": "desc"})); err == nil {
			t.Fatal("cursor filter substitution")
		}
	}
	if _, err := invoke(t, db, "synthetic_runtime", "SELECT banking.list_review_items($1,$2,$3,$4)::text", other, entity, book, raw(map[string]any{"source_account_id": sid, "state": nil, "limit": 2, "cursor": nil, "sort": "date", "order": "desc"})); err == nil {
		t.Fatal("source permission bypass")
	}
	for _, tc := range []struct {
		key, order string
		cursor     any
	}{{"amount", "desc", firstCursor}, {"date", "asc", firstCursor}, {"bogus", "asc", nil}, {"date", "bogus", nil}, {"date", "desc", "00"}} {
		if _, err := request(tc.key, tc.order, tc.cursor); err == nil {
			t.Fatal("invalid selector/cursor accepted")
		}
	}
	wire, _ := hex.DecodeString(firstCursor)
	var forged map[string]any
	_ = json.Unmarshal(wire, &forged)
	for _, field := range []string{"entity_id", "book_id", "source_account_id", "last_id"} {
		copy := map[string]any{}
		for k, v := range forged {
			copy[k] = v
		}
		copy[field] = "00000000-0000-4000-8000-999999999999"
		b, _ := json.Marshal(copy)
		if _, err := request("date", "desc", hex.EncodeToString(b)); err == nil {
			t.Fatal("foreign/missing anchor accepted", field)
		}
	}
	if _, err := invoke(t, db, "synthetic_runtime", "SELECT banking.sorted_review_page($1,$2,$3,$4)", reader, entity, book, "{}"); err == nil {
		t.Fatal("private helper exposed")
	}
	exec(t, c, "UPDATE banking.source_account SET source_revision=source_revision+1 WHERE source_account_id=$1", sid)
	if _, err := request("date", "desc", firstCursor); err == nil {
		t.Fatal("stale cursor accepted")
	}
	exec(t, c, "UPDATE banking.source_permission SET revoked_at=now() WHERE source_account_id=$1 AND principal_id=$2 AND capability='read'", sid, reader)
	if _, err := request("date", "desc", nil); err == nil {
		t.Fatal("revoked source read accepted")
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("read created journal")
	}
}

// TestSourceReviewSortAmbiguousFact proves the bounded lookup rejects rather than truncates a multi-fact group.
func TestSourceReviewSortAmbiguousFact(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	batch := importRegister(t, db, sid, nid, "ambiguous")
	importFinish(t, db, "ambiguous", importPublish(t, db, batch, importPublication(importRow("original", "12.50", 2))))
	old := value(t, c, "SELECT observation_id::text FROM banking.source_observation")
	extra := "00000000-0000-4000-8000-000000000901"
	exec(t, c, "BEGIN")
	exec(t, c, `INSERT INTO banking.source_observation SELECT entity_id,book_id,$1,source_account_id,namespace_id,'ambiguous-extra',current_revision,created_at FROM banking.source_observation WHERE observation_id=$2`, extra, old)
	exec(t, c, `INSERT INTO banking.observation_revision SELECT entity_id,book_id,$1,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields,recorded_at FROM banking.observation_revision WHERE observation_id=$2`, extra, old)
	exec(t, c, `INSERT INTO banking.review_evidence SELECT entity_id,book_id,decision_id,$1,review_id,1 FROM banking.review_assignment WHERE observation_id=$2`, extra, old)
	exec(t, c, `INSERT INTO banking.review_assignment SELECT entity_id,book_id,$1,review_id,decision_id FROM banking.review_assignment WHERE observation_id=$2`, extra, old)
	exec(t, c, "COMMIT")
	_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.list_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(map[string]any{"source_account_id": sid, "state": nil, "limit": 2, "cursor": nil, "sort": "date", "order": "desc"}))
	requireCode(t, err, "P2502")
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.query_source_activity($1,$2,$3,$4)::text", reader, entity, book, raw(activityInput(sid)))
	requireCode(t, err, "P2502")
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("ambiguous read posted")
	}
}

// TestSourceReviewSortMigrationRoundTrip preserves populated legacy projections when39 is reversed synthetically.
func TestSourceReviewSortMigrationRoundTrip(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	batch := importRegister(t, db, sid, nid, "round-trip")
	importFinish(t, db, "round-trip", importPublish(t, db, batch, importPublication(importRow("round-trip", "12.50", 2))))
	before, err := reviewList(t, db, sid, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, direction := range []string{"down", "up"} {
		bytes, err := fs.ReadFile(migrations.Files, "000039_review_sort."+direction+".sql")
		if err != nil {
			t.Fatal(err)
		}
		exec(t, c, string(bytes))
		after, err := reviewList(t, db, sid, 100, nil)
		if err != nil || after != before {
			t.Fatal("legacy projection/history changed", direction, err)
		}
		exists := value(t, c, "SELECT (to_regprocedure('banking.sorted_review_page(uuid,uuid,uuid,text)') IS NOT NULL)::text")
		if exists != fmt.Sprint(direction == "up") {
			t.Fatal("helper reversal")
		}
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("round-trip posted")
	}
}

// TestSourceReviewSortPagination3000 checks one full source snapshot, not three unrelated first pages.
func TestSourceReviewSortPagination3000(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	for batch := 0; batch < 3; batch++ {
		key := fmt.Sprintf("sort-volume-%d", batch)
		rows := make([]map[string]any, 1000)
		for i := range rows {
			rows[i] = importRow(fmt.Sprintf("entry-%04d", batch*1000+i), "12.50", i+2)
		}
		registered := importRegister(t, db, sid, nid, key)
		importFinish(t, db, key, importPublish(t, db, registered, importPublication(rows...)))
	}
	started := time.Now()
	seen := map[string]bool{}
	var cursor any
	pages := 0
	for {
		p := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.list_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(map[string]any{"source_account_id": sid, "state": nil, "limit": 100, "cursor": cursor, "sort": "date", "order": "desc"})))
		if p["matching_count"] != "3000" || p["start_index"] != fmt.Sprint(pages*100+1) {
			t.Fatal("volume count/index")
		}
		rows := p["items"].([]any)
		if len(rows) != 100 {
			t.Fatal("page bound")
		}
		for _, v := range rows {
			id := v.(map[string]any)["review_id"].(string)
			if seen[id] {
				t.Fatal("duplicate volume row")
			}
			seen[id] = true
		}
		pages++
		cursor = p["next_cursor"]
		if cursor == nil {
			break
		}
		if pages >= 30 {
			t.Fatal("end cursor missing")
		}
	}
	if len(seen) != 3000 || pages != 30 {
		t.Fatal("lost volume rows")
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("volume read posted")
	}
	t.Logf("3000 rows / 30 unique snapshot pages: %s; disposable PostgreSQL17 container, no hosted latency claim", time.Since(started))
}

// TestSourceReviewSortOriginalCurrencies preserves minor units and never converts into the book currency.
func TestSourceReviewSortOriginalCurrencies(t *testing.T) {
	for _, tc := range []struct{ code, id, small, large string }{{"JPY", "00000000-0000-4000-8000-000000000006", "2", "10"}, {"KWD", "00000000-0000-4000-8000-000000000007", "2.001", "10.01"}} {
		t.Run(tc.code, func(t *testing.T) {
			c, db := fixture(t)
			input := manifest()
			input["currency_version_id"] = tc.id
			receipt := importObject(t, call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'sort-currency',$4)::text", actor, entity, book, raw(input)))
			sid, nid := receipt["source_account_id"].(string), receipt["namespace_id"].(string)
			if _, err := mutate(t, db, sid, "set_source_import_permission", "grant", map[string]any{"expected_binding_revision": "1", "principal_id": reader, "enabled": true, "entity_import_enabled": true}); err != nil {
				t.Fatal(err)
			}
			rows := []map[string]any{}
			for i, amount := range []string{tc.large, tc.small} {
				r := importRow("entry-"+strings.ReplaceAll(amount, ".", "_"), amount, i+2)
				r["currency"], r["currency_version_id"], r["canonical_amount"] = tc.code, tc.id, amount
				rows = append(rows, r)
			}
			batch := importRegister(t, db, sid, nid, "sort-currency")
			importFinish(t, db, "sort-currency", importPublish(t, db, batch, importPublication(rows...)))
			result := call(t, db, "synthetic_runtime", "SELECT banking.list_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(map[string]any{"source_account_id": sid, "state": nil, "limit": 2, "cursor": nil, "sort": "amount", "order": "asc"}))
			items := importObject(t, result)["items"].([]any)
			query := searchInput(sid)
			query["sort"] = "amount"
			searched := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.search_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(query)))["items"].([]any)
			for i, want := range []string{tc.small, tc.large} {
				r := items[i].(map[string]any)
				if r["currency"] != tc.code || r["amount"] != want {
					t.Fatal("original currency/precision lost", r)
				}
				original := searched[i].(map[string]any)
				if original["currency"] != tc.code || original["amount"] != want {
					t.Fatal("search converted original units")
				}
			}
			if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
				t.Fatal("sort posted")
			}
		})
	}
}
