"use client";
import { AutomaticCategorization } from "./AutomaticCategorization";
import { CategorizationSummary } from "./CategorizationSummary";
import {
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
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
  type StoredPreview,
  type LegacyRow,
  type TagResult,
  type Query,
  type Row,
  type Selection,
  type Selector,
  type SourceChoice,
  type Workspace,
  type DetailRequest,
} from "./types";
import styles from "./collection.module.css";
import { RuleEditor } from "./RuleEditor";
import type { RuleSeed } from "./rules";
import { correctionRuleSeed } from "./ruleSeed";
import {
  RuleCatalogReview,
  RuleResultsView,
  RuleSuggestionView,
  RuleApplicationView,
} from "./RuleViews";
import { CollectionFilters } from "./CollectionFilters";
import { collectionFilterError } from "./filters";
import { CollectionDetail } from "./CollectionDetail";
import type { JournalReader } from "./journal";
import { TaxonomyEditor, TopicReview } from "./TaxonomyEditor";
import { TagCatalogReview, TagEditor, TagSelection } from "./TagEditor";

// Injected authorized projections/ports keep this component independent from unimplemented runtime authority.
export type CollectionReviewProps = {
  workspace: Workspace;
  sources: SourceChoice[];
  catalog: Catalog;
  port: CollectionPort;
  locked?: boolean;
  active?: boolean;
  reviewVisible?: boolean;
  canClassify?: boolean;
  onLockChange?: (locked: boolean) => void;
  onAcknowledged?: (seed?: RuleSeed) => void;
  ruleSeed?: RuleSeed | null;
  onSeedConsumed?: () => void;
  onRefreshCatalog?: () => void;
  onAutomaticRun?: (runID: string) => void;
  onPrepareConflict?: () => Promise<void> | void;
  refreshVersion?: number;
  newWorkLocked?: boolean;
  onDenied: () => void;
  readJournal?: JournalReader;
};
const initial: Selector = {
  merchant: { kind: "all" },
  amount: null,
  institution: { kind: "all" },
  legal_owner: { kind: "all" },
  book: { kind: "all" },
  status: { source: "all", journal: "all" },
  source_account_id: null,
  assignment: "all",
  starts_on: null,
  through_date: null,
  search: "",
  topic: { kind: "all" },
  tag: { kind: "all" },
  sort: { field: "date", order: "desc" },
};

// Authority/taxonomy/scope changes unmount all memory-only private rows, keys and pending requests.
export function CollectionReview(props: CollectionReviewProps) {
  return (
    <Review
      key={`${props.workspace.workspace_id}/${props.workspace.authority_revision}/${props.catalog.taxonomy_revision}/${props.catalog.tag_revision}/${props.catalog.ruleset_revision}`}
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
  reviewVisible = true,
  canClassify = true,
  onLockChange,
  onAcknowledged,
  onRefreshCatalog,
  onAutomaticRun,
  onPrepareConflict,
  refreshVersion = 0,
  newWorkLocked = false,
  readJournal,
  ruleSeed,
  onSeedConsumed,
  onDenied,
}: CollectionReviewProps) {
  const [automationLocked, setAutomationLocked] = useState(false);
  const topicChoiceLabel = useId();
  const afterAckSeed = useRef<RuleSeed | null>(null);
  const [detail, setDetail] = useState<DetailRequest | null>(null);
  const rowButtons = useRef(new Map<string, HTMLButtonElement>());
  const detailReturn = useRef<string | null>(null);
  const tagFilterLabel = useId();
  const [draft, setDraft] = useState<Selector>(initial),
    [selector, setSelector] = useState<Selector>(initial);
  const [query, setQuery] = useState<Query>({
    ...initial,
    profile: "private-collection-review-v3",
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
    intent = useRef<StoredPreview | null>(null),
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
    automationLocked ||
    newWorkLocked ||
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
          reciprocalRefreshPending ||
          automationLocked),
    );
  }, [
    ready,
    busy,
    preview,
    command,
    uncertain,
    denied,
    reciprocalRefreshPending,
    automationLocked,
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
    setDetail(null);
    if (reason instanceof CollectionFailure && reason.kind === "denied")
      afterAckSeed.current = null;
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

  // Focus the committed current review before paint, after React attaches its heading.
  useLayoutEffect(() => {
    const request = focusRequest.current;
    if (!request) return;
    if (locked || !reviewVisible) return;
    if (
      request.epoch !== epoch.current ||
      request.workspace !== workspace.workspace_id ||
      denied ||
      locked
    ) {
      focusRequest.current = null;
      return;
    }
    const target =
      request.view === "preview" &&
      preview?.workspace_id === workspace.workspace_id
        ? heading.current
        : request.view === "command" &&
            command?.workspace_id === workspace.workspace_id
          ? savedHeading.current
          : null;
    if (!target?.isConnected) return;
    target.focus();
    if (document.activeElement === target) focusRequest.current = null;
  }, [preview, command, denied, locked, reviewVisible, workspace.workspace_id]);

  // Every page request carries the exact selector and server-owned all-source snapshot.
  useEffect(() => {
    if (!active) {
      setDetail(null);
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
    setDetail(null);
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
      profile: "private-collection-review-v3",
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
    const filterError = collectionFilterError(draft, catalog);
    if (filterError) {
      setError(filterError);
      return;
    }
    setSelector(draft);
    read({
      ...draft,
      profile: "private-collection-review-v3",
      workspace_id: workspace.workspace_id,
      limit: 25,
      snapshot: null,
      cursor: null,
    });
  };
  // Reset follows the ordinary new-query path, clearing stale selection and pins.
  const resetFilters = () => {
    setDraft(initial);
    setSelector(initial);
    read({
      ...initial,
      profile: "private-collection-review-v3",
      workspace_id: workspace.workspace_id,
      limit: 25,
      snapshot: null,
      cursor: null,
    });
  };
  const appliedFilters = Object.keys(initial).filter(
    (key) =>
      JSON.stringify(selector[key as keyof Selector]) !==
      JSON.stringify(initial[key as keyof Selector]),
  ).length;
  const filtersChanged = JSON.stringify(draft) !== JSON.stringify(selector);
  const hasFilters =
    appliedFilters > 0 || JSON.stringify(draft) !== JSON.stringify(initial);

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
      if (n === epoch.current) {
        if (reason instanceof CollectionFailure && reason.kind === "rejected")
          setError(
            action.kind === "rule_catalog"
              ? "The rule preview was rejected. Check condition syntax and output choices, then review again."
              : "The preview was rejected. Check the selected labels or catalog changes, then review again.",
          );
        else failure(reason);
      }
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  };
  // An uncertain prepare retains its exact key/preview; retry never creates different intent.
  const prepare = async () => {
    if (!preview) return;
    const wasUncertain = uncertain || pendingOutcome.current !== null;
    if (newWorkLocked && !wasUncertain) return;
    if (!wasUncertain && Date.parse(preview.expires_at) <= Date.now()) {
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
        const seed = afterAckSeed.current;
        afterAckSeed.current = null;
        if (onAcknowledged)
          onAcknowledged(
            seed &&
              seed.workspace_id === current.workspace_id &&
              seed.command_id === current.command_id &&
              seed.prepare_key === current.prepare_key
              ? seed
              : undefined,
          );
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

  const selectionControls = (
    <div
      className={styles.selectionControls}
      role="group"
      aria-label="Selected transaction actions"
    >
      <div className={styles.selectionHeading}>
        <strong>{selected.length} selected</strong>
        <button
          type="button"
          className="secondary"
          disabled={blocked || !!preview || reading}
          onClick={() => {
            const first = selected[0];
            setSelected([]);
            if (first)
              rowButtons.current
                .get(selectionKey(first))
                ?.focus({ preventScroll: true });
          }}
        >
          Clear selection
        </button>
      </div>
      <div className={styles.actions}>
        <label>
          <span id={topicChoiceLabel}>Label selected transactions</span>
          <select
            aria-labelledby={topicChoiceLabel}
            value={choice}
            disabled={blocked || !!preview || !selected.length || !canClassify}
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
                <option key={topic.topic_id} value={`topic:${topic.topic_id}`}>
                  {topic.name}
                </option>
              ))}
          </select>
        </label>
        <button
          disabled={
            blocked || !!preview || !selected.length || !choice || !canClassify
          }
          onClick={() => void previewLabels()}
        >
          Preview labels for {selected.length} selected
        </button>
      </div>
      {!!selected.length && (
        <button
          disabled={blocked || !!preview || reading || !canClassify}
          onClick={() => void previewAction({ kind: "rules_apply" }, selected)}
        >
          Preview rules for {selected.length} selected
        </button>
      )}
      {!!selected.length && (
        <TagSelection
          tags={catalog.tags}
          count={selected.length}
          stale={
            page?.items.some(
              (row) =>
                selected.some(
                  (item) => selectionKey(item) === selectionKey(row),
                ) && row.owner_tags.state === "stale",
            ) ?? false
          }
          disabled={blocked || !!preview || reading || !canClassify}
          onPreview={(action) => void previewAction(action, selected)}
        />
      )}
    </div>
  );

  useEffect(() => {
    if (locked || denied) setDetail(null);
  }, [locked, denied]);
  useEffect(() => {
    if (detail || !detailReturn.current || locked || denied) return;
    const button = rowButtons.current.get(detailReturn.current);
    detailReturn.current = null;
    button?.focus({ preventScroll: true });
    button?.scrollIntoView?.({ block: "nearest" });
  }, [detail, page, locked, denied]);
  if (detail && !locked && !denied)
    return (
      <CollectionDetail
        request={detail}
        books={catalog.books}
        catalog={catalog}
        port={port}
        readJournal={readJournal}
        onBack={() => {
          detailReturn.current = selectionKey(detail);
          setDetail(null);
        }}
        onDenied={() => failure(new CollectionFailure("denied"))}
      />
    );
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
      {catalog.automatic_categorization?.state === "available" &&
        port.automation &&
        !command &&
        !preview && (
          <AutomaticCategorization
            workspace={workspace.workspace_id}
            port={port.automation}
            active={active}
            canClassify={canClassify}
            locked={
              newWorkLocked ||
              locked ||
              denied ||
              busy ||
              !ready ||
              !!command ||
              uncertain ||
              reciprocalRefreshPending
            }
            onLockChange={setAutomationLocked}
            onDenied={() => failure(new CollectionFailure("denied"))}
            onChanged={() =>
              onRefreshCatalog ? onRefreshCatalog() : refresh()
            }
            onRun={onAutomaticRun}
            onAttention={() => {
              const next: Selector = {
                ...selector,
                topic: { kind: "needs_attention" },
                status: { ...selector.status, source: "booked" },
              };
              setDraft(next);
              setSelector(next);
              read({
                ...next,
                profile: "private-collection-review-v3",
                workspace_id: workspace.workspace_id,
                limit: 25,
                cursor: null,
                snapshot: null,
              });
            }}
          />
        )}
      {!command && (
        <form className={styles.filters} onSubmit={applyFilters}>
          <fieldset disabled={blocked || reading || !!preview}>
            <div className={styles.primaryFilters}>
              <label className={styles.searchFilter}>
                Search transactions
                <input
                  type="search"
                  placeholder="Search imported descriptions"
                  maxLength={200}
                  value={draft.search}
                  onChange={(event) =>
                    setDraft({ ...draft, search: event.target.value })
                  }
                />
              </label>
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
              <button type="submit">Apply filters</button>
            </div>
            <details className={styles.moreFilters}>
              <summary aria-label="More filters">
                <span className={styles.filterDisclosureTitle}>
                  More filters
                </span>
                <span className={styles.filterDisclosureHint}>
                  Book, topic and date filters
                </span>
              </summary>
              <div className={styles.advancedFilters}>
                <label>
                  <span id={tagFilterLabel}>Tag filter</span>
                  <select
                    aria-labelledby={tagFilterLabel}
                    value={
                      draft.tag.kind === "tag"
                        ? "tag:" + draft.tag.tag_id
                        : draft.tag.kind
                    }
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        tag: event.target.value.startsWith("tag:")
                          ? { kind: "tag", tag_id: event.target.value.slice(4) }
                          : { kind: event.target.value as "all" | "untagged" },
                      })
                    }
                  >
                    <option value="all">All tags</option>
                    <option value="untagged">
                      Untagged or needing confirmation
                    </option>
                    {catalog.tags.map((tag) => (
                      <option key={tag.tag_id} value={"tag:" + tag.tag_id}>
                        {tag.name}
                        {tag.status === "retired" ? " (retired)" : ""}
                      </option>
                    ))}
                  </select>
                </label>

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
                                | "needs_attention"
                                | "uncategorized",
                            },
                      })
                    }
                  >
                    <option value="all">All topics</option>
                    <option value="unclassified">Not classified</option>
                    {catalog.automatic_categorization?.state ===
                      "available" && (
                      <option value="needs_attention">Needs attention</option>
                    )}
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
                <CollectionFilters
                  draft={draft}
                  catalog={catalog}
                  onChange={setDraft}
                />
              </div>
            </details>
            <div className={styles.filterStatus}>
              <p aria-live="polite">
                {filtersChanged
                  ? "Filters changed. Apply to update transactions."
                  : appliedFilters
                    ? `${appliedFilters} ${appliedFilters === 1 ? "filter" : "filters"} applied`
                    : "Showing all authorized transactions"}
              </p>
              {hasFilters && (
                <button
                  type="button"
                  className="secondary"
                  onClick={resetFilters}
                >
                  Reset filters
                </button>
              )}
            </div>
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
          {canClassify && (
            <div className={`${styles.management} ${styles.managementRules}`}>
              <RuleEditor
                catalog={catalog}
                sources={sources}
                disabled={blocked || !!preview || reading}
                seed={
                  ruleSeed?.workspace_id === workspace.workspace_id
                    ? ruleSeed
                    : null
                }
                onSeedConsumed={onSeedConsumed}
                onPreview={(rules) =>
                  void previewAction({ kind: "rule_catalog", rules }, [])
                }
              />
              <TaxonomyEditor
                topics={catalog.topics}
                disabled={blocked || !!preview || reading}
                onPreview={(topics) =>
                  void previewAction({ kind: "taxonomy", topics }, [])
                }
              />
              <TagEditor
                tags={catalog.tags}
                disabled={blocked || !!preview || reading}
                onPreview={(tags) =>
                  void previewAction({ kind: "tag_catalog", tags }, [])
                }
              />
            </div>
          )}
          {!!selected.length && selectionControls}
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
                    <button
                      type="button"
                      className={styles.detailLink}
                      ref={(node) => {
                        if (node)
                          rowButtons.current.set(selectionKey(row), node);
                        else rowButtons.current.delete(selectionKey(row));
                      }}
                      disabled={blocked || !!preview || reading}
                      onClick={() =>
                        setDetail({
                          profile: "private-collection-detail-v1",
                          snapshot: page.snapshot,
                          ...selectionOf(row),
                        })
                      }
                    >
                      {row.description}
                    </button>
                    <CategorizationSummary row={row} />
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
                      <RuleApplicationView
                        application={row.rule_application}
                        catalog={catalog}
                      />
                      <RuleSuggestionView
                        suggestion={row.rule_suggestion}
                        catalog={catalog}
                      />
                      <p>Owner topic: {topicLabel(row)}</p>
                      <p>
                        Strategic tags:{" "}
                        {row.owner_tags.state === "unclassified"
                          ? "Not classified"
                          : row.owner_tags.tags
                              .map(
                                (tag) =>
                                  tag.name +
                                  (tag.status === "retired"
                                    ? " (retired)"
                                    : ""),
                              )
                              .join(", ") || "None"}
                        {row.owner_tags.state === "stale"
                          ? " — needs confirmation"
                          : ""}
                      </p>
                      <p>
                        {row.provider_suggestion.state === "attributed"
                          ? `Provider suggestion: ${row.provider_suggestion.label}; taxonomy ${row.provider_suggestion.taxonomy_version}; reported confidence ${row.provider_suggestion.confidence}.`
                          : "Provider suggestion unavailable."}
                      </p>
                      <p>
                        {row.money.state === "available"
                          ? `Original currency: ${row.money.currency}; ${row.money.numeric_code === null ? "numeric code not supplied" : row.money.numeric_code}; valid from ${row.money.valid_from}${row.money.valid_to ? ` until ${row.money.valid_to}` : ""}.`
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
          {!selected.length && canClassify && (
            <p className={styles.selectionHint}>
              Select transactions to preview topics, tags or rules.
            </p>
          )}
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
        </>
      )}
      {preview && !command && !denied && (
        <div role="region" className={styles.review} aria-label="Label preview">
          <h3 tabIndex={-1} ref={heading}>
            {preview.action.kind === "rule_catalog"
              ? "Review rule changes"
              : preview.action.kind === "rules_apply"
                ? `Review rules for ${preview.selected_count} selected`
                : preview.action.kind === "tag_catalog"
                  ? "Review tag catalog"
                  : preview.action.kind === "tags"
                    ? `Review tags for ${preview.selected_count} selected`
                    : preview.action.kind === "taxonomy"
                      ? "Review topic changes"
                      : `Review ${preview.selected_count} selected transactions`}
          </h3>
          <p>
            {preview.action.kind === "rule_catalog"
              ? "Replace the private rule edition with "
              : preview.action.kind === "rules_apply"
                ? "Review current rule outcomes: "
                : preview.action.kind === "tag_catalog"
                  ? "Replace the strategic tag catalog with "
                  : preview.action.kind === "tags"
                    ? "Change strategic tags: "
                    : preview.action.kind === "taxonomy"
                      ? "Replace the private topic edition with "
                      : "Set owner topic to "}
            <strong>{actionLabel(preview.action, catalog)}</strong>. Source
            facts and journals stay unchanged.
          </p>
          {preview.action.kind === "rule_catalog" ? (
            <RuleCatalogReview rules={preview.action.rules} catalog={catalog} />
          ) : preview.action.kind === "tag_catalog" ? (
            <TagCatalogReview tags={preview.action.tags} />
          ) : preview.action.kind === "taxonomy" ? (
            <TopicReview topics={preview.action.topics} />
          ) : (
            <>
              <ReviewRows rows={preview.review_rows} />
              {preview.action.kind === "rules_apply" && (
                <RuleResultsView
                  results={preview.rule_results}
                  rows={preview.review_rows}
                  catalog={catalog}
                  applyCount={preview.apply_count}
                  abstainCount={preview.abstain_count}
                />
              )}
              {preview.action.kind === "tags" && (
                <TagResults
                  rows={preview.tag_results}
                  reviewRows={preview.review_rows}
                />
              )}
            </>
          )}
          <button
            disabled={busy || uncertain || newWorkLocked}
            onClick={() => void prepare()}
          >
            {preview.action.kind === "rule_catalog"
              ? "Save rule review"
              : preview.action.kind === "rules_apply"
                ? "Save rule application"
                : preview.action.kind === "tag_catalog"
                  ? "Save tag catalog review"
                  : preview.action.kind === "tags"
                    ? "Save tag review"
                    : preview.action.kind === "taxonomy"
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
            {command.preview.action.kind === "rule_catalog"
              ? "Saved rule review"
              : command.preview.action.kind === "rules_apply"
                ? "Saved rule application"
                : command.preview.action.kind === "tag_catalog"
                  ? "Saved tag catalog review"
                  : command.preview.action.kind === "tags"
                    ? "Saved tag review"
                    : command.preview.action.kind === "taxonomy"
                      ? "Saved topic review"
                      : "Saved label review"}
          </h3>
          <p>
            {command.preview.action.kind === "rule_catalog"
              ? "Private rule edition: "
              : command.preview.action.kind === "rules_apply"
                ? "Exact selected rule outcomes: "
                : command.preview.action.kind === "tag_catalog"
                  ? "Strategic tag catalog: "
                  : command.preview.action.kind === "tags"
                    ? `${command.preview.selected_count} selected transactions; strategic tags: `
                    : command.preview.action.kind === "taxonomy"
                      ? "Private topic edition: "
                      : `${command.preview.selected_count} selected transactions; owner topic: `}
            <strong>{actionLabel(command.preview.action, catalog)}</strong>.
          </p>
          {command.preview.action.kind === "rule_catalog" ? (
            <RuleCatalogReview
              rules={command.preview.action.rules}
              catalog={catalog}
            />
          ) : command.preview.action.kind === "tag_catalog" ? (
            <TagCatalogReview tags={command.preview.action.tags} />
          ) : command.preview.action.kind === "taxonomy" ? (
            <TopicReview topics={command.preview.action.topics} />
          ) : (
            <>
              <ReviewRows rows={command.preview.review_rows} />
              {command.preview.profile === "private-collection-preview-v3" &&
                command.preview.action.kind === "rules_apply" && (
                  <RuleResultsView
                    results={command.preview.rule_results}
                    rows={command.preview.review_rows}
                    catalog={catalog}
                    applyCount={command.preview.apply_count}
                    abstainCount={command.preview.abstain_count}
                  />
                )}
              {command.preview.profile !== "private-collection-preview-v1" &&
                command.preview.action.kind === "tags" && (
                  <TagResults
                    rows={command.preview.tag_results}
                    reviewRows={command.preview.review_rows}
                  />
                )}
            </>
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
                {command.preview.action.kind === "rule_catalog"
                  ? "Confirm rule changes"
                  : command.preview.action.kind === "rules_apply"
                    ? `Confirm rules for ${command.preview.selected_count} selected`
                    : command.preview.action.kind === "tag_catalog"
                      ? "Confirm tag catalog changes"
                      : command.preview.action.kind === "tags"
                        ? `Confirm tags for ${command.preview.selected_count} selected`
                        : command.preview.action.kind === "taxonomy"
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
                  ? command.preview.action.kind === "rule_catalog"
                    ? "Rules updated. No journal was created."
                    : command.preview.action.kind === "rules_apply"
                      ? "Rule results applied. No journal was created."
                      : command.preview.action.kind === "tag_catalog"
                        ? "Tag catalog updated. No journal was created."
                        : command.preview.action.kind === "tags"
                          ? "Tags applied. No journal was created."
                          : command.preview.action.kind === "taxonomy"
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
              {correctionRuleSeed(command) && (
                <button
                  className="secondary"
                  disabled={busy || uncertain || locked}
                  onClick={() => {
                    afterAckSeed.current = correctionRuleSeed(command);
                    void transition("acknowledge");
                  }}
                >
                  Acknowledge and create rule
                </button>
              )}
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
  if (action.kind === "rule_catalog") return `${action.rules.length} rules`;
  if (action.kind === "rules_apply") return "Reviewed current rule results";
  if (action.kind === "tag_catalog") return `${action.tags.length} tags`;
  if (action.kind === "tags")
    return `${action.operation === "add" ? "Add" : action.operation === "remove" ? "Remove" : "Replace with"} ${action.tag_ids.length} tags`;
  if (action.kind === "taxonomy") return `${action.topics.length} topics`;
  return action.kind === "uncategorized"
    ? "Explicit Uncategorized"
    : (catalog.topics.find((topic) => topic.topic_id === action.topic_id)
        ?.name ?? "Selected topic");
}
// The saved server projection exposes exact selected values without aggregation or hidden expansion.
function ReviewRows({ rows }: { rows: (Row | LegacyRow)[] }) {
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
function MoneyLabel({ row }: { row: Row | LegacyRow }) {
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

// Name each result from its exact retained selection; IDs stay internal to the mapping.
function TagResults({
  rows,
  reviewRows,
}: {
  rows: TagResult[];
  reviewRows: (Row | LegacyRow)[];
}) {
  const descriptions = new Map(
    reviewRows.map((row) => [
      selectionKey(row),
      `${row.booked_date} · ${row.account_name} · ${row.description}`,
    ]),
  );
  return (
    <ul aria-label="Resulting strategic tags" className={styles.reviewRows}>
      {rows.map((row) => (
        <li key={selectionKey(row)}>
          {descriptions.get(selectionKey(row)) ?? "Transaction unavailable"}:{" "}
          {row.tags
            .map(
              (tag) =>
                tag.name + (tag.status === "retired" ? " (retired)" : ""),
            )
            .join(", ") || "No tags"}
        </li>
      ))}
    </ul>
  );
}
