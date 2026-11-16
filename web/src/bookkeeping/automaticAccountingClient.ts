import type { components } from "../lib/api.generated";
import { APIError, type API } from "./api";

export type AccountingRule = components["schemas"]["AutomaticAccountingRule"];
export type AccountingConfiguration =
  components["schemas"]["AutomaticAccountingConfiguration"];
export type AccountingPolicy =
  components["schemas"]["AutomaticAccountingPolicy"];
export type AccountingStatus =
  components["schemas"]["AutomaticAccountingStatus"];
export type AccountingSettings =
  components["schemas"]["AutomaticAccountingSettings"];
export type AccountingReason =
  components["schemas"]["AutomaticAccountingReason"];

export const accountingReasons: Record<AccountingReason, string> = {
  unmapped_rule: "Add an accounting mapping",
  ambiguous_rule: "More than one accounting mapping matches",
  outside_policy_interval: "Booked date is outside the configured interval",
  manual_draft: "A saved manual accounting draft is protected",
  source_not_booked: "The source transaction is not booked",
  zero_amount: "Zero-value transactions cannot create an entry",
  source_held: "Resolve the source transaction hold",
  overlap_review_required: "Review overlapping source evidence",
  source_correction_required:
    "Posted evidence changed; a linked correction is needed",
  paired_transfer_required: "Transfer or card payment needs paired evidence",
  split_treatment_required: "Loan treatment needs a reviewed split",
  fx_required: "A reviewed foreign-exchange valuation is needed",
  unsupported_currency:
    "This currency is not supported by the configured treatment",
  unsupported_policy: "The accounting policy is not supported",
  inactive_account: "A mapped account is not active on the booked date",
  closed_period: "The accounting period is closed",
  source_unavailable: "Source evidence is unavailable",
  permission_changed: "Accounting access changed",
  saved_work: "Finish or cancel your saved review first",
  stale_input: "The source or policy changed; waiting for a fresh evaluation",
};
const record = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
const keys = (
  value: unknown,
  names: string[],
): value is Record<string, unknown> =>
  record(value) &&
  Object.keys(value).length === names.length &&
  names.every((name) => Object.hasOwn(value, name));
export const accountingID = (value: unknown): value is string =>
  typeof value === "string" &&
  /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(value) &&
  value !== "00000000-0000-0000-0000-000000000000";
const count = (value: unknown): value is string =>
  typeof value === "string" &&
  /^(0|[1-9][0-9]{0,18})$/.test(value) &&
  BigInt(value) <= 9223372036854775807n;
const text = (value: unknown, max: number): value is string =>
  typeof value === "string" &&
  value.trim().length > 0 &&
  [...value].length <= max;
export const accountingDate = (value: unknown): value is string =>
  typeof value === "string" &&
  /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/.test(value) &&
  value.slice(0, 4) !== "0000" &&
  Number.isFinite(Date.parse(value + "T00:00:00Z")) &&
  new Date(value + "T00:00:00Z").toISOString().slice(0, 10) === value;
const reason = (value: unknown): value is AccountingReason =>
  typeof value === "string" && Object.hasOwn(accountingReasons, value);

// Client validation closes the representation only; SQL owns account authority and exact financial eligibility.
export function validAccountingConfiguration(
  value: unknown,
): value is AccountingConfiguration {
  if (
    !keys(value, [
      "book_policy_id",
      "effective_from",
      "effective_to",
      "rules",
    ]) ||
    !accountingID(value.book_policy_id) ||
    !accountingDate(value.effective_from) ||
    !(
      value.effective_to === null ||
      (accountingDate(value.effective_to) &&
        value.effective_to > value.effective_from)
    ) ||
    !Array.isArray(value.rules) ||
    value.rules.length > 100
  )
    return false;
  return validAccountingRules(value.rules);
}
// The finite rule representation is shared by local YAML authoring and response validation.
export function validAccountingRules(
  value: unknown,
): value is AccountingRule[] {
  if (!Array.isArray(value) || value.length > 100) return false;
  const ids = new Set<string>(),
    names = new Set<string>();
  return value.every((rule) => {
    if (
      !keys(rule, [
        "rule_id",
        "name",
        "source_account_id",
        "direction",
        "description",
        "offset_account_id",
        "treatment",
      ]) ||
      !accountingID(rule.rule_id) ||
      !accountingID(rule.source_account_id) ||
      !accountingID(rule.offset_account_id) ||
      !text(rule.name, 80) ||
      /[\u0000-\u001f\u007f-\u009f]/.test(rule.name) ||
      !text(rule.description, 500) ||
      rule.description.includes("\u0000") ||
      typeof rule.direction !== "string" ||
      !["inflow", "outflow"].includes(rule.direction) ||
      typeof rule.treatment !== "string" ||
      !["expense", "income", "transfer", "card_payment", "loan"].includes(
        rule.treatment,
      ) ||
      ids.has(rule.rule_id) ||
      names.has(rule.name)
    )
      return false;
    ids.add(rule.rule_id);
    names.add(rule.name);
    return true;
  });
}
export function validAccountingPolicy(
  value: unknown,
  entity: string,
  book: string,
): value is AccountingPolicy {
  return (
    keys(value, [
      "profile",
      "entity_id",
      "book_id",
      "revision",
      "enabled",
      "policy",
      "supported_profile",
    ]) &&
    value.profile === "automatic-accounting-policy-v1" &&
    value.entity_id === entity &&
    value.book_id === book &&
    count(value.revision) &&
    typeof value.enabled === "boolean" &&
    value.supported_profile === "source-identity-usd-v1" &&
    (value.revision === "0"
      ? value.policy === null && !value.enabled
      : validAccountingConfiguration(value.policy))
  );
}
export function validAccountingStatus(
  value: unknown,
  entity: string,
  book: string,
): value is AccountingStatus {
  if (
    !keys(value, [
      "profile",
      "entity_id",
      "book_id",
      "enabled",
      "runtime_enabled",
      "policy_revision",
      "pending_count",
      "attention_count",
      "last_operation",
      "holds",
      "holds_truncated",
      "blocked_reason",
    ]) ||
    value.profile !== "automatic-accounting-status-v1" ||
    value.entity_id !== entity ||
    value.book_id !== book ||
    typeof value.enabled !== "boolean" ||
    typeof value.runtime_enabled !== "boolean" ||
    !count(value.policy_revision) ||
    !count(value.pending_count) ||
    !count(value.attention_count) ||
    typeof value.holds_truncated !== "boolean" ||
    !(
      value.blocked_reason === null ||
      (typeof value.blocked_reason === "string" &&
        ["saved_work", "permission_changed", "source_unavailable"].includes(
          value.blocked_reason,
        ))
    ) ||
    !Array.isArray(value.holds) ||
    value.holds.length > 100 ||
    value.holds_truncated !==
      BigInt(value.attention_count) > BigInt(value.holds.length) ||
    BigInt(value.attention_count) < BigInt(value.holds.length)
  )
    return false;
  const ids = new Set<string>();
  if (
    !value.holds.every((hold) => {
      if (
        !keys(hold, [
          "review_id",
          "source_account_id",
          "reason",
          "operation_id",
        ]) ||
        !accountingID(hold.review_id) ||
        !accountingID(hold.source_account_id) ||
        !reason(hold.reason) ||
        !(hold.operation_id === null || accountingID(hold.operation_id)) ||
        ids.has(hold.review_id)
      )
        return false;
      ids.add(hold.review_id);
      return true;
    })
  )
    return false;
  const operation = value.last_operation;
  return (
    operation === null ||
    (keys(operation, [
      "operation_id",
      "state",
      "review_id",
      "journal_id",
      "reason",
    ]) &&
      accountingID(operation.operation_id) &&
      accountingID(operation.review_id) &&
      typeof operation.state === "string" &&
      ["prepared", "committed", "stale", "held"].includes(operation.state) &&
      (operation.journal_id === null || accountingID(operation.journal_id)) &&
      (operation.reason === null || reason(operation.reason)) &&
      (operation.state === "committed"
        ? accountingID(operation.journal_id) && operation.reason === null
        : operation.state === "prepared"
          ? operation.journal_id === null && operation.reason === null
          : operation.journal_id === null && reason(operation.reason)))
  );
}
export const accountingBase = (entity: string, book: string) =>
  `/v1/entities/${encodeURIComponent(entity)}/books/${encodeURIComponent(book)}/automatic-accounting`;

// A read never starts a run; runtime_enabled is a trusted server signal distinct from saved policy.enabled.
export async function readAccounting(api: API, entity: string, book: string) {
  const [policy, status] = await Promise.all([
    api<unknown>(accountingBase(entity, book)),
    api<unknown>(accountingBase(entity, book) + "/status"),
  ]);
  if (
    !validAccountingPolicy(policy, entity, book) ||
    !validAccountingStatus(status, entity, book) ||
    policy.enabled !== status.enabled ||
    policy.revision !== status.policy_revision
  )
    throw new APIError(0, true, "read");
  return { policy, status };
}
// Unknown writes keep their original body/key for explicit retry; the transport never retries automatically.
export async function saveAccounting(
  api: API,
  entity: string,
  book: string,
  body: string,
) {
  const value = await api<unknown>(accountingBase(entity, book), body);
  if (!validAccountingPolicy(value, entity, book)) throw new APIError(0, true);
  return value;
}
