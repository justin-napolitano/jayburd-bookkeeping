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
  accounts: [{ account_id: "a", code: "1000", name: "Cash" }],
} as unknown as Book;
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

// Comparison fixtures come directly from the wire; the UI never calculates deltas.
describe("comparison review", () => {
  it("renders exact server comparisons and retains filters across pages", async () => {
    const compared = {
      ...fixture,
      comparison: {
        totals: {
          total_debits: "9007199254740993.02",
          total_credits: "9007199254740993.02",
        },
        difference: { total_debits: "-0.01", total_credits: "-0.01" },
      },
      rows: [
        {
          ...fixture.rows[0],
          comparison: {
            amounts: {
              debit_balance: "9007199254740993.02",
              credit_balance: "0",
              net: "9007199254740993.02",
            },
            difference: {
              debit_balance: "-0.01",
              credit_balance: "0",
              net: "-0.01",
            },
          },
        },
      ],
    };
    const api = vi.fn().mockResolvedValue(compared);
    render(
      <Reports api={api as API} entity={entity} book={book} locked={false} />,
    );
    fireEvent.click(screen.getByLabelText("Compare another period"));
    fireEvent.change(
      screen.getByLabelText("Comparison through date (inclusive)"),
      { target: { value: "2026-01-30" } },
    );
    request();
    await screen.findByText(/Snapshot 1/);
    expect(screen.getAllByText("-0.01")).toHaveLength(4);
    expect(screen.getAllByText("9007199254740993.02")).toHaveLength(4);
    fireEvent.click(screen.getByRole("button", { name: "Next report page" }));
    await screen.findByText(/Page 2\./);
    expect(api.mock.calls[1][0]).toContain("compare_through_date=2026-01-30");
    expect(api.mock.calls[1][0]).toContain("snapshot_revision=1");
    fireEvent.change(
      screen.getByLabelText("Comparison through date (inclusive)"),
      { target: { value: "2026-01-29" } },
    );
    expect(screen.queryByText(/Snapshot 1/)).toBeNull();
    expect(screen.queryAllByText("-0.01")).toHaveLength(0);
  });
  it("sends paired P&L comparison dates and excludes comparison when changing to GL", async () => {
    const api = vi
      .fn()
      .mockResolvedValue({ ...fixture, report: "profit-and-loss", rows: [] });
    render(
      <Reports api={api as API} entity={entity} book={book} locked={false} />,
    );
    fireEvent.change(screen.getByLabelText("Report type"), {
      target: { value: "profit-and-loss" },
    });
    fireEvent.change(screen.getByLabelText("Report starts on"), {
      target: { value: "2026-02-01" },
    });
    fireEvent.change(screen.getByLabelText("Report ends before"), {
      target: { value: "2026-03-01" },
    });
    fireEvent.click(screen.getByLabelText("Compare another period"));
    fireEvent.change(screen.getByLabelText("Comparison starts on"), {
      target: { value: "2026-01-01" },
    });
    fireEvent.change(screen.getByLabelText("Comparison ends before"), {
      target: { value: "2026-02-01" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Refresh report" }));
    await screen.findByText(/Snapshot 1/);
    expect(api.mock.calls[0][0]).toContain(
      "compare_starts_on=2026-01-01&compare_ends_before=2026-02-01",
    );
    fireEvent.change(screen.getByLabelText("Report type"), {
      target: { value: "general-ledger" },
    });
    expect(screen.queryByLabelText("Compare another period")).toBeNull();
    fireEvent.change(screen.getByLabelText("Report account"), {
      target: { value: "a" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Refresh report" }));
    await waitFor(() => expect(api).toHaveBeenCalledTimes(2));
    expect(api.mock.calls[1][0]).not.toContain("compare_");
  });
});

// Follow-ups must stay tied to the displayed report rather than silently obtaining a fresh revision.
describe("scoped report actions", () => {
  it("requires an explicit account range and pins its report revision", async () => {
    const api = vi
      .fn()
      .mockResolvedValueOnce(fixture)
      .mockResolvedValueOnce({
        ...fixture,
        report: "general-ledger",
        layout: "posting-v1",
        rows: [],
        next_cursor: null,
      });
    render(
      <Reports api={api as API} entity={entity} book={book} locked={false} />,
    );
    request();
    await screen.findByText(/Snapshot 1/);
    fireEvent.click(
      screen.getByRole("button", { name: "Review account 1000" }),
    );
    expect(
      (screen.getByLabelText("Drill starts on") as HTMLInputElement).value,
    ).toBe("");
    fireEvent.change(screen.getByLabelText("Drill starts on"), {
      target: { value: "2026-01-01" },
    });
    fireEvent.change(screen.getByLabelText("Drill ends before"), {
      target: { value: "2026-02-01" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Open account ledger" }),
    );
    await waitFor(() => expect(api).toHaveBeenCalledTimes(2));
    expect(api.mock.calls[1][0]).toContain("/reports/general-ledger?");
    expect(api.mock.calls[1][0]).toContain("account_id=a&snapshot_revision=1");
  });
  it("downloads all selected filters with pinned revision and revokes the object URL", async () => {
    const download = vi.fn().mockResolvedValue(new Blob(["synthetic"]));
    const api = Object.assign(vi.fn().mockResolvedValue(fixture), { download });
    const create = vi.fn().mockReturnValue("blob:synthetic"),
      revoke = vi.fn();
    vi.stubGlobal(
      "URL",
      class extends URL {
        static createObjectURL = create;
        static revokeObjectURL = revoke;
      },
    );
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => {});
    render(
      <Reports api={api as API} entity={entity} book={book} locked={false} />,
    );
    request();
    await screen.findByText(/Snapshot 1/);
    fireEvent.click(
      screen.getByRole("button", { name: "Download review CSV" }),
    );
    await waitFor(() => expect(revoke).toHaveBeenCalledWith("blob:synthetic"));
    expect(download).toHaveBeenCalledWith(
      "/v1/entities/e/books/b/reports/trial-balance/export.csv?through_date=2026-01-31&snapshot_revision=1",
    );
    expect(click).toHaveBeenCalledOnce();
    click.mockRestore();
    vi.unstubAllGlobals();
  });
  it.each([403, 409])(
    "clears all report values after export denial %s",
    async (status) => {
      const api = Object.assign(vi.fn().mockResolvedValue(fixture), {
        download: vi.fn().mockRejectedValue(new APIError(status, false)),
      });
      render(
        <Reports api={api as API} entity={entity} book={book} locked={false} />,
      );
      request();
      await screen.findByText(/Snapshot 1/);
      fireEvent.click(
        screen.getByRole("button", { name: "Download review CSV" }),
      );
      await screen.findByRole("alert");
      expect(screen.queryByText(/Snapshot 1/)).toBeNull();
      expect(screen.queryAllByText("9007199254740993.01")).toHaveLength(0);
    },
  );
  it("ignores a late export when filters changed", async () => {
    let resolve!: (value: Blob) => void;
    const download = vi.fn(
      () =>
        new Promise<Blob>((done) => {
          resolve = done;
        }),
    );
    const api = Object.assign(vi.fn().mockResolvedValue(fixture), { download });
    const create = vi.fn();
    vi.stubGlobal(
      "URL",
      class extends URL {
        static createObjectURL = create;
      },
    );
    render(
      <Reports api={api as API} entity={entity} book={book} locked={false} />,
    );
    request();
    await screen.findByText(/Snapshot 1/);
    fireEvent.click(
      screen.getByRole("button", { name: "Download review CSV" }),
    );
    fireEvent.change(screen.getByLabelText("Report through date (inclusive)"), {
      target: { value: "2026-02-01" },
    });
    resolve(new Blob(["late"]));
    await waitFor(() =>
      expect(screen.queryByText(/Preparing report/)).toBeNull(),
    );
    expect(create).not.toHaveBeenCalled();
    vi.unstubAllGlobals();
  });
});
