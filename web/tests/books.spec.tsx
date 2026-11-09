import {
  chooseFinanceTool,
  financeNavItem,
  openFinanceMenu,
} from "./finance-navigation";
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
    {
      account_id: "a3",
      code: "5000",
      name: "Supplies",
      category: "expense",
      normal_side: "debit",
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
      valid_to: "2027-01-01",
    },
  ],
  development_policy: "dev-accrual-usd-v1",
};

import type { Page } from "playwright-core";
import type {
  DurableCommand,
  RegisterCommand,
  Journal,
} from "../src/bookkeeping/types";
import type { components } from "../src/lib/api.generated";
// Fixed retained-source draft and historical manual receipt; neither is an editable entry form.
const reviewedPosting: components["schemas"]["PostJournal"] = {
  expected_revision: "0",
  policy_id: "p1",
  accounting_date: "2026-01-10",
  description: "Synthetic split",
  lines: [
    {
      account_id: "a3",
      side: "debit",
      transaction_currency: "USD",
      transaction_amount: "50.00",
      functional_amount: "50.00",
      valuation_basis: "identity",
    },
    {
      account_id: "a2",
      side: "debit",
      transaction_currency: "USD",
      transaction_amount: "50.00",
      functional_amount: "50.00",
      valuation_basis: "identity",
    },
    {
      account_id: "a1",
      side: "credit",
      transaction_currency: "USD",
      transaction_amount: "100.00",
      functional_amount: "100.00",
      valuation_basis: "identity",
    },
  ],
};
const historicalJournal: Journal = {
  entity_id: "e1",
  book_id: "b1",
  journal_id: "journal1",
  period_id: "period1",
  policy_id: "p1",
  book_revision: "1",
  accounting_date: reviewedPosting.accounting_date,
  description: reviewedPosting.description,
  functional_currency: "USD",
  functional_currency_version_id: "c1",
  kind: "manual",
  posted_at: "2026-01-10T00:00:00Z",
  posted_by: "fictional-owner",
  reason: null,
  reverses_journal_id: null,
  lines: reviewedPosting.lines.map((line, index) => ({
    ...line,
    line_number: index + 1,
    transaction_currency_version_id: "c1",
  })),
};
function legacyManual(
  state: DurableCommand["state"] = "prepared",
): DurableCommand {
  return {
    command_id: "legacy-command",
    entity_id: "e1",
    book_id: "b1",
    operation: "post",
    state,
    body: structuredClone(reviewedPosting),
    result: state === "committed" ? historicalJournal : null,
    acknowledged: false,
    created_at: "2026-01-10T00:00:00Z",
  };
}

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
    // This book-only fixture explicitly models the optional private feature as absent.
    if (
      path === "/v1/collection" ||
      path === "/v1/collection-recovery" ||
      path === "/v1/collection-onboarding/commands/pending"
    )
      return route.fulfill({ status: 404, json: { code: "not_found" } });
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
          state.command!.operation === "source_post"
            ? {
                review_id: "r1",
                draft_id: "d1",
                journal_id: "journal1",
                entity_id: "e1",
                book_id: "b1",
                book_revision: "1",
                review_revision: "2",
                source_revision: "1",
                journal: historicalJournal,
              }
            : state.command!.operation === "post"
              ? historicalJournal
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
        entity_id: "e1",
        book_id: "b1",
        through_date: url.searchParams.get("through_date"),
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
          {
            account_id: "a2",
            account_code: "3000",
            debit_balance: "0.000000000",
            credit_balance: "123.450000000",
          },
        ],
      },
    });
  });
  return state;
}
// The only editable choice is accounting treatment of an existing retained observation.
async function prepareSourcePosting(page: Page) {
  const draftRequests: unknown[] = [];
  const row = {
    review_id: "r1",
    description: "Synthetic retained statement purchase",
    booked_date: "2026-01-10",
    amount: "100.00",
    currency: "USD",
    currency_version_id: "c1",
    direction: "outflow",
    state: "needs_review",
    review_revision: "1",
    draft_id: null,
    journal_id: null,
  };
  await page.route("**/v1/entities/e1/books/b1/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    const reply = (json: unknown) => route.fulfill({ json });
    if (path.endsWith("/sources"))
      return reply({
        items: [
          {
            source_account_id: "s1",
            display_name: "Fictional retained statement",
            currency: "USD",
            state: "active",
          },
        ],
      });
    if (path.endsWith("/categories"))
      return reply({
        review_id: "r1",
        source_account_id: "s1",
        book_revision: "0",
        policy_id: "p1",
        accounting_date: row.booked_date,
        source_currency: "USD",
        functional_currency: "USD",
        fx_required: false,
        accounts: [
          {
            account_id: "a3",
            code: "5000",
            name: "Supplies",
            category: "expense",
            normal_side: "debit",
            monetary: false,
          },
        ],
      });
    if (path.endsWith("/drafts")) {
      draftRequests.push(route.request().postDataJSON());
      return reply({
        draft_id: "d1",
        review_id: "r1",
        review_revision: "2",
        source_revision: "1",
        binding_revision: "1",
        book_revision: "0",
        posting_body: reviewedPosting,
      });
    }
    if (path.endsWith("/review-items"))
      return reply({
        source_account_id: "s1",
        source_revision: "1",
        sort: "date",
        order: "desc",
        matching_count: "1",
        start_index: "1",
        items: [row],
        next_cursor: null,
      });
    if (path.endsWith("/review-items/r1"))
      return reply({
        ...row,
        source_account_id: "s1",
        source_revision: "1",
        binding_revision: "1",
        book_revision: "0",
        ledger_account_id: "a1",
        blockers: [],
        draft: null,
      });
    return route.fallback();
  });
  await chooseFinanceTool(page, "Review accounting");
  await page.getByLabel("Entity and book").selectOption("e1/b1");
  await page.getByLabel("Source account").selectOption("s1");
  await page
    .getByRole("button", {
      name: "Review Synthetic retained statement purchase",
    })
    .click();
  await page.getByLabel("Category account").selectOption("a3");
  await page.getByRole("button", { name: "Create reviewed draft" }).click();
  await expect(
    page.getByRole("table", { name: "Exact proposed journal lines" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Review posting command" }).click();
  return draftRequests;
}
test("retained-source draft keeps exact registration identity and recovers lost posting and acknowledgement", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page);
  server.registrationFailures = 1;
  server.executeLost = true;
  server.ackLost = true;
  await mount(<Harness />);
  const drafts = await prepareSourcePosting(page);
  expect(drafts).toHaveLength(1);
  await page.getByRole("button", { name: "Retry saving review" }).click();
  await expect(
    page.getByText("Last checked: ready for your confirmation.", {
      exact: false,
    }),
  ).toBeVisible();
  expect(server.registrations).toHaveLength(2);
  expect(server.registrations[0]).toEqual(server.registrations[1]);
  expect(server.executions).toBe(0);
  expect(JSON.parse(server.registrations[0].body!).operation).toBe(
    "source_post",
  );
  expect(JSON.parse(server.registrations[0].body!).body.review_id).toBe("r1");
  expect(
    JSON.parse(server.registrations[0].body!).body.posting_body.lines,
  ).toEqual(reviewedPosting.lines);
  await page.getByRole("button", { name: "Post journal", exact: true }).click();
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Entity and book")).toBeDisabled();
  expect(server.executions).toBe(1);
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(financeNavItem(page, "Review accounting")).toBeEnabled();
  await expect(page.getByLabel("Entity and book")).toBeEnabled();
  expect(server.executions).toBe(1);
});
test("remount recovers prepared work and a commit winning cancellation stays committed", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page);
  server.cancelRace = true;
  server.command = legacyManual();
  const component = await mount(<Harness />);
  await expect(
    page.getByText("Last checked: ready for your confirmation.", {
      exact: false,
    }),
  ).toBeVisible();
  await component.unmount();
  await mount(<Harness />);
  await expect(
    page.getByText("Last checked: ready for your confirmation.", {
      exact: false,
    }),
  ).toBeVisible();
  expect(server.executions).toBe(0);
  expect(server.registrations).toHaveLength(0);
  await expect(
    page.getByRole("button", { name: "Post journal", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Cancel saved review" }).click();
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Cancellation confirmed.", { exact: false }),
  ).toHaveCount(0);
});
for (const width of [390, 1440]) {
  test(`artifact-backed posting has no blank journal form and legacy work can cancel safely at ${width}px`, async ({
    mount,
    page,
  }) => {
    await page.setViewportSize({ width, height: 900 });
    const server = await mockCommands(page);
    server.command = legacyManual();
    await mount(<Harness />);
    await expect(
      page.getByText(/This saved manual request cannot be posted/),
    ).toBeVisible();
    await expect(financeNavItem(page, "New journal")).toHaveCount(0);
    await expect(financeNavItem(page, "Documents")).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Post journal", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByLabel(/Line [0-9]+ (original|functional) amount/),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Add split line", includeHidden: true }),
    ).toHaveCount(0);
    expect(server.command?.body).toEqual(reviewedPosting);
    await expect(
      page
        .getByRole("table", { name: "Journal lines for review" })
        .getByRole("row"),
    ).toHaveCount(4);
    await page.getByRole("button", { name: "Cancel saved review" }).click();
    await expect(page.getByText(/Cancellation confirmed/)).toBeVisible();
    expect(server.executions).toBe(0);
    expect(server.registrations).toHaveLength(0);
    await page.getByRole("button", { name: "Acknowledge result" }).click();
    await expect(financeNavItem(page, "Review accounting")).toBeEnabled();
    expect(server.executions).toBe(0);
    expect(server.command).toBeNull();
  });
}
test("setup registers review then executes and explicitly acknowledges", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page, true);
  await mount(<Harness />);
  await page
    .getByText("Create a custom entity and book", { exact: true })
    .click();
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
  await expect(financeNavItem(page, "Review accounting")).toBeEnabled();
});
test("Sandbox starter prepares the bounded reviewed setup without manual fields", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page, true);
  await mount(<Harness />);
  await page
    .getByRole("button", { name: "Review Sandbox starter book", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Create book", exact: true }),
  ).toBeVisible();
  expect(server.executions).toBe(0);
  expect(server.registrations).toHaveLength(1);
  const request = JSON.parse(server.registrations[0].body!);
  expect(request).toEqual({
    operation: "setup",
    entity_id: null,
    book_id: null,
    body: {
      entity_name: "Personal Sandbox",
      entity_kind: "personal",
      book_name: "Sandbox bank connections",
      starts_on: "2026-01-01",
      ends_before: "2027-01-01",
      functional_currency_version_id: "c1",
      accounts: [
        {
          code: "1000",
          name: "Sandbox checking",
          category: "asset",
          normal_side: "debit",
          monetary: true,
        },
        {
          code: "3000",
          name: "Opening equity",
          category: "equity",
          normal_side: "credit",
          monetary: false,
        },
        {
          code: "4000",
          name: "Uncategorized income",
          category: "revenue",
          normal_side: "credit",
          monetary: false,
        },
        {
          code: "5000",
          name: "Uncategorized expenses",
          category: "expense",
          normal_side: "debit",
          monetary: false,
        },
      ],
    },
  });
});
test("denied entity switch clears previous financial values", async ({
  mount,
  page,
}) => {
  await mockCommands(page);
  await mount(<Harness />);
  await page.getByLabel("Entity and book").selectOption("e1/b1");
  await chooseFinanceTool(page, "Chart of accounts");
  await page.getByRole("button", { name: "Load trial balance" }).click();
  await expect(page.getByRole("table")).toBeVisible();
  await expect(
    page.getByRole("cell", { name: "123.45", exact: true }),
  ).toHaveCount(4);
  await expect(
    page.getByRole("cell", { name: "0.00", exact: true }),
  ).toHaveCount(2);
  await page.getByLabel("Entity and book").selectOption("e2/b2");
  await expect(page.getByRole("table")).toHaveCount(0);
  await expect(page.getByLabel("Entity and book")).toHaveValue("");
  await expect(page.getByText("123.45", { exact: true })).toHaveCount(0);
});

test("denied recovery blocks new work and clears a previously visible command", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page);
  server.command = legacyManual();
  await mount(<Harness />);
  await expect(
    page.getByText("Synthetic split", { exact: true }),
  ).toBeVisible();
  await page.route("**/v1/commands/pending", (route) =>
    route.fulfill({ status: 404, json: { code: "not_found" } }),
  );
  await page.getByRole("button", { name: "Check saved work" }).click();
  await expect(page.getByRole("alert")).toContainText("access was denied");
  await expect(page.getByText("Synthetic split", { exact: true })).toHaveCount(
    0,
  );
  await expect(page.getByLabel("Entity and book")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Review journal", exact: true }),
  ).toHaveCount(0);
});

test("explicitly clears inaccessible saved work before first-book setup", async ({
  mount,
  page,
}) => {
  await mockCommands(page, true);
  let cleared = false;
  await page.route("**/v1/commands/pending**", (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/v1/commands/pending/clear") {
      cleared = true;
      return route.fulfill({ json: { cleared: true } });
    }
    return cleared
      ? route.fulfill({ json: { command: null } })
      : route.fulfill({ status: 403, json: { code: "permission_denied" } });
  });
  await mount(<Harness />);
  await expect(
    page.getByRole("button", { name: "Clear unavailable saved work" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Review Sandbox starter book" }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "Clear unavailable saved work" })
    .click();
  await expect(
    page.getByRole("button", { name: "Review Sandbox starter book" }),
  ).toBeVisible();
  await expect(financeNavItem(page, "Review accounting")).toBeEnabled();
});

test("successful acknowledgement followed by failed discovery keeps new work locked", async ({
  mount,
  page,
}) => {
  const server = await mockCommands(page);
  server.command = legacyManual("committed");
  await mount(<Harness />);
  await expect(
    page.getByText("Posting confirmed", { exact: true }),
  ).toBeVisible();
  await page.route("**/v1/commands/pending", (route) =>
    route.fulfill({ status: 503, json: { code: "temporarily_unavailable" } }),
  );
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(page.getByLabel("Entity and book")).toBeDisabled();
  await openFinanceMenu(page);
  await expect(financeNavItem(page, "New journal")).toHaveCount(0);
  await expect(financeNavItem(page, "Posted entries")).toBeDisabled();
});

// Product navigation follows one pinned account to its journal and back without a fake transaction feed.
test("book workspace balances, account activity and journal remain scoped on desktop and mobile", async ({
  mount,
  page,
}, testInfo) => {
  await mockCommands(page);
  let activityReads = 0;
  await page.route("**/reports/**", async (route) => {
    const url = new URL(route.request().url());
    if (!url.pathname.includes("/general-ledger")) return route.fallback();
    expect(url.searchParams.get("snapshot_revision")).toBe("0");
    expect(url.searchParams.get("account_id")).toBe("a1");
    const common = { entity_id: "e1", book_id: "b1", snapshot_revision: "0" };
    if (url.pathname.includes("/journals/"))
      return route.fulfill({
        json: {
          ...common,
          contract_version: "report-journal-v1",
          selector: {
            account_id: "a1",
            starts_on: "2026-01-01",
            ends_before: "2026-02-01",
            journal_id: "journal-one",
            line_number: 1,
          },
          journal: {
            entity_id: "e1",
            book_id: "b1",
            journal_id: "journal-one",
            book_revision: "0",
            functional_currency_version_id: "c1",
            accounting_date: "2026-01-15",
            kind: "manual",
            policy_id: "p1",
            description: "Synthetic reviewed capital",
            lines: [
              {
                line_number: 1,
                account_id: "a1",
                side: "debit",
                transaction_currency: "USD",
                transaction_amount: "123.45",
                functional_amount: "123.45",
                valuation_basis: "identity",
              },
              {
                line_number: 2,
                account_id: "a2",
                side: "credit",
                transaction_currency: "USD",
                transaction_amount: "123.45",
                functional_amount: "123.45",
                valuation_basis: "identity",
              },
            ],
          },
        },
      });
    activityReads++;
    return route.fulfill({
      json: {
        ...common,
        contract_version: "financial-reporting-v1",
        report: "general-ledger",
        layout: "posting-v1",
        generated_at: "2026-01-31T00:00:00Z",
        basis: "accrual",
        timezone: "UTC",
        functional_currency: "USD",
        functional_currency_version_id: "c1",
        minor_units: 2,
        policy_ids: ["p1"],
        filters: {
          report: "general-ledger",
          account_id: "a1",
          starts_on: "2026-01-01",
          ends_before: "2026-02-01",
        },
        comparison: null,
        next_cursor: null,
        totals: {
          opening: "0",
          period_debits: "123.45",
          period_credits: "0",
          closing: "123.45",
        },
        rows: [
          {
            id: "line:journal-one:1",
            journal_id: "journal-one",
            line_number: 1,
            accounting_date: "2026-01-15",
            amounts: {
              debit: "123.45",
              credit: "0",
              running_balance: "123.45",
            },
            transaction_amount: "123.45",
            transaction_currency: "USD",
            valuation_basis: "identity",
            kind: "manual",
            policy_id: "p1",
          },
        ],
      },
    });
  });
  await mount(<Harness />);
  await page.getByLabel("Entity and book").selectOption("e1/b1");
  await chooseFinanceTool(page, "Chart of accounts");
  await expect(
    page.getByRole("heading", { name: "Account balances", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Review journal", exact: true }),
  ).toHaveCount(0);
  await expect(page.locator("tfoot td")).toHaveText(["123.45", "123.45"]);
  await page.getByLabel("Through date", { exact: true }).fill("2026-01-31");
  await page.getByLabel("Activity ends before").fill("2026-02-01");
  await expect(page.locator("tfoot td")).toHaveText(["123.45", "123.45"]);
  await page.screenshot({
    path: testInfo.outputPath("book-workspace-desktop.png"),
    fullPage: true,
  });
  await testInfo.attach("book-workspace-desktop", {
    path: testInfo.outputPath("book-workspace-desktop.png"),
    contentType: "image/png",
  });
  await page.getByRole("button", { name: "1000 Cash", exact: true }).click();
  await expect(page.getByLabel("Report account")).toHaveValue("a1");
  await page
    .getByRole("button", { name: "Review journal line 1", exact: true })
    .click();
  await expect(
    page.getByRole("region", { name: "Journal detail" }),
  ).toContainText("Synthetic reviewed capital");
  await page
    .getByRole("button", { name: "Close journal detail", exact: true })
    .click();
  await openFinanceMenu(page);
  await financeNavItem(page, "Chart of accounts").focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("heading", { name: "Chart of accounts", exact: true }),
  ).toBeFocused();
  await expect(page.getByLabel("Through date", { exact: true })).toHaveValue(
    "2026-01-31",
  );
  await expect(page.getByLabel("Activity ends before")).toHaveValue(
    "2026-02-01",
  );
  await expect(page.locator("tfoot td")).toHaveText(["123.45", "123.45"]);
  expect(activityReads).toBe(1);
  await page.setViewportSize({ width: 390, height: 844 });
  // Wider glyph metrics reproduce the intrinsic grid pressure seen on Linux CI.
  await page.addStyleTag({
    content: ".book-nav button { font-family: monospace; }",
  });
  await openFinanceMenu(page);
  await expect(
    page.getByRole("navigation", { name: "Finance tools" }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: testInfo.outputPath("book-workspace-mobile.png"),
    fullPage: true,
  });
  await testInfo.attach("book-workspace-mobile", {
    path: testInfo.outputPath("book-workspace-mobile.png"),
    contentType: "image/png",
  });
  await page.getByLabel("Entity and book").selectOption("e2/b2");
  await expect(page.getByLabel("Entity and book")).toHaveValue("");
  await expect(page.getByText("123.45", { exact: true })).toHaveCount(0);
});

// An import prepared by another client must never be mistaken for setup or silently executed.
test("recovers an import safely until statement preview UI is available", async ({
  mount,
  page,
}) => {
  const state = await mockCommands(page);
  state.command = {
    command_id: "import-command",
    operation: "import",
    entity_id: "e1",
    book_id: "b1",
    state: "prepared",
    acknowledged: false,
    result: null,
    created_at: "2026-01-01T00:00:00Z",
    body: {
      batch_id: "batch1",
      preview_id: "preview1",
      preview_revision: "1",
      binding_revision: "1",
      source_revision: "0",
      profile: "jayburd-synthetic-statement-csv/1",
    },
  };
  await mount(<Harness />);
  await expect(
    page.getByRole("button", { name: "Review import in originating client" }),
  ).toBeDisabled();
  expect(state.executions).toBe(0);
  await page.getByRole("button", { name: "Cancel saved review" }).click();
  await expect(page.getByText(/Cancellation confirmed/)).toBeVisible();
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(financeNavItem(page, "Review accounting")).toBeEnabled();
  expect(state.executions).toBe(0);
});
