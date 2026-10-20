package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// credentialFunc injects synthetic credential behavior without reading external secrets.
type credentialFunc func(context.Context, CredentialReference) (ProviderCredentials, error)

// Resolve forwards the bounded synthetic resolution.
func (f credentialFunc) Resolve(c context.Context, r CredentialReference) (ProviderCredentials, error) {
	return f(c, r)
}

// providerFunc injects deterministic provider faults.
type providerFunc func(context.Context, FetchRequest) ([]byte, error)

// Fetch forwards the injected behavior without introducing retry.
func (f providerFunc) Fetch(c context.Context, r FetchRequest) ([]byte, error) { return f(c, r) }

// roundTripFunc replaces network routing only inside synthetic tests.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip forwards the explicitly injected fake or loopback transport.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// credentialsFixture returns fictional in-memory token material.
func credentialsFixture() ProviderCredentials {
	return ProviderCredentials{"synthetic-access", "synthetic-client", "synthetic-secret"}
}

// fixturePage emits original small synthetic provider JSON with numeric lexemes kept verbatim.
func fixturePage(id, amount, currency, cursor string, more bool) []byte {
	return []byte(fmt.Sprintf(`{"accounts":[{"account_id":"Account-A","type":"depository"}],"added":[{"account_id":"Account-A","transaction_id":%q,"amount":%s,"iso_currency_code":%q,"unofficial_currency_code":null,"date":"2026-01-02","pending":false,"pending_transaction_id":null,"name":"Fictional supplies"}],"modified":[],"removed":[],"next_cursor":%q,"has_more":%t,"request_id":"synthetic-request","transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`, id, amount, currency, cursor, more))
}

// fakeRepository models metadata/commit faults while real SQL is tested separately.
type fakeRepository struct {
	decoderCode                   string
	context                       workContext
	state                         string
	key                           string
	reason                        *string
	pages                         []json.RawMessage
	calls, claims                 int
	denied                        bool
	stageUnknown, completeUnknown bool
}

// RejectDecode models an atomic fixed-code diagnostic and abandonment without provider data.
func (f *fakeRepository) RejectDecode(ctx context.Context, s Scope, a attempt, code string) error {
	if err := f.Abandon(ctx, s, a, "decoder_rejected"); err != nil {
		return err
	}
	f.decoderCode = code
	return nil
}

// contextFixture supplies independent trusted metadata and exact engineering limits.
func contextFixture() workContext {
	var c workContext
	c.StreamID = testStream
	c.SetID = testSet
	c.Fence = "1"
	c.Provider = "plaid"
	c.Environment = "sandbox"
	c.ItemID = "Item-A"
	c.AccountID = "Account-A"
	c.ItemGeneration = "1"
	c.CredentialRef = testClaim
	c.CredentialVersion = "1"
	c.BindingRevision = "1"
	c.LeaseUntil = time.Now().Add(90 * time.Second)
	c.Currency.Code = "USD"
	c.Currency.VersionID = testEntity
	c.Currency.MinorUnit = 2
	c.Currency.ValidFrom = "2000-01-01"
	c.Limits.MaxPages = 10
	c.Limits.MaxChanges = 1000
	c.Limits.MaxBytes = 2 << 20
	c.Limits.MaxPageChanges = 500
	return c
}

// Resume mirrors current authorization before receipt resolution.
func (f *fakeRepository) Resume(_ context.Context, s Scope, key string) (resume, error) {
	if f.denied {
		return resume{}, failure("not_authorized", false)
	}
	if key != f.key || f.state == "" {
		return resume{State: "absent"}, nil
	}
	out := resume{State: f.state, SetID: &f.context.SetID, Fence: &f.context.Fence, Reason: f.reason}
	if f.state == "completed" {
		c := f.receipt()
		out.Completion = &c
	}
	return out, nil
}

// Claim initializes a new independent attempt from the last committed cursor.
func (f *fakeRepository) Claim(_ context.Context, s Scope, key string) (attempt, error) {
	f.claims++
	f.key = key
	f.state = "staging"
	f.context.SetID = stableID("fake-set", key)
	f.context.PageCount = 0
	f.context.NextPageIndex = 0
	f.context.ByteCount = 0
	f.context.ChangeCount = 0
	f.context.FinalPage = false
	f.context.RequestCursor = f.context.StartingCursor
	f.pages = nil
	return attempt{f.context.SetID, f.context.Fence}, nil
}

// Context refuses stale/denied metadata before any external call.
func (f *fakeRepository) Context(ctx context.Context, s Scope, a attempt) (workContext, error) {
	f.calls++
	if f.denied {
		return workContext{}, failure("not_authorized", false)
	}
	if ctx.Err() != nil {
		return workContext{}, failure("cancelled", true)
	}
	return f.context, nil
}

// Stage can lose a committed acknowledgement while retaining its exact page position.
func (f *fakeRepository) Stage(_ context.Context, s Scope, a attempt, i int, raw json.RawMessage) error {
	var p pageWire
	if json.Unmarshal(raw, &p) != nil {
		return failure("invalid_update", false)
	}
	f.pages = append(f.pages, raw)
	f.context.PageCount++
	f.context.NextPageIndex++
	f.context.ByteCount += len(raw)
	f.context.ChangeCount += len(p.Added) + len(p.Modified) + len(p.Removed)
	f.context.RequestCursor = nextCursor(p.NextCursor)
	f.context.FinalPage = !p.HasMore
	if f.stageUnknown {
		f.stageUnknown = false
		return failure("commit_unknown", true)
	}
	return nil
}

// receipt returns a fixed fake publication result without performing ledger work.
func (f *fakeRepository) receipt() completion {
	return completion{StreamID: testStream, SetID: f.context.SetID, Cursor: f.context.RequestCursor, CompletedGeneration: "1", SourceRevision: "1", ChangedCount: fmt.Sprint(f.context.ChangeCount), Readiness: "HISTORICAL_UPDATE_COMPLETE"}
}

// Complete can suppress a committed receipt once; resume must resolve it without refetch.
func (f *fakeRepository) Complete(_ context.Context, s Scope, a attempt) (completion, error) {
	f.state = "completed"
	if f.completeUnknown {
		f.completeUnknown = false
		return completion{}, failure("commit_unknown", true)
	}
	return f.receipt(), nil
}

// Abandon fences the fake set before private evidence cleanup.
func (f *fakeRepository) Abandon(_ context.Context, s Scope, a attempt, reason string) error {
	f.state = "abandoned"
	f.reason = &reason
	return nil
}

// runnerFixture composes private storage and test metadata with a real runner.
func runnerFixture(t testing.TB, repo *fakeRepository, p Provider) *Runner {
	return &Runner{repo: repo, provider: p, credentials: credentialFunc(func(context.Context, CredentialReference) (ProviderCredentials, error) {
		return credentialsFixture(), nil
	}), evidence: localEvidence(t), slots: make(chan struct{}, 1)}
}

// syntheticHTTP runs a real loopback server while checking that production routing remains pinned.
func syntheticHTTP(t testing.TB, handler http.HandlerFunc) *PlaidHTTP {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, _ := url.Parse(server.URL)
	p, e := NewPlaidHTTP(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://sandbox.plaid.com/transactions/sync" || r.Header.Get("Plaid-Version") != "2020-09-14" {
			t.Fatal("unpinned provider request")
		}
		copy := r.Clone(r.Context())
		u := *r.URL
		u.Scheme = target.Scheme
		u.Host = target.Host
		copy.URL = &u
		return server.Client().Transport.RoundTrip(copy)
	}))
	if e != nil {
		t.Fatal(e)
	}
	return p
}

// providerRequest verifies synthetic credentials and the mandatory account-filtered request.
func providerRequest(t testing.TB, r *http.Request) *string {
	t.Helper()
	body, e := io.ReadAll(r.Body)
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		ClientID string  `json:"client_id"`
		Secret   string  `json:"secret"`
		Access   string  `json:"access_token"`
		Cursor   *string `json:"cursor"`
		Count    int     `json:"count"`
		Options  struct {
			Account     string `json:"account_id"`
			Description bool   `json:"include_original_description"`
		} `json:"options"`
	}
	if json.Unmarshal(body, &v) != nil || r.Method != "POST" || v.ClientID != "synthetic-client" || v.Secret != "synthetic-secret" || v.Access != "synthetic-access" || v.Options.Account != "Account-A" || !v.Options.Description || v.Count != 500 {
		t.Fatal("provider request shape")
	}
	return v.Cursor
}

// TestPlaidExactMoney exercises original HTTP numeric lexemes through runner, decoder and durable raw storage.
func TestPlaidExactMoney(t *testing.T) {
	for _, tc := range []struct {
		code, lexeme, amount, direction string
		minor                           int
	}{{"USD", "9007199254740993.01", "9007199254740993.01", "outflow", 2}, {"USD", "-1.255e2", "125.5", "inflow", 2}, {"JPY", "125", "125", "outflow", 0}, {"KWD", "1.005", "1.005", "outflow", 3}} {
		t.Run(tc.code+tc.lexeme, func(t *testing.T) {
			raw := fixturePage("Txn-A", tc.lexeme, tc.code, "C1", false)
			p := syntheticHTTP(t, func(w http.ResponseWriter, r *http.Request) {
				if providerRequest(t, r) != nil {
					t.Fatal("initial cursor")
				}
				w.Write(raw)
			})
			repo := &fakeRepository{context: contextFixture()}
			repo.context.Currency.Code = tc.code
			repo.context.Currency.MinorUnit = tc.minor
			runner := runnerFixture(t, repo, p)
			out, e := runner.Run(context.Background(), scopeFixture(), testClaim)
			if e != nil || out.Status != "completed" {
				t.Fatal(out, e)
			}
			var page pageWire
			if json.Unmarshal(repo.pages[0], &page) != nil || page.Added[0].Amount != tc.amount || page.Added[0].AmountLexeme != tc.lexeme || page.Added[0].Direction != tc.direction {
				t.Fatal("money changed")
			}
			saved, found, e := runner.evidence.Get(context.Background(), EvidenceKey{Scope: scopeFixture(), SetID: repo.context.SetID, Page: 0})
			if e != nil || !found || string(saved) != string(raw) {
				t.Fatal("original bytes lost")
			}
		})
	}
}

// TestPlaidCompleteUpdate proves mutation restarts C0 and publication waits for a complete successful loop.
func TestPlaidCompleteUpdate(t *testing.T) {
	calls := 0
	p := syntheticHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		cursor := providerRequest(t, r)
		calls++
		switch calls {
		case 1, 3:
			if cursor != nil {
				t.Fatal("did not restart C0")
			}
			w.Write(fixturePage("Txn-A", "1.01", "USD", "C1", true))
		case 2:
			if cursor == nil || *cursor != "C1" {
				t.Fatal("wrong continuation")
			}
			w.WriteHeader(400)
			io.WriteString(w, `{"error_code":"TRANSACTIONS_SYNC_MUTATION_DURING_PAGINATION"}`)
		case 4:
			if cursor == nil || *cursor != "C1" {
				t.Fatal("wrong restart continuation")
			}
			w.Write(fixturePage("Txn-B", "-2.02", "USD", "C2", false))
		default:
			t.Fatal("unexpected request")
		}
	})
	repo := &fakeRepository{context: contextFixture()}
	runner := runnerFixture(t, repo, p)
	out, e := runner.Run(context.Background(), scopeFixture(), testClaim)
	if e != nil || out.ChangedCount != "2" || repo.claims != 2 || calls != 4 || len(repo.pages) != 2 {
		t.Fatal(out, e, calls)
	}
	out, e = runner.Run(context.Background(), scopeFixture(), stableID("acquisition-mutation-restart-v1", testEntity, testBook, testStream, testClaim))
	if e != nil || out.Status != "completed" || calls != 4 {
		t.Fatal("completed replay refetched")
	}
}

// TestRunnerDecodeDiagnostic keeps a fixed reason and abandons before any publication.
func TestRunnerDecodeDiagnostic(t *testing.T) {
	repo := &fakeRepository{context: contextFixture()}
	repo.context.Currency.ValidFrom = "2026-09-17"
	runner := runnerFixture(t, repo, providerFunc(func(context.Context, FetchRequest) ([]byte, error) {
		return fixturePage("Txn-A", "1.01", "USD", "C1", false), nil
	}))
	out, err := runner.Run(context.Background(), scopeFixture(), testClaim)
	if err == nil || err.Error() != "decoder_rejected" || out.Status != "abandoned" || repo.decoderCode != "unsupported_currency" || len(repo.pages) != 0 {
		t.Fatal("decoder reason lost or partially published", out, err, repo.decoderCode)
	}
}

// TestRunnerUnknownCommitAndRevocation suppresses refetch on uncertain commits and denied scope.
func TestRunnerUnknownCommitAndRevocation(t *testing.T) {
	for _, where := range []string{"stage", "complete"} {
		t.Run(where, func(t *testing.T) {
			calls := 0
			repo := &fakeRepository{context: contextFixture(), stageUnknown: where == "stage", completeUnknown: where == "complete"}
			p := providerFunc(func(context.Context, FetchRequest) ([]byte, error) {
				calls++
				return fixturePage("Txn-A", "1.01", "USD", "C1", false), nil
			})
			runner := runnerFixture(t, repo, p)
			if _, e := runner.Run(context.Background(), scopeFixture(), testClaim); e == nil || e.Error() != "commit_unknown" {
				t.Fatal(e)
			}
			out, e := runner.Run(context.Background(), scopeFixture(), testClaim)
			if e != nil || out.Status != "completed" || calls != 1 {
				t.Fatal("unknown commit refetched", out, e, calls)
			}
			repo.denied = true
			if _, e := runner.Run(context.Background(), scopeFixture(), testClaim); e == nil || calls != 1 {
				t.Fatal("denied replay disclosed receipt")
			}
		})
	}
	repo := &fakeRepository{context: contextFixture()}
	calls := 0
	runner := runnerFixture(t, repo, providerFunc(func(context.Context, FetchRequest) ([]byte, error) {
		calls++
		repo.denied = true
		return fixturePage("Txn-A", "1.01", "USD", "C1", false), nil
	}))
	if _, e := runner.Run(context.Background(), scopeFixture(), testClaim); e == nil || len(repo.pages) != 0 {
		t.Fatal("post-fetch revocation staged")
	}
	if _, found, _ := runner.evidence.Get(context.Background(), EvidenceKey{Scope: scopeFixture(), SetID: repo.context.SetID, Page: 0}); found {
		t.Fatal("revoked response retained")
	}
	runner.credentials = credentialFunc(func(context.Context, CredentialReference) (ProviderCredentials, error) {
		return ProviderCredentials{}, errors.New("private-secret-path")
	})
	repo.denied = false
	repo.state = ""
	if _, e := runner.Run(context.Background(), scopeFixture(), testClaim); e == nil || strings.Contains(e.Error(), "private") || calls != 1 {
		t.Fatal("credential failure leaked or fetched")
	}
}
