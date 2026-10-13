"use client";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { APIError, type API } from "./api";
import { formatAmount } from "./amount";
import type { Book, Entity } from "./types";
import type { components } from "../lib/api.generated";

// Internal UI selection; dates remain explicit and SQL owns accounting semantics.
export type ReportKind =
  | "trial-balance"
  | "general-ledger"
  | "balance-sheet"
  | "profit-and-loss";
// The reviewed HTTP schema owns report payloads; no local financial shape is duplicated.
export type FinancialReport = components["schemas"]["FinancialReport"];
// Navigation preserves the source revision; point reports require an explicit bounded range.
type AccountNavigation = {
  account: string;
  revision: string;
  start: string;
  end: string;
};
// The pinned journal envelope is owned by the reviewed OpenAPI schema.
type JournalDetail = components["schemas"]["ReportJournal"];
const names: Record<ReportKind, string> = {
  "trial-balance": "Trial balance",
  "general-ledger": "General ledger",
  "balance-sheet": "Balance sheet",
  "profit-and-loss": "Profit and loss",
};
const labels: Record<string, string> = {
  total_debits: "Total debits",
  total_credits: "Total credits",
  opening: "Opening balance",
  period_debits: "Period debits",
  period_credits: "Period credits",
  closing: "Closing balance",
  revenue: "Revenue",
  expense: "Expenses",
  net_income: "Net income",
  assets: "Assets",
  liabilities: "Liabilities",
  direct_equity: "Direct equity",
  unclosed_earnings: "Unclosed earnings",
  total_equity: "Total equity",
  debit_balance: "Debit balance",
  credit_balance: "Credit balance",
  net: "Net (debit positive)",
  debit: "Debit",
  credit: "Credit",
  running_balance: "Running balance",
  balance: "Balance",
};
// Signed amounts are formatted using strings only, with no rounding or numeric coercion.
export function reportAmount(value: string | undefined, scale: number): string {
  if (value === undefined) return "Unavailable";
  return value.startsWith("-")
    ? `-${formatAmount(value.slice(1), scale)}`
    : formatAmount(value, scale);
}
// Remount on scope, revision or command-lock changes so prior financial state cannot survive.
export function Reports(props: {
  api: API;
  entity: Entity;
  book: Book;
  locked: boolean;
}) {
  return (
    <ReportContent
      key={`${props.entity.entity_id}/${props.book.book_id}/${props.book.book_revision}/${props.locked}`}
      {...props}
    />
  );
}
// A scope owns every request; epochs discard late responses after edits or unmounts.
function ReportContent({
  api,
  entity,
  book,
  locked,
}: {
  api: API;
  entity: Entity;
  book: Book;
  locked: boolean;
}) {
  const [kind, setKind] = useState<ReportKind>("trial-balance");
  const [through, setThrough] = useState("");
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [account, setAccount] = useState("");
  const [compare, setCompare] = useState(false);
  const [compareThrough, setCompareThrough] = useState("");
  const [compareStart, setCompareStart] = useState("");
  const [compareEnd, setCompareEnd] = useState("");
  const [report, setReport] = useState<FinancialReport | null>(null);
  const [error, setError] = useState("");
  const [navigation, setNavigation] = useState<AccountNavigation | null>(null);
  const [detail, setDetail] = useState<JournalDetail | null>(null);
  const [actionError, setActionError] = useState("");
  const [actionBusy, setActionBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [page, setPage] = useState(0);
  const epoch = useRef(0);
  useEffect(
    () => () => {
      epoch.current++;
    },
    [],
  );
  const range = kind === "general-ledger" || kind === "profit-and-loss";
  // Invalidate before changing filters, hiding old totals even while a request is pending.
  const clear = () => {
    epoch.current++;
    setReport(null);
    setError("");
    setBusy(false);
    setPage(0);
    setNavigation(null);
    setDetail(null);
    setActionError("");
    setActionBusy(false);
  };
  // Refresh omits the revision; continuation preserves the snapshot and opaque cursor unchanged.
  const load = async (next = false, destination?: AccountNavigation) => {
    const requestedKind = destination ? "general-ledger" : kind;
    if (destination) {
      setKind("general-ledger");
      setStart(destination.start);
      setEnd(destination.end);
      setAccount(destination.account);
      setCompare(false);
    }
    setNavigation(null);
    setDetail(null);
    setActionError("");
    setActionBusy(false);
    const previous = report;
    const n = ++epoch.current;
    setReport(null);
    setError("");
    setBusy(true);
    const query = new URLSearchParams({ limit: "100" });
    if (destination) {
      query.set("starts_on", destination.start);
      query.set("ends_before", destination.end);
      query.set("account_id", destination.account);
      query.set("snapshot_revision", destination.revision);
    } else if (range) {
      query.set("starts_on", start);
      query.set("ends_before", end);
    } else query.set("through_date", through);
    if (!destination && kind === "general-ledger")
      query.set("account_id", account);
    if (!destination && compare && kind !== "general-ledger") {
      if (range) {
        query.set("compare_starts_on", compareStart);
        query.set("compare_ends_before", compareEnd);
      } else query.set("compare_through_date", compareThrough);
    }
    if (next && previous?.next_cursor) {
      query.set("snapshot_revision", previous.snapshot_revision);
      query.set("cursor", previous.next_cursor);
    }
    try {
      const result = await api<FinancialReport>(
        `/v1/entities/${entity.entity_id}/books/${book.book_id}/reports/${requestedKind}?${query}`,
      );
      if (n !== epoch.current) return;
      if (
        result.contract_version !== "financial-reporting-v1" ||
        result.layout !==
          (requestedKind === "balance-sheet"
            ? "basic-unclassified-v1"
            : requestedKind === "general-ledger"
              ? "posting-v1"
              : "account-v1") ||
        result.entity_id !== entity.entity_id ||
        result.book_id !== book.book_id ||
        result.report !== requestedKind ||
        (destination && result.snapshot_revision !== destination.revision) ||
        (next && result.snapshot_revision !== previous?.snapshot_revision)
      ) {
        throw new Error("scope mismatch");
      }
      setReport(result);
      setPage(next ? page + 1 : 1);
    } catch (e) {
      if (n !== epoch.current) return;
      setError(
        e instanceof APIError && e.status === 409
          ? "The book changed. Refresh report to start a new snapshot."
          : e instanceof APIError && [401, 403, 404].includes(e.status)
            ? "Report access is unavailable. Refresh your session or choose an authorized book."
            : "Report unavailable. Check the date range and policy, then refresh report.",
      );
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  };
  // A replacement authenticated client invalidates in-flight requests and visible data.
  useEffect(() => {
    epoch.current++;
    setReport(null);
    setError("");
    setBusy(false);
    setPage(0);
    setNavigation(null);
    setDetail(null);
    setActionError("");
    setActionBusy(false);
  }, [api]);
  // Any stale or denied follow-up invalidates all visible financial state.
  const failAction = (failure: unknown) => {
    if (
      failure instanceof APIError &&
      [401, 403, 404, 409].includes(failure.status)
    ) {
      clear();
      setError(
        failure.status === 409
          ? "The book changed. Refresh report to start a new snapshot."
          : "Report access is unavailable. Refresh your session or choose an authorized book.",
      );
    } else
      setActionError(
        "This review action is unavailable. Try again or refresh report.",
      );
  };
  // Fetch the same authorized revision and actual GL selection, never the standalone journal route.
  const inspectJournal = async (row: FinancialReport["rows"][number]) => {
    if (!report || !row.journal_id || !row.line_number) return;
    const n = ++epoch.current;
    setActionBusy(true);
    setActionError("");
    setDetail(null);
    const query = new URLSearchParams({
      account_id: account,
      starts_on: start,
      ends_before: end,
      line_number: String(row.line_number),
      snapshot_revision: report.snapshot_revision,
    });
    try {
      const result = await api<JournalDetail>(
        `/v1/entities/${entity.entity_id}/books/${book.book_id}/reports/general-ledger/journals/${row.journal_id}?${query}`,
      );
      if (n !== epoch.current) return;
      if (
        result.contract_version !== "report-journal-v1" ||
        result.entity_id !== entity.entity_id ||
        result.book_id !== book.book_id ||
        result.snapshot_revision !== report.snapshot_revision ||
        result.selector.account_id !== account ||
        result.selector.starts_on !== start ||
        result.selector.ends_before !== end ||
        result.selector.journal_id !== row.journal_id ||
        result.selector.line_number !== row.line_number ||
        result.journal.journal_id !== row.journal_id ||
        result.journal.entity_id !== entity.entity_id ||
        result.journal.book_id !== book.book_id ||
        result.journal.functional_currency_version_id !==
          report.functional_currency_version_id
      )
        throw new Error("scope");
      setDetail(result);
    } catch (failure) {
      if (n === epoch.current) failAction(failure);
    } finally {
      if (n === epoch.current) setActionBusy(false);
    }
  };
  // Explicit user download is memory-only and never places authentication in a URL.
  const download = async () => {
    if (!report || !api.download) return;
    const n = ++epoch.current;
    setActionBusy(true);
    setActionError("");
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(report.filters))
      if (key !== "report" && value !== undefined) query.set(key, value);
    query.set("snapshot_revision", report.snapshot_revision);
    try {
      const blob = await api.download(
        `/v1/entities/${entity.entity_id}/books/${book.book_id}/reports/${kind}/export.csv?${query}`,
      );
      if (n !== epoch.current) return;
      const href = URL.createObjectURL(blob);
      try {
        const link = document.createElement("a");
        link.href = href;
        link.download = "financial-reporting-review.csv";
        link.click();
      } finally {
        URL.revokeObjectURL(href);
      }
    } catch (failure) {
      if (n === epoch.current) failAction(failure);
    } finally {
      if (n === epoch.current) setActionBusy(false);
    }
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    void load();
  };
  const columns =
    kind === "trial-balance"
      ? ["debit_balance", "credit_balance", "net"]
      : kind === "general-ledger"
        ? ["debit", "credit", "running_balance"]
        : ["balance"];
  return (
    <section aria-labelledby="financial-reports-title">
      <h2 id="financial-reports-title">Financial reports</h2>
      <p>
        {entity.name} / {book.name}. Each report covers this book only.
      </p>
      <form onSubmit={submit}>
        <fieldset disabled={locked}>
          <div className="grid">
            <label>
              Report type
              <select
                value={kind}
                onChange={(e) => {
                  clear();
                  setKind(e.target.value as ReportKind);
                  setCompare(false);
                }}
              >
                {Object.entries(names).map(([value, name]) => (
                  <option key={value} value={value}>
                    {name}
                  </option>
                ))}
              </select>
            </label>
            {range ? (
              <>
                <label>
                  Report starts on
                  <input
                    type="date"
                    required
                    value={start}
                    onChange={(e) => {
                      clear();
                      setStart(e.target.value);
                    }}
                  />
                </label>
                <label>
                  Report ends before
                  <input
                    type="date"
                    required
                    value={end}
                    onChange={(e) => {
                      clear();
                      setEnd(e.target.value);
                    }}
                  />
                </label>
              </>
            ) : (
              <label>
                Report through date (inclusive)
                <input
                  type="date"
                  required
                  value={through}
                  onChange={(e) => {
                    clear();
                    setThrough(e.target.value);
                  }}
                />
              </label>
            )}
            {kind === "general-ledger" && (
              <label>
                Report account
                <select
                  required
                  value={account}
                  onChange={(e) => {
                    clear();
                    setAccount(e.target.value);
                  }}
                >
                  <option value="">Choose account</option>
                  {book.accounts.map((a) => (
                    <option key={a.account_id} value={a.account_id}>
                      {a.code} · {a.name}
                    </option>
                  ))}
                </select>
              </label>
            )}
          </div>
          {kind !== "general-ledger" && (
            <>
              <label>
                <input
                  type="checkbox"
                  checked={compare}
                  onChange={(e) => {
                    clear();
                    setCompare(e.target.checked);
                  }}
                />
                Compare another period
              </label>
              {compare &&
                (range ? (
                  <div className="grid">
                    <label>
                      Comparison starts on
                      <input
                        type="date"
                        required
                        value={compareStart}
                        onChange={(e) => {
                          clear();
                          setCompareStart(e.target.value);
                        }}
                      />
                    </label>
                    <label>
                      Comparison ends before
                      <input
                        type="date"
                        required
                        value={compareEnd}
                        onChange={(e) => {
                          clear();
                          setCompareEnd(e.target.value);
                        }}
                      />
                    </label>
                  </div>
                ) : (
                  <label>
                    Comparison through date (inclusive)
                    <input
                      type="date"
                      required
                      value={compareThrough}
                      onChange={(e) => {
                        clear();
                        setCompareThrough(e.target.value);
                      }}
                    />
                  </label>
                ))}
            </>
          )}
          <button disabled={busy}>Refresh report</button>
        </fieldset>
      </form>
      {busy && <p role="status">Reading financial report snapshot…</p>}
      {error && <p role="alert">{error}</p>}
      {actionBusy && <p role="status">Preparing report review…</p>}
      {actionError && <p role="alert">{actionError}</p>}
      {report && (
        <>
          {api.download && (
            <button
              disabled={busy || locked || actionBusy}
              onClick={() => void download()}
            >
              Download review CSV
            </button>
          )}
          {navigation && (
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void load(false, navigation);
              }}
            >
              <p>
                Choose a general ledger range of at most 366 days. The opening
                balance includes earlier history; this period's postings alone
                may not explain the full point-in-time balance. Snapshot{" "}
                {navigation.revision} remains pinned.
              </p>
              <label>
                Drill starts on
                <input
                  required
                  type="date"
                  value={navigation.start}
                  onChange={(event) =>
                    setNavigation({ ...navigation, start: event.target.value })
                  }
                />
              </label>
              <label>
                Drill ends before
                <input
                  required
                  type="date"
                  value={navigation.end}
                  onChange={(event) =>
                    setNavigation({ ...navigation, end: event.target.value })
                  }
                />
              </label>
              <button disabled={locked || busy || actionBusy}>
                Open account ledger
              </button>
              <button type="button" onClick={() => setNavigation(null)}>
                Cancel account review
              </button>
            </form>
          )}
          {detail && (
            <section aria-label="Journal detail">
              <h3>Journal {detail.journal.journal_id}</h3>
              <p>
                Report snapshot {detail.snapshot_revision}. Journal posting
                revision {detail.journal.book_revision}.
              </p>
              <p>
                {detail.journal.accounting_date} · {detail.journal.kind} ·
                Policy {detail.journal.policy_id}
              </p>
              <p>{detail.journal.description}</p>
              <table>
                <caption>
                  Journal original units and booked functional amounts
                </caption>
                <thead>
                  <tr>
                    <th>Line</th>
                    <th>Account</th>
                    <th>Side</th>
                    <th>Original</th>
                    <th>Functional</th>
                    <th>Valuation</th>
                  </tr>
                </thead>
                <tbody>
                  {detail.journal.lines.map((line) => (
                    <tr key={line.line_number}>
                      <td>{line.line_number}</td>
                      <td>{line.account_id}</td>
                      <td>{line.side}</td>
                      <td>
                        {line.transaction_amount} {line.transaction_currency} ·
                        Currency version {line.transaction_currency_version_id}
                      </td>
                      <td>
                        {reportAmount(
                          line.functional_amount,
                          report.minor_units,
                        )}{" "}
                        {report.functional_currency}
                      </td>
                      <td>
                        {line.valuation_basis}
                        {line.rate_observation_id &&
                          ` · Rate ${line.rate_observation_id}`}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <button onClick={() => setDetail(null)}>
                Close journal detail
              </button>
            </section>
          )}
          <p>
            Snapshot {report.snapshot_revision} · Generated{" "}
            {report.generated_at} · {report.basis} · {report.timezone}
          </p>
          <p>
            Functional currency: {report.functional_currency}.{" "}
            {range
              ? `${start} through the day before ${end}`
              : `As of ${through}`}
            .
          </p>
          {report.comparison && (
            <p>
              Comparison:{" "}
              {range
                ? `${compareStart} through the day before ${compareEnd}`
                : `As of ${compareThrough}`}
              . Difference is current minus comparison; all columns use this
              snapshot.
            </p>
          )}
          {kind === "balance-sheet" && (
            <p>
              Unclassified balance sheet. Unclosed earnings are cumulative; they
              are not period profit.
            </p>
          )}
          <div className="table-scroll">
            <table>
              <caption>
                {names[kind]} totals · complete selection ·{" "}
                {report.functional_currency}
              </caption>
              {report.comparison && (
                <thead>
                  <tr>
                    <th scope="col">Measure</th>
                    <th scope="col">Current</th>
                    <th scope="col">Comparison</th>
                    <th scope="col">Difference</th>
                  </tr>
                </thead>
              )}
              <tbody>
                {Object.entries(report.totals).map(([key, value]) => (
                  <tr key={key}>
                    <th scope="row">{labels[key] ?? key}</th>
                    <td>{reportAmount(value, report.minor_units)}</td>
                    {report.comparison && (
                      <>
                        <td>
                          {reportAmount(
                            report.comparison.totals[key],
                            report.minor_units,
                          )}
                        </td>
                        <td>
                          {reportAmount(
                            report.comparison.difference[key],
                            report.minor_units,
                          )}
                        </td>
                      </>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {report.rows.length === 0 ? (
            <p>
              No rows in this selection. Opening balances and totals above
              remain authoritative.
            </p>
          ) : (
            <div className="table-scroll">
              <table>
                <caption>
                  {names[kind]} · page {page} · functional amounts in{" "}
                  {report.functional_currency}
                </caption>
                <thead>
                  <tr>
                    <th scope="col">
                      {kind === "general-ledger" ? "Posting" : "Account"}
                    </th>
                    {kind !== "general-ledger" && <th scope="col">Category</th>}
                    {columns.flatMap((key) =>
                      report.comparison
                        ? [
                            <th scope="col" key={`${key}-current`}>
                              {labels[key]} · current
                            </th>,
                            <th scope="col" key={`${key}-prior`}>
                              {labels[key]} · comparison
                            </th>,
                            <th scope="col" key={`${key}-difference`}>
                              {labels[key]} · difference
                            </th>,
                          ]
                        : [
                            <th scope="col" key={key}>
                              {labels[key]}
                            </th>,
                          ],
                    )}
                    {kind === "general-ledger" && (
                      <th scope="col">Original units and provenance</th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {report.rows.map((row) => (
                    <tr key={row.id}>
                      <th scope="row">
                        {kind === "general-ledger" ? (
                          <>
                            {row.accounting_date}
                            <br />
                            Journal {row.journal_id} · line {row.line_number}
                            <button
                              disabled={busy || locked || actionBusy}
                              onClick={() => void inspectJournal(row)}
                            >
                              Review journal line {row.line_number}
                            </button>
                          </>
                        ) : row.id === "unclosed_earnings" ? (
                          "Unclosed earnings"
                        ) : (
                          <>
                            <span>
                              {row.account_code} · {row.account_name}
                            </span>
                            {row.account_id && (
                              <button
                                disabled={busy || locked || actionBusy}
                                onClick={() => {
                                  setDetail(null);
                                  setActionError("");
                                  setNavigation({
                                    account: row.account_id!,
                                    revision: report.snapshot_revision,
                                    start:
                                      kind === "profit-and-loss" ? start : "",
                                    end: kind === "profit-and-loss" ? end : "",
                                  });
                                }}
                              >
                                Review account {row.account_code}
                              </button>
                            )}
                          </>
                        )}
                      </th>
                      {kind !== "general-ledger" && (
                        <td>{row.category ?? "Derived equity"}</td>
                      )}
                      {columns.flatMap((key) =>
                        report.comparison
                          ? [
                              <td key={`${key}-current`}>
                                {reportAmount(
                                  row.amounts[key],
                                  report.minor_units,
                                )}
                              </td>,
                              <td key={`${key}-prior`}>
                                {reportAmount(
                                  row.comparison?.amounts[key],
                                  report.minor_units,
                                )}
                              </td>,
                              <td key={`${key}-difference`}>
                                {reportAmount(
                                  row.comparison?.difference[key],
                                  report.minor_units,
                                )}
                              </td>,
                            ]
                          : [
                              <td key={key}>
                                {reportAmount(
                                  row.amounts[key],
                                  report.minor_units,
                                )}
                              </td>,
                            ],
                      )}
                      {kind === "general-ledger" && (
                        <td>
                          {row.transaction_amount} {row.transaction_currency}
                          <br />
                          {row.valuation_basis} · {row.kind}
                          <br />
                          Policy {row.policy_id}
                          {row.rate_observation_id && (
                            <> · Rate {row.rate_observation_id}</>
                          )}
                          {row.reverses_journal_id && (
                            <> · Reverses {row.reverses_journal_id}</>
                          )}
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <p>
            Page {page}. Totals cover the complete selection, not only this
            page.
          </p>
          {report.next_cursor && (
            <button disabled={busy || locked} onClick={() => void load(true)}>
              Next report page
            </button>
          )}
          <details>
            <summary>Report policy provenance</summary>
            <p>Currency version {report.functional_currency_version_id}</p>
            <ul>
              {report.policy_ids.map((id) => (
                <li key={id}>{id}</li>
              ))}
            </ul>
          </details>
        </>
      )}
    </section>
  );
}
