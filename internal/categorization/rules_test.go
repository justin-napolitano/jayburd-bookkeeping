package categorization

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// catalog supplies independent fictional active topic identities, not accounting categories.
func catalog() Catalog {
	return Catalog{Scope: Scope{"owner-a", "workspace-a"}, TaxonomyRevision: "3", RulesRevision: "4", Topics: []string{"dining", "travel", "groceries"}, Tags: []string{"personal", "recurring"}}
}

// fact is an original-unit-free safe text projection; the evaluator cannot change money.
func fact() Fact {
	merchant := "Fictional Cafe"
	return Fact{Scope: catalog().Scope, ID: "o1", Revision: "2", Account: "checking", CurrencyVersion: "usd-a", Direction: "outflow", Date: "2026-10-01", Merchant: &merchant, Description: "Fictional Cafe purchase"}
}

// rule makes the exact-merchant oracle from433's independently written fixed cases.
func rule(id, topic string) Rule {
	return Rule{Scope: catalog().Scope, ID: id, Revision: "1", Conditions: []Condition{{"merchant", "exact", "Fictional Cafe"}}, Output: Output{TopicID: topic}}
}

// result compiles a fictional snapshot and requires one complete evaluation result.
func result(t *testing.T, rules []Rule, f Fact) Result {
	t.Helper()
	e, err := Compile(catalog(), rules)
	if err != nil {
		t.Fatal(err)
	}
	r, err := e.Evaluate([]Fact{f})
	if err != nil || len(r) != 1 {
		t.Fatalf("evaluation: %v", err)
	}
	return r[0]
}

// TestRulePrecedence preserves explicit manual choices over matching approved rules.
func TestRulePrecedence(t *testing.T) {
	f := fact()
	f.Override = &Override{SourceRevision: "2", TaxonomyRevision: "3", Output: Output{TopicID: "groceries"}}
	r := result(t, []Rule{rule("a", "dining")}, f)
	if r.Status != "override" || r.Output.TopicID != "groceries" || len(r.Rules) != 0 {
		t.Fatalf("unexpected result: %+v", r)
	}
}

// TestAccountBeforeWorkspaceRule makes account scope outrank even higher workspace priority.
func TestAccountBeforeWorkspaceRule(t *testing.T) {
	a, b := rule("a", "travel"), rule("b", "dining")
	a.Account = "checking"
	b.Priority = 999
	r := result(t, []Rule{b, a}, fact())
	if r.Status != "rule" || r.Output.TopicID != "travel" {
		t.Fatal(r)
	}
	f := fact()
	f.Account = "savings"
	r = result(t, []Rule{a, b}, f)
	if r.Output.TopicID != "dining" {
		t.Fatal(r)
	}
}

// TestRuleSpecificity checks exact, literal and regex rank while priority remains explicit.
func TestRuleSpecificity(t *testing.T) {
	a, b := rule("a", "dining"), rule("b", "travel")
	b.Conditions[0].Operator = "contains"
	r := result(t, []Rule{b, a}, fact())
	if r.Output.TopicID != "dining" {
		t.Fatal(r)
	}
	b.Priority = 1
	r = result(t, []Rule{a, b}, fact())
	if r.Output.TopicID != "travel" {
		t.Fatal(r)
	}
	b.Priority = 0
	b.Conditions = []Condition{{"description", "contains", "Cafe"}}
	a.Conditions = []Condition{{"description", "regex", "Cafe"}}
	r = result(t, []Rule{a, b}, fact())
	if r.Output.TopicID != "travel" {
		t.Fatal(r)
	}
}

// TestRuleConflictAbstains forbids input-order and UUID tie-breaking of semantic disagreement.
func TestRuleConflictAbstains(t *testing.T) {
	a, b := rule("z", "dining"), rule("a", "travel")
	for _, rules := range [][]Rule{{a, b}, {b, a}} {
		r := result(t, rules, fact())
		if r.Status != "conflict" || r.Output != nil || !reflect.DeepEqual(r.Rules, []Provenance{{"a", "1"}, {"z", "1"}}) {
			t.Fatal(r)
		}
	}
}

// TestRuleAgreement compares tag sets and emits all agreeing immutable rule identities.
func TestRuleAgreement(t *testing.T) {
	a, b := rule("a", "dining"), rule("b", "dining")
	a.Output.Tags = []string{"personal", "recurring"}
	b.Output.Tags = []string{"recurring", "personal"}
	r := result(t, []Rule{b, a}, fact())
	if r.Status != "rule" || len(r.Rules) != 2 || !reflect.DeepEqual(r.Output.Tags, []string{"personal", "recurring"}) {
		t.Fatal(r)
	}
	b.Output.Tags = nil
	r = result(t, []Rule{a, b}, fact())
	if r.Status != "conflict" {
		t.Fatal(r)
	}
}

// TestOwnerRuleScope rejects foreign scope before exposing even no-match results.
func TestOwnerRuleScope(t *testing.T) {
	r := rule("a", "dining")
	r.Scope.Principal = "owner-b"
	if _, err := Compile(catalog(), []Rule{r}); err != ErrInput {
		t.Fatal(err)
	}
	e, _ := Compile(catalog(), []Rule{rule("a", "dining")})
	for _, scope := range []Scope{{"owner-b", "workspace-a"}, {"owner-a", "workspace-b"}} {
		f := fact()
		f.Scope = scope
		if v, err := e.Evaluate([]Fact{fact(), f}); err != ErrInput || v != nil {
			t.Fatal("foreign or partial result", err)
		}
	}
}

// TestOverridePreserved never replaces a stale override with a fresh broad suggestion.
func TestOverridePreserved(t *testing.T) {
	for _, o := range []Override{{"1", "3", Output{TopicID: "groceries"}}, {"2", "2", Output{TopicID: "retired"}}} {
		f := fact()
		f.Override = &o
		r := result(t, []Rule{rule("a", "dining")}, f)
		if r.Status != "stale_override" || r.Output != nil || len(r.Rules) != 0 {
			t.Fatal(r)
		}
	}
}

// TestRuleConditions proves conjunctive direction/date/currency/account bounds and missing merchants.
func TestRuleConditions(t *testing.T) {
	r := rule("a", "dining")
	r.Conditions = append(r.Conditions, Condition{"date", "on_or_after", "2026-10-01"}, Condition{"date", "on_or_before", "2026-10-01"}, Condition{"currency_version", "exact", "usd-a"}, Condition{"direction", "exact", "outflow"}, Condition{"account", "exact", "checking"})
	if v := result(t, []Rule{r}, fact()); v.Status != "rule" {
		t.Fatal(v)
	}
	for _, edit := range []func(*Fact){func(f *Fact) { f.Date = "2026-09-30" }, func(f *Fact) { f.Date = "2026-10-02" }, func(f *Fact) { f.CurrencyVersion = "jpy-a" }, func(f *Fact) { f.Direction = "inflow" }, func(f *Fact) { f.Account = "savings" }, func(f *Fact) { f.Merchant = nil }, func(f *Fact) { m := "fictional cafe"; f.Merchant = &m }} {
		f := fact()
		edit(&f)
		if v := result(t, []Rule{r}, f); v.Status != "unclassified" {
			t.Fatal(v)
		}
	}
}

// TestRegexWorkBound rejects invalid languages and expansion before compilation allocates repeats.
func TestRegexWorkBound(t *testing.T) {
	for _, p := range []string{`(a)\1`, strings.Repeat("x", 129), `(a{1000}){1000}`, `(?:a{999}|b{999})+`, `(?=a)`, string([]byte{0xff})} {
		r := rule("a", "dining")
		r.Conditions = []Condition{{"description", "regex", p}}
		if _, err := Compile(catalog(), []Rule{r}); err != ErrInput {
			t.Fatalf("accepted invalid/oversize regex")
		}
	}
	r := rule("a", "dining")
	r.Conditions = []Condition{{"description", "regex", `(?i)fictional (cafe|market)`}}
	if v := result(t, []Rule{r}, fact()); v.Status != "rule" {
		t.Fatal(v)
	}
	e, _ := Compile(catalog(), nil)
	for _, text := range []string{strings.Repeat("x", 501), strings.Repeat("界", 501), string([]byte{0xff})} {
		f := fact()
		f.Description = text
		if v, err := e.Evaluate([]Fact{f}); err != ErrInput || v != nil {
			t.Fatal("oversize was false no-match")
		}
	}
	if _, err := Compile(catalog(), make([]Rule, 101)); err != ErrInput {
		t.Fatal(err)
	}
	if v, err := e.Evaluate(make([]Fact, 101)); err != ErrInput || v != nil {
		t.Fatal(err)
	}
}

// TestRuleInputClosed denies unknown operators, inactive outputs, duplicates and malformed dates.
func TestRuleInputClosed(t *testing.T) {
	for _, edit := range []func(*Rule){func(r *Rule) { r.Conditions[0].Operator = "script" }, func(r *Rule) { r.Conditions[0].Field = "raw_payload" }, func(r *Rule) { r.Output.TopicID = "inactive" }, func(r *Rule) { r.Output.Tags = []string{"foreign"} }, func(r *Rule) { r.Output.Tags = []string{"personal", "personal"} }, func(r *Rule) { r.Output.Uncategorized = true }, func(r *Rule) { r.Conditions = nil }, func(r *Rule) { r.Priority = 1000001 }} {
		r := rule("a", "dining")
		edit(&r)
		if _, err := Compile(catalog(), []Rule{r}); err != ErrInput {
			t.Fatal("accepted invalid rule")
		}
	}
	r := rule("a", "dining")
	if _, err := Compile(catalog(), []Rule{r, r}); err != ErrInput {
		t.Fatal("duplicate identity")
	}
	f := fact()
	f.Date = "2026-02-30"
	e, _ := Compile(catalog(), nil)
	if _, err := e.Evaluate([]Fact{f}); err != ErrInput {
		t.Fatal("invalid date")
	}
	f = fact()
	if _, err := e.Evaluate([]Fact{f, f}); err != ErrInput {
		t.Fatal("duplicate observation")
	}
	r.Output = Output{Uncategorized: true}
	v := result(t, []Rule{r}, fact())
	if !v.Output.Uncategorized {
		t.Fatal(v)
	}
}

// TestCompiledRuleImmutability proves caller and output slices cannot alter a future result.
func TestCompiledRuleImmutability(t *testing.T) {
	c := catalog()
	r := rule("a", "dining")
	r.Output.Tags = []string{"personal"}
	e, err := Compile(c, []Rule{r})
	if err != nil {
		t.Fatal(err)
	}
	c.Topics[0] = "changed"
	r.Conditions[0].Value = "changed"
	r.Output.Tags[0] = "changed"
	v, err := e.Evaluate([]Fact{fact()})
	if err != nil || v[0].Status != "rule" || v[0].Output.Tags[0] != "personal" {
		t.Fatal(v, err)
	}
	v[0].Output.Tags[0] = "changed"
	v[0].Output.TopicID = "changed"
	v, err = e.Evaluate([]Fact{fact()})
	if err != nil || v[0].Output.TopicID != "dining" || v[0].Output.Tags[0] != "personal" {
		t.Fatal(v, err)
	}
}

// TestConcurrentEvaluation checks the same immutable compiled ruleset under race detection.
func TestConcurrentEvaluation(t *testing.T) {
	e, _ := Compile(catalog(), []Rule{rule("a", "dining")})
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 50 {
				v, err := e.Evaluate([]Fact{fact()})
				if err != nil || v[0].Output.TopicID != "dining" {
					t.Error("nondeterministic result")
				}
			}
		})
	}
	wg.Wait()
}

// FuzzBoundedRegex checks arbitrary bounded patterns and deterministic repeated evaluation.
func FuzzBoundedRegex(f *testing.F) {
	for _, p := range []string{"Cafe", `(a+)+$`, `(a{1000}){1000}`, `(?i)cafe`, `(`} {
		f.Add(p, "Fictional Cafe purchase")
	}
	f.Fuzz(func(t *testing.T, p, text string) {
		r := rule("a", "dining")
		r.Conditions = []Condition{{"description", "regex", p}}
		e, err := Compile(catalog(), []Rule{r})
		if err != nil {
			return
		}
		x := fact()
		x.Description = text
		a, ea := e.Evaluate([]Fact{x})
		b, eb := e.Evaluate([]Fact{x})
		if ea != eb || !reflect.DeepEqual(a, b) {
			t.Fatal("nondeterministic evaluation")
		}
	})
}

// cancellationStep deterministically cancels at a work boundary without timing assumptions.
type cancellationStep struct {
	context.Context
	remaining int
}

func (c *cancellationStep) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		return context.Canceled
	}
	return nil
}

// TestRuleContextCompatibility preserves wrappers and rejects partial work on cancellation.
func TestRuleContextCompatibility(t *testing.T) {
	rules := []Rule{rule("a", "dining"), rule("b", "travel")}
	plain, e := Compile(catalog(), rules)
	if e != nil {
		t.Fatal(e)
	}
	aware, e := CompileContext(context.Background(), catalog(), rules)
	if e != nil {
		t.Fatal(e)
	}
	want, e := plain.Evaluate([]Fact{fact()})
	if e != nil {
		t.Fatal(e)
	}
	got, e := aware.EvaluateContext(context.Background(), []Fact{fact()})
	if e != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("context changed matching")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if engine, e := CompileContext(canceled, catalog(), rules); engine != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("compile ignored pre-cancellation")
	}
	if out, e := aware.EvaluateContext(canceled, []Fact{fact()}); out != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("evaluation ignored pre-cancellation")
	}
	if engine, e := CompileContext(&cancellationStep{Context: context.Background(), remaining: 4}, catalog(), rules); engine != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("compile returned partial engine")
	}
	facts := []Fact{fact(), fact()}
	facts[1].ID = "o2"
	if out, e := aware.EvaluateContext(&cancellationStep{Context: context.Background(), remaining: 7}, facts); out != nil || !errors.Is(e, context.Canceled) {
		t.Fatal("evaluation returned partial results")
	}
}
