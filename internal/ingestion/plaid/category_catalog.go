package plaid

import (
	_ "embed"
	"encoding/json"
)

//go:embed category_catalog.json
var categoryCatalogJSON []byte

// StarterTopic is a readable analytic bucket, not a ledger account or tax assertion.
type StarterTopic struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// StarterPair is one exact same-version provider pair; no cross-version conversion is implied.
type StarterPair struct {
	Version  string `json:"version"`
	Primary  string `json:"primary"`
	Detailed string `json:"detailed"`
}

// StarterCategoryCatalog records factual codes and app-authored labels from the reviewed public source.
type StarterCategoryCatalog struct {
	Edition      string         `json:"edition"`
	SourceSHA256 string         `json:"source_sha256"`
	Topics       []StarterTopic `json:"topics"`
	Pairs        []StarterPair  `json:"pairs"`
}

// StarterCategories returns a defensive copy; workspace topic IDs and current authority are SQL-owned.
func StarterCategories() StarterCategoryCatalog {
	var c StarterCategoryCatalog
	// Embedded reviewed data is checked by independent fixed corpus tests, never external input.
	if json.Unmarshal(categoryCatalogJSON, &c) != nil {
		panic("invalid embedded category catalog")
	}
	return c
}

// AutoEligibleCategory defines the conservative product threshold separately from vendor attribution.
func AutoEligibleCategory(c PersonalFinanceCategory) bool {
	reason, e := categoryReason(c)
	return e == nil && reason == "" && c.UnavailableReason == reason && c.Confidence != nil && (*c.Confidence == "HIGH" || *c.Confidence == "VERY_HIGH")
}
