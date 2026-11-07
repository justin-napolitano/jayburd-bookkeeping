package plaid_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

// categoryRow adds only fictional metadata to the existing independent exact-money row fixture.
func categoryRow(amount, currency, metadata string) string {
	base := row(amount, currency)
	if metadata == "" {
		return base
	}
	return base[:len(base)-1] + `,"personal_finance_category":` + metadata + `}`
}

// categoryJSON constructs independently chosen codes, not a provider dictionary or production map.
func categoryJSON(version, primary, detailed, confidence string) string {
	raw, _ := json.Marshal(map[string]any{"version": version, "primary": primary, "detailed": detailed, "confidence_level": confidence})
	return string(raw)
}

// categoryFixture decodes a current fictional page and requires exactly one correctly keyed sidecar row.
func categoryFixture(t testing.TB, metadata string) (plaid.Page, plaid.CategoryPage) {
	t.Helper()
	financial, sidecar, err := plaid.DecodePageWithCategories(page(categoryRow("24.01", "USD", metadata), "", ""), expected())
	if err != nil || len(sidecar.Attributions) != 1 {
		t.Fatalf("category fixture: %v", err)
	}
	if sidecar.Profile != plaid.CategoryProfile || sidecar.SourcePageSHA256 != financial.SHA256 || sidecar.Attributions[0].AccountID != "Account-A" || sidecar.Attributions[0].TransactionID != "Txn-A" || sidecar.Attributions[0].Change != "added" {
		t.Fatal("sidecar identity/provenance mismatch")
	}
	return financial, sidecar
}

// TestPFCVersionMapping uses explicit immutable editions and version-specific code-pair mappings.
func TestPFCVersionMapping(t *testing.T) {
	entries := []plaid.CategoryMapEntry{{Version: "v1", Primary: "FICTIONAL_FOOD", Detailed: "FICTIONAL_CAFE", TopicID: "dining-v1"}, {Version: "v2", Primary: "FICTIONAL_FOOD", Detailed: "FICTIONAL_CAFE", TopicID: "dining-v2"}}
	mapping, err := plaid.CompileCategoryMap("map-1", entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1", "v2"} {
		_, sidecar := categoryFixture(t, categoryJSON(version, "FICTIONAL_FOOD", "FICTIONAL_CAFE", "HIGH"))
		pfc := sidecar.Attributions[0].Category
		got, err := mapping.Suggest(pfc)
		if err != nil || got.Status != "suggested" || got.TopicID != "dining-"+version || got.MapEdition != "map-1" || got.Reason != "" {
			t.Fatal("wrong explicit version mapping", got, err)
		}
		if pfc.Version == nil || *pfc.Version != version || pfc.Confidence == nil || *pfc.Confidence != "HIGH" {
			t.Fatal("provider attribution changed")
		}
	}
}

// TestPFCOmittedUnknown separates unavailable metadata from explicit unknown version/code/confidence.
func TestPFCOmittedUnknown(t *testing.T) {
	mapping, _ := plaid.CompileCategoryMap("map-1", []plaid.CategoryMapEntry{{Version: "v2", Primary: "FICTIONAL_FOOD", Detailed: "FICTIONAL_CAFE", TopicID: "dining"}})
	for name, tc := range map[string]struct{ metadata, reason string }{
		"omitted": {"", "not_provided"}, "null": {"null", "null_category"},
		"empty":              {`{}`, "incomplete_category"},
		"missing primary":    {`{"detailed":"FICTIONAL_CAFE","version":"v2","confidence_level":"HIGH"}`, "incomplete_category"},
		"null detailed":      {`{"primary":"FICTIONAL_FOOD","detailed":null,"version":"v2","confidence_level":"HIGH"}`, "incomplete_category"},
		"versionless":        {`{"primary":"FICTIONAL_FOOD","detailed":"FICTIONAL_CAFE","confidence_level":"HIGH"}`, "version_missing"},
		"null version":       {`{"primary":"FICTIONAL_FOOD","detailed":"FICTIONAL_CAFE","version":null,"confidence_level":"HIGH"}`, "version_missing"},
		"future version":     {categoryJSON("v9", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "HIGH"), "unsupported_version"},
		"missing confidence": {`{"primary":"FICTIONAL_FOOD","detailed":"FICTIONAL_CAFE","version":"v2"}`, "confidence_missing"},
		"null confidence":    {`{"primary":"FICTIONAL_FOOD","detailed":"FICTIONAL_CAFE","version":"v2","confidence_level":null}`, "confidence_missing"},
		"unknown confidence": {categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "UNKNOWN"), "confidence_unknown"},
		"new confidence":     {categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "FUTURE_CONFIDENCE"), "confidence_unrecognized"},
		"unmapped pair":      {categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_OTHER", "HIGH"), "unmapped_category"},
	} {
		t.Run(name, func(t *testing.T) {
			_, sidecar := categoryFixture(t, tc.metadata)
			got, err := mapping.Suggest(sidecar.Attributions[0].Category)
			if err != nil || got.Status != "abstained" || got.Reason != tc.reason || got.TopicID != "" || got.MapEdition != "map-1" {
				t.Fatal("wrong abstention", got, err)
			}
		})
	}
}

// TestPFCConfidenceNotCalibration retains each documented level as text without inventing a score.
func TestPFCConfidenceNotCalibration(t *testing.T) {
	mapping, _ := plaid.CompileCategoryMap("map-1", []plaid.CategoryMapEntry{{Version: "v2", Primary: "FICTIONAL_FOOD", Detailed: "FICTIONAL_CAFE", TopicID: "dining"}})
	for _, confidence := range []string{"VERY_HIGH", "HIGH", "MEDIUM", "LOW", "UNKNOWN"} {
		_, sidecar := categoryFixture(t, categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", confidence))
		pfc := sidecar.Attributions[0].Category
		if pfc.Confidence == nil || *pfc.Confidence != confidence {
			t.Fatal("confidence attribution changed")
		}
		got, err := mapping.Suggest(pfc)
		if err != nil || (got.Status == "suggested") != (confidence != "UNKNOWN") {
			t.Fatal("unknown confidence did not abstain", err)
		}
		wire, _ := json.Marshal(pfc)
		if bytes.Contains(wire, []byte("probability")) || bytes.Contains(wire, []byte("score")) {
			t.Fatal("invented calibrated score")
		}
	}
}

// TestPFCNormalizedCompatibility proves enhanced opt-in leaves every existing financial JSON key/value intact.
func TestPFCNormalizedCompatibility(t *testing.T) {
	for _, metadata := range []string{"", "null", categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "HIGH"), categoryJSON("v9", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "UNKNOWN")} {
		raw := page(categoryRow("9007199254740993.01", "USD", metadata), "", "")
		legacy, err := plaid.DecodePage(raw, expected())
		if err != nil {
			t.Fatal(err)
		}
		enhanced, sidecar, err := plaid.DecodePageWithCategories(raw, expected())
		if err != nil || sidecar.SourcePageSHA256 != legacy.SHA256 || !reflect.DeepEqual(legacy, enhanced) {
			t.Fatal("enhanced decode changed legacy result", err)
		}
		a, _ := json.Marshal(legacy)
		b, _ := json.Marshal(enhanced)
		if !bytes.Equal(a, b) || bytes.Contains(a, []byte("personal_finance_category")) || bytes.Contains(a, []byte("attributions")) {
			t.Fatal("current closed financial wire changed")
		}
		wire, _ := json.Marshal(sidecar)
		if bytes.Contains(wire, []byte("amount")) || bytes.Contains(wire, []byte("original_description")) || bytes.Contains(wire, []byte("raw")) {
			t.Fatal("sidecar leaked unrelated financial/raw fields")
		}
	}
}

// TestPFCRevisionCorrection keeps financial Transaction equality while recording current category changes.
func TestPFCRevisionCorrection(t *testing.T) {
	first, a := categoryFixture(t, categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "HIGH"))
	second, b := categoryFixture(t, categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_MARKET", "LOW"))
	x, _ := json.Marshal(first.Added[0])
	y, _ := json.Marshal(second.Added[0])
	if !bytes.Equal(x, y) || first.SHA256 == second.SHA256 || a.SourcePageSHA256 == b.SourcePageSHA256 || *a.Attributions[0].Category.Detailed == *b.Attributions[0].Category.Detailed {
		t.Fatal("metadata correction changed finance or lost original provenance")
	}
	// Real source/metadata revision counters and posted anchors belong to the future SQL profile.
}

// TestPFCEnhancedAtomicity rejects malformed PFC after a valid first row without leaking partial output.
func TestPFCEnhancedAtomicity(t *testing.T) {
	valid := categoryRow("24.01", "USD", categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "HIGH"))
	for name, metadata := range map[string]string{
		"scalar": `42`, "array": `[]`, "boolean": `true`,
		"bad version type":     `{"primary":"FICTIONAL_FOOD","version":2}`,
		"bad primary type":     `{"primary":false,"version":"v2"}`,
		"bad confidence type":  `{"confidence_level":[]}`,
		"invalid code":         categoryJSON("v2", "food and drink", "FICTIONAL_CAFE", "HIGH"),
		"long code":            categoryJSON("v2", strings.Repeat("X", 129), "FICTIONAL_CAFE", "HIGH"),
		"long version":         categoryJSON(strings.Repeat("v", 17), "FICTIONAL_FOOD", "FICTIONAL_CAFE", "HIGH"),
		"long confidence":      categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", strings.Repeat("X", 33)),
		"unknown nested field": `{"primary":"FICTIONAL_FOOD","future_sensitive_field":"no-output"}`,
	} {
		t.Run(name, func(t *testing.T) {
			bad := strings.Replace(categoryRow("10", "USD", metadata), "Txn-A", "Txn-B", 1)
			raw := page(valid, bad, "")
			legacy, err := plaid.DecodePage(raw, expected())
			if err != nil || len(legacy.Added) != 1 || len(legacy.Modified) != 1 {
				t.Fatal("default inert metadata compatibility changed", err)
			}
			financial, sidecar, err := plaid.DecodePageWithCategories(raw, expected())
			if err == nil || plaid.ErrorCode(err) != "invalid_category_metadata" || !reflect.DeepEqual(financial, plaid.Page{}) || !reflect.DeepEqual(sidecar, plaid.CategoryPage{}) {
				t.Fatal("enhanced metadata failure leaked partial evidence", err)
			}
		})
	}
}

// TestPFCSourceMoneyAndScope preserves original units, date/readiness, filtering and keyed change kinds.
func TestPFCSourceMoneyAndScope(t *testing.T) {
	metadata := categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "HIGH")
	for _, tc := range []struct{ amount, currency, want string }{{"9007199254740993.01", "USD", "9007199254740993.01"}, {"125", "JPY", "125"}, {"-1.005", "KWD", "1.005"}} {
		raw := page(categoryRow(tc.amount, tc.currency, metadata), "", "")
		p, s, err := plaid.DecodePageWithCategories(raw, expected())
		if err != nil || p.Added[0].Amount != tc.want || p.Added[0].ISOCurrencyCode != tc.currency || p.Added[0].Date != "2026-01-02" || len(s.Attributions) != 1 {
			t.Fatal("source financial fact changed", err)
		}
		assertQuantity(t, p.Added[0])
	}
	pending := strings.Replace(categoryRow("24.01", "USD", metadata), `"pending":false`, `"pending":true`, 1)
	modified := strings.Replace(categoryRow("24.01", "USD", metadata), "Txn-A", "Txn-B", 1)
	p, s, err := plaid.DecodePageWithCategories(page(pending, modified, `{"account_id":"Account-A","transaction_id":"Txn-C"}`), expected())
	if err != nil || p.Added[0].EligibleForReview || len(s.Attributions) != 2 || s.Attributions[0].TransactionID != "Txn-A" || s.Attributions[1].TransactionID != "Txn-B" || s.Attributions[1].Change != "modified" {
		t.Fatal("keyed sidecar/removed/pending semantics changed", err)
	}
	for _, raw := range [][]byte{
		page(strings.Replace(categoryRow("1", "USD", metadata), "Account-A", "Account-B", 1), "", ""),
		page(categoryRow("1e-10", "USD", metadata), "", ""),
		page(categoryRow("1", "USD", metadata), categoryRow("1", "USD", metadata), ""),
	} {
		p, s, err := plaid.DecodePageWithCategories(raw, expected())
		if err == nil || !reflect.DeepEqual(p, plaid.Page{}) || !reflect.DeepEqual(s, plaid.CategoryPage{}) {
			t.Fatal("enhanced decode weakened atomic money/scope/duplicate rejection")
		}
	}
}

// TestPFCMapClosedImmutable rejects malformed catalogs and protects compiled mappings from caller edits.
func TestPFCMapClosedImmutable(t *testing.T) {
	entry := plaid.CategoryMapEntry{Version: "v2", Primary: "FICTIONAL_FOOD", Detailed: "FICTIONAL_CAFE", TopicID: "dining"}
	for _, entries := range [][]plaid.CategoryMapEntry{{entry, entry}, {{Version: "v9", Primary: entry.Primary, Detailed: entry.Detailed, TopicID: entry.TopicID}}, {{Version: "v2", Primary: "bad code", Detailed: entry.Detailed, TopicID: entry.TopicID}}, {{Version: "v2", Primary: entry.Primary, Detailed: entry.Detailed, TopicID: ""}}, make([]plaid.CategoryMapEntry, 1001)} {
		if m, err := plaid.CompileCategoryMap("map-1", entries); err == nil || m != nil || plaid.ErrorCode(err) != "invalid_category_map" {
			t.Fatal("invalid catalog accepted", err)
		}
	}
	if _, err := plaid.CompileCategoryMap("", nil); err == nil {
		t.Fatal("missing map edition accepted")
	}
	entries := []plaid.CategoryMapEntry{entry}
	m, _ := plaid.CompileCategoryMap("map-1", entries)
	entries[0].TopicID = "changed"
	_, sidecar := categoryFixture(t, categoryJSON("v2", entry.Primary, entry.Detailed, "HIGH"))
	pfc := sidecar.Attributions[0].Category
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 100 {
				got, err := m.Suggest(pfc)
				if err != nil || got.TopicID != "dining" {
					t.Error("compiled map mutated", err)
				}
			}
		})
	}
	workers.Wait()
	pfc.UnavailableReason = "forged"
	if _, err := m.Suggest(pfc); err == nil {
		t.Fatal("inconsistent projection accepted")
	}
	var absent *plaid.CategoryMap
	if _, err := absent.Suggest(plaid.PersonalFinanceCategory{}); err == nil {
		t.Fatal("nil map accepted")
	}
}

// FuzzPFCSidecar checks enhanced atomicity and exact legacy projection over arbitrary bounded category JSON.
func FuzzPFCSidecar(f *testing.F) {
	for _, metadata := range []string{"null", `{}`, categoryJSON("v2", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "HIGH"), `{"primary":false}`, categoryJSON("v9", "FICTIONAL_FOOD", "FICTIONAL_CAFE", "UNKNOWN")} {
		f.Add(metadata)
	}
	f.Fuzz(func(t *testing.T, metadata string) {
		if len(metadata) > 2048 {
			return
		}
		raw := page(categoryRow("9007199254740993.01", "USD", metadata), "", "")
		p, s, err := plaid.DecodePageWithCategories(raw, expected())
		if err != nil {
			if !reflect.DeepEqual(p, plaid.Page{}) || !reflect.DeepEqual(s, plaid.CategoryPage{}) {
				t.Fatal("partial output on failure")
			}
			return
		}
		legacy, err := plaid.DecodePage(raw, expected())
		if err != nil || !reflect.DeepEqual(p, legacy) || len(s.Attributions) != 1 || s.SourcePageSHA256 != p.SHA256 {
			t.Fatal("legacy or sidecar provenance drift")
		}
	})
}
