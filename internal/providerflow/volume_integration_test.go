//go:build integration

package providerflow

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

// volumeStatement authors source evidence, not journals or expected posting output.
func volumeStatement() string {
	var b strings.Builder
	b.WriteString("source_account_ref,entry_id,booked_on,currency,direction,amount,description\r\n")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "synthetic,volume-%04d,2026-01-05,USD,outflow,0.10,Fictional volume purchase\r\n", i)
	}
	return b.String()
}

// volumeReviews collects current authorized identities before any draft changes the source snapshot.
func volumeReviews(t *testing.T, f restoreFlowFixture, state string) []string {
	t.Helper()
	base := "/v1/entities/" + entity + "/books/" + book + "/review-items?source_account_id=" + f.sourceID + "&state=" + state + "&limit=100"
	path := base
	seen := map[string]bool{}
	cursors := map[string]bool{}
	var ids []string
	for pageNumber := 0; ; pageNumber++ {
		if pageNumber >= 10 {
			t.Fatal("source review exceeded fixed ten-page workload")
		}
		page := request(t, f.app, "GET", path, "", nil, 200)
		var rows []struct {
			ID        string `json:"review_id"`
			Amount    string `json:"amount"`
			Currency  string `json:"currency"`
			Direction string `json:"direction"`
			State     string `json:"state"`
		}
		if json.Unmarshal(page["items"], &rows) != nil || len(rows) != 100 {
			t.Fatal("volume review page omitted source observations")
		}
		for _, row := range rows {
			if row.ID == "" || seen[row.ID] || row.Amount != "0.1" || row.Currency != "USD" || row.Direction != "outflow" || row.State != state {
				t.Fatal("volume source identity, original units or exact amount changed")
			}
			seen[row.ID] = true
			ids = append(ids, row.ID)
		}
		var cursor *string
		if json.Unmarshal(page["next_cursor"], &cursor) != nil {
			t.Fatal("invalid volume cursor")
		}
		if cursor == nil {
			break
		}
		if *cursor == "" || cursors[*cursor] {
			t.Fatal("repeated volume cursor")
		}
		cursors[*cursor] = true
		path = base + "&cursor=" + url.QueryEscape(*cursor)
	}
	if len(ids) != 1000 || len(cursors) != 9 {
		t.Fatal("volume source enumeration count changed")
	}
	return ids
}

// volumeReceipt checks immutable identity/body/result through replay, independently of acknowledgement state.
func volumeReceipt(t *testing.T, saved, replay map[string]json.RawMessage) {
	t.Helper()
	for _, key := range []string{"command_id", "operation", "entity_id", "book_id", "body", "result"} {
		if string(saved[key]) != string(replay[key]) || len(saved[key]) == 0 {
			t.Fatalf("source-backed replay changed %s", key)
		}
	}
	if string(replay["state"]) != `"committed"` {
		t.Fatal("source-backed replay was not committed")
	}
}

// TestSourceBackedPostingVolume1000 proves a fixed sequential application workload, not a throughput SLA.
// Owner setup is synthetic; every import/draft/posting mutation uses the actual restricted runtime APIs.
func TestSourceBackedPostingVolume1000(t *testing.T) {
	started := time.Now()
	f := restoreFixture(t, restoreNewDB(t), volumeStatement())
	if scalar(t, f.admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || scalar(t, f.admin, "SELECT count(*)::text FROM integration.outbox_message") != "0" {
		t.Fatal("imported artifacts posted accounting entries without review")
	}
	// The fixture inserts two accounts, one policy and one period; each setup trigger adds one revision.
	if scalar(t, f.admin, "SELECT revision::text FROM ledger.book") != "4" {
		t.Fatal("volume fixture must start at fixed setup revision 4")
	}
	ids := volumeReviews(t, f, "needs_review")
	commands := make([]map[string]json.RawMessage, 0, 1000)
	journalIDs := map[string]bool{}
	postingStarted := time.Now()
	for i, reviewID := range ids {
		prepared := restorePrepare(t, f, reviewID, fmt.Sprintf("volume-post-%04d", i))
		commandID := stringField(t, prepared["command_id"])
		if string(prepared["operation"]) != `"source_post"` || string(prepared["state"]) != `"prepared"` {
			t.Fatal("volume bypassed saved source review")
		}
		path := "/v1/commands/" + commandID
		saved := request(t, f.app, "POST", path+"/execute", "", map[string]any{}, 200)
		replay := request(t, f.app, "POST", path+"/execute", "", map[string]any{}, 200)
		volumeReceipt(t, saved, replay)
		var result struct {
			JournalID    string `json:"journal_id"`
			BookRevision string `json:"book_revision"`
			Journal      struct {
				BookRevision string `json:"book_revision"`
			} `json:"journal"`
		}
		if json.Unmarshal(saved["result"], &result) != nil || result.JournalID == "" || journalIDs[result.JournalID] {
			t.Fatal("distinct source rows did not create distinct linked entries")
		}
		if result.BookRevision != fmt.Sprint(i+5) || result.Journal.BookRevision != fmt.Sprint(i+5) {
			t.Fatal("source-backed receipt did not advance the fixed book revision exactly once")
		}
		journalIDs[result.JournalID] = true
		ack := request(t, f.app, "POST", path+"/acknowledge", "", map[string]any{}, 200)
		volumeReceipt(t, saved, ack)
		if string(ack["acknowledged"]) != "true" {
			t.Fatal("explicit volume acknowledgement not retained")
		}
		commands = append(commands, saved)
	}
	// Fixed independent counts and exact amounts are authored separately from CSV generation and posting code.
	wantCounts := map[string]string{
		"ledger.journal_entry": "1000", "ledger.journal_line": "2000", "banking.review_posting": "1000",
		"banking.review_draft": "1000", "banking.source_observation": "1000", "banking.observation_revision": "1000",
		"integration.outbox_message": "1000", "integration.operation_receipt": "1000",
	}
	for table, want := range wantCounts {
		if scalar(t, f.admin, "SELECT count(*)::text FROM "+table) != want {
			t.Fatalf("wrong fixed volume effect count for %s", table)
		}
	}
	if scalar(t, f.admin, "SELECT count(*)::text FROM banking.review_item WHERE state='posted'") != "1000" {
		t.Fatal("volume must retain exactly 1000 posted source projections")
	}
	if scalar(t, f.admin, "SELECT revision::text FROM ledger.book") != "1004" {
		t.Fatal("volume book must finish at fixed setup 4 plus 1000 postings")
	}
	posted := volumeReviews(t, f, "posted")
	wanted := make(map[string]bool, 1000)
	for _, id := range ids {
		wanted[id] = true
	}
	for _, id := range posted {
		if !wanted[id] {
			t.Fatal("public posted projection changed source identity")
		}
	}
	if scalar(t, f.admin, "SELECT count(*)::text FROM ledger.interactive_command WHERE operation='source_post' AND state='committed' AND acknowledged_at IS NOT NULL") != "1000" {
		t.Fatal("volume saved-command lifecycle incomplete")
	}
	if scalar(t, f.admin, "SELECT trim_scale(sum(transaction_amount))::text FROM ledger.journal_line WHERE side='debit'") != "100" || scalar(t, f.admin, "SELECT trim_scale(sum(functional_amount))::text FROM ledger.journal_line WHERE side='credit'") != "100" {
		t.Fatal("volume exact original/functional totals changed")
	}
	if scalar(t, f.admin, "SELECT count(*)::text FROM ledger.journal_line WHERE account_id='"+account+"' AND side='credit'") != "1000" || scalar(t, f.admin, "SELECT count(*)::text FROM ledger.journal_line WHERE account_id='"+restoreExpense+"' AND side='debit'") != "1000" {
		t.Fatal("outflow source or chosen expense classification changed")
	}
	if scalar(t, f.admin, "SELECT count(*)::text FROM ledger.journal_line WHERE trim_scale(transaction_amount)::text<>'0.1' OR trim_scale(functional_amount)::text<>'0.1' OR transaction_currency_version_id<>'"+currency+"'") != "0" {
		t.Fatal("volume line amount/currency changed")
	}
	report := request(t, f.app, "GET", "/v1/entities/"+entity+"/books/"+book+"/reports/trial-balance?through_date=2026-01-31", "", nil, 200)
	var totals map[string]string
	if json.Unmarshal(report["totals"], &totals) != nil || totals["total_debits"] != "100" || totals["total_credits"] != "100" {
		t.Fatal("independent volume report totals changed")
	}
	// Replay old acknowledged receipts after all subsequent revisions; no source/journal/outbox effect may repeat.
	for _, i := range []int{999, 0, 499, 0} {
		saved := commands[i]
		replay := request(t, f.app, "POST", "/v1/commands/"+stringField(t, saved["command_id"])+"/execute", "", map[string]any{}, 200)
		volumeReceipt(t, saved, replay)
		if string(replay["acknowledged"]) != "true" {
			t.Fatal("historical replay lost acknowledgement")
		}
	}
	for table, want := range wantCounts {
		if scalar(t, f.admin, "SELECT count(*)::text FROM "+table) != want {
			t.Fatalf("historical replay duplicated %s", table)
		}
	}
	t.Logf("1000 retained source rows ->1000 reviewed source_post entries/2000 lines; exact USD100.00 per side; every execute replayed, four historical replays; posting=%s total=%s; isolated synthetic PG17, no SLA", time.Since(postingStarted), time.Since(started))
}
