import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { API } from "../bookkeeping/api";
import type { RuleSeed } from "./rules";
import { createCollectionJournalReader } from "./journal";
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
  newWorkLocked = false,
  onRecoveryNeeded,
  setupAvailable = false,
}: {
  api: API;
  active: boolean;
  bookLocked: boolean;
  onLockChange: (locked: boolean) => void;
  refreshVersion?: number;
  onPrepareConflict?: () => Promise<void> | void;
  onAcknowledged?: () => void;
  newWorkLocked?: boolean;
  onRecoveryNeeded?: () => void;
  setupAvailable?: boolean;
}) {
  const [seed, setSeed] = useState<{ api: API; value: RuleSeed } | null>(null);
  const [bootstrap, setBootstrap] = useState<CollectionBootstrap | null>(null);
  const [state, setState] = useState<
    "loading" | "ready" | "absent" | "denied" | "failed"
  >("loading");
  const [sessionLocked, setSessionLocked] = useState(true);
  // Opaque run IDs deduplicate catalog refresh across child remounts; no financial payload is retained.
  const automaticRuns = useRef<{ api: API; seen: Set<string> }>({
    api,
    seen: new Set(),
  });
  if (automaticRuns.current.api !== api)
    automaticRuns.current = { api, seen: new Set() };
  const epoch = useRef(0);
  const bootstrapAPI = useRef<API | null>(null);
  const refreshedVersion = useRef(refreshVersion);
  const readJournal = useMemo(() => createCollectionJournalReader(api), [api]);
  const port = useMemo(() => createCollectionPort(api), [api]);
  const load = useCallback(async () => {
    const n = ++epoch.current;
    bootstrapAPI.current = null;
    setState("loading");
    setBootstrap(null);
    setSessionLocked(true);
    try {
      const result = await discoverCollection(api);
      if (n !== epoch.current) return;
      setSeed((old) =>
        old &&
        result &&
        old.api === api &&
        old.value.workspace_id === result.workspace.workspace_id &&
        result.sources.some(
          (source) => source.source_id === old.value.source_id,
        )
          ? old
          : null,
      );
      bootstrapAPI.current = api;
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
  const scopeChanged = state === "ready" && bootstrapAPI.current !== api;
  const locked =
    state === "ready"
      ? scopeChanged || sessionLocked || refreshPending
      : state === "loading" ||
        state === "failed" ||
        state === "denied" ||
        refreshPending;
  useEffect(() => {
    onLockChange(locked);
  }, [locked, onLockChange]);
  if (scopeChanged)
    return (
      <p role="status">Checking private transaction access and recovery…</p>
    );
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
  if (state === "absent" && setupAvailable) return null;
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
        readJournal={readJournal}
        ruleSeed={
          seed?.api === api &&
          seed.value.workspace_id === bootstrap.workspace.workspace_id
            ? seed.value
            : null
        }
        onSeedConsumed={() => setSeed(null)}
        canClassify={bootstrap.canClassify}
        locked={bookLocked}
        active={active}
        reviewVisible={active || sessionLocked}
        onLockChange={setSessionLocked}
        newWorkLocked={newWorkLocked}
        onAcknowledged={(ruleSeed) => {
          setSeed(
            ruleSeed &&
              ruleSeed.workspace_id === bootstrap.workspace.workspace_id
              ? { api, value: ruleSeed }
              : null,
          );
          onAcknowledged?.();
          void load();
        }}
        onRefreshCatalog={() => void load()}
        onAutomaticRun={(runID) => {
          const key = bootstrap.workspace.workspace_id + "/" + runID;
          if (automaticRuns.current.seen.has(key)) return;
          automaticRuns.current.seen.add(key);
          void load();
        }}
        onPrepareConflict={onPrepareConflict}
        refreshVersion={refreshVersion}
        onDenied={() => {
          epoch.current++;
          setBootstrap(null);
          setSeed(null);
          setState("denied");
          onRecoveryNeeded?.();
        }}
      />
    </div>
  );
}
