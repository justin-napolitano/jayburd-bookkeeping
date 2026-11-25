import {
  isAlias,
  isMap,
  isNode,
  isScalar,
  isSeq,
  parseDocument,
  stringify,
} from "yaml";
import type { Book } from "./types";
import type { ReviewSource } from "./ReviewInbox";
import {
  accountingID,
  validAccountingRules,
  type AccountingRule,
} from "./automaticAccountingClient";

function fail(message: string): never {
  throw new Error(message);
}
const object = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
const keys = (
  value: unknown,
  required: string[],
  optional: string[] = [],
): value is Record<string, unknown> =>
  object(value) &&
  required.every((key) => Object.hasOwn(value, key)) &&
  Object.keys(value).every(
    (key) => required.includes(key) || optional.includes(key),
  );

// YAML is a bounded local format adapter; mappings are never evaluated or posted in the browser.
export function parseAccountingRules(
  text: string,
  book: Book,
  sources: ReviewSource[],
  previous: AccountingRule[],
): AccountingRule[] {
  if (new TextEncoder().encode(text).length > 131072)
    fail("Use a YAML rule file no larger than 128 KiB.");
  const doc = parseDocument(text, {
    version: "1.2",
    schema: "core",
    uniqueKeys: true,
    merge: false,
    strict: true,
  });
  if (doc.errors.length || doc.warnings.length)
    fail("Use one valid YAML document with unique keys and ordinary values.");
  let nodes = 0;
  const inspect = (node: unknown, depth: number) => {
    if (++nodes > 5000 || depth > 10)
      fail("The accounting rule pack is too complex.");
    if (node === null) return;
    if (
      isAlias(node) ||
      !isNode(node) ||
      node.tag ||
      ("anchor" in node && node.anchor)
    )
      fail("YAML aliases, anchors and explicit tags are not supported.");
    if (isMap(node))
      for (const pair of node.items) {
        if (
          !isScalar(pair.key) ||
          typeof pair.key.value !== "string" ||
          ["__proto__", "prototype", "constructor"].includes(pair.key.value)
        )
          fail("Use ordinary string keys in the rule pack.");
        inspect(pair.key, depth + 1);
        inspect(pair.value, depth + 1);
      }
    else if (isSeq(node)) {
      if (node.items.length > 100) fail("Use at most 100 accounting mappings.");
      for (const item of node.items) inspect(item, depth + 1);
    } else if (!isScalar(node)) fail("Unsupported YAML value.");
  };
  inspect(doc.contents, 0);
  const pack: unknown = doc.toJS({ maxAliasCount: 0 });
  if (
    !keys(pack, ["version", "rules"]) ||
    pack.version !== 1 ||
    !Array.isArray(pack.rules) ||
    pack.rules.length > 100
  )
    fail("Use version: 1 and a rules list with at most 100 entries.");
  const reference = (
    value: unknown,
    choices: { id: string; name: string }[],
    label: string,
  ): string => {
    if (!object(value) || Object.keys(value).length !== 1)
      fail(`${label} needs {name: Exact name} or {id: UUID}.`);
    const matches = Object.hasOwn(value, "id")
      ? choices.filter((choice) => choice.id === value.id)
      : Object.hasOwn(value, "name")
        ? choices.filter((choice) => choice.name === value.name)
        : [];
    if (matches.length !== 1 || !accountingID(matches[0].id))
      fail(`${label} is unavailable or ambiguous. Use an exported ID.`);
    return matches[0].id;
  };
  const rules: AccountingRule[] = pack.rules.map((item, index) => {
    if (
      !keys(
        item,
        ["name", "source", "direction", "description", "offset", "treatment"],
        ["id"],
      )
    )
      fail(`Mapping ${index + 1} has missing or unknown fields.`);
    if (
      typeof item.name !== "string" ||
      (item.id !== undefined && !accountingID(item.id))
    )
      fail(`Mapping ${index + 1} needs a valid name and optional UUID.`);
    if (
      item.id === undefined &&
      previous.some((rule) => rule.name === item.name)
    )
      fail(`Mapping ${index + 1} already exists. Export its ID to update it.`);
    return {
      rule_id: typeof item.id === "string" ? item.id : crypto.randomUUID(),
      name: item.name,
      source_account_id: reference(
        item.source,
        sources.map((source) => ({
          id: source.source_account_id,
          name: source.display_name,
        })),
        `Mapping ${index + 1} source`,
      ),
      direction: item.direction as AccountingRule["direction"],
      description: item.description as string,
      offset_account_id: reference(
        item.offset,
        book.accounts.map((account) => ({
          id: account.account_id,
          name: account.name,
        })),
        `Mapping ${index + 1} offset account`,
      ),
      treatment: item.treatment as AccountingRule["treatment"],
    };
  });
  // Reuse the closed transport validator without deriving any accounting or valuation result.
  if (!validAccountingRules(rules))
    fail(
      "Use unique mapping IDs/names, exact nonempty descriptions up to 500 characters, a valid direction and a supported treatment.",
    );
  return rules;
}

// Exported IDs update future policy editions while historical operations keep their original mapping.
export function exportAccountingRules(rules: AccountingRule[]): string {
  return (
    "# Accounting mappings. Exact descriptions; no merchant or tax inference.\n# IDs update existing mappings. This is the complete future rule set.\n" +
    stringify(
      {
        version: 1,
        rules: rules.map((rule) => ({
          id: rule.rule_id,
          name: rule.name,
          source: { id: rule.source_account_id },
          direction: rule.direction,
          description: rule.description,
          offset: { id: rule.offset_account_id },
          treatment: rule.treatment,
        })),
      },
      { lineWidth: 0 },
    )
  );
}

// The example is inert until edited, reviewed and saved; it is not a supplied accounting policy.
export function exampleAccountingRules(
  book: Book,
  sources: ReviewSource[],
): string {
  const expense = book.accounts.find(
    (account) =>
      account.category === "expense" && account.normal_side === "debit",
  );
  return (
    "# Fictional example. Choose your source, existing expense account and exact source description.\n" +
    stringify(
      {
        version: 1,
        rules: [
          {
            name: "Fictional service charge",
            source: sources[0]
              ? { id: sources[0].source_account_id }
              : { name: "Choose an existing source account" },
            direction: "outflow",
            description: "Fictional service charge",
            offset: expense
              ? { id: expense.account_id }
              : { name: "Choose an existing expense account" },
            treatment: "expense",
          },
        ],
      },
      { lineWidth: 0 },
    )
  );
}
