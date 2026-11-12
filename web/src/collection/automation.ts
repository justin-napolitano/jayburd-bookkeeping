import type { API } from "../bookkeeping/api";
import { APIError } from "../bookkeeping/api";
import { CollectionFailure } from "./types";

export type AutomaticRun = {
  profile: "private-collection-auto-run-v1";
  workspace_id: string;
  run_id: string;
  state: "prepared" | "committed" | "stale";
  processed_count: string;
  attention_count: string;
};
export type AutomationStatus = {
  profile: "private-collection-automation-v1";
  workspace_id: string;
  revision: string;
  enabled: boolean;
  processing_state: "disabled" | "idle" | "pending" | "blocked";
  pending_count: string;
  attention_count: string;
  blocked_reason:
    | null
    | "saved_work"
    | "category_catalog_unavailable"
    | "permission_changed";
  last_run: AutomaticRun | null;
};
export type AutomationRequest = {
  key: string;
  expected_revision: string;
  enabled: boolean;
};
export type AutomationPort = {
  read(workspace: string): Promise<AutomationStatus>;
  configure(
    workspace: string,
    request: AutomationRequest,
  ): Promise<AutomationStatus>;
};
export const automationReasons = [
  "missing_provider_category",
  "version_missing",
  "unsupported_version",
  "confidence_missing",
  "confidence_unknown",
  "confidence_unrecognized",
  "low_confidence",
  "unmapped_category",
  "conflicting_rules",
  "stale_override",
  "retired_category",
  "source_not_ready",
  "unavailable",
] as const;
export type Categorization = {
  state: "automatic" | "manual" | "needs_attention" | "pending" | "unavailable";
  origin: "rule" | "provider" | "manual" | "none";
  reason: (typeof automationReasons)[number] | null;
  rule_names: string[];
  provider: null | {
    name: string;
    version: string;
    primary: string;
    detailed: string;
    confidence: string;
    map_edition: string;
  };
  decision_id: string | null;
};
const record = (v: unknown): v is Record<string, unknown> =>
  !!v && typeof v === "object" && !Array.isArray(v);
const text = (v: unknown, max = 200): v is string =>
  typeof v === "string" &&
  Array.from(v).length > 0 &&
  Array.from(v).length <= max;
const exactKeys = (v: Record<string, unknown>, names: string[]) =>
  Object.keys(v).length === names.length &&
  names.every((name) => Object.hasOwn(v, name));
const revision = (v: unknown): v is string =>
  typeof v === "string" && /^(0|[1-9][0-9]{0,18})$/.test(v);
const id = (v: unknown): v is string =>
  typeof v === "string" &&
  /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(v) &&
  v !== "00000000-0000-0000-0000-000000000000";
// A malformed additive annotation never hides valid source facts or becomes evidence of an automatic effect.
export function categorizationOf(value: unknown): Categorization | undefined {
  if (value === undefined) return undefined;
  const unavailable: Categorization = {
    state: "unavailable",
    origin: "none",
    reason: "unavailable",
    rule_names: [],
    provider: null,
    decision_id: null,
  };
  if (
    !record(value) ||
    !exactKeys(value, [
      "state",
      "origin",
      "reason",
      "rule_names",
      "provider",
      "decision_id",
    ]) ||
    ![
      "automatic",
      "manual",
      "needs_attention",
      "pending",
      "unavailable",
    ].includes(String(value.state)) ||
    !["rule", "provider", "manual", "none"].includes(String(value.origin)) ||
    !(
      value.reason === null ||
      automationReasons.includes(
        value.reason as (typeof automationReasons)[number],
      )
    ) ||
    !Array.isArray(value.rule_names) ||
    value.rule_names.length > 100 ||
    !value.rule_names.every((v) => text(v, 80)) ||
    !(value.decision_id === null || id(value.decision_id))
  )
    return unavailable;
  if (value.provider !== null) {
    const provider = value.provider;
    if (
      !record(provider) ||
      !exactKeys(provider, [
        "name",
        "version",
        "primary",
        "detailed",
        "confidence",
        "map_edition",
      ]) ||
      provider.name !== "Plaid" ||
      provider.map_edition !== "plaid-pfc-primary-2026-11-11" ||
      !text(provider.version, 16) ||
      !/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(provider.version) ||
      !text(provider.primary, 128) ||
      !/^[A-Z][A-Z0-9_]*$/.test(provider.primary) ||
      !text(provider.detailed, 128) ||
      !/^[A-Z][A-Z0-9_]*$/.test(provider.detailed) ||
      !text(provider.confidence, 32) ||
      !/^[A-Z][A-Z0-9_]*$/.test(provider.confidence)
    )
      return unavailable;
    if (
      value.state === "automatic" &&
      value.origin === "provider" &&
      (!["HIGH", "VERY_HIGH"].includes(provider.confidence) ||
        !["v1", "v2"].includes(provider.version))
    )
      return unavailable;
  }
  if (
    value.state === "automatic" &&
    (!id(value.decision_id) ||
      value.reason !== null ||
      (value.origin !== "rule" && value.origin !== "provider") ||
      (value.origin === "rule" && value.rule_names.length === 0) ||
      (value.origin === "provider" && value.provider === null))
  )
    return unavailable;
  if (value.state === "manual" && value.origin !== "manual") return unavailable;
  return value as Categorization;
}
// Status is bound to the requested workspace; unknown/malformed counts are never zero.
export function validAutomationStatus(
  v: unknown,
  workspace: string,
): v is AutomationStatus {
  if (
    !record(v) ||
    !exactKeys(v, [
      "profile",
      "workspace_id",
      "revision",
      "enabled",
      "processing_state",
      "pending_count",
      "attention_count",
      "blocked_reason",
      "last_run",
    ]) ||
    v.profile !== "private-collection-automation-v1" ||
    v.workspace_id !== workspace ||
    !id(workspace) ||
    !revision(v.revision) ||
    typeof v.enabled !== "boolean" ||
    !["disabled", "idle", "pending", "blocked"].includes(
      String(v.processing_state),
    ) ||
    !revision(v.pending_count) ||
    !revision(v.attention_count) ||
    !(
      v.blocked_reason === null ||
      [
        "saved_work",
        "category_catalog_unavailable",
        "permission_changed",
      ].includes(String(v.blocked_reason))
    )
  )
    return false;
  if (
    (!v.enabled && v.processing_state !== "disabled") ||
    (v.enabled && v.processing_state === "disabled")
  )
    return false;
  if (v.last_run === null) return true;
  const r = v.last_run;
  return (
    record(r) &&
    exactKeys(r, [
      "profile",
      "workspace_id",
      "run_id",
      "state",
      "processed_count",
      "attention_count",
    ]) &&
    r.profile === "private-collection-auto-run-v1" &&
    r.workspace_id === workspace &&
    id(r.run_id) &&
    ["prepared", "committed", "stale"].includes(String(r.state)) &&
    revision(r.processed_count) &&
    revision(r.attention_count)
  );
}
// Configuration uses the ordinary authenticated API transport, never a read-only query or caller actor ID.
export function createAutomationPort(api: API): AutomationPort {
  const call = async (
    workspace: string,
    request?: AutomationRequest,
  ): Promise<AutomationStatus> => {
    if (
      !id(workspace) ||
      (request &&
        (!/^[A-Za-z0-9._:-]{1,128}$/.test(request.key) ||
          !revision(request.expected_revision) ||
          typeof request.enabled !== "boolean"))
    )
      throw new CollectionFailure("rejected");
    try {
      const result = await api<unknown>(
        `/v1/collections/${encodeURIComponent(workspace)}/automation`,
        request === undefined ? undefined : JSON.stringify(request),
      );
      if (
        !validAutomationStatus(result, workspace) ||
        (request && result.enabled !== request.enabled)
      )
        throw new CollectionFailure(request ? "uncertain" : "unavailable");
      return result;
    } catch (reason) {
      if (reason instanceof CollectionFailure) throw reason;
      if (reason instanceof APIError) {
        if ([401, 403, 404].includes(reason.status))
          throw new CollectionFailure("denied");
        if (request && reason.uncertain)
          throw new CollectionFailure("uncertain");
        if (reason.status === 409) throw new CollectionFailure("stale");
        if ([400, 422].includes(reason.status))
          throw new CollectionFailure("rejected");
      }
      throw new CollectionFailure(request ? "uncertain" : "unavailable");
    }
  };
  return {
    read: (workspace) => call(workspace),
    configure: (workspace, request) => call(workspace, request),
  };
}
