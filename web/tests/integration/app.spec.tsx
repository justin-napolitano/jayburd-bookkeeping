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
          Authorization: "Bearer synthetic-browser-proof",
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
  await page.getByLabel("Through date").fill("2026-01-31");
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
});
