package plaid

import "regexp"

// CategoryMapEntry is one explicitly reviewed taxonomy pair, never a provider-derived topic install.
type CategoryMapEntry struct {
	Version, Primary, Detailed, TopicID string
}

// CategorySuggestion is inert attributed output; it grants no source access and performs no mutation.
type CategorySuggestion struct {
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	MapEdition string `json:"map_edition"`
	TopicID    string `json:"topic_id,omitempty"`
}

// categoryKey preserves exact version and code pairs without ambiguous concatenated strings.
type categoryKey struct{ version, primary, detailed string }

// CategoryMap owns a defensive immutable copy of a bounded caller-supplied approved map edition.
// Current taxonomy/source authority and revision checks must be supplied by future304 integration.
type CategoryMap struct {
	edition string
	entries map[categoryKey]string
}

var categoryMapID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// CompileCategoryMap rejects duplicate pairs and unreviewed versions; it never installs provider codes.
func CompileCategoryMap(edition string, entries []CategoryMapEntry) (*CategoryMap, error) {
	if !categoryMapID.MatchString(edition) || len(entries) > 1000 {
		return nil, fail("invalid_category_map")
	}
	compiled := &CategoryMap{edition: edition, entries: make(map[categoryKey]string, len(entries))}
	for _, entry := range entries {
		if !oneOf(entry.Version, "v1", "v2") || !categoryCode.MatchString(entry.Primary) || !categoryCode.MatchString(entry.Detailed) || !categoryMapID.MatchString(entry.TopicID) {
			return nil, fail("invalid_category_map")
		}
		key := categoryKey{entry.Version, entry.Primary, entry.Detailed}
		if _, exists := compiled.entries[key]; exists {
			return nil, fail("invalid_category_map")
		}
		compiled.entries[key] = entry.TopicID
	}
	return compiled, nil
}

// Suggest recomputes metadata validity/reason and matches exactly one explicit versioned pair.
// No confidence threshold is inferred: known vendor levels are attribution, never a Jayburd score.
func (m *CategoryMap) Suggest(category PersonalFinanceCategory) (CategorySuggestion, error) {
	if m == nil {
		return CategorySuggestion{}, fail("invalid_category_map")
	}
	reason, err := categoryReason(category)
	if err != nil || reason != category.UnavailableReason {
		return CategorySuggestion{}, fail("invalid_category_metadata")
	}
	result := CategorySuggestion{Status: "abstained", Reason: reason, MapEdition: m.edition}
	if reason != "" {
		return result, nil
	}
	key := categoryKey{*category.Version, *category.Primary, *category.Detailed}
	topic, found := m.entries[key]
	if !found {
		result.Reason = "unmapped_category"
		return result, nil
	}
	result.Status, result.TopicID = "suggested", topic
	return result, nil
}
