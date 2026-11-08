import { describe, expect, it } from "vitest";
import {
  collectionCatalog,
  collectionPreview,
  collectionRow,
  collectionSnapshot,
  collectionWorkspace,
} from "../../tests/collection-fixture";
import {
  selectionOf,
  validPreview,
  validCommand,
  samePreview,
  type PreviewRequest,
  type V2Preview,
} from "./types";
import {
  canonicalRules,
  ruleCatalogError,
  validRuleCatalog,
  validRuleCondition,
  validRuleSuggestion,
  type RuleDefinition,
} from "./rules";
const id = (n: number) =>
  `85000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
const rule = (n = 1): RuleDefinition => ({
  rule_id: id(n),
  name: "Fictional rule " + n,
  status: "active",
  source_account_id: null,
  priority: 0,
  conditions: [
    { field: "merchant", operator: "exact", value: "Fictional Cafe" },
  ],
  output: { kind: "uncategorized", tag_ids: [] },
});
const request: PreviewRequest = {
  workspace_id: collectionWorkspace.workspace_id,
  snapshot: collectionSnapshot,
  selection: [selectionOf(collectionRow())],
  action: { kind: "rules_apply" },
};
describe("versioned private rule review", () => {
  it("bounds complete catalogs, active names, UTF8 conditions and aggregate bytes without evaluating regex", () => {
    expect(validRuleCatalog([rule()])).toBe(true);
    expect(validRuleCatalog([rule(), { ...rule(2), name: rule().name }])).toBe(
      false,
    );
    expect(
      validRuleCatalog([
        rule(),
        { ...rule(2), name: rule().name, status: "retired" },
      ]),
    ).toBe(true);
    expect(
      validRuleCatalog(Array.from({ length: 101 }, (_, i) => rule(i + 1))),
    ).toBe(false);
    expect(
      validRuleCondition({ field: "merchant", operator: "regex", value: "(" }),
    ).toBe(true); // Go alone validates the regex program.
    expect(
      validRuleCondition({
        field: "merchant",
        operator: "contains",
        value: "😀".repeat(32),
      }),
    ).toBe(true);
    expect(
      validRuleCondition({
        field: "merchant",
        operator: "contains",
        value: "😀".repeat(33),
      }),
    ).toBe(false);
    expect(
      validRuleCondition({ field: "amount", operator: "exact", value: "1" }),
    ).toBe(false);
    const retained = Array.from({ length: 1000 }, (_, i) => ({
      ...rule(i + 1),
      status: "retired" as const,
      conditions: Array.from({ length: 8 }, () => ({
        field: "description" as const,
        operator: "contains" as const,
        value: "x".repeat(128),
      })),
    }));
    expect(ruleCatalogError(retained, [], collectionCatalog, [])).toMatch(
      /256 KiB/,
    );
  });
  it("normalizes rule identity and tag sets while preserving condition order", () => {
    const r = rule();
    r.conditions.push({
      field: "description",
      operator: "contains",
      value: "Lunch",
    });
    expect(canonicalRules([rule(2), r]).map((value) => value.rule_id)).toEqual([
      id(1),
      id(2),
    ]);
    expect(canonicalRules([r])[0].conditions).toEqual(r.conditions);
    expect(
      ruleCatalogError(
        [{ ...rule(), status: "active" }],
        [{ ...rule(), status: "retired" }],
        collectionCatalog,
        [],
      ),
    ).toMatch(/reactivated/);
  });
  it("pins exact eligible results/counts and refuses inert-query eligibility", () => {
    const p = collectionPreview(request, [collectionRow()]);
    p.rule_results = [
      {
        ...request.selection[0],
        status: "rule",
        output: { kind: "uncategorized", tag_ids: [] },
        rules: [
          {
            rule_id: id(1),
            rule_revision: collectionSnapshot.ruleset_revision,
          },
        ],
        preserved: { topic: false, tags: false },
        will_apply: true,
      },
    ];
    p.apply_count = "1";
    p.abstain_count = "0";
    expect(validPreview(p, request)).toBe(true);
    expect(validPreview({ ...p, apply_count: "0" }, request)).toBe(false);
    expect(
      validPreview(
        {
          ...p,
          rule_results: [{ ...p.rule_results[0], observation_revision: "2" }],
        },
        request,
      ),
    ).toBe(false);
    expect(validRuleSuggestion(p.rule_results[0], true)).toBe(false);
    expect(
      validRuleSuggestion({
        ...p.rule_results[0],
        status: "conflict",
        output: null,
        will_apply: true,
      }),
    ).toBe(false);
    expect(
      samePreview(p, {
        ...p,
        rule_results: [{ ...p.rule_results[0], will_apply: false }],
      }),
    ).toBe(false);
  });
  it("preserves up to25 manual tags independently while newly authored outputs stop at20", () => {
    const result = {
      status: "rule",
      output: {
        kind: "uncategorized",
        tag_ids: Array.from({ length: 25 }, (_, i) => id(i + 20)),
      },
      rules: [{ rule_id: id(1), rule_revision: "1" }],
      preserved: { topic: false, tags: true },
      will_apply: false,
    };
    expect(validRuleSuggestion(result, true)).toBe(true);
    expect(
      validRuleSuggestion(
        { ...result, preserved: { topic: false, tags: false } },
        true,
      ),
    ).toBe(false);
    expect(validRuleCatalog([{ ...rule(), output: result.output }])).toBe(
      false,
    );
  });
  it("recovers the untouched v2 preview union and rejects fabricated v3 fields", () => {
    const modern = collectionPreview(
      { ...request, action: { kind: "uncategorized" } },
      [collectionRow()],
    );
    const {
      rule_results: _r,
      apply_count: _a,
      abstain_count: _b,
      ...base
    } = modern;
    const v2: V2Preview = {
      ...base,
      profile: "private-collection-preview-v2",
      action: { kind: "uncategorized" },
    };
    const command = {
      workspace_id: collectionWorkspace.workspace_id,
      command_id: "old-v2",
      prepare_key: "stable-v2",
      state: "prepared",
      acknowledged: false,
      preview: v2,
    };
    expect(validCommand(command, collectionWorkspace, collectionCatalog)).toBe(
      true,
    );
    expect(
      validCommand(
        { ...command, preview: { ...v2, rule_results: [] } },
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(false);
    expect(samePreview(v2, structuredClone(v2))).toBe(true);
    expect(samePreview(v2, modern)).toBe(false);
  });
});
