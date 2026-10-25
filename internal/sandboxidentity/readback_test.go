package sandboxidentity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxverification"
)

const privateCredentials = `{"client_id":"private-client","secret":"private-secret","access_token":"access-sandbox-private"}`

// readbackTransport routes only synthetic requests; no listener, DNS or credential is used.
type readbackTransport func(*http.Request) (*http.Response, error)

func (f readbackTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// trackedBody proves bounded read and close even when provider input is rejected.
type trackedBody struct {
	io.Reader
	closed bool
	read   int
}

func (b *trackedBody) Read(p []byte) (int, error) { n, e := b.Reader.Read(p); b.read += n; return n, e }
func (b *trackedBody) Close() error               { b.closed = true; return nil }

// TestPrivateReadbackExactRequestAndIndependentMap fixes the request profile and expected IDs independently of finance.
func TestPrivateReadbackExactRequestAndIndependentMap(t *testing.T) {
	want := `{"profile":"jayburd-sandbox-identity-map/1","fixture_id":"fresh-1","items":[{"logical_id":"a","transaction_id":"Provider-A"},{"logical_id":"b","transaction_id":"Provider-B"}]}` + "\n"
	before := []byte(fixture)
	for _, input := range []string{page, strings.NewReplacer("12.34", "987654.321", "2026-10-21", "2000-01-01", `"pending":false`, `"pending":true`).Replace(page)} {
		calls := 0
		body := &trackedBody{Reader: strings.NewReader(input)}
		out, n, err := Readback(context.Background(), before, []byte(privateCredentials), "Account-A", time.Now().Add(time.Minute), readbackTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "POST" || r.URL.String() != "https://sandbox.plaid.com/transactions/sync" || r.Header.Get("Plaid-Version") != "2020-09-14" || r.Header.Get("Content-Type") != "application/json" {
				t.Fatal("wrong private transport target/profile")
			}
			raw, e := io.ReadAll(r.Body)
			if e != nil {
				t.Fatal("request unreadable")
			}
			_ = r.Body.Close()
			fields, e := sandboxverification.StrictObject(raw)
			if e != nil || len(fields) != 5 || string(fields["count"]) != "500" || string(fields["options"]) != `{"account_id":"Account-A"}` || string(fields["client_id"]) != `"private-client"` || string(fields["secret"]) != `"private-secret"` || string(fields["access_token"]) != `"access-sandbox-private"` {
				t.Fatal("request widened scope or lost private injection")
			}
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > 15*time.Second {
				t.Fatal("request not bounded")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, Body: body}, nil
		}))
		if err != nil || string(out) != want || n != 2 || calls != 1 || !body.closed || !bytes.Equal(before, []byte(fixture)) {
			t.Fatal("independent map or single-request boundary failed")
		}
	}
}

// TestPrivateReadbackDeniesBeforeTransport covers closed private input and local authority expiry.
func TestPrivateReadbackDeniesBeforeTransport(t *testing.T) {
	never := readbackTransport(func(*http.Request) (*http.Response, error) { t.Fatal("invalid readback made request"); return nil, nil })
	for _, credential := range []string{"", `null`, privateCredentials + `{}`, strings.Replace(privateCredentials, `"secret":`, `"unknown":`, 1), strings.Replace(privateCredentials, `"secret":`, `"secret":"x","secret":`, 1), strings.Replace(privateCredentials, "access-sandbox-private", "access-production-private", 1), strings.Replace(privateCredentials, "private-secret", `private\nsecret`, 1), strings.Repeat("x", MaxCredentialBytes+1)} {
		if out, n, e := Readback(context.Background(), []byte(fixture), []byte(credential), "Account-A", time.Now().Add(time.Minute), never); e != ErrReadback || out != nil || n != 0 {
			t.Fatal("unsafe credential accepted")
		}
	}
	for _, expiry := range []time.Time{time.Time{}, time.Now().Add(-time.Second), time.Now().Add(61 * time.Minute)} {
		if _, _, e := Readback(context.Background(), []byte(fixture), []byte(privateCredentials), "Account-A", expiry, never); e != ErrReadback {
			t.Fatal("unbounded or expired authority accepted")
		}
	}
	for _, f := range []string{`{}`, strings.Replace(fixture, "Fictional B", "Fictional A", 1), strings.Replace(fixture, `"revision":"1"`, `"revision":"2"`, 1)} {
		if _, _, e := Readback(context.Background(), []byte(f), []byte(privateCredentials), "Account-A", time.Now().Add(time.Minute), never); e != ErrReadback {
			t.Fatal("unsupported expectation reached transport")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e := Readback(ctx, []byte(fixture), []byte(privateCredentials), "Account-A", time.Now().Add(time.Minute), never); e != ErrReadback {
		t.Fatal("canceled authority accepted")
	}
	// Independently construct501 unique identities to prove the producer's one-page bound.
	rows := make([]json.RawMessage, 501)
	for i := range rows {
		rows[i] = json.RawMessage(`{"logical_id":"id` + strconv.Itoa(i) + `","description":"tag` + strconv.Itoa(i) + `","revision":"1","current":true,"previous_revision":null,"source_state":"booked"}`)
	}
	raw, _ := json.Marshal(struct {
		Profile string            `json:"profile"`
		ID      string            `json:"fixture_id"`
		Rows    []json.RawMessage `json:"rows"`
		Links   []string          `json:"pending_links"`
	}{"jayburd-sandbox-verification/1", "large", rows, []string{}})
	if ValidateReadbackFixture(raw) != ErrProjection {
		t.Fatal("producer accepted more than one page's maximum")
	}
}

// TestPrivateReadbackFailuresCloseAndRedact proves no retry, redirect, partial map or raw diagnostic escapes.
func TestPrivateReadbackFailuresCloseAndRedact(t *testing.T) {
	for _, name := range []string{"status", "redirect", "media", "oversize", "invalid", "foreign", "missing", "duplicate", "incomplete", "cancel", "transport"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			input, status, media := page, 200, "application/json"
			switch name {
			case "status":
				status = 503
				input = "private-provider-diagnostic"
			case "redirect":
				status = 307
			case "media":
				media = "text/html"
			case "oversize":
				input = strings.Repeat("x", sandboxverification.MaxInputBytes)
			case "invalid":
				input = `{"private":"diagnostic"}`
			case "foreign":
				input = strings.ReplaceAll(page, "Account-A", "Foreign-A")
			case "missing":
				input = strings.Replace(page, "Provider-B", "", 1)
			case "duplicate":
				input = strings.Replace(page, "Provider-B", "Provider-A", 1)
			case "incomplete":
				input = strings.Replace(page, `"has_more":false`, `"has_more":true`, 1)
			}
			body, calls := &trackedBody{Reader: strings.NewReader(input)}, 0
			out, n, e := Readback(ctx, []byte(fixture), []byte(privateCredentials), "Account-A", time.Now().Add(time.Minute), readbackTransport(func(*http.Request) (*http.Response, error) {
				calls++
				if name == "transport" {
					return nil, errors.New("private-transport-diagnostic")
				}
				if name == "cancel" {
					cancel()
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{media}, "Location": []string{"https://foreign.example/"}}, Body: body}, nil
			}))
			if e != ErrReadback || e.Error() != "sandbox_identity_readback_failed" || out != nil || n != 0 || calls != 1 || name != "transport" && !body.closed || body.read > sandboxverification.MaxInputBytes-len(fixture)+1 {
				t.Fatal("failed readback leaked data, retried or left body open")
			}
		})
	}
}
