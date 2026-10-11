import { test, expect } from "playwright/test";

// Watch real command boundaries, including a safe replay and rejected unbalanced journal.
test("guided flow posts exact cents, replays once and rejects imbalance", async ({
  page,
}) => {
  test.setTimeout(90000);
  await page.goto("/");
  await page
    .getByRole("button", { name: "Watch a few transactions", exact: true })
    .click();
  await page.getByRole("button", { name: "Next step", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Next step", exact: true }),
  ).toBeEnabled();
  await page
    .getByRole("button", { name: "Run continuously", exact: true })
    .click();
  await expect(page.getByTestId("flow-result")).toBeVisible({ timeout: 60000 });
  await expect(page.getByText(/^Replay verified/)).toBeVisible();
  await expect(page.getByText(/^Expected rejection confirmed/)).toBeVisible();
});
