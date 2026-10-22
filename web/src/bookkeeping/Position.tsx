"use client";
import { useEffect, useMemo, useState } from "react";
import { APIError, type API } from "./api";
import { currencyAt, formatAmount } from "./amount";
import type { FinancialReport } from "./Reports";
import type { Book, Entity, Workspace } from "./types";

// Position shows independently authorized book reports; it never aggregates entities or currencies.
export type PositionProps = {
  entities: Entity[];
  currencies: Workspace["currencies"];
  api: API;
  locked: boolean;
  through: string;
  compareThrough: string;
  onOpenBook?: (entity: Entity, book: Book, revision?: string) => void;
};
type Result =
  | { kind: "loading" }
  | { kind: "ready"; report: FinancialReport }
  | {
      kind: "denied" | "stale" | "unavailable";
    };
type BookScope = { entity: Entity; book: Book; key: string };
// Only outstanding promises are shared per authenticated client; no rows or requests are persisted.
const pendingReads = new WeakMap<API, Set<Promise<void>>>();

// Decimal display changes punctuation only; no financial value passes through Number.
function displayAmount(value: string, scale: number, change = false): string {
  if (!/^-?(?:0|[1-9]\d*)(?:\.\d+)?$/.test(value)) return "Unavailable";
  const negative = value.startsWith("-");
  const unsigned = negative ? value.slice(1) : value;
  const formatted = formatAmount(unsigned, scale);
  const [whole, fraction] = formatted.split(".");
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  const sign = negative
    ? "−"
    : change && value !== "0" && !/^0(?:\.0+)?$/.test(value)
      ? "+"
      : "";
  return `${sign}${grouped}${fraction === undefined ? "" : `.${fraction}`}`;
}

// A read is accepted only for the requested book, dates, layout and versioned currency.
function validReport(
  report: FinancialReport,
  scope: BookScope,
  through: string,
  compareThrough: string,
  currencies: Workspace["currencies"],
): boolean {
  const currency = currencyAt(currencies, report.functional_currency, through);
  const policy = scope.book.policies.find(
    (entry) =>
      entry.effective_from <= through &&
      (!entry.effective_to || through < entry.effective_to),
  );
  return (
    report.contract_version === "financial-reporting-v1" &&
    report.report === "balance-sheet" &&
    report.layout === "basic-unclassified-v1" &&
    report.entity_id === scope.entity.entity_id &&
    report.book_id === scope.book.book_id &&
    report.filters.report === "balance-sheet" &&
    report.filters.through_date === through &&
    report.filters.compare_through_date === compareThrough &&
    /^\d+$/.test(report.snapshot_revision) &&
    !!report.generated_at &&
    !!currency &&
    currency.currency_version_id === report.functional_currency_version_id &&
    currency.minor_unit === report.minor_units &&
    policy?.functional_currency === report.functional_currency &&
    policy.functional_currency_version_id ===
      report.functional_currency_version_id &&
    report.policy_ids.includes(policy.policy_id) &&
    !!report.comparison &&
    [
      report.totals.assets,
      report.totals.liabilities,
      report.totals.total_equity,
      report.comparison.difference.total_equity,
    ].every(
      (amount) =>
        typeof amount === "string" &&
        /^-?(?:0|[1-9]\d*)(?:\.\d+)?$/.test(amount),
    )
  );
}

// The request boundary owns concurrency and discards all settled results after scope or client changes.
function PositionRows(
  props: PositionProps & { scopes: BookScope[]; signature: string },
) {
  const {
    api,
    currencies,
    locked,
    through,
    compareThrough,
    scopes,
    signature,
  } = props;
  const [snapshot, setSnapshot] = useState<{
    api: API;
    currencies: Workspace["currencies"];
    signature: string;
    results: Record<string, Result>;
  }>({ api, currencies, signature, results: {} });
  const results =
    snapshot.api === api &&
    snapshot.currencies === currencies &&
    snapshot.signature === signature
      ? snapshot.results
      : {};
  useEffect(() => {
    let active = true;
    setSnapshot({ api, currencies, signature, results: {} });
    if (locked || !through || !compareThrough || compareThrough >= through)
      return () => {
        active = false;
      };
    let cursor = 0;
    let inFlight = pendingReads.get(api);
    if (!inFlight) {
      inFlight = new Set();
      pendingReads.set(api, inFlight);
    }
    const running = inFlight;
    const worker = async () => {
      while (active && cursor < scopes.length) {
        while (active && running.size >= 4) await Promise.race(running);
        if (!active || cursor >= scopes.length) return;
        const scope = scopes[cursor++];
        const query = new URLSearchParams({
          through_date: through,
          compare_through_date: compareThrough,
          limit: "1",
        });
        let result: Result;
        let settled: Promise<void> | undefined;
        try {
          const request = api<FinancialReport>(
            `/v1/entities/${encodeURIComponent(scope.entity.entity_id)}/books/${encodeURIComponent(scope.book.book_id)}/reports/balance-sheet?${query}`,
          );
          settled = request.then(
            () => undefined,
            () => undefined,
          );
          running.add(settled);
          const report = await request;
          result = validReport(
            report,
            scope,
            through,
            compareThrough,
            currencies,
          )
            ? { kind: "ready", report }
            : { kind: "unavailable" };
        } catch (error) {
          result = {
            kind:
              error instanceof APIError &&
              [401, 403, 404].includes(error.status)
                ? "denied"
                : error instanceof APIError && error.status === 409
                  ? "stale"
                  : "unavailable",
          };
        } finally {
          if (settled) running.delete(settled);
        }
        if (active)
          setSnapshot((previous) => ({
            api,
            currencies,
            signature,
            results: {
              ...(previous.api === api &&
              previous.currencies === currencies &&
              previous.signature === signature
                ? previous.results
                : {}),
              [scope.key]: result,
            },
          }));
      }
    };
    for (let index = 0; index < Math.min(4, scopes.length); index++)
      void worker();
    return () => {
      active = false;
    };
  }, [api, currencies, locked, through, compareThrough, scopes, signature]);

  const invalidDate = !through || !compareThrough || compareThrough >= through;
  return (
    <div className="position-table-scroll">
      <table className="position-table">
        <caption>
          Each book is shown separately in its functional currency. Change since{" "}
          {compareThrough || "an earlier date"}.
        </caption>
        <thead>
          <tr>
            <th scope="col">Entity / Book</th>
            <th scope="col">Currency</th>
            <th scope="col">Assets</th>
            <th scope="col">Liabilities</th>
            <th scope="col">Book equity</th>
            <th scope="col">Equity change</th>
          </tr>
        </thead>
        <tbody>
          {props.entities.map((entity) => {
            const books = scopes.filter(
              (scope) => scope.entity.entity_id === entity.entity_id,
            );
            if (!books.length) return null;
            return [
              <tr
                className="position-entity-row"
                key={`${entity.entity_id}/heading`}
              >
                <th colSpan={6} scope="rowgroup">
                  {entity.name}
                </th>
              </tr>,
              ...books.map(({ book, key }, index) => {
                const result = results[key];
                const report = result?.kind === "ready" ? result.report : null;
                const provenanceId = `position-provenance-${entity.entity_id}-${book.book_id}`;
                const state = locked
                  ? "Locked while a command needs acknowledgement"
                  : invalidDate
                    ? "Choose an as-of date and an earlier comparison date"
                    : !result
                      ? "Loading"
                      : result.kind === "denied"
                        ? "Access unavailable"
                        : result.kind === "stale"
                          ? "Book changed — refresh Position"
                          : result.kind === "unavailable"
                            ? "Position unavailable"
                            : "";
                const scale = report?.minor_units ?? 0;
                const change = report?.comparison?.difference.total_equity;
                return (
                  <tr
                    className={`position-book-row${index % 2 === 0 ? " position-row-striped" : ""}`}
                    key={key}
                  >
                    <th scope="row">
                      {props.onOpenBook ? (
                        <button
                          type="button"
                          className="position-book-link"
                          aria-describedby={report ? provenanceId : undefined}
                          disabled={locked}
                          onClick={() =>
                            props.onOpenBook?.(
                              entity,
                              book,
                              report?.snapshot_revision,
                            )
                          }
                        >
                          {book.name}
                        </button>
                      ) : (
                        <span>{book.name}</span>
                      )}
                      {report && (
                        <span className="position-provenance" id={provenanceId}>
                          Snapshot revision {report.snapshot_revision},
                          generated {report.generated_at}
                        </span>
                      )}
                    </th>
                    {report ? (
                      <>
                        <td data-label="Currency">
                          {report.functional_currency}
                        </td>
                        <td data-label="Assets">
                          {displayAmount(report.totals.assets, scale)}
                        </td>
                        <td data-label="Liabilities">
                          {displayAmount(report.totals.liabilities, scale)}
                        </td>
                        <td
                          className="position-equity"
                          data-label="Book equity"
                        >
                          {displayAmount(report.totals.total_equity, scale)}
                        </td>
                        <td
                          data-label="Equity change"
                          className={
                            change?.startsWith("-")
                              ? "amount-negative"
                              : change && !/^0(?:\.0+)?$/.test(change)
                                ? "amount-positive"
                                : undefined
                          }
                        >
                          {displayAmount(change ?? "0", scale, true)}
                        </td>
                      </>
                    ) : (
                      <td
                        className="position-row-state"
                        colSpan={5}
                        role={
                          result?.kind === "denied" ||
                          result?.kind === "stale" ||
                          result?.kind === "unavailable"
                            ? "alert"
                            : "status"
                        }
                      >
                        {state}
                      </td>
                    )}
                  </tr>
                );
              }),
            ];
          })}
        </tbody>
      </table>
    </div>
  );
}

// The keyed reader remounts on a new scope/date/client, so old financial rows cannot flash.
export function Position(props: PositionProps) {
  const scopes = useMemo(
    () =>
      props.entities
        .filter((entity) => entity.permissions.includes("read"))
        .flatMap((entity) =>
          entity.books.map((book) => ({
            entity,
            book,
            key: `${entity.entity_id}/${book.book_id}`,
          })),
        ),
    [props.entities],
  );
  const scopeKey = `${props.through}/${props.compareThrough}/${props.locked}/${scopes.map((scope) => `${scope.key}@${scope.book.book_revision}`).join("|")}`;
  if (scopes.length === 0)
    return (
      <p className="position-empty" role="status">
        No books are available. Choose or create an authorized book.
      </p>
    );
  return <PositionRows {...props} scopes={scopes} signature={scopeKey} />;
}
