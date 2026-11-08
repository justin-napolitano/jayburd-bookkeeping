import { test, expect } from "@playwright/experimental-ct-react";
import { AccountLabelsHarness } from "./AccountLabelsHarness";

// This verifies only synthetic private presentation and immutable handoff, never a live bank identity.
test("account labels distinguish duplicate types with missing names and hide identifiers behind details", async ({
  mount,
  page,
}, info) => {
  await mount(<AccountLabelsHarness />);
  const review = page.getByRole("button", {
    name: "Review transactions for Bank account 1",
    exact: true,
  });
  await expect(review).toBeVisible();
  await expect(review).toBeEnabled();
  await review.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByLabel("Reviewed synthetic source")).toHaveText(
    "00000000-0000-4000-8000-000000000007",
  );
  await page
    .getByLabel("Known connection attempt ID")
    .fill("00000000-0000-4000-8000-000000000004");
  await page
    .getByRole("button", { name: "Inspect attempt", exact: true })
    .click();
  await expect(
    page.getByText("Bank account 2 — credit / credit card, USD", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByText("Bank account 3 — investment / brokerage, USD", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByText("This account cannot be bound in this profile."),
  ).toBeVisible();
  await expect(
    page.getByText("Provider account ID: private-card-b", { exact: true }),
  ).toBeHidden();
  const details = page.getByText(
    "Technical account details for Bank account 2",
    { exact: true },
  );
  await details.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByText("Provider account ID: private-card-b", { exact: true }),
  ).toBeVisible();
  await page.keyboard.press("Enter");
  await page.screenshot({
    path: info.outputPath("account-labels-desktop.png"),
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({
    path: info.outputPath("account-labels-mobile.png"),
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});
