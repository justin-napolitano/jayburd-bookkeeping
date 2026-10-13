import { test, expect } from "@playwright/experimental-ct-react";
import { ReportsHarness } from "./ReportsHarness";
// Fixed fixtures prove presentation and request lifecycle, not accounting correctness.
const common = {
  contract_version: "financial-reporting-v1",
  layout: "account-v1",
  entity_id: "entity",
  book_id: "one",
  snapshot_revision: "1",
  generated_at: "2026-01-31T00:00:00Z",
  basis: "accrual",
  timezone: "UTC",
  functional_currency: "USD",
  minor_units: 2,
  functional_currency_version_id: "currency",
  policy_ids: ["policy"],
  next_cursor: null,
};
test("four report views show exact totals, original units and revision-bound pages", async ({
  mount,
  page,
}) => {
  let changed = false;
  await page.route("**/reports/*?*", async (route) => {
    const url = new URL(route.request().url());
    const report = url.pathname.split("/").pop();
    if (url.searchParams.has("cursor")) {
      expect(url.searchParams.get("snapshot_revision")).toBe("1");
      await route.fulfill({ status: 409, body: "{}" });
      changed = true;
      return;
    }
    const totals =
      report === "trial-balance"
        ? { total_debits: "1300", total_credits: "1300" }
        : report === "balance-sheet"
          ? { assets: "1145", liabilities: "0", total_equity: "1145" }
          : report === "profit-and-loss"
            ? { revenue: "200", expense: "130", net_income: "70" }
            : {
                opening: "1000",
                period_debits: "11",
                period_credits: "0",
                closing: "1011",
              };
    const rows =
      report === "general-ledger"
        ? [
            {
              id: "line:one",
              accounting_date: "2026-01-20",
              journal_id: "journal",
              line_number: 1,
              amounts: { debit: "11", credit: "0", running_balance: "1011" },
              transaction_amount: "10",
              transaction_currency: "EUR",
              valuation_basis: "spot",
              kind: "original",
              policy_id: "policy",
              rate_observation_id: "rate",
            },
          ]
        : [
            {
              id: "account:cash",
              account_code: "1000",
              account_name: "Cash",
              category: "asset",
              amounts: {
                debit_balance: "1045",
                credit_balance: "0",
                net: "1045",
                balance: "1045",
              },
            },
          ];
    await route.fulfill({
      json: {
        ...common,
        report,
        layout:
          report === "balance-sheet"
            ? "basic-unclassified-v1"
            : report === "general-ledger"
              ? "posting-v1"
              : "account-v1",
        totals,
        rows,
        next_cursor: report === "general-ledger" && !changed ? "opaque" : null,
      },
    });
  });
  await mount(<ReportsHarness />);
  await page.getByLabel("Report through date (inclusive)").fill("2026-01-31");
  await page.getByRole("button", { name: "Refresh report" }).click();
  await expect(
    page.getByRole("cell", { name: "1300.00", exact: true }),
  ).toHaveCount(2);
  for (const kind of ["balance-sheet", "profit-and-loss", "general-ledger"]) {
    await page.getByLabel("Report type").selectOption(kind);
    await expect(page.getByText(/Snapshot 1/)).toHaveCount(0);
    if (kind !== "balance-sheet") {
      await page.getByLabel("Report starts on").fill("2026-01-01");
      await page.getByLabel("Report ends before").fill("2026-02-01");
    }
    if (kind === "general-ledger")
      await page.getByLabel("Report account").selectOption("cash");
    await page.getByRole("button", { name: "Refresh report" }).click();
    await expect(page.getByText(/Snapshot 1/)).toBeVisible();
    const totalRow = page
      .locator("table")
      .first()
      .getByRole("row")
      .filter({
        hasText:
          kind === "balance-sheet"
            ? "Assets"
            : kind === "profit-and-loss"
              ? "Net income"
              : "Closing balance",
      });
    await expect(totalRow.locator("td")).toHaveText(
      kind === "balance-sheet"
        ? "1145.00"
        : kind === "profit-and-loss"
          ? "70.00"
          : "1011.00",
    );
  }
  await expect(page.getByRole("cell", { name: /10 EUR/ })).toBeVisible();
  await page.screenshot({
    path: "/private/tmp/jayburd-reporting-screens.png",
    fullPage: true,
  });
  await page.getByRole("button", { name: "Next report page" }).click();
  await expect(page.getByRole("alert")).toContainText("book changed");
  await expect(page.getByRole("cell", { name: /10 EUR/ })).toHaveCount(0);
  await page.getByRole("button", { name: "Refresh report" }).click();
  await expect(page.getByRole("cell", { name: /10 EUR/ })).toBeVisible();
});
test("late requests cannot repopulate after a scope change or sign out", async ({
  mount,
  page,
}) => {
  let release!: () => void;
  let started!: () => void;
  let start = new Promise<void>((r) => {
    started = r;
  });
  await page.route("**/reports/*?*", async (route) => {
    started();
    await new Promise<void>((r) => {
      release = r;
    });
    await route.fulfill({
      json: {
        ...common,
        report: "trial-balance",
        totals: { total_debits: "999" },
        rows: [],
      },
    });
  });
  await mount(<ReportsHarness />);
  await page.getByLabel("Report through date (inclusive)").fill("2026-01-31");
  await page.getByRole("button", { name: "Refresh report" }).click();
  await start;
  await page.getByLabel("Test scope").selectOption("two");
  release();
  await expect(page.getByText(/Snapshot 1/)).toHaveCount(0);
  start = new Promise<void>((r) => {
    started = r;
  });
  await page.getByLabel("Report through date (inclusive)").fill("2026-01-31");
  await page.getByRole("button", { name: "Refresh report" }).click();
  await start;
  await page.getByRole("button", { name: "Sign out test session" }).click();
  release();
  await expect(
    page.getByRole("region", { name: "Financial reports" }),
  ).toHaveCount(0);
});
