// Package categorization computes inert, bounded suggestions from already-authorized inputs.
// It performs no I/O and cannot select records, grant access, persist labels or post journals.
package categorization

import (
	"errors"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrInput = errors.New("categorization input unavailable")

// Scope pins the private owner and workspace; equality is a guard, never authorization.
type Scope struct{ Principal, Workspace string }

// Output keeps explicit Uncategorized separate from a missing classification.
type Output struct {
	TopicID       string
	Uncategorized bool
	Tags          []string
}

// Condition uses one closed field/operator pair; all rule conditions are conjunctive.
type Condition struct{ Field, Operator, Value string }

// Rule describes one approved immutable version. Empty Account means workspace-wide.
type Rule struct {
	Scope                 Scope
	ID, Revision, Account string
	Priority              int
	Conditions            []Condition
	Output                Output
}

// Catalog is the current authorized active taxonomy and immutable ruleset identity.
type Catalog struct {
	Scope                           Scope
	TaxonomyRevision, RulesRevision string
	Topics, Tags                    []string
}

// Fact contains only the safe normalized text needed for matching, never raw provider data.
type Fact struct {
	Scope                                                   Scope
	ID, Revision, Account, CurrencyVersion, Direction, Date string
	Merchant                                                *string
	Description                                             string
	Override                                                *Override
}

// Override pins an explicit owner decision to its source and taxonomy revision.
type Override struct {
	SourceRevision, TaxonomyRevision string
	Output                           Output
}

// Provenance identifies every agreeing or conflicting winner without an arbitrary tie break.
type Provenance struct{ ID, Revision string }

// Result is a suggestion or explicit abstention, with source and ruleset pins.
type Result struct {
	ID, Revision, TaxonomyRevision, RulesRevision string
	Status                                        string
	Output                                        *Output
	Rules                                         []Provenance
}

// compiledCondition retains the validated immutable regexp for shared evaluation.
type compiledCondition struct {
	Condition
	pattern *regexp.Regexp
}

// compiledRule carries one immutable rule and its explicit specificity tier.
type compiledRule struct {
	rule        Rule
	conditions  []compiledCondition
	specificity int
}

// Engine owns defensive copies, so callers cannot alter a compiled ruleset during evaluation.
type Engine struct {
	catalog Catalog
	rules   []compiledRule
}

// Compile validates finite work, scope, active outputs and the complete closed rule language.
func Compile(c Catalog, rules []Rule) (*Engine, error) {
	if !validScope(c.Scope) || !identifier(c.TaxonomyRevision) || !identifier(c.RulesRevision) ||
		len(rules) > 100 || !validIDs(c.Topics, 500) || !validIDs(c.Tags, 500) {
		return nil, ErrInput
	}
	c.Topics = slices.Clone(c.Topics)
	c.Tags = slices.Clone(c.Tags)
	e := &Engine{catalog: c}
	seen := map[string]bool{}
	for _, r := range rules {
		if r.Scope != c.Scope || !identifier(r.ID) || !identifier(r.Revision) || seen[r.ID] ||
			(r.Account != "" && !identifier(r.Account)) || r.Priority < -1000000 || r.Priority > 1000000 ||
			len(r.Conditions) < 1 || len(r.Conditions) > 8 {
			return nil, ErrInput
		}
		seen[r.ID] = true
		out, ok := normalize(r.Output, c)
		if !ok {
			return nil, ErrInput
		}
		r.Output = out
		r.Conditions = slices.Clone(r.Conditions)
		cr := compiledRule{rule: r}
		for _, cond := range r.Conditions {
			cc, rank, ok := compileCondition(cond)
			if !ok {
				return nil, ErrInput
			}
			cr.conditions = append(cr.conditions, cc)
			if rank > cr.specificity {
				cr.specificity = rank
			}
		}
		e.rules = append(e.rules, cr)
	}
	return e, nil
}

// Evaluate rejects the whole batch on invalid input; it never expands the explicit selection.
func (e *Engine) Evaluate(facts []Fact) ([]Result, error) {
	if e == nil || len(facts) > 100 {
		return nil, ErrInput
	}
	seen := map[string]bool{}
	for _, f := range facts {
		if f.Scope != e.catalog.Scope || !identifier(f.ID) || !identifier(f.Revision) || seen[f.ID] ||
			!identifier(f.Account) || !identifier(f.CurrencyVersion) || !date(f.Date) ||
			(f.Direction != "inflow" && f.Direction != "outflow") || !text(f.Description) ||
			(f.Merchant != nil && !text(*f.Merchant)) {
			return nil, ErrInput
		}
		seen[f.ID] = true
		if f.Override != nil {
			if !identifier(f.Override.SourceRevision) || !identifier(f.Override.TaxonomyRevision) {
				return nil, ErrInput
			}
			// A stale taxonomy may contain a retired output. Preserve its audit elsewhere,
			// abstain here, and never fall through to a broad rule.
			if f.Override.TaxonomyRevision == e.catalog.TaxonomyRevision {
				if _, ok := normalize(f.Override.Output, e.catalog); !ok {
					return nil, ErrInput
				}
			}
		}
	}
	results := make([]Result, 0, len(facts))
	for _, f := range facts {
		results = append(results, e.evaluate(f))
	}
	return results, nil
}

// evaluate ranks scope before priority and specificity; tied disagreement always abstains.
func (e *Engine) evaluate(f Fact) Result {
	r := Result{ID: f.ID, Revision: f.Revision, TaxonomyRevision: e.catalog.TaxonomyRevision, RulesRevision: e.catalog.RulesRevision, Status: "unclassified"}
	if o := f.Override; o != nil {
		if o.SourceRevision != f.Revision || o.TaxonomyRevision != e.catalog.TaxonomyRevision {
			r.Status = "stale_override"
			return r
		}
		out, _ := normalize(o.Output, e.catalog)
		r.Output = &out
		r.Status = "override"
		return r
	}
	var winners []compiledRule
	for _, rule := range e.rules {
		if rule.rule.Account != "" && rule.rule.Account != f.Account {
			continue
		}
		matched := true
		for _, cond := range rule.conditions {
			if !cond.matches(f) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		comparison := 1
		if len(winners) > 0 {
			comparison = compareRank(rule, winners[0])
		}
		if comparison > 0 {
			winners = []compiledRule{rule}
		} else if comparison == 0 {
			winners = append(winners, rule)
		}
	}
	if len(winners) == 0 {
		return r
	}
	slices.SortFunc(winners, func(a, b compiledRule) int { return strings.Compare(a.rule.ID, b.rule.ID) })
	out := copyOutput(winners[0].rule.Output)
	r.Status = "rule"
	for _, w := range winners {
		r.Rules = append(r.Rules, Provenance{w.rule.ID, w.rule.Revision})
		if !equalOutput(out, w.rule.Output) {
			r.Status = "conflict"
		}
	}
	if r.Status != "conflict" {
		r.Output = &out
	}
	return r
}

// compileCondition checks syntax and expanded instruction work before regexp compilation.
func compileCondition(c Condition) (compiledCondition, int, bool) {
	cc := compiledCondition{Condition: c}
	if !utf8.ValidString(c.Value) || len(c.Value) == 0 || len(c.Value) > 128 {
		return cc, 0, false
	}
	switch c.Field {
	case "merchant", "description":
		switch c.Operator {
		case "exact", "contains":
		case "regex":
			parsed, err := syntax.Parse(c.Value, syntax.Perl)
			if err != nil {
				return cc, 0, false
			}
			// Bound expansion before Simplify allocates repeated subexpressions.
			if regexWork(parsed) > 1024 {
				return cc, 0, false
			}
			program, err := syntax.Compile(parsed.Simplify())
			if err != nil || len(program.Inst) > 1024 {
				return cc, 0, false
			}
			cc.pattern, err = regexp.Compile(c.Value)
			if err != nil {
				return cc, 0, false
			}
		default:
			return cc, 0, false
		}
	case "account", "currency_version":
		if c.Operator != "exact" || !identifier(c.Value) {
			return cc, 0, false
		}
	case "direction":
		if c.Operator != "exact" || (c.Value != "inflow" && c.Value != "outflow") {
			return cc, 0, false
		}
	case "date":
		if (c.Operator != "on_or_after" && c.Operator != "on_or_before") || !date(c.Value) {
			return cc, 0, false
		}
	default:
		return cc, 0, false
	}
	rank := 0
	if c.Operator == "regex" {
		rank = 1
	}
	if c.Field == "description" && (c.Operator == "contains" || c.Operator == "exact") {
		rank = 2
	}
	if c.Field == "merchant" && c.Operator == "contains" {
		rank = 3
	}
	if c.Field == "merchant" && c.Operator == "exact" {
		rank = 4
	}
	return cc, rank, true
}

// regexWork saturates a conservative expansion estimate before nested repeats allocate.
func regexWork(r *syntax.Regexp) int {
	const limit = 1025
	add := func(a, b int) int {
		if a >= limit || b >= limit || a+b >= limit {
			return limit
		}
		return a + b
	}
	if r.Op == syntax.OpLiteral {
		return add(1, len(r.Rune))
	}
	work := 1
	for _, child := range r.Sub {
		work = add(work, regexWork(child))
	}
	if r.Op == syntax.OpCapture {
		work = add(work, 2)
	}
	if r.Op == syntax.OpRepeat {
		n := r.Max
		if n < 0 {
			n = r.Min + 1
		}
		if n > 0 && work > 1024/n {
			return limit
		}
		work = add(1, work*n)
	}
	return work
}

// matches treats absent merchant as unavailable, never as a match against an empty string.
func (c compiledCondition) matches(f Fact) bool {
	v := ""
	switch c.Field {
	case "merchant":
		if f.Merchant == nil {
			return false
		}
		v = *f.Merchant
	case "description":
		v = f.Description
	case "account":
		v = f.Account
	case "currency_version":
		v = f.CurrencyVersion
	case "direction":
		v = f.Direction
	case "date":
		v = f.Date
	}
	switch c.Operator {
	case "exact":
		return v == c.Value
	case "contains":
		return strings.Contains(v, c.Value)
	case "regex":
		return c.pattern.MatchString(v)
	case "on_or_after":
		return v >= c.Value
	case "on_or_before":
		return v <= c.Value
	}
	return false
}

// compareRank never uses IDs or condition count to infer semantic precedence.
func compareRank(a, b compiledRule) int {
	as, bs := 0, 0
	if a.rule.Account != "" {
		as = 1
	}
	if b.rule.Account != "" {
		bs = 1
	}
	for _, pair := range [][2]int{{as, bs}, {a.rule.Priority, b.rule.Priority}, {a.specificity, b.specificity}} {
		if pair[0] > pair[1] {
			return 1
		}
		if pair[0] < pair[1] {
			return -1
		}
	}
	return 0
}

// normalize verifies active taxonomy identity and compares tag sets independently of order.
func normalize(o Output, c Catalog) (Output, bool) {
	if o.Uncategorized == (o.TopicID != "") || (!o.Uncategorized && !slices.Contains(c.Topics, o.TopicID)) || !validIDs(o.Tags, 20) {
		return Output{}, false
	}
	for _, id := range o.Tags {
		if !slices.Contains(c.Tags, id) {
			return Output{}, false
		}
	}
	o = copyOutput(o)
	slices.Sort(o.Tags)
	return o, true
}

// copyOutput prevents mutation of the engine through a returned tag slice.
func copyOutput(o Output) Output { o.Tags = slices.Clone(o.Tags); return o }

// equalOutput compares normalized sets, retaining explicit Uncategorized semantics.
func equalOutput(a, b Output) bool {
	return a.TopicID == b.TopicID && a.Uncategorized == b.Uncategorized && slices.Equal(a.Tags, b.Tags)
}

// validScope checks finite opaque identities; callers must separately prove current authority.
func validScope(s Scope) bool { return identifier(s.Principal) && identifier(s.Workspace) }

// identifier bounds opaque equality keys without treating their shape as a grant.
func identifier(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == ':' || c == '.') {
			return false
		}
	}
	return true
}

// validIDs rejects duplicate or unbounded taxonomy/tag identities.
func validIDs(ids []string, max int) bool {
	if len(ids) > max {
		return false
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !identifier(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

// text rejects malformed or oversized source strings instead of returning a false no-match.
func text(s string) bool {
	return utf8.ValidString(s) && len(s) <= 2000 && utf8.RuneCountInString(s) <= 500
}

// date accepts only actual canonical source dates, enabling exact inclusive lexical comparisons.
func date(s string) bool {
	d, err := time.Parse("2006-01-02", s)
	return err == nil && d.Year() >= 1 && d.Format("2006-01-02") == s
}
