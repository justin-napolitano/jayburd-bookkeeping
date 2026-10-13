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
  const [report, setReport] = useState<FinancialReport | null>(null);
  const [error, setError] = useState("");
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
  };
  // Refresh omits the revision; continuation preserves the snapshot and opaque cursor unchanged.
  const load = async (next = false) => {
    const previous = report;
    const n = ++epoch.current;
    setReport(null);
    setError("");
    setBusy(true);
    const query = new URLSearchParams({ limit: "100" });
    if (range) {
      query.set("starts_on", start);
      query.set("ends_before", end);
    } else query.set("through_date", through);
    if (kind === "general-ledger") query.set("account_id", account);
    if (next && previous?.next_cursor) {
      query.set("snapshot_revision", previous.snapshot_revision);
      query.set("cursor", previous.next_cursor);
    }
    try {
      const result = await api<FinancialReport>(
        `/v1/entities/${entity.entity_id}/books/${book.book_id}/reports/${kind}?${query}`,
      );
      if (n !== epoch.current) return;
      if (
        result.contract_version !== "financial-reporting-v1" ||
        result.layout !==
          (kind === "balance-sheet"
            ? "basic-unclassified-v1"
            : kind === "general-ledger"
              ? "posting-v1"
              : "account-v1") ||
        result.entity_id !== entity.entity_id ||
        result.book_id !== book.book_id ||
        result.report !== kind ||
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
  }, [api]);
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
          <button disabled={busy}>Refresh report</button>
        </fieldset>
      </form>
      {busy && <p role="status">Reading financial report snapshot…</p>}
      {error && <p role="alert">{error}</p>}
      {report && (
        <>
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
              <tbody>
                {Object.entries(report.totals).map(([key, value]) => (
                  <tr key={key}>
                    <th scope="row">{labels[key] ?? key}</th>
                    <td>{reportAmount(value, report.minor_units)}</td>
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
                    {columns.map((key) => (
                      <th scope="col" key={key}>
                        {labels[key]}
                      </th>
                    ))}
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
                          </>
                        ) : row.id === "unclosed_earnings" ? (
                          "Unclosed earnings"
                        ) : (
                          `${row.account_code} · ${row.account_name}`
                        )}
                      </th>
                      {kind !== "general-ledger" && (
                        <td>{row.category ?? "Derived equity"}</td>
                      )}
                      {columns.map((key) => (
                        <td key={key}>
                          {reportAmount(row.amounts[key], report.minor_units)}
                        </td>
                      ))}
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
