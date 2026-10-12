import { test, expect } from "playwright/test";

// Browser checks independent cent oracles against the real seeded ledger, then posts one cent.
test("generated review books preserve cents and recover a posted result after reload", async ({
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
  await page.getByLabel("Through date").fill("2026-12-31");
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
  await page.getByLabel("Through date").fill("2026-12-31");
  await page
    .getByRole("button", { name: "Load trial balance", exact: true })
    .click();
  await expect(
    page.locator("tbody tr").filter({ hasText: "1000" }).locator("td"),
  ).toHaveText(["0.29", "0.00"]);
  await expect(page.locator("tfoot td")).toHaveText(["0.30", "0.30"]);
  await page.getByLabel("Accounting date", { exact: true }).fill("2026-02-01");
  await page
    .getByRole("combobox", { name: "Policy", exact: true })
    .selectOption({ index: 1 });
  await page
    .getByLabel("Description", { exact: true })
    .fill("Synthetic one-cent review deposit");
  for (const i of [1, 2]) {
    await page.getByLabel(`Line ${i} account`, { exact: true }).selectOption({
      label: i === 1 ? "1000 · Bank" : "2000 · Owner capital",
    });
    await page
      .getByLabel(`Line ${i} original amount`, { exact: true })
      .fill("0.01");
    await page
      .getByLabel(`Line ${i} functional amount`, { exact: true })
      .fill("0.01");
  }
  await page
    .getByRole("button", { name: "Review journal", exact: true })
    .click();
  await page.getByRole("button", { name: "Post journal", exact: true }).click();
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await page.getByLabel("Entity and book").selectOption({
    label: "Fictional Cents Lab LLC / Synthetic review — USD functional",
  });
  await page.getByLabel("Through date").fill("2026-12-31");
  await page
    .getByRole("button", { name: "Load trial balance", exact: true })
    .click();
  await expect(
    page.locator("tbody tr").filter({ hasText: "1000" }).locator("td"),
  ).toHaveText(["0.30", "0.00"]);
  await expect(page.locator("tfoot td")).toHaveText(["0.31", "0.31"]);
});
