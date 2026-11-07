//go:build integration

package server

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var appProofLabels = flag.Bool("app-proof-labels", false, "seed only fictional private label browser proof")

const labelPreparedRecoveryActor = "00000000-0000-4000-8000-000000000091"
const labelCommittedRecoveryActor = "00000000-0000-4000-8000-000000000092"

// labelBrowserAuth is confined to the integration binary and cannot enroll an identity.
type labelBrowserAuth struct{}

func (labelBrowserAuth) Authenticate(_ context.Context, token string) (string, error) {
	switch token {
	case "synthetic.browser.proof":
		return "00000000-0000-4000-8000-000000000001", nil
	case "synthetic.other.proof":
		return "00000000-0000-4000-8000-000000000003", nil
	case "synthetic.prepared.proof":
		return labelPreparedRecoveryActor, nil
	case "synthetic.committed.proof":
		return labelCommittedRecoveryActor, nil
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
	bootstrapLabelRecoveryBrowser(t, ctx, admin, runtime, labelPreparedRecoveryActor, false)
	bootstrapLabelRecoveryBrowser(t, ctx, admin, runtime, labelCommittedRecoveryActor, true)
}

// bootstrapLabelRecoveryBrowser creates actual inaccessible work without exposing mutation test routes.
func bootstrapLabelRecoveryBrowser(t *testing.T, ctx context.Context, admin *pgx.Conn, runtime *pgxpool.Pool, actor string, committed bool) {
	t.Helper()
	subject, name, tag := "synthetic-prepared-recovery", "Fictional hidden prepared tag", "00000000-0000-4000-8000-000000000901"
	if committed {
		subject, name, tag = "synthetic-committed-recovery", "Fictional hidden committed tag", "00000000-0000-4000-8000-000000000902"
	}
	if _, err := admin.Exec(ctx, "INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'https://synthetic.example.invalid',$2)", actor, subject); err != nil {
		t.Fatal("synthetic recovery principal unavailable", err)
	}
	grant := labelBrowserCall(t, ctx, admin, true, "SELECT banking.provision_collection_owner($1,'recovery-browser-grant',$2)", actor, `{"expected_revision":null,"capabilities":["read","acquire","classify"],"evidence_ref":"00000000-0000-4000-8000-000000000005"}`)
	var workspace string
	if json.Unmarshal(grant["workspace_id"], &workspace) != nil {
		t.Fatal("synthetic recovery workspace malformed")
	}
	page := labelBrowserCall(t, ctx, runtime, false, "SELECT banking.query_collection_review_v2($1,$2,$3)", actor, workspace, `{"profile":"private-collection-review-v2","source_account_id":null,"assignment":"all","starts_on":null,"through_date":null,"search":"","topic":{"kind":"all"},"tag":{"kind":"all"},"sort":"date_desc","snapshot":null,"cursor":null,"limit":25}`)
	previewBody, err := json.Marshal(map[string]any{"snapshot": page["snapshot"], "selection": []any{}, "action": map[string]any{"kind": "tag_catalog", "tags": []map[string]string{{"tag_id": tag, "name": name, "status": "active"}}}})
	if err != nil {
		t.Fatal("synthetic recovery preview malformed")
	}
	preview := labelBrowserCall(t, ctx, runtime, false, "SELECT banking.preview_collection_labels_v2($1,$2,$3)", actor, workspace, string(previewBody))
	prepareBody, err := json.Marshal(map[string]any{"preview_id": preview["preview_id"], "prepare_key": "recovery-browser-private"})
	if err != nil {
		t.Fatal("synthetic recovery preparation malformed")
	}
	prepared := labelBrowserCall(t, ctx, runtime, false, "SELECT banking.prepare_collection_label_command_v2($1,$2,$3)", actor, workspace, string(prepareBody))
	var command struct {
		ID string `json:"command_id"`
	}
	if json.Unmarshal(prepared["command"], &command) != nil || command.ID == "" {
		t.Fatal("synthetic recovery command malformed")
	}
	if committed {
		labelBrowserCall(t, ctx, runtime, false, "SELECT banking.transition_collection_label_command_v2($1,$2,$3,'execute','{}')", actor, workspace, command.ID)
	}
	capabilities := []string{"read"}
	if committed {
		capabilities = []string{}
	}
	revokeBody, err := json.Marshal(map[string]any{"expected_revision": "1", "capabilities": capabilities, "evidence_ref": "00000000-0000-4000-8000-000000000005"})
	if err != nil {
		t.Fatal("synthetic recovery revocation malformed")
	}
	labelBrowserCall(t, ctx, admin, true, "SELECT banking.provision_collection_owner($1,'recovery-browser-revoke',$2)", actor, string(revokeBody))
}

// TestLabelBrowserBootstrapPostgres verifies the browser's real source/recovery fixtures without a listener.
func TestLabelBrowserBootstrapPostgres(t *testing.T) {
	if !*appProofLabels {
		t.Skip("use only the disposable tag/recovery browser fixture")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := fmt.Sprintf("tagrec_browser_%d", time.Now().UnixNano())
	admin, err := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
	if err != nil {
		t.Fatal("synthetic fixture database unavailable")
	}
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE tagrec_template"); err != nil {
		t.Fatal("synthetic browser clone unavailable", err)
	}
	_ = admin.Close(ctx)
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		cleanup, err := pgx.Connect(cleanupCtx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
		if err != nil {
			t.Error("synthetic browser cleanup connection unavailable")
			return
		}
		defer cleanup.Close(cleanupCtx)
		if _, err = cleanup.Exec(cleanupCtx, "DROP DATABASE "+pgx.Identifier{db}.Sanitize()); err != nil {
			t.Error("owned synthetic browser clone cleanup failed")
		}
	})
	owner, err := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname="+db+" sslmode=disable connect_timeout=5")
	if err != nil {
		t.Fatal("synthetic fixture owner unavailable")
	}
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	if _, err = owner.Exec(ctx, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES('00000000-0000-4000-8000-000000000001','https://synthetic.example.invalid','synthetic-browser');INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000002','USD','840',2,'2000-01-01',NULL,'synthetic-browser-fixture');`); err != nil {
		t.Fatal("synthetic browser initial identity unavailable", err)
	}
	pool, err := pgxpool.New(ctx, "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5 pool_max_conns=2")
	if err != nil {
		t.Fatal("synthetic browser restricted runtime unavailable")
	}
	t.Cleanup(pool.Close)
	bootstrapLabelBrowser(t, ctx, owner, pool)
	var state string
	if err = owner.QueryRow(ctx, "SELECT (SELECT count(*)FROM banking.collection_observation)::text||':'||(SELECT count(*)FROM ledger.book)::text||':'||(SELECT count(*)FROM ledger.journal_entry)::text").Scan(&state); err != nil || state != "4:0:0" {
		t.Fatal("browser fixture invented accounting state or lost original rows", err)
	}
	for _, tc := range []struct{ actor, state string }{{labelPreparedRecoveryActor, "prepared:false"}, {labelCommittedRecoveryActor, "committed:false"}} {
		status := labelBrowserCall(t, ctx, pool, false, "SELECT banking.collection_recovery_status($1)", tc.actor)
		if len(status) != 3 || string(status["state"]) != `"inaccessible"` || string(status["recovery_id"]) == "null" {
			t.Fatal("browser recovery fixture exposed or misclassified saved private work")
		}
		if err = owner.QueryRow(ctx, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM banking.collection_label_command WHERE principal_id=$1", tc.actor).Scan(&state); err != nil || state != tc.state {
			t.Fatal("browser recovery fixture changed the prepared/committed distinction", err)
		}
	}
}
