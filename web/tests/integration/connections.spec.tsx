import { chooseFinanceTool } from "../finance-navigation";
import { test, expect } from "@playwright/experimental-ct-react";
import type { Page } from "playwright-core";
import { ConnectionsHarness } from "./ConnectionsHarness";
const scope =
  "10000000-0000-4000-8000-000000000002/10000000-0000-4000-8000-000000000003";
// Real enrollment and sync feed the same explicit source-post flow; only response delivery is faulted.
test("TestSyncReviewPostJourney: enrollment, acquisition, exact posting and lost-response recovery", async ({
  mount,
  page,
}, info) => {
  page.setDefaultTimeout(10000);
  let component = await mount(<ConnectionsHarness />);
  await page.getByLabel("Entity and book").selectOption(scope);
  await chooseFinanceTool(page, "Accounts");
  const connect = page.getByRole("button", {
    name: "Connect bank",
    exact: true,
  });
  // Cancel after reload through the real API/SQL boundary, including a lost reply.
  await expect(connect).toBeEnabled();
  await connect.click();
  await expect(page.getByRole("link", { name: "Open Plaid" })).toBeVisible();
  await component.unmount();
  component = await mount(<ConnectionsHarness />);
  await page.getByLabel("Entity and book").selectOption(scope);
  await chooseFinanceTool(page, "Accounts");
  await page.route("**/connection-attempts/*/abandon", async (route) => {
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
    await route.abort("failed");
  });
  await page.getByRole("button", { name: "Cancel setup", exact: true }).click();
  await expect(connect).toBeEnabled();
  await page.unroute("**/connection-attempts/*/abandon");
  // A completed Link handoff can still be canceled from binding review. The
  // application removes the unbound Item before opening the replacement setup.
  await connect.click();
  await page
    .getByRole("button", { name: "Check bank setup completion" })
    .click();
  await expect(
    page.getByText(/Plaid has not reported completion yet/),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Check bank setup completion" })
    .click();
  await expect(
    page.getByRole("region", { name: "Choose accounts to register" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Review selected accounts" }),
  ).toBeDisabled();
  await page
    .getByRole("button", { name: "Cancel setup and start over" })
    .click();
  await page.getByRole("button", { name: "Confirm cancellation" }).click();
  await expect(page.getByRole("link", { name: "Open Plaid" })).toBeVisible();
  // Lose the exchange response after its commit: reload must discover recovery without a saved UUID.
  let exchangeDropped = false;
  await page.route("**/connection-attempts/*/complete", async (route) => {
    if (exchangeDropped) return route.continue();
    const response = await route.fetch();
    if (!response.ok())
      throw new Error(
        `connection completion failed: ${response.status()} ${await response.text()}`,
      );
    const result = await response.json();
    if (result.state === "link_ready") return route.fulfill({ response });
    exchangeDropped = true;
    await route.abort("failed");
  });
  await expect(page.getByRole("link", { name: "Open Plaid" })).toHaveAttribute(
    "rel",
    "noopener noreferrer",
  );
  await page
    .getByRole("button", { name: "Check bank setup completion" })
    .click();
  await expect(
    page.getByText(/Plaid has not reported completion yet/),
  ).toBeVisible();
  expect(exchangeDropped).toBe(false);
  // Drop the browser-held handoff before Plaid reports completion; discovery must recover it.
  await component.unmount();
  component = await mount(<ConnectionsHarness />);
  await page.getByLabel("Entity and book").selectOption(scope);
  await chooseFinanceTool(page, "Accounts");
  await page
    .getByRole("button", { name: "Check bank setup status", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Check bank setup completion" })
    .click();
  await expect.poll(() => exchangeDropped).toBe(true);
  await component.unmount();
  component = await mount(<ConnectionsHarness />);
  await page.getByLabel("Entity and book").selectOption(scope);
  await chooseFinanceTool(page, "Accounts");
  await expect(
    page.getByRole("button", { name: "Connect bank", exact: true }),
  ).toBeDisabled();
  await page
    .getByRole("button", { name: "Resume account selection", exact: true })
    .click();
  const choices = page.getByRole("region", {
    name: "Choose accounts to register",
  });
  await expect(choices.getByRole("checkbox")).toHaveCount(3);
  await expect(
    page.getByLabel(/^Account code for|^Currency version for|^Binding date$/),
  ).toHaveCount(0);
  // Register one, then append the other shared accounts without a new Item or exchange.
  await choices.getByRole("checkbox").nth(0).check();
  await page.getByRole("button", { name: "Review selected accounts" }).click();
  await page
    .getByRole("button", { name: "Register selected accounts" })
    .click();
  await page.getByRole("button", { name: "Done", exact: true }).click();
  const optionsReply = () =>
    page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname.endsWith("/registration") &&
        response.request().method() === "GET",
    );
  const beforeReply = optionsReply();
  await page
    .getByRole("button", { name: "Add shared accounts", exact: true })
    .click();
  const beforeAppend = await (await beforeReply).json();
  const firstSource = beforeAppend.accounts.find(
    (row: { registered: boolean }) => row.registered,
  );
  expect(firstSource.account_id).toBe("synthetic-flow-account");
  await expect(choices.getByRole("checkbox")).toHaveCount(2);
  for (const checkbox of await choices.getByRole("checkbox").all()) {
    await expect(checkbox).not.toBeChecked();
    await checkbox.check();
  }
  await page.getByRole("button", { name: "Review selected accounts" }).click();
  await page
    .getByRole("button", { name: "Register selected accounts" })
    .click();
  await page.getByRole("button", { name: "Done", exact: true }).click();
  const afterReply = optionsReply();
  await page
    .getByRole("button", { name: "Add shared accounts", exact: true })
    .click();
  const afterAppend = await (await afterReply).json();
  expect(
    afterAppend.accounts.find(
      (row: { account_id: string }) =>
        row.account_id === firstSource.account_id,
    ),
  ).toMatchObject({
    source_account_id: firstSource.source_account_id,
    stream_id: firstSource.stream_id,
    registered: true,
  });
  expect(afterAppend.current_sources).toBe("3");
  await expect(choices.getByRole("checkbox")).toHaveCount(0);
  const review = page.getByRole("button", {
    name: "Review transactions for Bank account 1",
    exact: true,
  });
  await expect(review).toBeVisible();
  await page
    .getByRole("button", {
      name: "View sync status for Bank account 1",
      exact: true,
    })
    .click();
  const status = page.getByRole("region", { name: "Account sync status" });
  await expect(status).toContainText("No completed import yet");
  await expect(status).toContainText("Import not completed yet");
  await page
    .getByRole("button", { name: "Pause account sync", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm pause", exact: true })
    .click();
  await expect(status).toContainText("Account acquisition: paused");
  await page
    .getByRole("button", { name: "Resume account sync", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm resume", exact: true })
    .click();
  await expect(status).toContainText("Account acquisition: active");
  // Resolve an unknown manual-control commit with the original request and key.
  let operationDropped = false;
  let firstKey = "";
  let firstBody = "";
  await page.route("**/connections/streams/*/operations", async (route) => {
    const key = route.request().headers()["idempotency-key"];
    const body = route.request().postData() ?? "";
    if (operationDropped) {
      expect(key).toBe(firstKey);
      expect(body).toBe(firstBody);
      return route.continue();
    }
    firstKey = key;
    firstBody = body;
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
    operationDropped = true;
    await route.abort("failed");
  });
  await page
    .getByRole("button", { name: "Request sync now", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm sync request", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Resolve saved request", exact: true })
    .click();
  await expect(status).toContainText("Sync is queued");
  await expect(status).toContainText("No completed import yet");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({
    path: info.outputPath("connections-mobile.png"),
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  expect(
    await page.evaluate(() => ({
      local: localStorage.length,
      session: sessionStorage.length,
    })),
  ).toEqual({ local: 0, session: 0 });
  const acquire = () =>
    page.request.post(new URL("/__proof/acquire", page.url()).href, {
      data: {},
      headers: {
        Authorization: "Bearer synthetic.browser.proof",
        Origin: "http://localhost:39083",
      },
    });
  const first = await acquire();
  expect(first.ok()).toBe(true);
  expect(await first.json()).toMatchObject({
    fetch_count: 1,
    exchange_count: 2,
    removal_count: 1,
    journal_count: "0",
    created_account_count: "3",
    source_count: "3",
    stream_count: "3",
  });
  await review.click();
  // The production Next bridge reaches the real authorized source summary and exact unit drill.
  const [metricEntity, metricBook] = scope.split("/");
  const metricURL = new URL(
    `/v1/entities/${metricEntity}/books/${metricBook}/source-activity/query`,
    page.url(),
  ).href;
  const metricHeaders = {
    Authorization: "Bearer synthetic.browser.proof",
    Origin: new URL(page.url()).origin,
  };
  const metricQuery = {
    profile: "source-activity-v1",
    source_account_id: firstSource.source_account_id,
    starts_on: "2026-01-01",
    through_date: "2026-12-31",
    search: "",
    kind: "summary",
    currency_version_id: null,
    merchant: null,
    snapshot: null,
    cursor: null,
    limit: 20,
  };
  const metricReply = await page.request.post(metricURL, {
    headers: metricHeaders,
    data: metricQuery,
  });
  expect(metricReply.status()).toBe(200);
  expect(metricReply.headers()["cache-control"]).toBe("no-store");
  const metric = await metricReply.json();
  expect(metric.units).toHaveLength(1);
  expect(metric.units[0].outflow).toBe("0.3");
  expect(metric.bank_balance).toBeNull();
  expect(metric.last_successful_sync).toBeNull();
  const drillReply = await page.request.post(metricURL, {
    headers: metricHeaders,
    data: {
      ...metricQuery,
      kind: "observations",
      currency_version_id: metric.units[0].currency_version_id,
      snapshot: metric.snapshot,
    },
  });
  expect(drillReply.status()).toBe(200);
  const drill = await drillReply.json();
  expect(drill.totals.outflow).toBe("0.3");
  expect(drill.items).toHaveLength(2);
  expect(
    drill.items.map((row: { amount: string }) => row.amount).sort(),
  ).toEqual(["0.1", "0.2"]);
  // The daily Overview renders that same real Next→Go→SQL projection without posting.
  await chooseFinanceTool(page, "Overview");
  await page
    .getByRole("combobox", { name: "Bank account", exact: true })
    .selectOption(firstSource.source_account_id);
  await expect(page.locator(".activity-money dd")).toHaveText([
    "0.00",
    "0.30",
    "−0.30",
  ]);
  await page.getByRole("button", { name: /View USD .* transactions/ }).click();
  await expect(page.locator(".activity-detail tbody tr")).toHaveCount(2);
  // Equal dates use generated UUID ties; compare exact quantities rather than random IDs.
  expect(
    (
      await page
        .locator(".activity-detail .activity-exact-amount")
        .allTextContents()
    ).sort(),
  ).toEqual(["0.10 USD", "0.20 USD"]);
  await page
    .getByRole("button", { name: "Review account transactions", exact: true })
    .click();
  await expect(
    page.getByRole("columnheader", { name: "Date" }),
  ).toHaveAttribute("aria-sort", "descending");
  const sortedReply = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return (
      url.pathname.endsWith("/review-items") &&
      url.searchParams.get("sort") === "amount"
    );
  });
  await page.getByRole("button", { name: "Amount", exact: true }).click();
  const sortedPage = await (await sortedReply).json();
  expect(sortedPage).toMatchObject({
    sort: "amount",
    order: "asc",
    start_index: "1",
  });
  expect(BigInt(sortedPage.matching_count)).toBeGreaterThan(0n);
  await page
    .getByRole("button", { name: "Review Fictional purchase A", exact: true })
    .click();
  await page
    .getByRole("combobox", { name: "Category account", exact: true })
    .selectOption("10000000-0000-4000-8000-000000000012");
  await page
    .getByRole("button", { name: "Create reviewed draft", exact: true })
    .click();
  await expect(
    page
      .getByRole("table", { name: "Exact proposed journal lines" })
      .locator("tbody tr td:nth-child(3)"),
  ).toHaveText(["0.1 USD", "0.1 USD"]);
  await page
    .getByRole("button", { name: "Review posting command", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Post journal", exact: true }),
  ).toBeEnabled();
  let dropped = false;
  let sourceCommand = "";
  let sourceJournal = "";
  await page.route("**/v1/commands/*/execute", async (route) => {
    if (dropped) return route.continue();
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
    const saved = await response.json();
    expect(saved.operation).toBe("source_post");
    expect(saved.state).toBe("committed");
    sourceCommand = saved.command_id;
    sourceJournal = saved.result.journal_id;
    expect(sourceJournal).toMatch(/^[0-9a-f-]{36}$/);
    dropped = true;
    await route.abort("failed");
  });
  await page.getByRole("button", { name: "Post journal", exact: true }).click();
  await expect.poll(() => dropped).toBe(true);
  await component.unmount();
  component = await mount(<ConnectionsHarness />);
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await page.getByLabel("Entity and book").selectOption(scope);
  await chooseFinanceTool(page, "Chart of accounts");
  await expect(page.locator("tfoot td")).toHaveText(["0.10", "0.10"]);
  const commandReplay = await page.request.post(
    new URL(`/v1/commands/${sourceCommand}/execute`, page.url()).href,
    {
      headers: {
        Authorization: "Bearer synthetic.browser.proof",
        Origin: new URL(page.url()).origin,
      },
      data: {},
    },
  );
  expect(commandReplay.status()).toBe(200);
  expect(await commandReplay.json()).toMatchObject({
    command_id: sourceCommand,
    operation: "source_post",
    state: "committed",
    acknowledged: true,
    result: { journal_id: sourceJournal },
  });

  const replay = await acquire();
  expect(replay.ok()).toBe(true);
  expect(await replay.json()).toMatchObject({
    fetch_count: 1,
    exchange_count: 2,
    removal_count: 1,
    journal_count: "1",
  });
  await chooseFinanceTool(page, "Accounts");
  await page
    .getByRole("button", {
      name: "Review transactions for Bank account 1",
      exact: true,
    })
    .click();
  await page.getByLabel("Review status").selectOption("posted");
  await page
    .getByRole("button", { name: "Review Fictional purchase A", exact: true })
    .click();
  await expect(page.getByText(/^Posted journal:/)).toContainText(sourceJournal);
  await page
    .getByRole("button", { name: "View posted journal", exact: true })
    .click();
  await expect(
    page
      .getByRole("table", {
        name: "Immutable posted journal lines",
        exact: true,
      })
      .locator("tbody tr td:nth-child(3)"),
  ).toHaveText(["0.1 USD", "0.1 USD"]);
  await verifySourcePostingReports(page, sourceJournal);
  expect(
    await page.evaluate(() => ({
      local: localStorage.length,
      session: sessionStorage.length,
    })),
  ).toEqual({ local: 0, session: 0 });
});

// Independent 0.10 outflow→expense oracles replace the obsolete unlinked
// 125.50 manual-entry report journey. Every report reads the one source posting.
async function verifySourcePostingReports(page: Page, journal: string) {
  await chooseFinanceTool(page, "Reports");
  const reports = page.getByRole("region", { name: "Financial reports" });
  await reports
    .getByLabel("Report through date (inclusive)")
    .fill("2026-12-31");
  let snapshot = "";
  for (const kind of [
    "trial-balance",
    "balance-sheet",
    "profit-and-loss",
    "general-ledger",
  ]) {
    await reports.getByLabel("Report type").selectOption(kind);
    if (["profit-and-loss", "general-ledger"].includes(kind)) {
      await reports.getByLabel("Report starts on").fill("2026-01-01");
      await reports.getByLabel("Report ends before").fill("2027-01-01");
    }
    if (kind === "general-ledger")
      await reports
        .getByLabel("Report account")
        .selectOption("10000000-0000-4000-8000-000000000012");
    const response = page.waitForResponse((r) =>
      r.url().includes(`/reports/${kind}?`),
    );
    await reports.getByRole("button", { name: "Run report" }).click();
    const actual = await response;
    expect(actual.status()).toBe(200);
    const body = await actual.json();
    expect(body.functional_currency).toBe("USD");
    if (!snapshot) snapshot = body.snapshot_revision;
    expect(body.snapshot_revision).toBe(snapshot);
    await reports.getByText("Report details", { exact: true }).click();
    await expect(
      reports.getByText(new RegExp(`Snapshot ${snapshot} ·`)),
    ).toBeVisible();
    await expect(
      reports
        .locator("table")
        .first()
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
    ).toHaveText(
      ["balance-sheet", "profit-and-loss"].includes(kind) ? "-0.10" : "0.10",
    );
    if (kind === "general-ledger")
      await expect(
        reports.getByRole("cell", { name: /0.1 USD/ }),
      ).toBeVisible();
  }
  const journalResponse = page.waitForResponse((r) =>
    r.url().includes("/reports/general-ledger/journals/"),
  );
  await reports
    .getByRole("button", { name: "Review journal line 2", exact: true })
    .click();
  const reply = await journalResponse;
  expect(reply.status()).toBe(200);
  expect(new URL(reply.url()).pathname).toContain(journal);
  const detail = reports.getByRole("region", { name: "Journal detail" });
  await expect(detail).toContainText("Fictional purchase A");
  await expect(detail).toContainText(`Report snapshot ${snapshot}`);
  await expect(detail.locator("tbody tr")).toHaveCount(2);
  await reports.getByRole("button", { name: "Close journal detail" }).click();
  await reports.getByLabel("Report type").selectOption("trial-balance");
  await reports.getByLabel("Compare another period").check();
  await reports
    .getByLabel("Comparison through date (inclusive)")
    .fill("2026-01-01");
  await reports.getByRole("button", { name: "Run report" }).click();
  await expect(
    reports
      .locator("table")
      .first()
      .getByRole("row")
      .filter({ hasText: "Total debits" })
      .locator("td"),
  ).toHaveText(["0.10", "0.00", "0.10"]);
  const exportResponse = page.waitForResponse((r) =>
    r.url().includes("/export.csv?"),
  );
  const download = page.waitForEvent("download");
  await reports.getByRole("button", { name: "Download review CSV" }).click();
  expect((await exportResponse).status()).toBe(200);
  const downloaded = await download;
  expect(downloaded.suggestedFilename()).toBe("financial-reporting-review.csv");
  const stream = await downloaded.createReadStream();
  expect(stream).not.toBeNull();
  let csv = "";
  for await (const chunk of stream!) csv += chunk.toString("utf8");
  expect(csv).toContain(
    '"total_debits","positive","0.1","zero","0","positive","0.1"\r\n',
  );
  await reports
    .getByRole("button", { name: "Review account 5000", exact: true })
    .click();
  await reports.getByLabel("Drill starts on").fill("2026-01-01");
  await reports.getByLabel("Drill ends before").fill("2027-01-01");
  const navigation = page.waitForResponse((r) =>
    r.url().includes("/reports/general-ledger?"),
  );
  await reports.getByRole("button", { name: "Open account ledger" }).click();
  const navigationResult = await navigation;
  expect(navigationResult.status()).toBe(200);
  expect(
    new URL(navigationResult.url()).searchParams.get("snapshot_revision"),
  ).toBe(snapshot);
  await expect(
    reports
      .locator("table")
      .first()
      .getByRole("row")
      .filter({ hasText: "Closing balance" })
      .locator("td"),
  ).toHaveText("0.10");
}
