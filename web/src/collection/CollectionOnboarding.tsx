import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { API } from "../bookkeeping/api";
import {
  createOnboardingClient,
  OnboardingFailure,
  onboardingPreviewDigest,
  type OnboardingAction,
  type OnboardingCatalog,
  type OnboardingCommand,
  type OnboardingPending,
  type OnboardingPreview,
} from "./onboarding";
import styles from "./collection.module.css";

export type OnboardingViewState = {
  supported: boolean;
  locked: boolean;
  pending: boolean;
};
type SavedIdentity = {
  command_id: string;
  recovery_id: string;
  prepare_key: string;
  state: OnboardingCommand["state"];
  digest: string;
};
async function identity(value: OnboardingCommand): Promise<SavedIdentity> {
  return {
    command_id: value.command_id,
    recovery_id: value.recovery_id,
    prepare_key: value.prepare_key,
    state: value.state,
    digest: await onboardingPreviewDigest(value.preview),
  };
}
// Mounted outside workspace discovery so setup and uncertain adoption survive tool navigation.
export function CollectionOnboarding({
  api,
  active,
  externalLocked,
  refreshVersion = 0,
  onLockChange,
  onStateChange,
  onChanged,
  onConflict,
  onConnections,
}: {
  api: API;
  active: boolean;
  externalLocked: boolean;
  refreshVersion?: number;
  onLockChange: (locked: boolean) => void;
  onStateChange?: (state: OnboardingViewState) => void;
  onChanged?: () => void;
  onConflict?: () => void | Promise<void>;
  onConnections?: () => void;
}) {
  const client = useMemo(() => createOnboardingClient(api), [api]);
  const owner = useRef(api);
  owner.current = api;
  const acceptedAPI = useRef<API | null>(null),
    epoch = useRef(0),
    running = useRef(false),
    known = useRef(false);
  const callbacks = useRef({
    onLockChange,
    onStateChange,
    onChanged,
    onConflict,
  });
  callbacks.current = { onLockChange, onStateChange, onChanged, onConflict };
  const [pending, setPending] = useState<OnboardingPending | null>(null),
    [catalog, setCatalog] = useState<OnboardingCatalog | null>(null),
    [preview, setPreview] = useState<OnboardingPreview | null>(null),
    [command, setCommand] = useState<OnboardingCommand | null>(null);
  const [ready, setReady] = useState(false),
    [supported, setSupported] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const [uncertain, setUncertain] = useState<
    "prepare" | "execute" | "cancel" | "acknowledge" | "clear" | null
  >(null);
  const [closureAvailable, setClosureAvailable] = useState(false);
  const intent = useRef<{
      preview_id: string;
      digest: string;
      key: string;
    } | null>(null),
    clearID = useRef<string | null>(null),
    saved = useRef<SavedIdentity | null>(null);
  const [focusVersion, setFocusVersion] = useState(0),
    heading = useRef<HTMLHeadingElement>(null);
  const focus = () => setFocusVersion((v) => v + 1);
  useEffect(() => {
    if (focusVersion && acceptedAPI.current === api) heading.current?.focus();
  }, [focusVersion, api]);
  const current = (n: number) => epoch.current === n && owner.current === api;
  const discover = useCallback(async () => {
    const n = ++epoch.current;
    setReady(false);
    setBusy(true);
    setError("");
    try {
      const result = await client.pending();
      if (!current(n)) return;
      known.current = true;
      acceptedAPI.current = api;
      setSupported(true);
      if (result.state === "available") {
        known.current = true;
        if (clearID.current && clearID.current !== result.recovery_id) {
          // An unrelated later claim is not authority to reveal its payload while A is unresolved.
          setCommand(null);
          setPreview(null);
          setCatalog(null);
          setPending({
            profile: "private-onboarding-pending-v1",
            state: "blocked",
            command: null,
            recovery_id: null,
          });
          setReady(true);
          setError(
            "Clearing the prior review is unconfirmed. Retry the same recovery request.",
          );
          return;
        }
        const old = saved.current,
          candidate = intent.current;
        const pin = await identity(result.command!);
        if (!current(n)) return;
        if (
          (old &&
            (old.command_id !== pin.command_id ||
              old.prepare_key !== pin.prepare_key ||
              old.recovery_id !== pin.recovery_id ||
              old.digest !== pin.digest ||
              (old.state !== "prepared" && old.state !== pin.state))) ||
          (candidate &&
            (candidate.key !== pin.prepare_key ||
              candidate.digest !== pin.digest))
        )
          throw new OnboardingFailure("unavailable");
        if (candidate) {
          setPending(result);
          setReady(true);
          setError(
            "Saving this review is unconfirmed. Retry the same saved review.",
          );
          return;
        }
        saved.current = pin;
        setCommand(result.command);
        setPreview(null);
        if (!clearID.current) {
          intent.current = null;
          setUncertain(null);
        }
      } else if (result.state === "inaccessible") {
        known.current = true;
        setClosureAvailable(true);
        if (saved.current && saved.current.recovery_id !== result.recovery_id) {
          setPending({
            profile: "private-onboarding-pending-v1",
            state: "blocked",
            command: null,
            recovery_id: null,
          });
          setCommand(null);
          setPreview(null);
          setCatalog(null);
          setReady(true);
          setError(
            "The prior setup outcome is unconfirmed. Resolve its exact recovery before new work.",
          );
          return;
        }
        setCatalog(null);
        setCommand(null);
        setPreview(null);
      } else if (intent.current || saved.current || clearID.current) {
        // None/another engine is not a receipt for our uncertain request.
        setCommand(null);
        setPreview(null);
        setCatalog(null);
        setError("The outcome is unconfirmed. Retry the same saved request.");
      }
      setPending(result);
      setReady(true);
    } catch (reason) {
      if (!current(n)) return;
      if (
        reason instanceof OnboardingFailure &&
        reason.kind === "unsupported" &&
        !known.current &&
        !intent.current &&
        !saved.current &&
        !clearID.current
      ) {
        acceptedAPI.current = api;
        setSupported(false);
        setPending(null);
        setReady(true);
      } else {
        setCatalog(null);
        setPreview(null);
        setCommand(null);
        setPending(null);
        setError(
          "Private setup recovery is unavailable. New work remains locked.",
        );
      }
    } finally {
      if (current(n)) setBusy(false);
    }
    // Current callbacks and intent refs do not restart discovery on each state update.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api, client]);
  useLayoutEffect(() => {
    epoch.current++;
    acceptedAPI.current = null;
    running.current = false;
    known.current = false;
    intent.current = null;
    clearID.current = null;
    saved.current = null;
    setReady(false);
    setSupported(false);
    setPending(null);
    setCatalog(null);
    setPreview(null);
    setCommand(null);
    setBusy(false);
    setUncertain(null);
    setClosureAvailable(false);
    setError("");
    setFocusVersion(0);
    return () => {
      epoch.current++;
    };
  }, [api]);
  useEffect(() => {
    if (!running.current) void discover();
  }, [discover, refreshVersion]);
  const loadCatalog = useCallback(async () => {
    const n = ++epoch.current;
    setBusy(true);
    setCatalog(null);
    setError("");
    try {
      const result = await client.catalog();
      if (current(n)) setCatalog(result);
    } catch {
      if (current(n))
        setError(
          "Private setup choices are unavailable. Retry checking setup.",
        );
    } finally {
      if (current(n)) setBusy(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api, client]);
  useEffect(() => {
    if (
      active &&
      ready &&
      supported &&
      pending?.state === "none" &&
      !intent.current &&
      !saved.current &&
      !clearID.current &&
      !preview &&
      !running.current
    )
      void loadCatalog();
  }, [active, ready, supported, pending, refreshVersion, loadCatalog]);
  const wrongSession = acceptedAPI.current !== api;
  const locked =
    wrongSession ||
    !ready ||
    busy ||
    !!uncertain ||
    !!intent.current ||
    !!clearID.current ||
    !!command ||
    (supported && pending?.state !== "none");
  useEffect(() => {
    callbacks.current.onLockChange(locked);
    callbacks.current.onStateChange?.({
      supported: !wrongSession && supported,
      locked,
      pending:
        !wrongSession &&
        ready &&
        pending?.state !== "blocked" &&
        (!!command ||
          pending?.state === "available" ||
          pending?.state === "inaccessible"),
    });
  }, [locked, supported, wrongSession, ready, command, pending?.state]);
  const fail = (reason: unknown, mutation: boolean) => {
    if (
      reason instanceof OnboardingFailure &&
      ["denied", "unsupported"].includes(reason.kind)
    ) {
      setCatalog(null);
      setPreview(null);
      setCommand(null);
      setPending(null);
      setReady(false);
      if (saved.current) setClosureAvailable(true);
    }
    setError(
      mutation
        ? "The outcome is unconfirmed. Check saved work or retry the same request."
        : "This review is no longer available. Check setup and review again.",
    );
  };
  const review = async (action: OnboardingAction) => {
    if (running.current || externalLocked || locked || !catalog) return;
    const n = ++epoch.current;
    running.current = true;
    setBusy(true);
    setError("");
    try {
      const value = await client.preview(action);
      if (!current(n)) return;
      if (
        value.operation !== action.operation ||
        (action.operation === "setup"
          ? !catalog.can_setup || value.workspace_id !== null
          : !catalog.can_adopt ||
            value.operation !== "adopt" ||
            catalog.workspace.state !== "ready" ||
            value.workspace_id !== catalog.workspace.workspace_id ||
            value.review.candidate_id !== action.candidate_id ||
            value.snapshot !== action.snapshot ||
            !catalog.candidates.some(
              (c) =>
                c.candidate_id === action.candidate_id &&
                c.snapshot === action.snapshot &&
                c.entity_id === value.review.entity_id &&
                c.book_id === value.review.book_id &&
                c.entity_name === value.review.entity_name &&
                c.book_name === value.review.book_name &&
                c.sources.length === value.review.sources.length &&
                c.sources.every((s) =>
                  value.review.sources.some(
                    (t) =>
                      t.source_id === s.source_id &&
                      t.display_name === s.display_name,
                  ),
                ),
            ))
      )
        throw new OnboardingFailure("unavailable");
      setPreview(value);
      focus();
    } catch (reason) {
      if (current(n)) fail(reason, false);
    } finally {
      if (current(n)) {
        running.current = false;
        setBusy(false);
      }
    }
  };
  const prepare = async () => {
    const old = intent.current;
    if (!old && preview && Date.parse(preview.expires_at) <= Date.now()) {
      setError(
        "This review expired. Go back to setup choices and review again.",
      );
      return;
    }
    if (
      running.current ||
      (!old &&
        (externalLocked ||
          !preview ||
          Date.parse(preview.expires_at) <= Date.now()))
    )
      return;
    const n = ++epoch.current;
    running.current = true;
    setBusy(true);
    setError("");
    let sent = false;
    try {
      const candidate = old ?? {
        preview_id: preview!.preview_id,
        digest: await onboardingPreviewDigest(preview!),
        key: crypto.randomUUID(),
      };
      if (!current(n)) return;
      intent.current = candidate;
      known.current = true;
      sent = true;
      const value = await client.prepare(candidate.preview_id, candidate.key);
      if (!current(n)) return;
      const pin = await identity(value);
      if (!current(n)) return;
      if (
        value.prepare_key !== candidate.key ||
        pin.digest !== candidate.digest
      )
        throw new OnboardingFailure("uncertain");
      saved.current = pin;
      intent.current = null;
      setUncertain(null);
      setCommand(value);
      setPreview(null);
      focus();
    } catch (reason) {
      if (!current(n)) return;
      if (!sent) {
        setError(
          "This review could not be pinned safely. Retry saving it before starting new work.",
        );
        return;
      }
      if (
        !old &&
        reason instanceof OnboardingFailure &&
        ["rejected", "conflict"].includes(reason.kind)
      ) {
        intent.current = null;
        setPreview(null);
        setCatalog(null);
        setUncertain(null);
        await callbacks.current.onConflict?.();
        running.current = false;
        await discover();
      } else {
        setUncertain("prepare");
        fail(reason, true);
      }
    } finally {
      if (current(n)) {
        running.current = false;
        setBusy(false);
      } else if (owner.current === api) {
        running.current = false;
        setBusy(false);
      }
    }
  };
  const transition = async (action: "execute" | "cancel" | "acknowledge") => {
    const previous = saved.current;
    if (
      !previous ||
      running.current ||
      (action === "execute" && externalLocked)
    )
      return;
    const n = ++epoch.current;
    running.current = true;
    setBusy(true);
    setError("");
    try {
      const value = await client.transition(previous, action);
      if (!current(n)) return;
      const pin = await identity(value);
      if (!current(n)) return;
      if (
        pin.command_id !== previous.command_id ||
        pin.prepare_key !== previous.prepare_key ||
        pin.recovery_id !== previous.recovery_id ||
        pin.digest !== previous.digest ||
        (previous.state !== "prepared" && value.state !== previous.state) ||
        (action === "acknowledge"
          ? !value.acknowledged || value.state !== previous.state
          : value.state === "prepared")
      )
        throw new OnboardingFailure("uncertain");
      saved.current = pin;
      setCommand(value);
      setUncertain(clearID.current ? "clear" : null);
      if (action === "acknowledge") {
        saved.current = null;
        setCommand(null);
        setCatalog(null);
        setPreview(null);
        callbacks.current.onChanged?.();
        running.current = false;
        await discover();
      } else focus();
    } catch (reason) {
      if (current(n)) {
        setUncertain(action);
        fail(reason, true);
      }
    } finally {
      if (owner.current === api) {
        running.current = false;
        setBusy(false);
      }
    }
  };
  const clear = async () => {
    const candidate =
      clearID.current ??
      (pending?.state === "inaccessible"
        ? pending.recovery_id
        : closureAvailable
          ? saved.current?.recovery_id
          : null);
    if (!candidate || running.current) return;
    clearID.current = candidate;
    known.current = true;
    const n = ++epoch.current;
    running.current = true;
    setBusy(true);
    setError("");
    try {
      await client.clear(candidate);
      if (!current(n)) return;
      clearID.current = null;
      setClosureAvailable(false);
      saved.current = null;
      intent.current = null;
      setCommand(null);
      setPreview(null);
      setUncertain(null);
      callbacks.current.onChanged?.();
      running.current = false;
      await discover();
    } catch (reason) {
      if (current(n)) {
        setUncertain("clear");
        fail(reason, true);
      }
    } finally {
      if (owner.current === api) {
        running.current = false;
        setBusy(false);
      }
    }
  };
  if (wrongSession || !ready)
    return (
      <section aria-label="Private transaction setup recovery">
        <p role="status">Checking private setup and saved work…</p>
        {error && <p role="alert">{error}</p>}
        <button disabled={busy} onClick={() => void discover()}>
          Check saved private setup
        </button>
        {uncertain === "prepare" && (
          <button disabled={busy} onClick={() => void prepare()}>
            Retry saving the same setup review
          </button>
        )}
        {uncertain === "clear" && (
          <button disabled={busy} onClick={() => void clear()}>
            Retry clearing the same setup review
          </button>
        )}
        {closureAvailable && saved.current && uncertain !== "clear" && (
          <button disabled={busy} onClick={() => void clear()}>
            Resolve exact setup recovery
          </button>
        )}
        {saved.current &&
          uncertain &&
          uncertain !== "prepare" &&
          uncertain !== "clear" && (
            <button disabled={busy} onClick={() => void transition(uncertain)}>
              {uncertain === "acknowledge"
                ? "Retry acknowledging setup result"
                : uncertain === "cancel"
                  ? "Retry cancelling the same setup review"
                  : "Retry applying the same setup review"}
            </button>
          )}
      </section>
    );
  if (!supported) return null;
  if (
    !active &&
    !command &&
    pending?.state !== "inaccessible" &&
    !uncertain &&
    !error
  )
    return null;
  const shown = command?.preview ?? preview;
  const panel = (
    <section
      className={styles.workspace}
      aria-label="Private transaction setup"
      aria-busy={busy}
    >
      <h2 ref={heading} tabIndex={-1}>
        {command
          ? "Saved private setup review"
          : preview
            ? "Review private transaction setup"
            : "Private transactions"}
      </h2>
      {error && <p role="alert">{error}</p>}
      {busy && <p role="status">Checking private setup…</p>}
      {pending?.state === "blocked" && (
        <p>
          Another saved request needs attention before private setup can
          continue.
        </p>
      )}
      {pending?.state === "inaccessible" && (
        <>
          <p>
            A saved private setup review is no longer accessible. Clearing it
            does not apply or reverse changes.
          </p>
          <button disabled={busy} onClick={() => void clear()}>
            Clear inaccessible setup review
          </button>
        </>
      )}
      {shown && (
        <>
          {shown.operation === "setup" ? (
            <p>
              Create your private transaction workspace to view transactions,
              connect eligible accounts, and manage labels and rules. This does
              not create a book, connect a bank or post a journal.
            </p>
          ) : (
            <>
              <h3>Use existing connection</h3>
              <p>
                {shown.review.entity_name} / {shown.review.book_name}
              </p>
              <p>
                Adopt existing history for all {shown.review.sources.length}{" "}
                registered accounts listed below. Existing automatic bank
                syncing for this connection will pause. This makes stored
                history available here; it does not start new syncing. No bank
                connection, provider fetch, new journal or copied transaction
                will be created.
              </p>
              <ul aria-label="Reviewed source accounts">
                {shown.review.sources.map((s) => (
                  <li key={s.source_id}>{s.display_name}</li>
                ))}
              </ul>
            </>
          )}
          {!command && (
            <>
              <button
                disabled={busy || externalLocked || !!intent.current}
                onClick={() => void prepare()}
              >
                Save setup review
              </button>
              <button
                disabled={busy || !!intent.current}
                onClick={() => {
                  setPreview(null);
                  void loadCatalog();
                }}
              >
                Back to setup choices
              </button>
            </>
          )}
        </>
      )}
      {command && (
        <>
          {command.state === "prepared" ? (
            <>
              <p>This saved review has not been applied.</p>
              <button
                disabled={busy || externalLocked}
                onClick={() => void transition("execute")}
              >
                {command.preview.operation === "setup"
                  ? "Create private workspace"
                  : "Adopt reviewed connection history"}
              </button>
              <button disabled={busy} onClick={() => void transition("cancel")}>
                Cancel saved setup review
              </button>
            </>
          ) : (
            <>
              <p role="status">
                {command.state === "cancelled"
                  ? "Private setup cancelled. No setup changes were applied."
                  : command.preview.operation === "setup"
                    ? "Private workspace created. No journal was created."
                    : "Existing connection history adopted. Existing automatic bank syncing paused. No journal was created."}
              </p>
              <button
                disabled={busy}
                onClick={() => void transition("acknowledge")}
              >
                Acknowledge setup result
              </button>
            </>
          )}
        </>
      )}
      {uncertain === "prepare" && (
        <button disabled={busy} onClick={() => void prepare()}>
          Retry saving the same setup review
        </button>
      )}
      {uncertain === "clear" && (
        <button disabled={busy} onClick={() => void clear()}>
          Retry clearing the same setup review
        </button>
      )}
      {closureAvailable &&
        saved.current &&
        pending?.state !== "inaccessible" &&
        uncertain !== "clear" && (
          <button disabled={busy} onClick={() => void clear()}>
            Resolve exact setup recovery
          </button>
        )}
      {!command &&
        saved.current &&
        uncertain &&
        uncertain !== "prepare" &&
        uncertain !== "clear" && (
          <button disabled={busy} onClick={() => void transition(uncertain)}>
            {uncertain === "acknowledge"
              ? "Retry acknowledging setup result"
              : uncertain === "cancel"
                ? "Retry cancelling the same setup review"
                : "Retry applying the same setup review"}
          </button>
        )}
      {!shown && !uncertain && pending?.state === "none" && (
        <>
          {catalog?.can_setup && (
            <>
              <p>
                Set up your private workspace to review transactions across your
                authorized accounts.
              </p>
              <button
                disabled={busy || externalLocked}
                onClick={() => void review({ operation: "setup" })}
              >
                Set up private transactions
              </button>
            </>
          )}
          {catalog?.can_adopt && (
            <>
              <h3>Use an existing connection</h3>
              <p>
                Review an authorized connection before adding its complete
                existing history to private transactions.
              </p>
              <ul aria-label="Eligible existing connections">
                {catalog.candidates.map((c) => (
                  <li key={c.candidate_id}>
                    <span>
                      {c.entity_name} / {c.book_name} · {c.sources.length}{" "}
                      source accounts
                    </span>{" "}
                    <button
                      disabled={busy || externalLocked}
                      onClick={() =>
                        void review({
                          operation: "adopt",
                          candidate_id: c.candidate_id,
                          snapshot: c.snapshot,
                        })
                      }
                    >
                      Review connection for {c.book_name}
                    </button>
                  </li>
                ))}
              </ul>
              {catalog.candidates.length === 0 && (
                <p>
                  No existing connection is currently eligible. Register
                  accounts through Connections, then check setup again.
                </p>
              )}
            </>
          )}
          {catalog &&
            !catalog.can_setup &&
            !catalog.can_adopt &&
            (catalog.workspace.state === "ready" ? (
              <>
                <p>
                  No existing connection is currently eligible for adoption.
                  Your private workspace is available; its current permissions
                  stay unchanged.
                </p>
                <p>
                  Register accounts through Connections, then check setup again.
                </p>
                {onConnections && (
                  <button
                    disabled={busy || externalLocked}
                    onClick={onConnections}
                  >
                    Open Connections
                  </button>
                )}
              </>
            ) : (
              <p>Private setup is not available with your current access.</p>
            ))}
        </>
      )}
      <button
        className="secondary"
        disabled={busy}
        onClick={() => void discover()}
      >
        Check saved private setup
      </button>
    </section>
  );
  return !shown &&
    !uncertain &&
    !error &&
    catalog?.workspace.state === "ready" &&
    catalog.candidates.length === 0 ? (
    <details>
      <summary>Private workspace and existing connections</summary>
      {panel}
    </details>
  ) : (
    panel
  );
}
