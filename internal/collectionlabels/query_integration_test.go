//go:build integration

package collectionlabels

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// query45Cleanup removes only this test's fictional clone after all its callers return.
func query45Cleanup(t *testing.T, owner *pgx.Conn, db string) {
	t.Helper()
	t.Cleanup(func() {
		_ = owner.Close(context.Background())
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
		if e != nil {
			t.Error("owned query fixture cleanup unavailable")
			return
		}
		defer c.Close(ctx)
		if _, e = c.Exec(ctx, "DROP DATABASE "+pgx.Identifier{db}.Sanitize()); e != nil {
			t.Error("owned query fixture cleanup failed")
		}
	})
}

// query45Selector is the independently authored complete v3 selector.
func query45Selector() map[string]any {
	v := tagrecSelector()
	v["profile"] = "private-collection-review-v3"
	v["merchant"] = map[string]any{"kind": "all"}
	v["amount"] = nil
	v["institution"] = map[string]any{"kind": "all"}
	v["legal_owner"] = map[string]any{"kind": "all"}
	v["book"] = map[string]any{"kind": "all"}
	v["status"] = map[string]any{"source": "all", "journal": "all"}
	v["sort"] = map[string]any{"field": "date", "order": "desc"}
	return v
}
func query45Call(t testing.TB, db, w, who string, v map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.query_collection_review_v3($1,$2,$3)", who, w, raw(v))
}
func query45Page(t testing.TB, db, w string, v map[string]any) map[string]any {
	t.Helper()
	r, e := query45Call(t, db, w, actor, v)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func query45Catalog(t testing.TB, db, w string) map[string]any {
	t.Helper()
	r, e := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.get_collection_taxonomy_v3($1,$2)", actor, w)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func query45Detail(t testing.TB, db, w string, snapshot any, row map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.get_collection_detail($1,$2,$3)", actor, w, raw(map[string]any{"profile": "private-collection-detail-v1", "snapshot": snapshot, "source_id": row["source_id"], "observation_id": row["observation_id"], "observation_revision": row["observation_revision"]}))
}

// TestQuery45OriginalUnits tests exact large magnitudes, literal merchants, held money and complete choices.
func TestQuery45OriginalUnits(t *testing.T) {
	owner, db := fixture(t)
	query45Cleanup(t, owner, db)
	w := provision(t, db, actor)
	exec(t, owner, `INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000008','EUR',NULL,2,'2000-01-01',NULL,'synthetic')`)
	_, sources := register(t, db, w, actor, "query-units", 1)
	a := change("large", "9007199254740993.01", "USD")
	a["merchant_name"] = "Fictional 50%_Cafe"
	b := change("small", "0.10", "USD")
	b["merchant_name"] = nil
	b["source_state"] = "pending"
	c := change("euro", "0.10", "EUR")
	c["direction"] = "inflow"
	d := change("hold", "1.001", "USD")
	e := change("removed", "2.00", "USD")
	if _, err := publish(t, db, w, sources[0], "units", "0", nil, "c1", []map[string]any{a, b, c, d, e}); err != nil {
		t.Fatal(err)
	}
	if _, err := publish(t, db, w, sources[0], "remove", "1", "c1", "c2", []map[string]any{{"operation": "removed", "transaction_id": "removed"}}); err != nil {
		t.Fatal(err)
	}
	base := query45Page(t, db, w, query45Selector())
	if base["matching_count"] != "5" {
		t.Fatal("complete original set")
	}
	cat := query45Catalog(t, db, w)
	if cat["profile"] != "private-collection-catalog-v3" || len(cat["currencies"].([]any)) != 2 || len(cat["books"].([]any)) != 0 {
		t.Fatal("complete observed catalog")
	}
	if cat["institution"].(map[string]any)["state"] != "unavailable" || cat["legal_owner"].(map[string]any)["state"] != "unknown" {
		t.Fatal("invented provenance")
	}
	for _, x := range base["items"].([]any) {
		row := x.(map[string]any)
		if row["institution"].(map[string]any)["state"] != "unavailable" || row["legal_owner"].(map[string]any)["state"] != "unknown" {
			t.Fatal("inferred metadata")
		}
	}
	for _, tc := range []struct {
		name   string
		filter map[string]any
		count  string
	}{
		{"literal punctuation", map[string]any{"merchant": map[string]any{"kind": "contains", "value": "50%_"}}, "1"},
		{"case folded exact", map[string]any{"merchant": map[string]any{"kind": "exact", "value": "fictional 50%_cafe"}}, "1"},
		{"missing", map[string]any{"merchant": map[string]any{"kind": "missing"}}, "1"},
		{"pending", map[string]any{"status": map[string]any{"source": "pending", "journal": "all"}}, "1"},
		{"held", map[string]any{"status": map[string]any{"source": "held", "journal": "all"}}, "1"},
		{"native unassigned", map[string]any{"book": map[string]any{"kind": "unassigned"}}, "5"},
		{"not denied", map[string]any{"book": map[string]any{"kind": "unavailable"}}, "0"},
		{"large exact", map[string]any{"amount": map[string]any{"currency_version_id": currency, "minimum": "9007199254740993.01", "maximum": "9007199254740993.01", "direction": "outflow"}}, "1"},
		{"small USD excludes EUR and hold", map[string]any{"amount": map[string]any{"currency_version_id": currency, "minimum": "0.10", "maximum": "0.1000", "direction": "all"}}, "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := query45Selector()
			for k, x := range tc.filter {
				v[k] = x
			}
			r := query45Page(t, db, w, v)
			if r["matching_count"] != tc.count {
				t.Fatalf("wanted count %s", tc.count)
			}
		})
	}
	for _, bad := range []string{"-1", "+1", "01", "1e2", "NaN", "1.001", "100000000000000000000000000000"} {
		v := query45Selector()
		v["amount"] = map[string]any{"currency_version_id": currency, "minimum": bad, "maximum": nil, "direction": "all"}
		_, err := query45Call(t, db, w, actor, v)
		code(t, err, "P2800")
	}
	v := query45Selector()
	v["sort"] = map[string]any{"field": "amount", "order": "asc"}
	r := query45Page(t, db, w, v)
	rows := r["items"].([]any)
	if rows[0].(map[string]any)["money"].(map[string]any)["currency"] != "EUR" || rows[len(rows)-1].(map[string]any)["money"].(map[string]any)["state"] != "held" {
		t.Fatal("currency grouping or held placement")
	}
	v["sort"] = map[string]any{"field": "amount", "order": "desc"}
	r = query45Page(t, db, w, v)
	rows = r["items"].([]any)
	if rows[0].(map[string]any)["money"].(map[string]any)["amount"] != "9007199254740993.01" || rows[len(rows)-1].(map[string]any)["money"].(map[string]any)["state"] != "held" {
		t.Fatal("exact amount order")
	}
	detail, err := query45Detail(t, db, w, base["snapshot"], rows[0].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	if detail["links"].(map[string]any)["state"] != "unassigned" {
		t.Fatal("native journal invention")
	}
	_, err = query45Call(t, db, w, other, query45Selector())
	code(t, err, "P2801")
	if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, owner, "SELECT count(*)::text FROM banking.collection_label_command") != "0" {
		t.Fatal("read caused effects")
	}
}

// TestQuery45ClosedInputs checks strict JSON, helper privileges and immutable v2 compatibility.
func TestQuery45ClosedInputs(t *testing.T) {
	owner, db := fixture(t)
	query45Cleanup(t, owner, db)
	w := provision(t, db, actor)
	before := tagrecCatalog(t, db, w)
	_ = query45Catalog(t, db, w)
	after := tagrecCatalog(t, db, w)
	if raw(before) != raw(after) {
		t.Fatal("v3 changed v2 catalog")
	}
	for _, mutate := range []func(map[string]any){
		func(v map[string]any) {
			v["institution"] = map[string]any{"kind": "institution", "institution_id": evidence}
		},
		func(v map[string]any) { v["legal_owner"] = map[string]any{"kind": "entity", "entity_id": entityID} },
		func(v map[string]any) { v["merchant"] = map[string]any{"kind": "all", "value": "hidden"} },
		func(v map[string]any) {
			v["merchant"] = map[string]any{"kind": "exact", "value": strings.Repeat("界", 101)}
		},
		func(v map[string]any) { v["sort"] = map[string]any{"field": "date", "order": "desc", "hidden": true} },
		func(v map[string]any) { delete(v, "amount") },
		func(v map[string]any) { v["status"] = nil },
	} {
		v := query45Selector()
		mutate(v)
		_, e := query45Call(t, db, w, actor, v)
		code(t, e, "P2800")
	}
	v := raw(query45Selector())
	v = strings.Replace(v, `"profile":`, `"profile":"private-collection-review-v3","profile":`, 1)
	_, e := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.query_collection_review_v3($1,$2,$3)", actor, w, v)
	code(t, e, "P2800")
	_, e = call(t, db, "synthetic_runtime", "repeatable read", "SELECT to_jsonb(r)FROM banking.collection_query_rows($1,$2)r", actor, w)
	code(t, e, "42501")
	_, e = call(t, db, "synthetic_mixed", "repeatable read", "SELECT banking.get_collection_taxonomy_v3($1,$2)", actor, w)
	code(t, e, "P2801")
}

// TestQuery45ThreeThousand proves global merchant/amount filters and every sort on bounded server pages.
func TestQuery45ThreeThousand(t *testing.T) {
	owner, db := fixture(t)
	query45Cleanup(t, owner, db)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "query-pages", 1)
	var start any
	for batch := 0; batch < 3; batch++ {
		changes := []map[string]any{}
		for i := 0; i < 1000; i++ {
			n := batch*1000 + i
			z := change(fmt.Sprintf("q-%04d", n), fmt.Sprintf("%d.01", n), "USD")
			z["description"] = fmt.Sprintf("Fictional %04d", n)
			z["merchant_name"] = fmt.Sprintf("Fictional merchant %d", n%3)
			z["booked_date"] = fmt.Sprintf("2026-10-%02d", 1+n%28)
			changes = append(changes, z)
		}
		next := fmt.Sprintf("c%d", batch)
		if _, e := publish(t, db, w, sources[0], fmt.Sprint(batch), fmt.Sprint(batch), start, next, changes); e != nil {
			t.Fatal(e)
		}
		start = next
	}
	for _, field := range []string{"date", "account", "description", "merchant", "amount", "currency", "book", "source_state", "journal_state", "topic", "tags", "institution", "legal_owner"} {
		for _, order := range []string{"asc", "desc"} {
			v := query45Selector()
			v["sort"] = map[string]any{"field": field, "order": order}
			v["merchant"] = map[string]any{"kind": "exact", "value": "Fictional merchant 1"}
			v["amount"] = map[string]any{"currency_version_id": currency, "minimum": "1000.01", "maximum": "1999.01", "direction": "outflow"}
			v["limit"] = 100
			seen := map[string]bool{}
			pages := 0
			began := time.Now()
			for {
				r := query45Page(t, db, w, v)
				if r["matching_count"] != "334" {
					t.Fatal("filter was applied after page/count", field, order, r["matching_count"])
				}
				for _, x := range r["items"].([]any) {
					row := x.(map[string]any)
					id := row["observation_id"].(string)
					if seen[id] {
						t.Fatal("duplicate source row")
					}
					seen[id] = true
				}
				pages++
				if r["next_cursor"] == nil {
					break
				}
				v["cursor"] = r["next_cursor"]
				v["snapshot"] = r["snapshot"]
			}
			if len(seen) != 334 || pages != 4 {
				t.Fatal("incomplete global pages")
			}
			t.Logf("%s %s 334-of-3000 in %s", field, order, time.Since(began))
		}
	}
	v := query45Selector()
	v["limit"] = 1
	r := query45Page(t, db, w, v)
	v["cursor"] = r["next_cursor"]
	v["merchant"] = map[string]any{"kind": "missing"}
	_, e := query45Call(t, db, w, actor, v)
	code(t, e, "P2800")
	if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("query posted")
	}
}

// TestQuery45DetailAuthorization proves source detail can survive denied book metadata without leaked anchors.
func TestQuery45DetailAuthorization(t *testing.T) {
	owner, db, item := legacyFixture(t)
	query45Cleanup(t, owner, db)
	w := provision(t, db, actor)
	if _, e := adopt(t, db, w, item); e != nil {
		t.Fatal(e)
	}
	exec(t, owner, `INSERT INTO banking.review_item(entity_id,book_id,review_id,source_account_id,review_revision,state)VALUES($1,$2,$3,$4,1,'posted')`, entityID, bookID, evidence, sourceID)
	exec(t, owner, `INSERT INTO banking.review_decision(entity_id,book_id,decision_id,review_id,expected_review_revision,kind,actor_id,reason,body)VALUES($1,$2,$3,$3,0,'initialize',$4,'synthetic_import','{}')`, entityID, bookID, evidence, actor)
	exec(t, owner, `INSERT INTO banking.review_evidence(entity_id,book_id,decision_id,observation_id,review_id,linked_revision)VALUES($1,$2,$3,'00000000-0000-4000-8000-000000000100',$3,1)`, entityID, bookID, evidence)
	exec(t, owner, `INSERT INTO banking.review_assignment VALUES($1,$2,'00000000-0000-4000-8000-000000000100',$3,$3)`, entityID, bookID, evidence)
	exec(t, owner, `INSERT INTO banking.review_draft(entity_id,book_id,draft_id,review_id,review_revision,source_revision,binding_revision,book_revision,posting_body,normalized_request,created_by,idempotency_key)VALUES($1,$2,$3,$3,1,0,1,1,'{}','{}',$4,'query45-draft')`, entityID, bookID, evidence, actor)
	exec(t, owner, `INSERT INTO ledger.interactive_command(command_id,principal_id,registration_key,operation,entity_id,book_id,request_body,state,result,result_entity_id,result_book_id,terminal_at,acknowledged_at)VALUES($1,$2,'query45-command','post',$3,$4,'{}','committed','{}',$3,$4,clock_timestamp(),clock_timestamp())`, evidence, actor, entityID, bookID)
	exec(t, owner, `INSERT INTO banking.review_posting(entity_id,book_id,review_id,draft_id,command_id,journal_id,result)SELECT $1,$2,$3,$3,$3,journal_id,'{}'FROM ledger.journal_entry WHERE entity_id=$1 AND book_id=$2`, entityID, bookID, evidence)
	before := legacyHash(t, owner)
	cat := query45Catalog(t, db, w)
	if len(cat["books"].([]any)) != 1 {
		t.Fatal("authorized book missing")
	}
	v := query45Selector()
	v["book"] = map[string]any{"kind": "book", "entity_id": entityID, "book_id": bookID}
	p := query45Page(t, db, w, v)
	if p["matching_count"] != "2" {
		t.Fatal("exact book filter")
	}
	var linked map[string]any
	for _, x := range p["items"].([]any) {
		r := x.(map[string]any)
		if r["journal_state"] == "posted" {
			linked = r
		}
	}
	if linked == nil {
		t.Fatal("posted source missing")
	}
	d, e := query45Detail(t, db, w, p["snapshot"], linked)
	if e != nil {
		t.Fatal(e)
	}
	links := d["links"].(map[string]any)
	if links["state"] != "available" || len(links["journals"].([]any)) != 1 {
		t.Fatal("actual journal anchor missing")
	}
	anchor := links["journals"].([]any)[0].(map[string]any)
	if anchor["entity_id"] != entityID || anchor["book_id"] != bookID || anchor["source_revision"] != "1" {
		t.Fatal("anchor scope/revision")
	}
	if linked["legal_owner"].(map[string]any)["state"] != "unknown" {
		t.Fatal("book became asserted owner")
	}
	exec(t, owner, `UPDATE ledger.membership SET revoked_at=clock_timestamp()WHERE entity_id=$1 AND principal_id=$2 AND permission='read'`, entityID, actor)
	_, e = query45Detail(t, db, w, p["snapshot"], linked)
	code(t, e, "P2803")
	fresh := query45Page(t, db, w, query45Selector())
	linked = fresh["items"].([]any)[0].(map[string]any)
	d, e = query45Detail(t, db, w, fresh["snapshot"], linked)
	if e != nil {
		t.Fatal(e)
	}
	if d["links"].(map[string]any)["state"] != "unavailable" || len(d["links"].(map[string]any)["journals"].([]any)) != 0 || strings.Contains(raw(d), "Fictional legacy") {
		t.Fatal("denied book detail leaked")
	}
	if len(query45Catalog(t, db, w)["books"].([]any)) != 0 {
		t.Fatal("denied catalog book leaked")
	}
	_, e = query45Call(t, db, w, actor, v)
	code(t, e, "P2801")
	if legacyHash(t, owner) != before {
		t.Fatal("query/detail changed source/journal anchors")
	}
}
