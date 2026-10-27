//go:build integration

package server

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

var appProofProduction = flag.Bool("app-proof-production", false, "Fictional Production profile only inside the disposable integration executable")

const productionJourneyBank = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/bank"
const productionJourneyLink = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/link"

// productionJourneyPolicy pins two independent fictional wrapping identities.
func productionJourneyPolicy() bankcredential.ProductionPolicy {
	return bankcredential.ProductionPolicy{BankKey: productionJourneyBank, LinkKey: productionJourneyLink}
}

// configureProductionJourney selects only already-created restricted fictional logins before enrollment.
func configureProductionJourney(t *testing.T, ctx context.Context, admin *pgx.Conn) {
	t.Helper()
	if _, err := admin.Exec(ctx, "CREATE ROLE synthetic_journey_operator LOGIN NOINHERIT IN ROLE bookkeeping_owner"); err != nil {
		t.Fatal("fictional operator creation failed", err)
	}
	operator, err := pgx.Connect(ctx, "host=database port=5432 user=synthetic_journey_operator dbname=app_proof sslmode=disable connect_timeout=5")
	if err != nil {
		t.Fatal("fictional operator login failed", err)
	}
	defer operator.Close(context.Background())
	b, _ := json.Marshal(map[string]any{"environment": "production", "api_login": "synthetic_app", "worker_login": "synthetic_connection_sync", "bank_key": productionJourneyBank, "link_key": productionJourneyLink, "enabled": true})
	if _, err := demoOwnerCall(ctx, operator, "SELECT integration.configure_provider_scope($1,$2,$3,'production-journey',$4)", connectionProofActor, connectionProofEntity, connectionProofBook, string(b)); err != nil {
		t.Fatal("fictional Production scope rejected", err)
	}
	// The bounded pilot uses controlled owner setup before bank access; this is not self-service preflight proof.
	if _, err := demoOwnerCall(ctx, operator, "SELECT to_jsonb(integration.creator_connection_setup($1,$2,$3,'synthetic_connection_sync','synthetic_connection_publisher',true))", connectionProofActor, connectionProofEntity, connectionProofBook); err != nil {
		t.Fatal("fictional controlled owner setup rejected", err)
	}
}

// productionJourneyWrapper substitutes external KMS only; the store's real AAD and AEAD remain exercised.
type productionJourneyWrapper struct{ key string }

// Wrap uses one fictional version for each distinct key capability.
func (w productionJourneyWrapper) Wrap(_ context.Context, key, aad []byte) (string, []byte, error) {
	b, _ := aes.NewCipher(bytes.Repeat([]byte{19}, 32))
	g, _ := cipher.NewGCM(b)
	return w.key + "/cryptoKeyVersions/7", g.Seal(nil, make([]byte, 12), key, aad), nil
}

// Unwrap authenticates the exact fictional key version and the production store's addressed AAD.
func (w productionJourneyWrapper) Unwrap(_ context.Context, version string, raw, aad []byte) ([]byte, error) {
	if version != w.key+"/cryptoKeyVersions/7" {
		return nil, bankcredential.ErrUnavailable
	}
	b, _ := aes.NewCipher(bytes.Repeat([]byte{19}, 32))
	g, _ := cipher.NewGCM(b)
	return g.Open(nil, make([]byte, 12), raw, aad)
}

// productionJourneyPage refuses foreign routing/filter/cursor before returning independently authored decimal bytes.
func productionJourneyPage(r *http.Request) (*http.Response, error) {
	if r.URL.String() != "https://production.plaid.com/transactions/sync" || r.Header.Get("Plaid-Version") != "2020-09-14" {
		return nil, errors.New("fictional Production route rejected")
	}
	var request struct {
		Cursor  *string `json:"cursor"`
		Options struct {
			Account string `json:"account_id"`
		} `json:"options"`
	}
	if json.NewDecoder(r.Body).Decode(&request) != nil || request.Options.Account != "synthetic-flow-account" {
		return nil, errors.New("fictional Production filter rejected")
	}
	cursor := ""
	if request.Cursor != nil {
		cursor = *request.Cursor
	}
	var added, next, status string
	more := false
	switch cursor {
	case "":
		added = "[]"
		next = "production-empty"
		status = "NOT_READY"
	case "production-empty":
		added = `[{"account_id":"synthetic-flow-account","transaction_id":"synthetic-cent-a","amount":0.10,"iso_currency_code":"USD","unofficial_currency_code":null,"date":"2026-10-13","pending":false,"pending_transaction_id":null,"name":"Fictional purchase A"}]`
		next = "production-page-a"
		status = "INITIAL_UPDATE_COMPLETE"
		more = true
	case "production-page-a":
		added = `[{"account_id":"synthetic-flow-account","transaction_id":"synthetic-cent-b","amount":0.20,"iso_currency_code":"USD","unofficial_currency_code":null,"date":"2026-10-13","pending":false,"pending_transaction_id":null,"name":"Fictional purchase B"}]`
		next = "production-history"
		status = "HISTORICAL_UPDATE_COMPLETE"
	default:
		return nil, errors.New("unexpected fictional cursor")
	}
	body := `{"accounts":[{"account_id":"synthetic-flow-account","type":"depository"}],"added":` + added + `,"modified":[],"removed":[],"next_cursor":"` + next + `","has_more":` + map[bool]string{false: "false", true: "true"}[more] + `,"transactions_update_status":"` + status + `","request_id":"fictional-production-journey"}`
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

// productionJourneyWrap adds only fixed fixture triggers in a test binary, never in application routes.
func productionJourneyWrap(f *connectionBrowserFixture, product http.Handler) http.Handler {
	history := false
	proof := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__proof/acquire" {
			f.ServeHTTP(w, r)
			return
		}
		if r.Method != "POST" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer synthetic.browser.proof" || r.Header.Get("Content-Type") != "application/json" {
			writeProblem(w, 400, "fixture_rejected")
			return
		}
		raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 64))
		if e != nil || strings.TrimSpace(string(raw)) != "{}" {
			writeProblem(w, 400, "fixture_rejected")
			return
		}
		var stream string
		if e = f.admin.QueryRow(r.Context(), "SELECT stream_id::text FROM integration.sync_stream WHERE entity_id=$1 AND book_id=$2 AND account_id='synthetic-flow-account'", connectionProofEntity, connectionProofBook).Scan(&stream); e != nil {
			writeProblem(w, 409, "fixture_binding_required")
			return
		}
		switch r.URL.Path {
		case "/__proof/history":
			if !history {
				tx, e := f.worker.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.Serializable})
				if e != nil {
					writeProblem(w, 500, "fixture_failed")
					return
				}
				defer tx.Rollback(context.Background())
				if _, e = tx.Exec(r.Context(), "SELECT integration.request_sync($1,$2,$3,'production-history')", connectionProofEntity, connectionProofBook, stream); e != nil || tx.Commit(r.Context()) != nil {
					writeProblem(w, 500, "fixture_failed")
					return
				}
				history = true
				f.claim = "10000000-0000-4000-8000-000000000031"
			}
			r.URL.Path = "/__proof/acquire"
			r.Body = io.NopCloser(strings.NewReader("{}"))
			f.ServeHTTP(w, r)
		case "/__proof/denied":
			before := f.transport.calls.Load()
			_, e = f.runner.Run(r.Context(), acquisition.Scope{EntityID: connectionProofEntity, BookID: connectionProofBook, StreamID: stream}, "10000000-0000-4000-8000-000000000032")
			if e == nil || before != f.transport.calls.Load() {
				writeProblem(w, 500, "fixture_denial_failed")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"denied_before_fetch":true}`))
		default:
			writeProblem(w, 404, "not_found")
		}
	})
	boundary, e := ledger.BrowserBoundary("http://localhost:39083", proof)
	if e != nil {
		panic("fixed fictional origin rejected")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/__proof/") {
			boundary.ServeHTTP(w, r)
			return
		}
		product.ServeHTTP(w, r)
	})
}
