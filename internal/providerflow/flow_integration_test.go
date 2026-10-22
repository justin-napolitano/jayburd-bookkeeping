//go:build integration

package providerflow

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/server"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhook"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhookhost"
)

var proof = flag.Bool("provider-flow-proof", false, "Use only disposable synthetic provider-flow database")

const actor = "00000000-0000-4000-8000-000000000001"
const entity = "00000000-0000-4000-8000-000000000002"
const book = "00000000-0000-4000-8000-000000000003"
const currency = "00000000-0000-4000-8000-000000000005"
const account = "00000000-0000-4000-8000-000000000011"
const worker = "00000000-0000-4000-8000-000000000020"
const claim = "00000000-0000-4000-8000-000000000030"

// connect uses only the isolated test database, with no inherited DSN or credential discovery.
func connect(t *testing.T, user string) *pgx.Conn {
	t.Helper()
	if !*proof {
		t.Skip("use make provider-flow-test")
	}
	c, e := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=5432 user=%s dbname=provider_flow_template sslmode=disable connect_timeout=5", user))
	if e != nil {
		t.Fatal("synthetic database unavailable")
	}
	t.Cleanup(func() { c.Close(context.Background()) })
	return c
}

// exec installs fictional fixtures only, never runtime product authority.
func exec(t *testing.T, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, e := c.Exec(context.Background(), q, args...); e != nil {
		t.Fatal("synthetic setup failed", e)
	}
}

// scalar reads aggregate proof facts without outputting source payloads.
func scalar(t *testing.T, c *pgx.Conn, q string) string {
	t.Helper()
	var s string
	if e := c.QueryRow(context.Background(), q).Scan(&s); e != nil {
		t.Fatal("synthetic aggregate failed")
	}
	return s
}

// ownerCall provisions reviewed mandates through the actual owner-only boundary.
func ownerCall(t *testing.T, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	tx, e := c.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal("owner transaction failed")
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(context.Background(), "SET LOCAL ROLE bookkeeping_owner"); e != nil {
		t.Fatal("owner role failed")
	}
	var out []byte
	if e = tx.QueryRow(context.Background(), q, args...).Scan(&out); e != nil {
		t.Fatal("owner operation failed", e)
	}
	if tx.Commit(context.Background()) != nil {
		t.Fatal("owner commit failed")
	}
}

// identity authenticates only an original synthetic bearer value; it is not an identity emulator.
type identity struct{}

// Authenticate maps the test credential to a fixed fictional principal.
func (identity) Authenticate(_ context.Context, token string) (string, error) {
	if token != "synthetic-flow" {
		return "", errors.New("denied")
	}
	return actor, nil
}

// provider supplies verified fictional Link/account results and counts external mutations.
type provider struct{ exchanges int }

// CreateLink returns an ephemeral synthetic value and fixed bounded expiry.
func (*provider) CreateLink(context.Context, connections.LinkRequest) (connections.LinkResult, error) {
	return connections.LinkResult{Token: connections.NewSecret("synthetic-link"), ExpiresAt: time.Now().Add(5 * time.Minute)}, nil
}

// VerifySession binds only the known synthetic session and public token.
func (*provider) VerifySession(_ context.Context, link, public connections.Secret, expectedItem string) error {
	if expectedItem != "" || link != connections.NewSecret("synthetic-link") || public != connections.NewSecret("synthetic-public") {
		return errors.New("unverified")
	}
	return nil
}

// Exchange counts the side effect so replay cannot be mistaken for success.
func (p *provider) Exchange(context.Context, connections.Secret) (connections.ExchangeResult, error) {
	p.exchanges++
	return connections.ExchangeResult{ItemID: "synthetic-flow-item", AccessToken: connections.NewSecret("synthetic-access")}, nil
}

// Manifest returns only server-verified account identity; there are no client-supplied balances.
func (*provider) Manifest(context.Context, connections.Secret) (connections.Manifest, error) {
	code := "USD"
	return connections.Manifest{ItemID: "synthetic-flow-item", Accounts: []connections.Account{{AccountID: "synthetic-flow-account", Type: "depository", Currency: &code}}, ConsentExpiresAt: "2099-01-01T00:00:00Z"}, nil
}

// Rotate is deliberately unavailable in this enrollment-to-review scenario.
func (*provider) Rotate(context.Context, connections.Secret) (connections.ExchangeResult, error) {
	return connections.ExchangeResult{}, errors.New("not in scenario")
}

// Remove is deliberately unavailable; disconnect must never call it.
func (*provider) Remove(context.Context, connections.Secret) error {
	return errors.New("not in scenario")
}

// secretStore retains only this test's synthetic token in memory behind the production interface.
type secretStore struct {
	stored        connections.StoredCredential
	writes, reads int
}

// Put requires immutable reference identity and exact synthetic replay.
func (s *secretStore) Put(_ context.Context, r connections.CredentialReference, token connections.Secret) error {
	if s.writes > 0 && (s.stored.Reference != r || s.stored.Token != token) {
		return errors.New("conflict")
	}
	s.stored = connections.StoredCredential{Reference: r, Token: token}
	s.writes++
	return nil
}

// Read returns only the exact immutable version requested.
func (s *secretStore) Read(_ context.Context, r connections.CredentialReference) (connections.StoredCredential, error) {
	s.reads++
	if s.writes == 0 || s.stored.Reference != r {
		return connections.StoredCredential{}, errors.New("missing")
	}
	return s.stored, nil
}

// Resolve checks the same stored reference before furnishing fictional runner credentials.
func (s *secretStore) Resolve(_ context.Context, r acquisition.CredentialReference) (acquisition.ProviderCredentials, error) {
	stored := s.stored.Reference
	if s.writes == 0 || stored.ReferenceID != r.ReferenceID || stored.Version != r.Version || stored.ItemID != r.ItemID || stored.Environment != r.Environment {
		return acquisition.ProviderCredentials{}, errors.New("denied")
	}
	credentials := acquisition.ProviderCredentials{ClientID: "synthetic-client", Secret: "synthetic-secret"}
	err := s.stored.Token.WithValue(func(value string) error { credentials.AccessToken = value; return nil })
	return credentials, err
}

// transport intercepts the pinned production adapter; no request reaches Plaid.
type transport struct{ calls int }

// RoundTrip verifies account filtering and serves exact original decimal bytes.
func (p *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	p.calls++
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

// request exercises authenticated HTTP and checks public output never exposes credential internals.
func request(t *testing.T, s *server.Server, method, path, key string, body any, want int) map[string]json.RawMessage {
	t.Helper()
	var payload io.Reader
	if body != nil {
		raw, e := json.Marshal(body)
		if e != nil {
			t.Fatal("request fixture")
		}
		payload = strings.NewReader(string(raw))
	}
	r := httptest.NewRequest(method, path, payload)
	r.Header.Set("Authorization", "Bearer synthetic-flow")
	r.Header.Set("Origin", "http://localhost:3000")
	r.Header.Set("Content-Type", "application/json")
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	s.HTTP.Handler.ServeHTTP(w, r)
	if w.Code != want {
		var problem struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &problem)
		t.Fatalf("HTTP %s: status %d code %s", method, w.Code, problem.Code)
	}
	for _, forbidden := range []string{"credential_ref", "credential_version", "claim_id", "token_sha256", "synthetic-access", "synthetic-secret"} {
		if strings.Contains(w.Body.String(), forbidden) {
			t.Fatal("private connection metadata leaked")
		}
	}
	var out map[string]json.RawMessage
	if json.Unmarshal(w.Body.Bytes(), &out) != nil {
		t.Fatal("invalid public result")
	}
	return out
}

// stringField decodes identifiers without coercing monetary numbers.
func stringField(t *testing.T, v json.RawMessage) string {
	t.Helper()
	var s string
	if json.Unmarshal(v, &s) != nil {
		t.Fatal("invalid string field")
	}
	return s
}

// flowWebhookKey provides only an ephemeral public verification key for the fictional socket flow.
type flowWebhookKey struct{ key webhook.JWK }

// Get performs no provider lookup and fails if the signed key ID changes.
func (k flowWebhookKey) Get(_ context.Context, id string) (webhook.JWK, error) {
	if id != k.key.ID {
		return webhook.JWK{}, errors.New("synthetic key unavailable")
	}
	return k.key, nil
}

// startFlowWebhook composes the real SQL receiver and listener using only a restricted test login.
func startFlowWebhook(t *testing.T, db *pgx.Conn) (string, string, string) {
	t.Helper()
	private, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal("synthetic signing key failed")
	}
	body := `{"webhook_type":"TRANSACTIONS","webhook_code":"SYNC_UPDATES_AVAILABLE","item_id":"synthetic-flow-item","environment":"sandbox","initial_update_complete":true,"historical_update_complete":true}`
	digest := sha256.Sum256([]byte(body))
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"iat": time.Now().Unix(), "request_body_sha256": hex.EncodeToString(digest[:])})
	token.Header["kid"] = "synthetic-flow-key"
	signed, e := token.SignedString(private)
	if e != nil {
		t.Fatal("synthetic signature failed")
	}
	keys := flowWebhookKey{webhook.JWK{Alg: "ES256", Curve: "P-256", ID: "synthetic-flow-key", Type: "EC", Use: "sig", X: base64.RawURLEncoding.EncodeToString(private.X.FillBytes(make([]byte, 32))), Y: base64.RawURLEncoding.EncodeToString(private.Y.FillBytes(make([]byte, 32))), Created: time.Now().Add(-time.Hour).Unix()}}
	host, e := webhookhost.NewSQL(keys, db, nil)
	if e != nil {
		t.Fatal("SQL webhook composition failed")
	}
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal("synthetic listener failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- host.Serve(ctx, listener) }()
	t.Cleanup(func() {
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Error("webhook shutdown failed")
			}
		case <-time.After(4 * time.Second):
			t.Error("webhook shutdown timed out")
		}
	})
	return "http://" + listener.Addr().String() + "/webhooks/plaid", signed, body
}

// postFlowWebhook sends exact fictional signed bytes and exposes only status to the test.
func postFlowWebhook(t *testing.T, url, token, body string) int {
	t.Helper()
	r, e := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if e != nil {
		t.Fatal("synthetic webhook request failed")
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Plaid-Verification", token)
	client := &http.Client{Timeout: 3 * time.Second}
	response, e := client.Do(r)
	if e != nil {
		t.Fatal("synthetic webhook delivery failed")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1024))
	return response.StatusCode
}

// TestProviderEnrollmentAcquisitionFlow connects real HTTP, enrollment SQL, durable evidence and runner SQL.
func TestProviderEnrollmentAcquisitionFlow(t *testing.T) {
	c := connect(t, "postgres")
	runtime := connect(t, "synthetic_runtime")
	sync := connect(t, "synthetic_sync")
	operator := connect(t, "synthetic_operator")
	exec(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','flow-owner'),($2,'synthetic','flow-worker')`, actor, worker)
	exec(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional Flow LLC','legal')`, entity)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$2,'import',NULL),($1,$3,'read',NULL),($1,$3,'import',NULL)`, entity, actor, worker)
	exec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Fictional Flow Book')`, entity, book)
	exec(t, c, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic')`, currency)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'1000','Fictional bank','asset','debit',true,'2026-01-01',NULL)`, entity, book, account)
	ownerCall(t, operator, `SELECT integration.set_connect_permission($1,$2,$3,$1,'flow-connect',true)`, actor, entity, book)
	ownerCall(t, operator, `SELECT integration.set_connection_worker($1,$2,$3,'flow-worker',$4)`, actor, entity, book, `{"service_principal_id":"`+worker+`","worker_login":"synthetic_sync","enabled":true}`)
	p := &provider{}
	encrypted, e := bankcredential.NewAPIStore(runtime, flowWrapper{})
	if e != nil {
		t.Fatal(e)
	}
	store := &lostCredentialAck{base: encrypted}
	service := connections.New(runtime, p, store)
	app, e := server.New(server.Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "synthetic-project", DatabaseURL: "synthetic"}, server.Dependencies{Auth: identity{}, Operations: &ledger.Service{DB: runtime}, Connections: service, Ready: func(context.Context) error { return nil }})
	if e != nil {
		t.Fatal("server composition failed")
	}
	base := "/v1/entities/" + entity + "/books/" + book
	begin := request(t, app, "POST", base+"/connection-attempts", "flow-begin", connections.BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}}, 200)
	var attempt map[string]json.RawMessage
	_ = json.Unmarshal(begin["attempt"], &attempt)
	id := stringField(t, attempt["attempt_id"])
	nonce := stringField(t, begin["nonce"])
	link := stringField(t, begin["link_token"])
	execute := connections.ExecuteRequest{Nonce: nonce, LinkToken: link, PublicToken: "synthetic-public"}
	request(t, app, "POST", base+"/connection-attempts/"+id+"/execute", "", execute, 409)
	request(t, app, "POST", base+"/connection-attempts/"+id+"/reconcile", "", map[string]any{}, 200)
	request(t, app, "POST", base+"/connection-attempts/"+id+"/execute", "", execute, 200)
	if p.exchanges != 1 || scalar(t, c, "SELECT count(*)::text FROM integration.bank_credential") != "1" {
		t.Fatal("replayed provider exchange or secret write")
	}
	bound := request(t, app, "POST", base+"/connection-attempts/"+id+"/bindings", "flow-bind", connections.BindingRequest{ConsentGrants: true, Bindings: []connections.Binding{{AccountID: "synthetic-flow-account", LedgerAccountID: account, CurrencyVersionID: currency, BindingDate: "2026-10-13", BindingEvidenceRef: claim}}}, 200)
	var streams []struct {
		StreamID string `json:"stream_id"`
		SourceID string `json:"source_account_id"`
		Overlap  bool   `json:"overlap_review_required"`
	}
	if json.Unmarshal(bound["streams"], &streams) != nil || len(streams) != 1 || streams[0].Overlap {
		t.Fatal("provider-first source not reviewable")
	}
	// The actual socket, signature verifier and SQL receiver must share one durable wakeup boundary.
	webhookDB := connect(t, "synthetic_webhook")
	ownerCall(t, operator, `SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','flow-webhook',true)`)
	webhookURL, signed, notification := startFlowWebhook(t, webhookDB)
	if scalar(t, c, `SELECT requested_generation::text FROM integration.sync_stream`) != "1" {
		t.Fatal("unexpected initial generation")
	}
	for _, bad := range []struct{ token, body string }{{"invalid", notification}, {signed, notification + " "}} {
		if postFlowWebhook(t, webhookURL, bad.token, bad.body) != 401 {
			t.Fatal("invalid notification accepted")
		}
	}
	if scalar(t, c, `SELECT count(*)::text FROM integration.webhook_receipt`) != "0" || scalar(t, c, `SELECT requested_generation::text FROM integration.sync_stream`) != "1" {
		t.Fatal("unverified socket delivery mutated SQL")
	}
	for range 2 {
		if postFlowWebhook(t, webhookURL, signed, notification) != 204 {
			t.Fatal("durable socket delivery rejected")
		}
	}
	if scalar(t, c, `SELECT count(*)::text FROM integration.webhook_receipt`) != "1" || scalar(t, c, `SELECT requested_generation::text FROM integration.sync_stream`) != "2" || scalar(t, c, `SELECT count(*)::text FROM integration.sync_wakeup WHERE reason='webhook'`) != "1" {
		t.Fatal("notification replay duplicated receipt or wakeup")
	}
	httpProvider := &transport{}
	client, e := acquisition.NewPlaidHTTP(httpProvider)
	if e != nil {
		t.Fatal("provider composition failed")
	}
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal("private directory setup failed")
	}
	disk, e := staging.Open(directory)
	if e != nil {
		t.Fatal("private storage failed")
	}
	defer disk.Close()
	evidence, e := acquisition.NewLocalEvidence(disk)
	if e != nil {
		t.Fatal("evidence composition failed")
	}
	resolver, e := bankcredential.NewResolver(sync, flowWrapper{}, func(context.Context) (acquisition.ProviderCredentials, error) {
		return acquisition.ProviderCredentials{ClientID: "synthetic-client", Secret: "synthetic-secret"}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	runner, e := acquisition.New(sync, client, resolver, evidence)
	if e != nil {
		t.Fatal("runner composition failed")
	}
	scope := acquisition.Scope{EntityID: entity, BookID: book, StreamID: streams[0].StreamID}
	out, e := runner.Run(context.Background(), scope, claim)
	if e != nil || out.Status != "completed" || out.ChangedCount != "2" {
		t.Fatal("acquisition did not publish", e)
	}
	out, e = runner.Run(context.Background(), scope, claim)
	if e != nil || out.Status != "completed" || httpProvider.calls != 1 {
		t.Fatal("replay refetched provider or failed")
	}
	if scalar(t, c, `SELECT count(*)::text FROM ledger.journal_entry`) != "0" || scalar(t, c, `SELECT count(*)::text FROM integration.outbox_message`) != "0" {
		t.Fatal("acquisition posted a journal")
	}
	if scalar(t, c, `SELECT sum(r.amount)::text FROM banking.source_observation o JOIN banking.observation_revision r ON (r.entity_id,r.book_id,r.observation_id,r.revision)=(o.entity_id,o.book_id,o.observation_id,o.current_revision)`) != "0.300000000" {
		t.Fatal("exact cents changed")
	}
	if scalar(t, c, `SELECT count(*)::text FROM banking.review_item WHERE state='needs_review'`) != "2" {
		t.Fatal("acquired evidence not reviewable")
	}
	request(t, app, "GET", base+"/connections", "", nil, 200)
	request(t, app, "GET", base+"/review-items?source_account_id="+streams[0].SourceID+"&state=needs_review", "", nil, 200)
	request(t, app, "POST", base+"/connections/streams/"+scope.StreamID+"/disconnect", "flow-disconnect", map[string]string{"expected_binding_revision": "1"}, 200)
	if postFlowWebhook(t, webhookURL, signed, notification) != 403 || scalar(t, c, `SELECT count(*)::text FROM integration.webhook_receipt`) != "1" || scalar(t, c, `SELECT requested_generation::text FROM integration.sync_stream`) != "2" {
		t.Fatal("disconnected stream accepted replay or mutated wakeup")
	}
	_, e = runner.Run(context.Background(), scope, claim)
	if e == nil || httpProvider.calls != 1 {
		t.Fatal("disconnected stream retained acquisition authority")
	}
	ownerCall(t, operator, `SELECT integration.set_connect_permission($1,$2,$3,$1,'flow-revoke',false)`, actor, entity, book)
	request(t, app, "GET", base+"/connection-attempts/"+id, "", nil, 404)
}

// flowWrapper provides a fictional KEK while exercising production envelope AEAD and SQL storage.
type flowWrapper struct{}

func (flowWrapper) Wrap(_ context.Context, key, aad []byte) (string, []byte, error) {
	block, _ := aes.NewCipher(bytes.Repeat([]byte{7}, 32))
	g, _ := cipher.NewGCM(block)
	return "projects/jayburd-synthetic-test/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1", g.Seal(nil, make([]byte, 12), key, aad), nil
}
func (flowWrapper) Unwrap(_ context.Context, version string, raw, aad []byte) ([]byte, error) {
	if version != "projects/jayburd-synthetic-test/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1" {
		return nil, bankcredential.ErrUnavailable
	}
	block, _ := aes.NewCipher(bytes.Repeat([]byte{7}, 32))
	g, _ := cipher.NewGCM(block)
	return g.Open(nil, make([]byte, 12), raw, aad)
}

// lostCredentialAck models a committed ciphertext whose acknowledgement never reaches the service.
type lostCredentialAck struct {
	base *bankcredential.APIStore
	lost bool
}
type lostScopedAck struct {
	root  *lostCredentialAck
	inner connections.SecretStore
}

func (s *lostCredentialAck) ForScope(scope connections.Scope) connections.SecretStore {
	return &lostScopedAck{s, s.base.ForScope(scope)}
}
func (s *lostCredentialAck) Put(context.Context, connections.CredentialReference, connections.Secret) error {
	return bankcredential.ErrUnavailable
}
func (s *lostCredentialAck) Read(context.Context, connections.CredentialReference) (connections.StoredCredential, error) {
	return connections.StoredCredential{}, bankcredential.ErrUnavailable
}
func (s *lostScopedAck) Put(ctx context.Context, r connections.CredentialReference, v connections.Secret) error {
	if e := s.inner.Put(ctx, r, v); e != nil {
		return e
	}
	if !s.root.lost {
		s.root.lost = true
		return bankcredential.ErrUnavailable
	}
	return nil
}
func (s *lostScopedAck) Read(ctx context.Context, r connections.CredentialReference) (connections.StoredCredential, error) {
	return s.inner.Read(ctx, r)
}
