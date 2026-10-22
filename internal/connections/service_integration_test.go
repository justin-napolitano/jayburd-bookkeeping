//go:build integration

package connections

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proof = flag.Bool("connection-service-proof", false, "Use only the disposable internal registry database")

const entity = "00000000-0000-4000-8000-000000000002"
const book = "00000000-0000-4000-8000-000000000003"
const actor = "00000000-0000-4000-8000-000000000001"
const reader = "00000000-0000-4000-8000-000000000020"
const other = "00000000-0000-4000-8000-000000000021"
const account = "00000000-0000-4000-8000-000000000011"
const currency = "00000000-0000-4000-8000-000000000005"

// proofConnect uses only the runner's private synthetic trust network, never an inherited DSN.
func proofConnect(t testing.TB, db, user string) *pgx.Conn {
	t.Helper()
	if !*proof {
		t.Skip("use scripts/test-connection-state.sh")
	}
	c, e := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=5432 user=%s dbname=%s sslmode=disable connect_timeout=5", user, db))
	if e != nil {
		t.Fatal(e)
	}
	return c
}

// proofExec is owner-only sqlFixture sqlSetup or inspection, never product privilege.
func proofExec(t testing.TB, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, e := c.Exec(context.Background(), q, args...); e != nil {
		t.Fatal(e)
	}
}

// proofValue reads fixed aggregate proof facts with the sqlFixture owner.
func proofValue(t testing.TB, c *pgx.Conn, q string, args ...any) string {
	t.Helper()
	var s string
	if e := c.QueryRow(context.Background(), q, args...).Scan(&s); e != nil {
		t.Fatal(e)
	}
	return s
}

// proofCode compares SQLSTATE without relying on potentially sensitive diagnostics.
func proofCode(t testing.TB, e error, want string) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(e, &p) || p.Code != want {
		t.Fatalf("want %s got %v", want, e)
	}
}

// sqlFixture installs only fictional ownership and independently chosen currency metadata.
func sqlFixture(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	db := "connection_service_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := proofConnect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	proofExec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE connection_service_template")
	c := proofConnect(t, db, "postgres")
	t.Cleanup(func() { c.Close(context.Background()) })
	proofExec(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','approver'),($2,'synthetic','reader'),($3,'synthetic','other')`, actor, reader, other)
	proofExec(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional LLC','legal')`, entity)
	proofExec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$3,'read',NULL),($1,$4,'read',NULL)`, entity, actor, reader, other)
	proofExec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Synthetic book')`, entity, book)
	proofExec(t, c, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000006','JPY','392',0,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000007','KWD','414',3,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000008','XXX','999',NULL,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000009','EUR','978',2,'2020-01-01','2025-01-01','synthetic')`, currency)
	proofExec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'1000','Fictional cash','asset','debit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000012','2000','Fictional liability','liability','credit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000013','1001','Fictional nonmonetary','asset','debit',false,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000014','1002','Fictional contra','asset','credit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000015','1003','Fictional second deposit','asset','debit',true,'2026-01-01',NULL)`, entity, book, account)
	return c, db
}

// proofRaw encodes synthetic exact values without floating-point amounts.
func proofRaw(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return string(b)
}

// proofInvoke uses a real login, bounded transactions and full serialization retry.
func proofInvoke(t testing.TB, db, user, q string, args ...any) (string, error) {
	t.Helper()
	c := proofConnect(t, db, user)
	defer c.Close(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	var result string
	var err error
	for i := 0; i < 3; i++ {
		tx, e := c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			return "", e
		}
		_, e = tx.Exec(ctx, "SET LOCAL statement_timeout='5s'; SET LOCAL lock_timeout='1s'")
		if e == nil && user == "synthetic_operator" {
			_, e = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner")
		}
		if e == nil {
			e = tx.QueryRow(ctx, q, args...).Scan(&result)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		_ = tx.Rollback(context.Background())
		if e == nil {
			return result, nil
		}
		err = e
		var p *pgconn.PgError
		if !errors.As(e, &p) || (p.Code != "40001" && p.Code != "40P01") {
			break
		}
	}
	return "", err
}

// proofCall requires a committed result from an actual restricted login.
func proofCall(t testing.TB, db, user, q string, args ...any) string {
	t.Helper()
	v, e := proofInvoke(t, db, user, q, args...)
	if e != nil {
		t.Fatal(e)
	}
	return v
}

// proofObject decodes only synthetic fixed SQL projection fields.
func proofObject(t testing.TB, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal([]byte(s), &v); e != nil {
		t.Fatal(e)
	}
	return v
}

// sqlSetup explicitly provisions connection management separately from acquisition.
func sqlSetup(t *testing.T) (*pgx.Conn, string) {
	c, db := sqlFixture(t)
	proofExec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'import',NULL),($1,$3,'import',NULL)`, entity, actor, reader)
	proofCall(t, db, "synthetic_operator", "SELECT integration.set_connect_permission($1,$2,$3,$1,'proofConnect',true)::text", actor, entity, book)
	proofCall(t, db, "synthetic_operator", "SELECT integration.set_connection_worker($1,$2,$3,'worker',$4)::text", actor, entity, book, proofRaw(map[string]any{"service_principal_id": reader, "worker_login": "synthetic_sync", "enabled": true}))
	return c, db
}

// sqlService composes the actual restricted login and original in-memory provider/storage adapters.
func sqlService(t *testing.T) (*Service, *pgx.Conn, *fakeProvider, *memorySecrets) {
	admin, db := sqlSetup(t)
	runtime := proofConnect(t, db, "synthetic_runtime")
	t.Cleanup(func() { runtime.Close(context.Background()) })
	_, _, provider, store := fixture()
	return New(runtime, provider, store), admin, provider, store
}

// sqlBegin returns only the public attempt identity and ephemeral handoff from the real service.
func sqlBegin(t *testing.T, s *Service, key string) (string, ExecuteRequest) {
	t.Helper()
	out, e := s.Begin(context.Background(), scope, key, BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		ID string `json:"attempt_id"`
	}
	if json.Unmarshal(out.Attempt, &v) != nil || !validID(v.ID) {
		t.Fatal("missing attempt")
	}
	return v.ID, ExecuteRequest{out.Nonce, out.LinkToken, "synthetic-public"}
}

// TestConnectionServiceSQLLifecycle proves real publication and explicit first-provider activation, without journals.
func TestConnectionServiceSQLLifecycle(t *testing.T) {
	s, admin, p, store := sqlService(t)
	id, input := sqlBegin(t, s, "first")
	if _, e := s.Execute(context.Background(), scope, id, input); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Execute(context.Background(), scope, id, input); e != nil {
		t.Fatal(e)
	}
	if count(p.calls, "exchange") != 1 || store.writes != 1 {
		t.Fatal("duplicate remote effect")
	}
	bind := BindingRequest{ConsentGrants: true, Bindings: []Binding{{AccountID: "account-one", LedgerAccountID: account, CurrencyVersionID: currency, BindingDate: "2026-01-05", BindingEvidenceRef: "00000000-0000-4000-8000-000000000080"}}}
	if _, e := s.Bind(context.Background(), scope, id, "bind", bind); e != nil {
		t.Fatal(e)
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_stream") != "1" || proofValue(t, admin, "SELECT overlap_review_required::text FROM banking.source_namespace") != "false" {
		t.Fatal("first provider did not activate")
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || proofValue(t, admin, "SELECT count(*)::text FROM integration.outbox_message") != "0" {
		t.Fatal("financial mutation")
	}
	before := len(p.calls)
	stream := proofValue(t, admin, "SELECT stream_id::text FROM integration.sync_stream")
	revision := proofValue(t, admin, "SELECT binding_revision::text FROM banking.source_account")
	if _, e := s.Disconnect(context.Background(), scope, stream, "disconnect", revision); e != nil {
		t.Fatal(e)
	}
	if len(p.calls) != before {
		t.Fatal("single account removed Item")
	}
}

// TestConnectionAutoAccountBinding proves three reviewed depository accounts create distinct chart and source accounts atomically.
func TestConnectionAutoAccountBinding(t *testing.T) {
	s, admin, provider, _ := sqlService(t)
	checking, savings, moneyMarket := "checking", "savings", "money market"
	provider.manifest.Accounts = []Account{
		{AccountID: "checking-one", Type: "depository", Subtype: &checking, Currency: ptr("USD")},
		{AccountID: "savings-one", Type: "depository", Subtype: &savings, Currency: ptr("USD")},
		{AccountID: "market-one", Type: "depository", Subtype: &moneyMarket, Currency: ptr("USD")},
	}
	id, input := sqlBegin(t, s, "auto-three")
	if _, err := s.Execute(context.Background(), scope, id, input); err != nil {
		t.Fatal(err)
	}
	ids := []string{"00000000-0000-4000-8000-000000000031", "00000000-0000-4000-8000-000000000032", "00000000-0000-4000-8000-000000000033"}
	accounts := []Account{provider.manifest.Accounts[0], provider.manifest.Accounts[1], provider.manifest.Accounts[2]}
	bindings := make([]Binding, 0, len(accounts))
	for index, providerAccount := range accounts {
		ledgerAccount := &NewLedgerAccount{AccountID: ids[index], Code: fmt.Sprintf("11%02d", index), Name: fmt.Sprintf("Sandbox depository %d", index+1), Category: "asset", NormalSide: "debit", Monetary: true, ActiveFrom: "2026-01-05"}
		bindings = append(bindings, Binding{AccountID: providerAccount.AccountID, LedgerAccountID: ids[index], CurrencyVersionID: currency, BindingDate: "2026-01-05", BindingEvidenceRef: fmt.Sprintf("00000000-0000-4000-8000-00000000009%d", index), NewLedgerAccount: ledgerAccount})
	}
	request := BindingRequest{ConsentGrants: true, Bindings: bindings}
	first, err := s.Bind(context.Background(), scope, id, "bind-auto-three", request)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := s.Bind(context.Background(), scope, id, "bind-auto-three", request)
	if err != nil || string(replayed) != string(first) {
		t.Fatal("automatic binding replay changed result", err)
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM ledger.account WHERE code LIKE '11%'") != "3" ||
		proofValue(t, admin, "SELECT count(*)::text FROM ledger.account WHERE account_id=ANY($1::uuid[]) AND category='asset' AND normal_side='debit' AND monetary", ids) != "3" ||
		proofValue(t, admin, "SELECT count(*)::text FROM banking.source_account") != "3" ||
		proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_stream WHERE state='active'") != "3" ||
		proofValue(t, admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("automatic binding did not preserve its account and journal invariants")
	}
}

// TestConnectionAutoAccountAtomicity proves an unsupported subtype rolls back all requested chart changes.
func TestConnectionAutoAccountAtomicity(t *testing.T) {
	s, admin, provider, _ := sqlService(t)
	checking, prepaid := "checking", "prepaid"
	provider.manifest.Accounts = []Account{
		{AccountID: "checking-one", Type: "depository", Subtype: &checking, Currency: ptr("USD")},
		{AccountID: "prepaid-one", Type: "depository", Subtype: &prepaid, Currency: ptr("USD")},
	}
	id, input := sqlBegin(t, s, "auto-reject")
	if _, err := s.Execute(context.Background(), scope, id, input); err != nil {
		t.Fatal(err)
	}
	bindings := []Binding{
		{AccountID: "checking-one", LedgerAccountID: "00000000-0000-4000-8000-000000000041", CurrencyVersionID: currency, BindingDate: "2026-01-05", BindingEvidenceRef: "00000000-0000-4000-8000-000000000091", NewLedgerAccount: &NewLedgerAccount{AccountID: "00000000-0000-4000-8000-000000000041", Code: "1200", Name: "Sandbox checking", Category: "asset", NormalSide: "debit", Monetary: true, ActiveFrom: "2026-01-05"}},
		{AccountID: "prepaid-one", LedgerAccountID: "00000000-0000-4000-8000-000000000042", CurrencyVersionID: currency, BindingDate: "2026-01-05", BindingEvidenceRef: "00000000-0000-4000-8000-000000000092", NewLedgerAccount: &NewLedgerAccount{AccountID: "00000000-0000-4000-8000-000000000042", Code: "1201", Name: "Sandbox prepaid", Category: "asset", NormalSide: "debit", Monetary: true, ActiveFrom: "2026-01-05"}},
	}
	if _, err := s.Bind(context.Background(), scope, id, "bind-auto-reject", BindingRequest{ConsentGrants: true, Bindings: bindings}); err == nil {
		t.Fatal("unsupported subtype accepted")
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM ledger.account WHERE code IN ('1200','1201')") != "0" || proofValue(t, admin, "SELECT count(*)::text FROM banking.source_account") != "0" {
		t.Fatal("failed automatic binding left partial state")
	}
}

// TestConnectionLiabilityBinding proves mixed reviewed products bind separately without posting.
func TestConnectionLiabilityBinding(t *testing.T) {
	s, admin, provider, _ := sqlService(t)
	checking, card, student, mortgage, auto, consumer, line, other := "checking", "credit card", "student", "mortgage", "auto", "consumer", "line of credit", "other"
	provider.manifest.Accounts = []Account{
		{AccountID: "checking-one", Type: "depository", Subtype: &checking, Currency: ptr("USD")},
		{AccountID: "card-one", Type: "credit", Subtype: &card, Currency: ptr("USD")},
		{AccountID: "student-one", Type: "loan", Subtype: &student, Currency: ptr("USD")},
		{AccountID: "mortgage-one", Type: "loan", Subtype: &mortgage, Currency: ptr("USD")},
		{AccountID: "auto-one", Type: "loan", Subtype: &auto, Currency: ptr("USD")},
		{AccountID: "consumer-one", Type: "loan", Subtype: &consumer, Currency: ptr("USD")},
		{AccountID: "line-one", Type: "loan", Subtype: &line, Currency: ptr("USD")},
		{AccountID: "other-one", Type: "loan", Subtype: &other, Currency: ptr("USD")},
	}
	id, input := sqlBegin(t, s, "liability-mixed")
	if _, err := s.Execute(context.Background(), scope, id, input); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for n := 51; n <= 57; n++ {
		ids = append(ids, fmt.Sprintf("00000000-0000-4000-8000-%012d", n))
	}
	kinds := []struct{ account, code, category, side string }{
		{"checking-one", "1300", "asset", "debit"},
		{"card-one", "2300", "liability", "credit"},
		{"student-one", "2301", "liability", "credit"},
		{"mortgage-one", "2302", "liability", "credit"},
		{"auto-one", "2303", "liability", "credit"},
		{"consumer-one", "2304", "liability", "credit"},
		{"line-one", "2305", "liability", "credit"},
	}
	bindings := make([]Binding, 0, 3)
	for n, kind := range kinds {
		bindings = append(bindings, Binding{AccountID: kind.account, LedgerAccountID: ids[n], CurrencyVersionID: currency, BindingDate: "2026-01-05", BindingEvidenceRef: fmt.Sprintf("00000000-0000-4000-8000-00000000006%d", n), NewLedgerAccount: &NewLedgerAccount{AccountID: ids[n], Code: kind.code, Name: "Synthetic " + kind.account, Category: kind.category, NormalSide: kind.side, Monetary: true, ActiveFrom: "2026-01-05"}})
	}
	request := BindingRequest{ConsentGrants: true, Bindings: bindings}
	// Unsupported loan substitution must roll back the entire mixed request.
	invalid := append([]Binding(nil), bindings...)
	invalid[len(invalid)-1].AccountID = "other-one"
	if _, err := s.Bind(context.Background(), scope, id, "liability-unsupported", BindingRequest{ConsentGrants: true, Bindings: invalid}); err == nil {
		t.Fatal("unreviewed loan accepted")
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM banking.source_account") != "0" || proofValue(t, admin, "SELECT count(*)::text FROM ledger.account WHERE account_id=ANY($1::uuid[])", ids) != "0" {
		t.Fatal("unsupported mixed request left partial accounts")
	}
	first, err := s.Bind(context.Background(), scope, id, "liability-mixed-bind", request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.Bind(context.Background(), scope, id, "liability-mixed-bind", request)
	if err != nil || string(first) != string(replay) {
		t.Fatal("liability binding replay changed result", err)
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM ledger.account WHERE account_id=ANY($1::uuid[]) AND category='liability' AND normal_side='credit' AND monetary", ids) != "6" ||
		proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_stream WHERE state='active'") != "7" ||
		proofValue(t, admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("liability binding invariants failed")
	}
	if _, err := s.Bind(context.Background(), scope, id, "liability-mixed-bind", BindingRequest{ConsentGrants: true, Bindings: bindings[:1]}); err == nil {
		t.Fatal("changed replay accepted")
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_stream WHERE account_id='other-one'") != "0" {
		t.Fatal("unsupported loan bound")
	}
}

// TestBindingReviewCancelSQL proves provider removal precedes terminal local abandonment.
func TestBindingReviewCancelSQL(t *testing.T) {
	s, admin, provider, _ := sqlService(t)
	id, input := sqlBegin(t, s, "cancel-binding-review")
	if _, err := s.Execute(context.Background(), scope, id, input); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Abandon(context.Background(), Scope{Actor: reader, Entity: entity, Book: book}, id); err == nil {
		t.Fatal("foreign actor canceled binding review")
	}
	result, err := s.Abandon(context.Background(), scope, id)
	if err != nil || proofObject(t, string(result))["state"] != "abandoned" {
		t.Fatal("binding review cancellation failed", err)
	}
	if count(provider.calls, "remove") != 1 ||
		proofValue(t, admin, "SELECT state FROM integration.sync_item") != "disconnected" ||
		proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_stream") != "0" ||
		proofValue(t, admin, "SELECT count(*)::text FROM banking.source_account") != "0" ||
		proofValue(t, admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("binding cancellation left active or financial state")
	}
	if _, err = s.Abandon(context.Background(), scope, id); err != nil || count(provider.calls, "remove") != 1 {
		t.Fatal("terminal cancellation replayed provider removal", err)
	}
	if proofValue(t, admin, "SELECT count(*)::text FROM integration.connection_attempt a WHERE integration.connection_unfinished(a)") != "0" {
		t.Fatal("confirmed cancellation still blocks enrollment")
	}
}

// TestBindingReviewCancelFailure keeps the fenced Item recoverable until removal succeeds.
func TestBindingReviewCancelFailure(t *testing.T) {
	s, admin, provider, _ := sqlService(t)
	id, input := sqlBegin(t, s, "cancel-binding-failure")
	if _, err := s.Execute(context.Background(), scope, id, input); err != nil {
		t.Fatal(err)
	}
	provider.fail = "remove"
	if _, err := s.Abandon(context.Background(), scope, id); err == nil {
		t.Fatal("provider failure reported cancellation")
	}
	if proofValue(t, admin, "SELECT state FROM integration.connection_attempt") != "disconnect_pending" ||
		proofValue(t, admin, "SELECT state FROM integration.sync_item") != "disconnect_pending" {
		t.Fatal("provider failure released fenced Item")
	}
	provider.fail = ""
	if _, err := s.Abandon(context.Background(), scope, id); err != nil || proofValue(t, admin, "SELECT state FROM integration.connection_attempt") != "abandoned" {
		t.Fatal("fenced cancellation did not resume", err)
	}
}

// TestBindingReviewBindCancelRace proves the database lock admits one terminal direction.
func TestBindingReviewBindCancelRace(t *testing.T) {
	s, admin, _, _ := sqlService(t)
	id, input := sqlBegin(t, s, "binding-cancel-race")
	if _, err := s.Execute(context.Background(), scope, id, input); err != nil {
		t.Fatal(err)
	}
	request := BindingRequest{ConsentGrants: true, Bindings: []Binding{{AccountID: "account-one", LedgerAccountID: account, CurrencyVersionID: currency, BindingDate: "2026-01-05", BindingEvidenceRef: "00000000-0000-4000-8000-000000000081"}}}
	start := make(chan struct{})
	results := make(chan error, 2)
	db := admin.Config().Database
	go func() {
		<-start
		_, err := proofInvoke(t, db, "synthetic_runtime", "SELECT integration.begin_binding_abandon($1,$2,$3,$4)::text", actor, entity, book, id)
		results <- err
	}()
	go func() {
		<-start
		_, err := proofInvoke(t, db, "synthetic_runtime", "SELECT integration.bind_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, id, "race-bind", proofRaw(request))
		results <- err
	}()
	close(start)
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatal("binding and cancellation did not serialize", first, second)
	}
	state := proofValue(t, admin, "SELECT state FROM integration.connection_attempt WHERE attempt_id=$1", id)
	if state != "active" && state != "disconnect_pending" {
		t.Fatal("unexpected race winner", state)
	}
}

// TestConnectionServiceSQLSecretUncertainty recovers a committed immutable secret write without exchanging again.
func TestConnectionServiceSQLSecretUncertainty(t *testing.T) {
	s, admin, p, store := sqlService(t)
	id, input := sqlBegin(t, s, "secret")
	store.lostPut = true
	_, e := s.Execute(context.Background(), scope, id, input)
	code(t, e, "credential_recovery_required")
	store.lostPut = false
	if _, e = s.Reconcile(context.Background(), scope, id); e != nil {
		t.Fatal(e)
	}
	if count(p.calls, "exchange") != 1 || store.writes != 1 || proofValue(t, admin, "SELECT state FROM integration.connection_attempt") != "binding_review" {
		t.Fatal("uncertain credential recovery duplicated effect")
	}
}

// TestConnectionServiceSQLRevocation proves live actor revocation after exchange blocks storage/publication.
func TestConnectionServiceSQLRevocation(t *testing.T) {
	s, admin, p, store := sqlService(t)
	id, input := sqlBegin(t, s, "revoke")
	p.hook = func(op string) {
		if op == "exchange" {
			proofExec(t, admin, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", actor)
		}
	}
	_, e := s.Execute(context.Background(), scope, id, input)
	code(t, e, "not_found")
	if store.writes != 0 || count(p.calls, "manifest") != 0 || proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_item") != "0" {
		t.Fatal("revoked actor published credentials")
	}
}

// TestConnectionServiceSQLRevocationAfterSecret leaves an orphan private version, never an active reference.
func TestConnectionServiceSQLRevocationAfterSecret(t *testing.T) {
	s, admin, _, store := sqlService(t)
	id, input := sqlBegin(t, s, "revoke-secret")
	store.hook = func() {
		proofExec(t, admin, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", actor)
	}
	_, e := s.Execute(context.Background(), scope, id, input)
	code(t, e, "not_found")
	if store.writes != 1 || store.reads != 0 || proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_item") != "0" {
		t.Fatal("revoked actor read/published secret reference")
	}
}

// lostClaimDB loses only the acknowledgement after an actual claim COMMIT, not the commit itself.
type lostClaimDB struct {
	db   Beginner
	lost bool
}

// BeginTx wraps actual transactions to inject commit acknowledgement loss.
func (d *lostClaimDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, e := d.db.BeginTx(ctx, o)
	if e != nil {
		return nil, e
	}
	return &lostClaimTx{Tx: tx, parent: d}, nil
}

type lostClaimTx struct {
	pgx.Tx
	parent *lostClaimDB
	claim  bool
}

// QueryRow identifies only the one-shot claim boundary.
func (t *lostClaimTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	t.claim = strings.Contains(q, "integration.claim_connection(")
	return t.Tx.QueryRow(ctx, q, args...)
}

// Commit commits for real before injecting one lost acknowledgement.
func (t *lostClaimTx) Commit(ctx context.Context) error {
	e := t.Tx.Commit(ctx)
	if e == nil && t.claim && !t.parent.lost {
		t.parent.lost = true
		return errors.New("synthetic lost claim reply")
	}
	return e
}

// TestConnectionServiceSQLUnknownClaim forbids the first provider call after an ambiguous SQL acknowledgement.
func TestConnectionServiceSQLUnknownClaim(t *testing.T) {
	s, admin, p, store := sqlService(t)
	id, input := sqlBegin(t, s, "claim")
	original := s.repo.(*sqlRepository)
	wrapped := &lostClaimDB{db: original.DB}
	s.repo = &sqlRepository{DB: wrapped}
	_, e := s.Execute(context.Background(), scope, id, input)
	code(t, e, "commit_unknown")
	if _, e = s.Execute(context.Background(), scope, id, input); e != nil {
		t.Fatal(e)
	}
	if !wrapped.lost || count(p.calls, "exchange") != 0 || count(p.calls, "verify") != 0 || store.writes != 0 || proofValue(t, admin, "SELECT state FROM integration.connection_attempt") != "exchange_claimed" {
		t.Fatal("unknown SQL claim leaked provider authority")
	}
}

// TestConnectionServiceSQLItemLifecycle proves same-Item reconnect and rotation publish one shared version.
func TestConnectionServiceSQLItemLifecycle(t *testing.T) {
	s, admin, p, _ := sqlService(t)
	p.manifest.Accounts = append(p.manifest.Accounts, Account{AccountID: "account-two", Type: "depository", Currency: ptr("USD")})
	id, input := sqlBegin(t, s, "new-two")
	if _, e := s.Execute(context.Background(), scope, id, input); e != nil {
		t.Fatal(e)
	}
	bindings := []Binding{{AccountID: "account-one", LedgerAccountID: account, CurrencyVersionID: currency, BindingDate: "2026-01-05", BindingEvidenceRef: "00000000-0000-4000-8000-000000000080"}, {AccountID: "account-two", LedgerAccountID: "00000000-0000-4000-8000-000000000015", CurrencyVersionID: currency, BindingDate: "2026-01-05", BindingEvidenceRef: "00000000-0000-4000-8000-000000000081"}}
	if _, e := s.Bind(context.Background(), scope, id, "bind-two", BindingRequest{ConsentGrants: true, Bindings: bindings}); e != nil {
		t.Fatal(e)
	}
	item := proofValue(t, admin, "SELECT sync_item_id::text FROM integration.sync_item")
	var streams []string
	if e := json.Unmarshal([]byte(proofValue(t, admin, "SELECT json_agg(stream_id ORDER BY stream_id)::text FROM integration.sync_stream")), &streams); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"reconnect", "rotate", "remove"} {
		generation := proofValue(t, admin, "SELECT generation::text FROM integration.sync_item")
		out, e := s.Begin(context.Background(), scope, mode, BeginRequest{Mode: mode, SyncItemID: &item, ExpectedGeneration: &generation, ApprovedStreamIDs: streams})
		if e != nil {
			t.Fatal(mode, e)
		}
		var info struct {
			ID string `json:"attempt_id"`
		}
		_ = json.Unmarshal(out.Attempt, &info)
		req := ExecuteRequest{Nonce: out.Nonce, LinkToken: out.LinkToken}
		if mode == "rotate" {
			p.token = NewSecret("rotated-synthetic-access")
		}
		p.hook = func(op string) {
			if op == "rotate" || op == "remove" {
				if proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_stream WHERE current_set IS NOT NULL") != "0" {
					t.Fatal("siblings not fenced before provider")
				}
				want := "credential_recovery_required"
				if op == "remove" {
					want = "disconnect_pending"
				}
				if proofValue(t, admin, "SELECT state FROM integration.sync_item") != want {
					t.Fatal("Item active during remote mutation")
				}
			}
		}
		if _, e = s.Execute(context.Background(), scope, info.ID, req); e != nil {
			t.Fatal(mode, e)
		}
		want := "active"
		if mode == "remove" {
			want = "disconnected"
		}
		if proofValue(t, admin, "SELECT state FROM integration.sync_item") != want {
			t.Fatal("wrong terminal state")
		}
	}
	if count(p.calls, "exchange") != 1 || count(p.calls, "rotate") != 1 || count(p.calls, "remove") != 1 || proofValue(t, admin, "SELECT count(*)::text FROM integration.sync_stream WHERE state='disconnected'") != "2" {
		t.Fatal("wrong shared lifecycle effects")
	}
}
