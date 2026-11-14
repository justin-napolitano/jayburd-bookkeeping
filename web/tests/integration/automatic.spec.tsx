import { test, expect } from "@playwright/experimental-ct-react";
import { resolve } from "node:path";
import { chooseFinanceTool, financeNavItem } from "../finance-navigation";
import { AutomaticHarness } from "./AutomaticHarness";

// Only delivery is fault-injected: the settings mutation commits through production Next, Go and SQL.
test("automatic categories persist YAML matches, protected choices, conflicts and settings recovery", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  let interactiveMutations = 0;
  let selectedRuleApplications = 0;
  page.on("request", (request) => {
    if (
      request.method() === "POST" &&
      /\/label-commands(?:\/|$)/.test(new URL(request.url()).pathname)
    )
      interactiveMutations++;
    if (
      request.method() === "POST" &&
      /"kind"\s*:\s*"rules_apply"/.test(request.postData() ?? "")
    )
      selectedRuleApplications++;
  });
  const component = await mount(<AutomaticHarness />);
  await chooseFinanceTool(page, "Transactions");
  const rows = page.getByRole("list", { name: "Transaction rows" });
  const routine = rows
    .getByRole("listitem")
    .filter({ hasText: "Fictional automatic cafe purchase" });
  const protectedRow = rows
    .getByRole("listitem")
    .filter({ hasText: "Fictional automatic protected choice" });
  const unknown = rows
    .getByRole("listitem")
    .filter({ hasText: "Fictional automatic unknown purchase" });
  await expect(
    routine.getByText("Fictional Automatic Food", { exact: true }),
  ).toBeVisible();
  await expect(
    routine.getByText("Automatic · Your rule", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Fictional Automatic Travel", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Your choice", { exact: true }),
  ).toBeVisible();
  await expect(
    unknown.getByText("Needs category", { exact: true }),
  ).toBeVisible();
  await expect(rows.getByRole("listitem")).toHaveCount(3);
  expect(interactiveMutations).toBe(0);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);

  const headers = {
    Authorization: "Bearer synthetic.automatic.proof",
    Origin: new URL(page.url()).origin,
  };
  const discovery = await page.request.get(
    new URL("/v1/collection", page.url()).href,
    { headers },
  );
  expect(discovery.status()).toBe(200);
  const workspace = (await discovery.json()).workspace_id;
  const settingsURL = new URL(
    `/v1/collections/${workspace}/automation`,
    page.url(),
  ).href;
  expect(
    (
      await page.request.get(settingsURL, {
        headers: {
          ...headers,
          Authorization: "Bearer synthetic.automatic.other",
        },
      })
    ).status(),
  ).toBe(404);
  expect(
    (
      await page.request.get(settingsURL, {
        headers: { ...headers, Origin: "https://foreign.invalid" },
      })
    ).status(),
  ).toBe(403);
  const initial = await (
    await page.request.get(settingsURL, { headers })
  ).json();
  expect(initial).toMatchObject({
    enabled: true,
    processing_state: "idle",
    pending_count: "0",
    attention_count: "1",
    last_run: { state: "committed", processed_count: "3" },
  });

  await page
    .getByRole("button", { name: "Needs attention", exact: true })
    .click();
  await expect(rows.getByRole("listitem")).toHaveCount(1);
  await expect(unknown).toBeVisible();
  await page.getByText("Book, topic and date filters", { exact: true }).click();
  await page
    .getByRole("button", { name: "Reset filters", exact: true })
    .click();
  await expect(rows.getByRole("listitem")).toHaveCount(3);

  await page.getByLabel("Category settings", { exact: true }).click();
  const attempts: string[] = [];
  let lost = false;
  await page.route("**/v1/collections/*/automation", async (route) => {
    if (route.request().method() !== "POST") {
      await route.continue();
      return;
    }
    attempts.push(route.request().postData() ?? "");
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    expect((await response.json()).enabled).toBe(false);
    if (!lost) {
      lost = true;
      await route.abort("failed");
    } else await route.fulfill({ response });
  });
  await page
    .getByRole("button", { name: "Turn off automatic categories", exact: true })
    .click();
  const retry = page.getByRole("button", {
    name: "Retry same settings change",
    exact: true,
  });
  await expect(retry).toBeVisible();
  await expect(financeNavItem(page, "Accounts")).toBeDisabled();
  // A read sees the committed pause, but cannot resolve the UI's unknown operation identity.
  expect(
    (await (await page.request.get(settingsURL, { headers })).json()).enabled,
  ).toBe(false);
  await expect(retry).toBeVisible();
  await retry.click();
  await expect.poll(() => attempts.length).toBe(2);
  expect(attempts[0]).toBe(attempts[1]);
  expect(JSON.parse(attempts[0])).toMatchObject({
    expected_revision: "0",
    enabled: false,
  });
  await expect(
    page.getByRole("button", {
      name: "Enable automatic categories",
      exact: true,
    }),
  ).toBeVisible();
  await expect(financeNavItem(page, "Accounts")).toBeEnabled();
  await page.unroute("**/v1/collections/*/automation");
  await page
    .getByRole("button", { name: "Enable automatic categories", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Enable automatic categories",
      exact: true,
    }),
  ).toHaveCount(0);
  await expect(
    routine.getByText("Automatic · Your rule", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Your choice", { exact: true }),
  ).toBeVisible();
  const final = await (await page.request.get(settingsURL, { headers })).json();
  expect(final).toMatchObject({
    enabled: true,
    revision: "2",
    pending_count: "0",
    attention_count: "1",
    last_run: { run_id: initial.last_run.run_id, state: "committed" },
  });
  expect(interactiveMutations).toBe(0);

  // Import actual files through the product authoring/review flow, retaining the seeded cafe rule.
  // Fixed expectations below are authored independently of the parser and Go matcher.
  const saveYamlPack = async (file: string, preserved: number) => {
    await page.getByText("Manage rules", { exact: true }).click();
    await page
      .getByRole("button", { name: "Import YAML", exact: true })
      .click();
    await page
      .getByLabel("YAML file", { exact: true })
      .setInputFiles(resolve(process.cwd(), "tests", "fixtures", file));
    await expect(
      page.getByLabel("Rule pack YAML", { exact: true }),
    ).toHaveValue(/Fictional Unknown Shop/);
    const beforeDraft = interactiveMutations;
    await page
      .getByRole("button", { name: "Validate YAML", exact: true })
      .click();
    await expect(
      page.getByRole("status").filter({
        hasText: `1 added · 0 updated · ${preserved} existing rules preserved`,
      }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Add to draft", exact: true })
      .click();
    expect(interactiveMutations).toBe(beforeDraft);
    await page
      .getByRole("button", { name: "Preview rule changes", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Save rule review", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Confirm rule changes", exact: true })
      .click();
    await expect(
      page
        .getByRole("status")
        .filter({ hasText: "Rules updated. No journal was created." }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Acknowledge result", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Refresh transactions", exact: true }),
    ).toBeEnabled();
  };
  const waitForRun = async (priorID: string, attention: string) => {
    await expect
      .poll(async () => {
        const state = await (
          await page.request.get(settingsURL, { headers })
        ).json();
        return (
          state.last_run?.run_id !== priorID &&
          state.last_run?.state === "committed" &&
          state.last_run?.processed_count === "3" &&
          state.processing_state === "idle" &&
          state.pending_count === "0" &&
          state.attention_count === attention
        );
      })
      .toBe(true);
    await page
      .getByRole("button", { name: "Refresh transactions", exact: true })
      .click();
    return (await (await page.request.get(settingsURL, { headers })).json())
      .last_run.run_id as string;
  };
  await saveYamlPack("automatic-yaml-eligible.yaml", 1);
  const eligibleRun = await waitForRun(initial.last_run.run_id, "0");
  await expect(
    unknown.getByText("Fictional Automatic Food", { exact: true }),
  ).toBeVisible();
  await expect(
    unknown.getByText("Automatic · Your rule", { exact: true }),
  ).toBeVisible();
  await expect(
    unknown
      .locator(".transaction-category")
      .getByText("Fictional YAML shop food", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Fictional Automatic Travel", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Your choice", { exact: true }),
  ).toBeVisible();

  await saveYamlPack("automatic-yaml-conflict.yaml", 2);
  const conflictRun = await waitForRun(eligibleRun, "1");
  await expect(
    unknown.getByText("Needs category", { exact: true }),
  ).toBeVisible();
  await expect(
    unknown.getByText("Your rules disagree", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Fictional Automatic Travel", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Your choice", { exact: true }),
  ).toBeVisible();
  await expect(
    routine.getByText("Automatic · Your rule", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Needs attention", exact: true })
    .click();
  await expect(rows.getByRole("listitem")).toHaveCount(1);
  await expect(unknown).toBeVisible();
  await page.getByText("Book, topic and date filters", { exact: true }).click();
  await page
    .getByRole("button", { name: "Reset filters", exact: true })
    .click();
  await expect(rows.getByRole("listitem")).toHaveCount(3);
  expect(selectedRuleApplications).toBe(0);
  // Two catalog reviews account for prepare/execute/ack requests; no per-transaction apply was invoked.
  expect(interactiveMutations).toBe(6);

  // A fresh mount discards every React draft and discovers the persisted editions/decisions again.
  await component.unmount();
  const reopened = await mount(<AutomaticHarness />);
  await chooseFinanceTool(page, "Transactions");
  await expect(
    unknown.getByText("Your rules disagree", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Your choice", { exact: true }),
  ).toBeVisible();
  await expect(
    protectedRow.getByText("Fictional Automatic Travel", { exact: true }),
  ).toBeVisible();
  await expect(
    routine.getByText("Fictional Automatic Food", { exact: true }),
  ).toBeVisible();
  const persisted = await (
    await page.request.get(settingsURL, { headers })
  ).json();
  expect(persisted).toMatchObject({
    enabled: true,
    pending_count: "0",
    attention_count: "1",
    last_run: { run_id: conflictRun, state: "committed", processed_count: "3" },
  });
  await page.screenshot({
    path: test.info().outputPath("fictional-automatic-mobile.png"),
    fullPage: true,
  });
  await page.setViewportSize({ width: 1440, height: 900 });
  await expect(
    routine.getByText("Fictional Automatic Food", { exact: true }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: test.info().outputPath("fictional-automatic-desktop.png"),
    fullPage: true,
  });
  await reopened.unmount();
});
