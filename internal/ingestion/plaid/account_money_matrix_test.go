package plaid_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

// TestPlaidAccountMoneyMatrix applies the same independently fixed currency expectations to every supported account pin.
func TestPlaidAccountMoneyMatrix(t *testing.T) {
	for _, account := range []struct{ kind, subtype string }{
		{"depository", "checking"}, {"credit", "credit card"},
		{"loan", "student"}, {"loan", "mortgage"}, {"loan", "auto"},
		{"loan", "consumer"}, {"loan", "line of credit"},
	} {
		t.Run(account.subtype, func(t *testing.T) {
			ex := expected()
			ex.AccountType = account.kind
			ex.AccountSubtype = account.subtype
			for _, money := range []struct{ currency, lexeme, amount, direction string }{
				{"USD", "9007199254740993.0100", "9007199254740993.01", "outflow"},
				{"USD", "-1255e-2", "12.55", "inflow"},
				{"JPY", "1e2", "100", "outflow"},
				{"JPY", "-100.0000", "100", "inflow"},
				{"KWD", "1.005000", "1.005", "outflow"},
				{"KWD", "-1.255e2", "125.5", "inflow"},
			} {
				raw := []byte(strings.Replace(string(page(row(money.lexeme, money.currency), "", "")), `"type":"depository"`, `"type":"`+account.kind+`","subtype":"`+account.subtype+`"`, 1))
				got, err := plaid.DecodePage(raw, ex)
				if err != nil || len(got.Added) != 1 {
					t.Fatal("exact account/currency quantity rejected", money.currency, err)
				}
				txn := got.Added[0]
				if txn.AmountLexeme != money.lexeme || txn.Amount != money.amount || txn.Direction != money.direction {
					t.Fatal("fixed account/currency quantity changed", money.currency)
				}
				assertQuantity(t, txn)
			}
			for _, bad := range []struct{ currency, lexeme, code string }{
				{"USD", "1e29", "amount_magnitude_exceeded"},
				{"USD", "1e-10", "amount_scale_exceeded"},
			} {
				raw := []byte(strings.Replace(string(page(row(bad.lexeme, bad.currency), "", "")), `"type":"depository"`, `"type":"`+account.kind+`","subtype":"`+account.subtype+`"`, 1))
				got, err := plaid.DecodePage(raw, ex)
				if plaid.ErrorCode(err) != bad.code || !reflect.DeepEqual(got, plaid.Page{}) {
					t.Fatal("account money boundary changed", bad.currency, plaid.ErrorCode(err))
				}
			}
			// A sub-minor-unit row must survive beside ordinary rows without gaining drafting eligibility.
			for _, held := range []struct{ currency, lexeme, amount string }{
				{"USD", "0.001", "0.001"}, {"JPY", "-0.1", "0.1"}, {"KWD", "1e-4", "0.0001"},
			} {
				raw := []byte(strings.Replace(string(page(row(held.lexeme, held.currency), "", "")), `"type":"depository"`, `"type":"`+account.kind+`","subtype":"`+account.subtype+`"`, 1))
				got, err := plaid.DecodePage(raw, ex)
				if err != nil || len(got.Added) != 1 || got.Added[0].EligibleForReview || got.Added[0].Amount != held.amount || got.Added[0].AmountLexeme != held.lexeme {
					t.Fatal("held source quantity lost or made eligible", held.currency, err)
				}
				assertQuantity(t, got.Added[0])
			}
		})
	}
}
