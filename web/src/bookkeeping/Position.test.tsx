import React from "react";
import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { Position } from "./Position";
import { APIError, type API } from "./api";
import type { FinancialReport } from "./Reports";
import type { Entity, Workspace } from "./types";

afterEach(cleanup);

const currencies = [
  {
    currency_version_id: "usd-v1",
    alphabetic_code: "USD",
    minor_unit: 2,
    valid_from: "2000-01-01",
    valid_to: null,
  },
  {
    currency_version_id: "eur-v1",
    alphabetic_code: "EUR",
    minor_unit: 2,
    valid_from: "2000-01-01",
    valid_to: null,
  },
] as Workspace["currencies"];
const entities = [
  {
    entity_id: "e1",
    name: "Fictional Studio LLC",
    kind: "legal",
    permissions: ["read"],
    books: [
      {
        book_id: "b1",
        name: "Operating",
        book_revision: "12",
        policies: [
          {
            policy_id: "p1",
            functional_currency: "USD",
            functional_currency_version_id: "usd-v1",
            effective_from: "2000-01-01",
            effective_to: null,
          },
        ],
        periods: [],
        accounts: [],
      },
      {
        book_id: "b2",
        name: "Reserve",
        book_revision: "3",
        policies: [
          {
            policy_id: "p2",
            functional_currency: "EUR",
            functional_currency_version_id: "eur-v1",
            effective_from: "2000-01-01",
            effective_to: null,
          },
        ],
        periods: [],
        accounts: [],
      },
    ],
  },
] as Entity[];
const through = "2026-10-21",
  compareThrough = "2026-09-30";

// Fixed server-shaped totals exercise display and scope checks without deriving expected math.
function report(
  book: string,
  date = through,
  equity = "22900.00",
  currency = "USD",
): FinancialReport {
  return {
    contract_version: "financial-reporting-v1",
    report: "balance-sheet",
    layout: "basic-unclassified-v1",
    entity_id: "e1",
    book_id: book,
    snapshot_revision: "12",
    generated_at: "2026-10-21T14:00:00Z",
    basis: "accrual",
    timezone: "UTC",
    functional_currency: currency,
    functional_currency_version_id: currency === "USD" ? "usd-v1" : "eur-v1",
    minor_units: 2,
    policy_ids: [book === "b1" ? "p1" : "p2"],
    filters: {
      report: "balance-sheet",
      through_date: date,
      compare_through_date: compareThrough,
    },
    totals: {
      assets: "25000.00",
      liabilities: "2100.00",
      total_equity: equity,
    },
    comparison: {
      totals: { total_equity: "18100.00" },
      difference: { total_equity: "4800.00" },
    },
    rows: [],
    next_cursor: null,
  };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((ready) => {
    resolve = ready;
  });
  return { promise, resolve };
}

it("shows each book and currency separately with exact display strings and revision evidence", async () => {
  const api = vi.fn((path: string) =>
    Promise.resolve(
      path.includes("/b1/")
        ? report("b1", through, "9007199254740993.01")
        : report("b2", through, "9000.00", "EUR"),
    ),
  ) as unknown as API;
  const open = vi.fn();
  render(
    <Position
      entities={entities}
      currencies={currencies}
      api={api}
      locked={false}
      through={through}
      compareThrough={compareThrough}
      onOpenBook={open}
    />,
  );
  await screen.findByText("9,007,199,254,740,993.01");
  const rows = screen.getAllByRole("row");
  expect(rows).toHaveLength(4);
  expect(within(rows[3]).getByText("EUR")).toBeDefined();
  expect(screen.getAllByText("+4,800.00")).toHaveLength(2);
  expect(screen.getAllByText(/Snapshot revision 12, generated/)).toHaveLength(
    2,
  );
  expect(
    screen
      .getByRole("button", { name: "Operating" })
      .getAttribute("aria-describedby"),
  ).toBe("position-provenance-e1-b1");
  expect(api).toHaveBeenCalledTimes(2);
  expect(String(vi.mocked(api).mock.calls[0][0])).toContain(
    "compare_through_date=2026-09-30",
  );
  fireEvent.click(screen.getByRole("button", { name: "Reserve" }));
  expect(open).toHaveBeenCalledWith(entities[0], entities[0].books[1], "12");
});

it("clears prior amounts on date change and discards a late old-date response", async () => {
  const old = deferred<FinancialReport>();
  const api = vi.fn((path: string) =>
    path.includes("through_date=2026-10-20")
      ? Promise.resolve(report("b1", "2026-10-20", "7.00"))
      : old.promise,
  ) as unknown as API;
  const one = [{ ...entities[0], books: [entities[0].books[0]] }];
  const view = render(
    <Position
      entities={one}
      currencies={currencies}
      api={api}
      locked={false}
      through={through}
      compareThrough={compareThrough}
    />,
  );
  view.rerender(
    <Position
      entities={one}
      currencies={currencies}
      api={api}
      locked={false}
      through="2026-10-20"
      compareThrough={compareThrough}
    />,
  );
  expect(screen.queryByText("22,900.00")).toBeNull();
  await screen.findByText("7.00");
  old.resolve(report("b1"));
  await waitFor(() => expect(screen.queryByText("22,900.00")).toBeNull());
});

it("keeps one healthy book visible when another is denied, and distinguishes lock from zero", async () => {
  const api = vi.fn((path: string) =>
    path.includes("/b1/")
      ? Promise.resolve(report("b1"))
      : Promise.reject(new APIError(403, false)),
  ) as unknown as API;
  const view = render(
    <Position
      entities={entities}
      currencies={currencies}
      api={api}
      locked={false}
      through={through}
      compareThrough={compareThrough}
    />,
  );
  await screen.findByText("22,900.00");
  await screen.findByText("Access unavailable");
  expect(screen.queryByText("0.00")).toBeNull();
  view.rerender(
    <Position
      entities={entities}
      currencies={currencies}
      api={api}
      locked={true}
      through={through}
      compareThrough={compareThrough}
      onOpenBook={vi.fn()}
    />,
  );
  expect(screen.queryByText("22,900.00")).toBeNull();
  expect(
    screen.getAllByText("Locked while a command needs acknowledgement"),
  ).toHaveLength(2);
  expect(
    (screen.getByRole("button", { name: "Operating" }) as HTMLButtonElement)
      .disabled,
  ).toBe(true);
});

it("treats mismatched server scope as unavailable and handles no authorized books", async () => {
  const api = vi.fn().mockResolvedValue(report("foreign")) as unknown as API;
  const one = [{ ...entities[0], books: [entities[0].books[0]] }];
  const view = render(
    <Position
      entities={one}
      currencies={currencies}
      api={api}
      locked={false}
      through={through}
      compareThrough={compareThrough}
    />,
  );
  await screen.findByText("Position unavailable");
  expect(screen.queryByText("22,900.00")).toBeNull();
  view.rerender(
    <Position
      entities={[]}
      currencies={currencies}
      api={api}
      locked={false}
      through={through}
      compareThrough={compareThrough}
    />,
  );
  expect(screen.getByText(/No books are available/)).toBeDefined();
});

it("shows a valid zero as a value while another book remains unavailable", async () => {
  const zero = report("b1", through, "0");
  zero.totals = { assets: "0", liabilities: "0", total_equity: "0" };
  zero.comparison = {
    totals: { total_equity: "0" },
    difference: { total_equity: "0" },
  };
  const api = vi.fn((path: string) =>
    path.includes("/b1/")
      ? Promise.resolve(zero)
      : Promise.reject(new APIError(503, true)),
  ) as unknown as API;
  render(
    <Position
      entities={entities}
      currencies={currencies}
      api={api}
      locked={false}
      through={through}
      compareThrough={compareThrough}
    />,
  );
  await screen.findByText("Position unavailable");
  const zeroRow = screen.getByRole("row", { name: /Operating/ });
  expect(within(zeroRow).getAllByText("0.00")).toHaveLength(4);
  expect(screen.queryByText("+0.00")).toBeNull();
});

it("discards old-client results when the authenticated client changes", async () => {
  const pending = deferred<FinancialReport>();
  const oldAPI = vi.fn().mockReturnValue(pending.promise) as unknown as API;
  const newAPI = vi
    .fn()
    .mockResolvedValue(report("b1", through, "19.00")) as unknown as API;
  const one = [{ ...entities[0], books: [entities[0].books[0]] }];
  const view = render(
    <Position
      entities={one}
      currencies={currencies}
      api={oldAPI}
      locked={false}
      through={through}
      compareThrough={compareThrough}
    />,
  );
  view.rerender(
    <Position
      entities={one}
      currencies={currencies}
      api={newAPI}
      locked={false}
      through={through}
      compareThrough={compareThrough}
    />,
  );
  await screen.findByText("19.00");
  pending.resolve(report("b1", through, "999.00"));
  await waitFor(() => expect(screen.queryByText("999.00")).toBeNull());
});

it("keeps at most four book reads in flight within one Position request", async () => {
  const books = Array.from({ length: 6 }, (_, index) => ({
    ...entities[0].books[0],
    book_id: `b${index + 1}`,
    name: `Book ${index + 1}`,
  }));
  const many = [{ ...entities[0], books }];
  const pending = Array.from({ length: 6 }, () => deferred<FinancialReport>());
  let calls = 0;
  const api = vi.fn(() => pending[calls++].promise) as unknown as API;
  render(
    <Position
      entities={many}
      currencies={currencies}
      api={api}
      locked={false}
      through={through}
      compareThrough={compareThrough}
    />,
  );
  expect(api).toHaveBeenCalledTimes(4);
  pending[0].resolve(report("b1"));
  await waitFor(() => expect(api).toHaveBeenCalledTimes(5));
  pending[1].resolve(report("b2"));
  await waitFor(() => expect(api).toHaveBeenCalledTimes(6));
  expect(api).toHaveBeenCalledTimes(6);
});

it("keeps the read bound across date changes while superseded requests settle", async () => {
  const many = [
    {
      ...entities[0],
      books: Array.from({ length: 4 }, (_, i) => ({
        ...entities[0].books[0],
        book_id: `b${i + 1}`,
        name: `Book ${i + 1}`,
      })),
    },
  ];
  const old = Array.from({ length: 4 }, () => deferred<FinancialReport>());
  let calls = 0;
  const api = vi.fn((path: string) => {
    if (calls++ < 4) return old[calls - 1].promise;
    const id = path.split("/")[5];
    return Promise.resolve(report(id, "2026-10-20", "19.00"));
  }) as unknown as API;
  const props = {
    entities: many,
    currencies,
    api,
    locked: false,
    through,
    compareThrough,
  };
  const view = render(<Position {...props} />);
  expect(api).toHaveBeenCalledTimes(4);
  view.rerender(<Position {...props} through="2026-10-20" />);
  expect(api).toHaveBeenCalledTimes(4);
  old[0].resolve(report("b1", through, "999.00"));
  await waitFor(() => expect(api).toHaveBeenCalledTimes(8));
  expect(screen.queryByText("999.00")).toBeNull();
  for (let i = 1; i < 4; i++)
    old[i].resolve(report(`b${i + 1}`, through, "999.00"));
});
