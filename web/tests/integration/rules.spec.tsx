import { chooseFinanceTool } from "../finance-navigation";
import { test, expect } from "@playwright/experimental-ct-react";
import { RulesHarness } from "./RulesHarness";

const food = "00000000-0000-4000-8000-000000000931";
const travel = "00000000-0000-4000-8000-000000000932";
type RuleRow = {
  source_id: string;
  observation_id: string;
  observation_revision: string;
  booked_date: string;
  description: string;
  merchant: string | null;
  source_state: string;
  money: { amount: string | null; currency: string | null };
  assignment: { state: string };
  journal_state: string;
  owner_topic: { state: string; topic_id?: string };
  rule_suggestion: { status: string; will_apply: boolean };
  rule_application: {
    state: string;
    fields?: { topic: boolean; tags: boolean };
  };
};

/** Fix original source evidence independently of analytic labels/provenance. */
function sourceEvidence(rows: RuleRow[]) {
  return rows.map((row) => ({
    source_id: row.source_id,
    observation_id: row.observation_id,
    observation_revision: row.observation_revision,
    booked_date: row.booked_date,
    description: row.description,
    merchant: row.merchant,
    source_state: row.source_state,
    money: row.money,
    assignment: row.assignment,
    journal_state: row.journal_state,
  }));
}

/** Only delivery is fault-injected; no suggestion or saved result is manufactured. */
test("private rules preserve manual fields, abstain on conflicts and recover an exact committed application", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  let component = await mount(<RulesHarness />);
  await chooseFinanceTool(page, "Transactions");
  await expect(
    page.getByRole("button", {
      name: "Fictional rule cafe purchase",
      exact: true,
    }),
  ).toBeVisible();
  const headers = {
    Authorization: "Bearer synthetic.rules.proof",
    Origin: new URL(page.url()).origin,
  };
  const discovery = await page.request.get(
    new URL("/v1/collection", page.url()).href,
    { headers },
  );
  expect(discovery.status()).toBe(200);
  const workspace = (await discovery.json()).workspace_id;
  const base = `/v3/collections/${workspace}`;
  const catalog = async () => {
    const response = await page.request.get(
      new URL(base + "/taxonomy", page.url()).href,
      { headers },
    );
    expect(response.status()).toBe(200);
    return response.json();
  };
  const query = async (): Promise<RuleRow[]> => {
    const response = await page.request.post(
      new URL(base + "/review/query", page.url()).href,
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
    const body = await response.json();
    expect(body.matching_count).toBe("3");
    expect(body.next_cursor).toBeNull();
    return body.items as RuleRow[];
  };
  const original = await query();
  expect(
    original.find(
      (r) => r.description === "Fictional rule exact large quantity",
    )?.money.amount,
  ).toBe("9007199254740993.01");
  expect(
    original.find((r) => r.description === "Fictional rule protected choice")
      ?.owner_topic.topic_id,
  ).toBe(travel);
  expect((await catalog()).rules).toEqual([]);

  await page.getByText("Manage rules", { exact: true }).click();
  const addRule = async (name: string, merchant: string, topic: string) => {
    // Add prepends an explicit empty draft; no suggested condition/output is accepted implicitly.
    await page.getByRole("button", { name: "Add rule", exact: true }).click();
    await page.getByLabel("Rule 1 name", { exact: true }).fill(name);
    await page
      .getByLabel("Rule 1 condition 1 field", { exact: true })
      .selectOption("merchant");
    await page
      .getByLabel("Rule 1 condition 1 operator", { exact: true })
      .selectOption("exact");
    await page
      .getByLabel("Rule 1 condition 1 value", { exact: true })
      .fill(merchant);
    await page
      .getByLabel("Rule 1 primary topic", { exact: true })
      .selectOption(topic);
  };
  await addRule("Fictional cafe rule", "Fictional Rules Cafe", food);
  await addRule("Fictional mega food rule", "Fictional Rules Mega", food);
  await addRule("Fictional mega travel rule", "Fictional Rules Mega", travel);
  await page
    .getByRole("button", { name: "Preview rule changes", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Review rule changes", exact: true }),
  ).toBeVisible();
  expect((await catalog()).rules).toEqual([]);
  expect(sourceEvidence(await query())).toEqual(sourceEvidence(original));
  await page
    .getByRole("button", { name: "Save rule review", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Saved rule review", exact: true }),
  ).toBeVisible();
  expect((await catalog()).rules).toEqual([]);
  await page
    .getByRole("button", { name: "Confirm rule changes", exact: true })
    .click();
  await expect(
    page.getByText("Rules updated. No journal was created.", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Fictional rule cafe purchase",
      exact: true,
    }),
  ).toBeVisible();
  expect((await catalog()).rules).toHaveLength(3);
  const proposed = await query();
  expect(sourceEvidence(proposed)).toEqual(sourceEvidence(original));
  expect(
    proposed.every((row) => row.rule_suggestion.will_apply === false),
  ).toBe(true);
  expect(proposed.every((row) => row.rule_application.state === "none")).toBe(
    true,
  );
  expect(
    proposed.find(
      (r) => r.description === "Fictional rule exact large quantity",
    )?.rule_suggestion.status,
  ).toBe("conflict");

  for (const description of [
    "Fictional rule protected choice",
    "Fictional rule cafe purchase",
    "Fictional rule exact large quantity",
  ]) {
    await page
      .getByRole("checkbox", { name: new RegExp(`^Select ${description} on `) })
      .check();
  }
  await page
    .getByRole("button", { name: "Preview rules for 3 selected", exact: true })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "Review rules for 3 selected",
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByText(
      "Eligible reviewed results: 2. Unchanged selected results: 1.",
      { exact: true },
    ),
  ).toBeVisible();
  const reviewed = page.getByRole("list", {
    name: "Reviewed rule results",
    exact: true,
  });
  await expect(
    reviewed
      .getByRole("listitem")
      .filter({ hasText: "Fictional rule protected choice" })
      .getByText("Preserved manual fields: primary topic.", { exact: true }),
  ).toBeVisible();
  await expect(
    reviewed
      .getByRole("listitem")
      .filter({ hasText: "Fictional rule exact large quantity" })
      .getByText("conflict · No change", { exact: true }),
  ).toBeVisible();
  expect(
    (await query()).every((row) => row.rule_application.state === "none"),
  ).toBe(true);
  await page
    .getByRole("button", { name: "Save rule application", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Saved rule application", exact: true }),
  ).toBeVisible();
  expect(
    (await query()).every((row) => row.rule_application.state === "none"),
  ).toBe(true);
  let executions = 0;
  let committedPreview = "";
  await page.route(
    "**/v3/collections/*/label-commands/*/execute",
    async (route) => {
      const response = await route.fetch();
      expect(response.status()).toBe(200);
      const receipt = await response.json();
      expect(receipt.command.state).toBe("committed");
      expect(receipt.command.preview.action.kind).toBe("rules_apply");
      expect(receipt.command.preview.apply_count).toBe("2");
      expect(receipt.command.preview.abstain_count).toBe("1");
      committedPreview = JSON.stringify(receipt.command.preview);
      await route.abort("failed");
      executions++;
    },
  );
  await page
    .getByRole("button", { name: "Confirm rules for 3 selected", exact: true })
    .click();
  await expect.poll(() => executions).toBe(1);
  await component.unmount();
  await page.unroute("**/v3/collections/*/label-commands/*/execute");
  component = await mount(<RulesHarness />);
  await expect(
    page.getByText("Rule results applied. No journal was created.", {
      exact: true,
    }),
  ).toBeVisible();
  const pendingResponse = await page.request.get(
    new URL(base + "/label-commands/pending", page.url()).href,
    { headers },
  );
  expect(pendingResponse.status()).toBe(200);
  const pending = await pendingResponse.json();
  expect(pending.command.state).toBe("committed");
  expect(JSON.stringify(pending.command.preview)).toBe(committedPreview);
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Fictional rule cafe purchase",
      exact: true,
    }),
  ).toBeVisible();
  expect(executions).toBe(1);
  const applied = await query();
  expect(sourceEvidence(applied)).toEqual(sourceEvidence(original));
  const protectedRow = applied.find(
    (r) => r.description === "Fictional rule protected choice",
  );
  expect(protectedRow?.owner_topic.topic_id).toBe(travel);
  expect(protectedRow?.rule_application.fields).toEqual({
    topic: false,
    tags: true,
  });
  expect(
    applied.find((r) => r.description === "Fictional rule cafe purchase")
      ?.owner_topic.topic_id,
  ).toBe(food);
  expect(
    applied.find((r) => r.description === "Fictional rule exact large quantity")
      ?.rule_application.state,
  ).toBe("none");
  expect(
    applied.every(
      (row) =>
        row.assignment.state === "unassigned" &&
        row.journal_state === "unposted",
    ),
  ).toBe(true);
  expect(
    (
      await (
        await page.request.get(
          new URL(base + "/label-commands/pending", page.url()).href,
          { headers },
        )
      ).json()
    ).command,
  ).toBeNull();
  expect(
    await page.evaluate(() => ({
      local: Object.keys(localStorage),
      session: Object.keys(sessionStorage),
    })),
  ).toEqual({ local: [], session: [] });
  await component.unmount();
});
