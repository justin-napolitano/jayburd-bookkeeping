package ingestion

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

const fixtureHeader = "source_account_ref,entry_id,booked_on,currency,direction,amount,description\r\n"
const fixtureRow = "SYNTH-CASH-01,P001,2026-01-05,USD,outflow,12.50,Paper shop"

// expectations pins a synthetic currency version; no runtime lookup is involved.
func expectations() Expectations {
	return Expectations{SourceAccountRef: "SYNTH-CASH-01", StartsOn: "2026-01-01", EndsBefore: "2026-02-01", Currency: Currency{Code: "USD", VersionID: "00000000-0000-0000-0000-000000000001", MinorUnit: 2, ValidFrom: "2000-01-01"}}
}

// rejected checks atomic failure and safe diagnostics rather than parser internals.
func rejected(t *testing.T, input string, e Expectations, code string) *DecodeError {
	t.Helper()
	result, err := Decode(strings.NewReader(input), e)
	if !reflect.DeepEqual(result, Statement{}) {
		t.Fatalf("partial result on rejection: %+v", result)
	}
	var problem *DecodeError
	if !errors.As(err, &problem) {
		t.Fatalf("wanted DecodeError, got %v", err)
	}
	for _, d := range problem.Diagnostics {
		if d.Code == code {
			return problem
		}
	}
	t.Fatalf("missing %s in %+v", code, problem)
	return nil
}

// TestStatementProfile preserves independent identical purchases and exact original fields.
func TestStatementProfile(t *testing.T) {
	input := fixtureHeader + fixtureRow + "\r\nSYNTH-CASH-01,P002,2026-01-05,USD,outflow,12.50,Paper shop\r\nSYNTH-CASH-01,D001,2026-01-06,USD,inflow,00100.000,\"Owner funding, \"\"synthetic\"\"\"\r\n"
	result, err := Decode(strings.NewReader(input), expectations())
	if err != nil {
		t.Fatal(err)
	}
	// SHA-256 expected digest was calculated independently with Python hashlib.
	if result.Profile != "jayburd-synthetic-statement-csv/1" || result.ByteCount != len(input) || result.SHA256 != "3fed709df3c25dceeeb0e88e9494d32cda2ff5f5b200f92569fe7d5d05d43061" {
		t.Fatalf("bad provenance: %+v", result)
	}
	want := []Row{
		{Record: 2, SourceAccountRef: "SYNTH-CASH-01", EntryID: "P001", BookedOn: "2026-01-05", Currency: "USD", Direction: "outflow", Amount: "12.50", Description: "Paper shop", CanonicalAmount: "12.5", CurrencyVersionID: "00000000-0000-0000-0000-000000000001"},
		{Record: 3, SourceAccountRef: "SYNTH-CASH-01", EntryID: "P002", BookedOn: "2026-01-05", Currency: "USD", Direction: "outflow", Amount: "12.50", Description: "Paper shop", CanonicalAmount: "12.5", CurrencyVersionID: "00000000-0000-0000-0000-000000000001"},
		{Record: 4, SourceAccountRef: "SYNTH-CASH-01", EntryID: "D001", BookedOn: "2026-01-06", Currency: "USD", Direction: "inflow", Amount: "00100.000", Description: "Owner funding, \"synthetic\"", CanonicalAmount: "100", CurrencyVersionID: "00000000-0000-0000-0000-000000000001"},
	}
	if !reflect.DeepEqual(result.Rows, want) {
		t.Fatalf("rows = %+v", result.Rows)
	}
	withoutFinal, err := Decode(strings.NewReader(strings.TrimSuffix(input, "\r\n")), expectations())
	if err != nil || !reflect.DeepEqual(result.Rows, withoutFinal.Rows) || result.SHA256 == withoutFinal.SHA256 {
		t.Fatal("byte identity must differ while rows remain equal")
	}
}

// TestStatementMoney uses hand-selected exact values across 0/2/3 decimal currencies.
func TestStatementMoney(t *testing.T) {
	cases := []struct {
		code           string
		minor          int
		raw, canonical string
	}{
		{"JPY", 0, "125", "125"}, {"JPY", 0, "000125.000000000", "125"},
		{"USD", 2, "125.5", "125.5"}, {"USD", 2, "0.01", "0.01"}, {"USD", 2, "12.500000000", "12.5"},
		{"KWD", 3, "1.005", "1.005"}, {"KWD", 3, "0.001", "0.001"},
		{"USD", 2, "99999999999999999999999999999.99", "99999999999999999999999999999.99"},
	}
	for _, c := range cases {
		t.Run(c.code+c.raw, func(t *testing.T) {
			e := expectations()
			e.Currency.Code = c.code
			e.Currency.MinorUnit = c.minor
			input := fixtureHeader + "SYNTH-CASH-01,M1,2026-01-05," + c.code + ",inflow," + c.raw + ",Synthetic"
			got, err := Decode(strings.NewReader(input), e)
			if err != nil || got.Rows[0].CanonicalAmount != c.canonical || got.Rows[0].Amount != c.raw {
				t.Fatalf("exact amount lost: %+v %v", got, err)
			}
		})
	}
	for _, raw := range []string{"0", "000.000", "-1", "+1", "1e2", "NaN", ".01", "1.", " 1", "1 ", "12.501", "999999999999999999999999999999", "1.0000000000"} {
		t.Run("invalid"+raw, func(t *testing.T) {
			rejected(t, fixtureHeader+strings.Replace(fixtureRow, "12.50", raw, 1), expectations(), "invalid_amount")
		})
	}
	e := expectations()
	e.Currency.Code = "JPY"
	e.Currency.MinorUnit = 0
	rejected(t, fixtureHeader+strings.ReplaceAll(strings.Replace(fixtureRow, "12.50", "1.1", 1), "USD", "JPY"), e, "invalid_amount")
}

// TestStatementSyntax rejects permissive CSV variants before any usable result exists.
func TestStatementSyntax(t *testing.T) {
	for _, c := range []struct{ name, input, code string }{
		{"empty", "", "invalid_header"}, {"headeronly", fixtureHeader, "no_records"},
		{"LF", strings.ReplaceAll(fixtureHeader+fixtureRow, "\r\n", "\n"), "invalid_line_ending"},
		{"CR", fixtureHeader + fixtureRow + "\r", "invalid_line_ending"},
		{"BOM", "\xef\xbb\xbf" + fixtureHeader + fixtureRow, "invalid_encoding"},
		{"unicode", fixtureHeader + fixtureRow + "é", "invalid_encoding"},
		{"control", fixtureHeader + fixtureRow + "\t", "invalid_encoding"},
		{"blank", fixtureHeader + "\r\n" + fixtureRow, "blank_record"},
		{"extra_final_blank", fixtureHeader + fixtureRow + "\r\n\r\n", "blank_record"},
		{"extra_field", fixtureHeader + fixtureRow + ",extra", "invalid_csv"},
		{"missing_field", fixtureHeader + "SYNTH-CASH-01,P001,2026-01-05,USD,outflow,12.50", "invalid_csv"},
		{"bare_quote", fixtureHeader + fixtureRow + "\"", "invalid_csv"},
		{"unclosed", fixtureHeader + strings.Replace(fixtureRow, "Paper shop", "\"Paper shop", 1), "invalid_csv"},
		{"multiline", fixtureHeader + strings.Replace(fixtureRow, "Paper shop", "\"Paper\r\nshop\"", 1), "invalid_csv"},
		{"quoted_header", "\"source_account_ref\"" + strings.TrimPrefix(fixtureHeader, "source_account_ref") + fixtureRow, "invalid_header"},
	} {
		t.Run(c.name, func(t *testing.T) { rejected(t, c.input, expectations(), c.code) })
	}
}

// TestStatementFields covers cross-account, immutable identity and calendar boundaries.
func TestStatementFields(t *testing.T) {
	for _, c := range []struct{ old, new, code string }{
		{"SYNTH-CASH-01", "OTHER-ACCOUNT", "account_mismatch"}, {"P001", "", "invalid_entry_id"},
		{"P001", strings.Repeat("A", 65), "invalid_entry_id"}, {"P001", "P 001", "invalid_entry_id"},
		{"2026-01-05", "2026-02-29", "invalid_date"}, {"2026-01-05", "0000-01-05", "invalid_date"},
		{"2026-01-05", "2026-1-05", "invalid_date"}, {"2026-01-05", "2026-02-01", "outside_interval"},
		{"2026-01-05", "2025-12-31", "outside_interval"}, {"USD", "EUR", "currency_mismatch"},
		{"outflow", "debit", "invalid_direction"}, {"Paper shop", "", "invalid_description"},
		{"Paper shop", strings.Repeat("x", 501), "invalid_description"},
	} {
		t.Run(c.code+c.new, func(t *testing.T) {
			rejected(t, fixtureHeader+strings.Replace(fixtureRow, c.old, c.new, 1), expectations(), c.code)
		})
	}
	rejected(t, fixtureHeader+fixtureRow+"\r\n"+fixtureRow, expectations(), "duplicate_entry_id")
	e := expectations()
	e.Currency.ValidTo = "2026-01-05"
	rejected(t, fixtureHeader+fixtureRow, e, "currency_not_valid")
	e.Currency.ValidTo = ""
	e.Currency.ValidFrom = "2026-01-06"
	rejected(t, fixtureHeader+fixtureRow, e, "currency_not_valid")
	for _, date := range []string{"2026-01-01", "2026-01-31"} {
		if _, err := Decode(strings.NewReader(fixtureHeader+strings.Replace(fixtureRow, "2026-01-05", date, 1)), expectations()); err != nil {
			t.Fatal(err)
		}
	}
	e = expectations()
	e.StartsOn = "2024-02-01"
	e.EndsBefore = "2024-03-01"
	if _, err := Decode(strings.NewReader(fixtureHeader+strings.Replace(fixtureRow, "2026-01-05", "2024-02-29", 1)), e); err != nil {
		t.Fatal(err)
	}
}

// TestStatementExpectations rejects invalid caller configuration before reading bytes.
func TestStatementExpectations(t *testing.T) {
	mutations := []func(*Expectations){
		func(e *Expectations) { e.SourceAccountRef = "bad account" }, func(e *Expectations) { e.StartsOn = e.EndsBefore },
		func(e *Expectations) { e.StartsOn = "2026-99-01" }, func(e *Expectations) { e.Currency.Code = "usd" },
		func(e *Expectations) { e.Currency.VersionID = "not-uuid" }, func(e *Expectations) { e.Currency.VersionID = "00000000-0000-0000-0000-000000000000" },
		func(e *Expectations) { e.Currency.MinorUnit = -1 }, func(e *Expectations) { e.Currency.MinorUnit = 10 },
		func(e *Expectations) { e.Currency.ValidTo = e.Currency.ValidFrom }, func(e *Expectations) { e.Currency.ValidFrom = "invalid" },
	}
	for i, mutate := range mutations {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			e := expectations()
			mutate(&e)
			reader := &countingReader{Reader: strings.NewReader(fixtureHeader + fixtureRow)}
			_, err := Decode(reader, e)
			if err == nil || reader.count != 0 {
				t.Fatalf("expectations accepted or read input: %v %d", err, reader.count)
			}
		})
	}
}

// countingReader records actual consumption to prove the byte sentinel bound.
type countingReader struct {
	io.Reader
	count int
}

// Read forwards bounded reads while accumulating bytes, never peeking at the rest.
func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.count += n
	return n, err
}

// errorReader provides synthetic sensitive text to exercise error redaction.
type errorReader struct{}

// Read fails without making its arbitrary error safe for exposure.
func (errorReader) Read([]byte) (int, error) { return 0, errors.New("private/path/secret-value") }

// TestStatementBounds exercises independent boundary values and atomic error caps.
func TestStatementBounds(t *testing.T) {
	reader := &countingReader{Reader: strings.NewReader(strings.Repeat("x", (1<<20)+200))}
	_, err := Decode(reader, expectations())
	var problem *DecodeError
	if !errors.As(err, &problem) || problem.Diagnostics[0].Code != "file_too_large" || reader.count != (1<<20)+1 {
		t.Fatalf("unbounded read: %d %v", reader.count, err)
	}
	// Exactly MaxBytes is not an oversize error, even when its CSV is invalid.
	rejected(t, strings.Repeat("x", 1<<20), expectations(), "invalid_header")
	for _, size := range []int{2048, 2049} {
		line := fixtureRow + strings.Repeat("x", size-len(fixtureRow))
		code := "invalid_description"
		if size == 2049 {
			code = "record_too_large"
		}
		rejected(t, fixtureHeader+line, expectations(), code)
	}
	var input strings.Builder
	input.WriteString(fixtureHeader)
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&input, "SYNTH-CASH-01,P%d,2026-01-05,USD,inflow,0.01,Synthetic\r\n", i)
	}
	got, err := Decode(strings.NewReader(input.String()), expectations())
	if err != nil || len(got.Rows) != 1000 {
		t.Fatalf("1000 rows: %v", err)
	}
	rejected(t, input.String()+fixtureRow, expectations(), "too_many_records")
	input.Reset()
	input.WriteString(fixtureHeader)
	for i := 0; i < 101; i++ {
		fmt.Fprintf(&input, "SYNTH-CASH-01,P%d,2026-01-05,USD,inflow,0,Synthetic\r\n", i)
	}
	problem = rejected(t, input.String(), expectations(), "invalid_amount")
	if problem.Total != 101 || len(problem.Diagnostics) != 100 || !problem.Truncated || problem.Diagnostics[0].Record != 2 {
		t.Fatalf("wrong diagnostics cap: %+v", problem)
	}
	// Accepted decoded field maxima differ from the larger structural record cap.
	e := expectations()
	e.SourceAccountRef = strings.Repeat("A", 64)
	valid := fixtureHeader + e.SourceAccountRef + "," + strings.Repeat("B", 64) + ",2026-01-05,USD,inflow,1," + strings.Repeat("x", 500)
	if _, err := Decode(strings.NewReader(valid), e); err != nil {
		t.Fatal(err)
	}
}

// TestStatementIOFailure rejects a valid prefix if reading the remainder fails.
func TestStatementIOFailure(t *testing.T) {
	for _, reader := range []io.Reader{nil, errorReader{}, io.MultiReader(strings.NewReader(fixtureHeader+fixtureRow), errorReader{})} {
		result, err := Decode(reader, expectations())
		if !reflect.DeepEqual(result, Statement{}) || err == nil || err.Error() != "synthetic statement rejected" {
			t.Fatalf("unsafe I/O outcome: %+v %v", result, err)
		}
		var problem *DecodeError
		if !errors.As(err, &problem) || problem.Diagnostics[0].Code != "read_failed" {
			t.Fatal("missing read failure")
		}
	}
}
