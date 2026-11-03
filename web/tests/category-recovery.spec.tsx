import { test, expect } from "@playwright/experimental-ct-react";
import { ReviewInboxHarness } from "./ReviewInboxHarness";

// Fictional source/date and exact original units; no provider or ledger mutation.
for (const mode of ["policy", "transient", "empty", "denied", "fx"] as const) {
  test(`TestCategorizationUnavailableKeepsDetail / TestCategorizationNoAutomaticDraft: ${mode}`, async ({
    mount,
    page,
  }) => {
    const writes: string[] = [];
    let recovered = false;
    let reads = 0;
    const row = {
      review_id: "r",
      state: "needs_review",
      review_revision: "1",
      booked_date: "2025-12-20",
      currency: "USD",
      currency_version_id: "usd",
      direction: "outflow",
      amount: "9007199254740993.01",
      description: "Fictional category recovery",
      draft_id: null,
      journal_id: null,
    };
    await page.route("**/v1/entities/e/books/b/**", async (route) => {
      const path = new URL(route.request().url()).pathname;
      if (route.request().method() !== "GET") writes.push(path);
      if (path.endsWith("/categories")) {
        reads++;
        if (!recovered && mode === "policy")
          return route.fulfill({
            status: 422,
            contentType: "application/problem+json",
            body: '{"code":"unsupported_policy","detail":"private synthetic diagnostic"}',
          });
        if (!recovered && mode === "transient")
          return route.fulfill({
            status: 503,
            json: { code: "temporarily_unavailable" },
          });
        if (mode === "denied")
          return route.fulfill({
            status: 403,
            json: { code: "permission_denied" },
          });
        return route.fulfill({
          json: {
            review_id: "r",
            source_account_id: "s",
            book_revision: "4",
            policy_id: "policy",
            accounting_date: row.booked_date,
            source_currency: "USD",
            functional_currency: "USD",
            fx_required: mode === "fx",
            accounts:
              !recovered && mode === "empty"
                ? []
                : [
                    {
                      account_id: "expense",
                      code: "5000",
                      name: "Supplies",
                      category: "expense",
                      normal_side: "debit",
                      monetary: false,
                    },
                  ],
          },
        });
      }
      if (path.endsWith("/sources"))
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
      if (path.endsWith("/review-items"))
        return route.fulfill({
          json: {
            source_account_id: "s",
            source_revision: "2",
            items: [row],
            next_cursor: null,
          },
        });
      return route.fulfill({
        json: {
          ...row,
          source_account_id: "s",
          source_revision: "2",
          binding_revision: "1",
          book_revision: "4",
          ledger_account_id: "cash",
          blockers: [],
          draft: null,
        },
      });
    });
    await mount(<ReviewInboxHarness />);
    await page.getByLabel("Source account").selectOption("s");
    await page
      .getByRole("button", { name: "Review Fictional category recovery" })
      .click();
    if (mode === "denied") {
      await expect(
        page.getByText("Access unavailable", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("heading", { name: row.description }),
      ).toHaveCount(0);
    } else {
      const detail = page.getByRole("region", { name: row.description });
      await expect(detail).toBeVisible();
      await expect(detail).toContainText(row.amount);
      await expect(detail).toContainText(row.booked_date);
      const expected =
        mode === "policy"
          ? "accounting policy cannot support a draft"
          : mode === "empty"
            ? "No eligible category accounts"
            : mode === "fx"
              ? "explicit FX valuation is required"
              : "Accounting categories could not be loaded";
      await expect(detail.getByRole("alert")).toContainText(expected);
      await expect(
        detail.getByRole("button", { name: "Create reviewed draft" }),
      ).toHaveCount(0);
      await expect(detail).not.toContainText("private synthetic diagnostic");
      if (mode !== "fx") {
        recovered = true;
        await page
          .getByRole("button", { name: "Retry categorization" })
          .click();
        await expect(page.getByLabel("Category account")).toBeVisible();
        await expect(page.getByLabel("Accounting date")).toHaveValue(
          row.booked_date,
        );
        await expect(
          page.getByRole("button", { name: "Create reviewed draft" }),
        ).toBeDisabled();
        expect(reads).toBe(2);
      }
    }
    expect(writes).toEqual([]);
  });
}
