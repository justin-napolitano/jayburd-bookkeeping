import { test, expect } from "@playwright/experimental-ct-react";
import { IntegrationHarness } from "./Harness";
import { RecoveryHarness } from "./RecoveryHarness";

/** Only delivery is fault-injected: actual commands commit through Next, Go and PostgreSQL. */
test("private topics and strategic tags recover committed responses without books or journals", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  let component = await mount(<IntegrationHarness />);
  await page
    .getByRole("button", { name: "All transactions", exact: true })
    .click();
  await expect(
    page.getByText("Fictional cafe purchase", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Fictional euro purchase", { exact: true }),
  ).toBeVisible();
  const headers = {
    Authorization: "Bearer synthetic.browser.proof",
    Origin: new URL(page.url()).origin,
  };
  const discovery = await page.request.get(
    new URL("/v1/collection", page.url()).href,
    { headers },
  );
  expect(discovery.status()).toBe(200);
  const workspace = (await discovery.json()).workspace_id;
  const base = `/v2/collections/${workspace}`;
  expect(
    (
      await page.request.get(new URL(base + "/taxonomy", page.url()).href, {
        headers: { ...headers, Authorization: "Bearer synthetic.other.proof" },
      })
    ).status(),
  ).toBe(404);
  expect(
    (
      await page.request.get(new URL(base + "/taxonomy", page.url()).href, {
        headers: { ...headers, Origin: "https://foreign.invalid" },
      })
    ).status(),
  ).toBe(403);
  const catalog = await (
    await page.request.get(new URL(base + "/taxonomy", page.url()).href, {
      headers,
    })
  ).json();
  expect(catalog.topics).toEqual([]);
  await page.getByText("Manage topics", { exact: true }).click();
  await page.getByRole("button", { name: "Add topic", exact: true }).click();
  await page.getByLabel("Topic 1 name", { exact: true }).fill("Fictional Food");
  await page
    .getByRole("button", { name: "Preview topic changes", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Review topic changes", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Save topic review", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Saved topic review", exact: true }),
  ).toBeVisible();
  let taxonomyExecutions = 0;
  await page.route(
    "**/v2/collections/*/label-commands/*/execute",
    async (route) => {
      const response = await route.fetch();
      expect(response.status()).toBe(200);
      expect((await response.json()).command.state).toBe("committed");
      await route.abort("failed");
      taxonomyExecutions++;
    },
  );
  await page
    .getByRole("button", { name: "Confirm topic changes", exact: true })
    .click();
  await expect.poll(() => taxonomyExecutions).toBe(1);
  await component.unmount();
  await page.unroute("**/v2/collections/*/label-commands/*/execute");
  component = await mount(<IntegrationHarness />);
  await expect(
    page.getByText("Topics updated. No journal was created.", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  expect(taxonomyExecutions).toBe(1);
  await expect(
    page.getByText("Fictional cafe purchase", { exact: true }),
  ).toBeVisible();
  const cafe = page
    .getByRole("listitem")
    .filter({ hasText: "Fictional cafe purchase" });
  await cafe.getByRole("checkbox").check();
  await page
    .getByLabel("Label selected transactions", { exact: true })
    .selectOption({ label: "Fictional Food" });
  await page
    .getByRole("button", { name: "Preview labels for 1 selected", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Save label review", exact: true })
    .click();
  let labelExecutions = 0;
  await page.route(
    "**/v2/collections/*/label-commands/*/execute",
    async (route) => {
      const response = await route.fetch();
      expect(response.status()).toBe(200);
      const receipt = await response.json();
      expect(receipt.command.state).toBe("committed");
      expect(receipt.command.preview.review_rows[0].money.amount).toBe("0.1");
      await route.abort("failed");
      labelExecutions++;
    },
  );
  await page
    .getByRole("button", { name: "Confirm labels for 1 selected", exact: true })
    .click();
  await expect.poll(() => labelExecutions).toBe(1);
  await component.unmount();
  await page.unroute("**/v2/collections/*/label-commands/*/execute");
  component = await mount(<IntegrationHarness />);
  await expect(
    page.getByText("Labels applied. No journal was created.", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  expect(labelExecutions).toBe(1);
  await expect(
    page.getByText("Fictional cafe purchase", { exact: true }),
  ).toBeVisible();
  await page.getByText("Manage tags", { exact: true }).click();
  await page.getByRole("button", { name: "Add tag", exact: true }).click();
  await page
    .getByLabel("Tag 1 name", { exact: true })
    .fill("Fictional Reduce Waste");
  await page
    .getByRole("button", { name: "Preview tag catalog", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Review tag catalog", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Save tag catalog review", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm tag catalog changes", exact: true })
    .click();
  await expect(
    page.getByText("Tag catalog updated. No journal was created.", {
      exact: true,
    }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await expect(
    page.getByText("Fictional cafe purchase", { exact: true }),
  ).toBeVisible();
  await cafe.getByRole("checkbox").check();
  await page.getByText("Change strategic tags", { exact: true }).click();
  await page.getByLabel("Tag operation", { exact: true }).selectOption("add");
  await page
    .getByRole("group", { name: "Strategic tags", exact: true })
    .getByRole("checkbox", { name: "Fictional Reduce Waste", exact: true })
    .check();
  await page
    .getByRole("button", { name: "Preview tags for 1 selected", exact: true })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "Review tags for 1 selected",
      exact: true,
    }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Save tag review", exact: true })
    .click();
  let tagExecutions = 0;
  await page.route(
    "**/v2/collections/*/label-commands/*/execute",
    async (route) => {
      const response = await route.fetch();
      expect(response.status()).toBe(200);
      const receipt = await response.json();
      expect(receipt.command.state).toBe("committed");
      expect(receipt.command.preview.action).toMatchObject({
        kind: "tags",
        operation: "add",
      });
      expect(receipt.command.preview.tag_results[0].tags[0].name).toBe(
        "Fictional Reduce Waste",
      );
      await route.abort("failed");
      tagExecutions++;
    },
  );
  await page
    .getByRole("button", { name: "Confirm tags for 1 selected", exact: true })
    .click();
  await expect.poll(() => tagExecutions).toBe(1);
  await component.unmount();
  await page.unroute("**/v2/collections/*/label-commands/*/execute");
  component = await mount(<IntegrationHarness />);
  await expect(
    page.getByText("Tags applied. No journal was created.", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  expect(tagExecutions).toBe(1);
  await expect(
    page.getByText("Fictional cafe purchase", { exact: true }),
  ).toBeVisible();
  const currentCatalog = await (
    await page.request.get(new URL(base + "/taxonomy", page.url()).href, {
      headers,
    })
  ).json();
  const query = {
    profile: "private-collection-review-v2",
    source_account_id: null,
    assignment: "all",
    starts_on: null,
    through_date: null,
    search: "",
    topic: { kind: "all" },
    tag: { kind: "all" },
    sort: "date_desc",
    snapshot: null,
    cursor: null,
    limit: 25,
  };
  const result = await page.request.post(
    new URL(base + "/review/query", page.url()).href,
    { headers, data: query },
  );
  expect(result.status()).toBe(200);
  const actual = await result.json();
  expect(actual.matching_count).toBe("4");
  expect(actual.partial).toBe(false);
  const food = actual.items.find(
    (row: { description: string }) =>
      row.description === "Fictional cafe purchase",
  );
  expect(food.owner_topic).toEqual({
    state: "topic",
    topic_id: currentCatalog.topics[0].topic_id,
    name: "Fictional Food",
  });
  expect(food.owner_tags).toEqual({
    state: "current",
    tags: [
      {
        tag_id: currentCatalog.tags[0].tag_id,
        name: "Fictional Reduce Waste",
        status: "active",
      },
    ],
  });
  expect(
    actual.items.find(
      (row: { description: string }) =>
        row.description === "Fictional exact large quantity",
    ).money.amount,
  ).toBe("9007199254740993.01");
  expect(
    actual.items.find(
      (row: { description: string }) =>
        row.description === "Fictional euro purchase",
    ).money.numeric_code,
  ).toBeNull();
  expect(
    actual.items.find(
      (row: { description: string }) =>
        row.description === "Fictional unsupported quantity",
    ).money,
  ).toMatchObject({
    state: "held",
    reason: "unsupported_currency",
    amount: null,
  });
  await page.screenshot({
    path: test.info().outputPath("fictional-private-labels-mobile.png"),
    fullPage: true,
  });
});

test("inaccessible prepared private work retains the same clear identity after response loss", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mount(<RecoveryHarness />);
  await expect(
    page.getByRole("heading", { name: "Saved private work", exact: true }),
  ).toBeVisible();
  const navigation = page.getByRole("button", {
    name: "All transactions",
    exact: true,
  });
  await expect(navigation).toBeDisabled();
  await expect(
    page.getByText("Fictional hidden prepared tag", { exact: true }),
  ).toHaveCount(0);
  const headers = {
    Authorization: "Bearer synthetic.prepared.proof",
    Origin: new URL(page.url()).origin,
  };
  const status = await page.request.get(
    new URL("/v1/collection-recovery", page.url()).href,
    { headers },
  );
  expect(status.status()).toBe(200);
  const candidate = await status.json();
  expect(Object.keys(candidate).sort()).toEqual([
    "profile",
    "recovery_id",
    "state",
  ]);
  expect(candidate.state).toBe("inaccessible");
  const attempts: unknown[] = [];
  let lostReplies = 0;
  await page.route("**/v1/collection-recovery/clear", async (route) => {
    attempts.push(route.request().postDataJSON());
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    const receipt = await response.json();
    expect(Object.keys(receipt).sort()).toEqual([
      "cleared",
      "profile",
      "recovery_id",
    ]);
    expect(receipt).toEqual({
      profile: "private-collection-recovery-clear-v1",
      recovery_id: candidate.recovery_id,
      cleared: true,
    });
    if (attempts.length === 1) {
      await route.abort("failed");
      lostReplies++;
    } else {
      await route.fulfill({ response });
    }
  });
  await page
    .getByRole("button", {
      name: "Clear inaccessible private review",
      exact: true,
    })
    .click();
  await expect.poll(() => lostReplies).toBe(1);
  await expect(
    page.getByRole("button", {
      name: "Retry clearing the same private review",
      exact: true,
    }),
  ).toBeVisible();
  await expect(navigation).toBeDisabled();
  // The database now has no old claim, but absence cannot resolve this in-memory unknown operation.
  await page
    .getByRole("button", { name: "Check private recovery", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Retry clearing the same private review",
      exact: true,
    }),
  ).toBeVisible();
  await expect(navigation).toBeDisabled();
  await page
    .getByRole("button", {
      name: "Retry clearing the same private review",
      exact: true,
    })
    .click();
  await expect.poll(() => attempts.length).toBe(2);
  await expect(navigation).toBeEnabled();
  expect(attempts).toEqual([
    { recovery_id: candidate.recovery_id },
    { recovery_id: candidate.recovery_id },
  ]);
  await expect(
    page.getByText("Fictional hidden prepared tag", { exact: true }),
  ).toHaveCount(0);
  const fresh = await page.request.get(
    new URL("/v1/collection-recovery", page.url()).href,
    { headers },
  );
  expect(await fresh.json()).toEqual({
    profile: "private-collection-recovery-v1",
    state: "none",
    recovery_id: null,
  });
  await page.screenshot({
    path: test.info().outputPath("fictional-prepared-recovery-mobile.png"),
    fullPage: true,
  });
});

test("inaccessible committed private work is acknowledged without revealing its result", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await mount(<RecoveryHarness committed />);
  await expect(
    page.getByRole("heading", { name: "Saved private work", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Fictional hidden committed tag", { exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "All transactions", exact: true }),
  ).toBeDisabled();
  let cleared = 0;
  await page.route("**/v1/collection-recovery/clear", async (route) => {
    const request = route.request().postDataJSON();
    expect(Object.keys(request)).toEqual(["recovery_id"]);
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    const receipt = await response.json();
    expect(Object.keys(receipt).sort()).toEqual([
      "cleared",
      "profile",
      "recovery_id",
    ]);
    expect(receipt.recovery_id).toBe(request.recovery_id);
    expect(receipt.cleared).toBe(true);
    await route.fulfill({ response });
    cleared++;
  });
  await page
    .getByRole("button", {
      name: "Clear inaccessible private review",
      exact: true,
    })
    .click();
  await expect.poll(() => cleared).toBe(1);
  await expect(
    page.getByRole("button", { name: "All transactions", exact: true }),
  ).toBeEnabled();
  await expect(
    page.getByText("Fictional hidden committed tag", { exact: true }),
  ).toHaveCount(0);
  await page.screenshot({
    path: test.info().outputPath("fictional-committed-recovery-desktop.png"),
    fullPage: true,
  });
});
