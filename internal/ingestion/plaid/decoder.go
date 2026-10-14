package plaid

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var iso = regexp.MustCompile(`^[A-Z]{3}$`)
var day = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
var instant = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$`)
var number = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

// DecodePage is atomic: any error returns an empty Page, including no unauthorized raw evidence.
func DecodePage(raw []byte, expected Expected) (Page, error) {
	if len(raw) > MaxBytes {
		return Page{}, fail("limit_exceeded")
	}
	if !validExpected(expected) {
		return Page{}, fail("invalid_expectation")
	}
	if err := strictJSON(raw); err != nil {
		return Page{}, err
	}
	page, err := decode(raw, expected)
	if err != nil {
		return Page{}, err
	}
	return page, nil
}

// validExpected checks reference syntax and interval consistency without substituting a currency catalog.
func validExpected(e Expected) bool {
	if !validID(e.AccountID) || len(e.Currencies) == 0 || len(e.Currencies) > 1000 {
		return false
	}
	ids := map[string]bool{}
	for _, c := range e.Currencies {
		if !iso.MatchString(c.Code) || !uuid.MatchString(c.VersionID) || c.VersionID == "00000000-0000-0000-0000-000000000000" || ids[c.VersionID] || c.MinorUnit < 0 || c.MinorUnit > 9 || !validDate(c.ValidFrom) || c.ValidTo != "" && (!validDate(c.ValidTo) || c.ValidTo <= c.ValidFrom) {
			return false
		}
		ids[c.VersionID] = true
	}
	return true
}

// decode interprets only the approved material projection and preserves the original bytes separately.
func decode(raw []byte, e Expected) (Page, error) {
	v, err := object(raw, "accounts added modified removed next_cursor has_more request_id transactions_update_status")
	if err != nil {
		return Page{}, err
	}
	p := Page{}
	if p.NextCursor, err = text(v, "next_cursor", 0, 256); err != nil {
		return p, err
	}
	if !validCursor(p.NextCursor) {
		return p, fail("invalid_cursor")
	}
	if err = json.Unmarshal(v["has_more"], &p.HasMore); err != nil || string(v["has_more"]) == "null" || v["has_more"] == nil {
		return p, fail("invalid_field")
	}
	if p.RequestID, err = text(v, "request_id", 1, 256); err != nil || !validID(p.RequestID) {
		return p, fail("invalid_field")
	}
	if p.UpdateStatus, err = text(v, "transactions_update_status", 1, 64); err != nil {
		return p, err
	}
	if !oneOf(p.UpdateStatus, "TRANSACTIONS_UPDATE_STATUS_UNKNOWN", "NOT_READY", "INITIAL_UPDATE_COMPLETE", "HISTORICAL_UPDATE_COMPLETE") {
		return p, fail("unknown_enum")
	}
	accounts, err := array(v["accounts"])
	if err != nil {
		return p, err
	}
	if len(accounts) > 1 {
		return p, fail("account_mismatch")
	}
	for _, a := range accounts {
		fields, x := object(a, "account_id type subtype balances mask name official_name verification_status verification_name verification_insights persistent_account_id apy holder_category")
		if x != nil {
			return p, x
		}
		id, x := text(fields, "account_id", 1, 256)
		if x != nil {
			return p, x
		}
		if id != e.AccountID {
			return p, fail("account_mismatch")
		}
		kind, x := text(fields, "type", 1, 32)
		if x != nil {
			return p, x
		}
		if kind != "depository" {
			return p, fail("unsupported_account")
		}
	}
	added, err := array(v["added"])
	if err != nil {
		return p, err
	}
	modified, err := array(v["modified"])
	if err != nil {
		return p, err
	}
	removed, err := array(v["removed"])
	if err != nil {
		return p, err
	}
	total := len(added) + len(modified) + len(removed)
	if total > MaxChanges {
		return p, fail("limit_exceeded")
	}
	if (len(added)+len(modified) > 0) && len(accounts) != 1 {
		return p, fail("account_mismatch")
	}
	if p.NextCursor == "" && (p.HasMore || total > 0 || p.UpdateStatus == "INITIAL_UPDATE_COMPLETE" || p.UpdateStatus == "HISTORICAL_UPDATE_COMPLETE") {
		return p, fail("invalid_cursor")
	}
	p.Added = make([]Transaction, 0, len(added))
	p.Modified = make([]Transaction, 0, len(modified))
	p.Removed = make([]Removed, 0, len(removed))
	seen := map[string]bool{}
	for group, input := range [][]json.RawMessage{added, modified} {
		for _, r := range input {
			t, x := transaction(r, e)
			if x != nil {
				return p, x
			}
			if seen[t.TransactionID] {
				return p, fail("duplicate_identity")
			}
			seen[t.TransactionID] = true
			if group == 0 {
				p.Added = append(p.Added, t)
			} else {
				p.Modified = append(p.Modified, t)
			}
		}
	}
	for _, r := range removed {
		f, x := object(r, "account_id transaction_id")
		if x != nil {
			return p, x
		}
		id, x := text(f, "account_id", 1, 256)
		if x != nil {
			return p, x
		}
		if id != e.AccountID {
			return p, fail("account_mismatch")
		}
		tid, x := text(f, "transaction_id", 1, 256)
		if x != nil || !validID(tid) {
			return p, fail("invalid_identity")
		}
		if seen[tid] {
			return p, fail("duplicate_identity")
		}
		seen[tid] = true
		p.Removed = append(p.Removed, Removed{AccountID: id, TransactionID: tid, Raw: bytes.Clone(r)})
	}
	p.Raw = bytes.Clone(raw)
	sum := sha256.Sum256(raw)
	p.SHA256 = hex.EncodeToString(sum[:])
	return p, nil
}

// transaction maps sign/date/state without assigning an accounting date, category or posting permission.
func transaction(raw []byte, e Expected) (Transaction, error) {
	f, err := object(raw, "account_id transaction_id amount iso_currency_code unofficial_currency_code date authorized_date datetime authorized_datetime pending pending_transaction_id name merchant_name original_description account_owner check_number location payment_meta payment_channel transaction_type personal_finance_category personal_finance_category_icon_url merchant_entity_id merchant_category_code logo_url website counterparties category category_id transaction_code")
	if err != nil {
		return Transaction{}, err
	}
	// Provider classification is inert evidence, never a ledger category or direction.
	if _, err = nullableText(f, "transaction_code", 256); err != nil {
		return Transaction{}, err
	}
	t := Transaction{}
	if t.AccountID, err = text(f, "account_id", 1, 256); err != nil {
		return t, err
	}
	if t.AccountID != e.AccountID {
		return t, fail("account_mismatch")
	}
	if t.TransactionID, err = text(f, "transaction_id", 1, 256); err != nil || !validID(t.TransactionID) {
		return t, fail("invalid_identity")
	}
	if err = json.Unmarshal(f["pending"], &t.Pending); err != nil || f["pending"] == nil || string(f["pending"]) == "null" {
		return t, fail("invalid_field")
	}
	if _, exists := f["pending_transaction_id"]; !exists {
		return t, fail("invalid_field")
	}
	if t.PendingTransactionID, err = nullableText(f, "pending_transaction_id", 256); err != nil {
		return t, err
	}
	if t.PendingTransactionID != nil && (!validID(*t.PendingTransactionID) || *t.PendingTransactionID == t.TransactionID || t.Pending) {
		return t, fail("invalid_lineage")
	}
	if t.Date, err = text(f, "date", 10, 10); err != nil || !validDate(t.Date) {
		return t, fail("invalid_date")
	}
	if t.AuthorizedDate, err = nullableText(f, "authorized_date", 10); err != nil {
		return t, err
	}
	if t.AuthorizedDate != nil && !validDate(*t.AuthorizedDate) {
		return t, fail("invalid_date")
	}
	if t.Datetime, err = nullableText(f, "datetime", 40); err != nil {
		return t, err
	}
	if t.AuthorizedDatetime, err = nullableText(f, "authorized_datetime", 40); err != nil {
		return t, err
	}
	for _, v := range []*string{t.Datetime, t.AuthorizedDatetime} {
		if v != nil && !validInstant(*v) {
			return t, fail("invalid_date")
		}
	}
	if t.ISOCurrencyCode, err = text(f, "iso_currency_code", 3, 3); err != nil || !iso.MatchString(t.ISOCurrencyCode) {
		return t, fail("unsupported_currency")
	}
	if v, ok := f["unofficial_currency_code"]; !ok || string(v) != "null" {
		return t, fail("unsupported_currency")
	}
	var selected Currency
	count := 0
	for _, c := range e.Currencies {
		if c.Code == t.ISOCurrencyCode && c.ValidFrom <= t.Date && (c.ValidTo == "" || t.Date < c.ValidTo) {
			selected = c
			count++
		}
	}
	if count != 1 {
		return t, fail("unsupported_currency")
	}
	t.CurrencyVersionID = selected.VersionID
	t.AmountLexeme = string(f["amount"])
	if t.Amount, t.Direction, err = exactAmount(t.AmountLexeme, selected.MinorUnit); err != nil {
		return t, err
	}
	t.EligibleForReview = !t.Pending && t.Direction != "zero"
	if t.Name, err = text(f, "name", 1, 500); err != nil {
		return t, err
	}
	if t.MerchantName, err = nullableText(f, "merchant_name", 500); err != nil {
		return t, err
	}
	if t.OriginalDescription, err = nullableText(f, "original_description", 500); err != nil {
		return t, err
	}
	for key, values := range map[string][]string{"payment_channel": {"online", "in store", "other"}, "transaction_type": {"digital", "place", "special", "unresolved"}} {
		if v, ok := f[key]; ok && string(v) != "null" {
			s, x := text(f, key, 1, 32)
			if x != nil {
				return t, x
			}
			if !oneOf(s, values...) {
				return t, fail("unknown_enum")
			}
		}
	}
	t.Raw = bytes.Clone(raw)
	return t, nil
}

// exactAmount normalizes bounded decimal tokens with integer decimal-position arithmetic only.
func exactAmount(lexeme string, minor int) (string, string, error) {
	if len(lexeme) > 128 || !number.MatchString(lexeme) {
		return "", "", fail("invalid_number")
	}
	negative := strings.HasPrefix(lexeme, "-")
	value := strings.TrimPrefix(lexeme, "-")
	exponent := 0
	if i := strings.IndexAny(value, "eE"); i >= 0 {
		n, err := strconv.ParseInt(value[i+1:], 10, 32)
		if err != nil || n < -100 || n > 100 {
			return "", "", fail("invalid_number")
		}
		exponent = int(n)
		value = value[:i]
	}
	fraction := 0
	if i := strings.IndexByte(value, '.'); i >= 0 {
		fraction = len(value) - i - 1
		value = value[:i] + value[i+1:]
	}
	value = strings.TrimLeft(value, "0")
	if value == "" {
		return "0", "zero", nil
	}
	scale := fraction - exponent
	for strings.HasSuffix(value, "0") {
		value = value[:len(value)-1]
		scale--
	}
	if len(value) > 38 {
		return "", "", fail("amount_out_of_range")
	}
	integers := len(value) - scale
	if integers < 0 {
		integers = 0
	}
	if integers > 29 || scale > 9 || scale > minor {
		return "", "", fail("amount_out_of_range")
	}
	var result string
	if scale <= 0 {
		result = value + strings.Repeat("0", -scale)
	} else if scale >= len(value) {
		result = "0." + strings.Repeat("0", scale-len(value)) + value
	} else {
		result = value[:len(value)-scale] + "." + value[len(value)-scale:]
	}
	direction := "outflow"
	if negative {
		direction = "inflow"
	}
	return result, direction, nil
}

// object rejects unreviewed material fields; inert metadata is scanned but never interpreted.
func object(raw []byte, allowed string) (map[string]json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	var m map[string]json.RawMessage
	if len(raw) == 0 || raw[0] != '{' || json.Unmarshal(raw, &m) != nil || m == nil {
		return nil, fail("invalid_field")
	}
	permitted := map[string]bool{}
	for _, key := range strings.Fields(allowed) {
		permitted[key] = true
	}
	for key := range m {
		if !permitted[key] {
			return nil, fail("unknown_field")
		}
	}
	return m, nil
}

// array distinguishes required empty arrays from missing/null values.
func array(raw []byte) ([]json.RawMessage, error) {
	var a []json.RawMessage
	if len(raw) == 0 || raw[0] != '[' || json.Unmarshal(raw, &a) != nil {
		return nil, fail("invalid_field")
	}
	return a, nil
}

// text requires an actual JSON string and counts Unicode codepoints without normalization.
func text(m map[string]json.RawMessage, key string, min, max int) (string, error) {
	var value string
	raw := m[key]
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &value) != nil || utf8.RuneCountInString(value) < min || utf8.RuneCountInString(value) > max {
		return "", fail("invalid_field")
	}
	return value, nil
}

// nullableText preserves null versus text; omitted optional values remain distinguishable in raw evidence.
func nullableText(m map[string]json.RawMessage, key string, max int) (*string, error) {
	raw, ok := m[key]
	if !ok || string(raw) == "null" {
		return nil, nil
	}
	value, err := text(m, key, 0, max)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// validID bounds opaque case-sensitive identifiers without interpreting paths or URLs.
func validID(value string) bool {
	if len(value) < 1 || len(value) > 256 {
		return false
	}
	for _, c := range value {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	return true
}

// validCursor bounds opaque cursor text without decoding or adopting its stream identity.
func validCursor(value string) bool {
	if len(value) > 256 {
		return false
	}
	for _, c := range value {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("+/=_-", c)) {
			return false
		}
	}
	return true
}

// validDate rejects nonexistent Gregorian dates and year zero.
func validDate(value string) bool {
	if !day.MatchString(value) || value[:4] == "0000" {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

// validInstant checks explicit timestamp syntax while preserving original offset and precision.
func validInstant(value string) bool {
	if !instant.MatchString(value) || !validDate(value[:10]) {
		return false
	}
	if !strings.HasSuffix(value, "Z") {
		zone := value[len(value)-6:]
		if zone[1:3] > "23" || zone[4:6] > "59" {
			return false
		}
	}
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

// oneOf holds unreviewed material enum additions for a future parser revision.
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
