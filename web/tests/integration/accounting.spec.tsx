import { expect, test } from "@playwright/experimental-ct-react";
import { chooseFinanceTool } from "../finance-navigation";
import { AccountingHarness } from "./AccountingHarness";
const entity = "e7100000-0000-4000-8000-000000000002",
  book = "e7100000-0000-4000-8000-000000000003";
const base = `/v1/entities/${entity}/books/${book}`;
const yaml = `version: 1
rules:
  - id: e7100000-0000-4000-8000-000000000401
    name: Fictional routine office paper
    source: {name: Fictional operating bank}
    direction: outflow
    description: Fictional automatic paper purchase
    offset: {name: Fictional office supplies}
    treatment: expense
  - id: e7100000-0000-4000-8000-000000000402
    name: Fictional protected draft mapping
    source: {name: Fictional operating bank}
    direction: outflow
    description: Fictional protected manual purchase
    offset: {name: Fictional office supplies}
    treatment: expense
`;

// No API results are mocked. Delivery of one already-committed configuration response is the sole fault injection.
test("one YAML policy posts eligible retained evidence once without per-entry commands", async ({
  mount,
  page,
}) => {
  test.setTimeout(90000);
  await page.setViewportSize({ width: 390, height: 844 });
  let interactive = 0;
  const attempts: string[] = [];
  page.on("request", (request) => {
    if (
      request.method() === "POST" &&
      /^\/v1\/(?:commands|post)(?:\/|$)/.test(new URL(request.url()).pathname)
    )
      interactive++;
  });
  let component = await mount(<AccountingHarness />);
  await page.getByLabel("Entity and book").selectOption(`${entity}/${book}`);
  await chooseFinanceTool(page, "Review accounting");
  await expect(
    page.getByText("Automation is available. This book’s policy is off.", {
      exact: true,
    }),
  ).toBeVisible();
  const headers = {
    Authorization: "Bearer synthetic.accounting.proof",
    Origin: new URL(page.url()).origin,
  };
  const read = async (path: string) => {
    const response = await page.request.get(new URL(path, page.url()).href, {
      headers,
    });
    expect(response.status()).toBe(200);
    return response.json();
  };
  const initial = await read(base + "/automatic-accounting/status");
  expect(initial).toMatchObject({
    enabled: false,
    runtime_enabled: true,
    policy_revision: "0",
    last_operation: null,
  });
  expect((await read(base + "/automatic-accounting")).revision).toBe("0");
  // A verified read-only foreign principal has no source/posting scope; origin denial remains enforced.
  for (const [token, origin, want] of [
    ["synthetic.accounting.foreign", headers.Origin, 404],
    ["synthetic.accounting.proof", "https://foreign.invalid", 403],
  ] as const) {
    expect(
      (
        await page.request.get(
          new URL(base + "/automatic-accounting/status", page.url()).href,
          { headers: { Authorization: `Bearer ${token}`, Origin: origin } },
        )
      ).status(),
    ).toBe(want);
  }
  await page
    .getByRole("button", { name: "Configure accounting", exact: true })
    .click();
  await page
    .getByLabel("Enable automatic posting for these mappings", { exact: true })
    .check();
  await page.getByLabel("Accounting YAML file", { exact: true }).setInputFiles({
    name: "fictional-accounting.yaml",
    mimeType: "application/yaml",
    buffer: Buffer.from(yaml),
  });
  await expect(
    page.getByLabel("Accounting mapping YAML", { exact: true }),
  ).toHaveValue(yaml);
  await page
    .getByRole("button", {
      name: "Review accounting configuration",
      exact: true,
    })
    .click();
  await expect(
    page.getByRole("table", { name: "Mappings to save" }).getByRole("row"),
  ).toHaveCount(3);
  expect((await read(base + "/automatic-accounting/status")).enabled).toBe(
    false,
  );
  expect(interactive).toBe(0);
  let lost = false;
  await page.route("**/automatic-accounting", async (route) => {
    if (route.request().method() !== "POST") {
      await route.continue();
      return;
    }
    attempts.push(route.request().postData() ?? "");
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    expect((await response.json()).enabled).toBe(true);
    if (!lost) {
      lost = true;
      await route.abort("failed");
    } else await route.fulfill({ response });
  });
  await page
    .getByRole("button", { name: "Save accounting configuration", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Retry same accounting configuration",
      exact: true,
    }),
  ).toBeVisible();
  expect(attempts).toHaveLength(1);
  expect((await read(base + "/automatic-accounting")).enabled).toBe(true);
  await page
    .getByRole("button", {
      name: "Retry same accounting configuration",
      exact: true,
    })
    .click();
  await expect.poll(() => attempts.length).toBe(2);
  expect(attempts[1]).toBe(attempts[0]);
  await page.unroute("**/automatic-accounting");
  await expect
    .poll(
      async () => {
        const status = await read(base + "/automatic-accounting/status");
        return {
          pending: status.pending_count,
          attention: status.attention_count,
          reasons: status.holds.map((h: { reason: string }) => h.reason).sort(),
        };
      },
      { timeout: 45000 },
    )
    .toEqual({
      pending: "0",
      attention: "2",
      reasons: ["manual_draft", "unmapped_rule"],
    });
  await page
    .getByRole("button", { name: "Refresh accounting status", exact: true })
    .click();
  await expect(
    page.getByText(
      "Eligible source transactions post automatically under this policy. Unresolved items stay in review.",
      { exact: true },
    ),
  ).toBeVisible();
  await page.getByText("Accounting exceptions (2)", { exact: true }).click();
  await expect(
    page.getByText("A saved manual accounting draft is protected", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByText("Add an accounting mapping", { exact: true }),
  ).toBeVisible();
  // Drill through the existing production review detail and immutable journal reader.
  const posted = await read(
    base + "/review-items/e7100000-0000-4000-8000-000000000200",
  );
  expect(posted).toMatchObject({
    state: "posted",
    description: "Fictional automatic paper purchase",
    evidence: [
      {
        external_entry_id: "FICTIONAL-STATEMENT-1",
        observation_revision: "1",
        source_state: "booked",
        amount: "12.5",
        direction: "outflow",
        description: "Fictional automatic paper purchase",
      },
    ],
  });
  expect(posted.journal_id).toMatch(/^[a-f0-9-]{36}$/);
  const protectedDetail = await read(
    base + "/review-items/e7100000-0000-4000-8000-000000000202",
  );
  expect(protectedDetail.state).toBe("draft_ready");
  expect(protectedDetail.draft).not.toHaveProperty("origin");
  expect(protectedDetail.journal_id).toBeNull();
  const missing = await read(
    base + "/review-items/e7100000-0000-4000-8000-000000000201",
  );
  expect(missing.journal_id).toBeNull();
  // The customer exception action opens an independently authorized source detail.
  await page
    .getByRole("listitem")
    .filter({ hasText: "Add an accounting mapping" })
    .getByRole("button", { name: "Review source transaction", exact: true })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "Fictional unmapped source purchase",
      exact: true,
    }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Back to transactions", exact: true })
    .click();
  // Latest operation may be a hold; open the committed source using the actual source list's posted filter.
  const filter = page.getByLabel("Review status", { exact: true });
  await filter.selectOption("posted");
  await page
    .getByRole("button", {
      name: "Review Fictional automatic paper purchase",
      exact: true,
    })
    .click();
  await page
    .getByRole("button", { name: "View posted journal", exact: true })
    .click();
  const lines = page.getByRole("table", {
    name: "Immutable posted journal lines",
  });
  await expect(lines.getByRole("row")).toHaveCount(3);
  await expect(lines.getByText("12.5 USD", { exact: true })).toHaveCount(2);
  const journal = await read(base + `/journals/${posted.journal_id}`);
  expect(journal.lines).toEqual(
    expect.arrayContaining([
      expect.objectContaining({
        account_id: "e7100000-0000-4000-8000-000000000007",
        side: "debit",
        transaction_amount: "12.5",
        functional_amount: "12.5",
        transaction_currency: "USD",
      }),
      expect.objectContaining({
        account_id: "e7100000-0000-4000-8000-000000000006",
        side: "credit",
        transaction_amount: "12.5",
        functional_amount: "12.5",
        transaction_currency: "USD",
      }),
    ]),
  );
  expect(interactive).toBe(0);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: test.info().outputPath("accounting-source-posted-mobile.png"),
    fullPage: true,
  });
  await component.unmount();
  component = await mount(<AccountingHarness />);
  await page.getByLabel("Entity and book").selectOption(`${entity}/${book}`);
  await chooseFinanceTool(page, "Review accounting");
  await expect(
    page.getByText(
      "Eligible source transactions post automatically under this policy. Unresolved items stay in review.",
      { exact: true },
    ),
  ).toBeVisible();
  expect((await read(base + "/automatic-accounting")).revision).toBe("1");
  expect(
    (await read(base + "/review-items/e7100000-0000-4000-8000-000000000200"))
      .journal_id,
  ).toBe(posted.journal_id);
  expect(interactive).toBe(0);
  await page.setViewportSize({ width: 1440, height: 900 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: test.info().outputPath("accounting-persisted-desktop.png"),
    fullPage: true,
  });
  await component.unmount();
});
