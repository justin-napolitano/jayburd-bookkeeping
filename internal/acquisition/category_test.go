package acquisition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
)

// categoryPageFixture preserves exact synthetic money while varying only current provider metadata.
func categoryPageFixture(metadata string) []byte {
	return bytes.Replace(fixturePage("Txn-A", "0.10", "USD", "C1", false), []byte(`"name":"Fictional supplies"`), []byte(`"name":"Fictional supplies","personal_finance_category":`+metadata), 1)
}

// TestCategoryRunnerResume binds every staged attribution to the financial page across a lost stage reply.
func TestCategoryRunnerResume(t *testing.T) {
	raw := categoryPageFixture(`{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"HIGH","version":"v2"}`)
	repo := &fakeRepository{context: contextFixture(), stageUnknown: true}
	calls := 0
	runner := runnerFixture(t, repo, providerFunc(func(context.Context, FetchRequest) ([]byte, error) { calls++; return raw, nil }))
	runner.categories = true
	scope := Scope{testEntity, testBook, testStream}
	if _, e := runner.Run(context.Background(), scope, testClaim); e == nil {
		t.Fatal("lost stage reply became success")
	}
	if _, e := runner.Run(context.Background(), scope, testClaim); e != nil {
		t.Fatal(e)
	}
	if calls != 1 || len(repo.pages) != 1 {
		t.Fatal("resume refetched or duplicated page")
	}
	var p pageWire
	if json.Unmarshal(repo.pages[0], &p) != nil || p.CategoryAttribution == nil {
		t.Fatal("missing sidecar")
	}
	c := p.CategoryAttribution
	if c.SourcePageSHA256 != p.SHA256 || len(c.Attributions) != 1 || c.Attributions[0].AccountID != "Account-A" || c.Attributions[0].TransactionID != "Txn-A" || c.Attributions[0].Change != "added" || *c.Attributions[0].Category.Version != "v2" {
		t.Fatal("sidecar identity drift")
	}
	if p.Added[0].Amount != "0.1" {
		t.Fatal("financial amount changed")
	}
	// A completed replay exposes the same receipt without fetching or staging again.
	if _, e := runner.Run(context.Background(), scope, testClaim); e != nil || calls != 1 || len(repo.pages) != 1 {
		t.Fatal("completed replay changed effects")
	}
}

// TestCategoryRunnerMalformedAndLegacy proves opt-in whole-page rejection and unchanged legacy inertness.
func TestCategoryRunnerMalformedAndLegacy(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		repo := &fakeRepository{context: contextFixture()}
		r := runnerFixture(t, repo, providerFunc(func(context.Context, FetchRequest) ([]byte, error) { return categoryPageFixture(`{"version":4}`), nil }))
		r.categories = enabled
		_, e := r.Run(context.Background(), Scope{testEntity, testBook, testStream}, testClaim)
		if enabled {
			if e == nil || len(repo.pages) != 0 || repo.decoderCode != "invalid_field" || repo.state != "abandoned" {
				t.Fatal("malformed category escaped atomic hold")
			}
		} else {
			if e != nil || len(repo.pages) != 1 || bytes.Contains(repo.pages[0], []byte("category_attribution")) {
				t.Fatal("legacy wire changed")
			}
		}
	}
}

// TestCategoryHTTPExplicitVersion proves the opt-in adds only the fixed v2 option without network I/O.
func TestCategoryHTTPExplicitVersion(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://sandbox.plaid.com/transactions/sync" {
				t.Fatal("endpoint changed")
			}
			var body map[string]json.RawMessage
			if json.NewDecoder(r.Body).Decode(&body) != nil {
				t.Fatal("body")
			}
			var options map[string]any
			_ = json.Unmarshal(body["options"], &options)
			if options["account_id"] != "Account-A" || options["include_original_description"] != true {
				t.Fatal("scope changed")
			}
			if enabled {
				if len(options) != 3 || options["personal_finance_category_version"] != "v2" {
					t.Fatal("missing exact version")
				}
			} else if len(options) != 2 {
				t.Fatal("legacy options changed")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		})
		constructor := NewPlaidHTTPForProfile
		if enabled {
			constructor = NewPlaidHTTPForProfileWithCategories
		}
		p, e := constructor(plaidprofile.Sandbox(), transport)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = p.Fetch(context.Background(), FetchRequest{AccountID: "Account-A", Environment: "sandbox", Credentials: credentialsFixture()}); e != nil {
			t.Fatal(e)
		}
	}
}

type categorySQLDB struct {
	faultDB
	query string
}
type categorySQLTx struct {
	*transactionFault
	owner *categorySQLDB
}
type categoryErrorRow struct{}

func (categoryErrorRow) Scan(...any) error { return errors.New("synthetic metadata stop") }
func (d *categorySQLDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, e := d.faultDB.BeginTx(ctx, o)
	if e != nil {
		return nil, e
	}
	return &categorySQLTx{tx.(*transactionFault), d}, nil
}
func (tx *categorySQLTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	if q == syncRoleQuery {
		return tx.transactionFault.QueryRow(ctx, q, args...)
	}
	tx.owner.query = q
	return categoryErrorRow{}
}

// TestCategorySQLFunctionSelection preserves role checks while selecting only explicit additive functions.
func TestCategorySQLFunctionSelection(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		db := &categorySQLDB{}
		repo := &SQLRepository{DB: db, Categories: enabled}
		scope := Scope{testEntity, testBook, testStream}
		a := attempt{testSet, "1"}
		_, _ = repo.Context(context.Background(), scope, a)
		want := "SELECT integration.sync_context($1,$2,$3,$4,$5)"
		if enabled {
			want = "SELECT integration.sync_category_context($1,$2,$3,$4,$5)"
		}
		if db.query != want {
			t.Fatal("wrong context function")
		}
		_ = repo.Stage(context.Background(), scope, a, 0, json.RawMessage(`{}`))
		want = "SELECT integration.stage_sync_page($1,$2,$3,$4,$5,$6)"
		if enabled {
			want = "SELECT integration.stage_sync_category_page($1,$2,$3,$4,$5,$6)"
		}
		if db.query != want {
			t.Fatal("wrong stage function")
		}
		_, _ = repo.Complete(context.Background(), scope, a)
		want = "SELECT integration.complete_sync($1,$2,$3,$4,$5)"
		if enabled {
			want = "SELECT integration.complete_sync_categories($1,$2,$3,$4,$5)"
		}
		if db.query != want {
			t.Fatal("wrong completion function")
		}
	}
}

// TestCategoryConstructorsRemainOptIn checks no constructor performs database or provider work.
func TestCategoryConstructorsRemainOptIn(t *testing.T) {
	db := &faultDB{}
	provider := providerFunc(func(context.Context, FetchRequest) ([]byte, error) { t.Fatal("constructor fetched"); return nil, nil })
	credentials := credentialFunc(func(context.Context, CredentialReference) (ProviderCredentials, error) {
		t.Fatal("constructor resolved")
		return ProviderCredentials{}, nil
	})
	evidence := localEvidence(t)
	legacy, e := New(db, provider, credentials, evidence)
	if e != nil || legacy.categories || legacy.repo.(*SQLRepository).Categories {
		t.Fatal("legacy enabled")
	}
	enhanced, e := NewWithCategories(db, provider, credentials, evidence)
	if e != nil || !enhanced.categories || !enhanced.repo.(*SQLRepository).Categories || db.begins != 0 {
		t.Fatal("opt-in incomplete or touched database")
	}
	if _, e := NewWithCategories(nil, provider, credentials, evidence); e == nil {
		t.Fatal("missing dependency accepted")
	}
}

// TestCategoryCumulativeWireBudget counts sidecar bytes against the unchanged whole-update cap.
func TestCategoryCumulativeWireBudget(t *testing.T) {
	raw := categoryPageFixture(`{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"HIGH","version":"v2"}`)
	for _, enabled := range []bool{false, true} {
		repo := &fakeRepository{context: contextFixture(), state: "staging"}
		r := runnerFixture(t, repo, providerFunc(func(context.Context, FetchRequest) ([]byte, error) { return raw, nil }))
		financial, _, e := r.decode(raw, repo.context.expected())
		if e != nil {
			t.Fatal(e)
		}
		wire, e := json.Marshal(pageWire{PageIndex: 0, Page: financial})
		if e != nil {
			t.Fatal(e)
		}
		repo.context.ByteCount = (2 << 20) - len(wire)
		r.categories = enabled
		_, _, e = r.acquire(context.Background(), Scope{testEntity, testBook, testStream}, attempt{repo.context.SetID, repo.context.Fence})
		if enabled {
			if e == nil || len(repo.pages) != 0 || repo.state != "abandoned" {
				t.Fatal("sidecar exceeded cumulative budget")
			}
		} else if e != nil || len(repo.pages) != 1 {
			t.Fatal("legacy exact budget changed")
		}
	}
}
