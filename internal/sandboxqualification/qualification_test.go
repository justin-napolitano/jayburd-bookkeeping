package sandboxqualification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

const safeInput = `{"environment":"sandbox","client_id":"synthetic-client","secret":"synthetic-secret","access_token":"access-sandbox-synthetic","account_id":"Account-A","currencies":[{"Code":"USD","VersionID":"00000000-0000-4000-8000-000000000005","MinorUnit":2,"ValidFrom":"2000-01-01","ValidTo":""}]}`

// fakeTransport performs no network and lets tests inspect the actual production HTTP request.
type fakeTransport func(*http.Request) (*http.Response, error)

// RoundTrip supplies only synthetic provider responses.
func (f fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// input constructs a strictly parsed fictional operator input.
func input(t *testing.T) *Input {
	t.Helper()
	v, e := ReadInput(strings.NewReader(safeInput))
	if e != nil {
		t.Fatal(e)
	}
	return v
}

// page defines independently specified provider bytes, including exact cents beyond float precision.
func page(cursor string, more bool, added string) string {
	return fmt.Sprintf(`{"accounts":[{"account_id":"Account-A","type":"depository"}],"added":[%s],"modified":[],"removed":[],"next_cursor":%q,"has_more":%t,"request_id":"fixture","transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`, added, cursor, more)
}

const transaction = `{"account_id":"Account-A","transaction_id":"transaction-secret-id","pending":false,"pending_transaction_id":null,"date":"2026-01-06","authorized_date":null,"datetime":null,"authorized_datetime":null,"iso_currency_code":"USD","unofficial_currency_code":null,"amount":9007199254740993.01,"name":"private-fixture-name"}`

// reply provides an ordinary HTTP body consumed by the real bounded adapter.
func reply(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

// TestProductionDecoderProbe verifies exact pinned requests, all pages and a repeat from the final cursor.
func TestProductionDecoderProbe(t *testing.T) {
	requests := 0
	transport := fakeTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://sandbox.plaid.com/transactions/sync" || r.Method != "POST" || r.Header.Get("Plaid-Version") != "2020-09-14" {
			t.Fatal("unpinned request")
		}
		body, _ := io.ReadAll(r.Body)
		var v map[string]any
		_ = json.Unmarshal(body, &v)
		if v["options"].(map[string]any)["account_id"] != "Account-A" {
			t.Fatal("scope missing")
		}
		wants := []any{nil, "C1", "C2"}
		if requests >= len(wants) || v["cursor"] != wants[requests] {
			t.Fatal("cursor sequence")
		}
		responses := []string{page("C1", true, transaction), page("C2", false, ""), page("C2", false, "")}
		out := responses[requests]
		requests++
		return reply(out), nil
	})
	r := Run(context.Background(), true, input(t), transport)
	if r.Status != "partial" || r.Initial != (Counts{2, 1}) || r.Repeat != (Counts{1, 0}) || requests != 3 {
		t.Fatalf("unexpected redacted result %+v", r)
	}
	encoded, _ := json.Marshal(r)
	for _, s := range []string{"synthetic-secret", "access-sandbox", "Account-A", "C1", "C2", "transaction-secret-id", "9007199254740993", "private-fixture-name"} {
		if strings.Contains(string(encoded), s) {
			t.Fatal("receipt disclosure")
		}
	}
	if r.Checks[len(r.Checks)-1].Reason != "empty_delta_observed" {
		t.Fatal("repeat check missing")
	}
}

// TestInputStrictAndPrivate rejects ambiguous/missing metadata before network and prevents accidental serialization.
func TestInputStrictAndPrivate(t *testing.T) {
	tests := []string{safeInput + ` {}`, strings.Replace(safeInput, `"environment":"sandbox"`, `"environment":"production"`, 1), strings.Replace(safeInput, `"secret":"synthetic-secret"`, `"secret":"a","secret":"b"`, 1), strings.Replace(safeInput, `"MinorUnit":2`, `"MinorUnit":null`, 1), strings.Replace(safeInput, `"MinorUnit":2,`, ``, 1), strings.Replace(safeInput, `"account_id"`, `"ACCOUNT_ID"`, 1), strings.Replace(safeInput, `access-sandbox-synthetic`, `access-production-synthetic`, 1), strings.Repeat(" ", 16385)}
	for n, s := range tests {
		if _, e := ReadInput(strings.NewReader(s)); e == nil {
			t.Fatalf("accepted invalid input %d", n)
		}
	}
	v := input(t)
	if _, e := json.Marshal(v); e == nil {
		t.Fatal("input serialized")
	}
	if strings.Contains(fmt.Sprintf("%v %#v", v, v), "synthetic-secret") {
		t.Fatal("formatted input exposed")
	}
}

// TestOptInNeverReadsProvider verifies the library's independent outbound guard.
func TestOptInNeverReadsProvider(t *testing.T) {
	calls := 0
	r := Run(context.Background(), false, input(t), fakeTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, fmt.Errorf("private") }))
	if calls != 0 || r.Status != "blocked" {
		t.Fatal("opt-in bypass")
	}
}

// TestProbeFailuresStayPartialOrBlocked covers scope, parser, progress, readiness and upstream failure with fixed output.
func TestProbeFailuresStayPartialOrBlocked(t *testing.T) {
	cases := []struct{ name, body, reason string }{
		{"mismatch", page("C1", false, strings.Replace(transaction, "Account-A", "Other-A", 1)), "decoder_account_mismatch"},
		{"duplicate", strings.Replace(page("C1", false, ""), `"has_more":false`, `"has_more":false,"has_more":true`, 1), "decoder_duplicate_key"},
		{"cursor encoding", page("private.cursor", false, ""), "decoder_invalid_cursor_encoding"},
		{"cursor progress", page("", false, ""), "decoder_invalid_cursor_progress"},
		{"readiness", strings.Replace(page("", false, ""), "HISTORICAL_UPDATE_COMPLETE", "NOT_READY", 1), "provider_not_ready"},
		{"cycle", page("C1", true, ""), "cursor_not_progressing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Run(context.Background(), true, input(t), fakeTransport(func(*http.Request) (*http.Response, error) { return reply(tc.body), nil }))
			if r.Status != "blocked" || r.Checks[0].Reason != tc.reason {
				t.Fatalf("unexpected %+v", r)
			}
		})
	}
	t.Run("upstream", func(t *testing.T) {
		r := Run(context.Background(), true, input(t), fakeTransport(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("access-sandbox-private") }))
		raw, _ := json.Marshal(r)
		if r.Checks[0].Reason != "provider_unavailable" || strings.Contains(string(raw), "private") {
			t.Fatal("unredacted error")
		}
	})
	t.Run("repeat_changes", func(t *testing.T) {
		n := 0
		r := Run(context.Background(), true, input(t), fakeTransport(func(*http.Request) (*http.Response, error) {
			n++
			if n == 1 {
				return reply(page("C1", false, "")), nil
			}
			return reply(page("C2", false, transaction)), nil
		}))
		if r.Status != "partial" || r.Checks[len(r.Checks)-1].Reason != "provider_changed" {
			t.Fatal("claimed repeat idempotency")
		}
	})
}

// TestPageBudgetStops requests prevents infinite cursor chains without introducing polling or retries.
func TestPageBudgetStops(t *testing.T) {
	n := 0
	r := Run(context.Background(), true, input(t), fakeTransport(func(*http.Request) (*http.Response, error) {
		n++
		return reply(page(fmt.Sprintf("C%d", n), true, "")), nil
	}))
	if n != 10 || r.Status != "blocked" || r.Checks[0].Reason != "profile_limit" {
		t.Fatal("page bound missing")
	}
}

// TestAggregateBudgets verifies byte/change limits across otherwise individually valid pages.
func TestAggregateBudgets(t *testing.T) {
	t.Run("bytes", func(t *testing.T) {
		n := 0
		r := Run(context.Background(), true, input(t), fakeTransport(func(*http.Request) (*http.Response, error) {
			n++
			return reply(page(fmt.Sprintf("C%d", n), true, "") + strings.Repeat(" ", 1100000)), nil
		}))
		if n != 2 || r.Checks[0].Reason != "profile_limit" {
			t.Fatal("aggregate byte limit missing")
		}
	})
	t.Run("changes", func(t *testing.T) {
		n := 0
		r := Run(context.Background(), true, input(t), fakeTransport(func(*http.Request) (*http.Response, error) {
			n++
			rows := make([]string, 500)
			for j := range rows {
				rows[j] = strings.Replace(transaction, "transaction-secret-id", fmt.Sprintf("t-%d-%d", n, j), 1)
			}
			return reply(page(fmt.Sprintf("C%d", n), true, strings.Join(rows, ","))), nil
		}))
		if n != 3 || r.Checks[0].Reason != "profile_limit" {
			t.Fatal("aggregate change limit missing")
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		calls := 0
		r := Run(ctx, true, input(t), fakeTransport(func(*http.Request) (*http.Response, error) { calls++; return reply(page("C1", false, "")), nil }))
		if calls != 0 || r.Status != "blocked" {
			t.Fatal("cancelled run reached transport")
		}
	})
}

// TestDecoderReasonRedaction verifies wrapped codes and arbitrary errors cannot leak values.
func TestDecoderReasonRedaction(t *testing.T) {
	for _, code := range []string{"limit_exceeded", "invalid_expectation", "invalid_json", "duplicate_key", "invalid_number", "invalid_cursor", "invalid_cursor_encoding", "invalid_cursor_progress", "invalid_field", "unknown_enum", "account_mismatch", "unsupported_account", "duplicate_identity", "invalid_identity", "invalid_lineage", "invalid_date", "unsupported_currency", "amount_out_of_range", "unknown_field"} {
		if decoderReason(fmt.Errorf("private wrapper: %w", &plaid.Error{Code: code})) != "decoder_"+code {
			t.Fatal("classification missing")
		}
	}
	for _, err := range []error{nil, fmt.Errorf("private provider detail"), &plaid.Error{Code: "private provider detail"}, (*plaid.Error)(nil)} {
		if decoderReason(err) != "decoder_rejected" {
			t.Fatal("unreviewed error escaped")
		}
	}
	body := strings.Replace(page("C1", false, transaction), `"name":`, `"private-field-marker":"private-value-marker","name":`, 1)
	r := Run(context.Background(), true, input(t), fakeTransport(func(*http.Request) (*http.Response, error) { return reply(body), nil }))
	encoded, _ := json.Marshal(r)
	if r.Checks[0].Reason != "decoder_unknown_field" || r.Initial != (Counts{}) || strings.Contains(string(encoded), "private-") {
		t.Fatal("receipt redaction or failure atomicity")
	}
}
