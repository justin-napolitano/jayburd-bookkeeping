//go:build integration

package ledger

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// requireLedgerSQLState distinguishes intended rejection from connection or SQL errors.
func requireLedgerSQLState(t testing.TB, err error, want string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != want {
		t.Fatalf("expected PostgreSQL rejection %s, got %v", want, err)
	}
}

// ledgerProofSnapshot captures complete synthetic durable rows, not just row counts.
func ledgerProofSnapshot(t testing.TB, c *pgx.Conn) map[string]string {
	t.Helper()
	state := make(map[string]string)
	for _, table := range []string{"ledger.book", "ledger.journal_entry", "ledger.journal_line", "audit.action", "integration.operation_receipt", "integration.outbox_message", "ledger.book_policy", "reference.currency", "reference.rate_observation"} {
		state[table] = scalar(t, c, "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY to_jsonb(r)::text),'[]'::jsonb)::text FROM "+table+" r")
	}
	return state
}

// requireLedgerUnchanged checks rejected operations cannot rewrite existing facts or receipts.
func requireLedgerUnchanged(t testing.TB, c *pgx.Conn, before map[string]string) {
	t.Helper()
	after := ledgerProofSnapshot(t, c)
	for table, expected := range before {
		if after[table] != expected {
			t.Fatalf("rejected operation changed %s", table)
		}
	}
}

// journalProofLine retains exact decimal strings and reference identities for independent assertions.
type journalProofLine struct {
	Number                                                              int
	Account, Side, Currency, Amount, Functional, Basis, RateObservation string
}

// journalProofLines reads stored synthetic rows in their persisted line order.
func journalProofLines(t testing.TB, c *pgx.Conn, journal string) []journalProofLine {
	t.Helper()
	rows, err := c.Query(context.Background(), `SELECT line_number,account_id::text,side,transaction_currency_version_id::text,
		trim_scale(transaction_amount)::text,trim_scale(functional_amount)::text,valuation_basis,rate_observation_id::text
		FROM ledger.journal_line WHERE entity_id=$1 AND book_id=$2 AND journal_id=$3 ORDER BY line_number`, entity, book, journal)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var lines []journalProofLine
	for rows.Next() {
		var line journalProofLine
		if err := rows.Scan(&line.Number, &line.Account, &line.Side, &line.Currency, &line.Amount, &line.Functional, &line.Basis, &line.RateObservation); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}

// TestSpotReversalHistoricalProvenance proves later rates cannot revalue an exact reversal.
func TestSpotReversalHistoricalProvenance(t *testing.T) {
	c, s, _ := fixtureDB(t, "spot_reversal_history")
	original := change(command("historical-spot"), func(v map[string]any) {
		for _, raw := range v["lines"].([]any) {
			line := raw.(map[string]any)
			line["transaction_currency"] = "EUR"
			line["transaction_amount"] = "100"
			line["functional_amount"] = "110"
			line["valuation_basis"] = "spot"
			line["rate_observation_id"] = rateID
		}
	})
	posted, err := s.Post(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	originalID := postedID(t, posted)
	originalHeader := scalar(t, c, "SELECT to_jsonb(h)::text FROM ledger.journal_entry h WHERE journal_id=$1", originalID)
	expectedOriginal := []journalProofLine{
		{1, accountA, "debit", eur, "100", "110", "spot", rateID},
		{2, accountB, "credit", eur, "100", "110", "spot", rateID},
	}
	if !reflect.DeepEqual(journalProofLines(t, c, originalID), expectedOriginal) {
		t.Fatal("original does not match independent EUR 100 / USD 110 expectations")
	}
	// Prove the different later rate is eligible through a separate real posting.
	laterRate := requestID()
	sql(t, c, `INSERT INTO reference.rate_observation VALUES($1,$2,$3,1.25,'2026-02-01','2026-02-01','synthetic','eur-usd-later','spot','synthetic-reviewed-spot-v1',NULL)`, laterRate, eur, usd)
	later := change(original, func(v map[string]any) {
		v["expected_revision"] = "1"
		v["accounting_date"] = "2026-02-01"
		for _, raw := range v["lines"].([]any) {
			line := raw.(map[string]any)
			line["functional_amount"] = "125"
			line["rate_observation_id"] = laterRate
		}
	})
	later.Key = "later-spot"
	later.Request = requestID()
	laterPosted, err := s.Post(context.Background(), later)
	if err != nil {
		t.Fatal(err)
	}
	expectedLater := []journalProofLine{
		{1, accountA, "debit", eur, "100", "125", "spot", laterRate},
		{2, accountB, "credit", eur, "100", "125", "spot", laterRate},
	}
	if !reflect.DeepEqual(journalProofLines(t, c, postedID(t, laterPosted)), expectedLater) {
		t.Fatal("later posting did not use the independently specified rate")
	}
	reverse := command("historical-reversal")
	reverse.Journal = originalID
	reverse.Body = `{"expected_revision":"2","accounting_date":"2026-02-01","reason":"Synthetic historical reversal"}`
	reversed, err := s.Reverse(context.Background(), reverse)
	if err != nil {
		t.Fatal(err)
	}
	reversalID := postedID(t, reversed)
	expectedReversal := []journalProofLine{
		{1, accountA, "credit", eur, "100", "110", "spot", rateID},
		{2, accountB, "debit", eur, "100", "110", "spot", rateID},
	}
	if !reflect.DeepEqual(journalProofLines(t, c, reversalID), expectedReversal) {
		t.Fatal("reversal changed historical amounts, references, sides or line order")
	}
	if scalar(t, c, `SELECT (entity_id=$2 AND book_id=$3 AND policy_id=$4 AND posted_by=$5
		AND reverses_journal_id=$6 AND kind='reversal' AND accounting_date='2026-02-01'
		AND period_id='00000000-0000-4000-8000-000000000009' AND book_revision=3
		AND reason='Synthetic historical reversal')::text FROM ledger.journal_entry WHERE journal_id=$1`, reversalID, entity, book, policyID, actor, originalID) != "true" {
		t.Fatal("reversal header lost scope, policy, target-period or original linkage")
	}
	if scalar(t, c, "SELECT to_jsonb(h)::text FROM ledger.journal_entry h WHERE journal_id=$1", originalID) != originalHeader ||
		!reflect.DeepEqual(journalProofLines(t, c, originalID), expectedOriginal) {
		t.Fatal("reversal rewrote original history")
	}
	// Replaying the reversal must return the original receipt without any durable change.
	before := ledgerProofSnapshot(t, c)
	replayed, err := s.Reverse(context.Background(), reverse)
	if err != nil || !replayed.Replayed || postedID(t, replayed) != reversalID {
		t.Fatal("historical reversal receipt did not replay", err)
	}
	requireLedgerUnchanged(t, c, before)
}

// TestReversalOfReversalDenied proves semantic rejection cannot append history or intent.
func TestReversalOfReversalDenied(t *testing.T) {
	c, s, _ := fixtureDB(t, "reverse_reversal")
	posted, err := s.Post(context.Background(), command("original"))
	if err != nil {
		t.Fatal(err)
	}
	reverse := command("reverse")
	reverse.Journal = postedID(t, posted)
	reverse.Body = `{"expected_revision":"1","accounting_date":"2026-02-01","reason":"Synthetic reversal"}`
	reversed, err := s.Reverse(context.Background(), reverse)
	if err != nil {
		t.Fatal(err)
	}
	before := ledgerProofSnapshot(t, c)
	reverse.Key = "reverse-the-reversal"
	reverse.Request = requestID()
	reverse.Journal = postedID(t, reversed)
	reverse.Body = `{"expected_revision":"2","accounting_date":"2026-02-01","reason":"Disallowed reversal"}`
	_, err = s.Reverse(context.Background(), reverse)
	requireFailure(t, err, "already_reversed")
	requireLedgerUnchanged(t, c, before)
	read := command("")
	read.Journal = postedID(t, posted)
	if _, err := s.Journal(context.Background(), read); err != nil {
		t.Fatal("authorized read failed after rejection", err)
	}
}
