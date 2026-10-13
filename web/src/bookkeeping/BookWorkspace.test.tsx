import React from "react";
import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { BookWorkspace } from "./BookWorkspace";
import { APIError, type API } from "./api";
import type { Book, Entity, TrialBalance } from "./types";
afterEach(cleanup);
const book = {
  book_id: "b",
  name: "Operating",
  book_revision: "1",
  policies: [
    {
      functional_currency: "USD",
      effective_from: "2026-01-01",
      effective_to: null,
    },
  ],
  periods: [{ starts_on: "2026-01-01", ends_before: "2026-02-01" }],
  accounts: [{ account_id: "a", code: "1000", name: "Cash" }],
} as Book;
const entity = {
  entity_id: "e",
  name: "Fictional LLC",
  permissions: ["read", "post"],
  books: [book],
} as Entity;
const currencies = [
  {
    currency_version_id: "c",
    alphabetic_code: "USD",
    minor_unit: 2,
    valid_from: "2000-01-01",
    valid_to: null,
  },
] as never[];
// Fixed exact strings independently detect unsafe numerical coercion in the landing view.
const balance = (
  date: string,
  amount = "9007199254740993.01",
): TrialBalance => ({
  entity_id: "e",
  book_id: "b",
  through_date: date,
  snapshot_revision: "1",
  functional_currency: "USD",
  total_debits: amount,
  total_credits: amount,
  accounts: [
    {
      account_id: "a",
      account_code: "1000",
      debit_balance: amount,
      credit_balance: "0",
    },
  ],
});
// A deferred read lets a newer date complete before an older financial response.
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}
it("initial balances use explicit book date and ignore superseded responses", async () => {
  const first = deferred<TrialBalance>(),
    second = deferred<TrialBalance>();
  const api = vi
    .fn()
    .mockImplementationOnce(() => first.promise)
    .mockImplementationOnce(() => second.promise) as unknown as API;
  render(
    <BookWorkspace
      entity={entity}
      book={book}
      currencies={currencies}
      api={api}
      locked={false}
      denied={vi.fn()}
    >
      <p>Manual form</p>
    </BookWorkspace>,
  );
  expect(
    (screen.getByLabelText("Through date") as HTMLInputElement).value,
  ).toBe("2026-01-31");
  expect(screen.queryByRole("table")).toBeNull();
  fireEvent.change(screen.getByLabelText("Through date"), {
    target: { value: "2026-01-30" },
  });
  second.resolve(balance("2026-01-30"));
  await screen.findAllByText("9007199254740993.01");
  first.resolve(balance("2026-01-31", "999"));
  await waitFor(() => expect(screen.queryByText("999.00")).toBeNull());
  expect(
    screen
      .getByRole("button", { name: "Overview" })
      .getAttribute("aria-current"),
  ).toBe("page");
});
it("recovery lock disables scope and navigation without fetching balances", () => {
  const api = vi.fn() as unknown as API;
  render(
    <BookWorkspace
      entity={entity}
      book={book}
      currencies={currencies}
      api={api}
      locked={true}
      denied={vi.fn()}
    >
      <p>Manual form</p>
    </BookWorkspace>,
  );
  expect(api).not.toHaveBeenCalled();
  expect(
    (screen.getByRole("button", { name: "New journal" }) as HTMLButtonElement)
      .disabled,
  ).toBe(true);
  expect(
    (screen.getByLabelText("Through date") as HTMLInputElement).disabled,
  ).toBe(true);
  expect(screen.queryByRole("table")).toBeNull();
});
it("denied balance clears values and notifies scope owner rather than rendering zeros", async () => {
  const denied = vi.fn();
  const api = vi
    .fn()
    .mockRejectedValue(new APIError(403, false)) as unknown as API;
  render(
    <BookWorkspace
      entity={entity}
      book={book}
      currencies={currencies}
      api={api}
      locked={false}
      denied={denied}
    >
      <p>Manual form</p>
    </BookWorkspace>,
  );
  await waitFor(() => expect(denied).toHaveBeenCalledOnce());
  expect(screen.queryByRole("table")).toBeNull();
  expect(screen.getByRole("alert").textContent).toContain("unavailable");
});
