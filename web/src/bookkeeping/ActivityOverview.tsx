"use client";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { APIError, type API } from "./api";
import { formatAmount } from "./amount";
import { accountLabels } from "./accountLabels";
import type { Book, Currency, Entity } from "./types";
import type { components } from "../lib/api.generated";

// Closed generated projections keep source quantities and private selectors on their reviewed wire.
type Page = components["schemas"]["ActivityPage"];
type Query = components["schemas"]["ActivityQuery"];
type Unit = components["schemas"]["ActivityUnit"];
type Merchant = components["schemas"]["ActivityMerchantSelector"];
type Source = components["schemas"]["ReviewSource"];
// Applied calendar dates are explicit and independent of the browser's wall clock.
type Range = { starts_on: string; through_date: string };
// A drill is memory-only and destroyed with its scope or headline snapshot.
type Detail = { query: Query; page: Page; title: string };
const allHistory: Range = {
  starts_on: "0001-01-01",
  through_date: "9999-12-31",
};
const exact = /^-?[0-9]{1,48}(?:\.[0-9]{1,9})?$/;

// Formatting preserves every decimal digit; the browser never aggregates or values money.
function money(value: string, minor: number): string {
  const formatted = formatAmount(value.replace(/^-/, ""), minor);
  const [whole, fraction] = formatted.split(".");
  return (
    (value.startsWith("-") ? "−" : "") +
    whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",") +
    (fraction === undefined ? "" : `.${fraction}`)
  );
}
// Revision pins are consistency metadata, never authorization or a bank-sync timestamp.
function sameSnapshot(a: Page["snapshot"], b: Page["snapshot"]): boolean {
  return (
    !!a &&
    !!b &&
    a.source_revision === b.source_revision &&
    a.binding_revision === b.binding_revision &&
    a.review_revision_sum === b.review_revision_sum &&
    a.filter_digest === b.filter_digest
  );
}
// Verify transport context and exact strings before rendering any financial response.
function matches(page: Page, q: Query, entity: string, book: string): boolean {
  const s = page?.selector;
  return (
    page?.profile === "source-activity-v1" &&
    page.entity_id === entity &&
    page.book_id === book &&
    page.source_account_id === q.source_account_id &&
    !!page.snapshot &&
    !!s &&
    s.profile === q.profile &&
    s.source_account_id === q.source_account_id &&
    s.starts_on === q.starts_on &&
    s.through_date === q.through_date &&
    s.search === q.search &&
    s.kind === q.kind &&
    s.currency_version_id === q.currency_version_id &&
    (s.merchant?.status ?? null) === (q.merchant?.status ?? null) &&
    (s.merchant?.label ?? null) === (q.merchant?.label ?? null) &&
    (!q.snapshot || sameSnapshot(page.snapshot, q.snapshot)) &&
    Array.isArray(page.units) &&
    Array.isArray(page.items) &&
    Array.isArray(page.merchant_groups) &&
    !!page.inventory &&
    page.units.every(
      (u) =>
        typeof u.inflow === "string" &&
        exact.test(u.inflow) &&
        typeof u.outflow === "string" &&
        exact.test(u.outflow) &&
        typeof u.net_inflow === "string" &&
        exact.test(u.net_inflow),
    ) &&
    page.items.every(
      (row) =>
        row.currency_version_id === q.currency_version_id &&
        typeof row.amount === "string" &&
        exact.test(row.amount),
    ) &&
    page.merchant_groups.every(
      (g) =>
        typeof g.inflow === "string" &&
        exact.test(g.inflow) &&
        typeof g.outflow === "string" &&
        exact.test(g.outflow),
    ) &&
    (!page.totals ||
      [page.totals.inflow, page.totals.outflow, page.totals.net_inflow].every(
        (v) => typeof v === "string" && exact.test(v),
      ))
  );
}
// Scope/lock changes unmount every financial snapshot and unfinished request generation.
export function ActivityOverview(props: {
  api: API;
  entity: Entity;
  book: Book;
  currencies: Currency[];
  locked: boolean;
  initialSource?: string;
  onSourceChoice?: (id: string) => void;
  onReview: (id: string) => void;
  onAccounts: () => void;
  denied: () => void;
}) {
  return (
    <Overview
      key={`${props.entity.entity_id}/${props.book.book_id}/${props.book.book_revision}/${props.locked}`}
      {...props}
    />
  );
}
// One source and explicit applied period own the headline and every exact server drill.
function Overview({
  api,
  entity,
  book,
  currencies,
  locked,
  initialSource = "",
  onSourceChoice,
  onReview,
  onAccounts,
  denied,
}: Parameters<typeof ActivityOverview>[0]) {
  const base = `/v1/entities/${encodeURIComponent(entity.entity_id)}/books/${encodeURIComponent(book.book_id)}`;
  const [sources, setSources] = useState<Source[]>([]),
    [source, setSource] = useState("");
  const [loadingSources, setLoadingSources] = useState(!locked),
    [sourceError, setSourceError] = useState("");
  const [applied, setApplied] = useState<Range>(allHistory),
    [mode, setMode] = useState("all"),
    [draft, setDraft] = useState<Range>({ starts_on: "", through_date: "" });
  const [summary, setSummary] = useState<Page | null>(null),
    [detail, setDetail] = useState<Detail | null>(null),
    [back, setBack] = useState<{ query: Query; title: string } | null>(null);
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const epoch = useRef(0),
    sourceEpoch = useRef(0),
    deniedRef = useRef(denied),
    detailHeading = useRef<HTMLHeadingElement>(null),
    summaryHeading = useRef<HTMLHeadingElement>(null);
  deniedRef.current = denied;
  const choiceRef = useRef(onSourceChoice);
  choiceRef.current = onSourceChoice;
  const labels = accountLabels(sources.map((s) => s.source_account_id));
  const selected = sources.find((s) => s.source_account_id === source);
  // Source discovery uses current server authority and never restores an unavailable account.
  const loadSources = useCallback(async () => {
    const n = ++sourceEpoch.current;
    setSourceError("");
    setLoadingSources(true);
    if (locked) {
      setLoadingSources(false);
      return;
    }
    try {
      const result = await api<components["schemas"]["Sources"]>(
        `${base}/sources`,
      );
      if (n !== sourceEpoch.current) return;
      if (
        !Array.isArray(result.items) ||
        new Set(result.items.map((s) => s.source_account_id)).size !==
          result.items.length
      )
        throw Error("invalid context");
      setSources(result.items);
      const remembered = result.items.find(
        (s) => s.source_account_id === initialSource,
      );
      setSource(
        remembered?.source_account_id ??
          (result.items.length === 1 ? result.items[0].source_account_id : ""),
      );
    } catch (failure) {
      if (n !== sourceEpoch.current) return;
      setSources([]);
      setSourceError("Accounts are unavailable. Retry loading accounts.");
      if (failure instanceof APIError && [403, 404].includes(failure.status))
        deniedRef.current();
    } finally {
      if (n === sourceEpoch.current) setLoadingSources(false);
    }
  }, [api, base, initialSource, locked]);
  useEffect(() => {
    void loadSources();
    return () => {
      sourceEpoch.current++;
    };
  }, [loadSources]);
  // Safe failures destroy totals and rows; an unavailable read never becomes a zero or a command outcome.
  const failed = (failure: unknown) => {
    setSummary(null);
    setDetail(null);
    setBack(null);
    setError(
      failure instanceof APIError && failure.status === 409
        ? "Transactions changed. Refresh the overview before opening a total."
        : "Account activity is unavailable. Refresh the overview to try again.",
    );
    if (failure instanceof APIError && [403, 404].includes(failure.status))
      deniedRef.current();
  };
  const readSummary = useCallback(async () => {
    const n = ++epoch.current;
    setSummary(null);
    setDetail(null);
    setBack(null);
    setError("");
    setBusy(false);
    if (locked || !source) return;
    setBusy(true);
    const q: Query = {
      profile: "source-activity-v1",
      source_account_id: source,
      ...applied,
      search: "",
      kind: "summary",
      currency_version_id: null,
      merchant: null,
      snapshot: null,
      cursor: null,
      limit: 20,
    };
    try {
      if (!api.query) throw Error("read unavailable");
      const p = await api.query<Page>(
        `${base}/source-activity/query`,
        JSON.stringify(q),
      );
      if (n !== epoch.current) return;
      if (!matches(p, q, entity.entity_id, book.book_id))
        throw Error("invalid context");
      setSummary(p);
    } catch (failure) {
      if (n === epoch.current) failed(failure);
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  }, [api, applied, base, book.book_id, entity.entity_id, locked, source]);
  useEffect(() => {
    void readSummary();
    return () => {
      epoch.current++;
    };
  }, [readSummary]);
  // Selector changes clear the old headline synchronously, before a new account name can appear beside it.
  const chooseSource = (id: string) => {
    if (id === source) return;
    epoch.current++;
    setSummary(null);
    setDetail(null);
    setBack(null);
    setError("");
    setSource(id);
    choiceRef.current?.(id);
  };
  const applyPeriod = (event: FormEvent) => {
    event.preventDefault();
    if (
      mode === "custom" &&
      (!draft.starts_on ||
        !draft.through_date ||
        draft.starts_on > draft.through_date)
    ) {
      setError("Choose an ordered start and end date.");
      return;
    }
    epoch.current++;
    setSummary(null);
    setDetail(null);
    setApplied(mode === "all" ? { ...allHistory } : { ...draft });
  };
  // Detail reads retain the exact headline pin, unit and merchant; every continuation reauthorizes.
  const readDetail = async (
    q: Query,
    title: string,
    parent: { query: Query; title: string } | null = null,
  ) => {
    const n = ++epoch.current;
    setDetail(null);
    setError("");
    setBusy(true);
    try {
      if (!api.query) throw Error("read unavailable");
      const p = await api.query<Page>(
        `${base}/source-activity/query`,
        JSON.stringify(q),
      );
      if (n !== epoch.current) return;
      if (!matches(p, q, entity.entity_id, book.book_id))
        throw Error("invalid context");
      setDetail({ query: q, page: p, title });
      setBack(parent);
      requestAnimationFrame(() => {
        if (n === epoch.current) detailHeading.current?.focus();
      });
    } catch (failure) {
      if (n === epoch.current) failed(failure);
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  };
  const open = (
    unit: Unit,
    kind: "observations" | "merchants",
    merchant: Merchant | null = null,
  ) => {
    if (!summary || locked) return;
    const q: Query = {
      profile: "source-activity-v1",
      source_account_id: source,
      ...applied,
      search: "",
      kind,
      currency_version_id: unit.currency_version_id,
      merchant,
      snapshot: summary.snapshot,
      cursor: null,
      limit: 20,
    };
    const parent =
      detail?.query.kind === "merchants"
        ? { query: detail.query, title: detail.title }
        : null;
    void readDetail(
      q,
      merchant
        ? (merchant.label ??
            (merchant.status === "not_provided"
              ? "Merchant not provided"
              : "Merchant unavailable"))
        : kind === "merchants"
          ? `${unit.currency} merchant activity`
          : `${unit.currency} transactions`,
      parent,
    );
  };
  const unitLabel = (unit: Unit) => {
    const meta = currencies.find(
      (c) => c.currency_version_id === unit.currency_version_id,
    );
    const index =
      (summary?.units
        .filter((u) => u.currency === unit.currency)
        .findIndex((u) => u.currency_version_id === unit.currency_version_id) ??
        0) + 1;
    return meta
      ? `${unit.currency} · reference from ${meta.valid_from}`
      : `${unit.currency} · currency reference ${index}`;
  };
  const detailUnit = summary?.units.find(
    (u) => u.currency_version_id === detail?.query.currency_version_id,
  );
  if (locked)
    return (
      <p role="status">Finish the pending action to review account activity.</p>
    );
  return (
    <section
      className="activity-overview"
      aria-label="Account overview"
      aria-busy={busy || loadingSources}
    >
      <div className="activity-toolbar">
        <label>
          Bank account
          <select
            value={source}
            disabled={loadingSources}
            onChange={(e) => chooseSource(e.target.value)}
          >
            <option value="">Choose an account</option>
            {sources.map((s) => (
              <option key={s.source_account_id} value={s.source_account_id}>
                {s.display_name || labels(s.source_account_id)} · {s.currency}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          className="secondary"
          disabled={busy || !source}
          onClick={() => void readSummary()}
        >
          Refresh overview
        </button>
        <button type="button" className="secondary" onClick={onAccounts}>
          Manage bank accounts
        </button>
      </div>
      {loadingSources && <p role="status">Loading accounts…</p>}
      {sourceError && (
        <div role="alert">
          <p>{sourceError}</p>
          <button onClick={() => void loadSources()}>Retry accounts</button>
        </div>
      )}
      {!loadingSources && !sourceError && sources.length === 0 && (
        <p>
          No connected accounts are available in this book. Manage bank accounts
          to register one.
        </p>
      )}
      {!loadingSources && sources.length > 1 && !source && (
        <p>Choose an account to view its imported activity.</p>
      )}
      {selected && (
        <>
          <form className="activity-period" onSubmit={applyPeriod}>
            <label>
              Period
              <select
                value={mode}
                onChange={(e) => {
                  setMode(e.target.value);
                  if (e.target.value === "custom")
                    setDraft({
                      starts_on:
                        summary?.available_from ??
                        book.periods[0]?.starts_on ??
                        "",
                      through_date: summary?.available_through ?? "",
                    });
                }}
              >
                <option value="all">All imported history</option>
                <option value="custom">Custom dates</option>
              </select>
            </label>
            {mode === "custom" && (
              <>
                <label>
                  From
                  <input
                    type="date"
                    required
                    min="0001-01-01"
                    max="9999-12-31"
                    value={draft.starts_on}
                    onChange={(e) =>
                      setDraft({ ...draft, starts_on: e.target.value })
                    }
                  />
                </label>
                <label>
                  Through
                  <input
                    type="date"
                    required
                    min={draft.starts_on || "0001-01-01"}
                    max="9999-12-31"
                    value={draft.through_date}
                    onChange={(e) =>
                      setDraft({ ...draft, through_date: e.target.value })
                    }
                  />
                </label>
              </>
            )}
            <button type="submit" disabled={busy}>
              Apply period
            </button>
          </form>
          <p className="activity-applied">
            Showing{" "}
            {applied.starts_on === allHistory.starts_on &&
            applied.through_date === allHistory.through_date
              ? "all imported history"
              : `${applied.starts_on} through ${applied.through_date}`}
            . Date edits apply when you choose Apply period.
          </p>
        </>
      )}
      {busy && <p role="status">Reading account activity…</p>}
      {error && <p role="alert">{error}</p>}
      {summary && (
        <>
          <div className="activity-heading">
            <h2 tabIndex={-1} ref={summaryHeading}>
              Money in and out
            </h2>
            <button onClick={() => onReview(source)}>
              Review account transactions
            </button>
          </div>
          <p>
            Account {summary.source_state} in this book. Observed history:{" "}
            {summary.available_from && summary.available_through
              ? `${summary.available_from} through ${summary.available_through}`
              : "No imported dates yet"}
            .
          </p>
          {summary.inventory.partial && (
            <p className="activity-attention" role="status">
              {summary.inventory.quality_hold_count} transactions are excluded
              for review. These totals cover eligible imported activity.
            </p>
          )}
          {summary.units.length === 0 && (
            <p>No eligible transactions match this period.</p>
          )}
          <ul className="activity-units">
            {summary.units.map((unit) => (
              <li key={unit.currency_version_id}>
                <div className="activity-unit-name">
                  <h3>{unitLabel(unit)}</h3>
                  <p>{unit.count} eligible transactions</p>
                </div>
                <dl className="activity-money">
                  <div>
                    <dt>Money in</dt>
                    <dd>{money(unit.inflow, unit.minor_unit)}</dd>
                  </div>
                  <div>
                    <dt>Money out</dt>
                    <dd>{money(unit.outflow, unit.minor_unit)}</dd>
                  </div>
                  <div>
                    <dt>Net movement</dt>
                    <dd>{money(unit.net_inflow, unit.minor_unit)}</dd>
                  </div>
                </dl>
                <div className="activity-unit-actions">
                  <button
                    className="secondary"
                    aria-label={`View ${unitLabel(unit)} transactions`}
                    disabled={busy}
                    onClick={() => open(unit, "observations")}
                  >
                    View {unit.currency} transactions
                  </button>
                  <button
                    className="secondary"
                    aria-label={`View ${unitLabel(unit)} merchants`}
                    disabled={busy}
                    onClick={() => open(unit, "merchants")}
                  >
                    View {unit.currency} merchants
                  </button>
                </div>
              </li>
            ))}
          </ul>
          <p className="activity-unknown">
            Bank balance unavailable. Import time unavailable. Imported movement
            includes transfers and card payments; available history does not
            establish complete bank history.
          </p>
          <dl className="activity-status">
            <div>
              <dt>Pending</dt>
              <dd>{summary.inventory.pending_count}</dd>
            </div>
            <div>
              <dt>Removed</dt>
              <dd>{summary.inventory.removed_count}</dd>
            </div>
            <div>
              <dt>Corrections flagged</dt>
              <dd>{summary.inventory.correction_attention_count}</dd>
            </div>
            <div>
              <dt>Zero-value records</dt>
              <dd>{summary.inventory.zero_count}</dd>
            </div>
          </dl>
        </>
      )}
      {detail && detailUnit && (
        <section
          className="activity-detail"
          aria-label="Matching account activity"
        >
          <div className="activity-heading">
            <h2 tabIndex={-1} ref={detailHeading}>
              {detail.title}
            </h2>
            <button
              className="secondary"
              disabled={busy}
              onClick={() => {
                if (back) void readDetail(back.query, back.title);
                else {
                  setDetail(null);
                  setBack(null);
                  requestAnimationFrame(() => summaryHeading.current?.focus());
                }
              }}
            >
              {back ? "Back to merchants" : "Close details"}
            </button>
          </div>
          <p>{unitLabel(detailUnit)} · same account and applied period.</p>
          {detail.page.totals && (
            <p className="activity-drill-totals">
              {detail.page.totals.count} matching transactions · money in{" "}
              {money(detail.page.totals.inflow, detailUnit.minor_unit)} · money
              out {money(detail.page.totals.outflow, detailUnit.minor_unit)}.
              Totals cover every matching row.
            </p>
          )}
          {detail.query.kind === "merchants" ? (
            <>
              <ul className="activity-merchants">
                {detail.page.merchant_groups.map((g) => (
                  <li key={JSON.stringify([g.status, g.label])}>
                    <div>
                      <strong>
                        {g.label ??
                          (g.status === "not_provided"
                            ? "Merchant not provided"
                            : "Merchant unavailable")}
                      </strong>
                      <p>
                        {g.count} transactions · money in{" "}
                        {money(g.inflow, detailUnit.minor_unit)} · money out{" "}
                        {money(g.outflow, detailUnit.minor_unit)}
                      </p>
                    </div>
                    <button
                      className="secondary"
                      onClick={() =>
                        open(detailUnit, "observations", {
                          status: g.status,
                          label: g.label,
                        })
                      }
                    >
                      View transactions
                    </button>
                  </li>
                ))}
              </ul>
              <p>
                {detail.page.group_count} merchant groups ·{" "}
                {detail.page.remaining_group_count} remaining after this page.
              </p>
            </>
          ) : (
            <>
              <div className="table-scroll">
                <table>
                  <caption>Exact matching transactions, oldest first</caption>
                  <thead>
                    <tr>
                      <th className="activity-date">Date</th>
                      <th>Description</th>
                      <th className="activity-direction">Direction</th>
                      <th className="activity-amount">Original amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {detail.page.items.map((row) => (
                      <tr key={row.observation_id}>
                        <td className="activity-date">{row.booked_date}</td>
                        <td>{row.description}</td>
                        <td className="activity-direction">
                          {row.direction === "inflow"
                            ? "Money in"
                            : "Money out"}
                        </td>
                        <td className="activity-amount">
                          <span className="activity-mobile-direction">
                            {row.direction === "inflow"
                              ? "Money in"
                              : "Money out"}
                          </span>
                          <span className="activity-exact-amount">
                            {money(row.amount, detailUnit.minor_unit)}{" "}
                            {row.currency}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {detail.page.items.length === 0 && (
                <p>No transactions match this exact selection.</p>
              )}
            </>
          )}
          {detail.page.next_cursor ? (
            <button
              disabled={busy}
              onClick={() =>
                void readDetail(
                  { ...detail.query, cursor: detail.page.next_cursor },
                  detail.title,
                  back,
                )
              }
            >
              Next page
            </button>
          ) : (
            <p>
              All matching{" "}
              {detail.query.kind === "merchants"
                ? "merchant groups"
                : "transactions"}{" "}
              have been shown.
            </p>
          )}
        </section>
      )}
    </section>
  );
}
