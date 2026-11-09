import { test, expect } from "playwright/test";
import { chooseFinanceTool } from "../finance-navigation";

// Historical synthetic mathematical fixtures remain read-only; they are not invented source artifacts.
test("historical review books preserve independent exact-cent oracles without manual posting", async ({
  page,
}) => {
  await page.goto("/");
  await expect(
    page.getByText("JAYBURD / SYNTHETIC DEMO", { exact: true }),
  ).toBeVisible();
  await page.getByLabel("Entity and book").selectOption({
    label:
      "Fictional Generator Trading LLC / Synthetic review — USD functional",
  });
  await chooseFinanceTool(page, "Chart of accounts");
  await page.getByLabel("Through date", { exact: true }).fill("2026-12-31");
  await page
    .getByRole("button", { name: "Load trial balance", exact: true })
    .click();
  await expect(
    page.locator("tbody tr").filter({ hasText: "1000" }).locator("td"),
  ).toHaveText(["5432.29", "0.00"]);
  await expect(page.locator("tfoot td")).toHaveText(["5749.26", "5749.26"]);
  await page.getByLabel("Entity and book").selectOption({
    label: "Fictional Cents Lab LLC / Synthetic review — USD functional",
  });
  await chooseFinanceTool(page, "Chart of accounts");
  await page.getByLabel("Through date", { exact: true }).fill("2026-12-31");
  await page
    .getByRole("button", { name: "Load trial balance", exact: true })
    .click();
  await expect(
    page.locator("tbody tr").filter({ hasText: "1000" }).locator("td"),
  ).toHaveText(["0.29", "0.00"]);
  await expect(page.locator("tfoot td")).toHaveText(["0.30", "0.30"]);
  await expect(
    page.getByRole("button", { name: "New journal", exact: true }),
  ).toHaveCount(0);
});
