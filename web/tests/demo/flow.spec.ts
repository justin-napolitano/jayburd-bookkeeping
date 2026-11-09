import { test, expect } from "playwright/test";

// The deployed demo offers the same retained-source workflow as the application.
// Low-level manual ledger mathematics remain separate internal component proofs.
test("demo offers imported evidence review without standalone manual entry runners", async ({
  page,
}) => {
  await page.goto("/");
  await expect(
    page.getByText("JAYBURD / SYNTHETIC DEMO", { exact: true }),
  ).toBeVisible();
  for (const name of [
    "Manual books",
    "Watch a few transactions",
    "Run thousands",
    "New journal",
    "Start volume run",
  ]) {
    await expect(page.getByRole("button", { name, exact: true })).toHaveCount(
      0,
    );
  }
  await expect(page.getByLabel("Accounting date", { exact: true })).toHaveCount(
    0,
  );
  await expect(
    page.getByRole("button", { name: /Load demo books|Demo books loaded/ }),
  ).toBeVisible();
});
