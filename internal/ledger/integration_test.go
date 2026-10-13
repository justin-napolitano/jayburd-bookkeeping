//go:build integration

package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proofPort = flag.Int("ledger-port", 0, "Port of the runner's disposable internal-network PostgreSQL")
var restored = flag.Bool("ledger-restored", false, "Verify only the runner's restored synthetic database")

const actor = "00000000-0000-4000-8000-000000000001"
const entity = "00000000-0000-4000-8000-000000000002"
const book = "00000000-0000-4000-8000-000000000003"
const policyID = "00000000-0000-4000-8000-000000000004"
const usd = "00000000-0000-4000-8000-000000000005"
const eur = "00000000-0000-4000-8000-000000000006"
const rateID = "00000000-0000-4000-8000-000000000007"
const accountA = "00000000-0000-4000-8000-000000000011"
const accountB = "00000000-0000-4000-8000-000000000012"

// connection always targets the runner's explicit synthetic internal-network database.
func connection(t testing.TB, db, user string) *pgx.Conn {
	t.Helper()
	if *proofPort < 1 || *proofPort > 65535 {
		t.Fatal("use scripts/test-ledger.sh; no inherited DSN accepted")
	}
	c, err := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=%d user=%s dbname=%s sslmode=disable connect_timeout=5", *proofPort, user, db))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// sql executes controlled fixture/admin SQL, never runtime authorization.
func sql(t testing.TB, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, err := c.Exec(context.Background(), q, args...); err != nil {
		t.Fatal(err)
	}
}

// scalar inspects only the disposable synthetic database for exact acceptance.
func scalar(t testing.TB, c *pgx.Conn, q string, args ...any) string {
	t.Helper()
	var s string
	if err := c.QueryRow(context.Background(), q, args...).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// fixtureDB clones migrated structure; controlled setup is outside runtime grants.
func fixtureDB(t *testing.T, name string) (*pgx.Conn, *Service, string) {
	t.Helper()
	db := "ledger_" + strings.ToLower(name)
	root := connection(t, "postgres", "postgres")
	defer root.Close(context.Background())
	sql(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE ledger_template")
	c := connection(t, db, "postgres")
	t.Cleanup(func() { c.Close(context.Background()) })
	sql(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','poster')`, actor)
	sql(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional LLC','legal')`, entity)
	sql(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'post',NULL),($1,$2,'read',NULL)`, entity, actor)
	sql(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Synthetic accrual')`, entity, book)
	sql(t, c, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic-fixture'),($2,'EUR','978',2,'2000-01-01',NULL,'synthetic-fixture')`, usd, eur)
	sql(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())`, entity, book, policyID, usd, actor)
	sql(t, c, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000008','2026-01-01','2026-02-01'),($1,$2,'00000000-0000-4000-8000-000000000009','2026-02-01','2026-03-01')`, entity, book)
	sql(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'Assets:Checking','Checking','asset','debit',true,'2026-01-01',NULL),($1,$2,$4,'Equity:Capital','Capital','equity','credit',false,'2026-01-01',NULL)`, entity, book, accountA, accountB)
	sql(t, c, `INSERT INTO reference.rate_observation VALUES($1,$2,$3,1.10,'2026-01-01','2026-01-01','synthetic','eur-usd','spot','synthetic-reviewed-spot-v1',NULL)`, rateID, eur, usd)
	sql(t, c, `UPDATE ledger.book SET revision=0`)
	return c, &Service{DB: &testDB{t: t, name: db}}, db
}

// testDB creates actual runtime-role sessions, not an owner session with assertions.
type testDB struct {
	t    testing.TB
	name string
}

// BeginTx owns a new runtime connection for one service attempt.
func (d *testDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	c := connection(d.t, d.name, "synthetic_runtime")
	tx, err := c.BeginTx(ctx, o)
	if err != nil {
		c.Close(context.Background())
		return nil, err
	}
	return &closingTx{Tx: tx, conn: c}, nil
}

// closingTx keeps the test connector equivalent to a pool's transaction lifecycle.
type closingTx struct {
	pgx.Tx
	conn *pgx.Conn
}

// Commit releases this attempt's connection even when commit fails.
func (t *closingTx) Commit(ctx context.Context) error {
	err := t.Tx.Commit(ctx)
	_ = t.conn.Close(ctx)
	return err
}

// Rollback closes abandoned sessions so locks cannot leak into another assertion.
func (t *closingTx) Rollback(ctx context.Context) error {
	err := t.Tx.Rollback(ctx)
	_ = t.conn.Close(ctx)
	return err
}

// command supplies trusted synthetic identity plus a balanced original request.
func command(key string) Command {
	return Command{Actor: actor, Entity: entity, Book: book, Key: key, Request: requestID(), Body: fmt.Sprintf(`{"expected_revision":"0","policy_id":%q,"accounting_date":"2026-01-01","description":"Synthetic capital","lines":[{"account_id":%q,"side":"debit","transaction_currency":"USD","transaction_amount":"1000","functional_amount":"1000","valuation_basis":"identity"},{"account_id":%q,"side":"credit","transaction_currency":"USD","transaction_amount":"1000","functional_amount":"1000","valuation_basis":"identity"}]}`, policyID, accountA, accountB)}
}

// change edits only an explicitly named synthetic request for a rejection case.
func change(c Command, f func(map[string]any)) Command {
	var v map[string]any
	_ = json.Unmarshal([]byte(c.Body), &v)
	f(v)
	b, _ := json.Marshal(v)
	c.Body = string(b)
	return c
}

// requireFailure checks safe semantic codes rather than brittle SQL messages.
func requireFailure(t testing.TB, err error, code string) {
	t.Helper()
	var f *Failure
	if !errors.As(err, &f) || f.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}

// postedID decodes only the generated journal identity for subsequent operations.
func postedID(t testing.TB, r Result) string {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatal(err)
	}
	return v["journal_id"].(string)
}

// assertEmpty proves a failure left every atomic component and revision unchanged.
func assertEmpty(t testing.TB, c *pgx.Conn) {
	t.Helper()
	for _, table := range []string{"ledger.journal_entry", "ledger.journal_line", "audit.action", "integration.operation_receipt", "integration.outbox_message"} {
		if scalar(t, c, "SELECT count(*)::text FROM "+table) != "0" {
			t.Fatal("partial write", table)
		}
	}
	if scalar(t, c, "SELECT revision::text FROM ledger.book") != "0" {
		t.Fatal("revision changed")
	}
}

// TestPostingInvariants rejects malformed/accounting input and rolls back every write.
func TestPostingInvariants(t *testing.T) {
	c, s, _ := fixtureDB(t, "invariants")
	cases := []struct {
		name, code string
		mutate     func(map[string]any)
	}{
		{"unbalanced", "invalid_journal", func(v map[string]any) {
			l := v["lines"].([]any)[0].(map[string]any)
			l["transaction_amount"] = "999"
			l["functional_amount"] = "999"
		}},
		{"empty", "invalid_journal", func(v map[string]any) { v["lines"] = []any{} }},
		{"unknown", "invalid_request", func(v map[string]any) { v["actor"] = actor }},
		{"unknown_account", "invalid_journal", func(v map[string]any) { v["lines"].([]any)[0].(map[string]any)["account_id"] = actor }},
		{"null", "invalid_request", func(v map[string]any) { v["description"] = nil }},
		{"bad_date", "invalid_request", func(v map[string]any) { v["accounting_date"] = "2026-02-30" }},
		{"stale", "revision_conflict", func(v map[string]any) { v["expected_revision"] = "1" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.Post(context.Background(), change(command(tc.name), tc.mutate))
			requireFailure(t, err, tc.code)
			assertEmpty(t, c)
		})
	}
	for _, raw := range []string{strings.Replace(command("dup").Body, `"description":`, `"description":"one","description":`, 1), strings.Replace(command("dup").Body, `"side":"debit"`, `"side":"debit","side":"debit"`, 1)} {
		cmd := command("dup")
		cmd.Body = raw
		_, err := s.Post(context.Background(), cmd)
		requireFailure(t, err, "invalid_request")
		assertEmpty(t, c)
	}
	// AFTER triggers fail after the target write; the deferred trigger fails at COMMIT.
	sql(t, c, `CREATE FUNCTION public.inject_failure() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'synthetic failure'; END$$`)
	for _, table := range []string{"ledger.book", "ledger.journal_entry", "ledger.journal_line", "audit.action", "integration.operation_receipt", "integration.outbox_message"} {
		sql(t, c, "CREATE TRIGGER fail AFTER INSERT OR UPDATE ON "+table+" FOR EACH ROW EXECUTE FUNCTION public.inject_failure()")
		_, err := s.Post(context.Background(), command("failure"))
		requireFailure(t, err, "internal_error")
		assertEmpty(t, c)
		sql(t, c, "DROP TRIGGER fail ON "+table)
	}
	sql(t, c, `CREATE CONSTRAINT TRIGGER fail AFTER INSERT ON integration.outbox_message DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION public.inject_failure()`)
	_, err := s.Post(context.Background(), command("commit-fail"))
	requireFailure(t, err, "internal_error")
	assertEmpty(t, c)
	sql(t, c, "DROP TRIGGER fail ON integration.outbox_message")
	if _, err = s.Post(context.Background(), command("failure")); err != nil {
		t.Fatal(err)
	}
}

// TestMoneyInputBounds exercises exact SQL validation and supported currency units.
func TestMoneyInputBounds(t *testing.T) {
	c, s, _ := fixtureDB(t, "money")
	for _, value := range []string{"0", "-1", "NaN", "Infinity", "1e3", "1000.0000000000", "1000.001", "100000000000000000000000000000"} {
		cmd := change(command(value), func(v map[string]any) {
			l := v["lines"].([]any)[0].(map[string]any)
			l["transaction_amount"] = value
			l["functional_amount"] = value
		})
		cmd.Key = "invalid"
		_, err := s.Post(context.Background(), cmd)
		requireFailure(t, err, "invalid_amount")
		assertEmpty(t, c)
	}
	for _, tc := range []struct{ x, w string }{{"1.005", "1"}, {"1.015", "1.02"}, {"2.675", "2.68"}} {
		if scalar(t, c, "SELECT trim_scale(ledger.round_even($1::numeric,2))::text", tc.x) != tc.w {
			t.Fatal("half-even mismatch")
		}
	}
	cmd := command("trailing")
	cmd.Body = strings.ReplaceAll(cmd.Body, `"1000"`, `"1000.000"`)
	if _, err := s.Post(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
}

// TestManualSpotPosting retains provenance and rejects invalid rate semantics.
func TestManualSpotPosting(t *testing.T) {
	c, s, _ := fixtureDB(t, "spot")
	cmd := change(command("spot"), func(v map[string]any) {
		for _, x := range v["lines"].([]any) {
			l := x.(map[string]any)
			l["transaction_currency"] = "EUR"
			l["transaction_amount"] = "100"
			l["functional_amount"] = "110"
			l["valuation_basis"] = "spot"
			l["rate_observation_id"] = rateID
		}
	})
	for _, field := range []string{"effective_at", "source", "base_currency_version_id", "quote_currency_version_id"} {
		// New immutable observations, not edits of historical references.
		id := requestID()
		query := `INSERT INTO reference.rate_observation SELECT $1::uuid,base_currency_version_id,quote_currency_version_id,quote_per_base,effective_at,observed_at,source,$1::uuid::text,purpose,policy_code,NULL FROM reference.rate_observation WHERE rate_observation_id=$2`
		switch field {
		case "effective_at":
			query = strings.Replace(query, "quote_per_base,effective_at", "quote_per_base,'2025-12-31'", 1)
		case "source":
			query = strings.Replace(query, "observed_at,source", "observed_at,'unreviewed'", 1)
		default:
			query = strings.Replace(query, "base_currency_version_id,quote_currency_version_id", "quote_currency_version_id,base_currency_version_id", 1)
		}
		sql(t, c, query, id, rateID)
		bad := change(cmd, func(v map[string]any) { v["lines"].([]any)[0].(map[string]any)["rate_observation_id"] = id })
		_, err := s.Post(context.Background(), bad)
		requireFailure(t, err, "invalid_journal")
		assertEmpty(t, c)
	}
	r, err := s.Post(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(r.Body), rateID) || !strings.Contains(string(r.Body), `"functional_amount": "110"`) {
		t.Fatal("valuation provenance missing")
	}
}

// TestSchemaPrivileges requires exact permission denials, unchanged state and usable authorized operations.
func TestSchemaPrivileges(t *testing.T) {
	c, s, db := fixtureDB(t, "privileges")
	// Seed real history so UPDATE/DELETE/TRUNCATE probes cannot pass vacuously.
	if _, err := s.Post(context.Background(), command("privilege-seed")); err != nil {
		t.Fatal(err)
	}
	before := ledgerProofSnapshot(t, c)
	runtime := connection(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	for _, q := range []string{"SELECT * FROM ledger.journal_entry", "UPDATE ledger.book SET revision=99", "DELETE FROM ledger.journal_line", "TRUNCATE ledger.journal_entry CASCADE", "CREATE TABLE public.evil(n int)", "CREATE TABLE ledger.evil(n int)", "SET ROLE bookkeeping_owner", "SELECT ledger.round_even(1.005,2)"} {
		_, err := runtime.Exec(context.Background(), q)
		requireLedgerSQLState(t, err, "42501")
		requireLedgerUnchanged(t, c, before)
	}
	sql(t, runtime, "CREATE TEMP TABLE journal_entry(n int); SET search_path=pg_temp,public")
	tx, err := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	cmd := change(command("safe"), func(v map[string]any) { v["expected_revision"] = "1" })
	var r Result
	err = tx.QueryRow(context.Background(), "SELECT * FROM ledger.post_journal($1,$2,$3,$4,$5,$6)", cmd.Actor, cmd.Entity, cmd.Book, cmd.Key, cmd.Request, cmd.Body).Scan(&r.Status, &r.Body, &r.Replayed)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	cmd.Journal = postedID(t, r)
	if _, err = s.Journal(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
}

// TestPostingReplayConcurrency proves one durable effect and semantic equality.
func TestPostingReplayConcurrency(t *testing.T) {
	c, s, _ := fixtureDB(t, "replay")
	cmd := command("same")
	lockTx, err := c.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lockTx.Exec(context.Background(), "SELECT 1 FROM ledger.book FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan Result, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			r, err := s.Post(context.Background(), cmd)
			results <- r
			errs <- err
		}()
	}
	close(start)
	deadline := time.Now().Add(3 * time.Second)
	blocked := false
	for time.Now().Before(deadline) {
		sql(t, c, "SELECT pg_stat_clear_snapshot()")
		var n int
		err := c.QueryRow(context.Background(), "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND usename='synthetic_runtime' AND wait_event_type='Lock'").Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n == 2 {
			blocked = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !blocked {
		t.Fatal("two callers did not reach lock barrier")
	}
	if err := lockTx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	close(errs)
	close(results)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	ids := map[string]bool{}
	replays := 0
	for r := range results {
		ids[postedID(t, r)] = true
		if r.Replayed {
			replays++
		}
	}
	if len(ids) != 1 || replays != 1 {
		t.Fatal("duplicate financial effect")
	}
	if scalar(t, c, "SELECT revision::text FROM ledger.book") != "1" || scalar(t, c, "SELECT count(*)::text FROM integration.outbox_message") != "1" {
		t.Fatal("duplicate revision/intent")
	}
	equivalent := change(cmd, func(v map[string]any) {
		v["policy_id"] = strings.ToUpper(policyID)
		for _, x := range v["lines"].([]any) {
			l := x.(map[string]any)
			l["transaction_amount"] = "1000.00"
			l["functional_amount"] = "1000.000"
		}
	})
	r, err := s.Post(context.Background(), equivalent)
	if err != nil || !r.Replayed {
		t.Fatal("normalized replay", err)
	}
	changed := change(cmd, func(v map[string]any) { ls := v["lines"].([]any); ls[0], ls[1] = ls[1], ls[0] })
	_, err = s.Post(context.Background(), changed)
	requireFailure(t, err, "idempotency_conflict")
	sql(t, c, `UPDATE ledger.period SET state='closed',closed_at=now(),closed_by=$1,close_revision=1 WHERE starts_on='2026-01-01'`, actor)
	r, err = s.Post(context.Background(), cmd)
	if err != nil || !r.Replayed {
		t.Fatal("replay after close", err)
	}
	sql(t, c, `UPDATE ledger.membership SET revoked_at=now() WHERE permission='post'`)
	_, err = s.Post(context.Background(), cmd)
	requireFailure(t, err, "permission_denied")
}

// TestJournalCorrectionHistory verifies reversal and trigger rejection without changing durable history.
func TestJournalCorrectionHistory(t *testing.T) {
	c, s, _ := fixtureDB(t, "correction")
	cmd := command("original")
	r, err := s.Post(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Journal = postedID(t, r)
	cmd.Key = "reverse"
	cmd.Body = `{"expected_revision":"1","accounting_date":"2026-02-01","reason":"Synthetic correction"}`
	if _, err = s.Reverse(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	cmd.Key = "reverse-again"
	cmd.Body = strings.Replace(cmd.Body, `"1"`, `"2"`, 1)
	_, err = s.Reverse(context.Background(), cmd)
	requireFailure(t, err, "already_reversed")
	before := ledgerProofSnapshot(t, c)
	for _, q := range []string{"UPDATE ledger.journal_entry SET description='rewrite'", "DELETE FROM ledger.journal_line"} {
		_, err = c.Exec(context.Background(), q)
		requireLedgerSQLState(t, err, "P2108")
		requireLedgerUnchanged(t, c, before)
	}
	cmd.Body = "2026-02-28"
	tb, err := s.TrialBalance(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	_ = json.Unmarshal(tb.Body, &v)
	if v["total_debits"] != "0" || v["total_credits"] != "0" {
		t.Fatal("reversal not included")
	}
}

// TestGoldenMonth drives the committed independent oracle through actual SQL.
func TestGoldenMonth(t *testing.T) {
	c, s, _ := fixtureDB(t, "golden")
	data, err := os.ReadFile("tests/corpus/golden-month.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Books []struct {
			Accounts []string `json:"accounts"`
			Entries  []struct {
				ID, Date, Kind, Reverses string
				Lines                    []struct {
					Account, Side, Currency, Amount string
					Functional                      string `json:"functional_amount"`
				}
			}
			Expected map[string]string `json:"expected_balances"`
		}
	}
	if err = json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{"Assets:Checking": accountA, "Equity:Capital": accountB}
	for _, a := range corpus.Books[0].Accounts {
		if ids[a] != "" {
			continue
		}
		id := requestID()
		ids[a] = id
		category := "expense"
		side := "debit"
		switch strings.Split(a, ":")[0] {
		case "Assets":
			category = "asset"
		case "Equity":
			category = "equity"
			side = "credit"
		case "Income":
			category = "revenue"
			side = "credit"
		case "Liabilities":
			category = "liability"
			side = "credit"
		}
		sql(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,$4,$4,$5,$6,false,'2026-01-01',NULL)`, entity, book, id, a, category, side)
	}
	sql(t, c, "UPDATE ledger.book SET revision=0")
	journals := map[string]string{}
	for i, e := range corpus.Books[0].Entries {
		cmd := command(e.ID)
		v := map[string]any{"expected_revision": fmt.Sprint(i), "accounting_date": e.Date}
		if e.Kind == "reversal" {
			cmd.Journal = journals[e.Reverses]
			v["reason"] = "Synthetic correction"
		} else {
			v["policy_id"] = policyID
			v["description"] = e.ID
			ls := []any{}
			for _, l := range e.Lines {
				ls = append(ls, map[string]any{"account_id": ids[l.Account], "side": l.Side, "transaction_currency": l.Currency, "transaction_amount": l.Amount, "functional_amount": l.Functional, "valuation_basis": "identity"})
			}
			v["lines"] = ls
		}
		body, _ := json.Marshal(v)
		cmd.Body = string(body)
		var r Result
		if e.Kind == "reversal" {
			r, err = s.Reverse(context.Background(), cmd)
		} else {
			r, err = s.Post(context.Background(), cmd)
		}
		if err != nil {
			t.Fatalf("%s: %v", e.ID, err)
		}
		journals[e.ID] = postedID(t, r)
	}
	cmd := command("")
	cmd.Body = "2026-01-31"
	r, err := s.TrialBalance(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		TotalDebits  string `json:"total_debits"`
		TotalCredits string `json:"total_credits"`
		Accounts     []struct {
			Code   string `json:"account_code"`
			Debit  string `json:"debit_balance"`
			Credit string `json:"credit_balance"`
		}
	}
	if err = json.Unmarshal(r.Body, &report); err != nil {
		t.Fatal(err)
	}
	if report.TotalDebits != "1300" || report.TotalCredits != "1300" {
		t.Fatal("golden totals", string(r.Body))
	}
	for _, a := range report.Accounts {
		dr, _ := new(big.Rat).SetString(a.Debit)
		cr, _ := new(big.Rat).SetString(a.Credit)
		want, _ := new(big.Rat).SetString(corpus.Books[0].Expected[a.Code])
		if dr.Sub(dr, cr).Cmp(want) != 0 {
			t.Fatal("oracle mismatch", a.Code)
		}
	}
	for account, want := range corpus.Books[0].Expected {
		got := scalar(t, c, "SELECT trim_scale(coalesce(sum(CASE side WHEN 'debit' THEN transaction_amount ELSE -transaction_amount END),0))::text FROM ledger.journal_line WHERE entity_id=$1 AND book_id=$2 AND account_id=$3", entity, book, ids[account])
		a, _ := new(big.Rat).SetString(got)
		b, _ := new(big.Rat).SetString(want)
		if a.Cmp(b) != 0 {
			t.Fatal("original units mismatch", account)
		}
	}
	for _, table := range []string{"ledger.journal_entry", "audit.action", "integration.operation_receipt", "integration.outbox_message"} {
		if scalar(t, c, "SELECT count(*)::text FROM "+table) != fmt.Sprint(len(corpus.Books[0].Entries)) {
			t.Fatal("atomic count", table)
		}
	}
}

// TestLedgerRestoreReplay proves a restored successful receipt cannot repost.
func TestLedgerRestoreReplay(t *testing.T) {
	var c *pgx.Conn
	var s *Service
	if *restored {
		c = connection(t, "ledger_restored", "postgres")
		defer c.Close(context.Background())
		s = &Service{DB: &testDB{t: t, name: "ledger_restored"}}
	} else {
		c, s, _ = fixtureDB(t, "restore")
	}
	r, err := s.Post(context.Background(), command("restore"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Replayed != *restored {
		t.Fatal("restore replay state")
	}
	if scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" {
		t.Fatal("duplicate after restore")
	}
}

// waitBlocked uses database lock evidence, not a guessed sleep, to order races.
func waitBlocked(t *testing.T, c *pgx.Conn, pid uint32) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var blocked bool
		if err := c.QueryRow(context.Background(), "SELECT cardinality(pg_blocking_pids($1))>0", int(pid)).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("expected lock barrier")
}

// TestPostingCloseRace forces each real database serialization order.
func TestPostingCloseRace(t *testing.T) {
	for _, closeFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(closeFirst), func(t *testing.T) {
			c, s, db := fixtureDB(t, fmt.Sprintf("race_%t", closeFirst))
			closer := connection(t, db, "postgres")
			defer closer.Close(context.Background())
			if closeFirst {
				tx, err := closer.Begin(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				_, err = tx.Exec(context.Background(), `UPDATE ledger.period SET state='closed',closed_at=now(),closed_by=$1,close_revision=0 WHERE starts_on='2026-01-01'`, actor)
				if err != nil {
					t.Fatal(err)
				}
				// Pin a runtime connection so its blocked PID is observable.
				runtime := connection(t, db, "synthetic_runtime")
				defer runtime.Close(context.Background())
				s.DB = runtime
				done := make(chan error, 1)
				cmd := change(command("race"), func(v map[string]any) { v["expected_revision"] = "1" })
				go func() { _, err := s.Post(context.Background(), cmd); done <- err }()
				waitBlocked(t, c, runtime.PgConn().PID())
				if err = tx.Commit(context.Background()); err != nil {
					t.Fatal(err)
				}
				requireFailure(t, <-done, "period_closed")
			} else {
				runtime := connection(t, db, "synthetic_runtime")
				defer runtime.Close(context.Background())
				tx, err := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
				if err != nil {
					t.Fatal(err)
				}
				cmd := command("race")
				var r Result
				err = tx.QueryRow(context.Background(), "SELECT * FROM ledger.post_journal($1,$2,$3,$4,$5,$6)", cmd.Actor, cmd.Entity, cmd.Book, cmd.Key, cmd.Request, cmd.Body).Scan(&r.Status, &r.Body, &r.Replayed)
				if err != nil {
					t.Fatal(err)
				}
				done := make(chan string, 1)
				go func() {
					var rev string
					err := closer.QueryRow(context.Background(), "SELECT revision::text FROM ledger.book FOR UPDATE").Scan(&rev)
					if err != nil {
						done <- "error"
					} else {
						done <- rev
					}
				}()
				waitBlocked(t, c, closer.PgConn().PID())
				if err = tx.Commit(context.Background()); err != nil {
					t.Fatal(err)
				}
				if <-done != "1" {
					t.Fatal("stale close evidence did not invalidate")
				}
			}
		})
	}
}

// TestEntityReferencesAndPrivileges checks all scope identities against actual SQL.
func TestEntityReferencesAndPrivileges(t *testing.T) {
	c, s, _ := fixtureDB(t, "scope")
	cmd := command("scope")
	cmd.Entity = actor
	_, err := s.Post(context.Background(), cmd)
	requireFailure(t, err, "not_found")
	cmd = command("scope")
	cmd.Book = actor
	_, err = s.Post(context.Background(), cmd)
	requireFailure(t, err, "not_found")
	cmd = command("scope")
	cmd.Actor = entity
	_, err = s.Post(context.Background(), cmd)
	requireFailure(t, err, "not_found")
	assertEmpty(t, c)
	sql(t, c, `UPDATE ledger.entity SET disabled_at=now()`)
	_, err = s.Post(context.Background(), command("disabled"))
	requireFailure(t, err, "not_found")
	assertEmpty(t, c)
}

// TestRevocationRace orders the administrative winner before a blocked poster.
func TestRevocationRace(t *testing.T) {
	for _, table := range []string{"principal", "entity", "membership"} {
		t.Run(table, func(t *testing.T) {
			c, s, db := fixtureDB(t, "revoke_"+table)
			admin := connection(t, db, "postgres")
			defer admin.Close(context.Background())
			tx, err := admin.Begin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			q := "UPDATE ledger." + table + " SET disabled_at=now()"
			code := "not_found"
			if table == "membership" {
				q = "UPDATE ledger.membership SET revoked_at=now() WHERE permission='post'"
				code = "permission_denied"
			}
			if _, err = tx.Exec(context.Background(), q); err != nil {
				t.Fatal(err)
			}
			runtime := connection(t, db, "synthetic_runtime")
			defer runtime.Close(context.Background())
			s.DB = runtime
			done := make(chan error, 1)
			go func() { _, err := s.Post(context.Background(), command("revoked")); done <- err }()
			waitBlocked(t, c, runtime.PgConn().PID())
			if err = tx.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}
			requireFailure(t, <-done, code)
			assertEmpty(t, c)
		})
	}
}

// TestTransactionIsolation rejects direct runtime invocation at a weaker isolation.
func TestTransactionIsolation(t *testing.T) {
	_, _, db := fixtureDB(t, "isolation")
	c := connection(t, db, "synthetic_runtime")
	defer c.Close(context.Background())
	cmd := command("weak")
	_, err := c.Exec(context.Background(), "SELECT * FROM ledger.post_journal($1,$2,$3,$4,$5,$6)", cmd.Actor, cmd.Entity, cmd.Book, cmd.Key, cmd.Request, cmd.Body)
	var p *pgconn.PgError
	if !errors.As(err, &p) || p.Code != "P2100" {
		t.Fatal("weak isolation accepted", err)
	}
}

// lostCommitDB wraps a real transaction to discard one successful commit reply.
type lostCommitDB struct {
	Beginner
	lost       bool
	replyError error // Optional synthetic database-reported completion uncertainty.
}

// BeginTx simulates only acknowledgement loss, not a fake database effect.
func (d *lostCommitDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, err := d.Beginner.BeginTx(ctx, o)
	if err != nil {
		return nil, err
	}
	return &lostCommitTx{Tx: tx, parent: d}, nil
}

// lostCommitTx retains actual commit/receipt semantics underneath the fault.
type lostCommitTx struct {
	pgx.Tx
	parent *lostCommitDB
}

// Commit loses the first acknowledgement after PostgreSQL has committed.
func (t *lostCommitTx) Commit(ctx context.Context) error {
	if err := t.Tx.Commit(ctx); err != nil {
		return err
	}
	if !t.parent.lost {
		t.parent.lost = true
		if t.parent.replyError != nil {
			return t.parent.replyError
		}
		return errors.New("synthetic connection lost after commit")
	}
	return nil
}

// TestCommitAcknowledgementLoss requires 503 then resolves the persisted receipt.
func TestCommitAcknowledgementLoss(t *testing.T) {
	c, s, _ := fixtureDB(t, "lost_ack")
	s.DB = &lostCommitDB{Beginner: s.DB}
	cmd := command("lost")
	r, err := s.Post(context.Background(), cmd)
	requireFailure(t, err, "temporarily_unavailable")
	if len(r.Body) != 0 {
		t.Fatal("returned unacknowledged success")
	}
	r, err = s.Post(context.Background(), cmd)
	if err != nil || !r.Replayed {
		t.Fatal("uncertain commit did not resolve through receipt", err)
	}
	if scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" {
		t.Fatal("duplicate uncertain commit")
	}
}

// TestPostingWinsRevocation proves already-serialized work commits before revoke.
func TestPostingWinsRevocation(t *testing.T) {
	for _, table := range []string{"principal", "entity", "membership"} {
		t.Run(table, func(t *testing.T) {
			c, s, db := fixtureDB(t, "wins_"+table)
			runtime := connection(t, db, "synthetic_runtime")
			defer runtime.Close(context.Background())
			tx, err := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if err != nil {
				t.Fatal(err)
			}
			cmd := command("wins")
			var r Result
			err = tx.QueryRow(context.Background(), "SELECT * FROM ledger.post_journal($1,$2,$3,$4,$5,$6)", cmd.Actor, cmd.Entity, cmd.Book, cmd.Key, cmd.Request, cmd.Body).Scan(&r.Status, &r.Body, &r.Replayed)
			if err != nil {
				t.Fatal(err)
			}
			admin := connection(t, db, "postgres")
			defer admin.Close(context.Background())
			q := "UPDATE ledger." + table + " SET disabled_at=now()"
			code := "not_found"
			if table == "membership" {
				q = "UPDATE ledger.membership SET revoked_at=now() WHERE permission='post'"
				code = "permission_denied"
			}
			done := make(chan error, 1)
			go func() { _, err := admin.Exec(context.Background(), q); done <- err }()
			waitBlocked(t, c, admin.PgConn().PID())
			if err = tx.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err = <-done; err != nil {
				t.Fatal(err)
			}
			if scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" {
				t.Fatal("serialized post lost")
			}
			_, err = s.Post(context.Background(), cmd)
			requireFailure(t, err, code)
		})
	}
}

// TestCurrencyScalesAndRounding posts real SQL conversions with 0/2/3-decimal units.
func TestCurrencyScalesAndRounding(t *testing.T) {
	for i, tc := range []struct {
		code, amount, rate, want string
		scale                    int
	}{{"JPY", "1", "1.005", "1", 0}, {"EUR", "1", "1.015", "1.02", 2}, {"KWD", "1.234", "1", "1.23", 3}, {"EUR", "1", "2.675", "2.68", 2}} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c, s, _ := fixtureDB(t, fmt.Sprintf("scales_%d", i))
			cid := eur
			if tc.code != "EUR" {
				cid = requestID()
				sql(t, c, `INSERT INTO reference.currency VALUES($1,$2,NULL,$3,'2000-01-01',NULL,'synthetic-fixture')`, cid, tc.code, tc.scale)
			}
			rid := requestID()
			sql(t, c, `INSERT INTO reference.rate_observation VALUES($1,$2,$3,$4::numeric,'2026-01-01','2026-01-01','synthetic',$5,'spot','synthetic-reviewed-spot-v1',NULL)`, rid, cid, usd, tc.rate, rid)
			cmd := change(command("scale"), func(v map[string]any) {
				for _, x := range v["lines"].([]any) {
					l := x.(map[string]any)
					l["transaction_currency"] = tc.code
					l["transaction_amount"] = tc.amount
					l["functional_amount"] = tc.want
					l["valuation_basis"] = "spot"
					l["rate_observation_id"] = rid
				}
			})
			if _, err := s.Post(context.Background(), cmd); err != nil {
				t.Fatal(err)
			}
			bad := change(cmd, func(v map[string]any) {
				v["expected_revision"] = "1"
				v["lines"].([]any)[0].(map[string]any)["functional_amount"] = "99"
			})
			bad.Key = "wrong-assertion"
			_, err := s.Post(context.Background(), bad)
			requireFailure(t, err, "invalid_journal")
		})
	}
}

// TestSecondBookIsolation reuses account UUIDs across books without sharing balances.
func TestSecondBookIsolation(t *testing.T) {
	c, s, _ := fixtureDB(t, "two_books")
	e2 := requestID()
	b2 := requestID()
	p2 := requestID()
	sql(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Second fictional LLC','legal')`, e2)
	sql(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'post',NULL),($1,$2,'read',NULL)`, e2, actor)
	sql(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Separate books')`, e2, b2)
	sql(t, c, `INSERT INTO ledger.book_policy SELECT $1,$2,$3,effective_from,effective_to,functional_currency_version_id,basis,timezone,posting_scale,rounding_mode,rate_policy_code,approved_by,approved_at FROM ledger.book_policy`, e2, b2, p2)
	sql(t, c, `INSERT INTO ledger.account SELECT $1,$2,account_id,code,name,category,normal_side,monetary,active_from,active_to FROM ledger.account WHERE entity_id=$3`, e2, b2, entity)
	sql(t, c, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,$3,'2026-01-01','2026-02-01')`, e2, b2, requestID())
	sql(t, c, "UPDATE ledger.book SET revision=0")
	first, err := s.Post(context.Background(), command("same-key"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := command("same-key")
	cmd.Entity = e2
	cmd.Book = b2
	cmd = change(cmd, func(v map[string]any) {
		v["policy_id"] = p2
		for _, x := range v["lines"].([]any) {
			l := x.(map[string]any)
			l["transaction_amount"] = "70"
			l["functional_amount"] = "70"
		}
	})
	if _, err = s.Post(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	cmd.Body = "2026-01-31"
	r, err := s.TrialBalance(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	_ = json.Unmarshal(r.Body, &report)
	if report["total_debits"] != "70" {
		t.Fatal("cross-book balance leak")
	}
	cmd.Journal = postedID(t, first)
	_, err = s.Journal(context.Background(), cmd)
	requireFailure(t, err, "not_found")
	// A line cannot reference an account belonging only to the other book.
	foreign := requestID()
	sql(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'foreign','Foreign','asset','debit',true,'2026-01-01',NULL)`, e2, b2, foreign)
	bad := change(command("foreign"), func(v map[string]any) {
		v["expected_revision"] = "1"
		v["lines"].([]any)[0].(map[string]any)["account_id"] = foreign
	})
	_, err = s.Post(context.Background(), bad)
	requireFailure(t, err, "invalid_journal")
}

// TestHTTPPostCommitBoundary exercises the real service through in-process HTTP.
func TestHTTPPostCommitBoundary(t *testing.T) {
	_, s, _ := fixtureDB(t, "http")
	s.DB = &lostCommitDB{Beginner: s.DB}
	h := Handler(authStub{}, s)
	cmd := command("http")
	path := "/v1/entities/" + entity + "/books/" + book + "/journals"
	for i, want := range []int{503, 201} {
		req := httptest.NewRequest("POST", path, strings.NewReader(cmd.Body))
		req.Header.Set("Authorization", "Bearer synthetic")
		req.Header.Set("Idempotency-Key", cmd.Key)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("attempt %d: %d %s", i, w.Code, w.Body.String())
		}
		if i == 1 && w.Header().Get("Idempotent-Replay") != "true" {
			t.Fatal("HTTP replay missing")
		}
	}
}

// TestReceiptActorAndArchive checks actor binding without weakening current access.
func TestReceiptActorAndArchive(t *testing.T) {
	c, s, _ := fixtureDB(t, "receipt_actor")
	cmd := command("bound")
	if _, err := s.Post(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	other := requestID()
	sql(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','other')`, other)
	sql(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'post',NULL)`, entity, other)
	changed := cmd
	changed.Actor = other
	_, err := s.Post(context.Background(), changed)
	requireFailure(t, err, "idempotency_conflict")
	sql(t, c, `UPDATE ledger.account SET active_to='2026-01-02' WHERE account_id=$1`, accountA)
	r, err := s.Post(context.Background(), cmd)
	if err != nil || !r.Replayed {
		t.Fatal("account archival affected old receipt", err)
	}
	reversal := cmd
	reversal.Journal = postedID(t, r)
	reversal.Key = "archived-reversal"
	reversal.Body = `{"expected_revision":"2","accounting_date":"2026-02-01","reason":"Synthetic"}`
	_, err = s.Reverse(context.Background(), reversal)
	requireFailure(t, err, "invalid_journal")
	sql(t, c, `UPDATE ledger.book SET disabled_at=now()`)
	_, err = s.Post(context.Background(), cmd)
	requireFailure(t, err, "not_found")
}

// TestSchemaCompatibility verifies controlled setup and the reporting migration's version state.
func TestSchemaCompatibility(t *testing.T) {
	c, _, _ := fixtureDB(t, "schema")
	if scalar(t, c, "SELECT version::text||':'||dirty::text FROM public.schema_migrations") != "11:false" {
		t.Fatal("migration state")
	}
	for _, q := range []string{
		`INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000099','USD','840',2,'2025-01-01',NULL,'synthetic')`,
		`INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) SELECT entity_id,book_id,'00000000-0000-4000-8000-000000000099','2026-01-15','2026-02-15' FROM ledger.book`,
		`INSERT INTO ledger.book_policy SELECT entity_id,book_id,'00000000-0000-4000-8000-000000000099','2026-01-15',effective_to,functional_currency_version_id,basis,timezone,posting_scale,rounding_mode,rate_policy_code,approved_by,approved_at FROM ledger.book_policy`,
	} {
		if _, err := c.Exec(context.Background(), q); err == nil {
			t.Fatal("overlap accepted")
		}
	}
}
