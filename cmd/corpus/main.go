// Command corpus checks original synthetic accounting examples and exports their
// functional-currency projection. It is a fixture oracle, never a posting engine.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Corpus records provenance and independently authored expected balances.
type Corpus struct {
	Version int      `json:"version"`
	ID      string   `json:"id"`
	Policy  string   `json:"policy"`
	Sources []string `json:"sources"`
	Books   []Book   `json:"books"`
}

// Book confines fixture identities and expected quantities to one accounting owner.
type Book struct {
	ID       string                       `json:"id"`
	Entity   string                       `json:"entity"`
	Currency string                       `json:"functional_currency"`
	Accounts []string                     `json:"accounts"`
	Entries  []Entry                      `json:"entries"`
	Expected map[string]string            `json:"expected_balances"`
	Units    map[string]map[string]string `json:"expected_units"`
}

// Entry is synthetic historical evidence, including proposed future FX cases.
type Entry struct {
	ID       string `json:"id"`
	Scenario string `json:"scenario"`
	Date     string `json:"date"`
	Kind     string `json:"kind"`
	Reverses string `json:"reverses,omitempty"`
	Lines    []Line `json:"lines"`
}

// Line preserves original units separately from booked functional value.
type Line struct {
	Account    string `json:"account"`
	Side       string `json:"side"`
	Currency   string `json:"currency"`
	Amount     string `json:"amount"`
	Functional string `json:"functional_amount"`
}

var decimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,28})(\.[0-9]{1,9})?$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
var accountPattern = regexp.MustCompile(`^(Assets|Liabilities|Equity|Income|Expenses)(:[A-Z][A-Za-z0-9]*)+$`)

// decimal rejects lossy or special syntax before exact rational conversion.
func decimal(s string, signed bool) (*big.Rat, error) {
	raw := s
	if signed {
		raw = strings.TrimPrefix(raw, "-")
	}
	if !decimalPattern.MatchString(raw) {
		return nil, fmt.Errorf("invalid decimal %q", s)
	}
	v, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid decimal %q", s)
	}
	return v, nil
}

// decode accepts only the documented fixture shape and one JSON document.
func decode(r io.Reader) (Corpus, error) {
	var c Corpus
	data, err := io.ReadAll(io.LimitReader(r, (8<<20)+1))
	if err != nil {
		return c, err
	}
	if len(data) > 8<<20 {
		return c, fmt.Errorf("fixture exceeds 8 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("trailing fixture content")
	}
	return c, nil
}

// add accumulates exact signed values without mutating the caller's quantity.
func add(m map[string]*big.Rat, key string, amount *big.Rat) {
	if m[key] == nil {
		m[key] = new(big.Rat)
	}
	m[key].Add(m[key], amount)
}

// validate checks fixed fixture oracles; it does not enforce runtime permissions,
// selected GAAP policy, period locking, rate selection or durable idempotency.
func validate(c Corpus) error {
	if c.Version != 1 || c.ID == "" || c.Policy == "" || len(c.Sources) == 0 || len(c.Books) == 0 {
		return fmt.Errorf("missing corpus metadata")
	}
	books := map[string]bool{}
	for _, b := range c.Books {
		if b.ID == "" || b.Entity == "" || books[b.ID] || !currencyPattern.MatchString(b.Currency) {
			return fmt.Errorf("invalid/duplicate book")
		}
		books[b.ID] = true
		accounts := map[string]bool{}
		for _, a := range b.Accounts {
			if accounts[a] || !accountPattern.MatchString(a) {
				return fmt.Errorf("invalid/duplicate account %s", a)
			}
			accounts[a] = true
		}
		if len(accounts) == 0 || len(b.Entries) == 0 || len(b.Expected) != len(accounts) || len(b.Units) != len(accounts) {
			return fmt.Errorf("incomplete oracle for %s", b.ID)
		}
		balances := map[string]*big.Rat{}
		units := map[string]map[string]*big.Rat{}
		entries := map[string]Entry{}
		reversed := map[string]bool{}
		for _, e := range b.Entries {
			if _, err := time.Parse("2006-01-02", e.Date); err != nil {
				return err
			}
			if e.ID == "" || e.Scenario == "" || len(e.Lines) < 2 || len(e.Lines) > 1000 {
				return fmt.Errorf("invalid entry %s", e.ID)
			}
			if _, exists := entries[e.ID]; exists {
				return fmt.Errorf("duplicate entry %s", e.ID)
			}
			switch e.Kind {
			case "manual", "reversal", "remeasurement", "settlement":
			default:
				return fmt.Errorf("unknown entry kind")
			}
			sum := new(big.Rat)
			for _, l := range e.Lines {
				if !accounts[l.Account] || !currencyPattern.MatchString(l.Currency) {
					return fmt.Errorf("invalid scoped account/currency in %s", e.ID)
				}
				if l.Side != "debit" && l.Side != "credit" {
					return fmt.Errorf("invalid side")
				}
				a, err := decimal(l.Amount, false)
				if err != nil {
					return err
				}
				f, err := decimal(l.Functional, false)
				if err != nil {
					return err
				}
				if f.Sign() <= 0 || (a.Sign() == 0 && e.Kind != "remeasurement") {
					return fmt.Errorf("invalid zero amount in %s", e.ID)
				}
				if l.Currency == b.Currency && a.Cmp(f) != 0 {
					return fmt.Errorf("identity amount mismatch in %s", e.ID)
				}
				if l.Side == "credit" {
					a.Neg(a)
					f.Neg(f)
				}
				sum.Add(sum, f)
				add(balances, l.Account, f)
				if units[l.Account] == nil {
					units[l.Account] = map[string]*big.Rat{}
				}
				add(units[l.Account], l.Currency, a)
			}
			if sum.Sign() != 0 {
				return fmt.Errorf("unbalanced entry %s", e.ID)
			}
			if e.Kind == "reversal" {
				original, ok := entries[e.Reverses]
				if !ok || original.Kind == "reversal" || reversed[e.Reverses] || len(original.Lines) != len(e.Lines) {
					return fmt.Errorf("invalid reversal link")
				}
				for i, l := range e.Lines {
					o := original.Lines[i]
					if l.Account != o.Account || l.Side == o.Side || l.Currency != o.Currency || l.Amount != o.Amount || l.Functional != o.Functional {
						return fmt.Errorf("reversal changed historical amounts")
					}
				}
				reversed[e.Reverses] = true
			} else if e.Reverses != "" {
				return fmt.Errorf("unexpected reversal link")
			}
			entries[e.ID] = e
		}
		for a := range accounts {
			s, ok := b.Expected[a]
			if !ok {
				return fmt.Errorf("missing expected balance %s", a)
			}
			expected, err := decimal(s, true)
			if err != nil {
				return err
			}
			actual := balances[a]
			if actual == nil {
				actual = new(big.Rat)
			}
			if expected.Cmp(actual) != 0 {
				return fmt.Errorf("balance mismatch %s/%s: got %s want %s", b.ID, a, actual.FloatString(9), s)
			}
			wants, ok := b.Units[a]
			if !ok || len(wants) == 0 {
				return fmt.Errorf("missing expected units %s", a)
			}
			for currency, value := range wants {
				if !currencyPattern.MatchString(currency) {
					return fmt.Errorf("invalid expected currency")
				}
				expected, err := decimal(value, true)
				if err != nil {
					return err
				}
				actual := units[a][currency]
				if actual == nil {
					actual = new(big.Rat)
				}
				if expected.Cmp(actual) != 0 {
					return fmt.Errorf("unit mismatch %s/%s", a, currency)
				}
			}
			for currency := range units[a] {
				if _, ok := wants[currency]; !ok {
					return fmt.Errorf("unasserted currency %s", currency)
				}
			}
		}
	}
	return nil
}

// export writes a compatible functional-currency projection with balance assertions.
// Original FX quantities remain in JSON; this projection is not a full FX-policy oracle.
func export(w io.Writer, b Book, format string) error {
	if format != "beancount" && format != "hledger" {
		return fmt.Errorf("unknown export format")
	}
	accounts := append([]string(nil), b.Accounts...)
	sort.Strings(accounts)
	entries := append([]Entry(nil), b.Entries...)
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Date < entries[j].Date })
	if format == "beancount" {
		fmt.Fprintln(w, `option "title" "Jayburd synthetic comparison"`)
		for _, a := range accounts {
			fmt.Fprintf(w, "%s open %s %s\n", entries[0].Date, a, b.Currency)
		}
	} else {
		fmt.Fprintf(w, "commodity %s\n  format 1.000000000 %s\n\n", b.Currency, b.Currency)
	}
	for _, e := range entries {
		if format == "beancount" {
			fmt.Fprintf(w, "\n%s * %q\n", e.Date, e.ID)
		} else {
			fmt.Fprintf(w, "\n%s %s\n", e.Date, e.ID)
		}
		for _, l := range e.Lines {
			sign := ""
			if l.Side == "credit" {
				sign = "-"
			}
			fmt.Fprintf(w, "  %s  %s%s %s\n", l.Account, sign, l.Functional, b.Currency)
		}
	}
	last, _ := time.Parse("2006-01-02", entries[len(entries)-1].Date)
	asof := last.AddDate(0, 0, 1).Format("2006-01-02")
	if format == "beancount" {
		for _, a := range accounts {
			fmt.Fprintf(w, "%s balance %s %s %s\n", asof, a, b.Expected[a], b.Currency)
		}
	} else {
		for _, a := range accounts {
			fmt.Fprintf(w, "\n%s assert %s\n  %s  0 %s = %s %s\n", asof, a, a, b.Currency, b.Expected[a], b.Currency)
		}
	}
	return nil
}

// run keeps the fixture CLI offline and makes exports opt-in, scoped to one book.
func run(args []string, out io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: corpus check FILE | export FILE BOOK beancount|hledger | audit-generator CSV")
	}
	f, err := os.Open(args[1])
	if err != nil {
		return err
	}
	defer f.Close()
	if args[0] == "audit-generator" && len(args) == 2 {
		return auditGenerator(f, out)
	}
	c, err := decode(f)
	if err != nil {
		return err
	}
	if err = validate(c); err != nil {
		return err
	}
	if args[0] == "check" && len(args) == 2 {
		fmt.Fprintf(out, "PASS: %s (%d synthetic books); fixture arithmetic only\n", c.ID, len(c.Books))
		return nil
	}
	if args[0] == "export" && len(args) == 4 {
		for _, b := range c.Books {
			if b.ID == args[2] {
				return export(out, b, args[3])
			}
		}
		return fmt.Errorf("unknown book")
	}
	return fmt.Errorf("invalid corpus command")
}

// main reports failure without claiming that fixture checks execute a ledger.
func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
