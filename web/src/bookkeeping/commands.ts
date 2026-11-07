import { useCallback, useEffect, useRef, useState } from "react";
import { APIError, type API } from "./api";
import type { DurableCommand, RegisterCommand, PendingCommand } from "./types";
// Registration retry material lives only in memory; the server discovers durable work after remount.
type Registration = { key: string; body: string; path?: string };
// One user command gates new work until a terminal result is explicitly acknowledged.
export function useCommands(api: API, onPrepareConflict?: () => void) {
  const [command, setCommand] = useState<DurableCommand | null>(null);
  const [registration, setRegistration] = useState<Registration | null>(null);
  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [accessDenied, setAccessDenied] = useState(false);
  const sessionEpoch = useRef(0);
  const alive = useRef(true);
  const running = useRef(false);
  const generation = useRef(0);
  const conflictCallback = useRef(onPrepareConflict);
  conflictCallback.current = onPrepareConflict;
  const uncertainRegistration = useRef<Registration | null>(null);
  const rejectedUncertainRegistration = useRef(false);
  // Discovery clears stale recovery errors while preserving a rejected action's explanation.
  const discover = useCallback(
    async (actionError = "") => {
      const version = ++generation.current;
      setReady(false);
      try {
        const result = await api<PendingCommand>("/v1/commands/pending");
        if (!alive.current || version !== generation.current) return null;
        if (
          uncertainRegistration.current &&
          rejectedUncertainRegistration.current
        ) {
          // Pending has no registration key: only exact-key replay can resolve this candidate.
          setError(
            "Registration is unconfirmed. Retry the same registration before new work.",
          );
          setAccessDenied(false);
          setReady(true);
          return null;
        }
        setError(actionError);
        setAccessDenied(false);
        setCommand(result.command);
        if (result.command) {
          uncertainRegistration.current = null;
          rejectedUncertainRegistration.current = false;
          setRegistration(null);
        }
        setReady(true);
        return result.command;
      } catch (e) {
        if (alive.current && version === generation.current) {
          setCommand(null);
          setAccessDenied(
            e instanceof APIError && [401, 403, 404].includes(e.status),
          );
          setError(
            e instanceof Error
              ? e.message
              : "Recovery is unavailable. New work remains locked.",
          );
        }
        return null;
      }
    },
    [api],
  );
  useEffect(() => {
    alive.current = true;
    sessionEpoch.current++;
    running.current = false;
    setCommand(null);
    setRegistration(null);
    uncertainRegistration.current = null;
    rejectedUncertainRegistration.current = false;
    setBusy(false);
    setError("");
    setAccessDenied(false);
    void discover();
    return () => {
      alive.current = false;
      sessionEpoch.current++;
      generation.current++;
    };
  }, [discover]);
  // A network error never invents a new key; discovery can observe the persisted registration.
  const register = async (candidate: Registration) => {
    if (running.current) return;
    const epoch = sessionEpoch.current;
    const current = () => alive.current && epoch === sessionEpoch.current;
    const wasUncertain = uncertainRegistration.current?.key === candidate.key;
    running.current = true;
    setBusy(true);
    setError("");
    setRegistration(candidate);
    try {
      const value = await api<DurableCommand>(
        candidate.path ?? "/v1/commands",
        candidate.body,
        candidate.key,
      );
      if (current()) {
        uncertainRegistration.current = null;
        rejectedUncertainRegistration.current = false;
        setRegistration(null);
        if (value.acknowledged) await discover();
        else {
          setCommand(value);
          setReady(true);
        }
        return current() ? value : undefined;
      }
    } catch (e) {
      if (!current()) return;
      const message =
        e instanceof Error ? e.message : "Command registration is unconfirmed.";
      setError(message);
      // Definitive rejection permits discarding only this failed candidate; a conflicting pending command is still discovered.
      if (e instanceof APIError && !e.uncertain && !wasUncertain) {
        setRegistration(null);
        if (e.status === 409) conflictCallback.current?.();
      } else {
        uncertainRegistration.current = candidate;
        if (wasUncertain && e instanceof APIError && !e.uncertain)
          rejectedUncertainRegistration.current = true;
      }
      await discover(e instanceof APIError && e.uncertain ? "" : message);
    } finally {
      if (current()) {
        running.current = false;
        setBusy(false);
      }
    }
  };
  const prepare = (input: RegisterCommand) => {
    if (!ready || command || registration || running.current) return;
    void register({ key: crypto.randomUUID(), body: JSON.stringify(input) });
  };
  // Keep the exact selection endpoint/body/key in memory; remount uses the shared SQL recovery gate.
  const prepareRegistration = (
    entity: string,
    book: string,
    attempt: string,
    accountIDs: string[],
  ) => {
    if (!ready || command || registration || running.current)
      return Promise.resolve(undefined);
    return register({
      key: crypto.randomUUID(),
      path: `/v1/entities/${encodeURIComponent(entity)}/books/${encodeURIComponent(book)}/connection-attempts/${encodeURIComponent(attempt)}/registration`,
      body: JSON.stringify({ account_ids: accountIDs }),
    });
  };
  // Explicit recovery clears only server-confirmed inaccessible work and never receives its payload.
  const clearInaccessible = async () => {
    if (!accessDenied || running.current) return;
    const epoch = sessionEpoch.current;
    const current = () => alive.current && epoch === sessionEpoch.current;
    running.current = true;
    setBusy(true);
    setError("");
    try {
      await api<{ cleared: boolean }>("/v1/commands/pending/clear", "{}");
      if (current()) await discover();
    } catch (e) {
      if (current()) {
        setError(
          e instanceof Error
            ? e.message
            : "Inaccessible saved work could not be cleared.",
        );
      }
    } finally {
      if (current()) {
        running.current = false;
        setBusy(false);
      }
    }
  };
  // Every action targets the durable command ID; retries cannot become a different financial request.
  const transition = async (action: "execute" | "cancel" | "acknowledge") => {
    if (!command || running.current) return;
    const epoch = sessionEpoch.current;
    const current = () => alive.current && epoch === sessionEpoch.current;
    running.current = true;
    setBusy(true);
    setError("");
    try {
      const value = await api<DurableCommand>(
        `/v1/commands/${command.command_id}/${action}`,
        "{}",
      );
      if (!current()) return;
      if (value.acknowledged) {
        setCommand(null);
        setRegistration(null);
        await discover();
      } else {
        setCommand(value);
        setReady(true);
      }
    } catch (e) {
      if (!current()) return;
      const message =
        e instanceof Error
          ? e.message
          : "Command outcome is unconfirmed. Check its saved state.";
      setError(message);
      await discover(e instanceof APIError && e.uncertain ? "" : message);
    } finally {
      if (current()) {
        running.current = false;
        setBusy(false);
      }
    }
  };
  return {
    command,
    accessDenied,
    busy,
    ready,
    error,
    registration,
    locked: !ready || busy || !!command || !!registration,
    prepare,
    prepareRegistration,
    clearInaccessible,
    transition,
    retryRegistration: () => registration && void register(registration),
    // Reciprocal conflict discovery cannot replace active work or an uncertain registration.
    refreshIfIdle: async () => {
      if (!running.current && !command && !registration) await discover();
    },
    reload: async () => {
      if (!running.current) {
        setError("");
        await discover();
      }
    },
  };
}
