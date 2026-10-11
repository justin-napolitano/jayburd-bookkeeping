package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// fixture loads only committed original synthetic examples, never provider data.
func fixture(t *testing.T, name string) Corpus {
	t.Helper()
	f, err := os.Open("../../tests/corpus/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, err := decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestCorpusFixtures compares every book and currency against fixed reviewed oracles.
func TestCorpusFixtures(t *testing.T) {
	for _, name := range []string{"golden-month", "fx-lifecycle"} {
		t.Run(name, func(t *testing.T) {
			if err := validate(fixture(t, name)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestCorpusRejectsMutations proves defects cannot silently become a new oracle.
func TestCorpusRejectsMutations(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Corpus)
	}{
		{"unbalanced", func(c *Corpus) {
			c.Books[0].Entries[0].Lines[0].Functional = "999"
			c.Books[0].Entries[0].Lines[0].Amount = "999"
		}},
		{"wrong_expected", func(c *Corpus) { c.Books[0].Expected["Assets:Checking"] = "0" }},
		{"missing_expected", func(c *Corpus) { delete(c.Books[0].Expected, "Assets:Checking") }},
		{"unknown_account", func(c *Corpus) { c.Books[0].Entries[0].Lines[0].Account = "Assets:Missing" }},
		{"scoped_account", func(c *Corpus) { c.Books[1].Entries[0].Lines[0].Account = "Assets:Savings" }},
		{"duplicate_entry", func(c *Corpus) { c.Books[0].Entries = append(c.Books[0].Entries, c.Books[0].Entries[0]) }},
		{"duplicate_book", func(c *Corpus) { c.Books[1].ID = c.Books[0].ID }},
		{"wrong_units", func(c *Corpus) { c.Books[0].Units["Assets:Checking"]["USD"] = "99" }},
		{"unasserted_units", func(c *Corpus) { c.Books[0].Units["Assets:Checking"] = map[string]string{"EUR": "0"} }},
		{"nonfinite", func(c *Corpus) { c.Books[0].Entries[0].Lines[0].Amount = "NaN" }},
		{"negative", func(c *Corpus) { c.Books[0].Entries[0].Lines[0].Amount = "-1000" }},
		{"float_exponent", func(c *Corpus) { c.Books[0].Entries[0].Lines[0].Amount = "1e3" }},
		{"excess_scale", func(c *Corpus) { c.Books[0].Entries[0].Lines[0].Amount = "1000.0000000001" }},
		{"zero", func(c *Corpus) { c.Books[0].Entries[0].Lines[0].Amount = "0" }},
		{"unknown_kind", func(c *Corpus) { c.Books[0].Entries[0].Kind = "mystery" }},
		{"invalid_date", func(c *Corpus) { c.Books[0].Entries[0].Date = "2026-02-30" }},
		{"altered_reversal", func(c *Corpus) {
			c.Books[0].Entries[10].Lines[0].Amount = "21"
			c.Books[0].Entries[10].Lines[0].Functional = "21"
			c.Books[0].Entries[10].Lines[1].Amount = "9"
			c.Books[0].Entries[10].Lines[1].Functional = "9"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := fixture(t, "golden-month")
			tc.change(&c)
			if validate(c) == nil {
				t.Fatal("mutation accepted")
			}
		})
	}
}

// TestFXExposureMutation catches quantity drift hidden by balanced functional amounts.
func TestFXExposureMutation(t *testing.T) {
	c := fixture(t, "fx-lifecycle")
	c.Books[0].Entries[1].Lines[0].Amount = "1"
	if validate(c) == nil {
		t.Fatal("remeasurement changed foreign exposure undetected")
	}
}

// TestDecodeBoundary catches unknown fields and extra documents in fixture inputs.
func TestDecodeBoundary(t *testing.T) {
	for _, s := range []string{`{"unknown":1}`, `{} {}`, `{`, `{}` + strings.Repeat(" ", 8<<20)} {
		if _, err := decode(strings.NewReader(s)); err == nil {
			t.Fatal("accepted malformed fixture")
		}
	}
}

// TestFunctionalExports verifies explicit expectations survive both export formats.
func TestFunctionalExports(t *testing.T) {
	c := fixture(t, "golden-month")
	for _, format := range []string{"beancount", "hledger"} {
		var out bytes.Buffer
		if err := export(&out, c.Books[0], format); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "1045 USD") || !strings.Contains(out.String(), "-1100 USD") {
			t.Fatal("export lost fixed balance assertions")
		}
	}
	var out bytes.Buffer
	if err := export(&out, c.Books[0], "unknown"); err == nil {
		t.Fatal("accepted unknown format")
	}
}

// FuzzDecimal exercises exact input parsing independently of any runtime money type.
func FuzzDecimal(f *testing.F) {
	for _, s := range []string{"0", "1.001", "99999999999999999999999999999.999999999", "NaN", "1e9", "-1", "0.0000000001"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, err := decimal(s, false)
		if err == nil && (!decimalPattern.MatchString(s) || v.Sign() < 0) {
			t.Fatal("accepted invalid unsigned decimal")
		}
		if decimalPattern.MatchString(s) && err != nil {
			t.Fatal("rejected finite bounded decimal")
		}
	})
}
