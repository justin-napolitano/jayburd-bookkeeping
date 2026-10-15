import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CommandPanel } from "./CommandPanel";
import type { useCommands } from "./commands";
import type { DurableCommand } from "./types";

afterEach(cleanup);

// A recovered import cannot be confirmed from IDs alone without its row preview.
const savedImport: DurableCommand = {
  command_id: "00000000-0000-4000-8000-000000000010",
  entity_id: "00000000-0000-4000-8000-000000000002",
  book_id: "00000000-0000-4000-8000-000000000003",
  operation: "import",
  state: "prepared",
  body: {
    batch_id: "00000000-0000-4000-8000-000000000011",
    preview_id: "00000000-0000-4000-8000-000000000012",
    preview_revision: "1",
    binding_revision: "1",
    source_revision: "0",
    profile: "jayburd-synthetic-statement-csv/1",
  },
  result: null,
  acknowledged: false,
  created_at: "2026-01-01T00:00:00Z",
};

// The panel tests decisions only; SQL tests own authoritative state transitions.
function recovery(command: DurableCommand): ReturnType<typeof useCommands> {
  return {
    command,
    accessDenied: false,
    busy: false,
    ready: true,
    error: "",
    registration: null,
    locked: true,
    prepare: vi.fn(),
    transition: vi.fn(),
    clearInaccessible: vi.fn(),
    retryRegistration: vi.fn(),
    reload: vi.fn(),
  };
}

describe("saved import recovery", () => {
  it("does not execute during render or permit confirmation without a preview", () => {
    const r = recovery(savedImport);
    render(<CommandPanel recovery={r} workspace={null} />);
    const confirm = screen.getByRole("button", {
      name: "Review import in originating client",
    });
    expect((confirm as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(confirm);
    expect(r.transition).not.toHaveBeenCalled();
    fireEvent.click(
      screen.getByRole("button", { name: "Cancel saved command" }),
    );
    expect(r.transition).toHaveBeenCalledExactlyOnceWith("cancel");
  });

  it("renders the saved exact counts and acknowledges without posting", () => {
    const r = recovery({
      ...savedImport,
      state: "committed",
      result: {
        entity_id: savedImport.entity_id!,
        book_id: savedImport.book_id!,
        batch_id: "00000000-0000-4000-8000-000000000011",
        created_count: "2",
        existing_count: "1",
        source_revision: "9007199254740993",
      },
    });
    render(<CommandPanel recovery={r} workspace={null} />);
    expect(
      screen.getByRole("heading", { name: "Import confirmed" }),
    ).toBeTruthy();
    expect(screen.getByText(/2 new review items/)).toBeTruthy();
    expect(screen.getByText(/No journals were posted/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Create book" })).toBeNull();
    expect(r.transition).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Acknowledge result" }));
    expect(r.transition).toHaveBeenCalledExactlyOnceWith("acknowledge");
  });
});

it("shows immutable source draft lines after remount and requires explicit posting", () => {
  const command: DurableCommand = {
    ...savedImport,
    operation: "source_post",
    body: {
      review_id: "00000000-0000-4000-8000-000000000020",
      draft_id: "00000000-0000-4000-8000-000000000021",
      review_revision: "2",
      source_revision: "2",
      binding_revision: "3",
      expected_book_revision: "0",
      posting_body: {
        expected_revision: "0",
        policy_id: "00000000-0000-4000-8000-000000000022",
        accounting_date: "2026-01-02",
        description: "Reviewed source purchase",
        lines: [
          {
            account_id: "00000000-0000-4000-8000-000000000023",
            side: "debit",
            transaction_currency: "USD",
            transaction_amount: "9007199254740993.01",
            functional_amount: "9007199254740993.01",
            valuation_basis: "identity",
          },
          {
            account_id: "00000000-0000-4000-8000-000000000024",
            side: "credit",
            transaction_currency: "USD",
            transaction_amount: "9007199254740993.01",
            functional_amount: "9007199254740993.01",
            valuation_basis: "identity",
          },
        ],
      },
    },
  };
  const r = recovery(command);
  const first = render(<CommandPanel recovery={r} workspace={null} />);
  expect(screen.getByText("Reviewed source purchase")).toBeTruthy();
  expect(screen.getAllByText(/9007199254740993.01/).length).toBeGreaterThan(0);
  expect(r.transition).not.toHaveBeenCalled();
  first.unmount();
  render(<CommandPanel recovery={r} workspace={null} />);
  expect(r.transition).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Post journal" }));
  expect(r.transition).toHaveBeenCalledExactlyOnceWith("execute");
});
