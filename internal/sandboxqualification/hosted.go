package sandboxqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

// RunHosted creates one disposable Sandbox Item; receipts never contain credentials or provider identifiers.
func RunHosted(parent context.Context, allow bool, clientID, secret []byte, transport http.RoundTripper) (receipt Receipt) {
	if !allow {
		return Blocked("opt_in_required")
	}
	if parent == nil || transport == nil || !validSecret(string(clientID)) || !validSecret(string(secret)) {
		return Blocked("input_invalid")
	}
	receipt = Blocked("offline")
	receipt.Mode = "sandbox-api-smoke"
	receipt.Checks = receipt.Checks[1:]
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	// call uses only closed endpoint literals and discards upstream diagnostics on all failures.
	call := func(ctx context.Context, path string, args map[string]any, out any) bool {
		switch path {
		case "/sandbox/public_token/create", "/item/public_token/exchange", "/accounts/get", "/item/remove":
		default:
			return false
		}
		args["client_id"], args["secret"] = string(clientID), string(secret)
		body, err := json.Marshal(args)
		if err != nil {
			return false
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sandbox.plaid.com"+path, bytes.NewReader(body))
		if err != nil {
			return false
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Plaid-Version", "2020-09-14")
		res, err := client.Do(req)
		if err != nil {
			return false
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return false
		}
		raw, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024+1))
		if err != nil || len(raw) > 2*1024*1024 {
			return false
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if !unique(d, 0) {
			return false
		}
		if _, err = d.Token(); err != io.EOF {
			return false
		}
		return json.Unmarshal(raw, out) == nil
	}
	// fail records only fixed stage names; failed create/exchange acknowledgement never triggers retry.
	fail := func(stage string) Receipt {
		receipt.Checks = append([]Check{{stage, "blocked", "provider_unconfirmed"}}, receipt.Checks...)
		return receipt
	}
	var created struct {
		PublicToken string `json:"public_token"`
	}
	if !call(ctx, "/sandbox/public_token/create", map[string]any{"institution_id": "ins_109508", "initial_products": []string{"transactions"}}, &created) || !strings.HasPrefix(created.PublicToken, "public-sandbox-") || !opaque.MatchString(created.PublicToken) {
		return fail("sandbox_item_create")
	}
	var exchanged struct {
		AccessToken string `json:"access_token"`
	}
	if !call(ctx, "/item/public_token/exchange", map[string]any{"public_token": created.PublicToken}, &exchanged) || !strings.HasPrefix(exchanged.AccessToken, "access-sandbox-") || !opaque.MatchString(exchanged.AccessToken) {
		return fail("sandbox_exchange_cleanup_unknown")
	}
	// Cleanup still runs when cancellation or sync failure ends the main diagnostic.
	defer func() {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(parent), 15*time.Second)
		defer stop()
		var removed struct {
			RequestID string `json:"request_id"`
		}
		check := Check{"sandbox_item_cleanup", "blocked", "removal_unconfirmed"}
		if call(cleanup, "/item/remove", map[string]any{"access_token": exchanged.AccessToken}, &removed) && removed.RequestID != "" {
			check.Status = "pass"
			check.Reason = "removal_acknowledged"
		}
		receipt.Checks = append(receipt.Checks, check)
	}()
	var accounts struct {
		Accounts []struct {
			AccountID string `json:"account_id"`
			Balances  struct {
				Currency string `json:"iso_currency_code"`
			} `json:"balances"`
		} `json:"accounts"`
	}
	if !call(ctx, "/accounts/get", map[string]any{"access_token": exchanged.AccessToken}, &accounts) {
		return fail("sandbox_accounts")
	}
	account := ""
	for _, a := range accounts.Accounts {
		if a.Balances.Currency == "USD" && opaque.MatchString(a.AccountID) {
			account = a.AccountID
			break
		}
	}
	if account == "" {
		return fail("sandbox_usd_account")
	}
	// This reference is fixture metadata only; no installed application currency identity is claimed.
	input := &Input{data: inputWire{Environment: "sandbox", ClientID: string(clientID), Secret: string(secret), AccessToken: exchanged.AccessToken, AccountID: account, Currencies: []plaid.Currency{{Code: "USD", VersionID: "00000000-0000-4000-8000-000000000005", MinorUnit: 2, ValidFrom: "2000-01-01"}}}}
	// Initial Sandbox transactions are asynchronous; retry only read-only not-ready diagnostics.
	for attempt := 0; attempt < 5; attempt++ {
		receipt = Run(ctx, true, input, transport)
		if len(receipt.Checks) == 0 || receipt.Checks[0].Reason != "provider_not_ready" || attempt == 4 {
			break
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			attempt = 4
		case <-timer.C:
		}
	}
	receipt.Mode = "sandbox-api-smoke"
	receipt.Checks = append([]Check{{"sandbox_api_auth", "pass", "fictional_item_exchanged"}}, receipt.Checks...)
	return receipt
}
