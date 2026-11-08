import { useMemo, useState } from "react";
import { CollectionReview } from "../src/collection/CollectionReview";
import type { Journal } from "../src/bookkeeping/types";
import {
  collectionCatalog,
  collectionFixture,
  collectionRow,
  collectionSources,
  collectionWorkspace,
} from "./collection-fixture";
import type { JournalReader } from "../src/collection/journal";
import {
  id,
  filterBook,
  filterCurrency,
} from "./collection-filter-detail-fixture";
const catalog = {
  ...collectionCatalog,
  books: [filterBook],
  currencies: [filterCurrency],
};

// Independent browser-only data demonstrates transport intent and focus, never SQL matching authority.
export function CollectionFiltersDetailHarness() {
  const [query, setQuery] = useState(""),
    [details, setDetails] = useState(0),
    [journals, setJournals] = useState(0);
  const fixture = useMemo(() => {
    const rows = Array.from({ length: 30 }, (_, i) =>
      collectionRow(i, {
        journal_state: "posted",
        assignment: {
          state: "assigned",
          books: [{ book_id: filterBook.book_id, name: filterBook.name }],
        },
      }),
    );
    const base = collectionFixture({ rows });
    return {
      ...base,
      port: {
        ...base.port,
        query: async (request: Parameters<typeof base.port.query>[0]) => {
          setQuery(JSON.stringify(request));
          return base.port.query(request);
        },
        detail: async (request: Parameters<typeof base.port.detail>[0]) => {
          setDetails((n) => n + 1);
          const result = await base.port.detail(request);
          return {
            ...result,
            links: {
              state: "available" as const,
              journals: [
                { ...filterBook, journal_id: id(4), source_revision: "1" },
              ],
            },
          };
        },
      },
    };
  }, []);
  const readJournal = useMemo<JournalReader>(
    () => async (identity) => {
      setJournals((n) => n + 1);
      return {
        ...identity,
        period_id: id(5),
        policy_id: id(6),
        posted_by: id(7),
        book_revision: "2",
        functional_currency: "USD",
        functional_currency_version_id: filterCurrency.currency_version_id,
        accounting_date: "2026-10-01",
        description: "Fictional original journal",
        kind: "manual",
        posted_at: "2026-10-01T12:00:00Z",
        reverses_journal_id: null,
        reason: null,
        lines: [
          {
            account_id: id(8),
            line_number: 1,
            side: "debit",
            transaction_amount: "9007199254740993.01",
            transaction_currency: "EUR",
            transaction_currency_version_id: id(10),
            functional_amount: "9907919180215092.311",
            valuation_basis: "spot",
          },
          {
            account_id: id(9),
            line_number: 2,
            side: "credit",
            transaction_amount: "9007199254740993.01",
            transaction_currency: "EUR",
            transaction_currency_version_id: id(10),
            functional_amount: "9907919180215092.311",
            valuation_basis: "spot",
          },
        ],
      } satisfies Journal;
    },
    [],
  );
  return (
    <main>
      <h1>Fictional filters and detail</h1>
      <CollectionReview
        workspace={collectionWorkspace}
        sources={collectionSources}
        catalog={catalog}
        port={fixture.port}
        readJournal={readJournal}
        onDenied={() => {}}
      />
      <output
        style={{ display: "block", overflowWrap: "anywhere" }}
        aria-label="Last fictional query"
      >
        {query}
      </output>
      <output aria-label="Fictional detail reads">{details}</output>
      <output aria-label="Fictional journal reads">{journals}</output>
    </main>
  );
}
