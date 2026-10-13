import { test, expect } from "@playwright/experimental-ct-react";
import { ReviewInboxHarness } from "./ReviewInboxHarness";
// Fixed independent fixtures test UI/request semantics, not SQL accounting correctness.
const row = {
  review_id: "r",
  state: "needs_review",
  review_revision: "1",
  booked_date: "2026-01-04",
  currency: "USD",
  currency_version_id: "usd",
  direction: "outflow",
  amount: "12.51",
  description: "Synthetic supplies",
  draft_id: null,
  journal_id: null,
};
const body = {
  expected_revision: "4",
  policy_id: "policy",
  accounting_date: row.booked_date,
  description: row.description,
  lines: [
    {
      account_id: "cash",
      side: "credit",
      transaction_currency: "USD",
      transaction_amount: "12.51",
      functional_amount: "12.51",
      valuation_basis: "identity",
    },
    {
      account_id: "expense",
      side: "debit",
      transaction_currency: "USD",
      transaction_amount: "12.51",
      functional_amount: "12.51",
      valuation_basis: "identity",
    },
  ],
};
test("source review is keyboard accessible and hands off an exact draft on desktop/mobile", async ({
  mount,
  page,
}, info) => {
  const writes: string[] = [];
  await page.route("**/v1/entities/e/books/b/**", async (route) => {
    const request = route.request(),
      url = new URL(request.url());
    if (request.method() === "POST") writes.push(url.pathname);
    let value: unknown;
    if (url.pathname.endsWith("/sources"))
      value = {
        items: [
          {
            source_account_id: "s",
            display_name: "Fictional LLC checking",
            currency: "USD",
            state: "active",
          },
        ],
      };
    else if (url.pathname.endsWith("/categories"))
      value = {
        review_id: "r",
        source_account_id: "s",
        book_revision: "4",
        policy_id: "policy",
        accounting_date: row.booked_date,
        source_currency: "USD",
        functional_currency: "USD",
        fx_required: false,
        accounts: [
          {
            account_id: "expense",
            code: "5000",
            name: "Supplies",
            category: "expense",
            normal_side: "debit",
            monetary: false,
          },
        ],
      };
    else if (url.pathname.endsWith("/drafts")) {
      expect(request.postDataJSON()).toEqual({
        expected_review_revision: "1",
        source_revision: "2",
        binding_revision: "1",
        expected_book_revision: "4",
        category_account_id: "expense",
        policy_id: "policy",
        accounting_date: row.booked_date,
        description: row.description,
      });
      expect(request.headers()["idempotency-key"]).toBeTruthy();
      value = {
        draft_id: "d",
        review_id: "r",
        review_revision: "2",
        source_revision: "3",
        binding_revision: "1",
        book_revision: "4",
        posting_body: body,
      };
    } else if (url.pathname.endsWith("/review-items"))
      value = {
        source_account_id: "s",
        source_revision: "2",
        items: [row],
        next_cursor: null,
      };
    else
      value = {
        ...row,
        source_account_id: "s",
        source_revision: "2",
        binding_revision: "1",
        book_revision: "4",
        ledger_account_id: "cash",
        blockers: [],
        draft: null,
      };
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(value),
    });
  });
  await mount(<ReviewInboxHarness />);
  await page.getByLabel("Source account").selectOption("s");
  const review = page.getByRole("button", {
    name: "Review Synthetic supplies",
  });
  await review.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByLabel("Category account")).toBeVisible();
  await expect(page.getByLabel("Accounting date")).toHaveAttribute(
    "readonly",
    "",
  );
  await page.getByLabel("Category account").selectOption("expense");
  await page.getByRole("button", { name: "Create reviewed draft" }).click();
  await expect(
    page.getByRole("table", { name: "Exact proposed journal lines" }),
  ).toBeVisible();
  expect(writes).toHaveLength(1);
  await page.screenshot({
    path: info.outputPath("review-desktop.png"),
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(
    page.getByRole("button", { name: "Review posting command" }),
  ).toBeVisible();
  await page.screenshot({
    path: info.outputPath("review-mobile.png"),
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.getByRole("button", { name: "Review posting command" }).click();
  await expect(
    page.getByRole("heading", {
      name: "Posting command awaiting confirmation",
    }),
  ).toBeVisible();
  expect(writes).toEqual(["/v1/entities/e/books/b/review-items/r/drafts"]);
});
