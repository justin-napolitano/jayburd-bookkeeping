import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { API } from "../bookkeeping/api";
import {
  createRecoveryClient,
  RecoveryFailure,
  type RecoveryStatus,
} from "./recovery";
import styles from "./collection.module.css";

export type RecoveryViewState = {
  state: "checking" | "none" | "available" | "inaccessible" | "unavailable";
  uncertain: boolean;
  canReview: boolean;
};

// Mount outside workspace discovery: historical ownership permits recovery, not disclosure or new grants.
export function CollectionRecovery({
  api,
  refreshVersion = 0,
  onLockChange,
  onAvailable,
  onCleared,
  onStateChange,
}: {
  api: API;
  refreshVersion?: number;
  onLockChange: (locked: boolean) => void;
  onAvailable?: () => void;
  onCleared?: () => void;
  onStateChange?: (state: RecoveryViewState) => void;
}) {
  const client = useMemo(() => createRecoveryClient(api), [api]);
  const [status, setStatus] = useState<RecoveryStatus | null>(null);
  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [uncertain, setUncertain] = useState(false);
  const epoch = useRef(0);
  const candidate = useRef<string | null>(null);
  const knownWork = useRef(false);
  const lastAvailable = useRef<string | null>(null);
  const callbacks = useRef({
    onLockChange,
    onAvailable,
    onCleared,
    onStateChange,
  });
  callbacks.current = { onLockChange, onAvailable, onCleared, onStateChange };
  const clearing = useRef(false);
  const discover = useCallback(
    async (explicitOpen = false) => {
      const n = ++epoch.current;
      setReady(false);
      setBusy(true);
      setError("");
      try {
        const result = await client.discover();
        if (n !== epoch.current) return;
        if (result === null && (knownWork.current || candidate.current))
          throw new RecoveryFailure("unavailable");
        if (result && result.state !== "none") knownWork.current = true;
        if (result?.state !== "available") lastAvailable.current = null;
        setStatus(result);
        setReady(true);
        if (candidate.current) {
          setUncertain(true);
          setError(
            "Clearing the saved private review is unconfirmed. Retry the same recovery request.",
          );
        } else setUncertain(false);
        if (
          result?.state === "available" &&
          (!candidate.current || candidate.current === result.recovery_id) &&
          (explicitOpen || result.recovery_id !== lastAvailable.current)
        ) {
          lastAvailable.current = result.recovery_id;
          callbacks.current.onAvailable?.();
        }
      } catch {
        if (n === epoch.current) {
          setStatus(null);
          setError("Private recovery is unavailable. New work remains locked.");
        }
      } finally {
        if (n === epoch.current) setBusy(false);
      }
    },
    [client],
  );
  useEffect(() => {
    epoch.current++;
    candidate.current = null;
    knownWork.current = false;
    lastAvailable.current = null;
    clearing.current = false;
    setStatus(null);
    setReady(false);
    setBusy(false);
    setError("");
    setUncertain(false);
    return () => {
      epoch.current++;
    };
  }, [client]);
  useEffect(() => {
    if (!clearing.current) void discover();
  }, [discover, refreshVersion]);
  const locked =
    !ready ||
    busy ||
    uncertain ||
    !!candidate.current ||
    (status !== null && status.state !== "none");
  useEffect(() => {
    callbacks.current.onLockChange(locked);
  }, [locked]);
  const viewState: RecoveryViewState["state"] =
    busy || (!ready && !error)
      ? "checking"
      : !ready
        ? "unavailable"
        : (status?.state ?? "none");
  const hasUnknownOutcome = uncertain || candidate.current !== null;
  const canReview =
    status?.state === "available" &&
    (!candidate.current || candidate.current === status.recovery_id);
  useEffect(() => {
    callbacks.current.onStateChange?.({
      state: viewState,
      uncertain: hasUnknownOutcome,
      canReview,
    });
  }, [viewState, hasUnknownOutcome, canReview]);

  const clear = async () => {
    if (busy || clearing.current) return;
    const recovery_id =
      candidate.current ??
      (status?.state === "inaccessible" ? status.recovery_id : null);
    if (!recovery_id) return;
    const wasUncertain = candidate.current !== null;
    candidate.current = recovery_id;
    knownWork.current = true;
    clearing.current = true;
    const n = ++epoch.current;
    setBusy(true);
    setError("");
    try {
      await client.clear(recovery_id);
      if (n !== epoch.current) return;
      candidate.current = null;
      setUncertain(false);
      setStatus(null);
      setReady(false);
      // Only the exact confirmed receipt invokes this callback; discovery:none never substitutes for it.
      callbacks.current.onCleared?.();
      clearing.current = false;
      await discover();
    } catch (reason) {
      if (n !== epoch.current) return;
      clearing.current = false;
      if (
        !wasUncertain &&
        reason instanceof RecoveryFailure &&
        reason.kind === "rejected"
      ) {
        candidate.current = null;
        setUncertain(false);
        await discover();
      } else {
        setUncertain(true);
        setError(
          "Clearing the saved private review is unconfirmed. Retry the same recovery request.",
        );
      }
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  };

  if (
    ready &&
    !busy &&
    !uncertain &&
    !candidate.current &&
    (status === null || status.state === "none")
  )
    return null;
  return (
    <section
      className={styles.workspace}
      aria-label="Private work recovery"
      aria-busy={busy}
    >
      <h2>Saved private work</h2>
      {busy && <p role="status">Checking saved private work…</p>}
      {error && <p role="alert">{error}</p>}
      {!uncertain && status?.state === "inaccessible" && (
        <p>
          A saved private review is no longer accessible. Clearing it never
          applies changes or reverses completed changes; it preserves the
          history.
        </p>
      )}
      {!uncertain && status?.state === "available" && (
        <p>Review your saved private work before starting another request.</p>
      )}
      {((!uncertain && status?.state === "inaccessible") ||
        candidate.current) && (
        <button disabled={busy} onClick={() => void clear()}>
          {candidate.current
            ? "Retry clearing the same private review"
            : "Clear inaccessible private review"}
        </button>
      )}
      <button
        className="secondary"
        disabled={busy}
        onClick={() => void discover(true)}
      >
        {status?.state === "available"
          ? "Open saved private review"
          : "Check private recovery"}
      </button>
    </section>
  );
}
