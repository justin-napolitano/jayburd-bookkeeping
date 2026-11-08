import { APIError, type API } from "../bookkeeping/api";

export type OnboardingCandidate = {
  candidate_id: string;
  entity_id: string;
  book_id: string;
  entity_name: string;
  book_name: string;
  snapshot: string;
  sources: { source_id: string; display_name: string }[];
};
export type OnboardingCatalog = {
  profile: "private-onboarding-catalog-v1";
  workspace:
    | { state: "absent" }
    | { state: "unavailable" }
    | {
        state: "ready";
        workspace_id: string;
        authority_revision: string;
        capabilities: string[];
      };
  can_setup: boolean;
  can_adopt: boolean;
  candidates: OnboardingCandidate[];
};
export type OnboardingAction =
  | { operation: "setup" }
  | { operation: "adopt"; candidate_id: string; snapshot: string };
export type OnboardingPreview = {
  profile: "private-onboarding-preview-v1";
  preview_id: string;
  expires_at: string;
  snapshot: string;
} & (
  | {
      operation: "setup";
      workspace_id: null;
      review: { capabilities: string[] };
    }
  | {
      operation: "adopt";
      workspace_id: string;
      review: OnboardingCandidate & {
        expected_generation: string;
        impact: "pause_legacy_acquisition";
      };
    }
);
export type OnboardingCommand = {
  profile: "private-onboarding-command-v1";
  command_id: string;
  recovery_id: string;
  prepare_key: string;
  state: "prepared" | "committed" | "cancelled";
  preview: OnboardingPreview;
  result: Record<string, unknown> | null;
  acknowledged: boolean;
};
export type OnboardingPending = {
  profile: "private-onboarding-pending-v1";
  state: "none" | "available" | "inaccessible" | "blocked";
  command: OnboardingCommand | null;
  recovery_id: string | null;
};
export class OnboardingFailure extends Error {
  constructor(
    public kind:
      | "denied"
      | "rejected"
      | "conflict"
      | "uncertain"
      | "unavailable"
      | "unsupported",
  ) {
    super(kind);
  }
}
const id = (v: unknown): v is string =>
  typeof v === "string" &&
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(v) &&
  v !== "00000000-0000-0000-0000-000000000000";
const hash = (v: unknown): v is string =>
  typeof v === "string" && /^[0-9a-f]{64}$/.test(v);
const rev = (v: unknown): v is string =>
  typeof v === "string" &&
  /^[1-9][0-9]{0,18}$/.test(v) &&
  BigInt(v) <= 9223372036854775807n;
const obj = (v: unknown): v is Record<string, unknown> =>
  !!v && typeof v === "object" && !Array.isArray(v);
const keys = (v: unknown, names: string[]): v is Record<string, unknown> =>
  obj(v) &&
  Object.keys(v).length === names.length &&
  names.every((k) => Object.hasOwn(v, k));
const text = (v: unknown, max: number) =>
  typeof v === "string" &&
  [...v].length > 0 &&
  [...v].length <= max &&
  v.trim() !== "" &&
  !/[\u0000-\u001f\u007f]/.test(v);
const caps = (v: unknown): v is string[] =>
  Array.isArray(v) &&
  v.every((c) => ["read", "acquire", "classify"].includes(c)) &&
  new Set(v).size === v.length;
const sources = (v: unknown) =>
  Array.isArray(v) &&
  v.length > 0 &&
  v.length <= 25 &&
  v.every(
    (s) =>
      keys(s, ["source_id", "display_name"]) &&
      id(s.source_id) &&
      text(s.display_name, 200),
  ) &&
  new Set(v.map((s) => s.source_id)).size === v.length;
const candidateFields = [
  "candidate_id",
  "entity_id",
  "book_id",
  "entity_name",
  "book_name",
  "snapshot",
  "sources",
];
function candidate(v: unknown, extra: string[] = []): v is OnboardingCandidate {
  return (
    keys(v, [...candidateFields, ...extra]) &&
    id(v.candidate_id) &&
    id(v.entity_id) &&
    id(v.book_id) &&
    text(v.entity_name, 200) &&
    text(v.book_name, 200) &&
    hash(v.snapshot) &&
    sources(v.sources)
  );
}
// Validate closed authority-bearing responses before rendering any private metadata.
export function validOnboardingCatalog(v: unknown): v is OnboardingCatalog {
  if (
    !keys(v, [
      "profile",
      "workspace",
      "can_setup",
      "can_adopt",
      "candidates",
    ]) ||
    v.profile !== "private-onboarding-catalog-v1" ||
    typeof v.can_setup !== "boolean" ||
    typeof v.can_adopt !== "boolean" ||
    !Array.isArray(v.candidates) ||
    v.candidates.length > 100 ||
    !v.candidates.every((c) => candidate(c)) ||
    new Set(v.candidates.map((c) => c.candidate_id)).size !==
      v.candidates.length
  )
    return false;
  const w = v.workspace;
  if (!obj(w)) return false;
  if (w.state === "ready") {
    if (
      !keys(w, [
        "state",
        "workspace_id",
        "authority_revision",
        "capabilities",
      ]) ||
      !id(w.workspace_id) ||
      !rev(w.authority_revision) ||
      !caps(w.capabilities)
    )
      return false;
  } else if (
    !keys(w, ["state"]) ||
    !["absent", "unavailable"].includes(String(w.state))
  )
    return false;
  return (
    (!v.can_setup || w.state === "absent") &&
    (!v.can_adopt ||
      (w.state === "ready" &&
        (w.capabilities as string[]).includes("read") &&
        (w.capabilities as string[]).includes("acquire"))) &&
    (v.can_adopt || v.candidates.length === 0)
  );
}
export function validOnboardingPreview(v: unknown): v is OnboardingPreview {
  if (
    !keys(v, [
      "profile",
      "preview_id",
      "operation",
      "expires_at",
      "workspace_id",
      "snapshot",
      "review",
    ]) ||
    v.profile !== "private-onboarding-preview-v1" ||
    !id(v.preview_id) ||
    !hash(v.snapshot) ||
    typeof v.expires_at !== "string" ||
    !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/.test(
      v.expires_at,
    ) ||
    !Number.isFinite(Date.parse(v.expires_at)) ||
    new Date(v.expires_at).toISOString() !==
      v.expires_at.slice(0, 19) +
        "." +
        (v.expires_at.includes(".")
          ? v.expires_at.slice(20, -1).padEnd(3, "0").slice(0, 3)
          : "000") +
        "Z"
  )
    return false;
  if (v.operation === "setup")
    return (
      v.workspace_id === null &&
      keys(v.review, ["capabilities"]) &&
      caps(v.review.capabilities) &&
      v.review.capabilities.length === 3
    );
  return (
    v.operation === "adopt" &&
    id(v.workspace_id) &&
    candidate(v.review, ["expected_generation", "impact"]) &&
    rev((v.review as Record<string, unknown>).expected_generation) &&
    (v.review as Record<string, unknown>).impact ===
      "pause_legacy_acquisition" &&
    v.review.snapshot === v.snapshot
  );
}
function canonical(v: unknown): unknown {
  return Array.isArray(v)
    ? v.map(canonical)
    : obj(v)
      ? Object.fromEntries(
          Object.keys(v)
            .sort()
            .map((k) => [k, canonical(v[k])]),
        )
      : v;
}
export function sameOnboardingPreview(
  a: OnboardingPreview,
  b: OnboardingPreview,
): boolean {
  return JSON.stringify(canonical(a)) === JSON.stringify(canonical(b));
}
// Retain only a cryptographic immutable-review pin when current disclosure is lost.
export async function onboardingPreviewDigest(
  preview: OnboardingPreview,
): Promise<string> {
  const bytes = new TextEncoder().encode(JSON.stringify(canonical(preview)));
  return Array.from(
    new Uint8Array(await crypto.subtle.digest("SHA-256", bytes)),
  )
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
export function validOnboardingCommand(v: unknown): v is OnboardingCommand {
  if (
    !keys(v, [
      "profile",
      "command_id",
      "recovery_id",
      "prepare_key",
      "state",
      "preview",
      "result",
      "acknowledged",
    ]) ||
    v.profile !== "private-onboarding-command-v1" ||
    !id(v.command_id) ||
    !id(v.recovery_id) ||
    typeof v.prepare_key !== "string" ||
    !/^[A-Za-z0-9._:-]{1,128}$/.test(v.prepare_key) ||
    !["prepared", "committed", "cancelled"].includes(String(v.state)) ||
    typeof v.acknowledged !== "boolean" ||
    !validOnboardingPreview(v.preview)
  )
    return false;
  if (v.state === "prepared") return v.result === null && !v.acknowledged;
  if (v.state === "cancelled")
    return keys(v.result, ["cancelled"]) && v.result.cancelled === true;
  const r = v.result;
  if (v.preview.operation === "setup")
    return (
      keys(r, ["workspace_id", "workspace_revision", "capabilities"]) &&
      id(r.workspace_id) &&
      rev(r.workspace_revision) &&
      caps(r.capabilities) &&
      r.capabilities.length === 3
    );
  if (
    !keys(r, ["collection_item_id", "item_generation", "sources"]) ||
    !id(r.collection_item_id) ||
    r.collection_item_id !== v.preview.review.candidate_id ||
    !rev(r.item_generation) ||
    !Array.isArray(r.sources) ||
    r.sources.length !== v.preview.review.sources.length
  )
    return false;
  const selected = new Set<string>();
  return (
    r.sources.every(
      (s) =>
        keys(s, ["source_account_id", "source_revision", "revision"]) &&
        id(s.source_account_id) &&
        !selected.has(s.source_account_id) &&
        !!selected.add(s.source_account_id) &&
        s.source_revision === "0" &&
        s.revision === "1",
    ) && selected.size === v.preview.review.sources.length
  );
}
export function validOnboardingPending(v: unknown): v is OnboardingPending {
  if (
    !keys(v, ["profile", "state", "command", "recovery_id"]) ||
    v.profile !== "private-onboarding-pending-v1"
  )
    return false;
  if (v.state === "none" || v.state === "blocked")
    return v.command === null && v.recovery_id === null;
  if (v.state === "inaccessible")
    return v.command === null && id(v.recovery_id);
  return (
    v.state === "available" &&
    id(v.recovery_id) &&
    validOnboardingCommand(v.command) &&
    v.command.recovery_id === v.recovery_id &&
    !v.command.acknowledged
  );
}
const base = "/v1/collection-onboarding";
// Fresh authenticated transport owns tokens; this adapter stores no identity or browser data.
export function createOnboardingClient(api: API) {
  const request = async (
    suffix: string,
    body: unknown | undefined,
    validate: (v: unknown) => boolean,
    mutation: boolean,
    query = false,
  ): Promise<unknown> => {
    try {
      const raw =
        body === undefined
          ? await api<unknown>(base + suffix)
          : query
            ? await (api.query
                ? api.query<unknown>(base + suffix, JSON.stringify(body))
                : api<unknown>(base + suffix, JSON.stringify(body)))
            : await api<unknown>(base + suffix, JSON.stringify(body));
      if (
        new TextEncoder().encode(JSON.stringify(raw)).length > 1048576 ||
        !validate(raw)
      )
        throw new OnboardingFailure(mutation ? "uncertain" : "unavailable");
      return raw;
    } catch (e) {
      if (e instanceof OnboardingFailure) throw e;
      if (e instanceof APIError) {
        if (!mutation && e.status === 404)
          throw new OnboardingFailure("unsupported");
        if ([401, 403, 404].includes(e.status))
          throw new OnboardingFailure("denied");
        if (!e.uncertain && e.status === 409)
          throw new OnboardingFailure("conflict");
        if (!e.uncertain && [400, 422].includes(e.status))
          throw new OnboardingFailure("rejected");
      }
      throw new OnboardingFailure(mutation ? "uncertain" : "unavailable");
    }
  };
  return {
    catalog: () =>
      request(
        "",
        undefined,
        validOnboardingCatalog,
        false,
      ) as Promise<OnboardingCatalog>,
    pending: () =>
      request(
        "/commands/pending",
        undefined,
        validOnboardingPending,
        false,
      ) as Promise<OnboardingPending>,
    preview: (action: OnboardingAction) =>
      request(
        "/previews",
        action,
        validOnboardingPreview,
        false,
        true,
      ) as Promise<OnboardingPreview>,
    prepare: (preview_id: string, prepare_key: string) =>
      request(
        "/commands",
        { preview_id, prepare_key },
        validOnboardingCommand,
        true,
      ) as Promise<OnboardingCommand>,
    transition: (
      command: Pick<OnboardingCommand, "command_id">,
      action: "execute" | "cancel" | "acknowledge",
    ) =>
      request(
        `/commands/${encodeURIComponent(command.command_id)}/${action}`,
        {},
        validOnboardingCommand,
        true,
      ) as Promise<OnboardingCommand>,
    clear: (recovery_id: string) =>
      request(
        "/recovery/clear",
        { recovery_id },
        (v) =>
          keys(v, ["profile", "recovery_id", "cleared"]) &&
          v.profile === "private-onboarding-clear-v1" &&
          v.recovery_id === recovery_id &&
          v.cleared === true,
        true,
      ),
  };
}
