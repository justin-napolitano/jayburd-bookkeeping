import { test, expect } from "@playwright/experimental-ct-react";
import { Harness } from "./Harness";
const book = {
  book_id: "b1",
  name: "Operating",
  book_revision: "0",
  policies: [
    {
      policy_id: "p1",
      functional_currency: "USD",
      functional_currency_version_id: "c1",
      effective_from: "2026-01-01",
      effective_to: null,
    },
  ],
  periods: [],
  accounts: [
    {
      account_id: "a1",
      code: "1000",
      name: "Cash",
      category: "asset",
      normal_side: "debit",
      monetary: true,
      active_from: "2026-01-01",
      active_to: null,
    },
    {
      account_id: "a2",
      code: "3000",
      name: "Capital",
      category: "equity",
      normal_side: "credit",
      monetary: false,
      active_from: "2026-01-01",
      active_to: null,
    },
  ],
};
const workspace = {
  entities: [
    {
      entity_id: "e1",
      name: "Synthetic LLC",
      kind: "legal",
      permissions: ["read", "post"],
      books: [book],
    },
    {
      entity_id: "e2",
      name: "Other synthetic LLC",
      kind: "legal",
      permissions: ["read"],
      books: [{ ...book, book_id: "b2" }],
    },
  ],
  currencies: [
    {
      currency_version_id: "c1",
      alphabetic_code: "USD",
      minor_unit: 2,
      valid_from: "2026-01-01",
      valid_to: null,
    },
  ],
  development_policy: "dev-accrual-usd-v1",
};

import type { Page } from "playwright-core";
import type { DurableCommand, RegisterCommand } from "../src/bookkeeping/types";
// This stateful synthetic server models only the command transport contract, never accounting rules.
async function mockCommands(page: Page, empty = false) {
  const state: {
    command: DurableCommand | null;
    registrations: { key: string | undefined; body: string | null }[];
    executions: number;
    registrationFailures: number;
    executeLost: boolean;
    ackLost: boolean;
    cancelRace: boolean;
    workspace: typeof workspace;
  } = {
    command: null,
    registrations: [],
    executions: 0,
    registrationFailures: 0,
    executeLost: false,
    ackLost: false,
    cancelRace: false,
    workspace: empty ? { ...workspace, entities: [] } : workspace,
  };
  await page.route("**/v1/**", async (route) => {
    const req = route.request(),
      url = new URL(req.url()),
      path = url.pathname;
    if (path === "/v1/workspace")
      return route.fulfill({ json: state.workspace });
    if (path === "/v1/commands/pending")
      return route.fulfill({ json: { command: state.command } });
    if (path === "/v1/commands") {
      state.registrations.push({
        key: req.headers()["idempotency-key"],
        body: req.postData(),
      });
      if (state.registrationFailures-- > 0)
        return route.fulfill({
          status: 503,
          json: { code: "temporarily_unavailable" },
        });
      const input = req.postDataJSON() as RegisterCommand;
      state.command = {
        ...input,
        command_id: "command1",
        state: "prepared",
        result: null,
        acknowledged: false,
        created_at: "2026-10-11T00:00:00Z",
      };
      return route.fulfill({ status: 201, json: state.command });
    }
    if (
      path.endsWith("/execute") ||
      (path.endsWith("/cancel") && state.cancelRace)
    ) {
      state.executions++;
      state.command = {
        ...state.command!,
        state: "committed",
        result:
          state.command!.operation === "post"
            ? ({ journal_id: "journal1", book_revision: "1" } as never)
            : ({ entity_id: "entity1", book_id: "book1" } as never),
      };
      if (state.executeLost) {
        state.executeLost = false;
        return route.fulfill({
          status: 503,
          json: { code: "temporarily_unavailable" },
        });
      }
      return route.fulfill({ json: state.command });
    }
    if (path.endsWith("/cancel")) {
      state.command = { ...state.command!, state: "cancelled" };
      return route.fulfill({ json: state.command });
    }
    if (path.endsWith("/acknowledge")) {
      const result = { ...state.command!, acknowledged: true };
      state.command = null;
      if (state.ackLost) {
        state.ackLost = false;
        return route.fulfill({
          status: 503,
          json: { code: "temporarily_unavailable" },
        });
      }
      return route.fulfill({ json: result });
    }
    if (path.includes("/e2/"))
      return route.fulfill({ status: 404, json: { code: "not_found" } });
    return route.fulfill({
      json: {
        through_date: "2026-01-01",
        snapshot_revision: "0",
        functional_currency: "USD",
        total_debits: "123.450000000",
        total_credits: "123.450000000",
        accounts: [
          {
            account_id: "a1",
            account_code: "1000",
            debit_balance: "123.450000000",
            credit_balance: "0.000000000",
          },
        ],
      },
    });
  });
  return state;
}
// Use accessible controls and exact decimal strings to prepare a split journal.
async function fillJournal(page: Page) {
  await page.getByLabel("Entity and book").selectOption("e1/b1");
  await page
    .getByRole("combobox", { name: "Policy", exact: true })
    .selectOption("p1");
  await page.getByLabel("Description", { exact: true }).fill("Synthetic split");
  await page.getByRole("button", { name: "Add split line" }).click();
  for (let i = 1; i <= 3; i++) {
    await page
      .getByLabel(`Line ${i} account`, { exact: true })
      .selectOption(i === 2 ? "a2" : "a1");
    await page
      .getByLabel(`Line ${i} original amount`, { exact: true })
      .fill(i === 2 ? "100.00" : "50.00");
    await page
      .getByLabel(`Line ${i} functional amount`, { exact: true })
      .fill(i === 2 ? "100.00" : "50.00");
  }
  await page
    .getByRole("button", { name: "Review journal", exact: true })
    .click();
}
test("split review retains registration identity and discovers lost execution and acknowledgement", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page);
  server.registrationFailures = 1;
  server.executeLost = true;
  server.ackLost = true;
  await mount(<Harness />);
  await fillJournal(page);
  await page.getByRole("button", { name: "Retry registration" }).click();
  await expect(
    page.getByText("Saved for review.", { exact: false }),
  ).toBeVisible();
  expect(server.registrations).toHaveLength(2);
  expect(server.registrations[0]).toEqual(server.registrations[1]);
  expect(server.executions).toBe(0);
  expect(JSON.parse(server.registrations[0].body!).body.lines).toHaveLength(3);
  await page.getByRole("button", { name: "Post journal", exact: true }).click();
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Entity and book")).toBeDisabled();
  expect(server.executions).toBe(1);
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(
    page.getByText("No unacknowledged command.", { exact: false }),
  ).toBeVisible();
  await expect(page.getByLabel("Entity and book")).toBeEnabled();
  expect(server.executions).toBe(1);
});
test("remount recovers prepared work and a commit winning cancellation stays committed", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page);
  server.cancelRace = true;
  const component = await mount(<Harness />);
  await fillJournal(page);
  await expect(
    page.getByText("Saved for review.", { exact: false }),
  ).toBeVisible();
  await component.unmount();
  await mount(<Harness />);
  await expect(
    page.getByText("Saved for review.", { exact: false }),
  ).toBeVisible();
  expect(server.executions).toBe(0);
  expect(server.registrations).toHaveLength(1);
  await page.getByRole("button", { name: "Cancel saved command" }).click();
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Cancellation confirmed.", { exact: false }),
  ).toHaveCount(0);
});
test("setup registers review then executes and explicitly acknowledges", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page, true);
  await mount(<Harness />);
  await page.getByText("Create an entity and book", { exact: true }).click();
  await page.getByLabel("Entity name", { exact: true }).fill("Fictional LLC");
  await page.getByLabel("Book name", { exact: true }).fill("Operating");
  await page.getByLabel("Period starts on").fill("2026-01-01");
  await page.getByLabel("Period ends before").fill("2027-01-01");
  await page.getByLabel("Functional currency metadata").selectOption("c1");
  for (let i = 1; i <= 2; i++) {
    await page
      .getByLabel(`Account ${i} code`, { exact: true })
      .fill(i === 1 ? "1000" : "3000");
    await page
      .getByLabel(`Account ${i} name`, { exact: true })
      .fill(i === 1 ? "Cash" : "Capital");
  }
  await page.getByRole("button", { name: "Review book setup" }).click();
  await expect(
    page.getByRole("button", { name: "Create book", exact: true }),
  ).toBeVisible();
  expect(server.executions).toBe(0);
  await page.getByRole("button", { name: "Create book", exact: true }).click();
  await expect(
    page.getByText("Book creation confirmed", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(
    page.getByText("No unacknowledged command.", { exact: false }),
  ).toBeVisible();
});
test("denied entity switch clears previous financial values", async ({
  mount,
  page,
}) => {
  await mockCommands(page);
  await mount(<Harness />);
  await page.getByLabel("Entity and book").selectOption("e1/b1");
  await page.getByRole("button", { name: "Load trial balance" }).click();
  await expect(page.getByRole("table")).toBeVisible();
  await expect(
    page.getByRole("cell", { name: "123.45", exact: true }),
  ).toHaveCount(3);
  await expect(
    page.getByRole("cell", { name: "0.00", exact: true }),
  ).toHaveCount(1);
  await page.getByLabel("Entity and book").selectOption("e2/b2");
  await expect(page.getByRole("table")).toHaveCount(0);
  await page.getByRole("button", { name: "Load trial balance" }).click();
  await expect(page.getByLabel("Entity and book")).toHaveValue("");
  await expect(page.getByText("123.45", { exact: true })).toHaveCount(0);
});

test("denied recovery blocks new work and clears a previously visible command", async ({
  mount,
  page,
}) => {
  await mockCommands(page);
  await mount(<Harness />);
  await fillJournal(page);
  await expect(
    page.getByText("Synthetic split", { exact: true }),
  ).toBeVisible();
  await page.route("**/v1/commands/pending", (route) =>
    route.fulfill({ status: 404, json: { code: "not_found" } }),
  );
  await page.getByRole("button", { name: "Check saved command" }).click();
  await expect(page.getByRole("alert")).toContainText("access was denied");
  await expect(page.getByText("Synthetic split", { exact: true })).toHaveCount(
    0,
  );
  await expect(page.getByLabel("Entity and book")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Review journal", exact: true }),
  ).toHaveCount(0);
});

test("successful acknowledgement followed by failed discovery keeps new work locked", async ({
  mount,
  page,
}) => {
  await mockCommands(page);
  await mount(<Harness />);
  await fillJournal(page);
  await page.getByRole("button", { name: "Post journal", exact: true }).click();
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await page.route("**/v1/commands/pending", (route) =>
    route.fulfill({ status: 503, json: { code: "temporarily_unavailable" } }),
  );
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(page.getByLabel("Entity and book")).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Review journal", exact: true }),
  ).toBeDisabled();
});
