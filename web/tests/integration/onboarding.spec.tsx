import { expect, test } from "@playwright/experimental-ct-react";
import { OnboardingHarness } from "./OnboardingHarness";

// No domain response is mocked: delivery loss follows a real committed API response.
test("authenticated absent workspace setup and complete legacy adoption recover across lost replies and remounts", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  let component = await mount(<OnboardingHarness />);
  const headers = {
    Authorization: "Bearer synthetic.onboarding.proof",
    Origin: new URL(page.url()).origin,
  };
  const get = async (path: string) => {
    const response = await page.request.get(new URL(path, page.url()).href, {
      headers,
    });
    expect(response.status()).toBe(200);
    return response.json();
  };
  const initial = await get("/v1/collection-onboarding");
  expect(initial.workspace).toEqual({ state: "absent" });
  expect(initial.can_setup).toBe(true);
  expect(initial.can_adopt).toBe(false);
  expect(initial.candidates).toEqual([]);
  await page
    .getByRole("button", { name: "All transactions", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Set up private transactions", exact: true })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "Review private transaction setup",
      exact: true,
    }),
  ).toBeFocused();
  expect((await get("/v1/collection-onboarding")).workspace.state).toBe(
    "absent",
  );
  let lostPrepare = 0,
    preparedID = "",
    preparedKey = "",
    preparedPreview = "";
  const preparePattern = "**/v1/collection-onboarding/commands";
  await page.route(preparePattern, async (route) => {
    const response = await route.fetch();
    expect(response.status()).toBe(201);
    const value = await response.json();
    expect(value.state).toBe("prepared");
    preparedID = value.command_id;
    preparedKey = value.prepare_key;
    preparedPreview = JSON.stringify(value.preview);
    await route.abort("failed");
    lostPrepare++;
  });
  await page
    .getByRole("button", { name: "Save setup review", exact: true })
    .click();
  await expect.poll(() => lostPrepare).toBe(1);
  await expect(
    page.getByRole("button", { name: "All transactions", exact: true }),
  ).toBeDisabled();
  await component.unmount();
  await page.unroute(preparePattern);
  component = await mount(<OnboardingHarness />);
  await expect(
    page.getByRole("button", { name: "Create private workspace", exact: true }),
  ).toBeEnabled();
  const savedSetup = await get("/v1/collection-onboarding/commands/pending");
  expect(savedSetup.command.command_id).toBe(preparedID);
  expect(savedSetup.command.prepare_key).toBe(preparedKey);
  expect(JSON.stringify(savedSetup.command.preview)).toBe(preparedPreview);
  expect((await get("/v1/collection-onboarding")).workspace.state).toBe(
    "absent",
  );
  await page
    .getByRole("button", { name: "Create private workspace", exact: true })
    .click();
  await expect(
    page.getByText("Private workspace created. No journal was created.", {
      exact: true,
    }),
  ).toBeVisible();
  let lostAck = 0;
  const ackPattern = "**/v1/collection-onboarding/commands/*/acknowledge";
  await page.route(ackPattern, async (route) => {
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    const value = await response.json();
    expect(value.command_id).toBe(preparedID);
    expect(value.state).toBe("committed");
    expect(value.acknowledged).toBe(true);
    await route.abort("failed");
    lostAck++;
  });
  await page
    .getByRole("button", { name: "Acknowledge setup result", exact: true })
    .click();
  await expect.poll(() => lostAck).toBe(1);
  await expect(
    page.getByRole("alert").filter({ hasText: "outcome is unconfirmed" }),
  ).toBeVisible();
  await page.unroute(ackPattern);
  await page
    .getByRole("button", { name: "Check saved private setup", exact: true })
    .click();
  expect((await get("/v1/collection-onboarding/commands/pending")).state).toBe(
    "none",
  );
  await expect(
    page.getByRole("button", { name: "All transactions", exact: true }),
  ).toBeDisabled();
  await page
    .getByRole("button", {
      name: "Retry acknowledging setup result",
      exact: true,
    })
    .click();
  await page
    .getByRole("button", {
      name: "Review connection for Fictional existing records",
      exact: true,
    })
    .click();
  const catalog = await get("/v1/collection-onboarding");
  expect(catalog.workspace.state).toBe("ready");
  expect(catalog.can_setup).toBe(false);
  expect(catalog.can_adopt).toBe(true);
  expect(catalog.candidates).toHaveLength(1);
  expect(catalog.candidates[0].sources).toHaveLength(1);
  await expect(
    page.getByText(
      /Existing automatic bank syncing for this connection will pause/,
    ),
  ).toBeVisible();
  const review = page.getByRole("list", { name: "Reviewed source accounts" });
  await expect(review.getByRole("listitem")).toHaveCount(1);
  await expect(review).toContainText(
    catalog.candidates[0].sources[0].display_name,
  );
  await page
    .getByRole("button", { name: "Save setup review", exact: true })
    .click();
  let lostExecute = 0,
    adoptID = "",
    adoptPreview = "";
  const executePattern = "**/v1/collection-onboarding/commands/*/execute";
  await page.route(executePattern, async (route) => {
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    const value = await response.json();
    expect(value.preview.operation).toBe("adopt");
    expect(value.state).toBe("committed");
    expect(value.preview.review.impact).toBe("pause_legacy_acquisition");
    expect(value.result.item_generation).toBe("2");
    adoptID = value.command_id;
    adoptPreview = JSON.stringify(value.preview);
    await route.abort("failed");
    lostExecute++;
  });
  await page
    .getByRole("button", {
      name: "Adopt reviewed connection history",
      exact: true,
    })
    .click();
  await expect.poll(() => lostExecute).toBe(1);
  await component.unmount();
  await page.unroute(executePattern);
  component = await mount(<OnboardingHarness />);
  await expect(
    page.getByText(
      "Existing connection history adopted. Existing automatic bank syncing paused. No journal was created.",
      { exact: true },
    ),
  ).toBeVisible();
  const committed = await get("/v1/collection-onboarding/commands/pending");
  expect(committed.command.command_id).toBe(adoptID);
  expect(JSON.stringify(committed.command.preview)).toBe(adoptPreview);
  expect(committed.command.acknowledged).toBe(false);
  await page
    .getByRole("button", { name: "Acknowledge setup result", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Fictional adopted cafe purchase",
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "All transactions", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  const workspace = (await get("/v1/collection")).workspace_id;
  const response = await page.request.post(
    new URL(`/v3/collections/${workspace}/review/query`, page.url()).href,
    {
      headers,
      data: {
        profile: "private-collection-review-v3",
        source_account_id: null,
        assignment: "all",
        starts_on: null,
        through_date: null,
        search: "",
        topic: { kind: "all" },
        tag: { kind: "all" },
        merchant: { kind: "all" },
        amount: null,
        institution: { kind: "all" },
        legal_owner: { kind: "all" },
        book: { kind: "all" },
        status: { source: "all", journal: "all" },
        sort: { field: "date", order: "desc" },
        snapshot: null,
        cursor: null,
        limit: 25,
      },
    },
  );
  expect(response.status()).toBe(200);
  const pageResult = await response.json();
  expect(pageResult.matching_count).toBe("2");
  expect(pageResult.next_cursor).toBeNull();
  expect(
    pageResult.items
      .map((r: { money: { amount: string } }) => r.money.amount)
      .sort(),
  ).toEqual(["0.10", "9007199254740993.01"]);
  for (const row of pageResult.items) {
    expect(row.observation_revision).toBe("1");
    expect(row.assignment.state).toBe("assigned");
    expect(row.journal_state).toBe("unposted");
    expect(row.owner_topic.state).toBe("unclassified");
  }
  expect((await get("/v1/collection-onboarding/commands/pending")).state).toBe(
    "none",
  );
  expect((await get("/v1/collection-onboarding")).candidates).toEqual([]);
  expect(
    await page.evaluate(() => [localStorage.length, sessionStorage.length]),
  ).toEqual([0, 0]);
});
