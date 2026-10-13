import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { Reports, reportAmount, type FinancialReport } from "./Reports";
import { APIError, type API } from "./api";
import type { Book, Entity } from "./types";
const book = {
  book_id: "b",
  name: "Operating",
  book_revision: "1",
  policies: [],
  periods: [],
  accounts: [],
} as Book;
const entity = {
  entity_id: "e",
  name: "Synthetic LLC",
  kind: "legal",
  permissions: ["read"],
  books: [book],
} as Entity;
// Fixed values are independent of report arithmetic and deliberately exceed JS safe integers.
const fixture = {
  contract_version: "financial-reporting-v1",
  layout: "account-v1",
  filters: { report: "trial-balance", through_date: "2026-01-31" },
  comparison: null,
  report: "trial-balance",
  entity_id: "e",
  book_id: "b",
  snapshot_revision: "1",
  generated_at: "2026-01-31T00:00:00Z",
  basis: "accrual",
  timezone: "UTC",
  functional_currency: "USD",
  functional_currency_version_id: "c",
  minor_units: 2,
  policy_ids: ["p"],
  next_cursor: "opaque-cursor",
  totals: {
    total_debits: "9007199254740993.01",
    total_credits: "9007199254740993.01",
  },
  rows: [
    {
      id: "account:a",
      selector: {
        entity_id: "e",
        book_id: "b",
        snapshot_revision: "1",
        account_id: "a",
        through_date: "2026-01-31",
      },
      account_id: "a",
      account_code: "1000",
      account_name: "Cash",
      category: "asset",
      amounts: {
        debit_balance: "9007199254740993.01",
        credit_balance: "0",
        net: "9007199254740993.01",
      },
    },
  ],
} as FinancialReport;
afterEach(cleanup);
// Submit explicit dates, never relying on current time or an implicit report period.
function request() {
  fireEvent.change(screen.getByLabelText("Report through date (inclusive)"), {
    target: { value: "2026-01-31" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Refresh report" }));
}
describe("financial report review", () => {
  it("retains exact signed decimals beyond safe integers", () => {
    expect(reportAmount("-9007199254740993.01", 2)).toBe(
      "-9007199254740993.01",
    );
    expect(reportAmount("-0.001", 2)).toBe("-0.001");
    expect(reportAmount("1", 2)).toBe("1.00");
  });
  it("binds pagination revision, clears stale totals, and refreshes a fresh snapshot", async () => {
    const api = vi
      .fn()
      .mockResolvedValueOnce(fixture)
      .mockRejectedValueOnce(new APIError(409, false))
      .mockResolvedValueOnce({ ...fixture, snapshot_revision: "2" });
    render(
      <Reports api={api as API} entity={entity} book={book} locked={false} />,
    );
    request();
    await screen.findByText(/Snapshot 1/);
    expect(screen.getAllByText("9007199254740993.01").length).toBeGreaterThan(
      0,
    );
    fireEvent.click(screen.getByRole("button", { name: "Next report page" }));
    await screen.findByRole("alert");
    expect(api.mock.calls[1][0]).toContain(
      "snapshot_revision=1&cursor=opaque-cursor",
    );
    expect(screen.queryByText(/Snapshot 1/)).toBeNull();
    expect(screen.queryAllByText("9007199254740993.01")).toHaveLength(0);
    fireEvent.click(screen.getByRole("button", { name: "Refresh report" }));
    await screen.findByText(/Snapshot 2/);
    expect(api.mock.calls[2][0]).not.toContain("cursor");
    expect(api.mock.calls[2][0]).not.toContain("snapshot_revision");
  });
  it("replaces page rows while retaining complete totals and the same revision", async () => {
    const second = {
      ...fixture,
      next_cursor: null,
      rows: [
        {
          ...fixture.rows[0],
          id: "account:other",
          account_code: "2000",
          account_name: "Other",
        },
      ],
    };
    const api = vi
      .fn()
      .mockResolvedValueOnce(fixture)
      .mockResolvedValueOnce(second);
    render(
      <Reports api={api as API} entity={entity} book={book} locked={false} />,
    );
    request();
    await screen.findByText("1000 · Cash");
    fireEvent.click(screen.getByRole("button", { name: "Next report page" }));
    await screen.findByText("2000 · Other");
    expect(screen.queryByText("1000 · Cash")).toBeNull();
    expect(screen.getByText(/Page 2\./)).toBeTruthy();
    expect(
      screen.queryByRole("button", { name: "Next report page" }),
    ).toBeNull();
    expect(screen.getAllByText("9007199254740993.01").length).toBeGreaterThan(
      0,
    );
  });
  it("discards delayed data after filters change and after unmount", async () => {
    let resolve!: (value: FinancialReport) => void;
    const api = vi.fn(
      () =>
        new Promise<FinancialReport>((r) => {
          resolve = r;
        }),
    );
    const view = render(
      <Reports api={api as API} entity={entity} book={book} locked={false} />,
    );
    request();
    fireEvent.change(screen.getByLabelText("Report through date (inclusive)"), {
      target: { value: "2026-02-01" },
    });
    resolve(fixture);
    await waitFor(() => expect(screen.queryByText(/Snapshot 1/)).toBeNull());
    fireEvent.click(screen.getByRole("button", { name: "Refresh report" }));
    view.unmount();
    resolve(fixture);
    render(
      <Reports
        api={vi.fn() as API}
        entity={{ ...entity, entity_id: "other" }}
        book={book}
        locked={false}
      />,
    );
    expect(screen.queryByText(/Snapshot 1/)).toBeNull();
  });
  it.each([403, 404, 500])(
    "hides prior data on error %i without leaking backend text",
    async (status) => {
      const api = vi
        .fn()
        .mockResolvedValueOnce(fixture)
        .mockRejectedValueOnce(new APIError(status, false));
      render(
        <Reports api={api as API} entity={entity} book={book} locked={false} />,
      );
      request();
      await screen.findByText(/Snapshot 1/);
      fireEvent.click(screen.getByRole("button", { name: "Refresh report" }));
      await screen.findByRole("alert");
      expect(screen.queryByText(/Snapshot 1/)).toBeNull();
      expect(screen.queryAllByText("9007199254740993.01")).toHaveLength(0);
    },
  );
  it("clears reports on direct scope rerender, client replacement and command locking", async () => {
    const api = vi.fn().mockResolvedValue(fixture) as API;
    const view = render(
      <Reports api={api} entity={entity} book={book} locked={false} />,
    );
    request();
    await screen.findByText(/Snapshot 1/);
    view.rerender(
      <Reports api={api} entity={entity} book={book} locked={true} />,
    );
    expect(screen.queryByText(/Snapshot 1/)).toBeNull();
    view.rerender(
      <Reports api={api} entity={entity} book={book} locked={false} />,
    );
    request();
    await screen.findByText(/Snapshot 1/);
    view.rerender(
      <Reports
        api={api}
        entity={{ ...entity, entity_id: "other" }}
        book={book}
        locked={false}
      />,
    );
    expect(screen.queryByText(/Snapshot 1/)).toBeNull();
    view.rerender(
      <Reports api={api} entity={entity} book={book} locked={false} />,
    );
    request();
    await screen.findByText(/Snapshot 1/);
    view.rerender(
      <Reports
        api={vi.fn() as API}
        entity={entity}
        book={book}
        locked={false}
      />,
    );
    expect(screen.queryByText(/Snapshot 1/)).toBeNull();
  });
  it.each([{ contract_version: "future-v2" }, { layout: "posting-v1" }])(
    "rejects incompatible report contract %j",
    async (mismatch) => {
      render(
        <Reports
          api={vi.fn().mockResolvedValue({ ...fixture, ...mismatch }) as API}
          entity={entity}
          book={book}
          locked={false}
        />,
      );
      request();
      await screen.findByRole("alert");
      expect(screen.queryByText(/Snapshot 1/)).toBeNull();
      expect(screen.queryAllByText("9007199254740993.01")).toHaveLength(0);
    },
  );
  it("rejects a mismatched scope response", async () => {
    render(
      <Reports
        api={vi.fn().mockResolvedValue({ ...fixture, book_id: "other" }) as API}
        entity={entity}
        book={book}
        locked={false}
      />,
    );
    request();
    await screen.findByRole("alert");
    expect(screen.queryByText(/Snapshot 1/)).toBeNull();
  });
});
