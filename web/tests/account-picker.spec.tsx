import { test, expect } from "@playwright/experimental-ct-react";
import { AccountPickerHarness } from "./AccountPickerHarness";
import { pickerOptions } from "./registration-options.fixture";
import { savedRegistration } from "./registration.fixture";
import type { DurableCommand } from "../src/bookkeeping/types";
const id = (n: number) =>
  `00000000-0000-4000-8000-${String(n).padStart(12, "0")}`;

// Mocked delivery loss proves browser recovery, not database authority; actual SQL has separate journey coverage.
test("picker selects explicit accounts, previews exact saved work and recovers lost responses once", async ({
  mount,
  page,
}, info) => {
  let command: DurableCommand | null = null,
    options = structuredClone(pickerOptions),
    prepares = 0,
    executes = 0,
    acks = 0;
  await page.route("**/v1/**", async (route) => {
    const req = route.request(),
      path = new URL(req.url()).pathname;
    if (path === "/v1/commands/pending")
      return route.fulfill({
        json: { command: command?.acknowledged ? null : command },
      });
    if (path.endsWith("/registration") && req.method() === "GET")
      return route.fulfill({ json: options });
    if (path.endsWith("/registration")) {
      prepares++;
      const selection = JSON.parse(req.postData()!).account_ids;
      expect(selection).toEqual(["private-card-a", "private-card-b"]);
      expect(req.headers()["idempotency-key"]).toBeTruthy();
      const body = savedRegistration.body;
      if (!("bindings" in body)) throw Error("synthetic registration body");
      command = {
        ...savedRegistration,
        body: {
          ...body,
          selection,
          initial: false,
          bindings: selection.map((account: string, index: number) => ({
            ...body.bindings[0],
            account_id: account,
            ledger_account_id: id(90 + index),
            currency_version_id: index === 0 ? id(8) : id(50),
            new_ledger_account: {
              account_id: id(90 + index),
              code: String(2100 + index),
              name: `Credit card ${index + 2}`,
              category: "liability",
              normal_side: "credit",
              monetary: true,
              active_from: "2026-11-02",
            },
          })),
        },
      };
      return route.abort("failed");
    }
    if (path.endsWith("/execute")) {
      executes++;
      command = {
        ...command!,
        state: "committed",
        result: {
          entity_id: pickerOptions.entity_id,
          book_id: pickerOptions.book_id,
          sync_item_id: pickerOptions.sync_item_id,
          streams: [],
        },
      };
      options = {
        ...options,
        can_register: false,
        current_sources: "3",
        remaining_sources: "0",
        blocked_reason: "capacity_exhausted",
        accounts: options.accounts.map((row, index) =>
          row.eligible
            ? {
                ...row,
                eligible: false,
                registered: true,
                reason: "already_registered",
                stream_id: id(200 + index),
                source_account_id: id(220 + index),
                source_state: "active",
              }
            : row,
        ),
      };
      return route.abort("failed");
    }
    if (path.endsWith("/acknowledge")) {
      acks++;
      command = { ...command!, acknowledged: true };
      return route.fulfill({ json: command });
    }
    throw Error("unexpected synthetic picker request");
  });
  let component = await mount(<AccountPickerHarness />);
  const checks = page.getByRole("checkbox");
  await expect(checks).toHaveCount(2);
  await expect(checks.nth(0)).not.toBeChecked();
  await expect(checks.nth(1)).not.toBeChecked();
  await expect(
    page.getByLabel(/Account code|Currency version|Binding date/),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Review selected accounts" }),
  ).toBeDisabled();
  await page.screenshot({
    path: info.outputPath("account-picker-desktop.png"),
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({
    path: info.outputPath("account-picker-mobile.png"),
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await checks.nth(0).focus();
  await page.keyboard.press("Space");
  await checks.nth(1).check();
  await page.getByRole("button", { name: "Review selected accounts" }).click();
  await expect(
    page.getByRole("heading", { name: "Accounts to register" }),
  ).toBeVisible();
  await expect(
    page.getByText(
      "2100 Credit card 2 · USD · Ledger available from 2026-11-02",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(
    page.getByText(
      "2101 Credit card 3 · EUR · Ledger available from 2026-11-02",
      { exact: true },
    ),
  ).toBeVisible();
  expect(prepares).toBe(1);
  expect(executes).toBe(0);
  await component.unmount();
  component = await mount(<AccountPickerHarness />);
  await expect(
    page.getByRole("button", { name: "Register selected accounts" }),
  ).toBeEnabled();
  expect(executes).toBe(0);
  await page
    .getByRole("button", { name: "Register selected accounts" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Account registration confirmed" }),
  ).toBeVisible();
  await component.unmount();
  component = await mount(<AccountPickerHarness />);
  await page.getByRole("button", { name: "Done", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "View transactions for Checking 1" }),
  ).toBeVisible();
  expect(prepares).toBe(1);
  expect(executes).toBe(1);
  expect(acks).toBe(1);
  await page
    .getByRole("button", { name: "View transactions for Checking 1" })
    .click();
  await expect(page.getByLabel("Reviewed source")).toHaveText(id(41));
  expect(
    await page.evaluate(() => ({
      local: localStorage.length,
      session: sessionStorage.length,
    })),
  ).toEqual({ local: 0, session: 0 });
});
