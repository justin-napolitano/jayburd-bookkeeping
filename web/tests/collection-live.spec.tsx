import { chooseFinanceTool, financeNavItem } from "./finance-navigation";
import { expect, test } from "@playwright/experimental-ct-react";
import type { Page } from "playwright-core";
import type { DurableCommand } from "../src/bookkeeping/types";
import { Harness } from "./Harness";
import {
  collectionRow,
  collectionSelector,
  collectionSnapshot,
} from "./collection-fixture";
import type {
  LabelCommand,
  Preview,
  Query,
  Topic,
} from "../src/collection/types";

const workspace = "10000000-0000-4000-8000-000000000001";
const source = "20000000-0000-4000-8000-000000000001";
const topic = "40000000-0000-4000-8000-000000000001";
// Independent HTTP fixtures exercise the production authenticated client; no application auth/store substitute.
async function serve(
  page: Page,
  scenario:
    | "normal"
    | "lost_prepare"
    | "pending_taxonomy"
    | "discovery_failure"
    | "book_unknown"
    | "private_prepare_conflict"
    | "book_prepare_conflict" = "normal",
) {
  const state = {
    taxonomy: "3",
    decision: "0",
    pending: null as LabelCommand | null,
    bookPending: null as DurableCommand | null,
    preview: null as Preview | null,
    executions: 0,
    acknowledgements: 0,
    dropPrepare: scenario === "lost_prepare",
    discoveryFails: scenario === "discovery_failure",
    denyQueries: false,
    capabilities: ["read", "classify"],
    topics: [
      { topic_id: topic, name: "Food", parent_id: null, status: "active" },
    ] as Topic[],
    requests: [] as { path: string; body: unknown; key?: string }[],
  };
  const snapshot = () => ({
    ...collectionSnapshot,
    taxonomy_revision: state.taxonomy,
    decision_revision: state.decision,
  });
  const row = (index: number) =>
    collectionRow(index, {
      workspace_id: workspace,
      source_id: source,
      observation_id: `30000000-0000-4000-8000-${String(index + 1).padStart(12, "0")}`,
      account_name: "Fictional private checking",
      provider_suggestion: { state: "unavailable" },
    });
  if (scenario === "pending_taxonomy") {
    const preview: Preview = {
      profile: "private-collection-preview-v3",
      rule_results: [],
      apply_count: "0",
      abstain_count: "0",
      tag_results: [],
      workspace_id: workspace,
      preview_id: "50000000-0000-4000-8000-000000000001",
      expires_at: "2099-01-01T00:00:00Z",
      snapshot: snapshot(),
      action: {
        kind: "taxonomy",
        topics: [{ ...state.topics[0], name: "Meals" }],
      },
      selection: [],
      selected_count: "0",
      review_rows: [],
    };
    state.pending = {
      workspace_id: workspace,
      command_id: "60000000-0000-4000-8000-000000000001",
      prepare_key: "70000000-0000-4000-8000-000000000001",
      state: "prepared",
      acknowledged: false,
      preview,
    };
  }
  await page.route("**/v*/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const body = request.postData() ? request.postDataJSON() : undefined;
    state.requests.push({
      path,
      body,
      key: request.headers()["idempotency-key"],
    });
    if (path === "/v1/workspace")
      return route.fulfill({
        json: {
          entities: [],
          currencies: [
            {
              currency_version_id: "90000000-0000-4000-8000-000000000001",
              alphabetic_code: "USD",
              minor_unit: 2,
              valid_from: "2026-01-01",
              valid_to: "2027-01-01",
            },
          ],
          development_policy: "dev-accrual-usd-v1",
        },
      });
    if (path === "/v1/commands/pending")
      return scenario === "book_unknown"
        ? route.fulfill({
            status: 503,
            json: { code: "temporarily_unavailable" },
          })
        : route.fulfill({ json: { command: state.bookPending } });
    if (path === "/v1/commands" && scenario === "book_prepare_conflict") {
      state.pending = {
        workspace_id: workspace,
        command_id: "60000000-0000-4000-8000-000000000002",
        prepare_key: "70000000-0000-4000-8000-000000000002",
        state: "prepared",
        acknowledged: false,
        preview: {
          profile: "private-collection-preview-v3",
          rule_results: [],
          apply_count: "0",
          abstain_count: "0",
          tag_results: [],
          workspace_id: workspace,
          preview_id: "50000000-0000-4000-8000-000000000002",
          expires_at: "2099-01-01T00:00:00Z",
          snapshot: snapshot(),
          selection: [],
          selected_count: "0",
          review_rows: [],
          action: {
            kind: "taxonomy",
            topics: [
              { ...state.topics[0], name: "Other context private winner" },
            ],
          },
        },
      };
      return route.fulfill({ status: 409, json: { code: "pending_command" } });
    }
    if (path === "/v1/collection")
      return state.discoveryFails
        ? route.fulfill({
            status: 503,
            json: { code: "temporarily_unavailable" },
          })
        : route.fulfill({
            json: {
              profile: "private-source-workspace-v1",
              workspace_id: workspace,
              workspace_revision: "1",
              capabilities: state.capabilities,
            },
          });
    if (path === `/v3/collections/${workspace}/taxonomy`)
      return route.fulfill({
        json: {
          profile: "private-collection-catalog-v3",
          workspace_id: workspace,
          authority_revision: "1",
          taxonomy_revision: state.taxonomy,
          ruleset_revision: "4",
          map_revision: "0",
          tag_revision: "1",
          books: [],
          currencies: [],
          institution: { state: "unavailable" },
          legal_owner: { state: "unknown" },
          rules: [],
          tags: [],
          topics: state.topics,
          sources: [{ source_id: source, name: "Fictional private checking" }],
        },
      });
    if (path.endsWith("/review/query")) {
      if (state.denyQueries)
        return route.fulfill({ status: 404, json: { code: "not_found" } });
      const q = { ...body, workspace_id: workspace } as Query;
      const start = q.cursor ? 25 : 0;
      return route.fulfill({
        json: {
          profile: "private-collection-review-v3",
          workspace_id: workspace,
          selector: collectionSelector(q),
          snapshot: snapshot(),
          matching_count: "3000",
          start_index: String(start + 1),
          items: Array.from({ length: 25 }, (_, index) => row(start + index)),
          next_cursor: start ? null : "fictional-page-25",
          partial: false,
        },
      });
    }
    if (path.endsWith("/labels/preview")) {
      state.preview = {
        ...body,
        profile: "private-collection-preview-v3",
        rule_results: [],
        apply_count: "0",
        abstain_count: "0",
        tag_results: [],
        workspace_id: workspace,
        preview_id: "50000000-0000-4000-8000-000000000001",
        expires_at: "2099-01-01T00:00:00Z",
        selected_count: String(body.selection.length),
        review_rows: body.selection.map(
          (selection: { observation_id: string }) =>
            (() => {
              const {
                institution: _i,
                legal_owner: _o,
                rule_suggestion: _s,
                rule_application: _a,
                ...base
              } = row(Number(selection.observation_id.slice(-12)) - 1);
              return base;
            })(),
        ),
      };
      return route.fulfill({ json: state.preview });
    }
    const wrap = () => ({
      profile: "private-collection-command-v3",
      workspace_id: workspace,
      command: state.pending,
    });
    if (path.endsWith("/label-commands/pending"))
      return route.fulfill({ json: wrap() });
    if (path.endsWith("/label-commands")) {
      if (scenario === "private_prepare_conflict") {
        state.bookPending = {
          command_id: "80000000-0000-4000-8000-000000000001",
          operation: "setup",
          entity_id: null,
          book_id: null,
          state: "prepared",
          acknowledged: false,
          created_at: "2026-01-01T00:00:00Z",
          result: null,
          body: {
            entity_name: "Other context ledger winner",
            entity_kind: "legal",
            book_name: "Other context book",
            starts_on: "2026-01-01",
            ends_before: "2027-01-01",
            functional_currency_version_id:
              "90000000-0000-4000-8000-000000000001",
            accounts: [],
          },
        };
        return route.fulfill({
          status: 409,
          json: { code: "collection_conflict" },
        });
      }
      state.pending = {
        workspace_id: workspace,
        command_id: "60000000-0000-4000-8000-000000000001",
        prepare_key: body.prepare_key,
        state: "prepared",
        acknowledged: false,
        preview: state.preview!,
      };
      if (state.dropPrepare) {
        state.dropPrepare = false;
        return route.fulfill({
          status: 503,
          json: { code: "temporarily_unavailable" },
        });
      }
      return route.fulfill({ status: 201, json: wrap() });
    }
    if (state.pending && /\/(execute|cancel|acknowledge)$/.test(path)) {
      if (path.endsWith("/execute")) {
        state.executions++;
        state.pending.state = "committed";
        if (state.pending.preview.action.kind === "taxonomy") {
          state.topics = state.pending.preview.action.topics;
          state.taxonomy = String(Number(state.taxonomy) + 1);
        } else state.decision = String(Number(state.decision) + 1);
      } else if (path.endsWith("/cancel")) state.pending.state = "cancelled";
      else {
        state.pending.acknowledged = true;
        state.acknowledgements++;
      }
      const response = structuredClone(wrap());
      if (state.pending.acknowledged) state.pending = null;
      return route.fulfill({ json: response });
    }
    return route.fulfill({ status: 404, json: { code: "not_found" } });
  });
  return state;
}

test("real client opens private transactions without books, pages3000 and recovers exact label intent", async ({
  mount,
  page,
}) => {
  const state = await serve(page, "lost_prepare");
  await mount(<Harness />);
  await chooseFinanceTool(page, "Transactions");
  await expect(
    page.getByText("1–25 of 3000 matching transactions"),
  ).toBeVisible();
  await page.getByRole("checkbox").first().check();
  await page.getByRole("button", { name: "Next page" }).click();
  await expect(
    page.getByText("26–50 of 3000 matching transactions"),
  ).toBeVisible();
  await expect(page.getByText("0 selected on this page")).toBeVisible();
  await page.getByRole("checkbox").first().check();
  await page
    .getByLabel("Label selected transactions", { exact: true })
    .selectOption(`topic:${topic}`);
  await expect(
    page.getByRole("combobox", {
      name: "Label selected transactions",
      exact: true,
    }),
  ).toHaveValue(`topic:${topic}`);
  await page
    .getByRole("button", { name: "Preview labels for 1 selected" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Review 1 selected transactions" }),
  ).toBeFocused();
  await expect(financeNavItem(page, "Overview")).toBeDisabled();
  await page.getByRole("button", { name: "Save label review" }).click();
  await expect(
    page.getByRole("region", { name: "All transactions" }).getByRole("alert"),
  ).toContainText("outcome is unknown");
  await page.getByRole("button", { name: "Recover saved review" }).click();
  await expect(
    page.getByRole("heading", { name: "Saved label review" }),
  ).toBeFocused();
  expect(state.executions).toBe(0);
  await page
    .getByRole("button", { name: "Confirm labels for 1 selected" })
    .click();
  await expect(
    page.getByText("Labels applied. No journal was created."),
  ).toBeVisible();
  expect(state.acknowledgements).toBe(0);
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(financeNavItem(page, "Overview")).toBeEnabled();
  const preview = state.requests.find((request) =>
    request.path.endsWith("/labels/preview"),
  )!.body as Preview;
  expect(preview.selection).toEqual([
    {
      workspace_id: workspace,
      source_id: source,
      observation_id: "30000000-0000-4000-8000-000000000026",
      observation_revision: "1",
    },
  ]);
  expect(
    state.requests
      .filter((request) => request.path.includes("/collections/"))
      .every((request) => !request.key),
  ).toBe(true);
  expect(
    await page.evaluate(() => [localStorage.length, sessionStorage.length]),
  ).toEqual([0, 0]);
});

test("pending taxonomy recovery gates book tools, preserves old edition on remount and refreshes only after ack", async ({
  mount,
  page,
}) => {
  const state = await serve(page, "pending_taxonomy");
  const component = await mount(<Harness />);
  await expect(
    page.getByRole("heading", { name: "Saved topic review" }),
  ).toBeFocused();
  await expect(financeNavItem(page, "Overview")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(financeNavItem(page, "Transactions")).not.toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(financeNavItem(page, "Transactions")).toBeDisabled();
  await page.getByRole("button", { name: "Confirm topic changes" }).click();
  await expect(
    page.getByText("Topics updated. No journal was created."),
  ).toBeVisible();
  expect(state.taxonomy).toBe("4");
  expect(state.acknowledgements).toBe(0);
  await expect(financeNavItem(page, "Overview")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await component.unmount();
  await mount(<Harness />);
  await expect(
    page.getByRole("heading", { name: "Saved topic review" }),
  ).toBeFocused();
  await expect(
    page.getByText("Topics updated. No journal was created."),
  ).toBeVisible();
  expect(state.executions).toBe(1);
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(financeNavItem(page, "Transactions")).toBeEnabled();
  // A confirmed explicit acknowledgement returns to its private task without another navigation click.
  await expect(financeNavItem(page, "Transactions")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(
    page.getByText("1–25 of 3000 matching transactions"),
  ).toBeVisible();
  await chooseFinanceTool(page, "Transactions");
  await page.getByText("Manage topics", { exact: true }).click();
  await expect(page.getByLabel("Topic 1 name")).toHaveValue("Meals");
  expect(state.acknowledgements).toBe(1);
});

test("unknown private discovery keeps new work disabled until explicit successful retry", async ({
  mount,
  page,
}) => {
  const state = await serve(page, "discovery_failure");
  await mount(<Harness />);
  await expect(
    page
      .getByRole("region", { name: "Private transaction recovery" })
      .getByRole("alert"),
  ).toContainText("Private transaction recovery is unavailable");
  await expect(financeNavItem(page, "Transactions")).toBeDisabled();
  await expect(financeNavItem(page, "Overview")).toHaveAttribute(
    "aria-current",
    "page",
  );
  state.discoveryFails = false;
  await page
    .getByRole("button", { name: "Retry private transaction recovery" })
    .click();
  await expect(financeNavItem(page, "Transactions")).toBeEnabled();
  expect(state.executions).toBe(0);
});

test("a failed private acknowledgement keeps Overview until a confirmed same-identity retry", async ({
  mount,
  page,
}) => {
  const state = await serve(page, "pending_taxonomy");
  await mount(<Harness />);
  await page.getByRole("button", { name: "Confirm topic changes" }).click();
  await expect(
    page.getByText("Topics updated. No journal was created."),
  ).toBeVisible();
  const acknowledgements: string[] = [];
  await page.route(
    "**/v3/collections/*/label-commands/*/acknowledge",
    async (route) => {
      acknowledgements.push(new URL(route.request().url()).pathname);
      if (acknowledgements.length === 1)
        return route.fulfill({
          status: 503,
          json: { code: "temporarily_unavailable" },
        });
      return route.fallback();
    },
  );
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(
    page.getByRole("region", { name: "All transactions" }).getByRole("alert"),
  ).toContainText("outcome is unknown");
  await expect(financeNavItem(page, "Overview")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(financeNavItem(page, "Transactions")).not.toHaveAttribute(
    "aria-current",
    "page",
  );
  await page
    .getByRole("button", { name: "Retry same acknowledgement" })
    .click();
  await expect(financeNavItem(page, "Transactions")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(
    page.getByText("1–25 of 3000 matching transactions"),
  ).toBeVisible();
  expect(acknowledgements).toHaveLength(2);
  expect(acknowledgements[1]).toBe(acknowledgements[0]);
  expect(state.acknowledgements).toBe(1);
});

test("book recovery uncertainty prevents private mutation and navigation", async ({
  mount,
  page,
}) => {
  const state = await serve(page, "book_unknown");
  await mount(<Harness />);
  await expect(financeNavItem(page, "Transactions")).toBeDisabled();
  await expect(financeNavItem(page, "Overview")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(
    page.getByRole("region", { name: "Saved work" }).getByRole("alert"),
  ).toContainText("Data is unavailable");
  expect(
    state.requests.some((request) => request.path.endsWith("/review/query")),
  ).toBe(false);
  expect(state.executions).toBe(0);
});

test("empty taxonomy creates stable topic intent through explicit preview and acknowledgement at390px", async ({
  mount,
  page,
}, info) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const state = await serve(page);
  state.topics = [];
  state.taxonomy = "1";
  await mount(<Harness />);
  await chooseFinanceTool(page, "Transactions");
  await expect(
    page.getByRole("heading", { name: "All transactions", exact: true }),
  ).toBeFocused();
  await expect(
    page.getByText("1–25 of 3000 matching transactions"),
  ).toBeVisible();
  await page.getByText("Manage topics", { exact: true }).click();
  await page.getByRole("button", { name: "Add topic" }).click();
  await page.getByLabel("Topic 1 name").fill("Fictional household food");
  await page.screenshot({
    path: info.outputPath("fictional-private-topic-editor.png"),
  });
  await page.getByRole("button", { name: "Preview topic changes" }).click();
  await expect(
    page.getByRole("heading", { name: "Review topic changes" }),
  ).toBeFocused();
  const request = state.requests.find((request) =>
    request.path.endsWith("/labels/preview"),
  )!.body as Preview;
  expect(request.selection).toEqual([]);
  expect(request.action.kind).toBe("taxonomy");
  if (request.action.kind === "taxonomy")
    expect(request.action.topics).toEqual([
      {
        topic_id: expect.stringMatching(/^[0-9a-f-]{36}$/),
        name: "Fictional household food",
        parent_id: null,
        status: "active",
      },
    ]);
  expect(state.executions).toBe(0);
  await page.getByRole("button", { name: "Save topic review" }).click();
  await page.getByRole("button", { name: "Confirm topic changes" }).click();
  await expect(
    page.getByText("Topics updated. No journal was created."),
  ).toBeVisible();
  expect(state.acknowledgements).toBe(0);
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(
    page.getByText("1–25 of 3000 matching transactions"),
  ).toBeVisible();
  await page.getByRole("checkbox").first().check();
  await expect(
    page
      .getByLabel("Label selected transactions", { exact: true })
      .locator("option")
      .filter({ hasText: "Fictional household food" }),
  ).toHaveCount(1);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

test("read-only discovery still checks pending work and never enables private classification", async ({
  mount,
  page,
}) => {
  const state = await serve(page);
  state.capabilities = ["read"];
  await mount(<Harness />);
  await chooseFinanceTool(page, "Transactions");
  await expect(
    page.getByText("1–25 of 3000 matching transactions"),
  ).toBeVisible();
  await expect(
    page.getByText("Topics are read-only with your current access."),
  ).toBeVisible();
  await expect(page.getByRole("checkbox").first()).toBeDisabled();
  await expect(page.getByRole("checkbox")).toHaveCount(25);
  for (const checkbox of await page.getByRole("checkbox").all()) {
    await expect(checkbox).toBeDisabled();
  }
  await expect(
    page.getByLabel("Label selected transactions", { exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("group", {
      name: "Selected transaction actions",
      exact: true,
    }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", {
      name: /^(Preview labels|Preview rules|Apply rules)/,
    }),
  ).toHaveCount(0);
  await expect(page.getByText("Manage topics", { exact: true })).toHaveCount(0);
  expect(
    state.requests.some((request) =>
      request.path.endsWith("/label-commands/pending"),
    ),
  ).toBe(true);
  expect(state.executions).toBe(0);
});

test("revoked read clears retained rows and keeps other work gated until explicit fresh discovery", async ({
  mount,
  page,
}) => {
  const state = await serve(page);
  await mount(<Harness />);
  await chooseFinanceTool(page, "Transactions");
  await expect(
    page.getByText("Fictional purchase 1", { exact: true }),
  ).toBeVisible();
  state.denyQueries = true;
  await page.getByRole("button", { name: "Refresh transactions" }).click();
  await expect(
    page.getByText("Access to private transactions is no longer available."),
  ).toBeVisible();
  await expect(
    page.getByText("Fictional purchase 1", { exact: true }),
  ).toHaveCount(0);
  await expect(financeNavItem(page, "Overview")).toBeDisabled();
  state.denyQueries = false;
  await page
    .getByRole("button", { name: "Retry private transaction access" })
    .click();
  await expect(
    page.getByText("Fictional purchase 1", { exact: true }),
  ).toBeVisible();
  await expect(financeNavItem(page, "Overview")).toBeEnabled();
});

test("a definite private prepare conflict discovers the other context's ledger command", async ({
  mount,
  page,
}) => {
  const state = await serve(page, "private_prepare_conflict");
  await mount(<Harness />);
  await chooseFinanceTool(page, "Transactions");
  await page.getByRole("checkbox").first().check();
  await page
    .getByLabel("Label selected transactions", { exact: true })
    .selectOption(`topic:${topic}`);
  await page
    .getByRole("button", { name: "Preview labels for 1 selected" })
    .click();
  await page.getByRole("button", { name: "Save label review" }).click();
  const saved = page.getByRole("region", {
    name: "Saved work",
    exact: true,
  });
  await expect(
    saved.getByText("Other context ledger winner", { exact: true }),
  ).toBeVisible();
  await expect(
    saved.getByRole("button", { name: "Create book", exact: true }),
  ).toBeVisible();
  await expect(financeNavItem(page, "Overview")).toBeDisabled();
  expect(
    state.requests.filter((request) => request.path === "/v1/commands/pending")
      .length,
  ).toBeGreaterThanOrEqual(2);
  expect(
    state.requests.filter((request) =>
      request.path.endsWith("/label-commands/pending"),
    ).length,
  ).toBeGreaterThanOrEqual(2);
  expect(
    state.requests.some((request) =>
      /\/(execute|acknowledge)$/.test(request.path),
    ),
  ).toBe(false);
  expect(state.pending).toBeNull();
});

test("a definite ledger prepare conflict discovers the other context's private command", async ({
  mount,
  page,
}) => {
  const state = await serve(page, "book_prepare_conflict");
  await mount(<Harness />);
  await expect(financeNavItem(page, "Transactions")).toBeEnabled();
  await page
    .getByRole("button", { name: "Review Sandbox starter book", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Saved topic review" }),
  ).toBeFocused();
  await expect(
    page.getByText("Other context private winner", { exact: true }),
  ).toBeVisible();
  await expect(financeNavItem(page, "Overview")).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Retry saving review", exact: true }),
  ).toHaveCount(0);
  expect(
    state.requests.filter((request) => request.path === "/v1/commands/pending")
      .length,
  ).toBeGreaterThanOrEqual(2);
  expect(
    state.requests.filter((request) =>
      request.path.endsWith("/label-commands/pending"),
    ).length,
  ).toBeGreaterThanOrEqual(2);
  expect(
    state.requests.some((request) =>
      /\/(execute|acknowledge)$/.test(request.path),
    ),
  ).toBe(false);
  expect(state.bookPending).toBeNull();
});
