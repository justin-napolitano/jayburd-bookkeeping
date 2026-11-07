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

// Injected authorized projections/ports keep this component independent from unimplemented runtime authority.
export type CollectionReviewProps = {
  workspace: Workspace;
  sources: SourceChoice[];
  catalog: Catalog;
  port: CollectionPort;
  locked?: boolean;
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
      key={`${props.workspace.workspace_id}/${props.workspace.authority_revision}/${props.catalog.taxonomy_revision}/${!!props.locked}`}
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
  const blocked = locked || denied || busy || !ready || !!command || uncertain;

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
      reason instanceof CollectionFailure && reason.kind === "stale"
        ? "Transactions or categories changed. Refresh before reviewing another selection."
        : mutation
          ? "The saved review outcome is unknown. Recover the saved review before continuing."
          : "Transactions are unavailable. Refresh to try again.",
    );
  }, []);

  // Discovery never executes or acknowledges and must finish before new label work is enabled.
  const recover = useCallback(async () => {
    const n = ++epoch.current;
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
    if (!locked) void recover();
    return () => {
      epoch.current++;
    };
  }, [recover, locked]);

  // Focus only the committed current review; frames can run before React attaches its heading.
  useEffect(() => {
    const request = focusRequest.current;
    if (!request) return;
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
    if (!ready || locked || denied || command || uncertain) return;
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
    ready,
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
    if (!page || !selected.length || !choice) return;
    const action: LabelAction =
      choice === "uncategorized"
        ? { kind: "uncategorized" }
        : { kind: "topic", topic_id: choice.slice(6) };
    const request = {
      workspace_id: workspace.workspace_id,
      snapshot: page.snapshot,
      selection: selected,
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
    if (Date.parse(preview.expires_at) <= Date.now()) {
      failure(new CollectionFailure("stale"));
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
        !samePreview(saved.preview, preview) ||
        saved.acknowledged
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
      if (n === epoch.current) failure(reason, true);
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
          (result.state === "prepared" || result.acknowledged)) ||
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
        refresh();
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
          <h2>All transactions</h2>
          <p>{workspace.name}</p>
        </div>
        <button
          type="button"
          className="secondary"
          disabled={blocked || reading}
          onClick={refresh}
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
                  maxLength={100}
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
                      disabled={blocked || !!preview || reading}
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
            Original imported activity. Topics help organize transactions; journal
            state is shown separately.
          </p>
          <div className={styles.actions}>
            <label>
              Label selected transactions
              <select
                value={choice}
                disabled={blocked || !!preview || !selected.length}
                onChange={(event) => {
                  setChoice(event.target.value);
                  setPreview(null);
                  prepareKey.current = null;
                }}
              >
                <option value="">Choose a topic</option>
                <option value="uncategorized">Explicit Uncategorized</option>
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
            <button
              disabled={blocked || !!preview || !selected.length || !choice}
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
        </>
      )}
      {preview && !command && !denied && (
        <div role="region" className={styles.review} aria-label="Label preview">
          <h3 tabIndex={-1} ref={heading}>
            Review {preview.selected_count} selected transactions
          </h3>
          <p>
            Set owner topic to{" "}
            <strong>{actionLabel(preview.action, catalog)}</strong>. Source
            facts and journals stay unchanged.
          </p>
          <ReviewRows rows={preview.review_rows} />
          <button disabled={busy || uncertain} onClick={() => void prepare()}>
            Save label review
          </button>
          <button
            className="secondary"
            disabled={busy || uncertain}
            onClick={() => {
              setPreview(null);
              prepareKey.current = null;
            }}
          >
            Back to selection
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
            Saved label review
          </h3>
          <p>
            {command.preview.selected_count} selected transactions; owner topic:{" "}
            <strong>{actionLabel(command.preview.action, catalog)}</strong>.
          </p>
          <ReviewRows rows={command.preview.review_rows} />
          {command.state === "prepared" ? (
            <>
              <p>
                {uncertain
                  ? "The saved review outcome is unresolved."
                  : "This saved review has not been applied."}
              </p>
              <button
                disabled={busy || uncertain}
                onClick={() => void transition("execute")}
              >
                Confirm labels for {command.preview.selected_count} selected
              </button>
              <button
                className="secondary"
                disabled={busy || uncertain}
                onClick={() => void transition("cancel")}
              >
                Cancel saved review
              </button>
            </>
          ) : (
            <>
              <p role="status">
                {command.state === "committed"
                  ? "Labels applied. No journal was created."
                  : "Saved label review cancelled."}
              </p>
              <button
                disabled={busy || uncertain}
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
