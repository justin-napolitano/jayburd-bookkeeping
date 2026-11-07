package plaid

import (
	"bytes"
	"encoding/json"
	"regexp"
)

// CategoryProfile names an explicit sidecar; it is not an accepted SQL publication profile.
const CategoryProfile = "jayburd-plaid-pfc-attribution/1"

// PersonalFinanceCategory retains bounded provider attribution and a fixed abstention reason.
// Presence distinguishes omitted metadata from provider null; neither implies a historical version.
type PersonalFinanceCategory struct {
	Presence          string  `json:"presence"`
	Version           *string `json:"version"`
	Primary           *string `json:"primary"`
	Detailed          *string `json:"detailed"`
	Confidence        *string `json:"confidence"`
	UnavailableReason string  `json:"unavailable_reason"`
}

// CategoryAttribution binds current metadata to exact provider identity, never array position.
type CategoryAttribution struct {
	AccountID     string                  `json:"account_id"`
	TransactionID string                  `json:"transaction_id"`
	Change        string                  `json:"change"`
	Category      PersonalFinanceCategory `json:"category"`
}

// CategoryPage carries only safe metadata, bound to the independently retained current source page.
// Removed identities intentionally have no category. This type has no current runner/API/SQL caller.
type CategoryPage struct {
	Profile          string                `json:"profile"`
	SourcePageSHA256 string                `json:"source_page_sha256"`
	Attributions     []CategoryAttribution `json:"attributions"`
}

var categoryCode = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,127}$`)
var categoryVersion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,15}$`)
var categoryConfidence = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,31}$`)

// DecodePageWithCategories is opt-in and atomic; the default financial decoder and wire stay unchanged.
// It reads only this request's bytes and never revisits retained historical source evidence.
func DecodePageWithCategories(raw []byte, expected Expected) (Page, CategoryPage, error) {
	financial, err := DecodePage(raw, expected)
	if err != nil {
		return Page{}, CategoryPage{}, err
	}
	sidecar := CategoryPage{Profile: CategoryProfile, SourcePageSHA256: financial.SHA256, Attributions: make([]CategoryAttribution, 0, len(financial.Added)+len(financial.Modified))}
	for group, transactions := range [][]Transaction{financial.Added, financial.Modified} {
		change := "added"
		if group == 1 {
			change = "modified"
		}
		for _, transaction := range transactions {
			category, err := currentCategory(transaction.Raw)
			if err != nil {
				return Page{}, CategoryPage{}, err
			}
			sidecar.Attributions = append(sidecar.Attributions, CategoryAttribution{AccountID: transaction.AccountID, TransactionID: transaction.TransactionID, Change: change, Category: category})
		}
	}
	return financial, sidecar, nil
}

// currentCategory interprets only already-validated current transaction bytes, retaining no raw data.
func currentCategory(raw []byte) (PersonalFinanceCategory, error) {
	var transaction map[string]json.RawMessage
	if json.Unmarshal(raw, &transaction) != nil {
		return PersonalFinanceCategory{}, fail("invalid_category_metadata")
	}
	rawCategory, exists := transaction["personal_finance_category"]
	category := PersonalFinanceCategory{Presence: "omitted", UnavailableReason: "not_provided"}
	if !exists {
		return category, nil
	}
	if bytes.Equal(bytes.TrimSpace(rawCategory), []byte("null")) {
		category.Presence, category.UnavailableReason = "null", "null_category"
		return category, nil
	}
	fields, err := object(rawCategory, "primary detailed confidence_level version")
	if err != nil {
		return PersonalFinanceCategory{}, fail("invalid_category_metadata")
	}
	category.Presence = "object"
	for _, field := range []struct {
		name   string
		max    int
		target **string
	}{
		{"primary", 128, &category.Primary}, {"detailed", 128, &category.Detailed},
		{"version", 16, &category.Version}, {"confidence_level", 32, &category.Confidence},
	} {
		value, err := nullableText(fields, field.name, field.max)
		if err != nil {
			return PersonalFinanceCategory{}, fail("invalid_category_metadata")
		}
		*field.target = value
	}
	category.UnavailableReason, err = categoryReason(category)
	if err != nil {
		return PersonalFinanceCategory{}, err
	}
	return category, nil
}

// categoryReason validates every present known field before choosing a deterministic abstention reason.
func categoryReason(category PersonalFinanceCategory) (string, error) {
	if category.Presence == "omitted" || category.Presence == "null" {
		if category.Version != nil || category.Primary != nil || category.Detailed != nil || category.Confidence != nil {
			return "", fail("invalid_category_metadata")
		}
		if category.Presence == "omitted" {
			return "not_provided", nil
		}
		return "null_category", nil
	}
	if category.Presence != "object" {
		return "", fail("invalid_category_metadata")
	}
	for _, value := range []*string{category.Primary, category.Detailed} {
		if value != nil && !categoryCode.MatchString(*value) {
			return "", fail("invalid_category_metadata")
		}
	}
	if category.Version != nil && !categoryVersion.MatchString(*category.Version) || category.Confidence != nil && !categoryConfidence.MatchString(*category.Confidence) {
		return "", fail("invalid_category_metadata")
	}
	if category.Primary == nil || category.Detailed == nil {
		return "incomplete_category", nil
	}
	if category.Version == nil {
		return "version_missing", nil
	}
	if !oneOf(*category.Version, "v1", "v2") {
		return "unsupported_version", nil
	}
	if category.Confidence == nil {
		return "confidence_missing", nil
	}
	if *category.Confidence == "UNKNOWN" {
		return "confidence_unknown", nil
	}
	if !oneOf(*category.Confidence, "VERY_HIGH", "HIGH", "MEDIUM", "LOW") {
		return "confidence_unrecognized", nil
	}
	return "", nil
}
