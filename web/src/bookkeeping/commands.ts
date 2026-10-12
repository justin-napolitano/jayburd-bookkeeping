import { useCallback, useEffect, useRef, useState } from "react";
import { APIError, type API } from "./api";
import type { DurableCommand, RegisterCommand, PendingCommand } from "./types";
// Registration retry material lives only in memory; the server discovers durable work after remount.
type Registration = { key: string; body: string };
// One user command gates new work until a terminal result is explicitly acknowledged.
export function useCommands(api: API) {
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
  // Failed discovery clears sensitive snapshots and keeps new work locked.
  const discover = useCallback(async () => {
    const version = ++generation.current;
    setReady(false);
    try {
      const result = await api<PendingCommand>("/v1/commands/pending");
      if (!alive.current || version !== generation.current) return null;
      setAccessDenied(false);
      setCommand(result.command);
      if (result.command) setRegistration(null);
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
  }, [api]);
  useEffect(() => {
    alive.current = true;
    sessionEpoch.current++;
    running.current = false;
    setCommand(null);
    setRegistration(null);
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
    running.current = true;
    setBusy(true);
    setError("");
    setRegistration(candidate);
    try {
      const value = await api<DurableCommand>(
        "/v1/commands",
        candidate.body,
        candidate.key,
      );
      if (current()) {
        setRegistration(null);
        if (value.acknowledged) await discover();
        else {
          setCommand(value);
          setReady(true);
        }
      }
    } catch (e) {
      if (!current()) return;
      setError(
        e instanceof Error ? e.message : "Command registration is unconfirmed.",
      );
      // Definitive rejection permits discarding only this failed candidate; a conflicting pending command is still discovered.
      if (e instanceof APIError && !e.uncertain) setRegistration(null);
      await discover();
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
      setError(
        e instanceof Error
          ? e.message
          : "Command outcome is unconfirmed. Check its saved state.",
      );
      await discover();
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
    transition,
    retryRegistration: () => registration && void register(registration),
    reload: async () => {
      if (!running.current) {
        setError("");
        await discover();
      }
    },
  };
}
