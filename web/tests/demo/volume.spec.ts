import { test, expect } from "playwright/test";

// Exercise one thousand real command lifecycles; fixed oracle is independent of UI generation.
test("one thousand committed deposits have exact cents and survive a ledger read", async ({
  page,
}) => {
  test.setTimeout(300000);
  await page.goto("/");
  await page
    .getByRole("button", { name: "Run thousands", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Start volume run", exact: true })
    .click();
  await expect(page.getByRole("status")).toHaveText(
    /Volume run verified|Stopped for review/,
    { timeout: 270000 },
  );
  await expect(
    page.getByText("Volume run verified", { exact: true }),
  ).toBeVisible({ timeout: 270000 });
  await expect(
    page.getByText(
      "Actual bank debit and capital credit from PostgreSQL: USD 495.55.",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(
    page.getByRole("progressbar", { name: "Committed transactions" }),
  ).toHaveAttribute("value", "1000");
});
