import { chooseFinanceTool, financeNavItem } from "./finance-navigation";
import { expect, test } from "@playwright/experimental-ct-react";
import { Harness } from "./Harness";
import {
  commitOnboarding,
  onboardingCatalog,
  onboardingCommand,
  onboardingPending,
  onboardingPreview,
} from "../src/collection/onboarding-fixture";
import type { OnboardingCommand } from "../src/collection/onboarding";

test("mobile composed setup recovers its own claim while sibling engines report busy", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  let command: OnboardingCommand | null = null,
    created = false,
    executions = 0;
  const preview = onboardingPreview();
  await page.route("**/v*/**", async (route) => {
    const p = new URL(route.request().url()).pathname;
    const reply = (json: unknown) => route.fulfill({ json });
    if (p === "/v1/workspace")
      return reply({
        entities: [],
        currencies: [],
        development_policy: "dev-accrual-usd-v1",
      });
    if (p === "/v1/collection-onboarding/commands/pending")
      return reply(onboardingPending(command));
    if (p === "/v1/collection-onboarding")
      return reply(onboardingCatalog(created));
    if (p === "/v1/collection-onboarding/previews") return reply(preview);
    if (p === "/v1/collection-onboarding/commands") {
      command = onboardingCommand(
        preview,
        route.request().postDataJSON().prepare_key,
      );
      return reply(command);
    }
    if (
      p.startsWith("/v1/collection-onboarding/commands/") &&
      p.endsWith("/execute")
    ) {
      executions++;
      created = true;
      command = commitOnboarding(command!);
      return reply(command);
    }
    if (
      p.startsWith("/v1/collection-onboarding/commands/") &&
      p.endsWith("/acknowledge")
    ) {
      const receipt = { ...command!, acknowledged: true };
      command = null;
      return reply(receipt);
    }
    if (p === "/v1/commands/pending" || p === "/v1/collection-recovery") {
      if (command)
        return route.fulfill({
          status: 409,
          json: { code: "command_conflict" },
        });
      return reply(
        p.includes("collection")
          ? {
              profile: "private-collection-recovery-v1",
              state: "none",
              recovery_id: null,
            }
          : { command: null },
      );
    }
    return route.fulfill({ status: 404, json: { code: "not_found" } });
  });
  let component = await mount(<Harness />);
  await chooseFinanceTool(page, "Transactions");
  const setup = page.getByRole("button", {
    name: "Set up private transactions",
    exact: true,
  });
  await expect(setup).toBeEnabled();
  expect((await setup.boundingBox())!.y).toBeLessThan(700);
  await setup.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("heading", {
      name: "Review private transaction setup",
      exact: true,
    }),
  ).toBeFocused();
  await page.screenshot({
    path: "test-results/onboarding-mobile.png",
    fullPage: true,
  });
  await page
    .getByRole("button", { name: "Save setup review", exact: true })
    .click();
  expect(executions).toBe(0);
  await component.unmount();
  component = await mount(<Harness />);
  await expect(
    page.getByRole("button", { name: "Create private workspace", exact: true }),
  ).toBeEnabled();
  await expect(
    page.getByRole("heading", { name: "Saved command", exact: true }),
  ).toBeHidden();
  await expect(
    page.getByText(
      "Private recovery is unavailable. New work remains locked.",
      { exact: true },
    ),
  ).toBeHidden();
  await expect(financeNavItem(page, "Transactions")).toBeDisabled();
  await page
    .getByRole("button", { name: "Create private workspace", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Acknowledge setup result", exact: true }),
  ).toBeEnabled();
  await page
    .getByRole("button", { name: "Acknowledge setup result", exact: true })
    .click();
  await expect(financeNavItem(page, "Transactions")).toBeEnabled();
  await expect(
    page.getByRole("button", {
      name: "Review connection for Fictional records",
      exact: true,
    }),
  ).toBeEnabled();
  expect(executions).toBe(1);
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.screenshot({
    path: "test-results/onboarding-desktop.png",
    fullPage: true,
  });
  expect(
    await page.evaluate(() => [
      document.documentElement.scrollWidth <= innerWidth,
      localStorage.length,
      sessionStorage.length,
    ]),
  ).toEqual([true, 0, 0]);
});

test("ordinary book acknowledgement refreshes the onboarding reservation and eligible connection catalog", async ({
  mount,
  page,
}) => {
  let bookPending = true,
    onboardingReads = 0;
  const bookCommand = {
    operation: "setup",
    command_id: "fictional-book-command",
    state: "cancelled",
    acknowledged: false,
    result: null,
    created_at: "2026-11-08T00:00:00Z",
    body: {
      entity_name: "Fictional studio",
      book_name: "Fictional records",
      starts_on: "2026-01-01",
      ends_before: "2027-01-01",
      accounts: [],
    },
  };
  await page.route("**/v*/**", async (route) => {
    const p = new URL(route.request().url()).pathname;
    const reply = (json: unknown) => route.fulfill({ json });
    if (p === "/v1/workspace")
      return reply({
        entities: [],
        currencies: [],
        development_policy: "dev-accrual-usd-v1",
      });
    if (p === "/v1/commands/pending")
      return reply({ command: bookPending ? bookCommand : null });
    if (p.endsWith("/fictional-book-command/acknowledge")) {
      bookPending = false;
      return reply({ ...bookCommand, acknowledged: true });
    }
    if (p === "/v1/collection-recovery")
      return reply({
        profile: "private-collection-recovery-v1",
        state: "none",
        recovery_id: null,
      });
    if (p === "/v1/collection-onboarding/commands/pending") {
      onboardingReads++;
      return reply({
        ...onboardingPending(),
        state: bookPending ? "blocked" : "none",
      });
    }
    if (p === "/v1/collection-onboarding")
      return reply(onboardingCatalog(true));
    return route.fulfill({ status: 404, json: { code: "not_found" } });
  });
  await mount(<Harness />);
  await expect(financeNavItem(page, "Transactions")).toBeDisabled();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await expect(financeNavItem(page, "Transactions")).toBeEnabled();
  await chooseFinanceTool(page, "Transactions");
  await expect(
    page.getByRole("button", {
      name: "Review connection for Fictional records",
      exact: true,
    }),
  ).toBeVisible();
  expect(onboardingReads).toBeGreaterThanOrEqual(2);
});
