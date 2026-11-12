package plaid

import "testing"

// TestStarterCatalogFixedInventory checks independently reviewed counts and version-specific traps.
func TestStarterCatalogFixedInventory(t *testing.T) {
	c := StarterCategories()
	if c.Edition != "plaid-pfc-primary-2026-11-11" || c.SourceSHA256 != "c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc" || len(c.Topics) != 18 || len(c.Pairs) != 231 {
		t.Fatal("catalog provenance/count drift")
	}
	topics := map[string]bool{}
	for _, v := range c.Topics {
		if topics[v.Code] || v.Name == "" {
			t.Fatal("topic duplicate")
		}
		topics[v.Code] = true
	}
	counts := map[string]int{}
	seen := map[StarterPair]bool{}
	for _, v := range c.Pairs {
		if !oneOf(v.Version, "v1", "v2") || !categoryCode.MatchString(v.Primary) || !categoryCode.MatchString(v.Detailed) || !topics[v.Primary] || seen[v] {
			t.Fatal("invalid catalog pair")
		}
		seen[v] = true
		counts[v.Version]++
	}
	if counts["v1"] != 104 || counts["v2"] != 127 || !seen[StarterPair{"v1", "INCOME", "INCOME_WAGES"}] || !seen[StarterPair{"v2", "INCOME", "INCOME_SALARY"}] || seen[StarterPair{"v1", "INCOME", "INCOME_SALARY"}] || !seen[StarterPair{"v2", "OTHER", "OTHER_OTHER"}] {
		t.Fatal("version inference or corpus drift")
	}
	c.Pairs[0].Primary = "MUTATED"
	c.Topics[0].Name = "MUTATED"
	fresh := StarterCategories()
	if fresh.Pairs[0].Primary == "MUTATED" || fresh.Topics[0].Name == "MUTATED" {
		t.Fatal("mutable catalog leaked")
	}
}

// TestAutoConfidencePolicy never upgrades missing/unknown vendor metadata into a probability or match.
func TestAutoConfidencePolicy(t *testing.T) {
	primary, detailed, version := "FOOD_AND_DRINK", "FOOD_AND_DRINK_COFFEE", "v2"
	for _, confidence := range []string{"VERY_HIGH", "HIGH", "MEDIUM", "LOW", "UNKNOWN", "NEW_LEVEL"} {
		c := PersonalFinanceCategory{Presence: "object", Primary: &primary, Detailed: &detailed, Version: &version, Confidence: &confidence}
		reason, _ := categoryReason(c)
		c.UnavailableReason = reason
		if AutoEligibleCategory(c) != (confidence == "HIGH" || confidence == "VERY_HIGH") {
			t.Fatal("wrong policy")
		}
		c.Version = nil
		c.UnavailableReason = "version_missing"
		if AutoEligibleCategory(c) {
			t.Fatal("inferred version")
		}
	}
}
