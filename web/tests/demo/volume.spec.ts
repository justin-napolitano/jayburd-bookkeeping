import { test, expect } from "playwright/test";

// Bulk acquisition retains 3,000 source observations; it is not a posting runner.
// acquisition.spec.ts performs the one explicit source-linked posting/replay.
test("large demo import remains evidence to review rather than automatic journals", async ({
  page,
}) => {
  await page.goto("/");
  const headers = {
    Authorization: "Bearer synthetic.browser.proof",
    Origin: "http://localhost:39083",
  };
  const initial = await page.request.get("/v1/demo/acquisition", { headers });
  expect(initial.status()).toBe(200);
  const progress = await initial.json();
  if (!progress.complete)
    await page
      .getByRole("button", { name: "Load demo books", exact: true })
      .click();
  await expect(
    page.getByText("3000 of 3000 transactions imported · Ready for review", {
      exact: true,
    }),
  ).toBeVisible({ timeout: 60000 });
  const response = await page.request.get("/v1/demo/acquisition", {
    headers,
  });
  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({
    profile: "synthetic-acquisition-v1",
    total_count: "3000",
    imported_count: "3000",
    complete: true,
  });
  await expect(
    page.getByText("3000 of 3000 transactions imported · Ready for review", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Start volume run", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Review posting command", exact: true }),
  ).toHaveCount(0);
});
