import { test, expect } from "@playwright/experimental-ct-react";
import type { Page } from "playwright-core";
import { Harness } from "./Harness";
import { positionReport, positionWorkspace } from "./position-fixture";

// Browser routes expose only fictional, server-shaped reads to the actual app shell.
async function servePosition(page: Page, deniedBook = "", pending = false) {
  const requests: string[] = [];
  await page.route("**/v1/**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/v1/workspace")
      return route.fulfill({ json: positionWorkspace });
    if (url.pathname === "/v1/commands/pending")
      return route.fulfill({
        json: {
          command: pending
            ? {
                command_id: "fictional-prepared-import",
                operation: "import",
                entity_id: "studio",
                book_id: "studio-operating",
                state: "prepared",
                acknowledged: false,
                result: null,
                created_at: "2026-10-21T14:00:00Z",
                body: {
                  batch_id: "fictional-batch",
                  preview_id: "fictional-preview",
                  preview_revision: "1",
                  binding_revision: "1",
                  source_revision: "0",
                  profile: "jayburd-synthetic-statement-csv/1",
                },
              }
            : null,
        },
      });
    if (url.pathname.endsWith("/reports/balance-sheet")) {
      requests.push(url.href);
      const book = url.pathname.split("/")[5];
      if (book === deniedBook)
        return route.fulfill({ status: 403, json: { code: "forbidden" } });
      return route.fulfill({
        json: positionReport(
          book,
          url.searchParams.get("through_date") ?? "",
          url.searchParams.get("compare_through_date") ?? "",
        ),
      });
    }
    return route.fulfill({ status: 404, json: { code: "not_found" } });
  });
  return requests;
}

// Fixed browser date makes initial as-of and comparison inputs deterministic.
async function prepare(page: Page) {
  await page.clock.install({ time: new Date("2026-10-21T15:00:00Z") });
  return servePosition(page);
}

test("all books renders exact separate position rows and scoped tools", async ({
  mount,
  page,
}, testInfo) => {
  const pageErrors: string[] = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));
  await page.setViewportSize({ width: 1440, height: 900 });
  const requests = await prepare(page);
  await mount(<Harness />);
  const table = page.locator(".position-table");
  await expect(table.locator(".position-book-row")).toHaveCount(5);
  await expect(page.getByLabel("Entity and book")).toHaveValue("");
  await expect(
    page.getByText("All entities and books", { exact: false }),
  ).toBeVisible();
  await expect(table.locator(".position-entity-row")).toHaveCount(3);
  await expect(table.locator(".position-row-striped")).toHaveCount(3);
  await expect(table.getByRole("cell", { name: "22,900.00" })).toHaveCount(1);
  await expect(table.getByRole("cell", { name: "+4,800.00" })).toHaveCount(1);
  await expect(table.getByRole("cell", { name: "−600.00" })).toHaveCount(1);
  expect(requests).toHaveLength(5);
  for (const request of requests) {
    const query = new URL(request).searchParams;
    expect(query.get("through_date")).toBe("2026-10-21");
    expect(query.get("compare_through_date")).toBe("2026-09-30");
  }
  await page.screenshot({
    path: testInfo.outputPath("position-desktop.png"),
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);

  await page.getByLabel("Entity and book").selectOption("entity:holdings");
  await expect(table.locator(".position-book-row")).toHaveCount(2);
  await page
    .getByLabel("Entity and book")
    .selectOption("holdings/holdings-europe");
  await expect(table.locator(".position-book-row")).toHaveCount(1);
  await expect(table.getByRole("button", { name: "Europe" })).toBeVisible();
  await page.getByLabel("Entity and book").selectOption("");
  await expect(table.locator(".position-book-row")).toHaveCount(5);

  await page.getByRole("button", { name: "Ledger", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Ledger" })).toBeVisible();
  await page.getByRole("button", { name: "Household" }).click();
  await expect(page.getByLabel("Entity and book")).toHaveValue(
    "personal/personal-household",
  );
  await expect(page.locator(".book-workspace .view-title")).toBeVisible();
  await expect(page.locator(".book-workspace .view-title")).toHaveText(
    "Ledger",
  );
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(
    page.getByRole("region", { name: "Financial reports" }),
  ).toBeVisible();
  expect(pageErrors).toEqual([]);
});

test("date edits clear old values and one denied book leaves the rest readable", async ({
  mount,
  page,
}) => {
  await page.clock.install({ time: new Date("2026-10-21T15:00:00Z") });
  await servePosition(page, "holdings-europe");
  await mount(<Harness />);
  const table = page.locator(".position-table");
  await expect(table.getByRole("cell", { name: "22,900.00" })).toBeVisible();
  await expect(table.getByText("Access unavailable")).toBeVisible();
  await page.getByLabel("As of date").fill("2026-10-20");
  await expect(table.getByRole("cell", { name: "22,900.00" })).toHaveCount(0);
  await expect(table.getByRole("cell", { name: "100.00" })).toHaveCount(8);
  await expect(table.getByText("Access unavailable")).toBeVisible();
});

test("ordinary Ledger navigation does not reuse an already refreshed Position handoff", async ({
  mount,
  page,
}) => {
  await prepare(page);
  const queries: URLSearchParams[] = [];
  await page.route("**/reports/general-ledger?*", async (route) => {
    queries.push(new URL(route.request().url()).searchParams);
    if (queries.length === 1)
      return route.fulfill({
        status: 409,
        json: { code: "report_snapshot_changed" },
      });
    return route.fulfill({
      json: {
        ...positionReport("studio-reserve"),
        report: "general-ledger",
        layout: "posting-v1",
        snapshot_revision: "13",
        comparison: null,
        totals: {
          opening: "0",
          period_debits: "8000",
          period_credits: "0",
          closing: "8000",
        },
      },
    });
  });
  await mount(<Harness />);
  await expect(
    page.locator(".position-equity").filter({ hasText: "8,000.00" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Reserve", exact: true }).click();
  await page
    .getByLabel("Report account")
    .selectOption("account-studio-reserve");
  await page.getByRole("button", { name: "Refresh report" }).click();
  await expect(page.getByRole("alert")).toContainText("book changed");
  expect(queries[0].get("snapshot_revision")).toBe("12");
  await page.getByRole("button", { name: "Refresh report" }).click();
  await expect(page.getByText(/Snapshot 13/)).toBeVisible();
  expect(queries[1].has("snapshot_revision")).toBe(false);
  await page.getByRole("button", { name: "Position", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Reserve", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Ledger", exact: true }).click();
  await page
    .getByLabel("Report account")
    .selectOption("account-studio-reserve");
  await page.getByRole("button", { name: "Refresh report" }).click();
  await expect(page.getByText(/Snapshot 13/)).toBeVisible();
  expect(queries[2].has("snapshot_revision")).toBe(false);
});

test("mobile Position keeps every amount visible without sideways scrolling", async ({
  mount,
  page,
}, testInfo) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await prepare(page);
  await mount(<Harness />);
  await expect(page.locator(".position-book-row")).toHaveCount(5);
  await page.screenshot({
    path: testInfo.outputPath("position-mobile.png"),
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  expect(
    await page
      .locator(".position-table-scroll")
      .evaluate((node) => node.scrollWidth > node.clientWidth),
  ).toBe(false);
});

test("a recovered command locks Position reads, scope and navigation", async ({
  mount,
  page,
}) => {
  await page.clock.install({ time: new Date("2026-10-21T15:00:00Z") });
  const requests = await servePosition(page, "", true);
  await mount(<Harness />);
  await expect(
    page.getByText("Saved for review.", { exact: false }),
  ).toBeVisible();
  await expect(page.locator(".position-book-row")).toHaveCount(5);
  await expect(
    page.getByText("Locked while a command needs acknowledgement"),
  ).toHaveCount(5);
  await expect(page.getByLabel("Entity and book")).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Ledger", exact: true }),
  ).toBeDisabled();
  expect(requests).toHaveLength(0);
});
