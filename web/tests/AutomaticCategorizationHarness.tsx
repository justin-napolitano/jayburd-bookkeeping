import { useMemo, useState } from "react";
import { CollectionReview } from "../src/collection/CollectionReview";
import {
  FinanceNavigation,
  type FinanceTool,
} from "../src/bookkeeping/FinanceNavigation";
import { CollectionFailure, type Query } from "../src/collection/types";
import type { AutomationStatus } from "../src/collection/automation";
import {
  collectionCatalog,
  collectionFixture,
  collectionPage,
  collectionRow,
  collectionSources,
  collectionWorkspace,
} from "./collection-fixture";
const workspace = {
  ...collectionWorkspace,
  workspace_id: "88000000-0000-4000-8000-000000000001",
};
const catalog = {
  ...collectionCatalog,
  automatic_categorization: { state: "available" as const },
};
// Fixed synthetic accepted outputs exercise rendering/recovery, not another matcher or runtime store.
export function AutomaticCategorizationHarness() {
  const [locked, setLocked] = useState(true),
    [tool, setTool] = useState<FinanceTool>("All transactions"),
    [requests, setRequests] = useState("[]"),
    [writes, setWrites] = useState(0);
  const fixture = useMemo(() => {
    const rows = Array.from({ length: 30 }, (_, i) =>
      collectionRow(i, {
        workspace_id: workspace.workspace_id,
        owner_topic:
          i === 2
            ? { state: "unclassified" }
            : {
                state: "topic",
                topic_id: "dining",
                name: i === 1 ? "Your saved category" : "Dining",
              },
        categorization:
          i === 2
            ? {
                state: "needs_attention",
                origin: "none",
                reason: "low_confidence",
                rule_names: [],
                provider: null,
                decision_id: "88000000-0000-4000-8000-000000000002",
              }
            : i === 1
              ? {
                  state: "manual",
                  origin: "manual",
                  reason: null,
                  rule_names: [],
                  provider: null,
                  decision_id: null,
                }
              : {
                  state: "automatic",
                  origin: "provider",
                  reason: null,
                  rule_names: [],
                  provider: {
                    name: "Plaid",
                    version: "v2",
                    primary: "FOOD_AND_DRINK",
                    detailed: "FOOD_AND_DRINK_COFFEE",
                    confidence: "HIGH",
                    map_edition: "plaid-pfc-primary-2026-11-11",
                  },
                  decision_id: "88000000-0000-4000-8000-000000000002",
                },
      }),
    );
    const base = collectionFixture({ rows });
    let enabled = true,
      drop = true;
    const recorded: unknown[] = [];
    const status = (): AutomationStatus => ({
      profile: "private-collection-automation-v1",
      workspace_id: workspace.workspace_id,
      revision: enabled ? "1" : "2",
      enabled,
      processing_state: enabled ? "idle" : "disabled",
      pending_count: "0",
      attention_count: "1",
      blocked_reason: null,
      last_run: null,
    });
    return {
      ...base,
      port: {
        ...base.port,
        query: async (q: Query) =>
          q.topic.kind === "needs_attention" && q.status.source === "booked"
            ? collectionPage(q, [rows[2]])
            : base.port.query(q),
        automation: {
          read: async () => status(),
          configure: async (_workspace: string, body: unknown) => {
            recorded.push(body);
            setRequests(JSON.stringify(recorded));
            if (drop) {
              drop = false;
              enabled = false;
              setWrites((n) => n + 1);
              throw new CollectionFailure("uncertain");
            }
            return status();
          },
        },
      },
    };
  }, []);
  return (
    <>
      <h1>Fictional automatic categories</h1>
      <div className="finance-workspace">
        <FinanceNavigation
          tool={tool}
          locked={locked}
          loading={false}
          onNavigate={setTool}
          onRefresh={() => {}}
        />
        <div className="finance-main">
          <CollectionReview
            workspace={workspace}
            catalog={catalog}
            sources={collectionSources}
            port={fixture.port}
            active={tool === "All transactions"}
            onLockChange={setLocked}
            onDenied={() => {}}
          />
        </div>
      </div>
      <output aria-label="Fictional automation requests" hidden>
        {requests}
      </output>
      <output aria-label="Fictional configuration effects" hidden>
        {writes}
      </output>
    </>
  );
}
