import React from "react";
import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { AutomaticAccounting } from "./AutomaticAccounting";
import { APIError, type API } from "./api";
import type { Book, Entity } from "./types";
import {
  parseAccountingRules,
  exportAccountingRules,
} from "./accountingRulesYaml";
import {
  validAccountingRules,
  validAccountingStatus,
} from "./automaticAccountingClient";
afterEach(cleanup);
const id = (n: number) =>
  `76000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
const e = id(1),
  b = id(2),
  source = id(3),
  offset = id(4),
  policyID = id(5);
const book = {
  book_id: b,
  book_revision: "1",
  accounts: [
    {
      account_id: offset,
      name: "Supplies",
      category: "expense",
      normal_side: "debit",
    },
  ],
  policies: [
    {
      policy_id: policyID,
      functional_currency: "USD",
      effective_from: "2026-01-01",
      effective_to: null,
    },
  ],
} as Book;
const sources = [
  {
    source_account_id: source,
    display_name: "Checking",
    currency: "USD",
    state: "active",
    currency_version_id: id(12),
    ledger_account_id: id(13),
    binding_revision: "1",
    source_revision: "1",
  },
] as const;
const rule = {
  rule_id: id(6),
  name: "Supplies",
  source_account_id: source,
  direction: "outflow" as const,
  description: "Synthetic paper",
  offset_account_id: offset,
  treatment: "expense" as const,
};
const config = {
  book_policy_id: policyID,
  effective_from: "2026-01-01",
  effective_to: null,
  rules: [rule],
};
const policy = (enabled = true) => ({
  profile: "automatic-accounting-policy-v1",
  entity_id: e,
  book_id: b,
  revision: "1",
  enabled,
  policy: config,
  supported_profile: "source-identity-usd-v1",
});
const status = (enabled = true, runtime = true) => ({
  profile: "automatic-accounting-status-v1",
  entity_id: e,
  book_id: b,
  enabled,
  runtime_enabled: runtime,
  policy_revision: "1",
  pending_count: "0",
  attention_count: "0",
  blocked_reason: null,
  last_operation: null,
  holds: [],
  holds_truncated: false,
});

it("routes an unposted provider correction hold to its source without inventing a journal", async () => {
  const { api, props } = fixture();
  api.mockImplementation(async (path: string) =>
    path.endsWith("/sources")
      ? { profile: "source-list-v1", items: sources }
      : path.endsWith("/status")
        ? {
            ...status(),
            attention_count: "1",
            holds: [
              {
                review_id: id(7),
                source_account_id: source,
                reason: "source_correction_required",
                operation_id: id(8),
              },
            ],
            last_operation: {
              operation_id: id(8),
              state: "held",
              review_id: id(7),
              journal_id: null,
              reason: "source_correction_required",
            },
          }
        : policy(),
  );
  render(<AutomaticAccounting {...props} />);
  await screen.findByText("Source evidence needs correction review");
  expect(screen.queryByText(/Posted evidence changed/)).toBeNull();
  fireEvent.click(screen.getByText(/Accounting exceptions/));
  fireEvent.click(
    screen.getByRole("button", { name: "Review source transaction" }),
  );
  expect(props.onInspectReview).toHaveBeenCalledWith(id(7), source);
  expect(api.mock.calls.some(([path]) => path.includes("/journals/"))).toBe(
    false,
  );
});
function fixture(enabled = true, runtime = true) {
  const api = vi.fn<(path: string, body?: string) => Promise<unknown>>(
    async (path: string, body?: string) =>
      path.endsWith("/sources")
        ? { profile: "source-list-v1", items: sources }
        : path.endsWith("/status")
          ? status(enabled, runtime)
          : body
            ? { ...policy(enabled), revision: "2" }
            : policy(enabled),
  );
  return {
    api,
    props: {
      api: api as unknown as API,
      entity: { entity_id: e } as Entity,
      book,
      locked: false,
      denied: vi.fn(),
      onInspectReview: vi.fn(),
    },
  };
}
it.each([
  [true, false, "Configuration is saved. Automatic posting is not active"],
  [false, true, "Automation is available. This book’s policy is off"],
  [true, true, "Eligible source transactions post automatically"],
  [false, false, "The policy is off. Background posting is not active"],
])("distinguishes policy %s and runtime %s", async (enabled, runtime, text) => {
  const { props } = fixture(enabled as boolean, runtime as boolean);
  render(<AutomaticAccounting {...props} />);
  await screen.findByText(new RegExp(text as string));
});
it("does not substitute enabled or zero status when a read fails", async () => {
  const { api, props } = fixture();
  api.mockRejectedValue(new APIError(503, true, "read"));
  render(<AutomaticAccounting {...props} />);
  await screen.findByRole("alert");
  expect(screen.queryByText(/0 awaiting accounting/)).toBeNull();
  expect(
    screen.getByText(/Whether automatic posting is active is unknown/),
  ).toBeTruthy();
});
it("hands an exception to the currently scoped source review", async () => {
  const { api, props } = fixture();
  api.mockImplementation(async (path: string) =>
    path.endsWith("/sources")
      ? { profile: "source-list-v1", items: sources }
      : path.endsWith("/status")
        ? {
            ...status(),
            attention_count: "1",
            holds: [
              {
                review_id: id(7),
                source_account_id: source,
                reason: "unmapped_rule",
                operation_id: id(8),
              },
            ],
          }
        : policy(),
  );
  render(<AutomaticAccounting {...props} />);
  fireEvent.click(await screen.findByText(/Accounting exceptions/));
  fireEvent.click(
    screen.getByRole("button", { name: "Review source transaction" }),
  );
  expect(props.onInspectReview).toHaveBeenCalledWith(id(7), source);
});
it("rejects contradictory operation states, count truncation and extra fields", () => {
  expect(validAccountingStatus(status(), e, b)).toBe(true);
  expect(
    validAccountingStatus({ ...status(), holds_truncated: true }, e, b),
  ).toBe(false);
  expect(validAccountingStatus({ ...status(), secret: "bad" }, e, b)).toBe(
    false,
  );
  for (const op of [
    { state: "prepared", journal_id: id(8), reason: null },
    { state: "held", journal_id: null, reason: null },
    { state: "committed", journal_id: null, reason: null },
  ])
    expect(
      validAccountingStatus(
        {
          ...status(),
          last_operation: { operation_id: id(9), review_id: id(7), ...op },
        },
        e,
        b,
      ),
    ).toBe(false);
});
it("round trips stable mapping IDs and resolves exact names", () => {
  const yaml = exportAccountingRules([rule]);
  expect(parseAccountingRules(yaml, book, [...sources], [rule])).toEqual([
    rule,
  ]);
  expect(
    parseAccountingRules(
      yaml
        .replace(`id: ${source}`, "name: Checking")
        .replace(`id: ${offset}`, "name: Supplies"),
      book,
      [...sources],
      [rule],
    ),
  ).toEqual([rule]);
});
it.each([
  "version: 1\nrules: &rules []",
  "version: 1\nrules: *missing",
  "version: 1\nrules: []\nother: true",
  "version: 1\nversion: 1\nrules: []",
  "version: 1\nrules: []\n---\nversion: 1\nrules: []",
])("rejects unsafe or unclosed YAML %s", (yaml) =>
  expect(() => parseAccountingRules(yaml, book, [...sources], [])).toThrow(),
);
it("bounds bytes, unknown references, duplicate rules and ambiguous names", () => {
  expect(() =>
    parseAccountingRules("x".repeat(131073), book, [...sources], []),
  ).toThrow();
  const yaml = exportAccountingRules([rule]);
  expect(() => parseAccountingRules(yaml, book, [], [])).toThrow();
  expect(() =>
    parseAccountingRules(
      exportAccountingRules([rule, { ...rule, rule_id: id(10) }]),
      book,
      [...sources],
      [],
    ),
  ).toThrow();
  expect(() =>
    parseAccountingRules(
      yaml.replace(`id: ${source}`, "name: Checking"),
      book,
      [...sources, { ...sources[0], source_account_id: id(11) }],
      [],
    ),
  ).toThrow();
});

it("keeps the original request after an unknown save and retries only on demand", async () => {
  const { api, props } = fixture();
  const writes: string[] = [];
  api.mockImplementation(async (path: string, body?: string) => {
    if (body) {
      writes.push(body);
      if (writes.length === 1) throw new APIError(0, true, "read");
      return { ...policy(), revision: "2" };
    }
    return path.endsWith("/sources")
      ? { profile: "source-list-v1", items: sources }
      : path.endsWith("/status")
        ? status()
        : policy();
  });
  render(<AutomaticAccounting {...props} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Configure accounting" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Review accounting configuration" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Save accounting configuration" }),
  );
  await screen.findByRole("button", {
    name: "Retry same accounting configuration",
  });
  expect(writes).toHaveLength(1);
  expect(
    (
      screen
        .getByLabelText("Accounting mapping YAML")
        .closest("fieldset") as HTMLFieldSetElement
    ).disabled,
  ).toBe(true);
  fireEvent.click(
    screen.getByRole("button", { name: "Retry same accounting configuration" }),
  );
  await waitFor(() => expect(writes).toHaveLength(2));
  expect(writes[1]).toBe(writes[0]);
});
it("invalidates pending file reads when configuration is closed or reviewed", async () => {
  const { props } = fixture();
  render(<AutomaticAccounting {...props} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Configure accounting" }),
  );
  let resolve!: (value: string) => void;
  const text = new Promise<string>((r) => {
    resolve = r;
  });
  const file = new File(["synthetic"], "rules.yaml");
  Object.defineProperty(file, "text", { value: () => text });
  fireEvent.change(screen.getByLabelText("Accounting YAML file"), {
    target: { files: [file] },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Review accounting configuration" }),
  );
  resolve("version: 1\nrules: []\n");
  await Promise.resolve();
  expect(
    (screen.getByLabelText("Accounting mapping YAML") as HTMLTextAreaElement)
      .value,
  ).toContain("Synthetic paper");
  expect(
    (
      screen.getByRole("button", {
        name: "Configure accounting",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
});
it("reloads actual saved policy after leaving an uncertain write, without an automatic POST", async () => {
  const { api, props } = fixture();
  let writes = 0;
  api.mockImplementation(async (path: string, body?: string) => {
    if (body) {
      writes++;
      throw new APIError(0, true, "read");
    }
    return path.endsWith("/sources")
      ? { profile: "source-list-v1", items: sources }
      : path.endsWith("/status")
        ? status()
        : policy();
  });
  const view = render(<AutomaticAccounting {...props} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Configure accounting" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Review accounting configuration" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Save accounting configuration" }),
  );
  await screen.findByRole("button", {
    name: "Retry same accounting configuration",
  });
  view.rerender(
    <AutomaticAccounting {...props} book={{ ...book, book_revision: "2" }} />,
  );
  await screen.findByText(/Eligible source transactions post automatically/);
  expect(writes).toBe(1);
  expect(
    screen.queryByRole("button", {
      name: "Retry same accounting configuration",
    }),
  ).toBeNull();
});
it("rejects policy/status snapshots from different policy editions", async () => {
  const { api, props } = fixture();
  api.mockImplementation(async (path: string) =>
    path.endsWith("/sources")
      ? { profile: "source-list-v1", items: sources }
      : path.endsWith("/status")
        ? { ...status(), policy_revision: "2" }
        : policy(),
  );
  render(<AutomaticAccounting {...props} />);
  await screen.findByRole("alert");
  expect(screen.queryByText(/0 awaiting accounting/)).toBeNull();
});

it("rejects coercible enum arrays and NUL descriptions before review", () => {
  for (const patch of [
    { direction: ["outflow"] },
    { treatment: ["expense"] },
    { direction: {} },
    { description: "synthetic\u0000paper" },
  ])
    expect(validAccountingRules([{ ...rule, ...patch }])).toBe(false);
  expect(
    validAccountingStatus(
      { ...status(), blocked_reason: ["saved_work"] },
      e,
      b,
    ),
  ).toBe(false);
  expect(
    validAccountingStatus(
      {
        ...status(),
        last_operation: {
          operation_id: id(8),
          review_id: id(7),
          state: ["prepared"],
          journal_id: null,
          reason: null,
        },
      },
      e,
      b,
    ),
  ).toBe(false);
});
