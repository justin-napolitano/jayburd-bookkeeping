import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { API } from "../bookkeeping/api";
import { CollectionReview } from "./CollectionReview";
import {
  createCollectionPort,
  discoverCollection,
  type CollectionBootstrap,
} from "./client";

// Always-mounted discovery gates other tools, including when private transactions is not the active view.
export function CollectionWorkspace({
  api,
  active,
  bookLocked,
  onLockChange,
  refreshVersion = 0,
  onPrepareConflict,
  onAcknowledged,
}: {
  api: API;
  active: boolean;
  bookLocked: boolean;
  onLockChange: (locked: boolean) => void;
  refreshVersion?: number;
  onPrepareConflict?: () => Promise<void> | void;
  onAcknowledged?: () => void;
}) {
  const [bootstrap, setBootstrap] = useState<CollectionBootstrap | null>(null);
  const [state, setState] = useState<
    "loading" | "ready" | "absent" | "denied" | "failed"
  >("loading");
  const [sessionLocked, setSessionLocked] = useState(true);
  const epoch = useRef(0);
  const refreshedVersion = useRef(refreshVersion);
  const port = useMemo(() => createCollectionPort(api), [api]);
  const load = useCallback(async () => {
    const n = ++epoch.current;
    setState("loading");
    setBootstrap(null);
    setSessionLocked(true);
    try {
      const result = await discoverCollection(api);
      if (n !== epoch.current) return;
      setBootstrap(result);
      setState(result ? "ready" : "absent");
    } catch {
      if (n === epoch.current) setState("failed");
    }
  }, [api]);
  useEffect(() => {
    void load();
    return () => {
      epoch.current++;
    };
  }, [load]);
  const refreshPending = refreshedVersion.current !== refreshVersion;
  useEffect(() => {
    if (!refreshPending) return;
    refreshedVersion.current = refreshVersion;
    // Existing sessions perform their own idle-only pending refresh without unmounting.
    if (state === "ready") return;
    void load();
  }, [refreshPending, refreshVersion, state, sessionLocked, load]);
  const locked =
    state === "ready"
      ? sessionLocked || refreshPending
      : state === "loading" ||
        state === "failed" ||
        state === "denied" ||
        refreshPending;
  useEffect(() => {
    onLockChange(locked);
  }, [locked, onLockChange]);
  if (state === "failed")
    return (
      <section aria-label="Private transaction recovery">
        <p role="alert">
          Private transaction recovery is unavailable. New work remains locked.
        </p>
        <button onClick={() => void load()}>
          Retry private transaction recovery
        </button>
      </section>
    );
  if (state === "loading")
    return (
      <p role="status">Checking private transaction access and recovery…</p>
    );
  if (state === "absent" || state === "denied")
    return active || state === "denied" ? (
      <section aria-label="All transactions">
        <h2>All transactions</h2>
        <p>
          {state === "denied"
            ? "Access to private transactions is no longer available."
            : "Private transactions are unavailable for this session."}
        </p>
        <button onClick={() => void load()}>
          Retry private transaction access
        </button>
      </section>
    ) : null;
  if (!bootstrap) return null;
  return (
    <div hidden={!active && !sessionLocked}>
      <CollectionReview
        workspace={bootstrap.workspace}
        sources={bootstrap.sources}
        catalog={bootstrap.catalog}
        port={port}
        canClassify={bootstrap.canClassify}
        locked={bookLocked}
        active={active}
        onLockChange={setSessionLocked}
        onAcknowledged={() => {
          onAcknowledged?.();
          void load();
        }}
        onRefreshCatalog={() => void load()}
        onPrepareConflict={onPrepareConflict}
        refreshVersion={refreshVersion}
        onDenied={() => {
          epoch.current++;
          setBootstrap(null);
          setState("denied");
        }}
      />
    </div>
  );
}
