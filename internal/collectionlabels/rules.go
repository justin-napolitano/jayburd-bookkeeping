package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/categorization"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// RuleProcessor evaluates only SQL-authorized inputs inside the transport's existing transaction.
// It cannot select another connection, commit, grant authority, or write a journal.
type RuleProcessor struct{}

var _ RuleHook = RuleProcessor{}

type ruleOutput struct {
	Kind    string   `json:"kind"`
	TopicID string   `json:"topic_id,omitempty"`
	TagIDs  []string `json:"tag_ids"`
}
type ruleCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}
type storedRule struct {
	ID         string          `json:"rule_id"`
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Account    *string         `json:"source_account_id"`
	Priority   int             `json:"priority"`
	Conditions []ruleCondition `json:"conditions"`
	Output     ruleOutput      `json:"output"`
}
type ruleProvenance struct {
	ID       string `json:"rule_id"`
	Revision string `json:"rule_revision"`
}
type preservedFields struct {
	Topic bool `json:"topic"`
	Tags  bool `json:"tags"`
}
type ruleEvaluation struct {
	Workspace   string           `json:"workspace_id"`
	Source      string           `json:"source_id"`
	Observation string           `json:"observation_id"`
	Revision    string           `json:"observation_revision"`
	Status      string           `json:"status"`
	Output      *ruleOutput      `json:"output"`
	Rules       []ruleProvenance `json:"rules"`
	Preserved   preservedFields  `json:"preserved"`
	WillApply   bool             `json:"will_apply"`
}
type manualRuleTopic struct {
	SourceRevision   string `json:"observation_revision"`
	TaxonomyRevision string `json:"taxonomy_revision"`
	Kind             string `json:"kind"`
	TopicID          string `json:"topic_id"`
}
type manualRuleTags struct {
	SourceRevision string   `json:"observation_revision"`
	TagRevision    string   `json:"tag_revision"`
	TagIDs         []string `json:"tag_ids"`
}
type ruleFact struct {
	Source      string           `json:"source_id"`
	Observation string           `json:"observation_id"`
	Revision    string           `json:"observation_revision"`
	Date        string           `json:"booked_date"`
	Merchant    *string          `json:"merchant"`
	Description string           `json:"description"`
	Currency    *string          `json:"currency_version_id"`
	Direction   *string          `json:"direction"`
	SourceState string           `json:"source_state"`
	MoneyState  string           `json:"money_state"`
	Topic       *manualRuleTopic `json:"manual_topic"`
	Tags        *manualRuleTags  `json:"manual_tags"`
}
type ruleContext struct {
	Profile   string `json:"profile"`
	Principal string `json:"principal_id"`
	Workspace string `json:"workspace_id"`
	Snapshot  struct {
		Taxonomy string `json:"taxonomy_revision"`
		Tags     string `json:"tag_revision"`
		Rules    string `json:"ruleset_revision"`
	} `json:"snapshot"`
	Action struct {
		Kind string `json:"kind"`
	} `json:"action"`
	Topics []string     `json:"topic_ids"`
	Tags   []string     `json:"tag_ids"`
	Rules  []storedRule `json:"rules"`
	Facts  []ruleFact   `json:"facts"`
}

// Process handles v3 inert page/detail suggestions and service-evaluated durable rule previews.
// The caller retains its ordinary result fence, COMMIT acknowledgement and whole-transaction retries.
func (RuleProcessor) Process(ctx context.Context, tx pgx.Tx, r Request, operation string) (json.RawMessage, bool, error) {
	if r.Version != 3 {
		return nil, false, nil
	}
	switch operation {
	case "query", "detail":
		return processRuleRead(ctx, tx, r, operation)
	case "preview":
		c, e := loadRuleContext(ctx, tx, r, r.Body, true)
		if e != nil {
			return nil, true, e
		}
		if c.Action.Kind != "rule_catalog" && c.Action.Kind != "rules_apply" {
			return nil, false, nil
		}
		results, e := evaluateRuleContext(ctx, c, c.Action.Kind == "rule_catalog")
		if e != nil {
			return nil, true, e
		}
		encoded, e := json.Marshal(map[string]any{"results": results})
		if e != nil {
			return nil, true, unavailable()
		}
		var body json.RawMessage
		e = tx.QueryRow(ctx, "select banking.publish_collection_rule_preview($1::uuid,$2::uuid,$3::text,$4::text)", r.Actor, r.Workspace, r.Body, string(encoded)).Scan(&body)
		return body, true, e
	default:
		return nil, false, nil
	}
}

// loadRuleContext never parses caller facts; PostgreSQL validates raw request keys and resolves authority first.
func loadRuleContext(ctx context.Context, tx pgx.Tx, r Request, raw string, writing bool) (ruleContext, error) {
	var body json.RawMessage
	var c ruleContext
	e := tx.QueryRow(ctx, "select banking.collection_rule_context($1::uuid,$2::uuid,$3::text,$4::boolean)", r.Actor, r.Workspace, raw, writing).Scan(&body)
	if e != nil {
		return c, e
	}
	if len(body) > 1048576 || json.Unmarshal(body, &c) != nil || c.Profile != "private-rule-context-v1" || c.Principal != r.Actor || c.Workspace != r.Workspace || len(c.Facts) > 100 || len(c.Rules) > 1000 {
		return c, unavailable()
	}
	return c, nil
}

// processRuleRead obtains the bounded page and matcher inventory under the same already-open RR snapshot.
func processRuleRead(ctx context.Context, tx pgx.Tx, r Request, operation string) (json.RawMessage, bool, error) {
	var body json.RawMessage
	var e error
	if operation == "query" {
		e = tx.QueryRow(ctx, "select banking.query_collection_review_v3($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
	} else {
		e = tx.QueryRow(ctx, "select banking.get_collection_detail($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
	}
	if e != nil {
		return nil, true, e
	}
	if len(body) > 1048576 {
		return nil, true, unavailable()
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(body, &envelope) != nil {
		return nil, true, unavailable()
	}
	rows := []json.RawMessage{}
	if operation == "query" {
		if json.Unmarshal(envelope["items"], &rows) != nil {
			return nil, true, unavailable()
		}
	} else {
		rows = append(rows, envelope["row"])
	}
	selection := []map[string]json.RawMessage{}
	for _, row := range rows {
		var v map[string]json.RawMessage
		if json.Unmarshal(row, &v) != nil {
			return nil, true, unavailable()
		}
		selection = append(selection, map[string]json.RawMessage{"workspace_id": v["workspace_id"], "source_id": v["source_id"], "observation_id": v["observation_id"], "observation_revision": v["observation_revision"]})
	}
	request, e := json.Marshal(map[string]any{"snapshot": envelope["snapshot"], "selection": selection, "action": map[string]string{"kind": "rules_apply"}})
	if e != nil {
		return nil, true, unavailable()
	}
	c, e := loadRuleContext(ctx, tx, r, string(request), false)
	if e != nil {
		return nil, true, e
	}
	results, e := evaluateRuleContext(ctx, c, false)
	if e != nil {
		return nil, true, e
	}
	if len(results) != len(rows) {
		return nil, true, unavailable()
	}
	byID := map[string]ruleEvaluation{}
	for _, result := range results {
		byID[result.Source+":"+result.Observation] = result
	}
	for i, row := range rows {
		var v map[string]json.RawMessage
		if json.Unmarshal(row, &v) != nil {
			return nil, true, unavailable()
		}
		var sid, oid string
		if json.Unmarshal(v["source_id"], &sid) != nil || json.Unmarshal(v["observation_id"], &oid) != nil {
			return nil, true, unavailable()
		}
		result, ok := byID[sid+":"+oid]
		if !ok {
			return nil, true, unavailable()
		}
		v["rule_suggestion"], e = json.Marshal(struct {
			Status    string           `json:"status"`
			Output    *ruleOutput      `json:"output"`
			Rules     []ruleProvenance `json:"rules"`
			Preserved preservedFields  `json:"preserved"`
			WillApply bool             `json:"will_apply"`
		}{result.Status, result.Output, result.Rules, result.Preserved, false})
		if e != nil {
			return nil, true, unavailable()
		}
		rows[i], e = json.Marshal(v)
		if e != nil {
			return nil, true, unavailable()
		}
	}
	if operation == "query" {
		envelope["items"], e = json.Marshal(rows)
	} else {
		envelope["row"] = rows[0]
	}
	if e != nil {
		return nil, true, unavailable()
	}
	body, e = json.Marshal(envelope)
	if e != nil || len(body) > 1048576 {
		return nil, true, unavailable()
	}
	return body, true, nil
}

// evaluateRuleContext ranks once per authorized edition; SQL independently protects manual fields and effects.
func evaluateRuleContext(ctx context.Context, c ruleContext, validateCandidate bool) ([]ruleEvaluation, error) {
	scope := categorization.Scope{Principal: c.Principal, Workspace: c.Workspace}
	rules := []categorization.Rule{}
	for _, r := range c.Rules {
		if ctx.Err() != nil {
			return nil, unavailable()
		}
		if r.Status != "active" {
			continue
		}
		conditions := make([]categorization.Condition, 0, len(r.Conditions))
		for _, v := range r.Conditions {
			conditions = append(conditions, categorization.Condition{Field: v.Field, Operator: v.Operator, Value: v.Value})
		}
		account := ""
		if r.Account != nil {
			account = *r.Account
		}
		rules = append(rules, categorization.Rule{Scope: scope, ID: r.ID, Revision: c.Snapshot.Rules, Account: account, Priority: r.Priority, Conditions: conditions, Output: categorization.Output{TopicID: r.Output.TopicID, Uncategorized: r.Output.Kind == "uncategorized", Tags: slices.Clone(r.Output.TagIDs)}})
	}
	if ctx.Err() != nil {
		return nil, unavailable()
	}
	engine, compileErr := categorization.CompileContext(ctx, categorization.Catalog{Scope: scope, TaxonomyRevision: c.Snapshot.Taxonomy, RulesRevision: c.Snapshot.Rules, Topics: c.Topics, Tags: c.Tags}, rules)
	if ctx.Err() != nil {
		return nil, unavailable()
	}
	if compileErr != nil && !errors.Is(compileErr, categorization.ErrInput) {
		return nil, unavailable()
	}
	if validateCandidate && compileErr != nil {
		return nil, &ledger.Failure{Status: 400, Code: "invalid_collection"}
	}
	results := make([]ruleEvaluation, 0, len(c.Facts))
	for _, f := range c.Facts {
		if ctx.Err() != nil {
			return nil, unavailable()
		}
		result := ruleEvaluation{Workspace: c.Workspace, Source: f.Source, Observation: f.Observation, Revision: f.Revision, Status: "unavailable", Rules: []ruleProvenance{}, Preserved: preservedFields{Topic: f.Topic != nil, Tags: f.Tags != nil}}
		stale := f.Topic != nil && (f.Topic.SourceRevision != f.Revision || f.Topic.TaxonomyRevision != c.Snapshot.Taxonomy) || f.Tags != nil && (f.Tags.SourceRevision != f.Revision || f.Tags.TagRevision != c.Snapshot.Tags)
		switch {
		case stale:
			result.Status = "stale_override"
		case f.Topic != nil && f.Tags != nil:
			result.Status = "override"
			result.Output = &ruleOutput{Kind: f.Topic.Kind, TopicID: f.Topic.TopicID, TagIDs: slices.Clone(f.Tags.TagIDs)}
		case compileErr == nil && f.SourceState == "booked" && f.MoneyState == "available" && f.Currency != nil && f.Direction != nil && (*f.Direction == "inflow" || *f.Direction == "outflow"):
			evaluated, e := engine.EvaluateContext(ctx, []categorization.Fact{{Scope: scope, ID: f.Source + ":" + f.Observation, Revision: f.Revision, Account: f.Source, CurrencyVersion: *f.Currency, Direction: *f.Direction, Date: f.Date, Merchant: f.Merchant, Description: f.Description}})
			if e != nil {
				return nil, unavailable()
			}
			if ctx.Err() != nil {
				return nil, unavailable()
			}
			v := evaluated[0]
			result.Status = v.Status
			for _, p := range v.Rules {
				result.Rules = append(result.Rules, ruleProvenance{ID: p.ID, Revision: p.Revision})
			}
			if v.Output != nil {
				kind := "topic"
				if v.Output.Uncategorized {
					kind = "uncategorized"
				}
				result.Output = &ruleOutput{Kind: kind, TopicID: v.Output.TopicID, TagIDs: slices.Clone(v.Output.Tags)}
				if f.Topic != nil {
					result.Output.Kind = f.Topic.Kind
					result.Output.TopicID = f.Topic.TopicID
				}
				if f.Tags != nil {
					result.Output.TagIDs = slices.Clone(f.Tags.TagIDs)
				}
			}
		}
		if result.Output != nil && result.Output.TagIDs == nil {
			result.Output.TagIDs = []string{}
		}
		results = append(results, result)
	}
	return results, nil
}
