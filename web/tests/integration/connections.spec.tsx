import { test, expect } from "@playwright/experimental-ct-react";
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
  await page.getByRole("button", { name: "Connections", exact: true }).click();
  const connect = page.getByRole("button", {
    name: "Connect account",
    exact: true,
  });
  // Cancel after reload through the real API/SQL boundary, including a lost reply.
  await expect(connect).toBeEnabled();
  await connect.click();
  await expect(
    page.getByRole("link", { name: "Open Plaid Sandbox" }),
  ).toBeVisible();
  await component.unmount();
  component = await mount(<ConnectionsHarness />);
  await page.getByLabel("Entity and book").selectOption(scope);
  await page.getByRole("button", { name: "Connections", exact: true }).click();
  await page.route("**/connection-attempts/*/abandon", async (route) => {
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
    await route.abort("failed");
  });
  await page.getByRole("button", { name: "Cancel setup", exact: true }).click();
  await expect(connect).toBeEnabled();
  await page.unroute("**/connection-attempts/*/abandon");
  // Lose the exchange response after its commit: reload must discover recovery without a saved UUID.
  let exchangeDropped = false;
  await page.route("**/connection-attempts/*/complete", async (route) => {
    if (exchangeDropped) return route.continue();
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
    const result = await response.json();
    if (result.state === "link_ready") return route.fulfill({ response });
    exchangeDropped = true;
    await route.abort("failed");
  });
  await expect(connect).toBeEnabled();
  await connect.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("link", { name: "Open Plaid Sandbox" }),
  ).toHaveAttribute("rel", "noopener noreferrer");
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
  await page.getByRole("button", { name: "Connections", exact: true }).click();
  await page
    .getByRole("listitem")
    .filter({ hasText: "new — link ready" })
    .getByRole("button", { name: /^Review attempt / })
    .click();
  await page
    .getByRole("button", { name: "Check bank setup completion" })
    .click();
  await expect.poll(() => exchangeDropped).toBe(true);
  await component.unmount();
  component = await mount(<ConnectionsHarness />);
  await page.getByLabel("Entity and book").selectOption(scope);
  await page.getByRole("button", { name: "Connections", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Connect account", exact: true }),
  ).toBeDisabled();
  await page
    .getByRole("listitem")
    .filter({ hasText: "new — binding review" })
    .getByRole("button", { name: /^Review attempt / })
    .click();
  await expect(
    page.getByRole("group", { name: "Review verified account bindings" }),
  ).toBeVisible();
  await page.getByLabel("Binding date").fill("2026-10-13");
  await expect(page.getByLabel(/^Ledger account for/)).toHaveCount(3);
  await expect(page.getByLabel(/^Account code for/)).toHaveCount(3);
  await page.getByRole("checkbox").check();
  await page.getByRole("button", { name: "Confirm selected bindings" }).click();
  const review = page.getByRole("button", {
    name: "Review transactions for synthetic-flow-account",
    exact: true,
  });
  await expect(review).toBeVisible();
  await page
    .getByRole("button", {
      name: "View sync status for synthetic-flow-account",
      exact: true,
    })
    .click();
  const status = page.getByRole("region", { name: "Account sync status" });
  await expect(status).toContainText("None recorded");
  await expect(status).toContainText("Unknown");
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
  await expect(status).toContainText("Sync work is queued");
  await expect(status).toContainText("None recorded");
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
    exchange_count: 1,
    journal_count: "0",
    created_account_count: "3",
    source_count: "3",
    stream_count: "3",
  });
  await review.click();
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
  await page.route("**/v1/commands/*/execute", async (route) => {
    if (dropped) return route.continue();
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
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
  await expect(page.locator("tfoot td")).toHaveText(["0.10", "0.10"]);
  const replay = await acquire();
  expect(replay.ok()).toBe(true);
  expect(await replay.json()).toMatchObject({
    fetch_count: 1,
    exchange_count: 1,
    journal_count: "1",
  });
  await page.getByRole("button", { name: "Connections", exact: true }).click();
  await page
    .getByRole("button", {
      name: "Review transactions for synthetic-flow-account",
      exact: true,
    })
    .click();
  await page.getByLabel("Review status").selectOption("posted");
  await page
    .getByRole("button", { name: "Review Fictional purchase A", exact: true })
    .click();
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
  expect(
    await page.evaluate(() => ({
      local: localStorage.length,
      session: sessionStorage.length,
    })),
  ).toEqual({ local: 0, session: 0 });
});
