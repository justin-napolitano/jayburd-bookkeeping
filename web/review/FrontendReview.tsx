import { useMemo, useState } from "react";
import "./review.css";
import {
  FinanceNavigation,
  financeToolLabel,
  type FinanceTool,
} from "../src/bookkeeping/FinanceNavigation";
import { CollectionReview } from "../src/collection/CollectionReview";
import { CollectionOnboarding } from "../src/collection/CollectionOnboarding";
import type { API } from "../src/bookkeeping/api";
import type {
  OnboardingCommand,
  OnboardingPreview,
} from "../src/collection/onboarding";
import {
  commitOnboarding,
  onboardingCatalog,
  onboardingCommand,
  onboardingPending,
  onboardingPreview,
} from "../src/collection/onboarding-fixture";
import {
  collectionCatalog,
  collectionFixture,
  collectionRow,
  collectionSources,
  collectionWorkspace,
} from "../tests/collection-fixture";

// In-memory fictional responses are confined to this review entry, never an application transport.
function sampleSetup(): API {
  let ready = false,
    adopted = false;
  let preview: OnboardingPreview | null = null;
  let saved: OnboardingCommand | null = null;
  return (async (path: string, body?: string) => {
    if (path === "/v1/collection-onboarding") {
      const catalog = onboardingCatalog(ready);
      if (adopted) {
        catalog.candidates = [];
        catalog.can_adopt = false;
      }
      return structuredClone(catalog);
    }
    if (path.endsWith("/pending"))
      return onboardingPending(saved?.acknowledged ? null : saved);
    if (path.endsWith("/previews")) {
      const action = JSON.parse(body!);
      preview = {
        ...onboardingPreview(action.operation === "adopt"),
        preview_id: crypto.randomUUID(),
      };
      return structuredClone(preview);
    }
    if (path === "/v1/collection-onboarding/commands" && preview) {
      saved = {
        ...onboardingCommand(preview, JSON.parse(body!).prepare_key),
        command_id: crypto.randomUUID(),
        recovery_id: crypto.randomUUID(),
      };
      return structuredClone(saved);
    }
    if (path.endsWith("/execute") && saved) {
      saved = commitOnboarding(saved);
      ready = true;
      if (saved.preview.operation === "adopt") adopted = true;
      return structuredClone(saved);
    }
    if (path.endsWith("/cancel") && saved) {
      saved = { ...saved, state: "cancelled", result: { cancelled: true } };
      return structuredClone(saved);
    }
    if (path.endsWith("/acknowledge") && saved) {
      saved = { ...saved, acknowledged: true };
      return structuredClone(saved);
    }
    throw new Error("This action is outside the fictional review.");
  }) as API;
}

const purchases = [
  ["Studio supplies", "Fictional Paper Co.", "48.90"],
  ["Team lunch", "Fictional Green Kitchen", "36.25"],
  ["Workspace subscription", "Fictional Desk", "24.00"],
  ["Train ticket", "Fictional Rail", "18.50"],
  ["Coffee with project team", "Fictional Cafe", "9.75"],
  [
    "Long description to check wrapping and controls in a smaller window",
    "Fictional Market",
    "129.99",
  ],
];

// Real navigation, review and recovery components receive only explicit synthetic ports.
export function FrontendReview() {
  const [tool, setTool] = useState<FinanceTool>("All transactions");
  const [setup, setSetup] = useState(false);
  const [locked, setLocked] = useState(false);
  const [version, setVersion] = useState(0);
  const [message, setMessage] = useState("");
  const fixture = useMemo(
    () =>
      collectionFixture({
        rows: purchases.map(([description, merchant, amount], index) => {
          const row = collectionRow(index, {
            description,
            merchant,
            booked_date: `2026-11-0${7 - index}`,
          });
          if (row.money.state === "available") row.money.amount = amount;
          return row;
        }),
      }),
    [version],
  );
  const setupAPI = useMemo(() => sampleSetup(), [version]);
  const reset = () => {
    if (!locked) {
      setVersion((v) => v + 1);
      setMessage("");
    }
  };
  const navigate = (next: FinanceTool) => {
    setTool(next);
    setSetup(false);
    setMessage("");
  };
  return (
    <div className="review-preview">
      <div className="preview-banner" role="note">
        <span>
          <strong>Design review</strong> · Fictional data, held in this page
          only
        </span>
        <button
          type="button"
          disabled={locked}
          onClick={() => {
            setSetup((v) => !v);
            setTool("All transactions");
          }}
        >
          {" "}
          {setup ? "Review sample transactions" : "Try workspace setup"}
        </button>
        <button type="button" disabled={locked} onClick={reset}>
          Reset sample
        </button>
      </div>
      <div className="finance-workspace">
        <FinanceNavigation
          tool={tool}
          locked={locked}
          loading={false}
          onNavigate={navigate}
          onRefresh={() =>
            setMessage(
              "The fictional sample is current. No bank request was made.",
            )
          }
        />
        <div className="finance-main">
          <div className="preview-context">
            <span>Personal workspace</span>
            <span>Source activity · USD</span>
          </div>
          {message && <p role="status">{message}</p>}
          {setup ? (
            <CollectionOnboarding
              key={version}
              api={setupAPI}
              active
              externalLocked={false}
              onLockChange={setLocked}
              onChanged={() =>
                setMessage(
                  "Sample setup updated. Continue to review or adopt its fictional existing connection.",
                )
              }
            />
          ) : tool === "All transactions" ? (
            <CollectionReview
              key={version}
              workspace={collectionWorkspace}
              sources={collectionSources}
              catalog={collectionCatalog}
              port={fixture.port}
              onLockChange={setLocked}
              onDenied={() =>
                setMessage("The fictional review is unavailable.")
              }
            />
          ) : tool === "Connections" ? (
            <section className="preview-accounts" aria-label="Sample accounts">
              <h1>Accounts</h1>
              <p>
                Recognizable accounts and their connection status, with setup in
                one place.
              </p>
              <ul>
                {collectionSources.map((source) => (
                  <li key={source.source_id}>
                    <div>
                      <strong>{source.name}</strong>
                      <p>Fictional account · Stored sample history</p>
                    </div>
                    <button
                      className="secondary"
                      onClick={() => navigate("All transactions")}
                    >
                      Review transactions
                    </button>
                  </li>
                ))}
              </ul>
              <button
                onClick={() => {
                  setSetup(true);
                  setTool("All transactions");
                }}
              >
                Review workspace setup
              </button>
              <p className="preview-note">
                Preview only. Account management in the signed-in application
                still uses its existing authorized Connections workflow.
              </p>
            </section>
          ) : (
            <section aria-label="Review navigation destination">
              <h1>{financeToolLabel(tool)}</h1>
              <p>
                This review page focuses on transaction reading and workspace
                setup. The signed-in app retains its existing authorized{" "}
                {financeToolLabel(tool).toLowerCase()} workflow.
              </p>
              <button onClick={() => navigate("All transactions")}>
                Review sample transactions
              </button>
            </section>
          )}
        </div>
      </div>
    </div>
  );
}
