import type { FinancialReport } from "../src/bookkeeping/Reports";
import type { Workspace } from "../src/bookkeeping/types";

// Fictional, fixed browser data; the fixture never models accounting or production authorization.
const books = [
  {
    entity: "studio",
    id: "studio-operating",
    name: "Operating",
    assets: "25000",
    liabilities: "2100",
    equity: "22900",
    prior: "18100",
    change: "4800",
  },
  {
    entity: "studio",
    id: "studio-reserve",
    name: "Reserve",
    assets: "8000",
    liabilities: "0",
    equity: "8000",
    prior: "7500",
    change: "500",
  },
  {
    entity: "holdings",
    id: "holdings-operating",
    name: "Operating",
    assets: "42000",
    liabilities: "15000",
    equity: "27000",
    prior: "27600",
    change: "-600",
  },
  {
    entity: "holdings",
    id: "holdings-europe",
    name: "Europe",
    assets: "12000",
    liabilities: "3000",
    equity: "9000",
    prior: "8100",
    change: "900",
  },
  {
    entity: "personal",
    id: "personal-household",
    name: "Household",
    assets: "18000",
    liabilities: "4500",
    equity: "13500",
    prior: "13750",
    change: "-250",
  },
] as const;
export type PositionBookID = (typeof books)[number]["id"];

// All five books use the currently supported synthetic USD functional policy.
export const positionWorkspace: Workspace = {
  development_policy: "dev-accrual-usd-v1",
  currencies: [
    {
      currency_version_id: "usd-v1",
      alphabetic_code: "USD",
      minor_unit: 2,
      valid_from: "2026-01-01",
      valid_to: "2027-01-01",
    },
  ],
  entities: [
    {
      entity_id: "studio",
      name: "Fictional Studio LLC",
      kind: "legal",
      permissions: ["read"],
      books: [],
    },
    {
      entity_id: "holdings",
      name: "Fictional Holdings LLC",
      kind: "legal",
      permissions: ["read"],
      books: [],
    },
    {
      entity_id: "personal",
      name: "Personal",
      kind: "personal",
      permissions: ["read"],
      books: [],
    },
  ],
};
for (const item of books) {
  positionWorkspace.entities
    .find((entity) => entity.entity_id === item.entity)!
    .books.push({
      book_id: item.id,
      name: item.name,
      book_revision: "12",
      policies: [
        {
          policy_id: "policy-usd",
          functional_currency: "USD",
          functional_currency_version_id: "usd-v1",
          effective_from: "2026-01-01",
          effective_to: null,
        },
      ],
      periods: [
        {
          period_id: "period-2026",
          starts_on: "2026-01-01",
          ends_before: "2027-01-01",
          state: "open",
        },
      ],
      accounts: [
        {
          account_id: `account-${item.id}`,
          code: "1000",
          name: "Fictional asset",
          category: "asset",
          normal_side: "debit",
          monetary: true,
          active_from: "2026-01-01",
          active_to: null,
        },
      ],
    });
}

// Server-shaped values are constants; date variants prove the UI invalidates old reads.
export function positionReport(
  bookID: string,
  through = "2026-10-21",
  compareThrough = "2026-09-30",
): FinancialReport {
  const item = books.find((candidate) => candidate.id === bookID);
  if (!item) throw new Error("Unknown fictional book");
  const earlier = through === "2026-10-20";
  return {
    contract_version: "financial-reporting-v1",
    report: "balance-sheet",
    layout: "basic-unclassified-v1",
    entity_id: item.entity,
    book_id: item.id,
    snapshot_revision: "12",
    generated_at: "2026-10-21T15:00:00Z",
    basis: "accrual",
    timezone: "UTC",
    functional_currency: "USD",
    functional_currency_version_id: "usd-v1",
    minor_units: 2,
    policy_ids: ["policy-usd"],
    filters: {
      report: "balance-sheet",
      through_date: through,
      compare_through_date: compareThrough,
    },
    totals: {
      assets: earlier ? "100" : item.assets,
      liabilities: earlier ? "0" : item.liabilities,
      total_equity: earlier ? "100" : item.equity,
    },
    comparison: {
      totals: { total_equity: earlier ? "50" : item.prior },
      difference: { total_equity: earlier ? "50" : item.change },
    },
    rows: [],
    next_cursor: null,
  };
}
