import { test, expect } from "@playwright/experimental-ct-react";
import { PlaidLinkHarness } from "./PlaidLinkHarness";
const sdk = "https://cdn.plaid.com/link/v2/stable/link-initialize.js";
// All provider bytes are intercepted; this proves browser lifecycle, not Plaid qualification.
const syntheticSDK = `window.Plaid = { create(config) {
  window.addEventListener('synthetic-plaid-success', () => config.onSuccess('synthetic-public'));
  return { open() { config.onLoad(); }, exit() { config.onExit(null); }, destroy() { document.body.dataset.plaidDestroyed = 'yes'; } };
} };`;
test("loads only after explicit connect and destroys after memory-only handoff", async ({
  mount,
  page,
}) => {
  let requests = 0;
  await page.route(sdk, async (route) => {
    requests++;
    await route.fulfill({
      contentType: "application/javascript",
      body: syntheticSDK,
    });
  });
  const view = await mount(<PlaidLinkHarness />);
  await expect(view.getByRole("status")).toHaveText("idle");
  expect(requests).toBe(0);
  await view.getByRole("button", { name: "Connect synthetic bank" }).click();
  await expect.poll(() => requests).toBe(1);
  await page.waitForFunction(() =>
    Boolean((window as Window & { Plaid?: unknown }).Plaid),
  );
  await page.evaluate(() =>
    window.dispatchEvent(new Event("synthetic-plaid-success")),
  );
  await expect(view.getByRole("status")).toHaveText("handoff ready");
  await expect(page.locator("body")).toHaveAttribute(
    "data-plaid-destroyed",
    "yes",
  );
  expect(
    await page.evaluate(() => localStorage.length + sessionStorage.length),
  ).toBe(0);
});
test("cancellation fences a late success callback", async ({ mount, page }) => {
  await page.route(sdk, (route) =>
    route.fulfill({
      contentType: "application/javascript",
      body: syntheticSDK,
    }),
  );
  const view = await mount(<PlaidLinkHarness />);
  await view.getByRole("button", { name: "Connect synthetic bank" }).click();
  await page.waitForFunction(() =>
    Boolean((window as Window & { Plaid?: unknown }).Plaid),
  );
  await view.getByRole("button", { name: "Cancel session" }).click();
  await page.evaluate(() =>
    window.dispatchEvent(new Event("synthetic-plaid-success")),
  );
  await expect(view.getByRole("status")).toHaveText("cancelled");
  await expect(page.locator("body")).toHaveAttribute(
    "data-plaid-destroyed",
    "yes",
  );
});
