import type { components } from "../lib/api.generated";
// Shared transport validation keeps automatic/manual reads on the same owned status projection.
export type StreamStatus = components["schemas"]["ConnectionStreamStatus"];
const revision = /^[1-9][0-9]*$/;
export const itemStates = [
  "active",
  "paused",
  "disconnected",
  "credential_recovery_required",
  "disconnect_pending",
  "reauthorization_required",
];
// A status is useful only for the exact currently selected Item/source/stream scope.
export const checkedStatus = (
  value: StreamStatus,
  expected: {
    stream_id: string;
    source_account_id: string;
    sync_item_id: string;
  },
) => {
  if (
    !value ||
    value.stream_id !== expected.stream_id ||
    value.source_account_id !== expected.source_account_id ||
    value.sync_item_id !== expected.sync_item_id ||
    !["active", "paused", "disconnected"].includes(value.state) ||
    !itemStates.includes(value.item_state) ||
    typeof value.stream_revision !== "string" ||
    !revision.test(value.stream_revision) ||
    typeof value.item_generation !== "string" ||
    !revision.test(value.item_generation) ||
    typeof value.binding_revision !== "string" ||
    !revision.test(value.binding_revision) ||
    typeof value.requested_generation !== "string" ||
    !revision.test(value.requested_generation) ||
    typeof value.completed_generation !== "string" ||
    !/^(0|[1-9][0-9]*)$/.test(value.completed_generation) ||
    BigInt(value.completed_generation) > BigInt(value.requested_generation) ||
    typeof value.queued !== "boolean" ||
    typeof value.lease_active !== "boolean" ||
    (value.delivery_state != null &&
      !["ready", "running", "retry_wait", "succeeded", "quarantined"].includes(
        value.delivery_state,
      )) ||
    (value.decoder_error != null &&
      ![
        "invalid_json",
        "duplicate_key",
        "limit_exceeded",
        "invalid_number",
        "invalid_expectation",
        "invalid_field",
        "unknown_field",
        "unknown_enum",
        "invalid_cursor_encoding",
        "invalid_cursor_progress",
        "account_mismatch",
        "unsupported_account",
        "invalid_identity",
        "duplicate_identity",
        "invalid_lineage",
        "invalid_date",
        "unsupported_currency",
        "amount_out_of_range",
        "amount_magnitude_exceeded",
        "amount_scale_exceeded",
        "currency_precision_exceeded",
      ].includes(value.decoder_error)) ||
    value.provider_updated_at !== null ||
    (value.last_completed_at !== null &&
      !Number.isFinite(Date.parse(value.last_completed_at))) ||
    ![
      "TRANSACTIONS_UPDATE_STATUS_UNKNOWN",
      "NOT_READY",
      "INITIAL_UPDATE_COMPLETE",
      "HISTORICAL_UPDATE_COMPLETE",
    ].includes(value.readiness) ||
    (value.last_error !== null &&
      ![
        "pagination_mutation",
        "provider_failure",
        "decoder_rejected",
        "cancelled",
        "item_changed",
        "lease_expired",
        "account_disconnected",
        "account_paused",
      ].includes(value.last_error))
  )
    throw Error("response");
  return value;
};
