import { useState } from "react";
import { RuleEditor } from "../src/collection/RuleEditor";
import type { RuleDefinition } from "../src/collection/rules";
import { collectionCatalog } from "../tests/collection-fixture";

const source = "e1000000-0000-4000-8000-000000000001";
const topic = "e2000000-0000-4000-8000-000000000001";
const catalog = {
  ...collectionCatalog,
  tags: [],
  topics: [
    {
      topic_id: topic,
      name: "Food and drink",
      parent_id: null,
      status: "active" as const,
    },
    {
      topic_id: "e2000000-0000-4000-8000-000000000002",
      name: "Transportation",
      parent_id: null,
      status: "active" as const,
    },
  ],
  rules: [
    {
      rule_id: "e3000000-0000-4000-8000-000000000001",
      name: "Cafe purchases",
      status: "active",
      source_account_id: source,
      priority: 0,
      conditions: [
        { field: "merchant", operator: "exact", value: "Fictional Cafe" },
      ],
      output: { kind: "topic", topic_id: topic, tag_ids: [] },
    },
    {
      rule_id: "e3000000-0000-4000-8000-000000000002",
      name: "Rail tickets",
      status: "active",
      source_account_id: null,
      priority: 0,
      conditions: [
        { field: "description", operator: "contains", value: "Fictional Rail" },
      ],
      output: {
        kind: "topic",
        topic_id: "e2000000-0000-4000-8000-000000000002",
        tag_ids: [],
      },
    },
  ] as RuleDefinition[],
};

// Standalone fictional authoring review; no server client, auth, providers or persistence.
export function RulePackReview() {
  const [preview, setPreview] = useState<RuleDefinition[] | null>(null);
  return (
    <>
      <div className="preview-banner" role="note">
        <span>
          <strong>Rule workspace review</strong> · Fictional data, changes stay
          in this page
        </span>
        <a href="#">Review transactions</a>
      </div>
      <main
        style={{ maxWidth: 1440, margin: "2rem auto", padding: "0 1.25rem" }}
      >
        <h1 style={{ fontSize: "1.6rem", marginBottom: ".25rem" }}>
          Categorization rules
        </h1>
        <p style={{ color: "#62716e", fontSize: ".9rem" }}>
          Manage recurring patterns in one place. Import YAML to make changes in
          bulk.
        </p>
        <RuleEditor
          catalog={catalog}
          sources={[{ source_id: source, name: "Fictional checking" }]}
          disabled={false}
          onPreview={setPreview}
        />
        {preview && (
          <section aria-label="Fictional change review">
            <h2>Review {preview.length} rules</h2>
            <p>
              This preview checks the authoring flow only. In the signed-in app,
              the server validates and saves your reviewed changes.
            </p>
            <ul>
              {preview.map((rule) => (
                <li key={rule.rule_id}>
                  {rule.name} · {rule.status}
                </li>
              ))}
            </ul>
          </section>
        )}
      </main>
    </>
  );
}
