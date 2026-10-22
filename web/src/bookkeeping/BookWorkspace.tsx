"use client";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { APIError, type API } from "./api";
import { Connections, type ConnectionLink } from "./Connections";
import { ReviewInbox } from "./ReviewInbox";
import { currencyAt, formatAmount } from "./amount";
import { Reports, type ReportContext, type ReportSelection } from "./Reports";
import type {
  Book,
  Entity,
  TrialBalance,
  Workspace,
  RegisterCommand,
} from "./types";

// Navigation exposes owned book operations; connection state never substitutes for balances.
export type BookView =
  | "Overview"
  | "Accounts"
  | "Account activity"
  | "Reports"
  | "New journal"
  | "Review transactions"
  | "Connections";
const views: BookView[] = [
  "Overview",
  "Accounts",
  "Account activity",
  "Reports",
  "New journal",
  "Review transactions",
  "Connections",
];

// An explicit visible starting range comes from this book's period metadata.
function initialContext(book: Book): ReportContext {
  const period = book.periods[0];
  const end = period?.ends_before ?? "";
  const through = end
    ? new Date(new Date(`${end}T00:00:00Z`).getTime() - 86400000)
        .toISOString()
        .slice(0, 10)
    : new Date().toISOString().slice(0, 10);
  return {
    through,
    start: period?.starts_on ?? book.policies[0]?.effective_from ?? "",
    end,
  };
}

// The selected book owns navigation and all requests; no scope survives a remount.
export function BookWorkspace({
  entity,
  book,
  currencies,
  api,
  locked,
  denied,
  children,
  prepare,
  connectionLink,
  activeView,
  onViewChange,
  through,
  onThroughChange,
  sourceRevision,
  navigationKey,
}: {
  entity: Entity;
  book: Book;
  currencies: Workspace["currencies"];
  api: API;
  locked: boolean;
  denied: () => void;
  children: ReactNode;
  prepare?: (command: RegisterCommand) => void;
  connectionLink?: ConnectionLink;
  activeView?: BookView;
  onViewChange?: (view: BookView) => void;
  through?: string;
  onThroughChange?: (date: string) => void;
  sourceRevision?: string;
  navigationKey?: number;
}) {
  const [reviewSource, setReviewSource] = useState("");
  const lastNavigation = useRef(navigationKey);
  // Generic tool navigation resets a prior connection filter; explicit connection handoffs keep it.
  useEffect(() => {
    if (lastNavigation.current !== navigationKey) {
      lastNavigation.current = navigationKey;
      if (activeView === "Review transactions") setReviewSource("");
    }
  }, [navigationKey, activeView]);
  const [localView, setLocalView] = useState<BookView>("Overview");
  const view = activeView ?? localView;
  // The outer tool bar may own navigation while this book retains its scoped filters.
  const setView = (next: BookView) => {
    setLocalView(next);
    onViewChange?.(next);
  };
  const [context, setContext] = useState<ReportContext>(() => ({
    ...initialContext(book),
    ...(through ? { through } : {}),
  }));
  const [selection, setSelection] = useState<ReportSelection>({
    key: 0,
    kind:
      activeView === "Account activity" ? "general-ledger" : "trial-balance",
    revision: sourceRevision,
  });
  const [report, setReport] = useState<TrialBalance | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const epoch = useRef(0);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    if (through && through !== context.through)
      setContext((current) => ({ ...current, through }));
  }, [through, context.through]);
  useEffect(() => {
    if (activeView) requestAnimationFrame(() => heading.current?.focus());
    if (activeView === "Account activity")
      setSelection((s) =>
        s.kind === "general-ledger"
          ? s
          : { key: s.key + 1, kind: "general-ledger" },
      );
    if (activeView === "Reports")
      setSelection((s) =>
        s.kind === "general-ledger"
          ? { key: s.key + 1, kind: "trial-balance" }
          : s,
      );
  }, [activeView]);
  const deniedRef = useRef(denied);
  deniedRef.current = denied;
  const policy = book.policies.find(
    (p) =>
      p.effective_from <= context.through &&
      (!p.effective_to || context.through < p.effective_to),
  );
  const scale = report
    ? currencyAt(currencies, report.functional_currency, report.through_date)
        ?.minor_unit
    : undefined;
  // Clear totals before a date change and ignore every superseded response.
  const readBalances = useCallback(async () => {
    const n = ++epoch.current;
    setReport(null);
    setError("");
    if (locked || !context.through) {
      setBusy(false);
      return;
    }
    setBusy(true);
    try {
      const result = await api<TrialBalance>(
        `/v1/entities/${entity.entity_id}/books/${book.book_id}/trial-balance?through_date=${encodeURIComponent(context.through)}`,
      );
      if (n !== epoch.current) return;
      if (
        result.entity_id !== entity.entity_id ||
        result.book_id !== book.book_id ||
        result.through_date !== context.through
      )
        throw new Error("scope mismatch");
      setReport(result);
    } catch (failure) {
      if (n !== epoch.current) return;
      setError(
        failure instanceof APIError && [403, 404].includes(failure.status)
          ? "Access to this book is unavailable."
          : "Balances are unavailable. Check the date and refresh.",
      );
      if (failure instanceof APIError && [403, 404].includes(failure.status))
        deniedRef.current();
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  }, [api, entity.entity_id, book.book_id, context.through, locked]);
  useEffect(() => {
    if (view === "Overview" || view === "Accounts") void readBalances();
    else {
      epoch.current++;
      setBusy(false);
    }
    return () => {
      epoch.current++;
    };
  }, [readBalances, view]);
  // Native buttons plus focused section headings make view changes keyboard-visible.
  const navigate = (next: BookView) => {
    if (next === "Review transactions") setReviewSource("");
    setView(next);
    if (next === "Account activity")
      setSelection((s) => ({ key: s.key + 1, kind: "general-ledger" }));
    if (next === "Reports")
      setSelection((s) =>
        s.kind === "general-ledger"
          ? { key: s.key + 1, kind: "trial-balance" }
          : s,
      );
    requestAnimationFrame(() => heading.current?.focus());
  };
  // Activity range edits preserve valid as-of balances; only the balance date cancels its read.
  const changeContext = (next: ReportContext) => {
    if (next.through !== context.through) {
      epoch.current++;
      setReport(null);
      setError("");
      setBusy(false);
    }
    setContext(next);
    onThroughChange?.(next.through);
  };
  // Account drill keeps the exact source revision; SQL rechecks it before showing activity.
  const openAccount = (account: string) => {
    setSelection((s) => ({
      key: s.key + 1,
      kind: "general-ledger",
      account,
      revision: report?.snapshot_revision,
    }));
    setView("Account activity");
    requestAnimationFrame(() => heading.current?.focus());
  };
  return (
    <div className="book-workspace">
      <div className="book-context" aria-label="Current book context">
        <div>
          <p className="eyebrow">{entity.name}</p>
          <h2>{book.name}</h2>
          <p>
            {policy?.functional_currency ??
              "Currency unavailable for this date"}{" "}
            functional · Workspace revision {book.book_revision}
          </p>
        </div>
        {!(
          activeView &&
          (view === "Reports" || view === "Account activity")
        ) && (
          <div className="context-dates">
            <label>
              Through date
              <input
                type="date"
                required
                value={context.through}
                disabled={locked}
                onChange={(e) =>
                  changeContext({ ...context, through: e.target.value })
                }
              />
            </label>
            <label>
              Activity starts on
              <input
                type="date"
                value={context.start}
                disabled={locked}
                onChange={(e) =>
                  changeContext({ ...context, start: e.target.value })
                }
              />
            </label>
            <label>
              Activity ends before
              <input
                type="date"
                value={context.end}
                disabled={locked}
                onChange={(e) =>
                  changeContext({ ...context, end: e.target.value })
                }
              />
            </label>
          </div>
        )}
      </div>
      {!activeView && (
        <nav className="book-nav" aria-label="Book navigation">
          {views.map((name) => (
            <button
              key={name}
              aria-current={view === name ? "page" : undefined}
              disabled={locked}
              onClick={() => navigate(name)}
            >
              {name}
            </button>
          ))}
        </nav>
      )}
      <h2 ref={heading} tabIndex={-1} className="view-title">
        {view === "Account activity" ? "Ledger" : view}
      </h2>
      {(view === "Overview" || view === "Accounts") && (
        <section className="balances" aria-labelledby="balances-title">
          <div className="heading">
            <div>
              <h3 id="balances-title">Account balances</h3>
              <p>
                Posted journals through {context.through || "a selected date"}.
                Each book is reviewed separately.
              </p>
            </div>
            <button
              disabled={busy || locked || !context.through}
              onClick={() => void readBalances()}
            >
              Load trial balance
            </button>
          </div>
          {busy && <p role="status">Reading account balances…</p>}
          {error && <p role="alert">{error}</p>}
          {!report && !busy && !error && (
            <p>Choose a date to read this book’s balances.</p>
          )}
          {report && (
            <>
              <p className="report-stamp">
                {report.functional_currency} · Snapshot{" "}
                {report.snapshot_revision} · {book.accounts.length} configured
                accounts
              </p>
              <div className="table-scroll">
                <table>
                  <caption>Account balances from posted journals</caption>
                  <thead>
                    <tr>
                      <th scope="col">Account</th>
                      <th scope="col">Debit</th>
                      <th scope="col">Credit</th>
                    </tr>
                  </thead>
                  <tbody>
                    {report.accounts.map((a) => (
                      <tr key={a.account_id}>
                        <th scope="row">
                          <button
                            className="account-link"
                            disabled={locked}
                            onClick={() => openAccount(a.account_id)}
                          >
                            {a.account_code}
                            <span>
                              {book.accounts.find(
                                (x) => x.account_id === a.account_id,
                              )?.name ?? "Account"}
                            </span>
                          </button>
                        </th>
                        <td>{formatAmount(a.debit_balance, scale)}</td>
                        <td>{formatAmount(a.credit_balance, scale)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr>
                      <th scope="row">Total</th>
                      <td>{formatAmount(report.total_debits, scale)}</td>
                      <td>{formatAmount(report.total_credits, scale)}</td>
                    </tr>
                  </tfoot>
                </table>
              </div>
              {report.accounts.length === 0 && (
                <p>
                  No accounts appear in this report. This is not a connected
                  bank feed.
                </p>
              )}
            </>
          )}
          {view === "Overview" && (
            <div className="workspace-actions">
              <button disabled={locked} onClick={() => navigate("Reports")}>
                Review financial reports
              </button>
              <p>
                Balance sheet, profit and loss, comparisons and CSV review
                exports.
              </p>
            </div>
          )}
        </section>
      )}
      <div hidden={view !== "Reports" && view !== "Account activity"}>
        <Reports
          api={api}
          entity={entity}
          book={book}
          locked={locked}
          ledgerOnly={view === "Account activity"}
          embedded={!!activeView}
          active={view === "Reports" || view === "Account activity"}
          context={context}
          onContextChange={changeContext}
          selection={selection}
          onAccessDenied={() => deniedRef.current()}
        />
      </div>
      {view === "Connections" && (
        <Connections
          api={api}
          entity={entity}
          book={book}
          currencies={currencies}
          locked={locked}
          denied={denied}
          link={connectionLink}
          onReview={(source) => {
            setReviewSource(source);
            setView("Review transactions");
            requestAnimationFrame(() => heading.current?.focus());
          }}
        />
      )}
      {view === "Review transactions" && prepare && (
        <ReviewInbox
          initialSource={reviewSource}
          api={api}
          entity={entity}
          book={book}
          currencies={currencies}
          locked={locked}
          prepare={prepare}
          denied={denied}
        />
      )}
      <div hidden={view !== "New journal"}>{children}</div>
    </div>
  );
}
