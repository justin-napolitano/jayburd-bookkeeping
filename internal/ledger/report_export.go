package ledger

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// exportMeasures fixes wire order without inferring any financial calculations.
var exportMeasures = strings.Split("net,debit_balance,credit_balance,balance,debit,credit,running_balance,transaction_amount,functional_amount,total_debits,total_credits,opening,period_debits,period_credits,closing,revenue,expense,net_income,assets,liabilities,direct_equity,unclosed_earnings,total_equity", ",")
var exportHeader = strings.Split("profile,report,layout,entity_id,book_id,snapshot_revision,functional_currency,functional_currency_version_id,minor_units,basis,timezone,generated_at,policy_ids,through_date,starts_on,ends_before,compare_through_date,compare_starts_on,compare_ends_before,selected_account_id,row_kind,row_id,account_id,category,normal_side,journal_id,line_number,accounting_date,posted_at,journal_kind,reverses_journal_id,policy_id,side,transaction_currency,transaction_currency_version_id,valuation_basis,rate_observation_id,measure,current_sign,current_magnitude,prior_sign,prior_magnitude,difference_sign,difference_magnitude", ",")
var exportUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var exportDecimal = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,28})(\.[0-9]{0,8}[1-9])?$`)
var exportInteger = regexp.MustCompile(`^(0|[1-9][0-9]{0,18})$`)
var exportCurrency = regexp.MustCompile(`^[A-Z]{3}$`)
var exportTimestamp = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$`)

// exportComparison preserves SQL-produced prior values and differences verbatim.
type exportComparison struct {
	Totals     map[string]string `json:"totals"`
	Amounts    map[string]string `json:"amounts"`
	Difference map[string]string `json:"difference"`
}

// exportPage projects only owned wire fields; descriptive metadata is never exported.
type exportPage struct {
	Contract        string                       `json:"contract_version"`
	Report          string                       `json:"report"`
	Entity          string                       `json:"entity_id"`
	Book            string                       `json:"book_id"`
	Revision        string                       `json:"snapshot_revision"`
	Generated       string                       `json:"generated_at"`
	Currency        string                       `json:"functional_currency"`
	CurrencyVersion string                       `json:"functional_currency_version_id"`
	MinorUnits      int                          `json:"minor_units"`
	Basis           string                       `json:"basis"`
	Timezone        string                       `json:"timezone"`
	Policies        []string                     `json:"policy_ids"`
	Filters         map[string]string            `json:"filters"`
	Totals          map[string]string            `json:"totals"`
	Comparison      *exportComparison            `json:"comparison"`
	Rows            []map[string]json.RawMessage `json:"rows"`
	Cursor          json.RawMessage              `json:"next_cursor"`
}

// FinancialReportCSV buffers the complete selection; each page and final read
// reauthorize and verify the requested revision before any download can escape.
func (s *Service) FinancialReportCSV(parent context.Context, c Command) (Result, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	return exportReport(ctx, c, s.FinancialReport, 10000, 10*1024*1024)
}

// exportReport exposes a narrow test seam for mid-export revocation and bounds.
func exportReport(ctx context.Context, c Command, read func(context.Context, Command) (Result, error), maxRows, maxBytes int) (Result, error) {
	first, err := read(ctx, c) // Preserve SQL authorization and raw duplicate-key validation.
	if err != nil {
		return Result{}, err
	}
	var request map[string]json.RawMessage
	var revision string
	if json.Unmarshal([]byte(c.Body), &request) != nil || json.Unmarshal(request["snapshot_revision"], &revision) != nil || !exportInteger.MatchString(revision) || request["cursor"] != nil || request["limit"] != nil {
		return Result{}, &Failure{400, "invalid_request"}
	}
	var baseline exportPage
	if decodeExportPage(first.Body, &baseline) != nil {
		return Result{}, exportInvalid()
	}
	provenance, err := exportProvenance(baseline, c, revision)
	if err != nil {
		return Result{}, err
	}
	var output bytes.Buffer
	if err = exportRecord(&output, exportHeader, maxBytes); err != nil {
		return Result{}, err
	}
	page := baseline
	count := 0
	seenRows := map[string]bool{}
	seenCursors := map[string]bool{}
	for {
		if ctx.Err() != nil {
			return Result{}, &Failure{503, "temporarily_unavailable"}
		}
		if !exportSamePage(baseline, page) {
			return Result{}, &Failure{409, "report_snapshot_changed"}
		}
		for _, row := range page.Rows {
			id, ok := exportString(row, "id")
			if !ok || seenRows[id] {
				return Result{}, exportInvalid()
			}
			seenRows[id] = true
			fields, amounts, comparison, err := exportRow(page.Report, row)
			if err != nil {
				return Result{}, err
			}
			if (comparison != nil) != (baseline.Comparison != nil) {
				return Result{}, exportInvalid()
			}
			if err = exportMeasuresTo(&output, provenance, fields, amounts, comparison, &count, maxRows, maxBytes); err != nil {
				return Result{}, err
			}
		}
		if len(page.Cursor) == 0 || string(page.Cursor) == "null" {
			break
		}
		if len(page.Rows) == 0 || seenCursors[string(page.Cursor)] {
			return Result{}, exportInvalid()
		}
		seenCursors[string(page.Cursor)] = true
		request["cursor"] = page.Cursor
		request["limit"] = json.RawMessage("500")
		raw, err := json.Marshal(request)
		if err != nil {
			return Result{}, exportInvalid()
		}
		next := c
		next.Body = string(raw)
		r, err := read(ctx, next)
		if err != nil {
			return Result{}, err
		}
		page = exportPage{}
		if decodeExportPage(r.Body, &page) != nil {
			return Result{}, exportInvalid()
		}
	}
	fields := make([]string, 17)
	fields[0] = "summary"
	fields[1] = "summary"
	if !exportExactKeys(baseline.Totals, exportTotalKeys(baseline.Report)) {
		return Result{}, exportInvalid()
	}
	summaryCmp := baseline.Comparison
	if summaryCmp != nil {
		summaryCmp = &exportComparison{Amounts: summaryCmp.Totals, Difference: summaryCmp.Difference}
	}
	if err = exportMeasuresTo(&output, provenance, fields, baseline.Totals, summaryCmp, &count, maxRows, maxBytes); err != nil {
		return Result{}, err
	}
	// A separate final authorization transaction catches a revocation or new posting
	// committed during serialization, while never claiming perpetual authorization.
	final, err := read(ctx, c)
	if err != nil {
		return Result{}, err
	}
	var last exportPage
	if decodeExportPage(final.Body, &last) != nil {
		return Result{}, exportInvalid()
	}
	if !exportSamePage(baseline, last) {
		return Result{}, &Failure{409, "report_snapshot_changed"}
	}
	if ctx.Err() != nil {
		return Result{}, &Failure{503, "temporarily_unavailable"}
	}
	return Result{Status: 200, Body: json.RawMessage(output.Bytes())}, nil
}

// exportSamePage binds all shared accounting provenance and full totals across reads.
func exportSamePage(a, b exportPage) bool {
	a.Generated = ""
	b.Generated = ""
	a.Rows = nil
	b.Rows = nil
	a.Cursor = nil
	b.Cursor = nil
	return reflect.DeepEqual(a, b)
}

// exportProvenance validates every repeated cell against a closed grammar.
func exportProvenance(p exportPage, c Command, revision string) ([]string, error) {
	if p.Contract != "financial-reporting-sql-v1" || p.Entity != strings.ToLower(c.Entity) || p.Book != strings.ToLower(c.Book) || !exportValidUUID(p.Entity) || !exportValidUUID(p.Book) || p.Revision != revision || !exportCurrency.MatchString(p.Currency) || !exportValidUUID(p.CurrencyVersion) || p.MinorUnits < 0 || p.MinorUnits > 9 || p.Basis != "accrual" || p.Timezone != "UTC" || !exportValidTime(p.Generated) || len(p.Policies) == 0 || len(p.Policies) > 1024 || len(exportTotalKeys(p.Report)) == 0 {
		return nil, exportInvalid()
	}
	for _, id := range p.Policies {
		if !exportValidUUID(id) {
			return nil, exportInvalid()
		}
	}
	if p.Filters["report"] != p.Report {
		return nil, exportInvalid()
	}
	var req map[string]string
	if json.Unmarshal([]byte(c.Body), &req) != nil {
		return nil, exportInvalid()
	}
	delete(req, "snapshot_revision")
	if req["account_id"] != "" {
		req["account_id"] = strings.ToLower(req["account_id"])
	}
	if !reflect.DeepEqual(req, p.Filters) {
		return nil, exportInvalid()
	}
	values := []string{"financial-reporting-csv-v1", p.Report, reportLayout(p.Report), p.Entity, p.Book, p.Revision, p.Currency, p.CurrencyVersion, string(rune('0' + p.MinorUnits)), p.Basis, p.Timezone, p.Generated, strings.Join(p.Policies, ";")}
	for _, key := range []string{"through_date", "starts_on", "ends_before", "compare_through_date", "compare_starts_on", "compare_ends_before"} {
		value := p.Filters[key]
		if value != "" && !exportValidDate(value) {
			return nil, exportInvalid()
		}
		values = append(values, value)
	}
	id := p.Filters["account_id"]
	if id != "" && !exportValidUUID(id) {
		return nil, exportInvalid()
	}
	values = append(values, id)
	compared := p.Filters["compare_through_date"] != "" || p.Filters["compare_starts_on"] != ""
	if (p.Comparison != nil) != compared {
		return nil, exportInvalid()
	}
	return values, nil
}

// exportRow maps only the approved row-kind field matrix and SQL amount keys.
func exportRow(kind string, row map[string]json.RawMessage) ([]string, map[string]string, *exportComparison, error) {
	fields := make([]string, 17)
	id, ok := exportString(row, "id")
	if !ok {
		return nil, nil, nil, exportInvalid()
	}
	fields[1] = id
	var amounts map[string]string
	if json.Unmarshal(row["amounts"], &amounts) != nil {
		return nil, nil, nil, exportInvalid()
	}
	var cmp *exportComparison
	if raw, ok := row["comparison"]; ok && json.Unmarshal(raw, &cmp) != nil {
		return nil, nil, nil, exportInvalid()
	}
	expected := []string{"balance"}
	if kind == "general-ledger" {
		fields[0] = "posting"
		for i, key := range []string{"account_id", "", "", "journal_id", "", "accounting_date", "posted_at", "kind", "reverses_journal_id", "policy_id", "side", "transaction_currency", "transaction_currency_version_id", "valuation_basis", "rate_observation_id"} {
			if key == "" {
				continue
			}
			value, present := exportString(row, key)
			optional := key == "reverses_journal_id" || key == "rate_observation_id"
			if (!present && !optional) || (optional && row[key] != nil && string(row[key]) != "null" && !present) {
				return nil, nil, nil, exportInvalid()
			}
			fields[i+2] = value
		}
		fields[6] = string(row["line_number"])
		lineNumber, lineErr := strconv.Atoi(fields[6])
		if lineErr != nil || lineNumber > 1000 {
			return nil, nil, nil, exportInvalid()
		}
		if !regexp.MustCompile(`^[1-9][0-9]{0,3}$`).MatchString(fields[6]) || id != "line:"+fields[5]+":"+fields[6] || !exportValidDate(fields[7]) || !exportValidTime(fields[8]) || !exportOneOf(fields[9], "manual", "reversal") || !exportOneOf(fields[12], "debit", "credit") || !exportCurrency.MatchString(fields[13]) || !exportOneOf(fields[15], "identity", "spot") {
			return nil, nil, nil, exportInvalid()
		}
		for _, i := range []int{2, 5, 11, 14} {
			if !exportValidUUID(fields[i]) {
				return nil, nil, nil, exportInvalid()
			}
		}
		for _, i := range []int{10, 16} {
			if fields[i] != "" && !exportValidUUID(fields[i]) {
				return nil, nil, nil, exportInvalid()
			}
		}
		if (fields[9] == "reversal") != (fields[10] != "") || (fields[15] == "spot") != (fields[16] != "") {
			return nil, nil, nil, exportInvalid()
		}
		if !exportExactKeys(amounts, []string{"debit", "credit", "running_balance"}) {
			return nil, nil, nil, exportInvalid()
		}
		for _, key := range []string{"transaction_amount", "functional_amount"} {
			value, ok := exportString(row, key)
			if !ok {
				return nil, nil, nil, exportInvalid()
			}
			amounts[key] = value
		}
		expected = []string{"debit", "credit", "running_balance", "transaction_amount", "functional_amount"}
	} else if id == "unclosed_earnings" && kind == "balance-sheet" {
		fields[0] = "subtotal"
	} else {
		fields[0] = "account"
		for i, key := range []string{"account_id", "category", "normal_side"} {
			value, ok := exportString(row, key)
			if !ok {
				return nil, nil, nil, exportInvalid()
			}
			fields[i+2] = value
		}
		if !exportValidUUID(fields[2]) || id != "account:"+fields[2] || !exportOneOf(fields[3], "asset", "liability", "equity", "revenue", "expense") || !exportOneOf(fields[4], "debit", "credit") {
			return nil, nil, nil, exportInvalid()
		}
		if kind == "trial-balance" {
			expected = []string{"net", "debit_balance", "credit_balance"}
		}
	}
	if !exportExactKeys(amounts, expected) {
		return nil, nil, nil, exportInvalid()
	}
	return fields, amounts, cmp, nil
}

// exportMeasuresTo preserves SQL arithmetic while rendering explicit signed magnitudes.
func exportMeasuresTo(out *bytes.Buffer, provenance, fields []string, amounts map[string]string, cmp *exportComparison, count *int, maxRows, maxBytes int) error {
	if cmp != nil && (!exportSameKeys(amounts, cmp.Amounts) || !exportSameKeys(amounts, cmp.Difference)) {
		return exportInvalid()
	}
	for _, measure := range exportMeasures {
		value, ok := amounts[measure]
		if !ok {
			continue
		}
		cells := append(append(append([]string{}, provenance...), fields...), measure)
		pair, err := exportSign(value)
		if err != nil {
			return err
		}
		cells = append(cells, pair...)
		if cmp == nil {
			cells = append(cells, "", "", "", "")
		} else {
			for _, v := range []string{cmp.Amounts[measure], cmp.Difference[measure]} {
				pair, err := exportSign(v)
				if err != nil {
					return err
				}
				cells = append(cells, pair...)
			}
		}
		(*count)++
		if *count > maxRows {
			return &Failure{422, "report_export_limit"}
		}
		if err = exportRecord(out, cells, maxBytes); err != nil {
			return err
		}
	}
	return nil
}

// exportRecord bounds the complete serialized line before adding it to the buffer.
func exportRecord(out *bytes.Buffer, cells []string, maxBytes int) error {
	if len(cells) != len(exportHeader) {
		return exportInvalid()
	}
	var line strings.Builder
	for i, value := range cells {
		if i > 0 {
			line.WriteByte(',')
		}
		line.WriteByte('"')
		line.WriteString(strings.ReplaceAll(value, "\"", "\"\""))
		line.WriteByte('"')
	}
	line.WriteString("\r\n")
	if out.Len()+line.Len() > maxBytes {
		return &Failure{422, "report_export_limit"}
	}
	out.WriteString(line.String())
	return nil
}

// exportSign forbids exponent/rounding and retains each SQL canonical digit.
func exportSign(value string) ([]string, error) {
	if !exportDecimal.MatchString(value) || value == "-0" {
		return nil, exportInvalid()
	}
	if value == "0" {
		return []string{"zero", "0"}, nil
	}
	if strings.HasPrefix(value, "-") {
		return []string{"negative", value[1:]}, nil
	}
	return []string{"positive", value}, nil
}

// exportTotalKeys is the exact SQL total schema for each report kind.
func exportTotalKeys(kind string) []string {
	switch kind {
	case "trial-balance":
		return []string{"total_debits", "total_credits"}
	case "general-ledger":
		return []string{"opening", "period_debits", "period_credits", "closing"}
	case "profit-and-loss":
		return []string{"revenue", "expense", "net_income"}
	case "balance-sheet":
		return []string{"assets", "liabilities", "direct_equity", "unclosed_earnings", "total_equity"}
	}
	return nil
}

// exportExactKeys rejects missing/unknown monetary measures rather than dropping them.
func exportExactKeys(values map[string]string, keys []string) bool {
	if len(values) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := values[key]; !ok {
			return false
		}
	}
	return true
}

// exportSameKeys enforces the complete current/prior/difference matrix.
func exportSameKeys(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, ok := b[key]; !ok {
			return false
		}
	}
	return true
}

// exportString excludes nulls, numbers and silently coerced values.
func exportString(row map[string]json.RawMessage, key string) (string, bool) {
	var value string
	raw, ok := row[key]
	valid := ok && string(raw) != "null" && json.Unmarshal(raw, &value) == nil
	return value, valid
}

// exportValidUUID limits identifiers to canonical, nonnil database UUID text.
func exportValidUUID(value string) bool {
	return exportUUID.MatchString(value) && value != "00000000-0000-0000-0000-000000000000"
}

// exportValidDate validates the calendar as well as ISO-style lexical spelling.
func exportValidDate(value string) bool {
	v, err := time.Parse("2006-01-02", value)
	return err == nil && v.Year() >= 1 && v.Format("2006-01-02") == value
}

// exportValidTime accepts only the database's generated UTC microsecond spelling.
func exportValidTime(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil && exportTimestamp.MatchString(value)
}

// exportOneOf ensures categorical cells cannot become arbitrary spreadsheet text.
func exportOneOf(value string, allowed ...string) bool {
	for _, v := range allowed {
		if value == v {
			return true
		}
	}
	return false
}

// exportInvalid never exposes unexpected database output or request details.
func exportInvalid() *Failure { return &Failure{500, "internal_error"} }

// decodeExportPage rejects missing/null structural fields instead of converting
// malformed SQL output into a valid-looking empty export or zero currency scale.
func decodeExportPage(raw json.RawMessage, p *exportPage) error {
	if len(raw) > 2097152 {
		return exportInvalid()
	}
	var shape map[string]json.RawMessage
	if json.Unmarshal(raw, &shape) != nil || json.Unmarshal(raw, p) != nil {
		return exportInvalid()
	}
	for _, key := range []string{"contract_version", "report", "entity_id", "book_id", "snapshot_revision", "generated_at", "functional_currency", "functional_currency_version_id", "minor_units", "basis", "timezone", "policy_ids", "filters", "totals", "comparison", "rows", "next_cursor"} {
		if shape[key] == nil {
			return exportInvalid()
		}
	}
	if len(shape["minor_units"]) != 1 || shape["minor_units"][0] < '0' || shape["minor_units"][0] > '9' || len(shape["rows"]) == 0 || shape["rows"][0] != '[' || len(shape["next_cursor"]) == 0 || (shape["next_cursor"][0] != '{' && string(shape["next_cursor"]) != "null") {
		return exportInvalid()
	}
	return nil
}
