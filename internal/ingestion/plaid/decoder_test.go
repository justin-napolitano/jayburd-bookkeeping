package plaid_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

// Fixed installed metadata is independent of decoder code and provider examples.
func expected() plaid.Expected {
	return plaid.Expected{AccountID: "Account-A", Currencies: []plaid.Currency{
		{Code: "USD", VersionID: "00000000-0000-4000-8000-000000000001", MinorUnit: 2, ValidFrom: "1900-01-01"},
		{Code: "JPY", VersionID: "00000000-0000-4000-8000-000000000002", MinorUnit: 0, ValidFrom: "1900-01-01"},
		{Code: "KWD", VersionID: "00000000-0000-4000-8000-000000000003", MinorUnit: 3, ValidFrom: "1900-01-01"},
	}}
}
func row(amount, currency string) string {
	return fmt.Sprintf(`{"account_id":"Account-A","transaction_id":"Txn-A","amount":%s,"iso_currency_code":%q,"unofficial_currency_code":null,"date":"2026-01-02","pending":false,"pending_transaction_id":null,"name":"Synthetic supplies"}`, amount, currency)
}
func page(added, modified, removed string) []byte {
	return []byte(fmt.Sprintf(`{"accounts":[{"account_id":"Account-A","type":"depository"}],"added":[%s],"modified":[%s],"removed":[%s],"next_cursor":"Y3Vyc29y","has_more":false,"request_id":"synthetic-request","transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`, added, modified, removed))
}

// rejected verifies both fixed redaction and atomic removal of every potential publication field.
func rejected(t *testing.T, raw []byte, ex plaid.Expected) string {
	t.Helper()
	value, err := plaid.DecodePage(raw, ex)
	if err == nil {
		t.Fatal("accepted invalid page")
	}
	if !reflect.DeepEqual(value, plaid.Page{}) {
		t.Fatal("partial evidence escaped failure")
	}
	safe, ok := err.(*plaid.Error)
	if !ok || safe.Code == "" || strings.Contains(safe.Code, "Account") || strings.Contains(safe.Code, "Synthetic") {
		t.Fatal("unsafe failure")
	}
	return safe.Code
}

func TestPlaidExactMoney(t *testing.T) {
	for _, tc := range []struct {
		lexeme, currency, amount, direction string
		ready                               bool
	}{
		{"12.50", "USD", "12.5", "outflow", true}, {"-1.255e2", "USD", "125.5", "inflow", true}, {"125", "JPY", "125", "outflow", true}, {"1.005", "KWD", "1.005", "outflow", true},
		{"9007199254740993.01", "USD", "9007199254740993.01", "outflow", true}, {"99999999999999999999999999999.99", "USD", "99999999999999999999999999999.99", "outflow", true},
		{"0", "USD", "0", "zero", false}, {"-0.00e100", "USD", "0", "zero", false}, {"0e-100", "JPY", "0", "zero", false},
		{"1.000000000000000000000000000000000000000000000000000000", "USD", "1", "outflow", true}, {"100000000000000000000000000000000000000000000000000000e-53", "USD", "1", "outflow", true},
	} {
		t.Run(tc.lexeme, func(t *testing.T) {
			raw := page(row(tc.lexeme, tc.currency), "", "")
			p, err := plaid.DecodePage(raw, expected())
			if err != nil {
				t.Fatal(err)
			}
			got := p.Added[0]
			if got.AmountLexeme != tc.lexeme || got.Amount != tc.amount || got.Direction != tc.direction || got.EligibleForReview != tc.ready {
				t.Fatalf("money mismatch %+v", got)
			}
			source, ok := new(big.Rat).SetString(tc.lexeme)
			if !ok {
				t.Fatal("oracle fixture")
			}
			quantity, _ := new(big.Rat).SetString(got.Amount)
			if got.Direction == "inflow" {
				quantity.Neg(quantity)
			}
			if source.Cmp(quantity) != 0 {
				t.Fatal("independent rational quantity mismatch")
			}
		})
	}
	for _, bad := range []struct{ amount, currency string }{{"0.001", "USD"}, {"1.1", "JPY"}, {"1.0001", "KWD"}, {"1e29", "USD"}, {"1e-10", "USD"}, {"1e101", "USD"}, {"0e-101", "USD"}, {"1e999999999999999999", "USD"}, {"NaN", "USD"}, {"+1", "USD"}, {"01", "USD"}, {`"12.50"`, "USD"}, {"null", "USD"}, {strings.Repeat("1", 129), "USD"}} {
		t.Run("reject_"+bad.amount, func(t *testing.T) { rejected(t, page(row(bad.amount, bad.currency), "", ""), expected()) })
	}
}
func TestPlaidIdentityAndStates(t *testing.T) {
	pending := strings.Replace(row("10", "USD"), `"pending":false`, `"pending":true`, 1)
	posted := strings.Replace(row("12", "USD"), `"pending_transaction_id":null`, `"pending_transaction_id":"Pending-A"`, 1)
	first, err := plaid.DecodePage(page(pending, "", `{"account_id":"Account-A","transaction_id":"Pending-A"}`), expected())
	if err != nil {
		t.Fatal(err)
	}
	if first.Added[0].EligibleForReview || !first.Added[0].Pending || first.Removed[0].TransactionID != "Pending-A" {
		t.Fatal("pending/tombstone mapping")
	}
	second, err := plaid.DecodePage(page(posted, "", ""), expected())
	if err != nil || !second.Added[0].EligibleForReview || *second.Added[0].PendingTransactionID != "Pending-A" {
		t.Fatal("explicit lineage mapping")
	}
	distinct := strings.Replace(row("12", "USD"), "Txn-A", "txn-a", 1)
	p, err := plaid.DecodePage(page(row("12", "USD")+","+distinct, "", ""), expected())
	if err != nil || len(p.Added) != 2 || p.Added[1].TransactionID != "txn-a" {
		t.Fatal("case-sensitive separate purchases collapsed")
	}
	for _, raw := range [][]byte{page(row("12", "USD"), row("12", "USD"), ""), page(row("12", "USD"), "", `{"account_id":"Account-A","transaction_id":"Txn-A"}`), page(strings.Replace(posted, "Pending-A", "Txn-A", 1), "", ""), page(strings.Replace(posted, `"pending":false`, `"pending":true`, 1), "", "")} {
		rejected(t, raw, expected())
	}
}
func TestPlaidMaterialSchema(t *testing.T) {
	good := string(page(row("12", "USD"), "", ""))
	cases := map[string]string{
		"top unknown":            strings.Replace(good, `"accounts":`, `"future_money":5,"accounts":`, 1),
		"row unknown":            strings.Replace(good, `"amount":12`, `"unreviewed_balance":20,"amount":12`, 1),
		"account mismatch":       strings.Replace(good, `"account_id":"Account-A"`, `"account_id":"Other"`, 1),
		"row account mismatch":   string(page(strings.Replace(row("12", "USD"), "Account-A", "Other", 1), "", "")),
		"removed mismatch":       string(page("", "", `{"account_id":"Other","transaction_id":"Removed"}`)),
		"credit account":         strings.Replace(good, `"depository"`, `"credit"`, 1),
		"null boolean":           strings.Replace(good, `"pending":false`, `"pending":null`, 1),
		"null array":             strings.Replace(good, `"modified":[]`, `"modified":null`, 1),
		"missing array":          strings.Replace(good, `"modified":[],`, "", 1),
		"missing linkage":        strings.Replace(good, `"pending_transaction_id":null,`, "", 1),
		"unofficial":             strings.Replace(good, `"unofficial_currency_code":null`, `"unofficial_currency_code":"BTC"`, 1),
		"missing ISO":            strings.Replace(good, `"iso_currency_code":"USD"`, `"iso_currency_code":null`, 1),
		"unknown currency":       strings.Replace(good, `"iso_currency_code":"USD"`, `"iso_currency_code":"ZZZ"`, 1),
		"invalid date":           strings.Replace(good, "2026-01-02", "2026-02-29", 1),
		"year zero":              strings.Replace(good, "2026-01-02", "0000-01-02", 1),
		"unknown enum":           strings.Replace(good, "HISTORICAL_UPDATE_COMPLETE", "SOMETHING_NEW", 1),
		"payment enum":           strings.Replace(good, `"name":`, `"payment_channel":"future","name":`, 1),
		"bad timestamp":          strings.Replace(good, `"name":`, `"datetime":"2026-01-02","name":`, 1),
		"offset hour":            strings.Replace(good, `"name":`, `"datetime":"2026-01-02T10:00:00+24:00","name":`, 1),
		"offset minute":          strings.Replace(good, `"name":`, `"datetime":"2026-01-02T10:00:00+01:60","name":`, 1),
		"long description":       strings.Replace(good, "Synthetic supplies", strings.Repeat("x", 501), 1),
		"empty advancing cursor": strings.Replace(good, "Y3Vyc29y", "", 1),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) { rejected(t, []byte(raw), expected()) })
	}
}
func TestPlaidLosslessAndDates(t *testing.T) {
	r := strings.Replace(row("-125.500e-1", "USD"), `"name":`, `"authorized_date":"2025-12-31","datetime":"2026-01-02T00:01:02.123456789+05:30","authorized_datetime":"2025-12-31T23:45:00Z","merchant_name":null,"location":{"new_inert_field":1.25},"name":`, 1)
	raw := append([]byte(" \n"), page(r, "", "")...)
	copyRaw := bytes.Clone(raw)
	p, err := plaid.DecodePage(raw, expected())
	if err != nil {
		t.Fatal(err)
	}
	got := p.Added[0]
	if got.Amount != "12.55" || *got.AuthorizedDate != "2025-12-31" || *got.Datetime != "2026-01-02T00:01:02.123456789+05:30" || *got.AuthorizedDatetime != "2025-12-31T23:45:00Z" {
		t.Fatal("source units or dates changed")
	}
	sum := sha256.Sum256(copyRaw)
	if p.SHA256 != hex.EncodeToString(sum[:]) || !bytes.Equal(p.Raw, copyRaw) || string(got.Raw) != r {
		t.Fatal("raw provenance changed")
	}
	raw[0] = '!'
	if !bytes.Equal(p.Raw, copyRaw) {
		t.Fatal("retained caller alias")
	}
	wire, err := json.Marshal(p)
	if err != nil || bytes.Contains(wire, []byte("location")) {
		t.Fatal("private raw included in normalized wire")
	}
}
func TestPlaidStrictJSON(t *testing.T) {
	good := string(page(row("12", "USD"), "", ""))
	for name, raw := range map[string][]byte{
		"duplicate":         []byte(strings.Replace(good, `"amount":12`, `"amount":12,"amount":13`, 1)),
		"escaped duplicate": []byte(strings.Replace(good, `"amount":12`, `"amount":12,"\u0061mount":13`, 1)),
		"nested duplicate":  []byte(strings.Replace(good, `"name":`, `"location":{"x":1,"x":2},"name":`, 1)),
		"trailing":          []byte(good + " {}"), "garbage": []byte(good + "x"),
		"surrogate":      []byte(strings.Replace(good, "Synthetic supplies", `\ud800`, 1)),
		"low surrogate":  []byte(strings.Replace(good, "Synthetic supplies", `\udfff`, 1)),
		"null char":      []byte(strings.Replace(good, "Synthetic supplies", `\u0000`, 1)),
		"invalid utf8":   bytes.Replace([]byte(good), []byte("Synthetic supplies"), []byte{255}, 1),
		"deep":           []byte(strings.Replace(good, `"name":`, `"location":`+strings.Repeat("[", 33)+"0"+strings.Repeat("]", 33)+`,"name":`, 1)),
		"inert exponent": []byte(strings.Replace(good, `"name":`, `"location":{"lat":1e999},"name":`, 1)),
	} {
		t.Run(name, func(t *testing.T) { rejected(t, raw, expected()) })
	}
	p, err := plaid.DecodePage([]byte(strings.Replace(good, "Synthetic supplies", `Caf\u00e9 \ud83d\ude00`, 1)), expected())
	if err != nil || p.Added[0].Name != "Café 😀" {
		t.Fatal("valid unicode lost")
	}
}
func TestPlaidBoundsAndReferences(t *testing.T) {
	rows := make([]string, 500)
	for i := range rows {
		rows[i] = strings.Replace(row("1", "USD"), "Txn-A", fmt.Sprintf("Txn-%d", i), 1)
	}
	if p, err := plaid.DecodePage(page(strings.Join(rows, ","), "", ""), expected()); err != nil || len(p.Added) != 500 {
		t.Fatal("500-row boundary", err)
	}
	rejected(t, page(strings.Join(rows, ",")+","+row("2", "USD"), "", ""), expected())
	raw := page(row("12", "USD"), "", "")
	limit := append(bytes.Clone(raw), bytes.Repeat([]byte(" "), plaid.MaxBytes-len(raw))...)
	if _, err := plaid.DecodePage(limit, expected()); err != nil {
		t.Fatal("byte boundary", err)
	}
	rejected(t, append(limit, ' '), expected())
	ex := expected()
	ex.Currencies[0].ValidTo = "2026-01-02"
	rejected(t, raw, ex)
	ex = expected()
	extra := ex.Currencies[0]
	extra.VersionID = "00000000-0000-4000-8000-000000000004"
	ex.Currencies = append(ex.Currencies, extra)
	rejected(t, raw, ex)
	ex = expected()
	ex.Currencies[0].MinorUnit = -1
	rejected(t, raw, ex)
}
func TestPlaidEmptyInitialization(t *testing.T) {
	raw := string(page("", "", ""))
	raw = strings.Replace(raw, "Y3Vyc29y", "", 1)
	raw = strings.Replace(raw, "HISTORICAL_UPDATE_COMPLETE", "NOT_READY", 1)
	p, err := plaid.DecodePage([]byte(raw), expected())
	if err != nil || p.UpdateStatus != "NOT_READY" || p.HasMore || p.NextCursor != "" || len(p.Added) != 0 {
		t.Fatal("empty initialization changed")
	}
	rejected(t, []byte(strings.Replace(raw, `"has_more":false`, `"has_more":true`, 1)), expected())
}

// FuzzPlaidPage checks atomic failure and stable source bytes over arbitrary bounded JSON.
func FuzzPlaidPage(f *testing.F) {
	for _, seed := range [][]byte{page(row("12.50", "USD"), "", ""), page("", row("-1.005", "KWD"), ""), []byte(`{"amount":1,"amount":2}`), []byte(`{"x":"\ud800"}`)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > plaid.MaxBytes+1 {
			return
		}
		p, err := plaid.DecodePage(raw, expected())
		if err != nil {
			if !reflect.DeepEqual(p, plaid.Page{}) {
				t.Fatal("partial failure")
			}
			return
		}
		if !bytes.Equal(p.Raw, raw) {
			t.Fatal("raw mutation")
		}
		for _, rows := range [][]plaid.Transaction{p.Added, p.Modified} {
			for _, r := range rows {
				if r.AccountID != "Account-A" {
					t.Fatal("account mismatch")
				}
				assertQuantity(t, r)
			}
		}
	})
}

// FuzzPlaidMoney uses math/big's independently implemented rational parser as quantity oracle.
func FuzzPlaidMoney(f *testing.F) {
	for _, s := range []string{"12.50", "-1.255e2", "0", "-0e100", "1e-100", "1e100", "9007199254740993.01", "1.005", "0.001", "1.000000000000000000000000000000000000000000000000", "NaN"} {
		f.Add(s, uint8(0))
	}
	f.Fuzz(func(t *testing.T, lex string, index uint8) {
		if len(lex) > 160 {
			return
		}
		code := []string{"USD", "JPY", "KWD"}[int(index)%3]
		p, err := plaid.DecodePage(page(row(lex, code), "", ""), expected())
		if err != nil {
			if !reflect.DeepEqual(p, plaid.Page{}) {
				t.Fatal("partial failure")
			}
			return
		}
		r := p.Added[0]
		assertQuantity(t, r)
		units := map[string]int64{"USD": 100, "JPY": 1, "KWD": 1000}
		n, _ := new(big.Rat).SetString(r.Amount)
		n.Mul(n, new(big.Rat).SetInt64(units[code]))
		if !n.IsInt() {
			t.Fatal("fractional minor unit")
		}
	})
}
func assertQuantity(t *testing.T, r plaid.Transaction) {
	t.Helper()
	source, ok := new(big.Rat).SetString(r.AmountLexeme)
	if !ok {
		t.Fatal("invalid original number")
	}
	result, ok := new(big.Rat).SetString(r.Amount)
	if !ok || result.Sign() < 0 {
		t.Fatal("invalid magnitude")
	}
	switch r.Direction {
	case "inflow":
		result.Neg(result)
		if source.Sign() >= 0 {
			t.Fatal("inflow sign")
		}
	case "outflow":
		if source.Sign() <= 0 {
			t.Fatal("outflow sign")
		}
	case "zero":
		if source.Sign() != 0 {
			t.Fatal("zero sign")
		}
	default:
		t.Fatal("unknown direction")
	}
	if result.Cmp(source) != 0 {
		t.Fatal("exact quantity changed")
	}
	if r.EligibleForReview != (source.Sign() != 0 && !r.Pending) {
		t.Fatal("source readiness changed")
	}
}

// TestPlaidTransactionCode accepts documented inert classification without changing money or scope.
func TestPlaidTransactionCode(t *testing.T) {
	for _, value := range []string{`null`, `"purchase"`, `"future_classification"`} {
		raw := page(strings.Replace(row("9007199254740993.01", "USD"), `"name":`, `"transaction_code":`+value+`,"name":`, 1), "", "")
		p, err := plaid.DecodePage(raw, expected())
		if err != nil {
			t.Fatal(err)
		}
		if p.Added[0].Amount != "9007199254740993.01" || p.Added[0].Direction != "outflow" {
			t.Fatal("classification changed money")
		}
		if !bytes.Contains(p.Added[0].Raw, []byte(`"transaction_code":`+value)) {
			t.Fatal("source evidence lost")
		}
		wire, _ := json.Marshal(p)
		if bytes.Contains(wire, []byte("transaction_code")) {
			t.Fatal("inert classification mapped")
		}
		rejected(t, bytes.Replace(raw, []byte("Account-A"), []byte("Other-A"), 1), expected())
		rejected(t, bytes.Replace(raw, []byte("9007199254740993.01"), []byte("1.001"), 1), expected())
	}
	for _, value := range []string{`42`, `{}`, `true`, `"` + strings.Repeat("x", 257) + `"`} {
		rejected(t, page(strings.Replace(row("1", "USD"), `"name":`, `"transaction_code":`+value+`,"name":`, 1), "", ""), expected())
	}
}

// TestPlaidEmptyAccountReadiness separates Item update status from account cursor availability.
func TestPlaidEmptyAccountReadiness(t *testing.T) {
	for _, status := range []string{"NOT_READY", "TRANSACTIONS_UPDATE_STATUS_UNKNOWN", "INITIAL_UPDATE_COMPLETE", "HISTORICAL_UPDATE_COMPLETE"} {
		raw := strings.Replace(string(page("", "", "")), "Y3Vyc29y", "", 1)
		raw = strings.Replace(raw, "HISTORICAL_UPDATE_COMPLETE", status, 1)
		got, err := plaid.DecodePage([]byte(raw), expected())
		if err != nil || got.NextCursor != "" || got.UpdateStatus != status {
			t.Fatal("empty account readiness rejected", err)
		}
		rejected(t, []byte(strings.Replace(raw, `"has_more":false`, `"has_more":true`, 1)), expected())
		for _, group := range []string{"added", "modified", "removed"} {
			change := row("1", "USD")
			if group == "removed" {
				change = `{"account_id":"Account-A","transaction_id":"Removed-A"}`
			}
			rejected(t, []byte(strings.Replace(raw, `"`+group+`":[]`, `"`+group+`":[`+change+`]`, 1)), expected())
		}
	}
}

// TestPlaidLiabilityAccountPin requires the returned account to match trusted type and subtype.
func TestPlaidLiabilityAccountPin(t *testing.T) {
	for _, tc := range []struct{ kind, subtype string }{{"credit", "credit card"}, {"loan", "student"}, {"loan", "mortgage"}} {
		e := expected()
		e.AccountType, e.AccountSubtype = tc.kind, tc.subtype
		raw := []byte(strings.Replace(string(page(row("1", "USD"), "", "")), `"type":"depository"`, `"type":"`+tc.kind+`","subtype":"`+tc.subtype+`"`, 1))
		if _, err := plaid.DecodePage(raw, e); err != nil {
			t.Fatal(tc, err)
		}
		rejected(t, []byte(strings.Replace(string(raw), tc.subtype, "other", 1)), e)
		rejected(t, page(row("1", "USD"), "", ""), e)
	}
}

// TestPlaidReviewedEnrichment keeps every reviewed field inert and preserves exact provider balance numbers.
func TestPlaidReviewedEnrichment(t *testing.T) {
	for _, metadata := range []string{
		`"business_finance_category":null,"client_customization":null,"running_balance":null,`,
		`"business_finance_category":null,"client_customization":null,"running_balance":9007199254740993.0100,`,
		`"business_finance_category":null,"client_customization":null,"running_balance":-1.255e2,`,
		`"business_finance_category":{"primary":"SYNTHETIC","detailed":"SYNTHETIC_SUPPLIES","confidence_level":"UNKNOWN"},"client_customization":{"custom_entity_id":"fictional-merchant"},"running_balance":null,`,
	} {
		raw := page(strings.Replace(row("9007199254740993.01", "USD"), `"name":`, metadata+`"name":`, 1), "", "")
		p, err := plaid.DecodePage(raw, expected())
		if err != nil || len(p.Added) != 1 {
			t.Fatal("documented enrichment rejected", err)
		}
		if p.Added[0].Amount != "9007199254740993.01" || p.Added[0].Direction != "outflow" {
			t.Fatal("enrichment changed money")
		}
		if !bytes.Contains(p.Added[0].Raw, []byte(metadata)) {
			t.Fatal("provider balance or enrichment bytes changed")
		}
		wire, _ := json.Marshal(p)
		for _, key := range []string{"business_finance_category", "client_customization", "running_balance"} {
			if bytes.Contains(wire, []byte(key)) || !bytes.Contains(p.Added[0].Raw, []byte(key)) {
				t.Fatal("inert mapping or raw retention changed")
			}
		}
		rejected(t, bytes.Replace(raw, []byte("Account-A"), []byte("Other-A"), 1), expected())
	}
	for _, fragment := range []string{`"running_balance":"12.34",`, `"running_balance":{},`, `"running_balance":true,`, `"client_customization":[],`, `"business_finance_category":true,`, `"new_material_field":null,`} {
		rejected(t, page(strings.Replace(row("1", "USD"), `"name":`, fragment+`"name":`, 1), "", ""), expected())
	}
}

// TestPlaidFullPublicSchemaFixture exercises every direct field in pinned PI11 using fictional data.
func TestPlaidFullPublicSchemaFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/complete-public-schema.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := plaid.DecodePage(raw, expected())
	if err != nil || len(p.Added) != 1 || len(p.Removed) != 1 {
		t.Fatal("complete public schema rejected", err)
	}
	if p.Added[0].Amount != "42" || !bytes.Contains(p.Added[0].Raw, []byte("9007199254740993")) {
		t.Fatal("source quantity changed")
	}
}
