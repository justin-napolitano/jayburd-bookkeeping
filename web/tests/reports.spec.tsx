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

test("comparison, account drill, pinned journal and download remain scoped", async ({
  mount,
  page,
}) => {
  let exported = false,
    inspected = false;
  await page.route("**/reports/**", async (route) => {
    const url = new URL(route.request().url());
    const filters = Object.fromEntries(url.searchParams);
    if (url.pathname.endsWith("/export.csv")) {
      expect(filters.snapshot_revision).toBe("1");
      expect(filters.cursor).toBeUndefined();
      expect(filters.limit).toBeUndefined();
      expect(route.request().headers().authorization).toBe(
        "Bearer synthetic-token",
      );
      exported = true;
      await route.fulfill({
        contentType: "text/csv; charset=utf-8; header=present",
        body: '"synthetic"\r\n"0.01"\r\n',
      });
      return;
    }
    if (url.pathname.includes("/journals/")) {
      expect(filters).toMatchObject({
        account_id: "cash",
        starts_on: "2026-01-01",
        ends_before: "2026-02-01",
        line_number: "1",
        snapshot_revision: "1",
      });
      inspected = true;
      await route.fulfill({
        json: {
          contract_version: "report-journal-v1",
          entity_id: "entity",
          book_id: "one",
          snapshot_revision: "1",
          selector: { ...filters, journal_id: "journal", line_number: 1 },
          journal: {
            entity_id: "entity",
            book_id: "one",
            journal_id: "journal",
            book_revision: "1",
            functional_currency_version_id: "currency",
            accounting_date: "2026-01-20",
            kind: "manual",
            policy_id: "policy",
            description: "Synthetic reviewed journal",
            lines: [
              {
                line_number: 1,
                account_id: "cash",
                side: "debit",
                transaction_amount: "10",
                transaction_currency: "EUR",
                transaction_currency_version_id: "eur",
                functional_amount: "11",
                valuation_basis: "spot",
                rate_observation_id: "rate",
              },
            ],
          },
        },
      });
      return;
    }
    const report = url.pathname.split("/").pop();
    if (report === "general-ledger") {
      expect(filters.snapshot_revision).toBe("1");
      await route.fulfill({
        json: {
          ...common,
          report,
          layout: "posting-v1",
          filters: {
            report,
            starts_on: filters.starts_on,
            ends_before: filters.ends_before,
            account_id: "cash",
          },
          comparison: null,
          totals: { opening: "0", closing: "11" },
          rows: [
            {
              id: "line:one",
              accounting_date: "2026-01-20",
              journal_id: "journal",
              line_number: 1,
              amounts: { debit: "11", credit: "0", running_balance: "11" },
              transaction_amount: "10",
              transaction_currency: "EUR",
              valuation_basis: "spot",
              kind: "manual",
              policy_id: "policy",
              rate_observation_id: "rate",
            },
          ],
        },
      });
      return;
    }
    expect(filters.compare_through_date).toBe("2026-01-01");
    await route.fulfill({
      json: {
        ...common,
        report,
        filters: {
          report,
          through_date: filters.through_date,
          compare_through_date: filters.compare_through_date,
        },
        comparison: {
          totals: { total_debits: "11.01", total_credits: "11.01" },
          difference: { total_debits: "-0.01", total_credits: "-0.01" },
        },
        totals: { total_debits: "11", total_credits: "11" },
        rows: [
          {
            id: "account:cash",
            account_id: "cash",
            account_code: "1000",
            account_name: "Cash",
            category: "asset",
            amounts: { debit_balance: "11", credit_balance: "0", net: "11" },
            comparison: {
              amounts: {
                debit_balance: "11.01",
                credit_balance: "0",
                net: "11.01",
              },
              difference: {
                debit_balance: "-0.01",
                credit_balance: "0",
                net: "-0.01",
              },
            },
          },
        ],
      },
    });
  });
  await mount(<ReportsHarness />);
  await page.getByLabel("Report through date (inclusive)").fill("2026-01-31");
  await page.getByLabel("Compare another period").check();
  await page
    .getByLabel("Comparison through date (inclusive)")
    .fill("2026-01-01");
  await page.getByRole("button", { name: "Refresh report" }).click();
  await expect(
    page.getByRole("cell", { name: "-0.01", exact: true }),
  ).toHaveCount(4);
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download review CSV" }).click();
  expect((await download).suggestedFilename()).toBe(
    "financial-reporting-review.csv",
  );
  expect(exported).toBe(true);
  await page.getByRole("button", { name: "Review account 1000" }).click();
  await expect(page.getByLabel("Drill starts on")).toHaveValue("");
  await page.getByLabel("Drill starts on").fill("2026-01-01");
  await page.getByLabel("Drill ends before").fill("2026-02-01");
  await page.getByRole("button", { name: "Open account ledger" }).click();
  await page.getByRole("button", { name: "Review journal line 1" }).click();
  await expect(
    page.getByRole("region", { name: "Journal detail" }),
  ).toContainText("Synthetic reviewed journal");
  expect(inspected).toBe(true);
  await page.getByLabel("Test scope").selectOption("two");
  await expect(
    page.getByRole("region", { name: "Journal detail" }),
  ).toHaveCount(0);
});
