import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  FinanceNavigation,
  financeToolLabel,
  type FinanceTool,
} from "./FinanceNavigation";

afterEach(cleanup);
const defaults = () => ({
  tool: "All transactions" as FinanceTool,
  locked: false,
  loading: false,
  onNavigate: vi.fn(),
  onRefresh: vi.fn(),
  onSignOut: vi.fn(),
});

describe("responsive finance navigation", () => {
  it("gives private and book tools distinct names without changing their receiving identities", async () => {
    const props = defaults();
    render(<FinanceNavigation {...props} />);
    const daily = screen.getByRole("group", { name: "Daily" });
    const books = screen.getByRole("group", { name: "Bookkeeping" });
    expect(
      within(daily)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Overview", "Transactions", "Accounts"]);
    expect(
      within(books)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual([
      "Imported activity",
      "Ledger",
      "Position",
      "Reports",
      "New journal",
      "Chart of accounts",
    ]);
    await userEvent.click(
      within(daily).getByRole("button", { name: "Transactions" }),
    );
    await userEvent.click(
      within(daily).getByRole("button", { name: "Accounts" }),
    );
    await userEvent.click(
      within(books).getByRole("button", { name: "Imported activity" }),
    );
    await userEvent.click(
      within(books).getByRole("button", { name: "Chart of accounts" }),
    );
    expect(props.onNavigate.mock.calls.map((call) => call[0])).toEqual([
      "All transactions",
      "Connections",
      "Review transactions",
      "Accounts",
    ]);
  });
  it("shows one current destination and follows externally restored tool state", () => {
    const props = defaults();
    const view = render(<FinanceNavigation {...props} />);
    expect(
      screen
        .getByRole("button", { name: "Transactions" })
        .getAttribute("aria-current"),
    ).toBe("page");
    view.rerender(<FinanceNavigation {...props} tool="Review transactions" />);
    expect(
      screen
        .getByRole("button", { name: "Imported activity" })
        .getAttribute("aria-current"),
    ).toBe("page");
    expect(
      screen
        .getByRole("button", { name: "Transactions" })
        .hasAttribute("aria-current"),
    ).toBe(false);
    expect(
      document.querySelectorAll('button[aria-current="page"]'),
    ).toHaveLength(1);
  });
  it("exposes a controlled mobile menu that closes after keyboard selection", async () => {
    const props = defaults();
    render(<FinanceNavigation {...props} />);
    const menu = screen.getByRole("button", { name: "Menu" });
    expect(menu.getAttribute("aria-expanded")).toBe("false");
    expect(
      document.getElementById(menu.getAttribute("aria-controls")!),
    ).toBeTruthy();
    menu.focus();
    await userEvent.keyboard("{Enter}");
    expect(menu.getAttribute("aria-expanded")).toBe("true");
    const ledger = screen.getByRole("button", { name: "Ledger" });
    ledger.focus();
    await userEvent.keyboard("{Enter}");
    expect(props.onNavigate).toHaveBeenCalledWith("Account activity");
    expect(menu.getAttribute("aria-expanded")).toBe("false");
  });
  it("Escape closes the open menu and restores its trigger without navigating", async () => {
    const props = defaults();
    render(<FinanceNavigation {...props} />);
    await userEvent.click(screen.getByRole("button", { name: "Menu" }));
    screen.getByRole("button", { name: "Reports" }).focus();
    await userEvent.keyboard("{Escape}");
    const menu = screen.getByRole("button", { name: "Menu" });
    expect(document.activeElement).toBe(menu);
    expect(menu.getAttribute("aria-expanded")).toBe("false");
    expect(props.onNavigate).not.toHaveBeenCalled();
  });
  it("keeps every destination and refresh locked while recovery owns the gate", async () => {
    const props = { ...defaults(), locked: true };
    render(<FinanceNavigation {...props} />);
    for (const button of within(
      screen.getByRole("navigation", { name: "Finance tools" }),
    ).getAllByRole("button")) {
      expect((button as HTMLButtonElement).disabled).toBe(true);
      fireEvent.click(button);
    }
    const refresh = screen.getByRole("button", { name: "Refresh workspace" });
    expect((refresh as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(refresh);
    expect(props.onNavigate).not.toHaveBeenCalled();
    expect(props.onRefresh).not.toHaveBeenCalled();
    // The disclosure remains readable while destinations are unavailable.
    await userEvent.click(screen.getByRole("button", { name: "Menu" }));
    expect(
      screen
        .getByRole("button", { name: "Close menu" })
        .getAttribute("aria-expanded"),
    ).toBe("true");
  });
  it("keeps refresh loading separate from tool selection and retains explicit sign out", async () => {
    const props = { ...defaults(), loading: true };
    render(<FinanceNavigation {...props} />);
    expect(
      (
        screen.getByRole("button", {
          name: "Refresh workspace",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    await userEvent.click(screen.getByRole("button", { name: "Sign out" }));
    expect(props.onSignOut).toHaveBeenCalledOnce();
    expect(financeToolLabel("Connections")).toBe("Accounts");
    expect(financeToolLabel("Accounts")).toBe("Chart of accounts");
    expect(financeToolLabel("Overview")).toBe("Book overview");
  });
});
