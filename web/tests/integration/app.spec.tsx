import { test, expect } from "@playwright/experimental-ct-react";
import { IntegrationHarness } from "./Harness";

/** Real browser forms traverse Next→Go→PostgreSQL; only response delivery is fault-injected. */
test("real setup and lost committed journal recover after remount with exactly one posting", async ({
  mount,
  page,
}) => {
  let component = await mount(<IntegrationHarness />);
  const workspaceURL = new URL("/v1/workspace", page.url()).href;
  expect((await page.request.get(workspaceURL)).status()).toBe(401);
  expect(
    (
      await page.request.get(workspaceURL, {
        headers: {
          Authorization: "Bearer synthetic.browser.proof",
          Origin: "https://foreign.example.invalid",
        },
      })
    ).status(),
  ).toBe(403);
  await expect(
    page.getByText(
      "No books are available. Create your first development book below.",
    ),
  ).toBeVisible();
  await page.getByText("Create an entity and book", { exact: true }).click();
  await page
    .getByLabel("Entity name", { exact: true })
    .fill("Synthetic browser LLC");
  await page.getByLabel("Book name", { exact: true }).fill("Operating");
  await page.getByLabel("Period starts on").fill("2026-01-01");
  await page.getByLabel("Period ends before").fill("2027-01-01");
  await page
    .getByLabel("Functional currency metadata")
    .selectOption("00000000-0000-4000-8000-000000000002");
  for (let i = 1; i <= 2; i++) {
    await page
      .getByLabel(`Account ${i} code`, { exact: true })
      .fill(i === 1 ? "1000" : "3000");
    await page
      .getByLabel(`Account ${i} name`, { exact: true })
      .fill(i === 1 ? "Cash" : "Capital");
  }
  await page
    .getByRole("button", { name: "Review book setup", exact: true })
    .click();
  const setupResponse = page.waitForResponse(
    (r) => r.url().endsWith("/execute") && r.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Create book", exact: true }).click();
  expect((await setupResponse).status()).toBe(200);
  await expect(
    page.getByText("Book creation confirmed", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await page
    .getByLabel("Entity and book")
    .selectOption({ label: "Synthetic browser LLC / Operating" });
  await page.getByRole("button", { name: "New journal", exact: true }).click();
  await page.getByLabel("Accounting date", { exact: true }).fill("2026-01-15");
  await page
    .getByRole("combobox", { name: "Policy", exact: true })
    .selectOption({ index: 1 });
  await page
    .getByLabel("Description", { exact: true })
    .fill("Synthetic initial capital");
  for (let i = 1; i <= 2; i++) {
    await page
      .getByLabel(`Line ${i} account`, { exact: true })
      .selectOption({ label: i === 1 ? "1000 · Cash" : "3000 · Capital" });
    await page
      .getByLabel(`Line ${i} original amount`, { exact: true })
      .fill("125.50");
    await page
      .getByLabel(`Line ${i} functional amount`, { exact: true })
      .fill("125.50");
  }
  await page
    .getByRole("button", { name: "Review journal", exact: true })
    .click();
  let lostCommand = "";
  let executeCalls = 0;
  await page.route("**/v1/commands/*/execute", async (route) => {
    ++executeCalls;
    const response = await route.fetch(); // The real transaction commits before the network loss.
    expect(response.status()).toBe(200);
    const command = await response.json();
    expect(command.state).toBe("committed");
    lostCommand = command.command_id;
    await route.abort("failed");
  });
  await page.getByRole("button", { name: "Post journal", exact: true }).click();
  await expect.poll(() => lostCommand).not.toBe("");
  await component.unmount();
  await page.unroute("**/v1/commands/*/execute");
  component = await mount(<IntegrationHarness />);
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText(lostCommand, { exact: true })).toBeVisible();
  expect(executeCalls).toBe(1); // Recovery performs discovery, not a new execution.
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await page
    .getByLabel("Entity and book")
    .selectOption({ label: "Synthetic browser LLC / Operating" });
  // Select the preserved trial-balance control, not the new report date field.
  await page.getByLabel("Through date", { exact: true }).fill("2026-01-31");
  await page
    .getByRole("button", { name: "Load trial balance", exact: true })
    .click();
  await expect(page.locator("tfoot td")).toHaveText(["125.50", "125.50"]);
  await expect(
    page.locator("tbody tr").filter({ hasText: "1000" }),
  ).toContainText("125.5");
  await expect(
    page.locator("tbody tr").filter({ hasText: "3000" }),
  ).toContainText("125.5");
  // Read the actual committed journal through every new report screen without adding posts.
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  const reports = page.getByRole("region", { name: "Financial reports" });
  await reports
    .getByLabel("Report through date (inclusive)")
    .fill("2026-01-31");
  for (const kind of [
    "trial-balance",
    "balance-sheet",
    "profit-and-loss",
    "general-ledger",
  ]) {
    await reports.getByLabel("Report type").selectOption(kind);
    if (["profit-and-loss", "general-ledger"].includes(kind)) {
      await reports.getByLabel("Report starts on").fill("2026-01-01");
      await reports.getByLabel("Report ends before").fill("2026-02-01");
    }
    if (kind === "general-ledger")
      await reports
        .getByLabel("Report account")
        .selectOption({ label: "1000 · Cash" });
    const response = page.waitForResponse((r) =>
      r.url().includes(`/reports/${kind}?`),
    );
    await reports.getByRole("button", { name: "Refresh report" }).click();
    expect((await response).status()).toBe(200);
    // Controlled setup advances revision before the one posted journal; revisions are not journal counts.
    await expect(reports.getByText(/Snapshot 5 ·/)).toBeVisible();
    const totals = reports.locator("table").first();
    await expect(
      totals
        .getByRole("row")
        .filter({
          hasText:
            kind === "trial-balance"
              ? "Total debits"
              : kind === "balance-sheet"
                ? "Assets"
                : kind === "profit-and-loss"
                  ? "Net income"
                  : "Closing balance",
        })
        .locator("td"),
    ).toHaveText(kind === "profit-and-loss" ? "0.00" : "125.50");
    if (kind === "general-ledger")
      await expect(
        reports.getByRole("cell", { name: /125.5 USD/ }),
      ).toBeVisible();
  }
  // Follow the actual GL line through the pinned authorized journal wrapper.
  const journalResponse = page.waitForResponse((r) =>
    r.url().includes("/reports/general-ledger/journals/"),
  );
  await reports
    .getByRole("button", { name: "Review journal line 1", exact: true })
    .click();
  expect((await journalResponse).status()).toBe(200);
  const detail = reports.getByRole("region", { name: "Journal detail" });
  await expect(detail).toContainText("Synthetic initial capital");
  await expect(detail).toContainText("Report snapshot 5");
  await expect(detail.locator("tbody tr")).toHaveCount(2);
  await reports.getByRole("button", { name: "Close journal detail" }).click();
  await reports.getByLabel("Report type").selectOption("trial-balance");
  await reports.getByLabel("Compare another period").check();
  await reports
    .getByLabel("Comparison through date (inclusive)")
    .fill("2026-01-01");
  await reports.getByRole("button", { name: "Refresh report" }).click();
  await expect(
    reports
      .locator("table")
      .first()
      .getByRole("row")
      .filter({ hasText: "Total debits" })
      .locator("td"),
  ).toHaveText(["125.50", "0.00", "125.50"]);
  const exportResponse = page.waitForResponse((r) =>
    r.url().includes("/export.csv?"),
  );
  const download = page.waitForEvent("download");
  await reports.getByRole("button", { name: "Download review CSV" }).click();
  expect((await exportResponse).status()).toBe(200);
  const downloaded = await download;
  expect(downloaded.suggestedFilename()).toBe("financial-reporting-review.csv");
  // Check the actual download bytes after the bridge/client, not only its filename.
  const stream = await downloaded.createReadStream();
  expect(stream).not.toBeNull();
  let csv = "";
  for await (const chunk of stream!) csv += chunk.toString("utf8");
  expect(csv).toContain(
    '"total_debits","positive","125.5","zero","0","positive","125.5"\r\n',
  );
  await reports
    .getByRole("button", { name: "Review account 1000", exact: true })
    .click();
  await reports.getByLabel("Drill starts on").fill("2026-01-01");
  await reports.getByLabel("Drill ends before").fill("2026-02-01");
  const navigation = page.waitForResponse((r) =>
    r.url().includes("/reports/general-ledger?"),
  );
  await reports.getByRole("button", { name: "Open account ledger" }).click();
  const navigationResult = await navigation;
  expect(navigationResult.status()).toBe(200);
  expect(
    new URL(navigationResult.url()).searchParams.get("snapshot_revision"),
  ).toBe("5");
  await expect(
    reports
      .locator("table")
      .first()
      .getByRole("row")
      .filter({ hasText: "Closing balance" })
      .locator("td"),
  ).toHaveText("125.50");
});
