import { useState } from "react";
import { RuleEditor } from "../src/collection/RuleEditor";
import type { RuleDefinition } from "../src/collection/rules";
import { collectionCatalog } from "./collection-fixture";

const catalog = {
  ...collectionCatalog,
  topics: [
    {
      topic_id: "d1000000-0000-4000-8000-000000000001",
      name: "Dining",
      parent_id: null,
      status: "active" as const,
    },
  ],
  tags: [],
};
// The harness observes inert previews only; it cannot invoke application persistence.
export function RuleWorkspaceHarness() {
  const [preview, setPreview] = useState<RuleDefinition[] | null>(null);
  return (
    <main style={{ maxWidth: 1400, padding: "1rem" }}>
      <RuleEditor
        catalog={catalog}
        sources={[]}
        disabled={false}
        onPreview={setPreview}
      />
      <output hidden aria-label="Synthetic rule preview">
        {preview ? JSON.stringify(preview) : "No preview"}
      </output>
    </main>
  );
}
