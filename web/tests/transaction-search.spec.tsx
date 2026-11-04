import { test, expect } from "@playwright/experimental-ct-react";
import { ReviewInboxHarness } from "./ReviewInboxHarness";

// Fifty-one fictional rows put the second coffee beyond the legacy first page.
const rows = Array.from({ length: 51 }, (_, i) => ({
  review_id: `r${i}`,
  state: "needs_review",
  review_revision: "1",
  booked_date: i === 50 ? "2026-01-02" : "2026-01-01",
  currency: "USD",
  currency_version_id: "usd",
  direction: "outflow",
  amount: "9007199254740993.01",
  description:
    i === 0 || i === 50 ? `Fictional Coffee ${i}` : `Fictional supplies ${i}`,
  draft_id: null,
  journal_id: null,
}));

for (const viewport of [1440, 390]) {
  test(`private search, dates, clear and whole-result pages at ${viewport}px`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width: viewport, height: 844 });
    const queries: Record<string, unknown>[] = [];
    const unexpectedWrites: string[] = [];
    let fail = false;
    let releaseFailure!: () => void;
    const heldFailure = new Promise<void>((resolve) => {
      releaseFailure = resolve;
    });
    await page.route("**/v1/entities/e/books/b/**", async (route) => {
      const req = route.request();
      const url = new URL(req.url());
      if (url.pathname.endsWith("/query")) {
        expect(url.search).toBe("");
        expect(req.method()).toBe("POST");
        expect(req.headers()["idempotency-key"]).toBeUndefined();
        const body = req.postDataJSON();
        queries.push(body);
        if (fail) {
          await heldFailure;
          return route.fulfill({
            status: 503,
            json: { code: "temporarily_unavailable" },
          });
        }
        const matches =
          body.search === "Coffee"
            ? [rows[0], rows[50]]
            : body.search
              ? []
              : rows;
        const items = body.cursor ? matches.slice(50) : matches.slice(0, 50);
        return route.fulfill({
          json: {
            profile: "source-review-query-v1",
            source_account_id: "s",
            source_revision: "7",
            items,
            next_cursor:
              !body.cursor && matches.length > 50 ? "opaque-page-two" : null,
            matching_count: String(matches.length),
            start_index: items.length ? (body.cursor ? "51" : "1") : "0",
            sort: body.sort,
            order: body.order,
            source_count: "51",
            available_from: "2026-01-01",
            available_through: "2026-01-02",
          },
        });
      }
      if (req.method() !== "GET") unexpectedWrites.push(url.pathname);
      if (url.pathname.endsWith("/sources"))
        return route.fulfill({
          json: {
            items: [
              {
                source_account_id: "s",
                display_name: "Fictional checking",
                currency: "USD",
                state: "active",
              },
            ],
          },
        });
      return route.fulfill({
        json: {
          source_account_id: "s",
          source_revision: "7",
          items: rows.slice(0, 50),
          next_cursor: "legacy-next",
          matching_count: "51",
          start_index: "1",
          sort: "date",
          order: "desc",
        },
      });
    });
    await mount(<ReviewInboxHarness />);
    await page.getByLabel("Source account").selectOption("s");
    await expect(
      page.getByRole("button", {
        name: "Review Fictional Coffee 0",
        exact: true,
      }),
    ).toBeVisible();
    expect(queries).toEqual([]);
    await page.getByText("Search and dates", { exact: true }).first().click();
    await page.getByLabel("Description contains").fill("Coffee");
    await page.getByLabel("From date").fill("2026-01-01");
    await page.getByLabel("Through date").fill("2026-01-02");
    expect(queries).toEqual([]);
    await page.getByRole("button", { name: "Apply filters" }).focus();
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("button", {
        name: "Review Fictional Coffee 50",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("status").filter({ hasText: /transactions/i }),
    ).toContainText("Transactions 1–2 of 2");
    await page.getByText("About this list", { exact: true }).click();
    await expect(
      page.getByText(/The available imported range is not proof/),
    ).toBeVisible();
    await page.getByText("About this list", { exact: true }).click();
    expect(queries[0]).toEqual({
      source_account_id: "s",
      state: "needs_review",
      limit: 50,
      cursor: null,
      sort: "date",
      order: "desc",
      search: "Coffee",
      starts_on: "2026-01-01",
      through_date: "2026-01-02",
    });
    await page.getByRole("button", { name: "Amount", exact: true }).click();
    await expect.poll(() => queries.length).toBe(2);
    expect(queries[1]).toMatchObject({
      search: "Coffee",
      cursor: null,
      sort: "amount",
    });
    await page.screenshot({
      path: info.outputPath(`search-${viewport}.png`),
      fullPage: true,
    });
    await page.getByLabel("Description contains").fill("Absent");
    fail = true;
    await page.getByRole("button", { name: "Apply filters" }).click();
    await expect(
      page.getByText("Loading review data…", { exact: true }),
    ).toBeVisible();
    await page.screenshot({
      path: info.outputPath(`loading-${viewport}.png`),
      fullPage: true,
    });
    releaseFailure();
    await expect(page.getByRole("alert")).toHaveText(
      "Data is unavailable right now. Retry the read.",
    );
    await page.screenshot({
      path: info.outputPath(`error-${viewport}.png`),
      fullPage: true,
    });
    await expect(page.getByText(/Imported history:/)).toHaveCount(0);
    fail = false;
    await page.getByRole("button", { name: "Refresh transactions" }).click();
    await expect(
      page.getByRole("status").filter({ hasText: /transactions/i }),
    ).toContainText("0 transactions of 0");
    await expect(page.getByText(/Imported history:/)).toContainText(
      "51 transactions",
    );
    await expect(
      page.getByText("No transactions match these filters.", { exact: true }),
    ).toBeVisible();
    await page.screenshot({
      path: info.outputPath(`empty-${viewport}.png`),
      fullPage: true,
    });
    await page.getByRole("button", { name: "Clear filters" }).click();
    await expect(
      page.getByRole("status").filter({ hasText: /transactions/i }),
    ).toContainText("Transactions 1–50 of 51");
    await page.getByRole("button", { name: "Next transactions" }).click();
    await expect(
      page.getByRole("status").filter({ hasText: /transactions/i }),
    ).toContainText("Transactions 51–51 of 51");
    expect(queries.at(-1)).toMatchObject({
      search: "",
      starts_on: null,
      through_date: null,
      cursor: "opaque-page-two",
    });
    expect(unexpectedWrites).toEqual([]);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    expect(
      await page.evaluate(() => ({
        local: localStorage.length,
        session: sessionStorage.length,
      })),
    ).toEqual({ local: 0, session: 0 });
  });
}
