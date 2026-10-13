// Package ingestion decodes bounded synthetic source evidence without granting
// authority, persisting records, or performing accounting mutations.
package ingestion

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"io"
	"regexp"
	"strings"
	"time"
)

// Profile and limits identify the accepted synthetic template, not a bank format.
const (
	Profile        = "jayburd-synthetic-statement-csv/1"
	MaxBytes       = 1 << 20
	MaxRecords     = 1000
	MaxRecordBytes = 2048
	MaxDiagnostics = 100
	header         = "source_account_ref,entry_id,booked_on,currency,direction,amount,description"
)

var (
	referencePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	currencyPattern  = regexp.MustCompile(`^[A-Z]{3}$`)
	uuidPattern      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	amountPattern    = regexp.MustCompile(`^[0-9]{1,29}(\.[0-9]{1,9})?$`)
)

// Currency supplies caller-reviewed reference metadata; Decode never resolves it.
type Currency struct {
	Code      string
	VersionID string
	MinorUnit int
	ValidFrom string
	ValidTo   string // Empty means no exclusive upper validity date.
}

// Expectations describes equality and date checks, not an authorized binding.
type Expectations struct {
	SourceAccountRef string
	StartsOn         string
	EndsBefore       string
	Currency         Currency
}

// Row retains decoded source values and exact-money provenance in input order.
type Row struct {
	Record            int
	SourceAccountRef  string
	EntryID           string
	BookedOn          string
	Currency          string
	Direction         string
	Amount            string // Original decimal lexeme after CSV unquoting.
	Description       string
	CanonicalAmount   string
	CurrencyVersionID string
}

// Statement is returned only when every bounded record passes validation.
type Statement struct {
	Profile   string
	SHA256    string
	ByteCount int
	Rows      []Row
}

// Diagnostic contains only fixed codes/field names and a physical record ordinal.
type Diagnostic struct {
	Code   string
	Record int // Header is 1; file/expectation failures use 0.
	Field  string
}

// DecodeError bounds displayed errors; Total counts detected errors, not unparsed rows.
type DecodeError struct {
	Diagnostics []Diagnostic
	Total       int
	Truncated   bool
}

// Error deliberately excludes raw inputs and underlying I/O error messages.
func (e *DecodeError) Error() string { return "synthetic statement rejected" }

// add retains a bounded diagnostic prefix while counting detected errors.
func (e *DecodeError) add(code string, record int, field string) {
	e.Total++
	if len(e.Diagnostics) < MaxDiagnostics {
		e.Diagnostics = append(e.Diagnostics, Diagnostic{Code: code, Record: record, Field: field})
	} else {
		e.Truncated = true
	}
}

// failure creates an atomic rejection without exposing partially decoded values.
func failure(code string, record int, field string) (Statement, error) {
	e := &DecodeError{}
	e.add(code, record, field)
	return Statement{}, e
}

// Decode reads at most MaxBytes+1 bytes and validates the complete synthetic file.
// The caller owns Reader deadlines/cancellation and authority/metadata provenance.
func Decode(reader io.Reader, expected Expectations) (Statement, error) {
	if !validExpectations(expected) {
		return failure("invalid_expectations", 0, "")
	}
	if reader == nil {
		return failure("read_failed", 0, "")
	}
	raw, err := io.ReadAll(io.LimitReader(reader, MaxBytes+1))
	if err != nil {
		return failure("read_failed", 0, "")
	}
	if len(raw) > MaxBytes {
		return failure("file_too_large", 0, "")
	}
	// Validate bytes before csv.Reader can normalize CRLF or ignore blank lines.
	record := 1
	for i := 0; i < len(raw); i++ {
		b := raw[i]
		if b == '\r' {
			if i+1 >= len(raw) || raw[i+1] != '\n' {
				return failure("invalid_line_ending", record, "")
			}
			i++
			record++
		} else if b == '\n' {
			return failure("invalid_line_ending", record, "")
		} else if b < 0x20 || b > 0x7e {
			return failure("invalid_encoding", record, "")
		}
	}
	lines := bytes.Split(raw, []byte("\r\n"))
	if len(lines) > 1 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 || string(lines[0]) != header {
		return failure("invalid_header", 1, "")
	}
	if len(lines) == 1 {
		return failure("no_records", 0, "")
	}
	if len(lines)-1 > MaxRecords {
		return failure("too_many_records", 0, "")
	}
	rows := make([]Row, 0, len(lines)-1)
	seen := make(map[string]struct{}, len(lines)-1)
	problems := &DecodeError{}
	for i, line := range lines {
		ordinal := i + 1
		if len(line) > MaxRecordBytes {
			return failure("record_too_large", ordinal, "")
		}
		if len(line) == 0 {
			return failure("blank_record", ordinal, "")
		}
		if i == 0 {
			continue
		}
		parser := csv.NewReader(bytes.NewReader(line))
		parser.FieldsPerRecord = 7
		fields, err := parser.Read()
		if err != nil {
			return failure("invalid_csv", ordinal, "")
		}
		row := Row{Record: ordinal, SourceAccountRef: fields[0], EntryID: fields[1], BookedOn: fields[2], Currency: fields[3], Direction: fields[4], Amount: fields[5], Description: fields[6], CurrencyVersionID: expected.Currency.VersionID}
		validateRow(&row, expected, seen, problems)
		rows = append(rows, row)
	}
	if problems.Total != 0 {
		return Statement{}, problems
	}
	digest := sha256.Sum256(raw)
	return Statement{Profile: Profile, SHA256: hex.EncodeToString(digest[:]), ByteCount: len(raw), Rows: rows}, nil
}

// validDate rejects year zero and noncanonical or impossible Gregorian dates.
func validDate(value string) bool {
	if len(value) != 10 || value[:4] == "0000" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

// validExpectations checks configuration shape without claiming reference authenticity.
func validExpectations(e Expectations) bool {
	c := e.Currency
	return referencePattern.MatchString(e.SourceAccountRef) &&
		validDate(e.StartsOn) && validDate(e.EndsBefore) && e.StartsOn < e.EndsBefore &&
		currencyPattern.MatchString(c.Code) && uuidPattern.MatchString(c.VersionID) &&
		c.VersionID != "00000000-0000-0000-0000-000000000000" &&
		c.MinorUnit >= 0 && c.MinorUnit <= 9 && validDate(c.ValidFrom) &&
		(c.ValidTo == "" || (validDate(c.ValidTo) && c.ValidFrom < c.ValidTo))
}

// validateRow checks source semantics independently of ledger rules and permissions.
func validateRow(row *Row, e Expectations, seen map[string]struct{}, problems *DecodeError) {
	if !referencePattern.MatchString(row.SourceAccountRef) || row.SourceAccountRef != e.SourceAccountRef {
		problems.add("account_mismatch", row.Record, "source_account_ref")
	}
	if !referencePattern.MatchString(row.EntryID) {
		problems.add("invalid_entry_id", row.Record, "entry_id")
	} else {
		if _, exists := seen[row.EntryID]; exists {
			problems.add("duplicate_entry_id", row.Record, "entry_id")
		}
		seen[row.EntryID] = struct{}{}
	}
	if !validDate(row.BookedOn) {
		problems.add("invalid_date", row.Record, "booked_on")
	} else {
		if row.BookedOn < e.StartsOn || row.BookedOn >= e.EndsBefore {
			problems.add("outside_interval", row.Record, "booked_on")
		}
		if row.BookedOn < e.Currency.ValidFrom || (e.Currency.ValidTo != "" && row.BookedOn >= e.Currency.ValidTo) {
			problems.add("currency_not_valid", row.Record, "currency")
		}
	}
	if row.Currency != e.Currency.Code {
		problems.add("currency_mismatch", row.Record, "currency")
	}
	if row.Direction != "inflow" && row.Direction != "outflow" {
		problems.add("invalid_direction", row.Record, "direction")
	}
	canonical, ok := canonicalAmount(row.Amount, e.Currency.MinorUnit)
	if !ok {
		problems.add("invalid_amount", row.Record, "amount")
	} else {
		row.CanonicalAmount = canonical
	}
	if len(row.Description) == 0 || len(row.Description) > 500 {
		problems.add("invalid_description", row.Record, "description")
	}
}

// canonicalAmount manipulates decimal digits only; significant scale cannot be rounded.
func canonicalAmount(original string, minorUnit int) (string, bool) {
	if !amountPattern.MatchString(original) {
		return "", false
	}
	integer, fraction, _ := strings.Cut(original, ".")
	integer = strings.TrimLeft(integer, "0")
	fraction = strings.TrimRight(fraction, "0")
	if len(fraction) > minorUnit || (integer == "" && fraction == "") {
		return "", false
	}
	if integer == "" {
		integer = "0"
	}
	if fraction != "" {
		return integer + "." + fraction, true
	}
	return integer, true
}
