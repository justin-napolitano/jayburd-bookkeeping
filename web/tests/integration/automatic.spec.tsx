import { test, expect } from "@playwright/experimental-ct-react";
import { chooseFinanceTool, financeNavItem } from "../finance-navigation";
import { AutomaticHarness } from "./AutomaticHarness";

// Only delivery is fault-injected: the settings mutation commits through production Next, Go and SQL.
test("automatic categories preserve owner choices and recover the same settings request", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  let interactiveMutations = 0;
  page.on("request", (request) => {
    if (
      request.method() === "POST" &&
      /\/label-commands(?:\/|$)/.test(new URL(request.url()).pathname)
    )
      interactiveMutations++;
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
  await component.unmount();
});
