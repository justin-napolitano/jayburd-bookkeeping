import type { Catalog, ReviewRow } from "./types";
import { categorizationOf } from "./automation";
const reasons: Record<string, string> = {
  missing_provider_category: "No bank category was supplied",
  version_missing: "The bank category version is missing",
  unsupported_version: "The bank category version is unsupported",
  confidence_missing: "Category confidence was not supplied",
  confidence_unknown: "Category confidence is unknown",
  confidence_unrecognized: "Category confidence is unrecognized",
  low_confidence: "The bank category needs your review",
  unmapped_category: "The bank category has no approved match",
  conflicting_rules: "Your rules disagree",
  stale_override: "Check your earlier choice",
  retired_category: "The earlier category is retired",
  source_not_ready: "Waiting for source information",
  unavailable: "Category details are unavailable",
};
// Current labels come from persisted effective fields, never an inert provider/rule suggestion.
export function CategorizationSummary({
  row,
  automaticAvailable = false,
}: {
  row: ReviewRow;
  automaticAvailable?: boolean;
}) {
  const metadata = categorizationOf(row.categorization);
  const c =
    metadata?.state === "automatic" &&
    !["topic", "uncategorized"].includes(row.owner_topic.state)
      ? {
          ...metadata,
          state: "unavailable" as const,
          reason: "unavailable" as const,
          origin: "none" as const,
        }
      : metadata;
  const category =
    row.owner_topic.state === "topic"
      ? row.owner_topic.name
      : row.owner_topic.state === "uncategorized"
        ? "Uncategorized"
        : row.owner_topic.state === "stale"
          ? "Needs confirmation"
          : c?.state === "pending"
            ? automaticAvailable
              ? "Categorizing…"
              : "Not categorized"
            : c?.state === "unavailable"
              ? "Category unavailable"
              : row.source_state === "booked"
                ? "Needs category"
                : "Not categorized";
  const origin =
    c?.state === "automatic"
      ? c.origin === "provider"
        ? "Automatic · Bank category"
        : "Automatic · Your rule"
      : c?.state === "manual"
        ? "Your choice"
        : c?.state === "pending"
          ? automaticAvailable
            ? "Categorization in progress"
            : null
          : c?.reason
            ? reasons[c.reason]
            : row.rule_application.state === "current" &&
                row.rule_application.fields.topic
              ? "Saved rule category"
              : row.owner_topic.state === "topic" ||
                  row.owner_topic.state === "uncategorized"
                ? "Saved category"
                : null;
  return (
    <div className="transaction-category">
      <strong>{category}</strong>
      {origin && <span>{origin}</span>}
      {c?.state === "automatic" && c.rule_names.length > 0 && (
        <span>{c.rule_names.join(", ")}</span>
      )}
    </div>
  );
}
// A current catalog name can explain only that same immutable ruleset edition.
export function winningRuleNames(
  rules: { rule_id: string; rule_revision: string }[],
  catalog?: Catalog,
): string[] {
  return rules.map((rule) =>
    rule.rule_revision === catalog?.ruleset_revision
      ? (catalog.rules.find((item) => item.rule_id === rule.rule_id)?.name ??
        "An earlier saved rule")
      : "An earlier saved rule",
  );
}
