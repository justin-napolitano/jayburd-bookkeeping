import { test, expect } from "@playwright/experimental-ct-react";
import type { Page } from "playwright-core";
import { Harness } from "./Harness";
import { positionWorkspace, positionReport } from "./position-fixture";

// A fixed readable source set exercises UI defaults and scope without a provider or mutation.
async function serveCompact(page: Page, multiple = false) {
  const reads: string[] = [];
  const writes: string[] = [];
  const row = (id: string) => ({
    review_id: id,
    description: `Fictional grocer ${id}`,
    booked_date: "2026-10-20",
    amount: "12.34",
    currency: "USD",
    currency_version_id: "usd-v1",
    direction: "outflow",
    state: "needs_review",
    review_revision: "1",
    draft_id: null,
    journal_id: null,
  });
  await page.clock.install({ time: new Date("2026-10-21T15:00:00Z") });
  await page.route("**/v1/**", async (route) => {
    const url = new URL(route.request().url());
    if (
      route.request().method() !== "GET" &&
      !url.pathname.endsWith("/source-activity/query")
    )
      writes.push(url.pathname);
    if (url.pathname === "/v1/workspace")
      return route.fulfill({ json: positionWorkspace });
    if (url.pathname === "/v1/commands/pending")
      return route.fulfill({ json: { command: null } });
    if (url.pathname.endsWith("/reports/balance-sheet"))
      return route.fulfill({
        json: positionReport(
          url.pathname.split("/")[5],
          url.searchParams.get("through_date")!,
          url.searchParams.get("compare_through_date")!,
        ),
      });
    reads.push(url.href);
    const source = {
      source_account_id: "checking",
      display_name: "Fictional checking",
      currency: "USD",
      state: "active",
    };
    if (url.pathname.endsWith("/sources"))
      return route.fulfill({
        json: {
          items: multiple
            ? [
                source,
                {
                  ...source,
                  source_account_id: "savings",
                  display_name: "Fictional savings",
                },
              ]
            : [source],
        },
      });
    if (url.pathname.endsWith("/review-items"))
      return route.fulfill({
        json: {
          source_account_id: url.searchParams.get("source_account_id"),
          source_revision: "1",
          sort: url.searchParams.get("sort"),
          order: url.searchParams.get("order"),
          matching_count: "20",
          start_index: "1",
          items: Array.from({ length: 20 }, (_, i) => row(String(i + 1))),
          next_cursor: null,
        },
      });
    if (url.pathname.endsWith("/categories"))
      return route.fulfill({
        json: {
          review_id: url.pathname.split("/").at(-2),
          source_account_id: "checking",
          book_revision: "12",
          policy_id: "policy-usd",
          accounting_date: "2026-10-20",
          source_currency: "USD",
          functional_currency: "USD",
          fx_required: false,
          accounts: [],
        },
      });
    return route.fulfill({
      json: {
        ...row(url.pathname.split("/").at(-1)!),
        source_account_id: "checking",
        source_revision: "1",
        binding_revision: "1",
        book_revision: "12",
        ledger_account_id: "account-personal-household",
        draft: null,
        blockers: [],
      },
    });
  });
  return { reads, writes };
}

test("390px transaction entry shows the first row and every action without ledger dates", async ({
  mount,
  page,
}, info) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const state = await serveCompact(page);
  await mount(<Harness />);
  await expect(page.getByLabel("Entity and book")).toBeEnabled();
  await page
    .getByLabel("Entity and book")
    .selectOption("personal/personal-household");
  await page.getByRole("button", { name: "Transactions", exact: true }).click();
  await expect(page.getByLabel("Source account")).toHaveValue("checking");
  await expect(
    page.getByRole("heading", { name: "Review transactions", exact: true }),
  ).toHaveCount(1);
  await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1);
  await expect(page.getByLabel("Activity starts on")).toHaveCount(0);
  const first = page.getByRole("row").filter({
    has: page.getByRole("button", {
      name: "Review Fictional grocer 1",
      exact: true,
    }),
  });
  await expect(first).toBeInViewport();
  const box = await first.boundingBox();
  expect(box!.y + box!.height).toBeLessThanOrEqual(844);
  const action = first.getByRole("button");
  await expect(action).toBeInViewport();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: info.outputPath("compact-transactions-mobile.png"),
    fullPage: false,
  });
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.screenshot({
    path: info.outputPath("compact-transactions-desktop.png"),
    fullPage: false,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  const last = page.getByRole("button", {
    name: "Review Fictional grocer 20",
    exact: true,
  });
  await last.click();
  await expect(
    page.getByRole("heading", { name: "Fictional grocer 20", exact: true }),
  ).toBeFocused();
  await expect(
    page.getByRole("table", { name: /Imported transactions/ }),
  ).toBeHidden();
  await expect(
    page.getByRole("button", { name: "Back to transactions" }),
  ).toBeEnabled();
  await page.screenshot({
    path: info.outputPath("focused-transaction-mobile.png"),
    fullPage: false,
  });
  const pagesBefore = state.reads.filter((x) =>
    new URL(x).pathname.endsWith("/review-items"),
  ).length;
  await page.getByRole("button", { name: "Back to transactions" }).click();
  await expect(last).toBeFocused();
  expect(
    state.reads.filter((x) => new URL(x).pathname.endsWith("/review-items")),
  ).toHaveLength(pagesBefore);
  expect(state.writes).toEqual([]);
  await page.addStyleTag({ content: ":root { font-size: 200%; }" });
  await expect(last).toBeFocused();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

test("manual source preference survives ordinary Position navigation but never another book scope", async ({
  mount,
  page,
}) => {
  const state = await serveCompact(page, true);
  await mount(<Harness />);
  await expect(page.getByLabel("Entity and book")).toBeEnabled();
  await page
    .getByLabel("Entity and book")
    .selectOption("personal/personal-household");
  await page.getByRole("button", { name: "Transactions", exact: true }).click();
  await expect(page.getByLabel("Source account")).toBeEnabled();
  await expect(page.getByLabel("Source account")).toHaveValue("");
  await page.getByLabel("Source account").selectOption("savings");
  await expect(
    page.getByRole("button", {
      name: "Review Fictional grocer 1",
      exact: true,
    }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Position", exact: true }).click();
  await page.getByRole("button", { name: "Transactions", exact: true }).click();
  await expect(page.getByLabel("Source account")).toHaveValue("savings");
  await page
    .getByLabel("Entity and book")
    .selectOption("studio/studio-operating");
  await expect(page.getByLabel("Source account")).toBeEnabled();
  await expect(page.getByLabel("Source account")).toHaveValue("");
  expect(state.writes).toEqual([]);
});
