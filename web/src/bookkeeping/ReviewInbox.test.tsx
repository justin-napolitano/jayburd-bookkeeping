import React from "react";
import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import {
  ReviewInbox,
  type ReviewInboxProps,
  type ReviewDraft,
} from "./ReviewInbox";
import { APIError, type API } from "./api";
import type { Book, Currency, Entity } from "./types";
afterEach(cleanup);
const base = "/v1/entities/e/books/b";
const row = {
  review_id: "r",
  state: "needs_review",
  review_revision: "1",
  booked_date: "2026-01-04",
  currency: "USD",
  currency_version_id: "usd",
  direction: "outflow",
  amount: "9007199254740993.01",
  description: "Synthetic supplies",
  draft_id: null,
  journal_id: null,
};
const detail = {
  ...row,
  source_account_id: "s",
  source_revision: "2",
  binding_revision: "1",
  book_revision: "4",
  ledger_account_id: "cash",
  blockers: [],
  draft: null,
};
const categories = {
  review_id: "r",
  source_account_id: "s",
  book_revision: "4",
  policy_id: "policy",
  accounting_date: "2026-01-04",
  source_currency: "USD",
  functional_currency: "USD",
  fx_required: false,
  accounts: [
    {
      account_id: "expense",
      code: "5000",
      name: "Supplies",
      category: "expense",
      normal_side: "debit",
      monetary: false,
    },
  ],
};
const draft = {
  draft_id: "d",
  review_id: "r",
  review_revision: "2",
  source_revision: "3",
  binding_revision: "1",
  book_revision: "4",
  posting_body: {
    expected_revision: "4",
    policy_id: "policy",
    accounting_date: "2026-01-04",
    description: "Synthetic supplies",
    lines: [
      {
        account_id: "cash",
        side: "credit",
        transaction_currency: "USD",
        transaction_amount: row.amount,
        functional_amount: row.amount,
        valuation_basis: "identity",
      },
      {
        account_id: "expense",
        side: "debit",
        transaction_currency: "USD",
        transaction_amount: row.amount,
        functional_amount: row.amount,
        valuation_basis: "identity",
      },
    ],
  },
} as ReviewDraft;
// Independent fixtures deliberately exceed JS safe integer precision.
function fixture() {
  const api = vi.fn(async (path: string): Promise<unknown> => {
    if (path.endsWith("/sources"))
      return {
        items: [
          {
            source_account_id: "s",
            display_name: "Checking",
            currency: "USD",
            state: "active",
          },
        ],
      };
    if (path.includes("/review-items?"))
      return {
        source_account_id: "s",
        source_revision: "2",
        items: [row],
        next_cursor: "opaque/page+2",
      };
    if (path.endsWith("/categories")) return categories;
    if (path.endsWith("/drafts")) return draft;
    return detail;
  });
  const props: ReviewInboxProps = {
    api: api as unknown as API,
    entity: { entity_id: "e" } as Entity,
    book: {
      book_id: "b",
      book_revision: "4",
      accounts: [
        { account_id: "cash", name: "Checking" },
        { account_id: "expense", name: "Supplies" },
      ],
    } as Book,
    currencies: [{ currency_version_id: "usd", minor_unit: 2 }] as Currency[],
    locked: false,
    prepare: vi.fn(),
    denied: vi.fn(),
  };
  return { api, props };
}
async function choose() {
  await screen.findByRole("option", { name: /Checking/ });
  fireEvent.change(screen.getByLabelText("Source account"), {
    target: { value: "s" },
  });
  fireEvent.click(
    await screen.findByRole("button", { name: "Review Synthetic supplies" }),
  );
  await screen.findByLabelText("Category account");
}
it("keeps source review exact and creates a draft before handing off explicit posting", async () => {
  const { api, props } = fixture();
  render(<ReviewInbox {...props} />);
  await choose();
  expect(screen.getAllByText(/9007199254740993.01/).length).toBeGreaterThan(0);
  expect(props.prepare).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText("Category account"), {
    target: { value: "expense" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Create reviewed draft" }),
  );
  const button = await screen.findByRole("button", {
    name: "Review posting command",
  });
  const call = api.mock.calls.find((c) =>
    c[0].endsWith("/drafts"),
  ) as unknown as [string, string, string];
  expect(JSON.parse(call[1])).toEqual({
    expected_review_revision: "1",
    source_revision: "2",
    binding_revision: "1",
    expected_book_revision: "4",
    category_account_id: "expense",
    policy_id: "policy",
    accounting_date: "2026-01-04",
    description: "Synthetic supplies",
  });
  expect(props.prepare).not.toHaveBeenCalled();
  fireEvent.click(button);
  expect(props.prepare).toHaveBeenCalledWith({
    operation: "source_post",
    entity_id: "e",
    book_id: "b",
    body: {
      review_id: "r",
      draft_id: "d",
      review_revision: "2",
      source_revision: "3",
      binding_revision: "1",
      expected_book_revision: "4",
      posting_body: draft.posting_body,
    },
  });
  expect(api.mock.calls.some((c) => c[0].includes("/execute"))).toBe(false);
});
it("retries unknown draft outcome with identical body/key and freezes edited inputs", async () => {
  const { api, props } = fixture();
  render(<ReviewInbox {...props} />);
  await choose();
  fireEvent.change(screen.getByLabelText("Category account"), {
    target: { value: "expense" },
  });
  api.mockRejectedValueOnce(new APIError(503, true));
  fireEvent.click(
    screen.getByRole("button", { name: "Create reviewed draft" }),
  );
  fireEvent.click(
    await screen.findByRole("button", { name: "Retry same draft request" }),
  );
  await screen.findByRole("button", { name: "Review posting command" });
  const calls = api.mock.calls.filter((c) => c[0].endsWith("/drafts"));
  expect(calls).toHaveLength(2);
  expect(calls[1]).toEqual(calls[0]);
});
it("uses the opaque page cursor and removes stale rows on snapshot failure", async () => {
  const { api, props } = fixture();
  render(<ReviewInbox {...props} />);
  await screen.findByRole("option", { name: /Checking/ });
  fireEvent.change(screen.getByLabelText("Source account"), {
    target: { value: "s" },
  });
  const next = await screen.findByRole("button", { name: "Next transactions" });
  api.mockRejectedValueOnce(new APIError(409, false));
  fireEvent.click(next);
  await screen.findByRole("alert");
  expect(screen.queryByRole("table")).toBeNull();
  expect(api.mock.calls.at(-1)?.[0]).toBe(
    `${base}/review-items?source_account_id=s&limit=50&state=needs_review&cursor=opaque%2Fpage%2B2`,
  );
});
it("clears denied data and never displays raw infrastructure errors", async () => {
  const { api, props } = fixture();
  api.mockRejectedValueOnce(new APIError(403, false));
  render(<ReviewInbox {...props} />);
  await screen.findByRole("alert");
  expect(props.denied).toHaveBeenCalled();
  expect(screen.queryByRole("table")).toBeNull();
});
it("hidden and locked inboxes do not request data", () => {
  const { api, props } = fixture();
  const view = render(<ReviewInbox {...props} active={false} />);
  expect(api).not.toHaveBeenCalled();
  view.rerender(<ReviewInbox {...props} locked />);
  expect(api).not.toHaveBeenCalled();
});
it("a late response from a previous book cannot repopulate the new book", async () => {
  const { api, props } = fixture();
  let resolve!: (value: unknown) => void;
  api.mockImplementationOnce(
    () =>
      new Promise((r) => {
        resolve = r;
      }) as never,
  );
  const view = render(<ReviewInbox {...props} />);
  view.rerender(
    <ReviewInbox {...props} book={{ ...props.book, book_id: "new" }} />,
  );
  await screen.findByRole("option", { name: /Checking/ });
  resolve({
    items: [{ source_account_id: "old", display_name: "Stale account" }],
  });
  await waitFor(() => expect(screen.queryByText(/Stale account/)).toBeNull());
});
it("blocks unsupported FX without hiding original source units", async () => {
  const { api, props } = fixture();
  const original = api.getMockImplementation()!;
  api.mockImplementation(async (path) =>
    path.endsWith("/categories")
      ? { ...categories, fx_required: true }
      : original(path),
  );
  render(<ReviewInbox {...props} />);
  await screen.findByRole("option", { name: /Checking/ });
  fireEvent.change(screen.getByLabelText("Source account"), {
    target: { value: "s" },
  });
  fireEvent.click(
    await screen.findByRole("button", { name: "Review Synthetic supplies" }),
  );
  await screen.findByText(/explicit FX valuation/);
  expect(
    screen.queryByRole("button", { name: "Create reviewed draft" }),
  ).toBeNull();
  expect(screen.getAllByText(/9007199254740993.01/).length).toBeGreaterThan(0);
});
it("retries source discovery without inventing an empty successful source list", async () => {
  const { api, props } = fixture();
  api.mockRejectedValueOnce(new Error("private backend diagnostic"));
  render(<ReviewInbox {...props} />);
  const retry = await screen.findByRole("button", {
    name: "Retry source accounts",
  });
  expect(screen.queryByText(/private backend/)).toBeNull();
  expect(screen.queryByText(/No source accounts/)).toBeNull();
  fireEvent.click(retry);
  await screen.findByRole("option", { name: /Checking/ });
  expect(api).toHaveBeenCalledTimes(2);
});
it("posted historical draft has no action that can prepare another posting", async () => {
  const { api, props } = fixture();
  const original = api.getMockImplementation()!;
  api.mockImplementation(async (path) =>
    path.endsWith("/review-items/r")
      ? { ...detail, state: "posted", journal_id: "journal", draft }
      : original(path),
  );
  render(<ReviewInbox {...props} />);
  await screen.findByRole("option", { name: /Checking/ });
  fireEvent.change(screen.getByLabelText("Source account"), {
    target: { value: "s" },
  });
  fireEvent.click(
    await screen.findByRole("button", { name: "Review Synthetic supplies" }),
  );
  await screen.findByText("Posted journal draft");
  expect(
    screen.queryByRole("button", { name: "Review posting command" }),
  ).toBeNull();
  expect(
    screen.queryByRole("button", { name: "Create reviewed draft" }),
  ).toBeNull();
  expect(props.prepare).not.toHaveBeenCalled();
});
it("stale draft requires a new category review before posting becomes available", async () => {
  const { api, props } = fixture();
  const original = api.getMockImplementation()!;
  api.mockImplementation(async (path) =>
    path.endsWith("/review-items/r")
      ? { ...detail, state: "draft_ready", blockers: ["draft_stale"], draft }
      : original(path),
  );
  render(<ReviewInbox {...props} />);
  await choose();
  await screen.findByText("Previous draft requires review");
  expect(
    screen.queryByRole("button", { name: "Review posting command" }),
  ).toBeNull();
  fireEvent.change(screen.getByLabelText("Category account"), {
    target: { value: "expense" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Create reviewed draft" }),
  );
  await screen.findByRole("button", { name: "Review posting command" });
  expect(props.prepare).not.toHaveBeenCalled();
});
it("reads the posted journal itself with current scope and no mutation", async () => {
  const { api, props } = fixture();
  const original = api.getMockImplementation()!;
  api.mockImplementation(async (path) =>
    path.endsWith("/review-items/r")
      ? { ...detail, state: "posted", journal_id: "journal", draft }
      : path.endsWith("/journals/journal")
        ? {
            entity_id: "e",
            book_id: "b",
            journal_id: "journal",
            accounting_date: row.booked_date,
            description: "Actual immutable journal",
            book_revision: "5",
            functional_currency: "USD",
            lines: draft.posting_body.lines,
          }
        : original(path),
  );
  render(<ReviewInbox {...props} />);
  await screen.findByRole("option", { name: /Checking/ });
  fireEvent.change(screen.getByLabelText("Source account"), {
    target: { value: "s" },
  });
  fireEvent.click(
    await screen.findByRole("button", { name: "Review Synthetic supplies" }),
  );
  fireEvent.click(
    await screen.findByRole("button", { name: "View posted journal" }),
  );
  await screen.findByRole("table", { name: "Immutable posted journal lines" });
  expect(screen.getByText(/Actual immutable journal/)).toBeTruthy();
  expect(api.mock.calls.filter((c) => c[0].includes("/journals/"))).toEqual([
    [`${base}/journals/journal`],
  ]);
  expect(api.mock.calls.every((c) => c.length === 1)).toBe(true);
});
