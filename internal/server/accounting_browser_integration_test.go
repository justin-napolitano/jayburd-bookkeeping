//go:build integration

package server

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/autopost"
)

var appProofAccounting = flag.Bool("app-proof-accounting", false, "prove source-backed automatic accounting in the disposable browser fixture")

const accountingBrowserEntity = "e7100000-0000-4000-8000-000000000002"
const accountingBrowserBook = "e7100000-0000-4000-8000-000000000003"
const accountingBrowserSource = "e7100000-0000-4000-8000-000000000004"
const accountingBrowserPolicy = "e7100000-0000-4000-8000-000000000005"
const accountingBrowserExpense = "e7100000-0000-4000-8000-000000000007"
const accountingBrowserNamespace = "e7100000-0000-4000-8000-000000000008"
const accountingBrowserForeign = "e7100000-0000-4000-8000-000000000009"
const accountingBrowserActor = "00000000-0000-4000-8000-000000000001"

// accountingBrowserAuth is compiled only into the explicit synthetic integration binary.
type accountingBrowserAuth struct{}

func (accountingBrowserAuth) Authenticate(_ context.Context, token string) (string, error) {
	switch token {
	case "synthetic.accounting.proof":
		return accountingBrowserActor, nil
	case "synthetic.accounting.foreign":
		return accountingBrowserForeign, nil
	}
	return "", errors.New("synthetic identity rejected")
}

// bootstrapAccountingBrowser supplies retained fictional statement evidence, not posted journals or automatic policy.
// Browser configuration, restricted SQL and the production runner must create the sole financial effect.
func bootstrapAccountingBrowser(t *testing.T, ctx context.Context, admin *pgx.Conn, pool *pgxpool.Pool) {
	t.Helper()
	tx, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal("accounting fixture begin", err)
	}
	defer tx.Rollback(context.Background())
	// All identifiers and values below are fixed synthetic fixtures; never a runtime or external input.
	_, err = tx.Exec(ctx, fmt.Sprintf(`
 INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES('%s','https://synthetic.example.invalid','accounting-foreign');
 INSERT INTO ledger.entity(entity_id,name,kind)VALUES('%s','Fictional Automatic Accounting Studio','legal');
 INSERT INTO ledger.book(entity_id,book_id,name)VALUES('%s','%s','Automatic accounting / USD');
 INSERT INTO ledger.membership SELECT '%s','%s',permission,NULL FROM unnest(ARRAY['read','prepare','post'])permission;
 INSERT INTO ledger.membership VALUES('%s','%s','read',NULL);
 INSERT INTO ledger.book_policy VALUES('%s','%s','%s','2026-01-01',NULL,'00000000-0000-4000-8000-000000000002','accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1','%s',clock_timestamp());
 INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before)VALUES('%s','%s','e7100000-0000-4000-8000-000000000010','2026-01-01','2027-01-01');
 INSERT INTO ledger.account VALUES('%s','%s','e7100000-0000-4000-8000-000000000006','1000','Fictional operating bank','asset','debit',true,'2026-01-01',NULL),('%s','%s','%s','6000','Fictional office supplies','expense','debit',false,'2026-01-01',NULL);
 INSERT INTO banking.source_account(entity_id,book_id,source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by)VALUES('%s','%s','%s','e7100000-0000-4000-8000-000000000006','00000000-0000-4000-8000-000000000002','2026-01-01','e7100000-0000-4000-8000-000000000011','%s');
 INSERT INTO banking.source_namespace(entity_id,book_id,namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by)VALUES('%s','%s','%s','%s','jayburd-synthetic-statement-csv/1','e7100000-0000-4000-8000-000000000012','FICTIONAL-ACCOUNTING-STATEMENT',false,'%s');
 INSERT INTO banking.source_permission SELECT '%s','%s','%s','%s',capability,NULL FROM unnest(ARRAY['read','review'])capability;
 `, accountingBrowserForeign, accountingBrowserEntity, accountingBrowserEntity, accountingBrowserBook,
		accountingBrowserEntity, accountingBrowserActor, accountingBrowserEntity, accountingBrowserForeign,
		accountingBrowserEntity, accountingBrowserBook, accountingBrowserPolicy, accountingBrowserActor,
		accountingBrowserEntity, accountingBrowserBook, accountingBrowserEntity, accountingBrowserBook, accountingBrowserEntity, accountingBrowserBook, accountingBrowserExpense,
		accountingBrowserEntity, accountingBrowserBook, accountingBrowserSource, accountingBrowserActor,
		accountingBrowserEntity, accountingBrowserBook, accountingBrowserNamespace, accountingBrowserSource, accountingBrowserActor,
		accountingBrowserEntity, accountingBrowserBook, accountingBrowserSource, accountingBrowserActor))
	if err != nil {
		t.Fatal("accounting fixture metadata rejected", err)
	}
	for i, row := range []struct{ description, amount string }{{"Fictional automatic paper purchase", "12.50"}, {"Fictional unmapped source purchase", "4.25"}, {"Fictional protected manual purchase", "3.75"}} {
		observation := fmt.Sprintf("e7100000-0000-4000-8000-%012d", 100+i)
		review := fmt.Sprintf("e7100000-0000-4000-8000-%012d", 200+i)
		decision := fmt.Sprintf("e7100000-0000-4000-8000-%012d", 300+i)
		for _, step := range []struct {
			query string
			args  []any
		}{
			{`INSERT INTO banking.source_observation(entity_id,book_id,observation_id,source_account_id,namespace_id,external_entry_id)VALUES($1,$2,$3,$4,$5,$6)`, []any{accountingBrowserEntity, accountingBrowserBook, observation, accountingBrowserSource, accountingBrowserNamespace, fmt.Sprintf("FICTIONAL-STATEMENT-%d", i+1)}},
			{`INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields)VALUES($1,$2,$3,1,'booked','2026-01-14','00000000-0000-4000-8000-000000000002','outflow',$4::text,$4::text::numeric,$5,'{}')`, []any{accountingBrowserEntity, accountingBrowserBook, observation, row.amount, row.description}},
			{`INSERT INTO banking.review_item(entity_id,book_id,review_id,source_account_id,state)VALUES($1,$2,$3,$4,'needs_review')`, []any{accountingBrowserEntity, accountingBrowserBook, review, accountingBrowserSource}},
			{`INSERT INTO banking.review_decision(entity_id,book_id,decision_id,review_id,expected_review_revision,kind,actor_id,reason,body)VALUES($1,$2,$3,$4,0,'initialize',$5,'synthetic_import','{}')`, []any{accountingBrowserEntity, accountingBrowserBook, decision, review, accountingBrowserActor}},
			{`INSERT INTO banking.review_evidence VALUES($1,$2,$3,$4,$5,1)`, []any{accountingBrowserEntity, accountingBrowserBook, decision, observation, review}},
			{`INSERT INTO banking.review_assignment VALUES($1,$2,$3,$4,$5)`, []any{accountingBrowserEntity, accountingBrowserBook, observation, review, decision}},
		} {
			if _, err = tx.Exec(ctx, step.query, step.args...); err != nil {
				t.Fatal("accounting retained evidence rejected", err)
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal("accounting fixture commit", err)
	}
	// Protect a real manual artifact-backed draft; no manual posting command is manufactured.
	call := func(query string, args ...any) (json.RawMessage, error) {
		runtimeTx, e := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			return nil, e
		}
		defer runtimeTx.Rollback(context.Background())
		var value json.RawMessage
		if e = runtimeTx.QueryRow(ctx, query, args...).Scan(&value); e != nil {
			return nil, e
		}
		if e = runtimeTx.Commit(ctx); e != nil {
			return nil, e
		}
		return value, nil
	}
	var detail map[string]any
	var raw json.RawMessage
	if raw, err = call(`SELECT banking.get_review_item($1,$2,$3,'e7100000-0000-4000-8000-000000000202')`, accountingBrowserActor, accountingBrowserEntity, accountingBrowserBook); err != nil {
		t.Fatal("accounting manual detail", err)
	}
	if json.Unmarshal(raw, &detail) != nil {
		t.Fatal("accounting manual detail shape")
	}
	body, _ := json.Marshal(map[string]any{"expected_review_revision": detail["review_revision"], "source_revision": detail["source_revision"], "binding_revision": detail["binding_revision"], "expected_book_revision": detail["book_revision"], "category_account_id": accountingBrowserExpense, "policy_id": accountingBrowserPolicy, "accounting_date": "2026-01-14", "description": "Protected fictional manual draft"})
	if raw, err = call(`SELECT banking.create_review_draft($1,$2,$3,'e7100000-0000-4000-8000-000000000202','accounting-fixture-manual',$4)`, accountingBrowserActor, accountingBrowserEntity, accountingBrowserBook, string(body)); err != nil {
		t.Fatal("accounting manual protection", err)
	}
	if automaticAccountingReady(ctx, pool) != nil {
		t.Fatal("accounting runtime ACL rejected")
	}
	policy, err := (&autopost.Service{DB: pool}).GetPolicy(ctx, autopost.Request{Actor: accountingBrowserActor, Entity: accountingBrowserEntity, Book: accountingBrowserBook})
	if err != nil {
		t.Fatal("accounting initial policy", err)
	}
	var settings struct {
		Revision string `json:"revision"`
		Enabled  bool   `json:"enabled"`
	}
	if json.Unmarshal(policy, &settings) != nil || settings.Revision != "0" || settings.Enabled {
		t.Fatal("accounting policy must be absent/off initially")
	}
}

// composeAccountingBrowser uses the same default-off gates and actual runner as cmd/api composition.
func composeAccountingBrowser(t *testing.T, ctx context.Context, admin *pgx.Conn, pool *pgxpool.Pool, config *Config, dependencies *Dependencies) {
	bootstrapAccountingBrowser(t, ctx, admin, pool)
	config.AutomaticAccountingEnabled = true
	config.AutomaticSourcePosting = true
	service := &autopost.Service{DB: pool}
	dependencies.Auth = accountingBrowserAuth{}
	dependencies.Accounting = service
	dependencies.SourcePosting = service
	dependencies.Ready = func(ctx context.Context) error {
		if err := runtimeReady(ctx, pool); err != nil {
			return err
		}
		return automaticAccountingReady(ctx, pool)
	}
}
