import { test, expect } from "@playwright/experimental-ct-react";
import { AccountLabelsHarness } from "./AccountLabelsHarness";

// This verifies only synthetic private presentation and immutable handoff, never a live bank identity.
test("account labels preserve unknown names and discover unfinished setup without an identifier lookup", async ({
  mount,
  page,
}, info) => {
  await mount(<AccountLabelsHarness recoverable recoveryOnEarlierPage />);
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
  await expect(page.getByLabel("Known connection attempt ID")).toHaveCount(0);
  const recovery = page.getByRole("region", {
    name: "Unfinished connections",
    exact: true,
  });
  await expect(
    recovery.getByRole("button", { name: "Review bank setup", exact: true }),
  ).toHaveCount(0);
  await recovery
    .getByRole("button", { name: "Check earlier bank setups", exact: true })
    .click();
  await recovery
    .getByRole("button", { name: "Review bank setup", exact: true })
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
  ).toHaveCount(0);
  await expect(
    page.getByText("Technical account details for Bank account 2", {
      exact: true,
    }),
  ).toHaveCount(0);
  await expect(
    page.getByText("private-fixture-item", { exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Inspect attempt", exact: true }),
  ).toHaveCount(0);
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
