//go:build integration

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// demoLine retains original currency quantities; no floating-point conversion occurs.
type demoLine struct{ code, side, amount string }

// demoJournal names an immutable source sample, distinct from its generated ledger ID.
type demoJournal struct {
	source, date string
	lines        []demoLine
}

// demoBook separates fictional organizations and fixed independent balance oracles.
type demoBook struct {
	name, currency string
	accounts       []map[string]any
	entries        []demoJournal
	want           map[string]string
}

// demoAccount supplies reviewed account semantics, never inferred from account numbers.
func demoAccount(code, name, category, side string, monetary bool) map[string]any {
	return map[string]any{"code": code, "name": name, "category": category, "normal_side": side, "monetary": monetary}
}

// demoBooks preserves T04 JE1–4 cents exactly; the separate USD fixture probes 0.10+0.20.
// Source, mapping and rejection limitations are recorded in testdata/demo/README.md.
func demoBooks() []demoBook {
	return []demoBook{
		{name: "Fictional Generator Trading LLC", currency: "EUR", accounts: []map[string]any{
			demoAccount("1000", "Bank", "asset", "debit", true), demoAccount("1200", "Accounts receivable", "asset", "debit", true), demoAccount("1400", "Input VAT — synthetic", "asset", "debit", true), demoAccount("1600", "Accounts payable", "liability", "credit", true), demoAccount("1776", "Output VAT — synthetic", "liability", "credit", true), demoAccount("3000", "Sales revenue", "revenue", "credit", false), demoAccount("4000", "Materials and services", "expense", "debit", false),
		}, entries: []demoJournal{
			{"JE000000001", "2026-01-21", []demoLine{{"1200", "debit", "5749.26"}, {"3000", "credit", "4831.31"}, {"1776", "credit", "917.95"}}},
			{"JE000000002", "2026-01-24", []demoLine{{"4000", "debit", "266.36"}, {"1400", "debit", "50.61"}, {"1600", "credit", "316.97"}}},
			{"JE000000003", "2026-01-21", []demoLine{{"1000", "debit", "5749.26"}, {"1200", "credit", "5749.26"}}},
			{"JE000000004", "2026-01-24", []demoLine{{"1600", "debit", "316.97"}, {"1000", "credit", "316.97"}}},
		}, want: map[string]string{"1000": "5432.29", "1200": "0", "1400": "50.61", "1600": "0", "1776": "-917.95", "3000": "-4831.31", "4000": "266.36"}},
		{name: "Fictional Cents Lab LLC", currency: "USD", accounts: []map[string]any{
			demoAccount("1000", "Bank", "asset", "debit", true), demoAccount("2000", "Owner capital", "equity", "credit", false), demoAccount("4400", "Bank fees", "expense", "debit", false),
		}, entries: []demoJournal{
			{"cents-10", "2026-01-21", []demoLine{{"1000", "debit", "0.10"}, {"2000", "credit", "0.10"}}},
			{"cents-20", "2026-01-21", []demoLine{{"1000", "debit", "0.20"}, {"2000", "credit", "0.20"}}},
			{"cents-fee", "2026-01-24", []demoLine{{"4400", "debit", "0.01"}, {"1000", "credit", "0.01"}}},
		}, want: map[string]string{"1000": "0.29", "2000": "-0.30", "4400": "0.01"}},
	}
}

// validateDemoBooks rejects fractional cents and unbalanced sources instead of plugging differences.
func validateDemoBooks(books []demoBook) error {
	for _, b := range books {
		balances := map[string]*big.Rat{}
		for _, j := range b.entries {
			total := new(big.Rat)
			for _, l := range j.lines {
				if !regexp.MustCompile(`^[0-9]+\.[0-9]{2}$`).MatchString(l.amount) {
					return fmt.Errorf("invalid cents: %s", j.source)
				}
				n, ok := new(big.Rat).SetString(l.amount)
				if !ok || n.Sign() <= 0 {
					return fmt.Errorf("invalid amount: %s", j.source)
				}
				if l.side == "credit" {
					n.Neg(n)
				} else if l.side != "debit" {
					return fmt.Errorf("invalid side")
				}
				total.Add(total, n)
				if balances[l.code] == nil {
					balances[l.code] = new(big.Rat)
				}
				balances[l.code].Add(balances[l.code], n)
			}
			if total.Sign() != 0 {
				return fmt.Errorf("unbalanced generator journal: %s", j.source)
			}
		}
		if len(balances) != len(b.want) {
			return fmt.Errorf("oracle account mismatch")
		}
		for code, want := range b.want {
			n, ok := new(big.Rat).SetString(want)
			if !ok || balances[code] == nil || balances[code].Cmp(n) != 0 {
				return fmt.Errorf("cents oracle mismatch: %s", code)
			}
		}
	}
	return nil
}

// seedDemo uses real restricted setup/post operations; admin only supplies synthetic references and verifies stored values.
func seedDemo(ctx context.Context, admin *pgx.Conn, pool *pgxpool.Pool) error {
	books := demoBooks()
	if err := validateDemoBooks(books); err != nil {
		return err
	}
	const actor = "00000000-0000-4000-8000-000000000001"
	const usd = "00000000-0000-4000-8000-000000000002"
	const eur = "00000000-0000-4000-8000-000000000003"
	_, err := admin.Exec(ctx, `INSERT INTO reference.currency VALUES($1,'EUR','978',2,'2000-01-01',NULL,'T04-synthetic-demo');`, eur)
	if err != nil {
		return err
	}
	rates := map[string]string{"2026-01-21": "00000000-0000-4000-8000-000000000021", "2026-01-24": "00000000-0000-4000-8000-000000000024"}
	for date, id := range rates {
		_, err = admin.Exec(ctx, `INSERT INTO reference.rate_observation VALUES($1,$2,$3,1,$4::date,$4::date,'synthetic',$5,'spot','synthetic-reviewed-spot-v1',NULL)`, id, eur, usd, date, "demo-only-parity-not-market-"+date)
		if err != nil {
			return err
		}
	}
	s := &ledger.Service{DB: pool}
	for index, b := range books {
		body, _ := json.Marshal(map[string]any{"entity_name": b.name, "entity_kind": "legal", "book_name": "Synthetic review — USD functional", "starts_on": "2026-01-01", "ends_before": "2027-01-01", "functional_currency_version_id": usd, "accounts": b.accounts})
		request := fmt.Sprintf("00000000-0000-4000-8000-%012d", 100+index)
		r, err := s.Setup(ctx, ledger.Command{Actor: actor, Key: fmt.Sprintf("demo-setup-%d", index), Request: request, Body: string(body)})
		if err != nil {
			return fmt.Errorf("demo setup: %w", err)
		}
		var setup struct {
			Entity   string `json:"entity_id"`
			Book     string `json:"book_id"`
			Policy   string `json:"policy_id"`
			Revision string `json:"book_revision"`
			Accounts []struct {
				ID   string `json:"account_id"`
				Code string `json:"code"`
			} `json:"accounts"`
		}
		if err = json.Unmarshal(r.Body, &setup); err != nil {
			return err
		}
		ids := map[string]string{}
		for _, a := range setup.Accounts {
			if a.Code == "" || a.ID == "" || ids[a.Code] != "" {
				return fmt.Errorf("invalid setup account mapping")
			}
			ids[a.Code] = a.ID
		}
		for ji, j := range b.entries {
			lines := []map[string]any{}
			for _, l := range j.lines {
				line := map[string]any{"account_id": ids[l.code], "side": l.side, "transaction_currency": b.currency, "transaction_amount": l.amount, "functional_amount": l.amount, "valuation_basis": "identity"}
				if b.currency == "EUR" {
					line["valuation_basis"] = "spot"
					line["rate_observation_id"] = rates[j.date]
				}
				lines = append(lines, line)
			}
			body, _ = json.Marshal(map[string]any{"expected_revision": setup.Revision, "policy_id": setup.Policy, "accounting_date": j.date, "description": "Synthetic demo / " + j.source, "lines": lines})
			r, err = s.Post(ctx, ledger.Command{Actor: actor, Entity: setup.Entity, Book: setup.Book, Key: "demo-" + j.source, Request: fmt.Sprintf("00000000-0000-4000-8000-%012d", 200+index*20+ji), Body: string(body)})
			if err != nil {
				return fmt.Errorf("demo post %s: %w", j.source, err)
			}
			var posted struct {
				Revision string `json:"book_revision"`
			}
			if err = json.Unmarshal(r.Body, &posted); err != nil {
				return err
			}
			setup.Revision = posted.Revision
		}
		for code, want := range b.want {
			var valid bool
			err = admin.QueryRow(ctx, `SELECT coalesce(sum(CASE side WHEN 'debit' THEN transaction_amount ELSE -transaction_amount END),0)=$4::numeric AND coalesce(sum(CASE side WHEN 'debit' THEN functional_amount ELSE -functional_amount END),0)=$4::numeric FROM ledger.journal_line WHERE entity_id=$1 AND book_id=$2 AND account_id=$3`, setup.Entity, setup.Book, ids[code], want).Scan(&valid)
			if err != nil {
				return err
			}
			if !valid {
				return fmt.Errorf("stored cents oracle mismatch: %s", code)
			}
		}
	}
	return nil
}

// TestDemoCentsOracle keeps expected totals independent and proves one-cent/fractional-cent mutations fail.
func TestDemoCentsOracle(t *testing.T) {
	if err := validateDemoBooks(demoBooks()); err != nil {
		t.Fatal(err)
	}
	for _, amount := range []string{"5749.27", "5749.261"} {
		b := demoBooks()
		b[0].entries[0].lines[0].amount = amount
		if validateDemoBooks(b) == nil {
			t.Fatalf("accepted corrupted amount %s", amount)
		}
	}
	books := demoBooks()
	books[1].want["1000"] = "0.30"
	if validateDemoBooks(books) == nil {
		t.Fatal("accepted wrong independent oracle")
	}
}
