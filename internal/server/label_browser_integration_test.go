//go:build integration

package server

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var appProofLabels = flag.Bool("app-proof-labels", false, "seed only fictional private label browser proof")

// labelBrowserAuth is confined to the integration binary and cannot enroll an identity.
type labelBrowserAuth struct{}

func (labelBrowserAuth) Authenticate(_ context.Context, token string) (string, error) {
	switch token {
	case "synthetic.browser.proof":
		return "00000000-0000-4000-8000-000000000001", nil
	case "synthetic.other.proof":
		return "00000000-0000-4000-8000-000000000003", nil
	}
	return "", errors.New("synthetic identity rejected")
}

// labelBrowserCall commits the real restricted entrypoint before fixture setup advances.
func labelBrowserCall(t *testing.T, ctx context.Context, db interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}, operator bool, query string, args ...any) map[string]json.RawMessage {
	t.Helper()
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal("synthetic label transaction unavailable", err)
	}
	defer tx.Rollback(context.Background())
	if operator {
		if _, err = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner"); err != nil {
			t.Fatal("synthetic label operator unavailable", err)
		}
	}
	var raw []byte
	if err = tx.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		t.Fatal("synthetic label setup rejected", err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal("synthetic label setup commit unavailable", err)
	}
	var result map[string]json.RawMessage
	if json.Unmarshal(raw, &result) != nil {
		t.Fatal("synthetic label result malformed")
	}
	return result
}

// bootstrapLabelBrowser uses the actual42 source entrypoints and no book or provider.
func bootstrapLabelBrowser(t *testing.T, ctx context.Context, admin *pgx.Conn, runtime *pgxpool.Pool) {
	t.Helper()
	const actor = "00000000-0000-4000-8000-000000000001"
	if _, err := admin.Exec(ctx, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES('00000000-0000-4000-8000-000000000003','https://synthetic.example.invalid','synthetic-label-other');INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000004','EUR',NULL,2,'2000-01-01',NULL,'synthetic-label-fixture');`); err != nil {
		t.Fatal("synthetic label reference unavailable", err)
	}
	grant := labelBrowserCall(t, ctx, admin, true, "SELECT banking.provision_collection_owner($1,'label-browser-grant',$2)", actor, `{"expected_revision":null,"capabilities":["read","acquire","classify"],"evidence_ref":"00000000-0000-4000-8000-000000000005"}`)
	var workspace string
	if json.Unmarshal(grant["workspace_id"], &workspace) != nil {
		t.Fatal("synthetic workspace malformed")
	}
	registered := labelBrowserCall(t, ctx, runtime, false, "SELECT banking.register_collection_item($1,$2,'label-browser-register',$3)", actor, workspace, `{"provider":"plaid","environment":"sandbox","item_id":"SYNTHETIC-LABEL-BROWSER","credential_ref":"00000000-0000-4000-8000-000000000006","credential_version":"1","consent_evidence_ref":"00000000-0000-4000-8000-000000000007","consent_expires_at":"2099-01-01T00:00:00Z","accounts":[{"account_id":"SYNTHETIC-LABEL-USD","display_name":"Fictional dollar account"},{"account_id":"SYNTHETIC-LABEL-EUR","display_name":"Fictional euro account"}]}`)
	var sources []struct {
		Source string `json:"source_account_id"`
	}
	if json.Unmarshal(registered["sources"], &sources) != nil || len(sources) != 2 {
		t.Fatal("synthetic source manifest malformed")
	}
	for i, source := range sources {
		changes := []map[string]any{}
		if i == 0 {
			changes = []map[string]any{
				{"operation": "added", "transaction_id": "SYNTHETIC-LABEL-CAFE", "booked_date": "2026-10-03", "description": "Fictional cafe purchase", "merchant_name": "Fictional Cafe", "source_state": "booked", "currency_code": "USD", "amount": "0.10", "direction": "outflow"},
				{"operation": "added", "transaction_id": "SYNTHETIC-LABEL-HUGE", "booked_date": "2026-10-02", "description": "Fictional exact large quantity", "merchant_name": nil, "source_state": "booked", "currency_code": "USD", "amount": "9007199254740993.01", "direction": "outflow"},
				{"operation": "added", "transaction_id": "SYNTHETIC-LABEL-HOLD", "booked_date": "2026-10-01", "description": "Fictional unsupported quantity", "merchant_name": nil, "source_state": "booked", "currency_code": "ZZZ", "amount": "11.01", "direction": "outflow"},
			}
		} else {
			changes = []map[string]any{{"operation": "added", "transaction_id": "SYNTHETIC-LABEL-EURO", "booked_date": "2026-10-04", "description": "Fictional euro purchase", "merchant_name": nil, "source_state": "booked", "currency_code": "EUR", "amount": "1.23", "direction": "outflow"}}
		}
		body, err := json.Marshal(map[string]any{"expected_item_generation": "1", "expected_source_revision": "0", "starting_cursor": nil, "next_cursor": "synthetic-complete", "readiness": "HISTORICAL_UPDATE_COMPLETE", "changes": changes})
		if err != nil {
			t.Fatal("synthetic label manifest malformed")
		}
		labelBrowserCall(t, ctx, runtime, false, "SELECT banking.publish_collection_batch($1,$2,$3,$4,$5)", actor, workspace, source.Source, "label-browser-publish-"+source.Source, string(body))
	}
}
