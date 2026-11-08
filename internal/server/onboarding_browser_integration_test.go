//go:build integration

package server

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const onboardingBrowserActor = "00000000-0000-4000-8000-000000000094"
const onboardingBrowserEntity = "00000000-0000-4000-8000-000000000940"
const onboardingBrowserBook = "00000000-0000-4000-8000-000000000941"
const onboardingBrowserSource = "00000000-0000-4000-8000-000000000942"
const onboardingBrowserItem = "00000000-0000-4000-8000-000000000943"

// onboardingBrowserAuth exists only in the synthetic integration binary; it grants no production identity.
type onboardingBrowserAuth struct{}

func (onboardingBrowserAuth) Authenticate(ctx context.Context, token string) (string, error) {
	if token == "synthetic.onboarding.proof" {
		return onboardingBrowserActor, nil
	}
	if token == "synthetic.onboarding.other" {
		return "00000000-0000-4000-8000-000000000095", nil
	}
	if token == "" {
		return "", errors.New("synthetic identity rejected")
	}
	return (labelBrowserAuth{}).Authenticate(ctx, token)
}

// bootstrapOnboardingBrowser creates legacy evidence only. The real browser must create its absent workspace and adopt it.
func bootstrapOnboardingBrowser(t *testing.T, ctx context.Context, admin *pgx.Conn, _ *pgxpool.Pool) {
	t.Helper()
	tx, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal("synthetic onboarding fixture unavailable", err)
	}
	defer tx.Rollback(context.Background())
	statements := []string{
		`INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES('00000000-0000-4000-8000-000000000094','https://synthetic.example.invalid','synthetic-onboarding-owner'),('00000000-0000-4000-8000-000000000095','https://synthetic.example.invalid','synthetic-onboarding-other')`,
		`INSERT INTO ledger.entity(entity_id,name,kind)VALUES('00000000-0000-4000-8000-000000000940','Fictional onboarding studio','legal')`,
		`INSERT INTO ledger.book(entity_id,book_id,name)VALUES('00000000-0000-4000-8000-000000000940','00000000-0000-4000-8000-000000000941','Fictional existing records')`,
		`INSERT INTO ledger.membership SELECT '00000000-0000-4000-8000-000000000940', '00000000-0000-4000-8000-000000000094',permission,NULL FROM unnest(ARRAY['read','import','connect','post'])permission`,
		`INSERT INTO ledger.book_policy VALUES('00000000-0000-4000-8000-000000000940','00000000-0000-4000-8000-000000000941','00000000-0000-4000-8000-000000000944','2026-01-01',NULL,'00000000-0000-4000-8000-000000000002','accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1','00000000-0000-4000-8000-000000000094',now())`,
		`INSERT INTO ledger.account VALUES('00000000-0000-4000-8000-000000000940','00000000-0000-4000-8000-000000000941','00000000-0000-4000-8000-000000000945','1000','Fictional existing checking','asset','debit',true,'2026-01-01',NULL)`,
		`INSERT INTO banking.source_account(entity_id,book_id,source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by)VALUES('00000000-0000-4000-8000-000000000940','00000000-0000-4000-8000-000000000941','00000000-0000-4000-8000-000000000942','00000000-0000-4000-8000-000000000945','00000000-0000-4000-8000-000000000002','2026-01-05','00000000-0000-4000-8000-000000000946','00000000-0000-4000-8000-000000000094')`,
		`INSERT INTO banking.source_namespace(entity_id,book_id,namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by)VALUES('00000000-0000-4000-8000-000000000940','00000000-0000-4000-8000-000000000941','00000000-0000-4000-8000-000000000947','00000000-0000-4000-8000-000000000942','jayburd-synthetic-statement-csv/1','00000000-0000-4000-8000-000000000947','FICTIONAL-ONBOARDING',false,'00000000-0000-4000-8000-000000000094')`,
		`INSERT INTO banking.source_permission SELECT '00000000-0000-4000-8000-000000000940','00000000-0000-4000-8000-000000000941','00000000-0000-4000-8000-000000000942','00000000-0000-4000-8000-000000000094',cap,NULL FROM unnest(ARRAY['read','import'])cap`,
		`INSERT INTO integration.sync_item(entity_id,book_id,sync_item_id,provider,environment,item_id,credential_ref,credential_version,consent_principal,consent_evidence_ref,consent_expires_at,synthetic)VALUES('00000000-0000-4000-8000-000000000940','00000000-0000-4000-8000-000000000941','00000000-0000-4000-8000-000000000943','plaid','sandbox','SYNTHETIC-ONBOARDING-ITEM','00000000-0000-4000-8000-000000000948',1,'00000000-0000-4000-8000-000000000094','00000000-0000-4000-8000-000000000946','2099-01-01',true)`,
		`INSERT INTO integration.sync_stream(entity_id,book_id,sync_item_id,source_account_id,namespace_id,account_id,worker_login,service_principal,cursor_value,completed_generation,readiness)VALUES('00000000-0000-4000-8000-000000000940','00000000-0000-4000-8000-000000000941','00000000-0000-4000-8000-000000000943','00000000-0000-4000-8000-000000000942','00000000-0000-4000-8000-000000000947','SYNTHETIC-ONBOARDING-ACCOUNT','synthetic_sync','00000000-0000-4000-8000-000000000095','synthetic-original-cursor',1,'HISTORICAL_UPDATE_COMPLETE')`,
	}
	for _, statement := range statements {
		if _, err = tx.Exec(ctx, statement); err != nil {
			t.Fatal("synthetic onboarding metadata rejected", err)
		}
	}
	for i, amount := range []string{"0.10", "9007199254740993.01"} {
		observation := []string{"00000000-0000-4000-8000-000000000949", "00000000-0000-4000-8000-000000000950"}[i]
		description := []string{"Fictional adopted cafe purchase", "Fictional adopted exact large quantity"}[i]
		if _, err = tx.Exec(ctx, `INSERT INTO banking.source_observation(entity_id,book_id,observation_id,source_account_id,namespace_id,external_entry_id)VALUES($1,$2,$3,$4,'00000000-0000-4000-8000-000000000947',$3::text)`, onboardingBrowserEntity, onboardingBrowserBook, observation, onboardingBrowserSource); err != nil {
			t.Fatal("synthetic onboarding observation rejected", err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields)VALUES($1,$2,$3,1,'booked','2026-10-01','00000000-0000-4000-8000-000000000002','outflow',$4,$4::numeric,$5,'{}')`, onboardingBrowserEntity, onboardingBrowserBook, observation, amount, description); err != nil {
			t.Fatal("synthetic onboarding original revision rejected", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal("synthetic onboarding fixture commit unavailable", err)
	}
	var workspaces, journals int
	if err = admin.QueryRow(ctx, `SELECT (SELECT count(*) FROM banking.collection_workspace WHERE owner_principal_id=$1),(SELECT count(*) FROM ledger.journal_entry WHERE entity_id=$2 AND book_id=$3)`, onboardingBrowserActor, onboardingBrowserEntity, onboardingBrowserBook).Scan(&workspaces, &journals); err != nil || workspaces != 0 || journals != 0 {
		t.Fatal("synthetic onboarding fixture must start without workspace or journals", err)
	}
}
