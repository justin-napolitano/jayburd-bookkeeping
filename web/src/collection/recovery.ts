import { APIError, type API } from "../bookkeeping/api";

export type RecoveryStatus = {
  profile: "private-collection-recovery-v1";
  state: "none" | "available" | "inaccessible";
  recovery_id: string | null;
};
export type RecoveryReceipt = {
  profile: "private-collection-recovery-clear-v1";
  recovery_id: string;
  cleared: true;
};
export class RecoveryFailure extends Error {
  constructor(public readonly kind: "unavailable" | "rejected" | "uncertain") {
    super("Private recovery unavailable");
  }
}
const record = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
const recoveryID = (value: unknown): value is string =>
  typeof value === "string" &&
  /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(value) &&
  value !== "00000000-0000-0000-0000-000000000000";

// Global owner recovery carries only an opaque claim identity, never a private workspace or stored payload.
export function createRecoveryClient(api: API) {
  return {
    async discover(): Promise<RecoveryStatus | null> {
      let value: unknown;
      try {
        value = await api<unknown>("/v1/collection-recovery");
      } catch (reason) {
        if (reason instanceof APIError && reason.status === 404) return null;
        throw new RecoveryFailure("unavailable");
      }
      if (
        !record(value) ||
        Object.keys(value).length !== 3 ||
        value.profile !== "private-collection-recovery-v1" ||
        !["none", "available", "inaccessible"].includes(String(value.state)) ||
        (value.state === "none"
          ? value.recovery_id !== null
          : !recoveryID(value.recovery_id))
      )
        throw new RecoveryFailure("unavailable");
      return value as RecoveryStatus;
    },
    async clear(recovery_id: string): Promise<RecoveryReceipt> {
      if (!recoveryID(recovery_id)) throw new RecoveryFailure("rejected");
      let value: unknown;
      try {
        value = await api<unknown>(
          "/v1/collection-recovery/clear",
          JSON.stringify({ recovery_id }),
        );
      } catch (reason) {
        throw new RecoveryFailure(
          reason instanceof APIError && !reason.uncertain
            ? "rejected"
            : "uncertain",
        );
      }
      if (
        !record(value) ||
        Object.keys(value).length !== 3 ||
        value.profile !== "private-collection-recovery-clear-v1" ||
        value.recovery_id !== recovery_id ||
        value.cleared !== true
      )
        throw new RecoveryFailure("uncertain");
      return value as RecoveryReceipt;
    },
  };
}
