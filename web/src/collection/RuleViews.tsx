import type { Catalog, Row } from "./types";
import { selectionKey } from "./types";
import type {
  RuleApplication,
  RuleDefinition,
  RuleOutput,
  RuleResult,
  RuleSuggestion,
} from "./rules";
import styles from "./collection.module.css";
export function ruleOutputLabel(output: RuleOutput, catalog: Catalog): string {
  const topic =
    output.kind === "uncategorized"
      ? "Explicit Uncategorized"
      : (catalog.topics.find((topic) => topic.topic_id === output.topic_id)
          ?.name ?? `Topic ${output.topic_id}`);
  return (
    topic +
    (output.tag_ids.length
      ? "; tags: " +
        output.tag_ids
          .map(
            (id) => catalog.tags.find((tag) => tag.tag_id === id)?.name ?? id,
          )
          .join(", ")
      : "; no tags")
  );
}
function WinningRules({ rules }: { rules: RuleSuggestion["rules"] }) {
  return rules.length ? (
    <ul>
      {rules.map((rule) => (
        <li key={rule.rule_id}>
          Rule {rule.rule_id}, revision {rule.rule_revision}
        </li>
      ))}
    </ul>
  ) : null;
}
// Current proposals never establish the provenance or eligibility of a historical accepted decision.
export function RuleSuggestionView({
  suggestion,
  catalog,
}: {
  suggestion: RuleSuggestion;
  catalog: Catalog;
}) {
  return (
    <div>
      <p>
        {suggestion.status === "rule"
          ? "Current rule suggestion"
          : suggestion.status === "override"
            ? "Manual decisions take precedence"
            : suggestion.status === "conflict"
              ? "Equal-rank rules disagree; no suggestion is chosen"
              : suggestion.status === "stale_override"
                ? "A manual decision needs confirmation; rules abstain"
                : suggestion.status === "unavailable"
                  ? "Rules are unavailable for this source"
                  : "No current rule matches"}
        .
        {suggestion.output
          ? " " + ruleOutputLabel(suggestion.output, catalog) + "."
          : ""}
      </p>
      <WinningRules rules={suggestion.rules} />
      <p>
        Suggestions are inert. Historical changes require an explicit selected
        preview and confirmation.
      </p>
    </div>
  );
}
export function RuleApplicationView({
  application,
}: {
  application: RuleApplication;
}) {
  if (application.state === "none") return <p>No saved rule application.</p>;
  return (
    <div>
      <p>
        {application.state === "current"
          ? "Saved rule application"
          : application.state === "stale"
            ? "Saved rule application needs confirmation"
            : "Saved rule application is superseded by manual decisions"}
        ; ruleset {application.ruleset_revision}. Effective fields:{" "}
        {[
          application.fields.topic ? "primary topic" : null,
          application.fields.tags ? "tags" : null,
        ]
          .filter(Boolean)
          .join(", ") || "none"}
        .
      </p>
      <WinningRules rules={application.rules} />
    </div>
  );
}
export function RuleCatalogReview({
  rules,
  catalog,
}: {
  rules: RuleDefinition[];
  catalog: Catalog;
}) {
  return (
    <ul className={styles.reviewRows} aria-label="Reviewed rule catalog">
      {rules.map((rule) => (
        <li key={rule.rule_id} className={styles.ruleResult}>
          <strong>
            {rule.name} · {rule.status}
          </strong>
          <span>
            {rule.source_account_id
              ? "Account " + rule.source_account_id
              : "Whole private workspace"}
            ; priority {rule.priority}
          </span>
          <ol>
            {rule.conditions.map((condition, index) => (
              <li key={index}>
                {condition.field} {condition.operator}:{" "}
                <code>{condition.value}</code>
              </li>
            ))}
          </ol>
          <p>{ruleOutputLabel(rule.output, catalog)}</p>
        </li>
      ))}
      {!rules.length && <li>No rules in this edition.</li>}
    </ul>
  );
}
// Render SQL's exact selected result/skip eligibility; do not derive whole-query counts from current rows.
export function RuleResultsView({
  results,
  rows,
  catalog,
  applyCount,
  abstainCount,
}: {
  results: RuleResult[];
  rows: Row[];
  catalog: Catalog;
  applyCount: string;
  abstainCount: string;
}) {
  return (
    <div>
      <p>
        Eligible reviewed results: {applyCount}. Unchanged selected results:{" "}
        {abstainCount}.
      </p>
      <ul className={styles.reviewRows} aria-label="Reviewed rule results">
        {results.map((result) => (
          <li key={selectionKey(result)} className={styles.ruleResult}>
            <strong>
              {rows.find((row) => selectionKey(row) === selectionKey(result))
                ?.description ?? result.observation_id}
            </strong>
            <span>
              {result.status.replaceAll("_", " ")} ·{" "}
              {result.will_apply ? "Eligible reviewed result" : "No change"}
            </span>
            {result.output && <p>{ruleOutputLabel(result.output, catalog)}</p>}
            <p>
              Preserved manual fields:{" "}
              {[
                result.preserved.topic ? "primary topic" : null,
                result.preserved.tags ? "tags" : null,
              ]
                .filter(Boolean)
                .join(", ") || "none"}
              .
            </p>
            <WinningRules rules={result.rules} />
          </li>
        ))}
      </ul>
    </div>
  );
}
