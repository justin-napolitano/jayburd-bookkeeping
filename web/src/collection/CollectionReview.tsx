"use client";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { formatAmount } from "../bookkeeping/amount";
import {
  CollectionFailure,
  samePreview,
  selectionKey,
  selectionOf,
  validCommand,
  validPage,
  validPreview,
  type Catalog,
  type CollectionPort,
  type LabelAction,
  type LabelCommand,
  type Page,
  type Preview,
  type Query,
  type Row,
  type Selection,
  type Selector,
  type SourceChoice,
  type Workspace,
} from "./types";
import styles from "./collection.module.css";
import { TaxonomyEditor, TopicReview } from "./TaxonomyEditor";

// Injected authorized projections/ports keep this component independent from unimplemented runtime authority.
export type CollectionReviewProps = {
  workspace: Workspace;
  sources: SourceChoice[];
  catalog: Catalog;
  port: CollectionPort;
  locked?: boolean;
  active?: boolean;
  canClassify?: boolean;
  onLockChange?: (locked: boolean) => void;
  onAcknowledged?: () => void;
  onRefreshCatalog?: () => void;
  onPrepareConflict?: () => Promise<void> | void;
  refreshVersion?: number;
  onDenied: () => void;
};
const initial: Selector = {
  source_account_id: null,
  assignment: "all",
  starts_on: null,
  through_date: null,
  search: "",
  topic: { kind: "all" },
  sort: "date_desc",
};

// Authority/taxonomy/scope changes unmount all memory-only private rows, keys and pending requests.
export function CollectionReview(props: CollectionReviewProps) {
  return (
    <Review
      key={`${props.workspace.workspace_id}/${props.workspace.authority_revision}/${props.catalog.taxonomy_revision}`}
      {...props}
    />
  );
}

// Visible state separates a read failure from an uncertain saved command without inventing outcomes.
function Review({
  workspace,
  sources,
  catalog,
  port,
  locked = false,
  active = true,
  canClassify = true,
  onLockChange,
  onAcknowledged,
  onRefreshCatalog,
  onPrepareConflict,
  refreshVersion = 0,
  onDenied,
}: CollectionReviewProps) {
  const [draft, setDraft] = useState<Selector>(initial),
    [selector, setSelector] = useState<Selector>(initial);
  const [query, setQuery] = useState<Query>({
    ...initial,
    profile: "private-collection-review-v1",
    workspace_id: workspace.workspace_id,
    limit: 25,
    snapshot: null,
    cursor: null,
  });
  const [page, setPage] = useState<Page | null>(null),
    [selected, setSelected] = useState<Selection[]>([]);
  const [choice, setChoice] = useState(""),
    [preview, setPreview] = useState<Preview | null>(null),
    [command, setCommand] = useState<LabelCommand | null>(null);
  const [ready, setReady] = useState(false),
    [reading, setReading] = useState(false),
    [busy, setBusy] = useState(false),
    [denied, setDenied] = useState(false);
  const [error, setError] = useState(""),
    [uncertain, setUncertain] = useState(false);
  const epoch = useRef(0),
    prepareKey = useRef<string | null>(null),
    pendingOutcome = useRef<"prepare" | "transition" | null>(null),
    commandId = useRef<string | null>(null),
    intent = useRef<Preview | null>(null),
    knownTerminal = useRef<"committed" | "cancelled" | null>(null),
    retryAction = useRef<"execute" | "cancel" | "acknowledge" | null>(null),
    onDeniedRef = useRef(onDenied),
    heading = useRef<HTMLHeadingElement>(null),
    savedHeading = useRef<HTMLHeadingElement>(null),
    focusRequest = useRef<{
      epoch: number;
      workspace: string;
      view: "preview" | "command";
    } | null>(null);
  onDeniedRef.current = onDenied;
  const recoveredVersion = useRef(refreshVersion);
  const reciprocalRefreshPending = recoveredVersion.current !== refreshVersion;
  const blocked =
    locked ||
    denied ||
    busy ||
    !ready ||
    !!command ||
    uncertain ||
    reciprocalRefreshPending;
  const title = useRef<HTMLHeadingElement>(null);
  const wasActive = useRef(active);
  useEffect(() => {
    if (active && !wasActive.current && !command && !preview)
      title.current?.focus();
    wasActive.current = active;
  }, [active, command, preview]);
  const lockCallback = useRef(onLockChange);
  const conflictCallback = useRef(onPrepareConflict);
  conflictCallback.current = onPrepareConflict;
  lockCallback.current = onLockChange;
  useEffect(() => {
    lockCallback.current?.(
      !denied &&
        (!ready ||
          busy ||
          !!preview ||
          !!command ||
          uncertain ||
          reciprocalRefreshPending),
    );
  }, [
    ready,
    busy,
    preview,
    command,
    uncertain,
    denied,
    reciprocalRefreshPending,
  ]);
  useEffect(() => {
    if (locked) {
      setPage(null);
      setSelected([]);
    }
  }, [locked]);

  // Denial destroys private state; fixed messages never echo raw service errors.
  const failure = useCallback((reason: unknown, mutation = false) => {
    setPage(null);
    setSelected([]);
    if (reason instanceof CollectionFailure && reason.kind === "denied") {
      epoch.current++;
      setDenied(true);
      setPreview(null);
      setCommand(null);
      prepareKey.current = null;
      pendingOutcome.current = null;
      commandId.current = null;
      focusRequest.current = null;
      intent.current = null;
      knownTerminal.current = null;
      retryAction.current = null;
      setUncertain(false);
      setError("Access to these transactions is no longer available.");
      onDeniedRef.current();
      return;
    }
    if (mutation) setUncertain(true);
    setError(
      mutation
        ? "The saved review outcome is unknown. Recover the saved review before continuing."
        : reason instanceof CollectionFailure && reason.kind === "stale"
          ? "Transactions or categories changed. Refresh before reviewing another selection."
          : reason instanceof CollectionFailure && reason.kind === "rejected"
            ? "The review was rejected. Refresh before reviewing another request."
            : "Transactions are unavailable. Refresh to try again.",
    );
  }, []);

  // Discovery never executes or acknowledges and must finish before new label work is enabled.
  const recover = useCallback(async () => {
    const n = ++epoch.current;
    setReady(false);
    setBusy(true);
    setError("");
    setPage(null);
    setSelected([]);
    try {
      const pending = await port.pending(workspace.workspace_id);
      if (n !== epoch.current) return;
      if (
        pending !== null &&
        (!validCommand(pending, workspace, catalog) || pending.acknowledged)
      )
        throw Error("invalid saved context");
      if (
        pendingOutcome.current &&
        (pending === null ||
          (pendingOutcome.current === "prepare" &&
            pending.prepare_key !== prepareKey.current) ||
          (pendingOutcome.current === "transition" &&
            (pending.command_id !== commandId.current ||
              pending.prepare_key !== prepareKey.current)) ||
          (intent.current !== null &&
            !samePreview(pending.preview, intent.current)) ||
          (knownTerminal.current !== null &&
            pending.state !== knownTerminal.current))
      ) {
        setUncertain(true);
        setError(
          "The same saved review is not available yet. Retry recovery before starting another selection.",
        );
        return;
      }
      pendingOutcome.current = null;
      commandId.current = pending?.command_id ?? null;
      intent.current = pending?.preview ?? null;
      knownTerminal.current =
        pending && pending.state !== "prepared" ? pending.state : null;
      retryAction.current = null;
      if (pending)
        focusRequest.current = {
          epoch: n,
          workspace: workspace.workspace_id,
          view: "command",
        };
      setCommand(pending);
      setUncertain(false);
      if (pending) {
        setPreview(null);
        prepareKey.current = pending.prepare_key;
      }
      setReady(true);
    } catch (reason) {
      if (n === epoch.current) failure(reason, true);
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  }, [port, workspace, catalog, failure]);
  useEffect(() => {
    void recover();
    return () => {
      epoch.current++;
    };
  }, [recover]);
  useEffect(() => {
    if (
      !reciprocalRefreshPending ||
      !ready ||
      busy ||
      preview ||
      command ||
      uncertain ||
      denied ||
      pendingOutcome.current
    )
      return;
    // Recheck pending in this session; never remount a newly busy or uncertain prepare.
    recoveredVersion.current = refreshVersion;
    void recover();
  }, [
    reciprocalRefreshPending,
    refreshVersion,
    ready,
    busy,
    preview,
    command,
    uncertain,
    denied,
    recover,
  ]);

  // Focus only the committed current review; frames can run before React attaches its heading.
  useEffect(() => {
    const request = focusRequest.current;
    if (!request) return;
    if (locked) return;
    focusRequest.current = null;
    if (
      request.epoch !== epoch.current ||
      request.workspace !== workspace.workspace_id ||
      denied ||
      locked
    )
      return;
    if (
      request.view === "preview" &&
      preview?.workspace_id === workspace.workspace_id
    )
      heading.current?.focus();
    if (
      request.view === "command" &&
      command?.workspace_id === workspace.workspace_id
    )
      savedHeading.current?.focus();
  }, [preview, command, denied, locked, workspace.workspace_id]);

  // Every page request carries the exact selector and server-owned all-source snapshot.
  useEffect(() => {
    if (!active) {
      setPage(null);
      setSelected([]);
      setReading(false);
      return;
    }
    if (
      !ready ||
      locked ||
      denied ||
      command ||
      uncertain ||
      reciprocalRefreshPending
    )
      return;
    const n = ++epoch.current;
    setReading(true);
    setPage(null);
    setSelected([]);
    setError("");
    void port
      .query(query)
      .then((result) => {
        if (n !== epoch.current) return;
        if (!validPage(result, query, workspace, catalog))
          throw Error("invalid query context");
        setPage(result);
      })
      .catch((reason) => {
        if (n === epoch.current) failure(reason);
      })
      .finally(() => {
        if (n === epoch.current) setReading(false);
      });
    return () => {
      if (n === epoch.current) epoch.current++;
    };
  }, [
    query,
    active,
    ready,
    reciprocalRefreshPending,
    locked,
    denied,
    command,
    uncertain,
    port,
    workspace,
    catalog,
    failure,
  ]);

  // A new selector or page cannot retain an old selection, reviewed preview or prepare identity.
  const read = (next: Query) => {
    epoch.current++;
    focusRequest.current = null;
    setPage(null);
    setSelected([]);
    setPreview(null);
    prepareKey.current = null;
    setError("");
    setQuery(next);
  };
  const refresh = () =>
    read({
      ...selector,
      profile: "private-collection-review-v1",
      workspace_id: workspace.workspace_id,
      limit: 25,
      snapshot: null,
      cursor: null,
    });
  const applyFilters = (event: FormEvent) => {
    event.preventDefault();
    if (Array.from(draft.search).length > 100) {
      setError(
        "Search supports up to 100 characters. Shorten the search and retry.",
      );
      return;
    }
    if (
      draft.starts_on &&
      draft.through_date &&
      draft.starts_on > draft.through_date
    ) {
      setError("Choose an ordered start and end date.");
      return;
    }
    setSelector(draft);
    read({
      ...draft,
      profile: "private-collection-review-v1",
      workspace_id: workspace.workspace_id,
      limit: 25,
      snapshot: null,
      cursor: null,
    });
  };
  const select = (row: Row, checked: boolean) => {
    setPreview(null);
    prepareKey.current = null;
    setError("");
    setSelected((old) =>
      checked
        ? [...old, selectionOf(row)]
        : old.filter((item) => selectionKey(item) !== selectionKey(row)),
    );
  };
  // Preview is inert and compares every selected composite identity/revision before showing a review.
  const previewLabels = async () => {
    if (!selected.length || !choice) return;
    const action: LabelAction =
      choice === "uncategorized"
        ? { kind: "uncategorized" }
        : { kind: "topic", topic_id: choice.slice(6) };
    await previewAction(action, selected);
  };
  // Both taxonomy and labels share the exact inert-preview and durable recovery engine.
  const previewAction = async (action: LabelAction, selection: Selection[]) => {
    if (!page || blocked || !canClassify) return;
    const request = {
      workspace_id: workspace.workspace_id,
      snapshot: page.snapshot,
      selection,
      action,
    };
    const n = ++epoch.current;
    setBusy(true);
    setError("");
    setPreview(null);
    prepareKey.current = null;
    try {
      const result = await port.preview(request);
      if (n !== epoch.current) return;
      if (
        !validPreview(result, request) ||
        Date.parse(result.expires_at) <= Date.now()
      )
        throw new CollectionFailure("stale");
      focusRequest.current = {
        epoch: n,
        workspace: workspace.workspace_id,
        view: "preview",
      };
      setPreview(result);
    } catch (reason) {
      if (n === epoch.current) failure(reason);
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  };
  // An uncertain prepare retains its exact key/preview; retry never creates different intent.
  const prepare = async () => {
    if (!preview) return;
    const wasUncertain = uncertain || pendingOutcome.current !== null;
    if (Date.parse(preview.expires_at) <= Date.now()) {
      if (!wasUncertain) {
        setPreview(null);
        prepareKey.current = null;
      }
      failure(new CollectionFailure("stale"), wasUncertain);
      return;
    }
    const key = prepareKey.current ?? crypto.randomUUID();
    prepareKey.current = key;
    intent.current = preview;
    pendingOutcome.current = "prepare";
    const n = ++epoch.current;
    setBusy(true);
    setError("");
    try {
      const saved = await port.prepare({
        workspace_id: workspace.workspace_id,
        preview_id: preview.preview_id,
        prepare_key: key,
      });
      if (n !== epoch.current) return;
      if (
        !validCommand(saved, workspace, catalog) ||
        saved.prepare_key !== key ||
        saved.preview.preview_id !== preview.preview_id ||
        !samePreview(saved.preview, preview)
      )
        throw Error("invalid saved intent");
      pendingOutcome.current = null;
      commandId.current = saved.command_id;
      knownTerminal.current = saved.state === "prepared" ? null : saved.state;
      focusRequest.current = {
        epoch: n,
        workspace: workspace.workspace_id,
        view: "command",
      };
      setCommand(saved);
      setPreview(null);
      setPage(null);
      setSelected([]);
      setUncertain(false);
    } catch (reason) {
      if (n === epoch.current) {
        if (
          !wasUncertain &&
          reason instanceof CollectionFailure &&
          ["stale", "rejected"].includes(reason.kind)
        ) {
          pendingOutcome.current = null;
          intent.current = null;
          prepareKey.current = null;
          setPreview(null);
          failure(reason);
          // A rejected prepare may conflict with another tab's saved command.
          // Keep the shared gate closed until fresh pending discovery completes.
          setReady(false);
          await Promise.all([
            recover(),
            reason.kind === "stale" ? conflictCallback.current?.() : undefined,
          ]);
        } else failure(reason, true);
      }
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  };
  // Only a user action invokes a saved command, and all replies must retain its exact immutable intent.
  const transition = async (action: "execute" | "cancel" | "acknowledge") => {
    if (!command) return;
    pendingOutcome.current = "transition";
    retryAction.current = action;
    const current = command,
      n = ++epoch.current;
    setBusy(true);
    setError("");
    try {
      const result = await port[action]({
        workspace_id: workspace.workspace_id,
        command_id: current.command_id,
      });
      if (n !== epoch.current) return;
      if (
        !validCommand(result, workspace, catalog) ||
        result.command_id !== current.command_id ||
        result.prepare_key !== current.prepare_key ||
        result.preview.preview_id !== current.preview.preview_id ||
        !samePreview(result.preview, current.preview) ||
        ((action === "execute" || action === "cancel") &&
          result.state === "prepared") ||
        (action === "acknowledge" &&
          (!result.acknowledged ||
            result.state === "prepared" ||
            result.state !== current.state))
      )
        throw Error("invalid command outcome");
      setUncertain(false);
      pendingOutcome.current = null;
      retryAction.current = null;
      if (action === "acknowledge") {
        commandId.current = null;
        intent.current = null;
        knownTerminal.current = null;
        setCommand(null);
        prepareKey.current = null;
        // The validated acknowledgement completed before a new read advances its epoch.
        setBusy(false);
        if (onAcknowledged) onAcknowledged();
        else {
          setReady(true);
          refresh();
        }
      } else {
        knownTerminal.current =
          result.state === "prepared" ? null : result.state;
        focusRequest.current = {
          epoch: n,
          workspace: workspace.workspace_id,
          view: "command",
        };
        setCommand(result);
      }
    } catch (reason) {
      if (n === epoch.current) failure(reason, true);
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  };

  if (locked)
    return (
      <p role="status">Finish the pending action to review all transactions.</p>
    );
  return (
    <section
      className={styles.workspace}
      aria-label="All transactions"
      aria-busy={busy || reading}
    >
      <div className={styles.heading}>
        <div>
          <h2 ref={title} tabIndex={-1}>
            All transactions
          </h2>
          <p>{workspace.name}</p>
        </div>
        <button
          type="button"
          className="secondary"
          disabled={blocked || reading}
          onClick={() => (onRefreshCatalog ? onRefreshCatalog() : refresh())}
          aria-label="Refresh transactions"
        >
          Refresh
        </button>
      </div>
      {!command && (
        <form className={styles.filters} onSubmit={applyFilters}>
          <fieldset disabled={blocked || reading || !!preview}>
            <div className={styles.primaryFilters}>
              <label>
                Account
                <select
                  value={draft.source_account_id ?? ""}
                  onChange={(event) =>
                    setDraft({
                      ...draft,
                      source_account_id: event.target.value || null,
                    })
                  }
                >
                  <option value="">All authorized accounts</option>
                  {sources.map((source) => (
                    <option key={source.source_id} value={source.source_id}>
                      {source.name}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Search transactions
                <input
                  type="search"
                  maxLength={200}
                  value={draft.search}
                  onChange={(event) =>
                    setDraft({ ...draft, search: event.target.value })
                  }
                />
              </label>
              <button type="submit">Apply filters</button>
            </div>
            <details className={styles.moreFilters}>
              <summary>Book, topic and date filters</summary>
              <div className={styles.advancedFilters}>
                <label>
                  Book assignment
                  <select
                    value={draft.assignment}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        assignment: event.target
                          .value as Selector["assignment"],
                      })
                    }
                  >
                    <option value="all">All assignments</option>
                    <option value="assigned">Assigned to a book</option>
                    <option value="unassigned">Unassigned</option>
                  </select>
                </label>
                <label>
                  Topic filter
                  <select
                    value={
                      draft.topic.kind === "topic"
                        ? `topic:${draft.topic.topic_id}`
                        : draft.topic.kind
                    }
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        topic: event.target.value.startsWith("topic:")
                          ? {
                              kind: "topic",
                              topic_id: event.target.value.slice(6),
                            }
                          : {
                              kind: event.target.value as
                                | "all"
                                | "unclassified"
                                | "uncategorized",
                            },
                      })
                    }
                  >
                    <option value="all">All topics</option>
                    <option value="unclassified">Not classified</option>
                    <option value="uncategorized">
                      Explicit Uncategorized
                    </option>
                    {catalog.topics.map((topic) => (
                      <option
                        key={topic.topic_id}
                        value={`topic:${topic.topic_id}`}
                      >
                        {topic.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  From
                  <input
                    type="date"
                    min="0001-01-01"
                    max="9999-12-31"
                    value={draft.starts_on ?? ""}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        starts_on: event.target.value || null,
                      })
                    }
                  />
                </label>
                <label>
                  Through
                  <input
                    type="date"
                    min={draft.starts_on ?? "0001-01-01"}
                    max="9999-12-31"
                    value={draft.through_date ?? ""}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        through_date: event.target.value || null,
                      })
                    }
                  />
                </label>
                <label>
                  Date order
                  <select
                    value={draft.sort}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        sort: event.target.value as Selector["sort"],
                      })
                    }
                  >
                    <option value="date_desc">Newest first</option>
                    <option value="date_asc">Oldest first</option>
                  </select>
                </label>
              </div>
            </details>
          </fieldset>
        </form>
      )}
      {(!ready || reading || busy) && !denied && (
        <p role="status">
          {!ready
            ? "Checking saved label review…"
            : busy
              ? "Checking review…"
              : "Loading transactions…"}
        </p>
      )}
      {error && <p role="alert">{error}</p>}
      {uncertain && !denied && (
        <div className={styles.recovery}>
          <button disabled={busy} onClick={() => void recover()}>
            Recover saved review
          </button>
          {!command && preview && (
            <button
              className="secondary"
              disabled={busy}
              onClick={() => void prepare()}
            >
              Retry saving same review
            </button>
          )}
          {command && retryAction.current && (
            <button
              className="secondary"
              disabled={busy}
              onClick={() => void transition(retryAction.current!)}
            >
              Retry same{" "}
              {retryAction.current === "acknowledge"
                ? "acknowledgement"
                : retryAction.current === "cancel"
                  ? "cancellation"
                  : "confirmation"}
            </button>
          )}
        </div>
      )}
      {page && !command && (
        <>
          <div className={styles.resultSummary}>
            <p className={styles.count}>
              {page.items.length
                ? `${page.start_index}–${(BigInt(page.start_index) + BigInt(page.items.length) - 1n).toString()}`
                : "0"}{" "}
              of {page.matching_count} matching transactions
            </p>
            <p>{selected.length} selected on this page</p>
          </div>
          {page.partial && (
            <p role="status" className={styles.partial}>
              Some source history is unavailable. Account history is incomplete.
            </p>
          )}
          {!page.items.length ? (
            <p>
              No transactions match these filters. Change filters or refresh
              imported activity.
            </p>
          ) : (
            <ul className={styles.rows} aria-label="Transaction rows">
              {page.items.map((row) => (
                <li key={selectionKey(row)} className={styles.row}>
                  <label className={styles.check}>
                    <input
                      type="checkbox"
                      aria-label={`Select ${row.description} on ${row.booked_date} from ${row.account_name}`}
                      checked={selected.some(
                        (item) => selectionKey(item) === selectionKey(row),
                      )}
                      disabled={blocked || !!preview || reading || !canClassify}
                      onChange={(event) => select(row, event.target.checked)}
                    />
                    <span className={styles.date}>{row.booked_date}</span>
                  </label>
                  <div className={styles.description}>
                    <strong>{row.description}</strong>
                    <span>{row.account_name}</span>
                    <span>
                      {row.merchant === null
                        ? "Merchant not provided"
                        : `Merchant: ${row.merchant}`}
                    </span>
                    <div className={styles.states}>
                      <span>Source: {row.source_state}</span>
                      <span>{assignmentLabel(row)}</span>
                      <span>
                        Journal: {row.journal_state.replaceAll("_", " ")}
                      </span>
                    </div>
                    <details>
                      <summary>Labels and provenance</summary>
                      <p>Owner topic: {topicLabel(row)}</p>
                      <p>
                        {row.provider_suggestion.state === "attributed"
                          ? `Provider suggestion: ${row.provider_suggestion.label}; taxonomy ${row.provider_suggestion.taxonomy_version}; reported confidence ${row.provider_suggestion.confidence}.`
                          : "Provider suggestion unavailable."}
                      </p>
                      <p>
                        {row.money.state === "available"
                          ? `Original currency reference: ${row.money.currency_version_id}; ${row.money.numeric_code === null ? "numeric code not supplied" : row.money.numeric_code}; valid from ${row.money.valid_from}${row.money.valid_to ? ` until ${row.money.valid_to}` : ""}.`
                          : `Money unavailable: ${row.money.reason.replaceAll("_", " ")}.`}
                      </p>
                    </details>
                  </div>
                  <p className={styles.money}>
                    <MoneyLabel row={row} />
                  </p>
                </li>
              ))}
            </ul>
          )}
          <p className={styles.note}>
            Original imported activity. Topics help organize transactions;
            journal state is shown separately.
          </p>
          <div className={styles.actions}>
            <label>
              Label selected transactions
              <select
                value={choice}
                disabled={
                  blocked || !!preview || !selected.length || !canClassify
                }
                onChange={(event) => {
                  setChoice(event.target.value);
                  setPreview(null);
                  prepareKey.current = null;
                }}
              >
                <option value="">Choose a topic</option>
                <option value="uncategorized">Explicit Uncategorized</option>
                {catalog.topics
                  .filter((topic) => topic.status === "active")
                  .map((topic) => (
                    <option
                      key={topic.topic_id}
                      value={`topic:${topic.topic_id}`}
                    >
                      {topic.name}
                    </option>
                  ))}
              </select>
            </label>
            <button
              disabled={
                blocked ||
                !!preview ||
                !selected.length ||
                !choice ||
                !canClassify
              }
              onClick={() => void previewLabels()}
            >
              Preview labels for {selected.length} selected
            </button>
          </div>
          <div className={styles.paging}>
            <button
              className="secondary"
              disabled={
                blocked || !!preview || reading || query.cursor === null
              }
              onClick={refresh}
            >
              First page
            </button>
            <button
              disabled={
                blocked || !!preview || reading || page.next_cursor === null
              }
              onClick={() =>
                read({
                  ...query,
                  snapshot: page.snapshot,
                  cursor: page.next_cursor,
                })
              }
            >
              Next page
            </button>
          </div>
          {!canClassify && (
            <p>Topics are read-only with your current access.</p>
          )}
          {canClassify && (
            <TaxonomyEditor
              topics={catalog.topics}
              disabled={blocked || !!preview || reading}
              onPreview={(topics) =>
                void previewAction({ kind: "taxonomy", topics }, [])
              }
            />
          )}
        </>
      )}
      {preview && !command && !denied && (
        <div role="region" className={styles.review} aria-label="Label preview">
          <h3 tabIndex={-1} ref={heading}>
            {preview.action.kind === "taxonomy"
              ? "Review topic changes"
              : `Review ${preview.selected_count} selected transactions`}
          </h3>
          <p>
            {preview.action.kind === "taxonomy"
              ? "Replace the private topic edition with "
              : "Set owner topic to "}
            <strong>{actionLabel(preview.action, catalog)}</strong>. Source
            facts and journals stay unchanged.
          </p>
          {preview.action.kind === "taxonomy" ? (
            <TopicReview topics={preview.action.topics} />
          ) : (
            <ReviewRows rows={preview.review_rows} />
          )}
          <button disabled={busy || uncertain} onClick={() => void prepare()}>
            {preview.action.kind === "taxonomy"
              ? "Save topic review"
              : "Save label review"}
          </button>
          <button
            className="secondary"
            disabled={busy || uncertain}
            onClick={() => {
              setPreview(null);
              prepareKey.current = null;
            }}
          >
            {preview.action.kind === "taxonomy"
              ? "Back to topic editor"
              : "Back to selection"}
          </button>
        </div>
      )}
      {command && !denied && (
        <div
          role="region"
          className={styles.review}
          aria-label="Saved label review"
        >
          <h3 tabIndex={-1} ref={savedHeading}>
            {command.preview.action.kind === "taxonomy"
              ? "Saved topic review"
              : "Saved label review"}
          </h3>
          <p>
            {command.preview.action.kind === "taxonomy"
              ? "Private topic edition: "
              : `${command.preview.selected_count} selected transactions; owner topic: `}
            <strong>{actionLabel(command.preview.action, catalog)}</strong>.
          </p>
          {command.preview.action.kind === "taxonomy" ? (
            <TopicReview topics={command.preview.action.topics} />
          ) : (
            <ReviewRows rows={command.preview.review_rows} />
          )}
          {command.acknowledged && (
            <p>
              This result was acknowledged in another session. Acknowledge here
              to continue.
            </p>
          )}
          {command.state === "prepared" ? (
            <>
              <p>
                {uncertain
                  ? "The saved review outcome is unresolved."
                  : "This saved review has not been applied."}
              </p>
              <button
                disabled={busy || uncertain || locked || !canClassify}
                onClick={() => void transition("execute")}
              >
                {command.preview.action.kind === "taxonomy"
                  ? "Confirm topic changes"
                  : `Confirm labels for ${command.preview.selected_count} selected`}
              </button>
              <button
                className="secondary"
                disabled={busy || uncertain || locked}
                onClick={() => void transition("cancel")}
              >
                Cancel saved review
              </button>
            </>
          ) : (
            <>
              <p role="status">
                {command.state === "committed"
                  ? command.preview.action.kind === "taxonomy"
                    ? "Topics updated. No journal was created."
                    : "Labels applied. No journal was created."
                  : command.preview.action.kind === "taxonomy"
                    ? "Saved topic review cancelled."
                    : "Saved label review cancelled."}
              </p>
              <button
                disabled={busy || uncertain || locked}
                onClick={() => void transition("acknowledge")}
              >
                Acknowledge result
              </button>
            </>
          )}
        </div>
      )}
    </section>
  );
}

// These labels never infer an unassigned source from a denied book projection.
function assignmentLabel(row: Row): string {
  return row.assignment.state === "assigned"
    ? `Books: ${row.assignment.books.map((book) => book.name).join(", ")}`
    : row.assignment.state === "unassigned"
      ? "Unassigned to a book"
      : "Book assignment unavailable";
}
function topicLabel(row: Row): string {
  return row.owner_topic.state === "topic"
    ? row.owner_topic.name
    : row.owner_topic.state === "uncategorized"
      ? "Explicit Uncategorized"
      : row.owner_topic.state === "stale"
        ? "Needs confirmation after source change"
        : "Not classified";
}
function actionLabel(action: LabelAction, catalog: Catalog): string {
  if (action.kind === "taxonomy") return `${action.topics.length} topics`;
  return action.kind === "uncategorized"
    ? "Explicit Uncategorized"
    : (catalog.topics.find((topic) => topic.topic_id === action.topic_id)
        ?.name ?? "Selected topic");
}
// The saved server projection exposes exact selected values without aggregation or hidden expansion.
function ReviewRows({ rows }: { rows: Row[] }) {
  return (
    <ul className={styles.reviewRows}>
      {rows.map((row) => (
        <li key={selectionKey(row)}>
          <span>
            {row.booked_date} · {row.account_name} · {row.description}
          </span>
          <span>
            <MoneyLabel row={row} />
          </span>
        </li>
      ))}
    </ul>
  );
}

// Holds disclose unavailable money without substituting zero, a direction or a guessed currency.
function MoneyLabel({ row }: { row: Row }) {
  return row.money.state === "held" ? (
    <>
      <span>Money unavailable</span>
      <span>{row.money.reason.replaceAll("_", " ")}</span>
    </>
  ) : (
    <>
      <span>
        {row.money.direction === "inflow"
          ? "In"
          : row.money.direction === "outflow"
            ? "Out"
            : "Zero movement"}
      </span>
      <strong>
        {formatAmount(row.money.amount, row.money.minor_unit)}{" "}
        {row.money.currency}
      </strong>
    </>
  );
}
