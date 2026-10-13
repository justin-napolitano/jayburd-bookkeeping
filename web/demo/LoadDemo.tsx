import { useEffect, useRef, useState } from "react";
import type { API } from "../src/bookkeeping/api";

// Local synthetic acquisition status; the server owns durable progress and counts.
export type DemoAcquisition = {
  profile: "synthetic-acquisition-v1";
  total_count: string;
  imported_count: string;
  complete: boolean;
  blocked: boolean;
  pending_command_id: string | null;
  batches: {
    name: string;
    entity_id: string;
    book_id: string;
    source_account_id: string;
    batch_id: string;
    state: string;
    imported_count: string;
  }[];
};
// Composition supplies the existing authenticated, memory-only API client.
export type LoadDemoProps = {
  api: API;
  locked: boolean;
  onComplete: () => void;
  onActiveChange: (active: boolean) => void;
};
const path = "/v1/demo/acquisition";
// Mount discovers progress only. Every series of mutations requires an explicit click.
export function LoadDemo({
  api,
  locked,
  onComplete,
  onActiveChange,
}: LoadDemoProps) {
  const [status, setStatus] = useState<DemoAcquisition | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [paused, setPaused] = useState(false);
  const activeChange = useRef(onActiveChange);
  activeChange.current = onActiveChange;
  const alive = useRef(false);
  const running = useRef(false);
  const stop = useRef(false);
  const epoch = useRef(0);
  const lockedRef = useRef(locked);
  lockedRef.current = locked;
  useEffect(() => {
    alive.current = true;
    running.current = false;
    const version = ++epoch.current;
    setStatus(null);
    void api<DemoAcquisition>(path)
      .then((value) => {
        if (alive.current && version === epoch.current) setStatus(value);
      })
      .catch(() => {
        if (alive.current && version === epoch.current)
          setError(
            "Demo progress is unavailable. Retry to reconcile with the server.",
          );
      });
    return () => {
      alive.current = false;
      stop.current = true;
      epoch.current++;
      activeChange.current(false);
    };
  }, [api]);
  // Unknown replies stop automatic writes; a later click first reconciles durable status.
  const load = async () => {
    if (running.current || lockedRef.current) return;
    running.current = true;
    activeChange.current(true);
    stop.current = false;
    setPaused(false);
    setBusy(true);
    setError("");
    const version = ++epoch.current;
    const current = () => alive.current && version === epoch.current;
    try {
      let value = await api<DemoAcquisition>(path);
      if (!current()) return;
      setStatus(value);
      for (
        let step = 0;
        step < 64 && !value.complete && !value.blocked;
        step++
      ) {
        if (!current() || stop.current || lockedRef.current) break;
        value = await api<DemoAcquisition>(`${path}/step`, "{}");
        if (!current()) return;
        setStatus(value);
      }
      if (!current()) return;
      if (value.complete) onComplete();
      else if (!value.blocked) setPaused(true);
    } catch {
      if (current()) {
        setError(
          "The last step is unconfirmed. Resume to reconcile saved progress before continuing.",
        );
        try {
          const value = await api<DemoAcquisition>(path);
          if (current()) {
            setStatus(value);
            if (value.complete) {
              setError("");
              onComplete();
            }
          }
        } catch {
          if (current()) setStatus(null);
        }
      }
    } finally {
      if (current()) {
        running.current = false;
        setBusy(false);
        activeChange.current(false);
      }
    }
  };
  return (
    <section aria-labelledby="load-demo-title">
      <h2 id="load-demo-title">Demo books</h2>
      <p>
        Load 3,000 synthetic transactions across three fictional LLCs. Imported
        transactions await review; loading does not post journals or change
        posted balances.
      </p>
      <p role="status">
        {status
          ? `${status.imported_count} of ${status.total_count} transactions imported${status.complete ? " · Ready for review" : busy ? " · Loading" : paused ? " · Paused" : ""}`
          : "Progress unavailable"}
      </p>
      {status?.batches.length ? (
        <ul>
          {status.batches.map((batch) => (
            <li key={batch.book_id}>
              {batch.name}: {batch.imported_count} imported · {batch.state}
            </li>
          ))}
        </ul>
      ) : null}
      {status?.blocked && (
        <p role="alert">
          Another command needs attention. Resolve it in command recovery before
          resuming.
        </p>
      )}
      {error && <p role="alert">{error}</p>}
      {busy ? (
        <button
          type="button"
          onClick={() => {
            stop.current = true;
            setPaused(true);
          }}
        >
          Pause after this step
        </button>
      ) : (
        <button
          type="button"
          disabled={locked || status?.complete}
          onClick={() => void load()}
        >
          {status?.complete
            ? "Demo books loaded"
            : status?.imported_count !== "0" || error || paused
              ? "Resume demo loading"
              : "Load demo books"}
        </button>
      )}
    </section>
  );
}
