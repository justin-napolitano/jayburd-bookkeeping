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
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

var appProofConnections = flag.Bool("app-proof-connections", false, "Inject fictional connection and acquisition dependencies only in disposable browser proof")

const connectionProofActor = "00000000-0000-4000-8000-000000000001"
const connectionProofCurrency = "00000000-0000-4000-8000-000000000002"
const connectionProofEntity = "10000000-0000-4000-8000-000000000002"
const connectionProofBook = "10000000-0000-4000-8000-000000000003"
const connectionProofBank = "10000000-0000-4000-8000-000000000011"
const connectionProofExpense = "10000000-0000-4000-8000-000000000012"
const connectionProofClaim = "10000000-0000-4000-8000-000000000030"

// connectionProofProvider supplies verified fictional Link/account results and counts external mutations.
type connectionProofProvider struct {
	exchanges   atomic.Int64
	completions atomic.Int64
}

// CreateLink returns an ephemeral synthetic value and fixed bounded expiry.
func (*connectionProofProvider) CreateLink(context.Context, connections.LinkRequest) (connections.LinkResult, error) {
	return connections.LinkResult{HostedURL: connections.NewSecret("https://secure.plaid.com/hl/synthetic"), Token: connections.NewSecret("synthetic-link"), ExpiresAt: time.Now().Add(5 * time.Minute)}, nil
}

// CompletedPublicToken provides one pending check before a fictional hosted completion.
func (p *connectionProofProvider) CompletedPublicToken(_ context.Context, link connections.Secret) (connections.Secret, error) {
	if link != connections.NewSecret("synthetic-link") {
		return connections.Secret{}, errors.New("unverified")
	}
	if p.completions.Add(1) == 1 {
		return connections.Secret{}, &connections.Failure{Code: "link_session_pending"}
	}
	return connections.NewSecret("synthetic-public"), nil
}

// VerifySession binds only the known synthetic session and public token.
func (*connectionProofProvider) VerifySession(_ context.Context, link, public connections.Secret, expectedItem string) error {
	if expectedItem != "" || link != connections.NewSecret("synthetic-link") || public != connections.NewSecret("synthetic-public") {
		return errors.New("unverified")
	}
	return nil
}

// Exchange counts the side effect so replay cannot be mistaken for success.
func (p *connectionProofProvider) Exchange(context.Context, connections.Secret) (connections.ExchangeResult, error) {
	p.exchanges.Add(1)
	return connections.ExchangeResult{ItemID: "synthetic-flow-item", AccessToken: connections.NewSecret("synthetic-access")}, nil
}

// Manifest returns only server-verified account identity; there are no client-supplied balances.
func (*connectionProofProvider) Manifest(context.Context, connections.Secret) (connections.Manifest, error) {
	code := "USD"
	return connections.Manifest{ItemID: "synthetic-flow-item", Accounts: []connections.Account{{AccountID: "synthetic-flow-account", Type: "depository", Currency: &code}}, ConsentExpiresAt: "2099-01-01T00:00:00Z"}, nil
}

// Rotate is deliberately unavailable in this enrollment-to-review scenario.
func (*connectionProofProvider) Rotate(context.Context, connections.Secret) (connections.ExchangeResult, error) {
	return connections.ExchangeResult{}, errors.New("not in scenario")
}

// Remove is deliberately unavailable; disconnect must never call it.
func (*connectionProofProvider) Remove(context.Context, connections.Secret) error {
	return errors.New("not in scenario")
}

// connectionProofTransport intercepts the pinned production adapter; no request reaches Plaid.
type connectionProofTransport struct{ calls atomic.Int64 }

// RoundTrip verifies account filtering and serves exact original decimal bytes.
func (p *connectionProofTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	p.calls.Add(1)
	if r.URL.String() != "https://sandbox.plaid.com/transactions/sync" || r.Header.Get("Plaid-Version") != "2020-09-14" {
		return nil, errors.New("wrong destination")
	}
	raw, e := io.ReadAll(r.Body)
	if e != nil {
		return nil, e
	}
	var request struct {
		Options struct {
			AccountID string `json:"account_id"`
		} `json:"options"`
	}
	if json.Unmarshal(raw, &request) != nil || request.Options.AccountID != "synthetic-flow-account" {
		return nil, errors.New("wrong scope")
	}
	body := `{"accounts":[{"account_id":"synthetic-flow-account","type":"depository"}],"added":[{"account_id":"synthetic-flow-account","transaction_id":"synthetic-cent-a","amount":0.10,"iso_currency_code":"USD","unofficial_currency_code":null,"date":"2026-10-13","pending":false,"pending_transaction_id":null,"name":"Fictional purchase A"},{"account_id":"synthetic-flow-account","transaction_id":"synthetic-cent-b","amount":0.20,"iso_currency_code":"USD","unofficial_currency_code":null,"date":"2026-10-13","pending":false,"pending_transaction_id":null,"name":"Fictional purchase B"}],"modified":[],"removed":[],"next_cursor":"synthetic-cursor","has_more":false,"transactions_update_status":"HISTORICAL_UPDATE_COMPLETE","request_id":"synthetic-request"}`
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

// connectionBrowserFixture owns only disposable test dependencies and serializes its trigger.
type connectionBrowserFixture struct {
	mu        sync.Mutex
	service   *connections.Service
	runner    *acquisition.Runner
	admin     *pgx.Conn
	provider  *connectionProofProvider
	transport *connectionProofTransport
}

// bootstrapConnectionBrowser supplies creator evidence but leaves connection provisioning to the runtime API.
func bootstrapConnectionBrowser(t *testing.T, ctx context.Context, admin *pgx.Conn, pool *pgxpool.Pool) *connectionBrowserFixture {
	t.Helper()
	_, err := admin.Exec(ctx, `
 CREATE ROLE synthetic_connection_sync LOGIN IN ROLE bookkeeping_sync;
 CREATE ROLE synthetic_connection_publisher LOGIN IN ROLE bookkeeping_publisher;
 INSERT INTO ledger.entity(entity_id,name,kind) VALUES('10000000-0000-4000-8000-000000000002','Fictional Connection Proof LLC','legal');
 INSERT INTO ledger.book(entity_id,book_id,name) VALUES('10000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000003','Connection proof / USD');
 INSERT INTO ledger.membership(entity_id,principal_id,permission) SELECT '10000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', unnest(ARRAY['read','prepare','post']);
 INSERT INTO ledger.setup_receipt(principal_id,idempotency_key,normalized_request,result,entity_id,book_id,request_id)
 VALUES('00000000-0000-4000-8000-000000000001','synthetic-browser-creator','{}','{}','10000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000003',gen_random_uuid());
 INSERT INTO ledger.book_policy VALUES('10000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000003','10000000-0000-4000-8000-000000000004','2026-01-01','2027-01-01','00000000-0000-4000-8000-000000000002','accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1','00000000-0000-4000-8000-000000000001',clock_timestamp());
 INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES('10000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000003','10000000-0000-4000-8000-000000000006','2026-01-01','2027-01-01');
 INSERT INTO ledger.account VALUES('10000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000003','10000000-0000-4000-8000-000000000011','1000','Fictional connection bank','asset','debit',true,'2026-01-01',NULL),('10000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000003','10000000-0000-4000-8000-000000000012','5000','Fictional connection expenses','expense','debit',false,'2026-01-01',NULL);
 `)
	if err != nil {
		t.Fatal("synthetic connection setup failed", err)
	}
	workerPool, err := pgxpool.New(ctx, "host=database port=5432 user=synthetic_connection_sync dbname=app_proof sslmode=disable connect_timeout=5 pool_max_conns=2")
	if err != nil {
		t.Fatal("synthetic worker unavailable")
	}
	t.Cleanup(workerPool.Close)
	root := t.TempDir()
	if os.Chmod(root, 0700) != nil {
		t.Fatal("private evidence setup failed")
	}
	disk, err := staging.Open(root)
	if err != nil {
		t.Fatal("private evidence unavailable")
	}
	t.Cleanup(func() { disk.Close() })
	evidence, err := acquisition.NewLocalEvidence(disk)
	if err != nil {
		t.Fatal("evidence composition failed")
	}
	provider, transport := &connectionProofProvider{}, &connectionProofTransport{}
	store, err := bankcredential.NewAPIStore(pool, browserKeyWrapper{})
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := bankcredential.NewResolver(workerPool, browserKeyWrapper{}, func(context.Context) (acquisition.ProviderCredentials, error) {
		return acquisition.ProviderCredentials{ClientID: "synthetic-client", Secret: "synthetic-secret"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	client, err := acquisition.NewPlaidHTTP(transport)
	if err != nil {
		t.Fatal("synthetic provider composition failed")
	}
	runner, err := acquisition.New(workerPool, client, resolver, evidence)
	if err != nil {
		t.Fatal("synthetic runner composition failed")
	}
	service, err := connections.NewForBookCreators(pool, provider, store, connections.WorkerLogins{Sync: "synthetic_connection_sync", Publisher: "synthetic_connection_publisher"})
	if err != nil {
		t.Fatal("creator onboarding composition failed", err)
	}
	return &connectionBrowserFixture{service: service, runner: runner, admin: admin, provider: provider, transport: transport}
}

// ServeHTTP triggers only the single fictional binding; it accepts no caller scope or provider data.
func (f *connectionBrowserFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path != "/__proof/acquire" || r.Method != http.MethodPost || r.URL.RawQuery != "" {
		writeProblem(w, 404, "not_found")
		return
	}
	if r.Header.Get("Authorization") != "Bearer synthetic.browser.proof" {
		writeProblem(w, 401, "unauthenticated")
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		writeProblem(w, 400, "invalid_request")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64))
	if err != nil || strings.TrimSpace(string(raw)) != "{}" {
		writeProblem(w, 400, "invalid_request")
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var stream, source, revision string
	err = f.admin.QueryRow(ctx, `SELECT s.stream_id::text,s.source_account_id::text,a.binding_revision::text FROM integration.sync_stream s JOIN banking.source_account a USING(entity_id,book_id,source_account_id) WHERE s.entity_id=$1 AND s.book_id=$2`, connectionProofEntity, connectionProofBook).Scan(&stream, &source, &revision)
	if err != nil {
		writeProblem(w, 409, "binding_required")
		return
	}
	// Explicit test-owner review grant is separate from the product's connection binding grants.
	var granted bool
	err = f.admin.QueryRow(ctx, `SELECT EXISTS(SELECT FROM banking.source_permission WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3 AND principal_id=$4 AND capability='review' AND revoked_at IS NULL)`, connectionProofEntity, connectionProofBook, source, connectionProofActor).Scan(&granted)
	if err != nil {
		writeProblem(w, 500, "proof_failed")
		return
	}
	if !granted {
		body, _ := json.Marshal(map[string]any{"expected_binding_revision": revision, "principal_id": connectionProofActor, "enabled": true})
		_, err = demoOwnerCall(ctx, f.admin, `SELECT banking.set_source_review_permission($1,$2,$3,$4,'browser-review',$5)`, connectionProofActor, connectionProofEntity, connectionProofBook, source, string(body))
		if err != nil {
			writeProblem(w, 500, "proof_failed")
			return
		}
	}
	result, err := f.runner.Run(ctx, acquisition.Scope{EntityID: connectionProofEntity, BookID: connectionProofBook, StreamID: stream}, connectionProofClaim)
	if err != nil {
		writeProblem(w, 500, "proof_failed")
		return
	}
	var journals string
	if err = f.admin.QueryRow(ctx, `SELECT count(*)::text FROM ledger.journal_entry WHERE entity_id=$1 AND book_id=$2`, connectionProofEntity, connectionProofBook).Scan(&journals); err != nil {
		writeProblem(w, 500, "proof_failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": result.Status, "changed_count": result.ChangedCount, "fetch_count": f.transport.calls.Load(), "exchange_count": f.provider.exchanges.Load(), "journal_count": journals})
}

// wrap installs the synthetic trigger behind the same origin boundary, only in this test executable.
func (f *connectionBrowserFixture) wrap(product http.Handler) http.Handler {
	boundary, err := ledger.BrowserBoundary("http://localhost:39083", f)
	if err != nil {
		panic("fixed proof origin invalid")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/__proof/") {
			boundary.ServeHTTP(w, r)
			return
		}
		product.ServeHTTP(w, r)
	})
}

// browserKeyWrapper substitutes only the external KMS capability; SQL and envelope encryption are real.
type browserKeyWrapper struct{}

func (browserKeyWrapper) Wrap(_ context.Context, key, aad []byte) (string, []byte, error) {
	b, _ := aes.NewCipher(bytes.Repeat([]byte{7}, 32))
	g, _ := cipher.NewGCM(b)
	return "projects/123456789/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1", g.Seal(nil, make([]byte, 12), key, aad), nil
}
func (browserKeyWrapper) Unwrap(_ context.Context, version string, raw, aad []byte) ([]byte, error) {
	if version != "projects/123456789/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1" {
		return nil, bankcredential.ErrUnavailable
	}
	b, _ := aes.NewCipher(bytes.Repeat([]byte{7}, 32))
	g, _ := cipher.NewGCM(b)
	return g.Open(nil, make([]byte, 12), raw, aad)
}
