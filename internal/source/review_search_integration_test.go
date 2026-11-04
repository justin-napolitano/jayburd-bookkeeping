//go:build integration

package source

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

// searchInput keeps transport selectors explicit; expected fixture outcomes are independently fixed below.
func searchInput(sid string) map[string]any {
	return map[string]any{"source_account_id": sid, "state": nil, "limit": 2, "cursor": nil, "sort": "date", "order": "asc", "search": "", "starts_on": nil, "through_date": nil}
}

// TestSourceReviewSearch proves literal/date matching, scope, snapshot, legacy and reversible read DDL.
func TestSourceReviewSearch(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	empty := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.search_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(searchInput(sid))))
	if empty["source_count"] != "0" || empty["matching_count"] != "0" || empty["available_from"] != nil || empty["available_through"] != nil {
		t.Fatal("empty source invents coverage")
	}
	rows := []map[string]any{}
	for i, f := range []struct{ name, day, amount string }{{"Alpha supplies", "2026-01-01", "2"}, {"ALPHA refund", "2026-01-02", "10"}, {"100%_Store", "2026-01-02", "12.50"}, {"Later alpha", "2026-01-03", "0.01"}} {
		r := importRow(fmt.Sprint("search-", i), f.amount, i+2)
		r["booked_on"], r["description"], r["canonical_amount"] = f.day, f.name, f.amount
		if f.amount == "12.50" {
			r["canonical_amount"] = "12.5"
		}
		rows = append(rows, r)
	}
	batch := importRegister(t, db, sid, nid, "search-fixture")
	importFinish(t, db, "search-import", importPublish(t, db, batch, importPublication(rows...)))
	request := func(v map[string]any) (string, error) {
		return invoke(t, db, "synthetic_runtime", "SELECT banking.search_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(v))
	}
	_, denied := invoke(t, db, "synthetic_runtime", "SELECT banking.search_review_items($1,$2,$3,$4)::text", other, entity, book, raw(searchInput(sid)))
	requireCode(t, denied, "P2501")
	v := searchInput(sid)
	v["search"] = "alpha"
	v["starts_on"] = "2026-01-02"
	v["through_date"] = "2026-01-03"
	v["limit"] = 1
	first, err := request(v)
	if err != nil {
		t.Fatal(err)
	}
	p := importObject(t, first)
	if p["matching_count"] != "2" || p["source_count"] != "4" || p["available_from"] != "2026-01-01" || p["available_through"] != "2026-01-03" || p["items"].([]any)[0].(map[string]any)["description"] != "ALPHA refund" {
		t.Fatal("independent matching/coverage fixture")
	}
	cursor := p["next_cursor"]
	v["cursor"] = cursor
	second, err := request(v)
	if err != nil {
		t.Fatal(err)
	}
	p = importObject(t, second)
	if p["start_index"] != "2" || p["next_cursor"] != nil || p["items"].([]any)[0].(map[string]any)["amount"] != "0.01" {
		t.Fatal("second matching row/original amount")
	}
	for _, key := range []string{"search", "starts_on", "through_date", "sort", "order", "state", "source_account_id"} {
		changed := searchInput(sid)
		for k, x := range v {
			changed[k] = x
		}
		switch key {
		case "search":
			changed[key] = "Alpha"
		case "starts_on":
			changed[key] = "2026-01-01"
		case "through_date":
			changed[key] = "2026-01-04"
		case "sort":
			changed[key] = "amount"
		case "order":
			changed[key] = "desc"
		case "state":
			changed[key] = "needs_review"
		default:
			changed[key] = "00000000-0000-4000-8000-000000000999"
		}
		if _, err := request(changed); err == nil {
			t.Fatalf("changed cursor selector accepted: %s", key)
		}
	}
	decoded, _ := hex.DecodeString(cursor.(string))
	var cur map[string]any
	_ = json.Unmarshal(decoded, &cur)
	for _, key := range []string{"entity_id", "book_id", "last_id"} {
		changed := map[string]any{}
		for k, x := range cur {
			changed[k] = x
		}
		changed[key] = "00000000-0000-4000-8000-000000000999"
		v["cursor"] = hex.EncodeToString([]byte(raw(changed)))
		if _, err := request(v); err == nil {
			t.Fatal("foreign/missing anchor accepted", key)
		}
	}
	for _, tc := range []struct {
		search string
		count  string
	}{{"%_", "1"}, {"' OR 1=1 --", "0"}, {"absent", "0"}, {"", "4"}} {
		q := searchInput(sid)
		q["search"] = tc.search
		out, err := request(q)
		if err != nil {
			t.Fatal(err)
		}
		page := importObject(t, out)
		if page["matching_count"] != tc.count || page["source_count"] != "4" {
			t.Fatal("literal search/empty coverage", tc.search)
		}
	}
	for _, tc := range []struct {
		key   string
		value any
	}{{"starts_on", "2026-02-30"}, {"through_date", "0000-01-01"}, {"search", nil}, {"search", strings.Repeat("x", 101)}, {"unknown", true}, {"limit", 101}, {"limit", 1.5}, {"sort", "random"}} {
		q := searchInput(sid)
		q[tc.key] = tc.value
		if _, err := request(q); err == nil {
			t.Fatal("invalid query accepted", tc.key)
		}
	}
	q := searchInput(sid)
	q["starts_on"] = "2026-01-03"
	q["through_date"] = "2026-01-01"
	if _, err := request(q); err == nil {
		t.Fatal("reversed dates")
	}
	if _, err := invoke(t, db, "synthetic_runtime", "SELECT banking.search_review_items($1,$2,$3,$4)::text", reader, entity, book, strings.TrimSuffix(raw(searchInput(sid)), "}")+`,"search":"duplicate"}`); err == nil {
		t.Fatal("duplicate JSON accepted")
	}
	legacy, err := reviewList(t, db, sid, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, direction := range []string{"down", "up"} {
		sql, err := fs.ReadFile(migrations.Files, "000040_review_search."+direction+".sql")
		if err != nil {
			t.Fatal(err)
		}
		exec(t, c, string(sql))
		after, err := reviewList(t, db, sid, 100, nil)
		if err != nil || after != legacy {
			t.Fatal("legacy changed across read DDL", err)
		}
	}
	v["cursor"] = cursor
	exec(t, c, `UPDATE banking.source_account SET source_revision=source_revision+1 WHERE source_account_id=$1`, sid)
	_, err = request(v)
	requireCode(t, err, "P2503")
	exec(t, c, `UPDATE banking.source_permission SET revoked_at=now() WHERE source_account_id=$1 AND principal_id=$2 AND capability='read'`, sid, reader)
	_, err = request(searchInput(sid))
	requireCode(t, err, "P2501")
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("search posted")
	}
}

// TestSourceReviewSearch3000 follows every page of a filtered cold-statistics source snapshot.
func TestSourceReviewSearch3000(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	for batch := 0; batch < 3; batch++ {
		key := fmt.Sprintf("search-volume-%d", batch)
		rows := make([]map[string]any, 1000)
		for i := range rows {
			rows[i] = importRow(fmt.Sprintf("search-%04d", batch*1000+i), "12.50", i+2)
			rows[i]["description"] = "Fictional merchant"
			rows[i]["booked_on"] = "2026-01-02"
		}
		registered := importRegister(t, db, sid, nid, key)
		importFinish(t, db, key, importPublish(t, db, registered, importPublication(rows...)))
	}
	q := searchInput(sid)
	q["limit"] = 100
	q["search"] = "MERCHANT"
	q["starts_on"] = "2026-01-02"
	q["through_date"] = "2026-01-02"
	started := time.Now()
	seen := map[string]bool{}
	pages := 0
	for {
		p := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.search_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(q)))
		if p["matching_count"] != "3000" || p["source_count"] != "3000" || p["start_index"] != fmt.Sprint(pages*100+1) {
			t.Fatal("volume count")
		}
		items := p["items"].([]any)
		if len(items) != 100 {
			t.Fatal("page bound")
		}
		for _, item := range items {
			id := item.(map[string]any)["review_id"].(string)
			if seen[id] {
				t.Fatal("duplicate search page")
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
	if len(seen) != 3000 || pages != 30 || value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("lost rows or read wrote journal")
	}
	t.Logf("3000 matching rows,30 unique pages: %s; synthetic PostgreSQL17, no hosted scale claim", time.Since(started))
}
