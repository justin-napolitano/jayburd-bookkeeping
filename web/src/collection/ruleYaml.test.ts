import { describe, expect, it } from "vitest";
import { exampleRulesYaml, exportRulesYaml, importRulesYaml } from "./ruleYaml";
import { canonicalRules, type RuleDefinition } from "./rules";

const id = (n: number) =>
  `ab000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
const catalog = {
  topics: [
    {
      topic_id: id(1),
      name: "Food and drink",
      status: "active" as const,
      parent_id: null,
    },
    {
      topic_id: id(2),
      name: "Travel",
      status: "active" as const,
      parent_id: null,
    },
    {
      topic_id: id(3),
      name: "Old category",
      status: "retired" as const,
      parent_id: null,
    },
  ],
  tags: [{ tag_id: id(4), name: "Fictional trip", status: "active" as const }],
};
const sources = [{ source_id: id(5), name: "Fictional checking" }];
const existing: RuleDefinition = {
  rule_id: id(6),
  name: "Existing rule",
  status: "active",
  priority: 10,
  source_account_id: id(5),
  conditions: [
    { field: "merchant", operator: "exact", value: "Fictional\nCafe" },
  ],
  output: { kind: "topic", topic_id: id(1), tag_ids: [id(4)] },
};
const newPack = `version: 1
rules:
  - name: Fictional transport
    account: {name: Fictional checking}
    conditions:
      - field: merchant
        operator: exact
        value: Fictional Cab
    topic: {name: Travel}
    tags: [{name: Fictional trip}]
`;

describe("portable YAML rule packs", () => {
  it.each([
    ["blank name", { name: "" }],
    ["non-finite priority", { priority: NaN }],
    ["missing topic", { output: { kind: "topic", topic_id: "", tag_ids: [] } }],
  ])("rejects exporting a current draft with %s", (_label, patch) => {
    const draft = { ...existing, ...patch } as RuleDefinition;
    expect(() => exportRulesYaml([draft], catalog, sources)).toThrow(
      "Complete or remove invalid draft rules",
    );
  });

  it("imports friendly names into a complete preserved edition without changing prior objects", () => {
    const before = JSON.stringify(existing);
    const result = importRulesYaml(newPack, [existing], catalog, sources);
    expect(result).toHaveLength(2);
    expect(result.find((rule) => rule.rule_id === id(6))).toEqual(existing);
    const created = result.find((rule) => rule.name === "Fictional transport")!;
    expect(created.rule_id).toMatch(/^[a-f0-9-]{36}$/);
    expect(created.source_account_id).toBe(id(5));
    expect(created.priority).toBe(0);
    expect(created.output).toEqual({
      kind: "topic",
      topic_id: id(2),
      tag_ids: [id(4)],
    });
    expect(JSON.stringify(existing)).toBe(before);
  });

  it("exports stable IDs and round-trips exact multiline values and retired missing references", () => {
    const retired: RuleDefinition = {
      ...existing,
      rule_id: id(7),
      name: "Old rule",
      status: "retired",
      source_account_id: id(90),
      output: { kind: "topic", topic_id: id(91), tag_ids: [id(92)] },
    };
    const previous = [existing, retired];
    const yaml = exportRulesYaml(previous, catalog, sources);
    expect(yaml).toContain(`id: ${id(6)}`);
    expect(importRulesYaml(yaml, previous, catalog, sources)).toEqual(
      canonicalRules(previous),
    );
  });

  it("updates only an explicit ID, keeps omitted history, and allows explicit retirement", () => {
    const sibling = { ...existing, rule_id: id(7), name: "Sibling rule" };
    const yaml = `version: 1
rules:
  - id: ${id(6)}
    name: Renamed rule
    status: retired
    priority: 30
    conditions: [{field: description, operator: contains, value: Fictional}]
    topic: uncategorized
`;
    const result = importRulesYaml(yaml, [existing, sibling], catalog, sources);
    expect(result).toEqual(
      canonicalRules([
        {
          rule_id: id(6),
          name: "Renamed rule",
          status: "retired",
          priority: 30,
          source_account_id: null,
          conditions: [
            { field: "description", operator: "contains", value: "Fictional" },
          ],
          output: { kind: "uncategorized", tag_ids: [] },
        },
        sibling,
      ]),
    );
  });

  it("rejects repeated no-ID imports instead of overwriting an existing name", () => {
    const prior = importRulesYaml(newPack, [], catalog, sources);
    expect(() => importRulesYaml(newPack, prior, catalog, sources)).toThrow(
      "Export its ID",
    );
    expect(
      importRulesYaml(
        exportRulesYaml(prior, catalog, sources),
        prior,
        catalog,
        sources,
      ),
    ).toEqual(prior);
  });

  it("rejects duplicate rule IDs and active names within a pack", () => {
    const row = `  - id: ${id(10)}\n    name: Duplicate\n    conditions: [{field: merchant, operator: exact, value: Fictional}]\n    topic: uncategorized\n`;
    expect(() =>
      importRulesYaml("version: 1\nrules:\n" + row + row, [], catalog, sources),
    ).toThrow("repeats a rule ID");
    expect(() =>
      importRulesYaml(
        "version: 1\nrules:\n" + row + row.replace(id(10), id(11)),
        [],
        catalog,
        sources,
      ),
    ).toThrow("unique active names");
  });

  it("rejects unknown and ambiguous account/topic/tag names without falling back", () => {
    expect(() =>
      importRulesYaml(
        newPack.replace("{name: Travel}", "{name: Missing}"),
        [],
        catalog,
        sources,
      ),
    ).toThrow("unknown or ambiguous");
    expect(() =>
      importRulesYaml(
        newPack.replace("{name: Fictional checking}", `{id: ${id(99)}}`),
        [],
        catalog,
        sources,
      ),
    ).toThrow("currently available");
    const ambiguous = {
      ...catalog,
      topics: [...catalog.topics, { ...catalog.topics[1], topic_id: id(99) }],
    };
    expect(() => importRulesYaml(newPack, [], ambiguous, sources)).toThrow(
      "unknown or ambiguous",
    );
    expect(() =>
      importRulesYaml(
        newPack.replace("{name: Fictional trip}", "{name: Unknown tag}"),
        [],
        catalog,
        sources,
      ),
    ).toThrow("unknown or ambiguous");
  });

  it("never rewrites or reactivates retired rule history", () => {
    const retired = { ...existing, status: "retired" as const };
    const yaml = exportRulesYaml([retired], catalog, sources);
    expect(() =>
      importRulesYaml(
        yaml.replace("status: retired", "status: active"),
        [retired],
        catalog,
        sources,
      ),
    ).toThrow("retired history");
    expect(() =>
      importRulesYaml(
        yaml.replace("name: Existing rule", "name: Changed rule"),
        [retired],
        catalog,
        sources,
      ),
    ).toThrow("retired history");
  });

  it("retains server-only regex evaluation and rejects numeric/date coercion", () => {
    const regex = newPack
      .replace("operator: exact", "operator: regex")
      .replace("value: Fictional Cab", 'value: "("');
    expect(
      importRulesYaml(regex, [], catalog, sources)[0].conditions[0].value,
    ).toBe("(");
    expect(() =>
      importRulesYaml(
        newPack.replace("value: Fictional Cab", "value: 123"),
        [],
        catalog,
        sources,
      ),
    ).toThrow("Condition values must be strings");
    expect(() =>
      importRulesYaml(
        newPack.replace("value: Fictional Cab", `value: '${"😀".repeat(33)}'`),
        [],
        catalog,
        sources,
      ),
    ).toThrow("128 UTF-8 bytes");
  });

  it.each([
    ["duplicate keys", "version: 1\nversion: 1\nrules: []"],
    ["multiple documents", "version: 1\nrules: []\n---\nversion: 1\nrules: []"],
    ["aliases", "version: 1\nrules: &rules [*rules]"],
    ["anchors", "version: 1\nrules: &rules []"],
    ["explicit tags", "version: !!int 1\nrules: []"],
    ["custom tags", "version: !evil 1\nrules: []"],
    ["merge keys", "version: 1\nrules: []\n<<: {}"],
    ["prototype keys", "version: 1\nrules: []\n__proto__: {}"],
    ["unknown profile keys", "version: 1\nrules: []\nexecute: true"],
    ["unsupported version", "version: 2\nrules: []"],
  ])("rejects %s", (_label, text) => {
    expect(() => importRulesYaml(text, [], catalog, sources)).toThrow();
  });

  it("enforces input, nesting, count and active-output limits before accepting a draft", () => {
    expect(() =>
      importRulesYaml("#".repeat(262145), [], catalog, sources),
    ).toThrow("256 KiB");
    expect(() =>
      importRulesYaml("[".repeat(20) + "]".repeat(20), [], catalog, sources),
    ).toThrow("deeply nested");
    expect(() =>
      importRulesYaml(
        `version: 1\nrules: [${Array(1001).fill("null").join(",")}]`,
        [],
        catalog,
        sources,
      ),
    ).toThrow("1,000");
    expect(() =>
      importRulesYaml(
        newPack.replace("{name: Travel}", "{name: Old category}"),
        [],
        catalog,
        sources,
      ),
    ).toThrow("current active primary topic");
  });

  it("provides an inert fictional example and an empty-pack round trip", () => {
    const sample = exampleRulesYaml(catalog, sources);
    const result = importRulesYaml(sample, [], catalog, sources);
    expect(result[0].name).toBe("Fictional cafe");
    expect(result[0].conditions[0].value).toBe("Fictional Cafe");
    expect(
      importRulesYaml(
        exportRulesYaml([], catalog, sources),
        [],
        catalog,
        sources,
      ),
    ).toEqual([]);
    expect(
      importRulesYaml("version: 1\nrules: []", [existing], catalog, sources),
    ).toEqual([existing]);
  });
});
