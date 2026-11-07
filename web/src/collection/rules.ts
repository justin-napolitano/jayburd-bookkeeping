import type { Catalog, Selection, SourceChoice } from "./types";

export type RuleOutput =
  | { kind: "topic"; topic_id: string; tag_ids: string[] }
  | { kind: "uncategorized"; tag_ids: string[] };
export type RuleCondition = {
  field:
    | "merchant"
    | "description"
    | "account"
    | "currency_version"
    | "direction"
    | "date";
  operator: "exact" | "contains" | "regex" | "on_or_after" | "on_or_before";
  value: string;
};
export type RuleDefinition = {
  rule_id: string;
  name: string;
  status: "active" | "retired";
  source_account_id: string | null;
  priority: number;
  conditions: RuleCondition[];
  output: RuleOutput;
};
export type RuleSuggestion = {
  status:
    | "rule"
    | "override"
    | "conflict"
    | "unclassified"
    | "stale_override"
    | "unavailable";
  output: RuleOutput | null;
  rules: { rule_id: string; rule_revision: string }[];
  preserved: { topic: boolean; tags: boolean };
  will_apply: boolean;
};
export type RuleResult = Selection & RuleSuggestion;
const object = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
const id = (value: unknown): value is string =>
  typeof value === "string" &&
  /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(value) &&
  value !== "00000000-0000-0000-0000-000000000000";
const revision = (value: unknown): value is string =>
  typeof value === "string" && /^(0|[1-9][0-9]{0,18})$/.test(value);
const actualDate = (value: string) =>
  /^\d{4}-\d{2}-\d{2}$/.test(value) &&
  value.slice(0, 4) !== "0000" &&
  Number.isFinite(Date.parse(value + "T00:00:00Z")) &&
  new Date(value + "T00:00:00Z").toISOString().slice(0, 10) === value;
const exactKeys = (value: Record<string, unknown>, keys: string[]) =>
  Object.keys(value).length === keys.length &&
  keys.every((key) => Object.hasOwn(value, key));
export function validRuleOutput(
  value: unknown,
  maximum = 20,
): value is RuleOutput {
  return (
    object(value) &&
    (value.kind === "topic"
      ? exactKeys(value, ["kind", "topic_id", "tag_ids"]) && id(value.topic_id)
      : value.kind === "uncategorized" &&
        exactKeys(value, ["kind", "tag_ids"])) &&
    Array.isArray(value.tag_ids) &&
    value.tag_ids.length <= maximum &&
    value.tag_ids.every(id) &&
    new Set(value.tag_ids).size === value.tag_ids.length
  );
}
// Regex is only bounded as text here; the single Go evaluator validates syntax/work and performs matching.
export function validRuleCondition(value: unknown): value is RuleCondition {
  if (
    !object(value) ||
    !exactKeys(value, ["field", "operator", "value"]) ||
    typeof value.value !== "string" ||
    new TextEncoder().encode(value.value).length < 1 ||
    new TextEncoder().encode(value.value).length > 128
  )
    return false;
  if (value.field === "merchant" || value.field === "description")
    return ["exact", "contains", "regex"].includes(String(value.operator));
  if (value.field === "account" || value.field === "currency_version")
    return value.operator === "exact" && id(value.value);
  if (value.field === "direction")
    return (
      value.operator === "exact" && ["inflow", "outflow"].includes(value.value)
    );
  return (
    value.field === "date" &&
    ["on_or_after", "on_or_before"].includes(String(value.operator)) &&
    actualDate(value.value)
  );
}
// All retained IDs survive complete editions; a helpful client bound complements authoritative SQL admission.
export function validRuleCatalog(value: unknown): value is RuleDefinition[] {
  if (
    !Array.isArray(value) ||
    value.length > 1000 ||
    new TextEncoder().encode(JSON.stringify(value)).length > 262144
  )
    return false;
  const ids = new Set<string>(),
    names = new Set<string>();
  let active = 0;
  return value.every((rule) => {
    if (
      !object(rule) ||
      !exactKeys(rule, [
        "rule_id",
        "name",
        "status",
        "source_account_id",
        "priority",
        "conditions",
        "output",
      ]) ||
      !id(rule.rule_id) ||
      ids.has(rule.rule_id) ||
      typeof rule.name !== "string" ||
      [...rule.name].length < 1 ||
      [...rule.name].length > 80 ||
      !rule.name.trim() ||
      /[\u0000-\u001f\u007f-\u009f]/.test(rule.name) ||
      !["active", "retired"].includes(String(rule.status)) ||
      (rule.source_account_id !== null && !id(rule.source_account_id)) ||
      !Number.isInteger(rule.priority) ||
      Number(rule.priority) < -1000000 ||
      Number(rule.priority) > 1000000 ||
      !Array.isArray(rule.conditions) ||
      rule.conditions.length < 1 ||
      rule.conditions.length > 8 ||
      !rule.conditions.every(validRuleCondition) ||
      !validRuleOutput(rule.output)
    )
      return false;
    ids.add(rule.rule_id);
    if (rule.status === "active") {
      if (++active > 100 || names.has(rule.name)) return false;
      names.add(rule.name);
    }
    return true;
  });
}
export function ruleCatalogError(
  rules: RuleDefinition[],
  previous: RuleDefinition[],
  catalog: Pick<Catalog, "topics" | "tags">,
  sources: SourceChoice[],
): string | null {
  if (new TextEncoder().encode(JSON.stringify(rules)).length > 262144)
    return "The complete rule catalog exceeds the 256 KiB review limit. Retained history is never truncated.";
  if (!validRuleCatalog(rules))
    return "Use unique active names of 1–80 characters, at most 100 active rules, 1–8 valid conditions per rule and up to 20 output tags. Condition values are limited to 128 UTF-8 bytes; regex syntax is reviewed by the server.";
  for (const old of previous) {
    const current = rules.find((rule) => rule.rule_id === old.rule_id);
    if (!current || (old.status === "retired" && current.status !== "retired"))
      return "Keep every existing rule identity. Retired rules cannot be reactivated.";
  }
  for (const rule of rules.filter((rule) => rule.status === "active")) {
    if (
      rule.source_account_id !== null &&
      !sources.some((source) => source.source_id === rule.source_account_id)
    )
      return "Choose a currently authorized source account or the whole private workspace.";
    const output = rule.output;
    if (
      (output.kind === "topic" &&
        !catalog.topics.some(
          (topic) =>
            topic.topic_id === output.topic_id && topic.status === "active",
        )) ||
      output.tag_ids.some(
        (tag) =>
          !catalog.tags.some(
            (item) => item.tag_id === tag && item.status === "active",
          ),
      )
    )
      return "Active rules require a current active primary topic and active output tags, or explicit Uncategorized.";
  }
  return null;
}
// Suggestions remain inert; only the exact saved preview may carry affirmative application eligibility.
export function validRuleSuggestion(
  value: unknown,
  inert = false,
): value is RuleSuggestion {
  if (
    !object(value) ||
    ![
      "rule",
      "override",
      "conflict",
      "unclassified",
      "stale_override",
      "unavailable",
    ].includes(String(value.status)) ||
    !object(value.preserved) ||
    !exactKeys(value.preserved, ["topic", "tags"]) ||
    typeof value.preserved.topic !== "boolean" ||
    typeof value.preserved.tags !== "boolean" ||
    typeof value.will_apply !== "boolean" ||
    (inert && value.will_apply) ||
    (value.will_apply && value.status !== "rule") ||
    !Array.isArray(value.rules) ||
    value.rules.length > 100
  )
    return false;
  const ids = new Set<string>();
  if (
    !value.rules.every((rule) => {
      if (
        !object(rule) ||
        !exactKeys(rule, ["rule_id", "rule_revision"]) ||
        !id(rule.rule_id) ||
        ids.has(rule.rule_id) ||
        !revision(rule.rule_revision) ||
        rule.rule_revision === "0"
      )
        return false;
      ids.add(rule.rule_id);
      return true;
    })
  )
    return false;
  if (value.status === "rule" || value.status === "override")
    return (
      validRuleOutput(value.output, value.preserved.tags ? 25 : 20) &&
      (value.status !== "rule" || value.rules.length > 0)
    );
  return value.output === null;
}
// Catalog/action arrays are identity-keyed sets; condition ordering and values remain exact review content.
export function canonicalRules(rules: RuleDefinition[]): RuleDefinition[] {
  return [...rules]
    .sort((a, b) =>
      a.rule_id < b.rule_id ? -1 : a.rule_id > b.rule_id ? 1 : 0,
    )
    .map((rule) => ({
      ...rule,
      output: { ...rule.output, tag_ids: [...rule.output.tag_ids].sort() },
    }));
}

export type RuleApplication =
  | { state: "none" }
  | {
      state: "current" | "stale" | "superseded";
      ruleset_revision: string;
      rules: { rule_id: string; rule_revision: string }[];
      fields: { topic: boolean; tags: boolean };
    };
// Persisted origin is independent of a new inert suggestion and may refer to historical ruleset pins.
export function validRuleApplication(value: unknown): value is RuleApplication {
  if (!object(value)) return false;
  if (value.state === "none") return exactKeys(value, ["state"]);
  if (
    !exactKeys(value, ["state", "ruleset_revision", "rules", "fields"]) ||
    !["current", "stale", "superseded"].includes(String(value.state)) ||
    !revision(value.ruleset_revision) ||
    value.ruleset_revision === "0" ||
    !object(value.fields) ||
    !exactKeys(value.fields, ["topic", "tags"]) ||
    typeof value.fields.topic !== "boolean" ||
    typeof value.fields.tags !== "boolean" ||
    !Array.isArray(value.rules) ||
    value.rules.length < 1 ||
    value.rules.length > 100
  )
    return false;
  const ids = new Set<string>();
  return value.rules.every((rule) => {
    if (
      !object(rule) ||
      !exactKeys(rule, ["rule_id", "rule_revision"]) ||
      !id(rule.rule_id) ||
      ids.has(rule.rule_id) ||
      !revision(rule.rule_revision) ||
      rule.rule_revision === "0" ||
      rule.rule_revision !== value.ruleset_revision
    )
      return false;
    ids.add(rule.rule_id);
    return true;
  });
}

export type RuleSeed = {
  workspace_id: string;
  command_id: string;
  prepare_key: string;
  source_id: string;
  condition: RuleCondition;
  output: RuleOutput | null;
};
