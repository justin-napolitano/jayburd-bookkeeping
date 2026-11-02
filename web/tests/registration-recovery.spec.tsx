import { test, expect } from "@playwright/experimental-ct-react";
import { RegistrationRecoveryHarness } from "./RegistrationRecoveryHarness";
import { savedRegistration } from "./registration.fixture";
import type { DurableCommand } from "../src/bookkeeping/types";

// Only HTTP response delivery is lost; mocked state is not claimed as SQL/provider authority evidence.
test("saved registration survives lost prepare/execute responses and remount before explicit acknowledgement", async ({
  mount,
  page,
}, info) => {
  let command: DurableCommand | null = null;
  const preparations: { body: string | null; key: string | undefined }[] = [];
  let executes = 0,
    acknowledgements = 0;
  await page.route("**/v1/**", async (route) => {
    const request = route.request(),
      path = new URL(request.url()).pathname;
    if (path === "/v1/commands/pending")
      return route.fulfill({
        json: { command: command?.acknowledged ? null : command },
      });
    if (path.endsWith("/registration")) {
      preparations.push({
        body: request.postData(),
        key: request.headers()["idempotency-key"],
      });
      command = structuredClone(savedRegistration);
      return route.abort("failed");
    }
    if (path.endsWith("/execute")) {
      executes++;
      command = {
        ...savedRegistration,
        state: "committed",
        result: {
          entity_id: savedRegistration.entity_id!,
          book_id: savedRegistration.book_id!,
          sync_item_id: "00000000-0000-4000-8000-000000000005",
          streams: [],
        },
      };
      return route.abort("failed");
    }
    if (path.endsWith("/acknowledge")) {
      acknowledgements++;
      command = { ...command!, acknowledged: true };
      return route.fulfill({ json: command });
    }
    throw new Error("unexpected synthetic registration request");
  });
  let component = await mount(<RegistrationRecoveryHarness />);
  const prepare = page.getByRole("button", {
    name: "Prepare selected account",
  });
  await expect(prepare).toBeEnabled();
  await prepare.click();
  await expect(
    page.getByRole("heading", { name: "Accounts to register" }),
  ).toBeVisible();
  expect(preparations).toHaveLength(1);
  expect(JSON.parse(preparations[0].body!)).toEqual({
    account_ids: ["synthetic-checking"],
  });
  expect(preparations[0].key).toBeTruthy();
  expect(executes).toBe(0);
  expect(acknowledgements).toBe(0);
  await component.unmount();
  component = await mount(<RegistrationRecoveryHarness />);
  await expect(page.getByText(/1001 Checking 1/)).toBeVisible();
  expect(preparations).toHaveLength(1);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({
    path: info.outputPath("registration-recovery-mobile.png"),
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page
    .getByRole("button", { name: "Register selected accounts" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Account registration confirmed" }),
  ).toBeVisible();
  expect(executes).toBe(1);
  expect(acknowledgements).toBe(0);
  await component.unmount();
  component = await mount(<RegistrationRecoveryHarness />);
  await expect(page.getByRole("button", { name: "Done" })).toBeVisible();
  expect(executes).toBe(1);
  await page.getByRole("button", { name: "Done" }).click();
  await expect(prepare).toBeEnabled();
  expect(acknowledgements).toBe(1);
  expect(
    await page.evaluate(() => ({
      local: localStorage.length,
      session: sessionStorage.length,
    })),
  ).toEqual({ local: 0, session: 0 });
});
