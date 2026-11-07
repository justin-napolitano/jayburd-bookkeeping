import { test, expect } from "@playwright/experimental-ct-react";
import type { Page } from "playwright-core";
import { Harness } from "./Harness";
import { positionReport, positionWorkspace } from "./position-fixture";

// Read-only fixtures record every scope and mutation; navigation cannot invent a bank selection.
async function serveEntry(page: Page, sourceState = "ready") {
  const reads: string[] = [];
  const mutations: string[] = [];
  await page.clock.install({ time: new Date("2026-10-21T15:00:00Z") });
  await page.route("**/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    // Private discovery is separate from this book-only read/navigation fixture.
    if (
      url.pathname === "/v1/collection" ||
      url.pathname === "/v1/collection-recovery"
    )
      return route.fulfill({ status: 404, json: { code: "not_found" } });
    if (request.method() !== "GET") mutations.push(url.pathname);
    if (url.pathname === "/v1/workspace")
      return route.fulfill({ json: positionWorkspace });
    if (url.pathname === "/v1/commands/pending")
      return route.fulfill({ json: { command: null } });
    if (url.pathname.endsWith("/reports/balance-sheet"))
      return route.fulfill({
        json: positionReport(
          url.pathname.split("/")[5],
          url.searchParams.get("through_date") ?? "",
          url.searchParams.get("compare_through_date") ?? "",
        ),
      });
    reads.push(url.pathname);
    if (url.pathname.endsWith("/sources")) {
      if (sourceState === "unavailable" || sourceState === "denied")
        return route.fulfill({
          status: sourceState === "denied" ? 403 : 503,
          json: {
            code: sourceState === "denied" ? "forbidden" : "unavailable",
          },
        });
      return route.fulfill({
        json: {
          items:
            sourceState === "empty"
              ? []
              : [
                  {
                    source_account_id: "fictional-household-checking",
                    display_name: "Fictional checking",
                    currency: "USD",
                    state: "active",
                  },
                ],
        },
      });
    }
    if (url.pathname.endsWith("/review-items"))
      return route.fulfill({
        json: {
          source_account_id: "fictional-household-checking",
          source_revision: "1",
          sort: "date",
          order: "desc",
          matching_count: "0",
          start_index: "1",
          items: [],
          next_cursor: null,
        },
      });
    return route.fulfill({ status: 404, json: { code: "not_found" } });
  });
  return { reads, mutations };
}

test("Transactions is a keyboard-visible primary task with explicit all-books scope", async ({
  mount,
  page,
}, info) => {
  const state = await serveEntry(page);
  await mount(<Harness />);
  const nav = page.getByRole("navigation", { name: "Finance tools" });
  const entry = nav.getByRole("button", { name: "Transactions", exact: true });
  await expect(entry).toBeVisible();
  await entry.focus();
  await page.keyboard.press("Enter");
  await expect(entry).toHaveAttribute("aria-current", "page");
  await expect(
    page.getByRole("heading", { level: 1, name: "Review transactions" }),
  ).toBeFocused();
  await expect(page.getByLabel("Entity and book")).toHaveValue("");
  expect(state.reads).toEqual([]);
  await page.getByRole("button", { name: "Household", exact: true }).click();
  await expect(page.getByLabel("Entity and book")).toHaveValue(
    "personal/personal-household",
  );
  await page
    .getByLabel("Source account")
    .selectOption("fictional-household-checking");
  await expect(
    page.getByText("No transactions match this status."),
  ).toBeVisible();
  expect(
    state.reads.every((path) =>
      path.startsWith("/v1/entities/personal/books/personal-household/"),
    ),
  ).toBe(true);
  await page.getByRole("button", { name: "Position", exact: true }).click();
  await expect(
    page.getByText(
      "These are posted ledger balances. Imported bank transactions are reviewed separately.",
    ),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "View imported transactions", exact: true })
    .click();
  await expect(page.getByLabel("Entity and book")).toHaveValue(
    "personal/personal-household",
  );
  await expect(page.getByLabel("Source account")).toBeEnabled();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(entry).toBeInViewport();
  await entry.focus();
  await expect(entry).toBeFocused();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: info.outputPath("transaction-entry-mobile.png"),
    fullPage: true,
  });
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.getByRole("button", { name: "Position", exact: true }).click();
  await expect(
    page
      .locator(".position-book-row")
      .getByRole("cell", { name: "18,000.00", exact: true }),
  ).toBeVisible();
  await page.screenshot({
    path: info.outputPath("position-import-handoff.png"),
    fullPage: true,
  });
  expect(state.mutations).toEqual([]);
});

for (const sourceState of ["empty", "unavailable", "denied"]) {
  test(`primary entry preserves ${sourceState} source behavior without registering or posting`, async ({
    mount,
    page,
  }) => {
    const state = await serveEntry(page, sourceState);
    await mount(<Harness />);
    await expect(page.getByLabel("Entity and book")).toBeEnabled();
    await page
      .getByLabel("Entity and book")
      .selectOption("personal/personal-household");
    await page
      .getByRole("button", { name: "Transactions", exact: true })
      .click();
    if (sourceState === "empty")
      await expect(
        page.getByText("No source accounts are available in this book."),
      ).toBeVisible();
    if (sourceState === "unavailable")
      await expect(
        page.getByRole("button", { name: "Retry source accounts" }),
      ).toBeVisible();
    if (sourceState === "denied") {
      await expect(page.getByLabel("Entity and book")).toHaveValue("");
      await expect(
        page.getByRole("combobox", { name: "Source account", exact: true }),
      ).toHaveCount(0);
    }
    await expect.poll(() => state.reads.length).toBeGreaterThan(0);
    expect(state.mutations).toEqual([]);
  });
}
