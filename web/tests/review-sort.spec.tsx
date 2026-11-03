import { test, expect } from "@playwright/experimental-ct-react";
import { ReviewInboxHarness } from "./ReviewInboxHarness";

// Request-level proof distinguishes a full server sort from rearranging a fetched page.
test("every transaction column sorts across pages with keyboard access and a clear end", async ({
  mount,
  page,
}, info) => {
  const reads: URL[] = [];
  const long =
    "Fictional merchant with a deliberately long description and location for narrow screen review";
  await page.route("**/v1/entities/e/books/b/**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname.endsWith("/sources")) {
      await route.fulfill({
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
      return;
    }
    reads.push(url);
    const next = url.searchParams.has("cursor");
    await route.fulfill({
      json: {
        source_account_id: "s",
        source_revision: "1",
        sort: url.searchParams.get("sort"),
        order: url.searchParams.get("order"),
        matching_count: "51",
        start_index: next ? "51" : "1",
        items: [
          {
            review_id: next ? "last" : "first",
            booked_date: "2026-01-31",
            description: long,
            direction: "outflow",
            amount: "10.01",
            currency: "USD",
            currency_version_id: "usd",
            state: "needs_review",
            review_revision: "1",
            draft_id: null,
            journal_id: null,
          },
        ],
        next_cursor: next ? null : "synthetic-cursor",
      },
    });
  });
  await mount(<ReviewInboxHarness />);
  await page.getByLabel("Source account").selectOption("s");
  await expect(
    page.getByRole("columnheader", { name: "Date" }),
  ).toHaveAttribute("aria-sort", "descending");
  await expect(page.getByText("Transactions 1–1 of 51")).toBeVisible();
  expect(reads.at(-1)?.searchParams.get("sort")).toBe("date");
  expect(reads.at(-1)?.searchParams.get("order")).toBe("desc");
  await page
    .getByRole("button", { name: "Next transactions", exact: true })
    .click();
  await expect(page.getByText(/End of matching transactions/)).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Next transactions", exact: true }),
  ).toBeDisabled();
  await page.getByRole("button", { name: "Amount", exact: true }).focus();
  await page.keyboard.press("Enter");
  await expect(page.getByText("Transactions 1–1 of 51")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Amount", exact: true }),
  ).toBeFocused();
  expect(reads.at(-1)?.searchParams.has("cursor")).toBe(false);
  for (const [label, key] of [
    ["Amount", "amount"],
    ["Description", "description"],
    ["Direction", "direction"],
    ["Status", "status"],
    ["Date", "date"],
  ]) {
    const button = page.getByRole("button", { name: label, exact: true });
    await button.click();
    await expect(
      page.getByRole("columnheader", { name: label }),
    ).toHaveAttribute(
      "aria-sort",
      label === "Amount" ? "descending" : "ascending",
    );
    expect(reads.at(-1)?.searchParams.get("sort")).toBe(key);
    expect(reads.at(-1)?.searchParams.has("cursor")).toBe(false);
    if (label !== "Amount") {
      await button.click();
      await expect(
        page.getByRole("columnheader", { name: label }),
      ).toHaveAttribute("aria-sort", "descending");
    }
  }
  await page.getByLabel("Review status").selectOption("");
  await expect(page.getByText("Transactions 1–1 of 51")).toBeVisible();
  expect(reads.at(-1)?.searchParams.has("state")).toBe(false);
  const rowButton = page.getByRole("button", {
    name: "Review " + long,
    exact: true,
  });
  await expect(rowButton).toHaveText("Review");
  await page.screenshot({
    path: info.outputPath("transaction-sort-desktop.png"),
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(
    page.getByRole("button", { name: "Date", exact: true }),
  ).toBeVisible();
  const actionBox = await rowButton.boundingBox();
  expect(actionBox).not.toBeNull();
  expect(actionBox!.x).toBeGreaterThanOrEqual(0);
  expect(actionBox!.x + actionBox!.width).toBeLessThanOrEqual(390);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page
    .getByRole("region", { name: "Transaction table", exact: true })
    .focus();
  await expect(
    page.getByRole("region", { name: "Transaction table", exact: true }),
  ).toBeFocused();
  await page.screenshot({
    path: info.outputPath("transaction-sort-mobile.png"),
    fullPage: true,
  });
  expect(
    reads.every(
      (url) =>
        !url.searchParams.has("amount") && !url.searchParams.has("description"),
    ),
  ).toBe(true);
});
