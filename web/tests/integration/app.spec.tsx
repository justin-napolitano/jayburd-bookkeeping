import { chooseFinanceTool } from "../finance-navigation";
import { test, expect } from "@playwright/experimental-ct-react";
import { IntegrationHarness } from "./Harness";

/** Real browser forms traverse Next→Go→PostgreSQL; only response delivery is fault-injected. */
test("expired enrollment preserves book setup recovery and rejects unlinked postings", async ({
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
          Authorization: "Bearer synthetic.browser.proof",
          Origin: "https://foreign.example.invalid",
        },
      })
    ).status(),
  ).toBe(403);
  await expect(page.getByText("No books are available.")).toBeVisible();
  await page
    .getByText("Create a custom entity and book", { exact: true })
    .click();
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
  let lostSetup = "";
  let setupExecutions = 0;
  await page.route("**/v1/commands/*/execute", async (route) => {
    setupExecutions++;
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    const command = await response.json();
    expect(command.state).toBe("committed");
    lostSetup = command.command_id;
    await route.abort("failed");
  });
  await page.getByRole("button", { name: "Create book", exact: true }).click();
  await expect.poll(() => lostSetup).not.toBe("");
  await component.unmount();
  await page.unroute("**/v1/commands/*/execute");
  component = await mount(<IntegrationHarness />);
  await expect(
    page.getByText("Book creation confirmed", { exact: true }),
  ).toBeVisible();
  // Recovery proves its exact identity through the real authorized response;
  // a customer must not need the transport command UUID to continue.
  const recovered = await page.request.get(
    new URL("/v1/commands/pending", page.url()).href,
    {
      headers: {
        Authorization: "Bearer synthetic.browser.proof",
        Origin: new URL(page.url()).origin,
      },
    },
  );
  expect(recovered.status()).toBe(200);
  expect(await recovered.json()).toMatchObject({
    command: {
      command_id: lostSetup,
      operation: "setup",
      state: "committed",
      acknowledged: false,
    },
  });
  await expect(page.getByText(lostSetup, { exact: true })).toHaveCount(0);
  expect(setupExecutions).toBe(1);
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await page
    .getByLabel("Entity and book")
    .selectOption({ label: "Synthetic browser LLC / Operating" });
  await chooseFinanceTool(page, "Position");
  await expect(page.locator(".position-equity")).toHaveText("0.00");
  const [entity, book] = (
    await page.getByLabel("Entity and book").inputValue()
  ).split("/");
  // Private query POST reaches real SQL validation/authorization through the production Next bridge.
  const queryURL = new URL(
    `/v1/entities/${entity}/books/${book}/review-items/query`,
    page.url(),
  ).href;
  const queryHeaders = {
    Authorization: "Bearer synthetic.browser.proof",
    Origin: new URL(page.url()).origin,
  };
  const query = {
    source_account_id: "00000000-0000-4000-8000-000000000999",
    state: null,
    limit: 50,
    cursor: null,
    sort: "date",
    order: "desc",
    search: "Fictional private query",
    starts_on: null,
    through_date: null,
  };
  expect(
    (
      await page.request.post(queryURL, {
        headers: queryHeaders,
        data: { ...query, search: 5 },
      })
    ).status(),
  ).toBe(400);
  expect(
    (
      await page.request.post(queryURL, { headers: queryHeaders, data: query })
    ).status(),
  ).toBe(404);
  const deniedEnrollment = await page.request.post(
    new URL(
      `/v1/entities/${entity}/books/${book}/connection-attempts`,
      page.url(),
    ).href,
    {
      headers: {
        Authorization: "Bearer synthetic.browser.proof",
        "Idempotency-Key": "expired-enrollment-proof",
        Origin: new URL(page.url()).origin,
      },
      data: {
        mode: "new",
        sync_item_id: null,
        expected_generation: null,
        approved_stream_ids: [],
      },
    },
  );
  expect(deniedEnrollment.status()).toBe(503);
  // Book setup is not evidence for an economic event. Public application
  // posting must start with an imported source artifact, never a blank entry.
  await expect(
    page.getByRole("button", { name: "New journal", exact: true }),
  ).toHaveCount(0);
  await expect(page.getByLabel("Accounting date", { exact: true })).toHaveCount(
    0,
  );
  for (const [path, body] of [
    [`/v1/entities/${entity}/books/${book}/journals`, {}],
    [
      `/v1/entities/${entity}/books/${book}/journals/00000000-0000-4000-8000-000000000999/reversals`,
      {},
    ],
    ["/v1/commands", { operation: "post" }],
  ] as const) {
    const response = await page.request.post(new URL(path, page.url()).href, {
      headers: { ...queryHeaders, "Idempotency-Key": "reject-unlinked-post" },
      data: body,
    });
    expect(response.status()).toBe(422);
    // Next intentionally strips upstream diagnostic codes. The Go public-handler
    // proof separately requires artifact_required and no posting SQL; this actual
    // bridge journey requires its exact existing opaque Problem profile.
    expect(await response.json()).toEqual({
      type: "about:blank",
      title: "Request unavailable",
      status: 422,
    });
  }
  await chooseFinanceTool(page, "Chart of accounts");
  await page.getByLabel("Through date", { exact: true }).fill("2026-01-31");
  await page
    .getByRole("button", { name: "Load trial balance", exact: true })
    .click();
  await expect(page.locator("tfoot td")).toHaveText(["0.00", "0.00"]);
  // Nonzero posting/report/drill/export oracles are exercised in the real
  // connection acquisition→source_post journey, not relabeled manual requests.
});
