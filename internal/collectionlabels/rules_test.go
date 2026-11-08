package collectionlabels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// ruleUnitContext is a fixed authorized-context oracle; unit scope equality is not database authority.
func ruleUnitContext() ruleContext {
	var c ruleContext
	c.Principal = "owner-a"
	c.Workspace = "workspace-a"
	c.Snapshot.Taxonomy = "2"
	c.Snapshot.Tags = "3"
	c.Snapshot.Rules = "4"
	c.Topics = []string{"dining", "travel"}
	c.Tags = []string{"tag-rule"}
	currency, direction := "dated-usd", "outflow"
	c.Rules = []storedRule{{ID: "rule-a", Status: "active", Conditions: []ruleCondition{{Field: "merchant", Operator: "exact", Value: "Fictional Cafe"}}, Output: ruleOutput{Kind: "topic", TopicID: "dining", TagIDs: []string{"tag-rule"}}}}
	c.Facts = []ruleFact{{Source: "source-a", Observation: "observation-a", Revision: "5", Date: "2026-10-01", Merchant: ptrRule("Fictional Cafe"), Description: "Fictional purchase", Currency: &currency, Direction: &direction, SourceState: "booked", MoneyState: "available"}}
	return c
}
func ptrRule(s string) *string { return &s }

// TestRule46FieldwiseModel covers protected fields, stale precedence, conflict and unavailable fallthrough.
func TestRule46FieldwiseModel(t *testing.T) {
	for _, tc := range []struct {
		name          string
		modify        func(*ruleContext)
		status, topic string
		tags          int
	}{
		{"new rule", func(c *ruleContext) {}, "rule", "dining", 1},
		{"manual topic only", func(c *ruleContext) {
			c.Facts[0].Topic = &manualRuleTopic{SourceRevision: "5", TaxonomyRevision: "2", Kind: "topic", TopicID: "travel"}
		}, "rule", "travel", 1},
		{"manual 25 tags", func(c *ruleContext) {
			ids := []string{}
			for i := 0; i < 25; i++ {
				ids = append(ids, fmt.Sprintf("manual-tag-%d", i))
			}
			c.Facts[0].Tags = &manualRuleTags{SourceRevision: "5", TagRevision: "3", TagIDs: ids}
		}, "rule", "dining", 25},
		{"both manual", func(c *ruleContext) {
			c.Facts[0].Topic = &manualRuleTopic{SourceRevision: "5", TaxonomyRevision: "2", Kind: "uncategorized"}
			c.Facts[0].Tags = &manualRuleTags{SourceRevision: "5", TagRevision: "3", TagIDs: []string{}}
		}, "override", "", 0},
		{"stale source", func(c *ruleContext) {
			c.Facts[0].Topic = &manualRuleTopic{SourceRevision: "4", TaxonomyRevision: "2", Kind: "topic", TopicID: "travel"}
		}, "stale_override", "", 0},
		{"stale taxonomy", func(c *ruleContext) {
			c.Facts[0].Topic = &manualRuleTopic{SourceRevision: "5", TaxonomyRevision: "1", Kind: "topic", TopicID: "travel"}
		}, "stale_override", "", 0},
		{"stale tags", func(c *ruleContext) {
			c.Facts[0].Tags = &manualRuleTags{SourceRevision: "5", TagRevision: "2", TagIDs: []string{}}
		}, "stale_override", "", 0},
		{"held", func(c *ruleContext) { c.Facts[0].MoneyState = "held"; c.Facts[0].Currency = nil }, "unavailable", "", 0},
		{"pending", func(c *ruleContext) { c.Facts[0].SourceState = "pending" }, "unavailable", "", 0},
		{"zero", func(c *ruleContext) { c.Facts[0].Direction = ptrRule("zero") }, "unavailable", "", 0},
		{"missing merchant", func(c *ruleContext) { c.Facts[0].Merchant = nil }, "unclassified", "", 0},
		{"retired active output", func(c *ruleContext) { c.Topics = []string{"travel"} }, "unavailable", "", 0},
		{"equal rank disagreement", func(c *ruleContext) {
			r := c.Rules[0]
			r.ID = "rule-b"
			r.Output.TopicID = "travel"
			c.Rules = append(c.Rules, r)
		}, "conflict", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := ruleUnitContext()
			tc.modify(&c)
			r, e := evaluateRuleContext(context.Background(), c, false)
			if e != nil {
				t.Fatal(e)
			}
			if len(r) != 1 || r[0].Status != tc.status || r[0].WillApply {
				t.Fatal("classification or inertness")
			}
			if tc.status == "rule" || tc.status == "override" {
				if r[0].Output == nil || r[0].Output.TopicID != tc.topic || len(r[0].Output.TagIDs) != tc.tags {
					t.Fatal("fieldwise override changed")
				}
			} else if r[0].Output != nil {
				t.Fatal("abstention produced output")
			}
		})
	}
	c := ruleUnitContext()
	c.Rules[0].Conditions[0] = ruleCondition{Field: "merchant", Operator: "regex", Value: "["}
	_, e := evaluateRuleContext(context.Background(), c, true)
	var f *ledger.Failure
	if !errors.As(e, &f) || f.Status != 400 || f.Code != "invalid_collection" {
		t.Fatal("invalid new regex not definite")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e = evaluateRuleContext(ctx, ruleUnitContext(), false)
	if e == nil {
		t.Fatal("cancelled work continued")
	}
}

// TestRule46AdmittedDeadline exercises all catalog/condition/fact caps without a backtracking engine.
func TestRule46AdmittedDeadline(t *testing.T) {
	c := ruleUnitContext()
	c.Rules = nil
	c.Facts = nil
	for i := 0; i < 100; i++ {
		r := storedRule{ID: fmt.Sprintf("rule-%03d", i), Status: "active", Output: ruleOutput{Kind: "topic", TopicID: "dining", TagIDs: []string{}}}
		for j := 0; j < 8; j++ {
			r.Conditions = append(r.Conditions, ruleCondition{Field: "description", Operator: "regex", Value: "(a?){100}a{100}"})
		}
		c.Rules = append(c.Rules, r)
		f := ruleUnitContext().Facts[0]
		f.Observation = fmt.Sprintf("observation-%03d", i)
		f.Description = strings.Repeat("a", 500)
		c.Facts = append(c.Facts, f)
	}
	probe := c
	probe.Facts = nil
	if _, err := evaluateRuleContext(context.Background(), probe, true); err != nil {
		t.Fatalf("workload was not admitted: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	began := time.Now()
	_, e := evaluateRuleContext(ctx, c, false)
	elapsed := time.Since(began)
	if e == nil || elapsed > 2*time.Second {
		t.Fatalf("admitted batch ignored cooperative deadline: %s", elapsed)
	}
	t.Logf("100 rules x8 admitted regex conditions x100 facts stopped in %s", elapsed)
}
