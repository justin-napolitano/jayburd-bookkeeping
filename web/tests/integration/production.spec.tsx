import { test, expect } from "@playwright/experimental-ct-react";
import { ConnectionsHarness } from "./ConnectionsHarness";
const scope =
  "10000000-0000-4000-8000-000000000002/10000000-0000-4000-8000-000000000003";

// Independent literal expectations cover the actual Production adapters with fictional external capabilities.
test("TestProductionProfileJourney: one binding, history, recovery and authorized bookkeeping", async ({
  mount,
  page,
}) => {
  page.setDefaultTimeout(15000);
  let component = await mount(<ConnectionsHarness />);
  const connections = async () => {
    await page.getByLabel("Entity and book").selectOption(scope);
    await page.getByText("Tools", { exact: true }).click();
    await page
      .getByRole("button", { name: "Connections", exact: true })
      .click();
  };
  const trigger = (path: string) =>
    page.request.post(new URL(path, page.url()).href, {
      data: {},
      headers: {
        Authorization: "Bearer synthetic.browser.proof",
        Origin: "http://localhost:39083",
      },
    });
  await connections();
  await page
    .getByRole("button", { name: "Register accounts", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Check bank setup completion" })
    .click();
  await expect(
    page.getByText(/Plaid has not reported completion yet/),
  ).toBeVisible();
  // Lose the reply after real completion committed. Reload must find the original attempt.
  let completionDropped = false;
  await page.route("**/connection-attempts/*/complete", async (route) => {
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
    await route.abort("failed");
    completionDropped = true;
  });
  await page
    .getByRole("button", { name: "Check bank setup completion" })
    .click();
  // Await the injected failure before unmount/unroute can cancel its active handler.
  await expect.poll(() => completionDropped).toBe(true);
  await component.unmount();
  component = await mount(<ConnectionsHarness />);
  await connections();
  await page.unroute("**/connection-attempts/*/complete");
  await page
    .getByRole("button", { name: "Resume account selection", exact: true })
    .click();
  const choices = page.getByRole("region", {
    name: "Choose accounts to register",
  });
  await expect(choices.getByRole("checkbox")).toHaveCount(1);
  await expect(
    page.getByLabel(/^Account code for|^Currency version for|^Binding date$/),
  ).toHaveCount(0);
  for (const checkbox of await choices.getByRole("checkbox").all()) {
    await expect(checkbox).not.toBeChecked();
    await checkbox.check();
  }
  await page.getByRole("button", { name: "Review selected accounts" }).click();
  await page
    .getByRole("button", { name: "Register selected accounts" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Account registration confirmed" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Done", exact: true }).click();
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
  await page
    .getByRole("button", { name: "Pause account sync", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm pause", exact: true })
    .click();
  await expect(status).toContainText("Account acquisition: paused");
  expect((await trigger("/__proof/denied")).ok()).toBe(true);
  await page
    .getByRole("button", { name: "Resume account sync", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm resume", exact: true })
    .click();
  const empty = await trigger("/__proof/acquire");
  expect(empty.ok()).toBe(true);
  expect(await empty.json()).toMatchObject({
    fetch_count: 1,
    exchange_count: 1,
    source_count: "1",
    stream_count: "1",
    journal_count: "0",
  });
  const emptyReplay = await trigger("/__proof/acquire");
  expect(await emptyReplay.json()).toMatchObject({
    fetch_count: 1,
    journal_count: "0",
  });
  const history = await trigger("/__proof/history");
  expect(history.ok()).toBe(true);
  expect(await history.json()).toMatchObject({
    fetch_count: 3,
    exchange_count: 1,
    journal_count: "0",
  });
  const again = await trigger("/__proof/history");
  expect(await again.json()).toMatchObject({
    fetch_count: 3,
    journal_count: "0",
  });
  await review.click();
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
  await page.getByText("Tools", { exact: true }).click();
  await page.getByRole("button", { name: "Accounts", exact: true }).click();
  await expect(page.locator("tfoot td")).toHaveText(["0.10", "0.10"]);
  expect(await (await trigger("/__proof/history")).json()).toMatchObject({
    fetch_count: 3,
    exchange_count: 1,
    journal_count: "1",
  });
  await connections();
  await page.getByText("Manage Bank account 1", { exact: true }).click();
  await page
    .getByRole("button", {
      name: "Disconnect Bank account 1",
      exact: true,
    })
    .click();
  await page
    .getByRole("button", { name: "Confirm disconnect", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Disconnect Bank account 1",
      exact: true,
    }),
  ).toBeDisabled();
  expect(await (await trigger("/__proof/denied")).json()).toEqual({
    denied_before_fetch: true,
  });
  expect(
    await page.evaluate(() => ({
      local: localStorage.length,
      session: sessionStorage.length,
    })),
  ).toEqual({ local: 0, session: 0 });
});
