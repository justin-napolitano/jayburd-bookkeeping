package acquisition

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

// historyContext fixes independent metadata on both sides of the source-version boundary.
func historyContext() workContext {
	c := contextFixture()
	c.Currency.ValidFrom = "2026-09-17"
	end := c.Currency.ValidFrom
	c.CurrencyVersions = []currencyMetadata{
		{Code: "USD", VersionID: testBook, MinorUnit: 2, ValidFrom: "2001-01-01", ValidTo: &end},
		c.Currency,
	}
	return c
}

// TestCurrencyVersionContext rejects untrusted history and preserves exact interval resolution.
func TestCurrencyVersionContext(t *testing.T) {
	c := historyContext()
	if !validCurrencies(c) || !validCurrencies(contextFixture()) {
		t.Fatal("valid or legacy reference context rejected")
	}
	for _, tc := range []struct{ date, version string }{{"2026-09-16", testBook}, {"2026-09-17", testEntity}} {
		b := strings.ReplaceAll(string(fixturePage("Txn-A", "1.01", "USD", "C1", false)), "2026-01-02", tc.date)
		page, err := plaid.DecodePage([]byte(b), c.expected())
		if err != nil || len(page.Added) != 1 || page.Added[0].CurrencyVersionID != tc.version || page.Added[0].Amount != "1.01" {
			t.Fatal("dated reference or exact cents lost", err)
		}
	}
	// One provider page can cross a currency metadata release boundary.
	var combined map[string]json.RawMessage
	if err := json.Unmarshal(fixturePage("Txn-Old", "1.01", "USD", "C1", false), &combined); err != nil {
		t.Fatal(err)
	}
	var older []json.RawMessage
	if err := json.Unmarshal(combined["added"], &older); err != nil {
		t.Fatal(err)
	}
	newer := strings.ReplaceAll(string(older[0]), "Txn-Old", "Txn-New")
	newer = strings.ReplaceAll(newer, "2026-01-02", "2026-09-17")
	combined["added"] = json.RawMessage("[" + string(older[0]) + "," + newer + "]")
	wire, err := json.Marshal(combined)
	if err != nil {
		t.Fatal(err)
	}
	page, err := plaid.DecodePage(wire, c.expected())
	if err != nil || len(page.Added) != 2 || page.Added[0].CurrencyVersionID != testBook || page.Added[1].CurrencyVersionID != testEntity {
		t.Fatal("mixed history page rejected", err)
	}
	for _, mutate := range []func(*workContext){
		func(c *workContext) { c.CurrencyVersions = []currencyMetadata{} },
		func(c *workContext) { c.CurrencyVersions = c.CurrencyVersions[:1] },
		func(c *workContext) { c.CurrencyVersions[0].Code = "EUR" },
		func(c *workContext) { c.CurrencyVersions[0].VersionID = c.Currency.VersionID },
		func(c *workContext) { c.CurrencyVersions[0].ValidTo = nil },
		func(c *workContext) { c.CurrencyVersions[1].ValidFrom = "2026-09-16" },
		func(c *workContext) { c.CurrencyVersions[1].MinorUnit = 0 },
		func(c *workContext) { c.CurrencyVersions = append(c.CurrencyVersions, make([]currencyMetadata, 31)...) },
	} {
		bad := historyContext()
		mutate(&bad)
		if validCurrencies(bad) {
			t.Fatal("malformed or changed history accepted")
		}
	}
	for _, tc := range []struct{ date, amount, code string }{
		{"2000-12-31", "1.01", "USD"}, {"2026-09-16", "1.001", "USD"}, {"2026-09-16", "1.01", "EUR"},
	} {
		b := strings.ReplaceAll(string(fixturePage("Txn-A", tc.amount, tc.code, "C1", false)), "2026-01-02", tc.date)
		if _, err := plaid.DecodePage([]byte(b), c.expected()); err == nil {
			t.Fatal("missing coverage, precision or identity accepted")
		}
	}
}
