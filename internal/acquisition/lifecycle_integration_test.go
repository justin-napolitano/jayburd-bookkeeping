//go:build integration

package acquisition

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// lifecycleRow constructs fictional provider bytes, retaining numeric tokens before decoding.
func lifecycleRow(id, amount, day string, pending bool, parent *string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"account_id":"Account-A","transaction_id":%q,"amount":%s,"iso_currency_code":"USD","unofficial_currency_code":null,"date":%q,"pending":%t,"pending_transaction_id":%s,"name":%q}`, id, amount, day, pending, raw(parent), "Fictional "+id))
}

// lifecyclePage preserves separate provider operation arrays and explicit pagination.
func lifecyclePage(cursor string, more bool, added, modified []json.RawMessage, removed []string) []byte {
	removals := []map[string]string{}
	for _, id := range removed {
		removals = append(removals, map[string]string{"account_id": "Account-A", "transaction_id": id})
	}
	if added == nil {
		added = []json.RawMessage{}
	}
	if modified == nil {
		modified = []json.RawMessage{}
	}
	return []byte(raw(map[string]any{"accounts": []map[string]string{{"account_id": "Account-A", "type": "depository"}}, "added": added, "modified": modified, "removed": removals, "next_cursor": cursor, "has_more": more, "transactions_update_status": "HISTORICAL_UPDATE_COMPLETE", "request_id": "synthetic-lifecycle"}))
}

// TestAcquisitionLifecycleOracle compares real adapter/SQL publication to fixed independent expected facts.
func TestAcquisitionLifecycleOracle(t *testing.T) {
	owner, db, stream := setup(t)
	s := Scope{entity, book, stream["stream_id"].(string)}
	exec(t, owner, `INSERT INTO banking.source_permission VALUES($1,$2,$3,$4,'review',NULL)`, entity, book, sourceID, actor)
	parent := "Pending-A"
	pages := [][]byte{
		lifecyclePage("C1", false, []json.RawMessage{lifecycleRow("Pending-A", "10.00", "2026-01-02", true, nil), lifecycleRow("Hold-B", "9.50", "2026-01-02", true, nil), lifecycleRow("Correct-C", "-0.20", "2026-01-03", false, nil)}, nil, nil),
		lifecyclePage("C2", true, []json.RawMessage{lifecycleRow("Booked-A", "12.50", "2026-01-04", false, &parent), lifecycleRow("Unmatched-D", "-1.25e2", "2026-01-04", false, nil)}, nil, nil),
		lifecyclePage("C3", false, nil, []json.RawMessage{lifecycleRow("Correct-C", "-0.21", "2026-01-05", false, nil)}, []string{"Pending-A", "Hold-B"}),
	}
	n := 0
	runner, _, calls := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
		cursor := providerRequest(t, r)
		expected := []string{"", "C1", "C2"}
		if n >= len(pages) {
			t.Error("unexpected fetch")
			return
		}
		if n == 0 && cursor != nil || n > 0 && (cursor == nil || *cursor != expected[n]) {
			t.Error("wrong persisted cursor")
		}
		// The second update's first page cannot expose its replacement or advance C1.
		if n == 2 && (value(t, owner, "SELECT cursor_value FROM integration.sync_stream") != "C1" || value(t, owner, "SELECT count(*)::text FROM banking.source_observation") != "3") {
			t.Error("partial publication")
		}
		w.Write(pages[n])
		n++
	})
	if out, e := runner.Run(context.Background(), s, claimID); e != nil || out.ChangedCount != "3" {
		t.Fatal("initial lifecycle", out, e)
	}
	call(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'lifecycle')::text", entity, book, s.StreamID)
	second := "00000000-0000-4000-8000-000000000085"
	if out, e := runner.Run(context.Background(), s, second); e != nil || out.ChangedCount != "5" {
		t.Fatal("complete lifecycle", out, e)
	}
	if _, e := runner.Run(context.Background(), s, second); e != nil || *calls != 3 {
		t.Fatal("terminal replay fetched", e)
	}
	// Literal expectations are independent of the fixture constructors and decoder.
	want := []string{
		"Booked-A|1|booked|2026-01-04|USD|outflow|12.5|12.50|overlap_hold",
		"Correct-C|2|booked|2026-01-05|USD|inflow|0.21|-0.21|correction_hold",
		"Hold-B|2|removed|2026-01-02|USD|outflow|9.5|9.50|correction_hold",
		"Pending-A|2|removed|2026-01-02|USD|outflow|10|10.00|correction_hold",
		"Unmatched-D|1|booked|2026-01-04|USD|inflow|125|-1.25e2|overlap_hold",
	}
	got := value(t, owner, `SELECT string_agg(o.external_entry_id||'|'||q.revision::text||'|'||q.source_state||'|'||q.booked_date::text||'|'||c.alphabetic_code||'|'||q.direction||'|'||trim_scale(q.amount)::text||'|'||q.provider_amount_lexeme||'|'||r.state,E'\n' ORDER BY o.external_entry_id) FROM banking.source_observation o JOIN banking.observation_revision q ON (q.entity_id,q.book_id,q.observation_id,q.revision)=(o.entity_id,o.book_id,o.observation_id,o.current_revision) JOIN reference.currency c USING(currency_version_id) JOIN banking.review_assignment a ON (a.entity_id,a.book_id,a.observation_id)=(o.entity_id,o.book_id,o.observation_id) JOIN banking.review_item r ON (r.entity_id,r.book_id,r.review_id)=(a.entity_id,a.book_id,a.review_id)`)
	if got != strings.Join(want, "\n") {
		t.Fatalf("fixed lifecycle oracle mismatch: %s", got)
	}
	for query, expected := range map[string]string{
		"SELECT cursor_value FROM integration.sync_stream":         "C3",
		"SELECT count(*)::text FROM banking.observation_revision":  "8",
		"SELECT count(*)::text FROM banking.provider_pending_link": "1",
		`SELECT q.provider_amount_lexeme||'|'||trim_scale(q.amount)::text||'|'||q.booked_date::text||'|'||q.source_state FROM banking.observation_revision q JOIN banking.source_observation o USING(entity_id,book_id,observation_id) WHERE o.external_entry_id='Correct-C' AND q.revision=1`: "-0.20|0.2|2026-01-03|booked",
		"SELECT count(*)::text FROM ledger.journal_entry":       "0",
		"SELECT count(*)::text FROM integration.outbox_message": "0",
		`SELECT p.external_entry_id||'|'||b.external_entry_id FROM banking.provider_pending_link l JOIN banking.source_observation p ON p.observation_id=l.pending_observation_id JOIN banking.source_observation b ON b.observation_id=l.booked_observation_id`: "Pending-A|Booked-A",
	} {
		if value(t, owner, query) != expected {
			t.Fatal("lifecycle aggregate mismatch")
		}
	}
	// The API projection must expose the current exact correction, not its original revision.
	review := value(t, owner, `SELECT a.review_id::text FROM banking.review_assignment a JOIN banking.source_observation o USING(entity_id,book_id,observation_id) WHERE o.external_entry_id='Correct-C'`)
	detail := object(t, call(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", actor, entity, book, review))
	evidence := detail["evidence"].([]any)[0].(map[string]any)
	if evidence["amount"] != "0.21" || evidence["booked_date"] != "2026-01-05" || evidence["observation_revision"] != "2" || detail["state"] != "correction_hold" {
		t.Fatal("review projection differs from independent correction oracle")
	}
}

// TestAcquisitionPageFaultPreservesPublication injects second-page faults after a known completed cursor.
func TestAcquisitionPageFaultPreservesPublication(t *testing.T) {
	for _, kind := range []string{"rate_limit", "institution_unavailable", "malformed", "bound_overflow", "timeout"} {
		t.Run(kind, func(t *testing.T) {
			owner, db, stream := setup(t)
			s := Scope{entity, book, stream["stream_id"].(string)}
			n := 0
			runner, _, calls := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
				providerRequest(t, r)
				n++
				switch n {
				case 1:
					w.Write(fixturePage("Original", "1.01", "USD", "C0", false))
				case 2:
					w.Write(fixturePage("Unpublished", "2.02", "USD", "C1", true))
				case 3:
					switch kind {
					case "timeout":
						<-r.Context().Done()
					case "rate_limit":
						w.Header().Set("Retry-After", "60")
						w.WriteHeader(429)
						w.Write([]byte(`{"error_code":"RATE_LIMIT_EXCEEDED"}`))
					case "institution_unavailable":
						w.WriteHeader(503)
						w.Write([]byte(`{"error_code":"INSTITUTION_DOWN"}`))
					case "malformed":
						w.Write([]byte(`{"added":`))
					case "bound_overflow":
						w.Write([]byte(strings.Repeat(" ", MaxRawBytes+1)))
					}
				default:
					t.Error("automatic refetch")
				}
			})
			if _, e := runner.Run(context.Background(), s, claimID); e != nil {
				t.Fatal(e)
			}
			if kind == "timeout" {
				runner.provider.(*PlaidHTTP).client.Timeout = 250 * time.Millisecond
			}
			call(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'fault')::text", entity, book, s.StreamID)
			second := "00000000-0000-4000-8000-000000000085"
			if _, e := runner.Run(context.Background(), s, second); e == nil {
				t.Fatal("fault accepted")
			}
			if *calls != 3 {
				t.Fatal("fault automatically retried")
			}
			for query, want := range map[string]string{"SELECT cursor_value FROM integration.sync_stream": "C0", "SELECT count(*)::text FROM banking.source_observation": "1", "SELECT count(*)::text FROM banking.observation_revision": "1", "SELECT count(*)::text FROM ledger.journal_entry": "0", "SELECT count(*)::text FROM integration.outbox_message": "0"} {
				if value(t, owner, query) != want {
					t.Fatal("failed update changed completed publication")
				}
			}
		})
	}
}
