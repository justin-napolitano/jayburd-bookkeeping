import { useMemo, useState } from "react";
import { CollectionReview } from "../src/collection/CollectionReview";
import {
  CollectionFailure,
  type CollectionPort,
  type LabelCommand,
  type Preview,
} from "../src/collection/types";
import type { RuleSeed } from "../src/collection/rules";
import {
  collectionCatalog,
  collectionFixture,
  collectionPreview,
  collectionRow,
  collectionSources,
  collectionWorkspace,
} from "./collection-fixture";
const id = (n: number) =>
  `87000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
const rules = [
  {
    rule_id: id(1),
    name: "Fictional account rule",
    status: "active" as const,
    source_account_id: null,
    priority: 0,
    conditions: [
      {
        field: "merchant" as const,
        operator: "exact" as const,
        value: "Fictional Cafe",
      },
    ],
    output: { kind: "uncategorized" as const, tag_ids: [] },
  },
  {
    rule_id: id(2),
    name: "Fictional conflicting rule",
    status: "active" as const,
    source_account_id: null,
    priority: 0,
    conditions: [
      {
        field: "merchant" as const,
        operator: "exact" as const,
        value: "Fictional Cafe",
      },
    ],
    output: { kind: "topic" as const, topic_id: id(3), tag_ids: [] },
  },
];
rules.push({
  rule_id: id(5),
  name: "Fictional corner rule",
  status: "active",
  source_account_id: null,
  priority: 0,
  conditions: [
    { field: "merchant", operator: "exact", value: "Fictional Corner" },
  ],
  output: { kind: "uncategorized", tag_ids: [] },
});
const catalog = {
  ...collectionCatalog,
  topics: [
    {
      topic_id: id(3),
      name: "Manual topic",
      parent_id: null,
      status: "active" as const,
    },
  ],
  rules,
};

const emptyCatalog = { ...catalog, rules: [] };

// Explicit expected results exercise UI/recovery only; this fixture is not a second matcher or application store.
export function CollectionRulesHarness({
  scenario = "results",
}: {
  scenario?: "results" | "catalog" | "seed";
}) {
  const [executions, setExecutions] = useState(0),
    [acks, setAcks] = useState(0),
    [version, setVersion] = useState(0),
    [seed, setSeed] = useState<RuleSeed | null>(null),
    [prepares, setPrepares] = useState("");
  const fixture = useMemo(() => {
    const rows = [
      collectionRow(0, {
        owner_topic: { state: "topic", topic_id: id(3), name: "Manual topic" },
        rule_suggestion: {
          status: "conflict",
          output: null,
          rules: rules
            .slice(0, 2)
            .map((rule) => ({ rule_id: rule.rule_id, rule_revision: "4" })),
          preserved: { topic: false, tags: false },
          will_apply: false,
        },
        rule_application: {
          state: "stale",
          ruleset_revision: "3",
          rules: [{ rule_id: id(1), rule_revision: "3" }],
          fields: { topic: true, tags: false },
        },
      }),
      collectionRow(1, {
        merchant: "Fictional Corner",
        owner_topic: { state: "topic", topic_id: id(3), name: "Manual topic" },
        rule_suggestion: {
          status: "rule",
          output: { kind: "topic", topic_id: id(3), tag_ids: [] },
          rules: [{ rule_id: id(5), rule_revision: "4" }],
          preserved: { topic: true, tags: false },
          will_apply: false,
        },
      }),
    ];
    const base = collectionFixture({ rows });
    let latest: Preview | null = null,
      saved: LabelCommand | null = null,
      dropPrepare = scenario === "results",
      dropAck = scenario === "seed";
    const keys: unknown[] = [];
    const port: CollectionPort = {
      ...base.port,
      preview: async (request) => {
        latest = collectionPreview(
          request,
          request.selection.map(
            (item) =>
              rows.find((row) => row.observation_id === item.observation_id)!,
          ),
        );
        if (request.action.kind === "rules_apply") {
          latest.rule_results = request.selection.map((item, index) =>
            index === 0
              ? {
                  ...item,
                  status: "conflict",
                  output: null,
                  rules: rules.slice(0, 2).map((rule) => ({
                    rule_id: rule.rule_id,
                    rule_revision: "4",
                  })),
                  preserved: { topic: false, tags: false },
                  will_apply: false,
                }
              : {
                  ...item,
                  status: "rule",
                  output: { kind: "topic", topic_id: id(3), tag_ids: [] },
                  rules: [{ rule_id: id(5), rule_revision: "4" }],
                  preserved: { topic: true, tags: false },
                  will_apply: true,
                },
          );
          latest.apply_count = String(
            latest.rule_results.filter((result) => result.will_apply).length,
          );
          latest.abstain_count = String(
            latest.rule_results.length - Number(latest.apply_count),
          );
        }
        return structuredClone(latest);
      },
      prepare: async (request) => {
        keys.push(request);
        setPrepares(JSON.stringify(keys));
        if (!saved)
          saved = {
            workspace_id: request.workspace_id,
            command_id: id(4),
            prepare_key: request.prepare_key,
            state: "prepared",
            acknowledged: false,
            preview: structuredClone(latest!),
          };
        if (dropPrepare) {
          dropPrepare = false;
          throw new CollectionFailure("uncertain");
        }
        return structuredClone(saved);
      },
      pending: async () =>
        saved && !saved.acknowledged ? structuredClone(saved) : null,
      execute: async () => {
        saved = { ...saved!, state: "committed" };
        setExecutions((n) => n + 1);
        return structuredClone(saved);
      },
      cancel: async () => {
        saved = { ...saved!, state: "cancelled" };
        return structuredClone(saved);
      },
      acknowledge: async () => {
        saved = { ...saved!, acknowledged: true };
        if (dropAck) {
          dropAck = false;
          throw new CollectionFailure("uncertain");
        }
        setAcks((n) => n + 1);
        const result = structuredClone(saved);
        saved = null;
        return result;
      },
    };
    return { port };
  }, [scenario]);
  return (
    <main>
      <h1>Fictional rule review</h1>
      <CollectionReview
        key={version}
        workspace={collectionWorkspace}
        sources={collectionSources}
        catalog={scenario === "catalog" ? emptyCatalog : catalog}
        port={fixture.port}
        ruleSeed={seed}
        onSeedConsumed={() => setSeed(null)}
        onAcknowledged={(value) => {
          setSeed(value ?? null);
          setVersion((n) => n + 1);
        }}
        onDenied={() => setSeed(null)}
      />
      <output aria-label="Fictional rule executions">{executions}</output>
      <output aria-label="Fictional rule acknowledgements">{acks}</output>
      <output
        aria-label="Fictional rule prepares"
        style={{ display: "block", overflowWrap: "anywhere" }}
      >
        {prepares}
      </output>
    </main>
  );
}
