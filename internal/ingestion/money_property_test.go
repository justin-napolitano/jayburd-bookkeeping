package ingestion_test

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion"
)

// propertyCurrency selects only independently pinned synthetic 0/2/3-minor-unit metadata.
func propertyCurrency(choice uint8) ingestion.Expectations {
	codes := []string{"JPY", "USD", "KWD"}
	scales := []int{0, 2, 3}
	i := int(choice) % len(codes)
	return ingestion.Expectations{SourceAccountRef: "PROPERTY-ACCOUNT", StartsOn: "2026-01-01", EndsBefore: "2026-02-01", Currency: ingestion.Currency{Code: codes[i], MinorUnit: scales[i], VersionID: "00000000-0000-4000-8000-000000000123", ValidFrom: "2000-01-01"}}
}

// moneyOracle scans bytes and accumulates a base-ten integer; it uses no decoder
// regexp, canonicalizer, fixture helper or production money predicate.
func moneyOracle(raw string, minor int) (*big.Int, string, bool) {
	numerator := new(big.Int)
	ten := big.NewInt(10)
	fractionDigits, integerDigits := 0, 0
	dot := false
	for i := 0; i < len(raw); i++ {
		digit := raw[i]
		if digit == '.' && !dot {
			dot = true
			continue
		}
		if digit < '0' || digit > '9' {
			return nil, "", false
		}
		if dot {
			fractionDigits++
			if fractionDigits > 9 {
				return nil, "", false
			}
		} else {
			integerDigits++
			if integerDigits > 29 {
				return nil, "", false
			}
		}
		numerator.Mul(numerator, ten).Add(numerator, big.NewInt(int64(digit-'0')))
	}
	if integerDigits == 0 || (dot && fractionDigits == 0) || numerator.Sign() <= 0 {
		return nil, "", false
	}
	denominator := new(big.Int).Exp(ten, big.NewInt(int64(fractionDigits)), nil)
	scaled := new(big.Int).Mul(numerator, new(big.Int).Exp(ten, big.NewInt(int64(minor)), nil))
	units, remainder := new(big.Int), new(big.Int)
	units.QuoRem(scaled, denominator, remainder)
	if remainder.Sign() != 0 {
		return nil, "", false
	}
	return units, minorUnitText(units, minor), true
}

// minorUnitText renders a nonnegative integer quantity at a known scale through
// division, independently from the decoder's trimming of the original lexeme.
func minorUnitText(units *big.Int, minor int) string {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(minor)), nil)
	whole, fraction := new(big.Int), new(big.Int)
	whole.QuoRem(units, scale, fraction)
	if fraction.Sign() == 0 {
		return whole.String()
	}
	tail := fmt.Sprintf("%0*s", minor, fraction.String())
	return whole.String() + "." + strings.TrimRight(tail, "0")
}

// propertyFile quotes arbitrary candidate text so structural injection never
// turns one fuzzed amount into a different accepted amount or record.
func propertyFile(t testing.TB, e ingestion.Expectations, amounts, ids []string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("source_account_ref,entry_id,booked_on,currency,direction,amount,description\r\n")
	for i, amount := range amounts {
		// Quote without newline normalization: arbitrary candidate bytes must reach
		// Decode unchanged, including CR/control bytes expected to be rejected.
		fmt.Fprintf(&b, "%s,%s,2026-01-05,%s,outflow,\"%s\",Synthetic identical purchase\r\n", e.SourceAccountRef, ids[i], e.Currency.Code, strings.ReplaceAll(amount, "\"", "\"\""))
	}
	return b.String()
}

// checkMoneyProperty compares acceptance and exact quantity to the independent
// integer oracle, and requires every rejected input to return no usable rows.
func checkMoneyProperty(t testing.TB, raw string, choice uint8) {
	t.Helper()
	e := propertyCurrency(choice)
	units, canonical, valid := moneyOracle(raw, e.Currency.MinorUnit)
	result, err := ingestion.Decode(strings.NewReader(propertyFile(t, e, []string{raw}, []string{"PURCHASE-1"})), e)
	if !valid {
		if err == nil || !reflect.DeepEqual(result, ingestion.Statement{}) {
			t.Fatalf("invalid amount accepted or partial result: %q", raw)
		}
		return
	}
	if err != nil || len(result.Rows) != 1 {
		t.Fatalf("valid amount rejected: %q, %v", raw, err)
	}
	row := result.Rows[0]
	if row.Amount != raw || row.CanonicalAmount != canonical || row.Currency != e.Currency.Code || row.CurrencyVersionID != e.Currency.VersionID || row.EntryID != "PURCHASE-1" || row.Record != 2 {
		t.Fatalf("quantity/provenance changed: %q => %+v, expected %q", raw, row, canonical)
	}
	actual, ok := new(big.Rat).SetString(row.CanonicalAmount)
	if !ok {
		t.Fatal("canonical is not rational")
	}
	expected := new(big.Rat).SetFrac(units, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e.Currency.MinorUnit)), nil))
	original, ok := new(big.Rat).SetString(raw)
	if !ok || actual.Cmp(expected) != 0 || original.Cmp(expected) != 0 {
		t.Fatalf("exact unit drift: %q", raw)
	}
}

// TestDecoderMoneyOracle anchors the oracle itself to manually specified values,
// including lexical bounds that numerical equivalence alone cannot justify.
func TestDecoderMoneyOracle(t *testing.T) {
	for _, tc := range []struct {
		raw              string
		minor            int
		units, canonical string
		valid            bool
	}{
		{"000125.000000000", 0, "125", "125", true},
		{"125.50", 2, "12550", "125.5", true},
		{"0.01", 2, "1", "0.01", true},
		{"1.005", 3, "1005", "1.005", true},
		{"0.001", 3, "1", "0.001", true},
		{"99999999999999999999999999999.99", 2, "9999999999999999999999999999999", "99999999999999999999999999999.99", true},
		{"0.001", 2, "", "", false}, {"1.1", 0, "", "", false}, {"0", 2, "", "", false},
		{"000000000000000000000000000001", 2, "", "", false}, {"1.0000000000", 2, "", "", false},
		{".1", 2, "", "", false}, {"1.", 2, "", "", false}, {"-1", 2, "", "", false}, {"1e2", 2, "", "", false},
	} {
		t.Run(tc.raw+fmt.Sprint(tc.minor), func(t *testing.T) {
			units, canonical, valid := moneyOracle(tc.raw, tc.minor)
			if valid != tc.valid || (valid && (units.String() != tc.units || canonical != tc.canonical)) {
				t.Fatalf("oracle disagrees with fixed expectation: %q", tc.raw)
			}
		})
	}
	// Same contract boundary is exercised against all three actual decoder currencies.
	for _, raw := range []string{"0.01", "0.001", "1", "000125.000000000", "99999999999999999999999999999.999999999", "000000000000000000000000000001", "1.0000000000", "0", "-1", "+1", "1e2", "NaN", "Infinity", " 1", "1 ", "1,2", "1\x00", "１", "١", "1\n2", "1\r\n2", "1\r", "\"1\"", "1..0"} {
		for choice := uint8(0); choice < 3; choice++ {
			checkMoneyProperty(t, raw, choice)
		}
	}
}

// FuzzDecoderMoney explores arbitrary lexemes through the real CSV decoder;
// fixed seeds execute in ordinary CI, while scheduled budgets expand coverage.
func FuzzDecoderMoney(f *testing.F) {
	for _, raw := range []string{"1", "0.01", "0.001", "000125.000000000", "99999999999999999999999999999.99", "99999999999999999999999999999.999999999", "000000000000000000000000000001", "1.0000000000", "0", "-1", "+1", "1e2", "NaN", " 1", "1,2", "1\x00", "１", "1\r\n2"} {
		for choice := uint8(0); choice < 3; choice++ {
			f.Add(raw, choice)
		}
	}
	f.Fuzz(func(t *testing.T, raw string, choice uint8) {
		// The amount grammar has at most 39 bytes; larger lexical failures are covered
		// while capping each fuzz fixture, rather than benchmarking the file-size guard.
		if len(raw) > 4096 {
			return
		}
		checkMoneyProperty(t, raw, choice)
	})
}

// FuzzDecoderRepresentationIdentity generates exact positive minor units, adds
// harmless zero padding, and checks purchase IDs rather than content deduplication.
func FuzzDecoderRepresentationIdentity(f *testing.F) {
	for _, units := range []uint64{0, 1, 9, 10, 99, 100, 1005, 9007199254740993, ^uint64(0)} {
		for choice := uint8(0); choice < 3; choice++ {
			f.Add(units, choice, uint8(3), uint8(4), false)
			f.Add(units, choice, uint8(0), uint8(0), true)
		}
	}
	f.Fuzz(func(t *testing.T, seed uint64, choice, padding, tail uint8, duplicate bool) {
		e := propertyCurrency(choice)
		units := new(big.Int).SetUint64(seed)
		units.Add(units, big.NewInt(1))
		canonical := minorUnitText(units, e.Currency.MinorUnit)
		parts := strings.Split(canonical, ".")
		leading := int(padding) % (30 - len(parts[0]))
		raw := strings.Repeat("0", leading) + canonical
		decimals := 0
		if len(parts) == 2 {
			decimals = len(parts[1])
		}
		zeros := int(tail) % (10 - decimals)
		if zeros > 0 {
			if len(parts) == 1 {
				raw += "."
			}
			raw += strings.Repeat("0", zeros)
		}
		checkMoneyProperty(t, raw, choice)
		ids := []string{"PURCHASE-A", "PURCHASE-B"}
		if duplicate {
			ids[1] = ids[0]
		}
		result, err := ingestion.Decode(strings.NewReader(propertyFile(t, e, []string{raw, raw}, ids)), e)
		if duplicate {
			if err == nil || !reflect.DeepEqual(result, ingestion.Statement{}) {
				t.Fatal("duplicate identity accepted or partial rows leaked")
			}
			return
		}
		if err != nil || len(result.Rows) != 2 {
			t.Fatalf("distinct identical purchases lost: %v", err)
		}
		for i, row := range result.Rows {
			exact, ok := new(big.Rat).SetString(row.CanonicalAmount)
			if !ok {
				t.Fatal("canonical amount is not rational")
			}
			exact.Mul(exact, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e.Currency.MinorUnit)), nil)))
			if exact.Cmp(new(big.Rat).SetInt(units)) != 0 {
				t.Fatal("generated integer units changed")
			}
			if row.EntryID != ids[i] || row.Record != i+2 || row.Amount != raw || row.CanonicalAmount != canonical {
				t.Fatal("purchase identity/order/quantity changed")
			}
		}
		// Different lexemes of the same quantity retain original evidence independently.
		equivalent, err := ingestion.Decode(strings.NewReader(propertyFile(t, e, []string{raw, canonical}, ids)), e)
		if err != nil || len(equivalent.Rows) != 2 || equivalent.Rows[0].CanonicalAmount != equivalent.Rows[1].CanonicalAmount || equivalent.Rows[0].Amount != raw || equivalent.Rows[1].Amount != canonical {
			t.Fatal("equivalent quantities lost their source representation", err)
		}
	})
}
