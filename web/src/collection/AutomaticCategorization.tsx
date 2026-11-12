import { useCallback, useEffect, useRef, useState } from "react";
import { CollectionFailure } from "./types";
import {
  validAutomationStatus,
  type AutomationPort,
  type AutomationRequest,
  type AutomationStatus,
} from "./automation";

type Props = {
  workspace: string;
  port: AutomationPort;
  canClassify: boolean;
  locked: boolean;
  active?: boolean;
  onLockChange: (locked: boolean) => void;
  onDenied: () => void;
  onChanged: () => void;
  onRun?: (runID: string) => void;
  onAttention: () => void;
};
// Only a confirmed configuration response resolves a live unknown request; reads never replay a mutation.
export function AutomaticCategorization({
  workspace,
  port,
  canClassify,
  locked,
  active = true,
  onLockChange,
  onDenied,
  onChanged,
  onRun,
  onAttention,
}: Props) {
  const [status, setStatus] = useState<AutomationStatus | null>(null),
    [running, setRunning] = useState(false),
    [intent, setIntent] = useState<AutomationRequest | null>(null),
    [uncertain, setUncertain] = useState(false),
    [error, setError] = useState("");
  const epoch = useRef(0),
    owner = useRef({ port, workspace }),
    flight = useRef(false),
    pollCount = useRef(0),
    observedRun = useRef<string | null>(null),
    callbacks = useRef({ onLockChange, onDenied, onChanged, onRun });
  const readable = useRef(false);
  readable.current = active && !locked && !running && !uncertain;
  callbacks.current = { onLockChange, onDenied, onChanged, onRun };
  useEffect(() => {
    callbacks.current.onLockChange(running || uncertain);
  }, [running, uncertain]);
  const load = useCallback(
    async (quiet = false) => {
      if (flight.current || !readable.current) return;
      const n = ++epoch.current;
      if (!quiet) setStatus(null);
      setError("");
      try {
        const value = await port.read(workspace);
        if (n !== epoch.current || !readable.current) return;
        if (!validAutomationStatus(value, workspace))
          throw new CollectionFailure("unavailable");
        // A reused immutable port result must still advance the pending poll.
        setStatus({ ...value });
        if (
          value.last_run?.state === "committed" &&
          observedRun.current !== value.last_run.run_id
        ) {
          observedRun.current = value.last_run.run_id;
          callbacks.current.onRun?.(value.last_run.run_id);
        }
      } catch (reason) {
        if (n !== epoch.current || !readable.current) return;
        setStatus(null);
        if (reason instanceof CollectionFailure && reason.kind === "denied")
          callbacks.current.onDenied();
        else
          setError(
            "Automatic category status is unavailable. Refresh status before changing settings.",
          );
      }
    },
    [port, workspace],
  );
  useEffect(() => {
    owner.current = { port, workspace };
    pollCount.current = 0;
    observedRun.current = null;
    flight.current = false;
    setRunning(false);
    setIntent(null);
    setUncertain(false);
    setStatus(null);
    return () => {
      epoch.current++;
      flight.current = false;
    };
  }, [port, workspace]);
  useEffect(() => {
    if (active && !locked && !uncertain) void load();
    // Pause reads without discarding the outcome of a settings request already in flight.
    else if (!flight.current) epoch.current++;
  }, [load, active, locked, uncertain]);
  // At most six status reads while pending; work continues server-side if this view closes.
  useEffect(() => {
    if (
      !active ||
      locked ||
      running ||
      uncertain ||
      status?.processing_state !== "pending" ||
      pollCount.current >= 6
    )
      return;
    const timer = setTimeout(() => {
      pollCount.current++;
      void load(true);
    }, 5000);
    return () => clearTimeout(timer);
  }, [active, locked, running, uncertain, status, load]);
  const configure = async (retry = false) => {
    if (
      flight.current ||
      locked ||
      !canClassify ||
      (!retry && (!status || uncertain))
    )
      return;
    const request = retry
      ? intent
      : {
          key: crypto.randomUUID(),
          expected_revision: status!.revision,
          enabled: !status!.enabled,
        };
    if (!request) return;
    const wasUncertain = uncertain,
      n = ++epoch.current;
    flight.current = true;
    setRunning(true);
    setIntent(request);
    setError("");
    try {
      const value = await port.configure(workspace, request);
      if (n !== epoch.current) return;
      if (
        !validAutomationStatus(value, workspace) ||
        value.enabled !== request.enabled
      )
        throw new CollectionFailure("uncertain");
      setIntent(null);
      setUncertain(false);
      pollCount.current = 0;
      setStatus(value);
      // A replay receipt can predate another tab's configuration. A fresh read supplies current display state.
      try {
        const current = await port.read(workspace);
        if (n !== epoch.current) return;
        if (!validAutomationStatus(current, workspace))
          throw new CollectionFailure("unavailable");
        setStatus(current);
      } catch (reason) {
        if (n === epoch.current) {
          if (reason instanceof CollectionFailure && reason.kind === "denied")
            callbacks.current.onDenied();
          setStatus(null);
          setError(
            "Your settings change was confirmed. Current status is unavailable; refresh category status.",
          );
        }
      }
      if (n === epoch.current) callbacks.current.onChanged();
    } catch (reason) {
      if (n !== epoch.current) return;
      setStatus(null);
      const unknown =
        wasUncertain ||
        !(reason instanceof CollectionFailure) ||
        reason.kind === "uncertain";
      setUncertain(unknown);
      if (unknown)
        setError(
          "The settings result is unknown. Retry the same settings change before starting other work.",
        );
      else {
        setIntent(null);
        setError(
          "The settings change was not accepted. Refresh category status before trying again.",
        );
      }
      if (reason instanceof CollectionFailure && reason.kind === "denied")
        callbacks.current.onDenied();
    } finally {
      if (n === epoch.current) {
        flight.current = false;
        setRunning(false);
      }
    }
  };
  if (owner.current.port !== port || owner.current.workspace !== workspace)
    return <p role="status">Checking current automatic category access…</p>;
  return (
    <section
      aria-label="Automatic categories"
      className="automatic-categories"
      aria-busy={running}
    >
      <div className="automatic-category-heading">
        <strong>Automatic categories</strong>
        <span>
          {status
            ? status.enabled
              ? "On"
              : "Off"
            : error
              ? "Category status unavailable."
              : "Checking category status…"}
        </span>
      </div>
      {status && (
        <>
          {status.processing_state === "pending" && (
            <p>Categorizing imported transactions…</p>
          )}
          {status.processing_state === "blocked" && (
            <p>
              {status.blocked_reason === "saved_work"
                ? "Finish your saved work before automatic categorization continues."
                : status.blocked_reason === "category_catalog_unavailable"
                  ? "The automatic category catalog needs attention."
                  : "Automatic categorization is waiting for current access."}
            </p>
          )}
          <button
            type="button"
            className="secondary"
            aria-label="Needs attention"
            disabled={locked || running || uncertain}
            onClick={onAttention}
          >
            Needs attention · {status.attention_count}
          </button>
          {!status.enabled && canClassify && (
            <button
              type="button"
              disabled={locked || running || uncertain}
              onClick={() => void configure()}
            >
              Enable automatic categories
            </button>
          )}
        </>
      )}
      {error && <p role="alert">{error}</p>}
      {uncertain && (
        <button
          type="button"
          disabled={locked || running}
          onClick={() => void configure(true)}
        >
          Retry same settings change
        </button>
      )}
      <details>
        <summary aria-label="Category settings">Settings</summary>
        <p>When enabled, routine transactions are categorized automatically.</p>
        {status && (
          <p>
            {status.pending_count} awaiting categorization ·{" "}
            {status.attention_count} need attention
          </p>
        )}
        <p>
          Categories organize transactions. Your choices are protected;
          accounting entries still require confirmation.
        </p>
        {status?.enabled && canClassify && (
          <button
            type="button"
            className="secondary"
            disabled={locked || running || uncertain}
            onClick={() => void configure()}
          >
            Turn off automatic categories
          </button>
        )}
        {!uncertain && (
          <button
            type="button"
            className="secondary"
            disabled={locked || running}
            onClick={() => void load()}
          >
            Refresh category status
          </button>
        )}
      </details>
    </section>
  );
}
