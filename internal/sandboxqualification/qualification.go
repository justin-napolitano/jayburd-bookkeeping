// Package sandboxqualification runs opt-in, redacted provider diagnostics without claiming application qualification.
package sandboxqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

const Profile = "plaid-sandbox-diagnostic-v1"

var invalid = errors.New("qualification_input_invalid")
var opaque = regexp.MustCompile(`^[!-~]{1,256}$`)
var secretText = regexp.MustCompile(`^[!-~]+$`)

// validSecret enforces a byte bound before accepting printable operator credentials.
func validSecret(s string) bool { return len(s) > 0 && len(s) <= 8192 && secretText.MatchString(s) }

// Input keeps operator-owned credentials and scope private; only ReadInput can construct it externally.
type Input struct{ data inputWire }

// inputWire is exclusively the explicit pipe input, never a receipt or log value.
type inputWire struct {
	Environment string           `json:"environment"`
	ClientID    string           `json:"client_id"`
	Secret      string           `json:"secret"`
	AccessToken string           `json:"access_token"`
	AccountID   string           `json:"account_id"`
	Currencies  []plaid.Currency `json:"currencies"`
	Cursor      *string          `json:"cursor"`
}

// String prevents whole-input diagnostics from exposing credential or provider values.
func (*Input) String() string { return "[qualification input redacted]" }

// GoString also suppresses Go-syntax formatting.
func (i *Input) GoString() string { return i.String() }

// MarshalJSON rejects accidental input persistence.
func (*Input) MarshalJSON() ([]byte, error) { return nil, invalid }

// ReadInput strictly parses one bounded owner pipe object; caller must enforce opt-in before reading.
func ReadInput(r io.Reader) (*Input, error) {
	if r == nil {
		return nil, invalid
	}
	raw, e := io.ReadAll(io.LimitReader(r, 16385))
	if e != nil || len(raw) > 16384 || !utf8.Valid(raw) {
		return nil, invalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if !unique(d, 0) {
		return nil, invalid
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, invalid
	}
	var shape map[string]json.RawMessage
	if json.Unmarshal(raw, &shape) != nil {
		return nil, invalid
	}
	for key := range shape {
		switch key {
		case "environment", "client_id", "secret", "access_token", "account_id", "currencies", "cursor":
		default:
			return nil, invalid
		}
	}
	var currencies []map[string]json.RawMessage
	if json.Unmarshal(shape["currencies"], &currencies) != nil {
		return nil, invalid
	}
	for _, c := range currencies {
		for key := range c {
			switch key {
			case "Code", "VersionID", "MinorUnit", "ValidFrom", "ValidTo":
			default:
				return nil, invalid
			}
		}
		if len(c["MinorUnit"]) == 0 || bytes.Equal(c["MinorUnit"], []byte("null")) {
			return nil, invalid
		}
	}
	var w inputWire
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&w) != nil || w.Environment != "sandbox" || !validSecret(w.ClientID) || !validSecret(w.Secret) || !validSecret(w.AccessToken) || len(w.AccessToken) < 16 || w.AccessToken[:15] != "access-sandbox-" || !opaque.MatchString(w.AccountID) || len(w.Currencies) == 0 || len(w.Currencies) > 16 || w.Cursor != nil && !opaque.MatchString(*w.Cursor) {
		return nil, invalid
	}
	// Validate installed-reference syntax through the production decoder before any request.
	probe := []byte(`{"accounts":[],"added":[],"modified":[],"removed":[],"next_cursor":"","has_more":false,"request_id":"validation","transactions_update_status":"NOT_READY"}`)
	if _, e = plaid.DecodePage(probe, plaid.Expected{AccountID: w.AccountID, Currencies: w.Currencies}); e != nil {
		return nil, invalid
	}
	return &Input{w}, nil
}

// unique rejects duplicate keys throughout nested objects before ordinary struct decoding.
func unique(d *json.Decoder, depth int) bool {
	if depth > 16 {
		return false
	}
	token, e := d.Token()
	if e != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return true
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			s, ok := k.(string)
			if e != nil || !ok || seen[s] {
				return false
			}
			seen[s] = true
			if !unique(d, depth+1) {
				return false
			}
		}
		end, e := d.Token()
		return e == nil && end == json.Delim('}')
	case '[':
		for d.More() {
			if !unique(d, depth+1) {
				return false
			}
		}
		end, e := d.Token()
		return e == nil && end == json.Delim(']')
	}
	return false
}

// Check is a fixed outcome vocabulary with no upstream identifiers or diagnostics.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// Counts describes provider deltas observed, never accounting totals or persisted transactions.
type Counts struct {
	Pages   int `json:"pages"`
	Changes int `json:"changes"`
}

// Receipt cannot represent full qualification success; unresolved application gates stay explicit.
type Receipt struct {
	Profile string  `json:"profile"`
	Mode    string  `json:"mode"`
	Status  string  `json:"status"`
	Initial Counts  `json:"initial"`
	Repeat  Counts  `json:"repeat"`
	Checks  []Check `json:"checks"`
}

// Blocked returns a closed reason without reading credentials or creating transport.
func Blocked(reason string) Receipt {
	switch reason {
	case "offline", "opt_in_required", "input_unavailable", "input_invalid", "invalid_invocation":
	default:
		reason = "input_invalid"
	}
	return Receipt{Profile: Profile, Mode: "offline", Status: "blocked", Checks: []Check{{"execution", "blocked", reason}, {"real_link_exchange", "blocked", "application_proof_required"}, {"provider_to_review_no_journals", "blocked", "application_proof_required"}, {"dynamic_pending_posted", "blocked", "application_proof_required"}, {"signed_webhook_delivery", "blocked", "application_proof_required"}}}
}

// Run requires explicit opt-in, uses fixed production transport/decoder, and never persists raw pages.
func Run(parent context.Context, allow bool, input *Input, transport http.RoundTripper) Receipt {
	if !allow {
		return Blocked("opt_in_required")
	}
	if parent == nil || input == nil || transport == nil {
		return Blocked("input_invalid")
	}
	r := Blocked("offline")
	r.Mode = "sandbox-sync"
	r.Checks = r.Checks[1:]
	provider, e := acquisition.NewPlaidHTTP(transport)
	if e != nil {
		return Blocked("input_invalid")
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	cursor, ready, count, reason := round(ctx, provider, input, input.data.Cursor)
	r.Initial = count
	if reason != "" {
		r.Checks = append([]Check{{"initial_sync", "blocked", reason}}, r.Checks...)
		return r
	}
	r.Status = "partial"
	r.Checks = append([]Check{{"initial_sync", "pass", "exact_decode_complete"}}, r.Checks...)
	history := Check{"historical_readiness", "blocked", "history_not_complete"}
	if ready == "HISTORICAL_UPDATE_COMPLETE" {
		history.Status = "pass"
		history.Reason = "provider_reports_complete"
	}
	r.Checks = append(r.Checks, history)
	_, _, count, reason = round(ctx, provider, input, &cursor)
	r.Repeat = count
	repeat := Check{"repeat_delta", "blocked", reason}
	if reason == "" {
		if count.Changes == 0 {
			repeat.Status = "pass"
			repeat.Reason = "empty_delta_observed"
		} else {
			repeat.Reason = "provider_changed"
		}
	}
	r.Checks = append(r.Checks, repeat)
	return r
}

// round follows at most ten pages and retains only counts/cursors; it does not implement SQL publication or retry policy.
func round(ctx context.Context, provider acquisition.Provider, input *Input, cursor *string) (string, string, Counts, string) {
	var counts Counts
	total := 0
	seen := map[string]bool{}
	if cursor != nil {
		seen[*cursor] = true
	}
	for n := 0; n < 10; n++ {
		raw, e := provider.Fetch(ctx, acquisition.FetchRequest{AccountID: input.data.AccountID, Cursor: cursor, Credentials: acquisition.ProviderCredentials{AccessToken: input.data.AccessToken, ClientID: input.data.ClientID, Secret: input.data.Secret}})
		if e != nil {
			return "", "", counts, "provider_unavailable"
		}
		total += len(raw)
		if total > plaid.MaxBytes {
			return "", "", counts, "profile_limit"
		}
		page, e := plaid.DecodePage(raw, plaid.Expected{AccountID: input.data.AccountID, Currencies: input.data.Currencies})
		if e != nil {
			return "", "", counts, "decoder_rejected"
		}
		counts.Pages++
		changes := len(page.Added) + len(page.Modified) + len(page.Removed)
		counts.Changes += changes
		if counts.Changes > 1000 {
			return "", "", counts, "profile_limit"
		}
		if ctx.Err() != nil {
			return "", "", counts, "execution_cancelled"
		}
		if page.NextCursor == "" || page.UpdateStatus == "NOT_READY" || page.UpdateStatus == "TRANSACTIONS_UPDATE_STATUS_UNKNOWN" {
			return "", "", counts, "provider_not_ready"
		}
		if seen[page.NextCursor] && (page.HasMore || changes > 0) {
			return "", "", counts, "cursor_not_progressing"
		}
		if !page.HasMore {
			return page.NextCursor, page.UpdateStatus, counts, ""
		}
		seen[page.NextCursor] = true
		next := page.NextCursor
		cursor = &next
	}
	return "", "", counts, "profile_limit"
}
