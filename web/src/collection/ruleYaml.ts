import {
  isAlias,
  isMap,
  isNode,
  isScalar,
  isSeq,
  parseDocument,
  stringify,
} from "yaml";
import {
  canonicalRules,
  ruleCatalogError,
  validRuleCatalog,
  validRuleCondition,
  type RuleDefinition,
} from "./rules";
import type { Catalog, SourceChoice } from "./types";

type RuleCatalog = Pick<Catalog, "topics" | "tags">;
type Reference = { id: string } | { name: string };
const MAX_BYTES = 262144;
const uuid = (value: unknown): value is string =>
  typeof value === "string" &&
  /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(value) &&
  value !== "00000000-0000-0000-0000-000000000000";
const object = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);
function fail(message: string): never {
  throw new Error(message);
}
function keys(
  value: unknown,
  required: string[],
  optional: string[] = [],
): asserts value is Record<string, unknown> {
  if (
    !object(value) ||
    required.some((key) => !Object.hasOwn(value, key)) ||
    Object.keys(value).some(
      (key) => !required.includes(key) && !optional.includes(key),
    )
  )
    fail(
      `Use only ${[...required, ...optional].join(", ")} in this YAML mapping.`,
    );
}

// Inspect the bounded syntax tree before materializing objects; YAML never executes application logic.
function readYaml(text: string): unknown {
  if (!text.trim()) fail("Paste YAML or choose a .yaml or .yml rule pack.");
  if (new TextEncoder().encode(text).length > MAX_BYTES)
    fail("Rule packs are limited to 256 KiB.");
  const document = parseDocument(text, {
    version: "1.2",
    schema: "core",
    uniqueKeys: true,
    merge: false,
    strict: true,
  });
  if (document.errors.length || document.warnings.length)
    fail(
      "Invalid YAML. Use one document, unique mapping keys and plain YAML values.",
    );
  let count = 0;
  const inspect = (node: unknown, depth: number): void => {
    if (++count > 25000 || depth > 12)
      fail("The YAML rule pack is too deeply nested or complex.");
    if (node === null) return;
    if (
      isAlias(node) ||
      !isNode(node) ||
      node.tag ||
      ("anchor" in node && node.anchor)
    )
      fail(
        "YAML anchors, aliases and explicit tags are not supported in rule packs.",
      );
    if (isMap(node)) {
      for (const pair of node.items) {
        if (
          !isScalar(pair.key) ||
          typeof pair.key.value !== "string" ||
          ["__proto__", "prototype", "constructor"].includes(pair.key.value)
        )
          fail("Rule pack mapping keys must be ordinary strings.");
        inspect(pair.key, depth + 1);
        inspect(pair.value, depth + 1);
      }
    } else if (isSeq(node)) {
      if (node.items.length > 1000)
        fail("A rule pack may contain at most 1,000 retained rules.");
      for (const item of node.items) inspect(item, depth + 1);
    } else if (!isScalar(node))
      fail("Unsupported YAML value in the rule pack.");
  };
  inspect(document.contents, 0);
  return document.toJS({ maxAliasCount: 0 });
}

// Names are exact and must be unambiguous; IDs survive renames and are used by exports.
function reference(
  value: unknown,
  choices: { id: string; name: string }[],
  label: string,
  allowMissingID = false,
): string {
  if (!object(value))
    fail(`${label} must use {name: Exact name} or {id: UUID}.`);
  const names = Object.keys(value);
  if (names.length !== 1 || !["id", "name"].includes(names[0]))
    fail(`${label} must use exactly one name or id.`);
  if (Object.hasOwn(value, "id")) {
    if (!uuid(value.id)) fail(`${label} has an invalid UUID.`);
    if (!allowMissingID && !choices.some((choice) => choice.id === value.id))
      fail(`${label} does not identify a currently available choice.`);
    return value.id;
  }
  if (typeof value.name !== "string" || !value.name.trim())
    fail(`${label} needs an exact name.`);
  const matches = choices.filter((choice) => choice.name === value.name);
  if (matches.length !== 1)
    fail(`${label} name is unknown or ambiguous. Use an exported ID instead.`);
  return matches[0].id;
}

// Compare retained history as typed content, independently of YAML mapping/property order.
const ruleContent = (rule: RuleDefinition) =>
  JSON.stringify([
    rule.rule_id,
    rule.name,
    rule.status,
    rule.source_account_id,
    rule.priority,
    rule.conditions.map((condition) => [
      condition.field,
      condition.operator,
      condition.value,
    ]),
    rule.output.kind,
    rule.output.kind === "topic" ? rule.output.topic_id : null,
    [...rule.output.tag_ids].sort(),
  ]);

// Import only prepares a complete edition for the existing server preview/save/confirm workflow.
// Omitted identities remain; retiring is explicit, and retired history cannot be rewritten.
export function importRulesYaml(
  text: string,
  previous: RuleDefinition[],
  catalog: RuleCatalog,
  sources: SourceChoice[],
): RuleDefinition[] {
  const pack = readYaml(text);
  keys(pack, ["version", "rules"]);
  if (
    pack.version !== 1 ||
    !Array.isArray(pack.rules) ||
    pack.rules.length > 1000
  )
    fail("Use version: 1 and a rules list with at most 1,000 entries.");
  const topicChoices = catalog.topics.map((topic) => ({
    id: topic.topic_id,
    name: topic.name,
  }));
  const tagChoices = catalog.tags.map((tag) => ({
    id: tag.tag_id,
    name: tag.name,
  }));
  const accountChoices = sources.map((source) => ({
    id: source.source_id,
    name: source.name,
  }));
  const supplied = new Set<string>();
  const imported = pack.rules.map((item, index): RuleDefinition => {
    const label = `Rule ${index + 1}`;
    keys(
      item,
      ["name", "conditions", "topic"],
      ["id", "status", "account", "priority", "tags"],
    );
    if (typeof item.name !== "string") fail(`${label} needs a name.`);
    if (item.id !== undefined && !uuid(item.id))
      fail(`${label} has an invalid rule ID.`);
    if (item.id === undefined && previous.some((old) => old.name === item.name))
      fail(
        `${label} already exists. Export its ID to update it; names never silently overwrite rules.`,
      );
    const id = (item.id as string | undefined) ?? crypto.randomUUID();
    if (supplied.has(id)) fail(`${label} repeats a rule ID in this pack.`);
    supplied.add(id);
    if (
      item.status !== undefined &&
      item.status !== "active" &&
      item.status !== "retired"
    )
      fail(`${label} status must be active or retired.`);
    const status = item.status === "retired" ? "retired" : "active";
    const previousRule = previous.find((old) => old.rule_id === id);
    const retained = status === "retired" && previousRule?.status === "retired";
    if (
      !Array.isArray(item.conditions) ||
      item.conditions.length < 1 ||
      item.conditions.length > 8 ||
      !item.conditions.every(validRuleCondition)
    )
      fail(
        `${label} needs 1–8 valid conditions. Condition values must be strings of at most 128 UTF-8 bytes.`,
      );
    const tags = item.tags ?? [];
    if (!Array.isArray(tags) || tags.length > 20)
      fail(`${label} allows at most 20 output tags.`);
    const tag_ids = tags.map((tag) =>
      reference(tag, tagChoices, `${label} tag`, retained),
    );
    const output =
      item.topic === "uncategorized"
        ? { kind: "uncategorized" as const, tag_ids }
        : {
            kind: "topic" as const,
            topic_id: reference(
              item.topic,
              topicChoices,
              `${label} topic`,
              retained,
            ),
            tag_ids,
          };
    const rule: RuleDefinition = {
      rule_id: id,
      name: item.name,
      status,
      source_account_id:
        item.account === undefined || item.account === null
          ? null
          : reference(
              item.account,
              accountChoices,
              `${label} account`,
              retained,
            ),
      priority: item.priority === undefined ? 0 : (item.priority as number),
      conditions: item.conditions,
      output,
    };
    if (
      previousRule?.status === "retired" &&
      ruleContent(rule) !== ruleContent(previousRule)
    )
      fail(`${label} is retired history and cannot be changed or reactivated.`);
    return rule;
  });
  const combined = previous.map(
    (old) => imported.find((rule) => rule.rule_id === old.rule_id) ?? old,
  );
  combined.push(
    ...imported.filter(
      (rule) => !previous.some((old) => old.rule_id === rule.rule_id),
    ),
  );
  const error = ruleCatalogError(combined, previous, catalog, sources);
  if (error) fail(error);
  return canonicalRules(combined);
}

// Export stable identities rather than guessing which duplicate or renamed display name was intended.
export function exportRulesYaml(
  rules: RuleDefinition[],
  _catalog: RuleCatalog,
  _sources: SourceChoice[],
): string {
  // A download must not turn an incomplete editor draft into an apparently reusable pack.
  if (!validRuleCatalog(rules))
    fail("Complete or remove invalid draft rules before exporting YAML.");
  return (
    "# Jayburd rule pack. IDs preserve existing rules; omit id only for a new rule.\n" +
    "# References accept {id: UUID} or {name: Exact name}. Import does not save changes.\n" +
    stringify(
      {
        version: 1,
        rules: canonicalRules(rules).map((rule) => ({
          id: rule.rule_id,
          name: rule.name,
          status: rule.status,
          account:
            rule.source_account_id === null
              ? null
              : { id: rule.source_account_id },
          priority: rule.priority,
          conditions: rule.conditions.map((condition) => ({ ...condition })),
          topic:
            rule.output.kind === "uncategorized"
              ? "uncategorized"
              : { id: rule.output.topic_id },
          tags: rule.output.tag_ids.map((id): Reference => ({ id })),
        })),
      },
      { lineWidth: 0 },
    )
  );
}

// The sample is fictional and inert; it names only an unambiguous active current topic.
export function exampleRulesYaml(
  catalog: RuleCatalog,
  _sources: SourceChoice[],
): string {
  const topic = catalog.topics.find(
    (candidate) =>
      candidate.status === "active" &&
      catalog.topics.filter((other) => other.name === candidate.name).length ===
        1,
  );
  return (
    "# Fictional example: edit the merchant and category before importing.\n" +
    stringify(
      {
        version: 1,
        rules: [
          {
            name: "Fictional cafe",
            conditions: [
              { field: "merchant", operator: "exact", value: "Fictional Cafe" },
            ],
            topic: topic ? { name: topic.name } : "uncategorized",
          },
        ],
      },
      { lineWidth: 0 },
    )
  );
}
