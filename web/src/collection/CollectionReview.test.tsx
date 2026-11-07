import {
  act,
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  collectionCatalog,
  collectionFixture,
  collectionPage,
  collectionQuery,
  collectionRow,
  collectionSources,
  collectionWorkspace,
  heldCollectionRow,
  nullNumericCollectionRow,
  removedHeldCollectionRow,
} from "../../tests/collection-fixture";
import { CollectionReview } from "./CollectionReview";
import {
  CollectionFailure,
  selectionKey,
  validPage,
  type Page,
  type Query,
} from "./types";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
// The harness injects fictional callbacks; it provides no real endpoint or runtime authorization.
async function start(options: Parameters<typeof collectionFixture>[0] = {}) {
  const fixture = collectionFixture(options),
    denied = vi.fn(),
    user = userEvent.setup();
  const view = render(
    <CollectionReview
      workspace={collectionWorkspace}
      sources={collectionSources}
      catalog={collectionCatalog}
      port={fixture.port}
      onDenied={denied}
    />,
  );
  await screen.findByText(/matching transactions/);
  return { ...fixture, denied, user, ...view };
}
async function previewOne(
  h: Awaited<ReturnType<typeof start>>,
  choice = "topic:dining",
) {
  await h.user.click(screen.getAllByRole("checkbox")[0]);
  const control = screen.getByLabelText("Label selected transactions", {
    exact: true,
  });
  await h.user.selectOptions(control, choice);
  expect(
    screen.getByRole("combobox", {
      name: "Label selected transactions",
      exact: true,
    }),
  ).toBe(control);
  await h.user.click(
    screen.getByRole("button", { name: "Preview labels for 1 selected" }),
  );
  await screen.findByRole("heading", {
    name: "Review 1 selected transactions",
  });
}

describe("private collection review", () => {
  it("retains an uncertain prepare identity through a book recovery lock without remount or implicit rediscovery", async () => {
    const h = await start({ scenario: "lost_prepare" });
    await previewOne(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save label review" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    const saved = h.calls.prepares[0];
    const pendingReads = h.calls.pending;
    const props = {
      workspace: collectionWorkspace,
      sources: collectionSources,
      catalog: collectionCatalog,
      port: h.port,
      onDenied: h.denied,
    };
    h.rerender(<CollectionReview {...props} locked />);
    expect(
      screen.queryByRole("button", { name: "Retry saving same review" }),
    ).toBeNull();
    h.rerender(<CollectionReview {...props} locked={false} />);
    await screen.findByRole("button", { name: "Retry saving same review" });
    expect(h.calls.pending).toBe(pendingReads);
    await h.user.click(
      screen.getByRole("button", { name: "Retry saving same review" }),
    );
    await screen.findByRole("heading", { name: "Saved label review" });
    expect(h.calls.prepares).toEqual([saved, saved]);
    expect(h.calls.execute).toBe(0);
    expect(h.calls.acknowledge).toBe(0);
  });
  it("supports100 astral search characters and rejects excess codepoints without truncating intent", async () => {
    const h = await start();
    const input = screen.getByRole("searchbox", {
      name: "Search transactions",
    });
    const search = "🍎".repeat(100);
    await h.user.click(input);
    await h.user.paste(search);
    await h.user.click(screen.getByRole("button", { name: "Apply filters" }));
    await waitFor(() => expect(h.calls.queries.at(-1)?.search).toBe(search));
    await screen.findByText(
      "No transactions match these filters. Change filters or refresh imported activity.",
    );
    await h.user.clear(input);
    await h.user.click(input);
    await h.user.paste("a".repeat(101));
    const queries = h.calls.queries.length;
    await h.user.click(screen.getByRole("button", { name: "Apply filters" }));
    expect(screen.getByRole("alert").textContent).toContain(
      "up to 100 characters",
    );
    expect(h.calls.queries).toHaveLength(queries);
  });
  it("shows removed held money as removed/unavailable without a fabricated zero", async () => {
    await start({ rows: [removedHeldCollectionRow()] });
    expect(screen.getByText("Source: removed")).toBeTruthy();
    expect(screen.getByText("Money unavailable")).toBeTruthy();
    expect(screen.queryByText(/0\.00 USD/)).toBeNull();
  });
  it("focuses the preview only after its DOM commits even if animation frames run immediately", async () => {
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
    const h = await start();
    expect(document.activeElement).not.toBe(
      screen.getByRole("heading", { name: "All transactions" }),
    );
    await previewOne(h);
    expect(document.activeElement).toBe(
      screen.getByRole("heading", { name: "Review 1 selected transactions" }),
    );
  });
  it("renders a missing stored numeric currency code without inventing one or holding valid money", async () => {
    await start({ rows: [nullNumericCollectionRow()] });
    expect(screen.getByText("12.50 USD")).toBeTruthy();
    expect(screen.getByText(/numeric code not supplied/)).toBeTruthy();
    expect(screen.queryByText("Money unavailable")).toBeNull();
    expect(screen.queryByText(/840/)).toBeNull();
  });
  it("renders exact large money, unavailable held values and distinct source/book/journal/provider states", async () => {
    const row = collectionRow();
    if (row.money.state === "available")
      row.money.amount = "9007199254740993.01";
    await start({ rows: [row, heldCollectionRow()] });
    expect(screen.getByText("9007199254740993.01 USD")).toBeTruthy();
    const held = screen
      .getByText("Fictional unsupported_currency hold")
      .closest("li")!;
    expect(within(held).getByText("Money unavailable")).toBeTruthy();
    expect(within(held).queryByText(/0(?:\.00)? USD/)).toBeNull();
    expect(screen.getAllByText("Book assignment unavailable")).toHaveLength(2);
    expect(screen.queryByText("Unassigned to a book")).toBeNull();
    expect(screen.getAllByText("Journal: unavailable")).toHaveLength(2);
    expect(
      screen.getByText(/Provider suggestion: Food and drink/),
    ).toBeTruthy();
    expect(screen.getAllByText("Owner topic: Not classified")).toHaveLength(2);
  });
  it("uses server pages and clears page selections without aggregating all 3000 rows", async () => {
    const h = await start();
    expect(screen.getByText("1–25 of 3000 matching transactions")).toBeTruthy();
    expect(screen.getAllByRole("checkbox")).toHaveLength(25);
    await h.user.click(screen.getAllByRole("checkbox")[0]);
    await h.user.click(screen.getByRole("button", { name: "Next page" }));
    await screen.findByText("26–50 of 3000 matching transactions");
    expect(screen.getByText("0 selected on this page")).toBeTruthy();
    expect(h.calls.queries).toHaveLength(2);
    expect(h.calls.queries[1].cursor).toBe("page:25");
    expect(h.calls.queries[1].snapshot?.source_snapshot).toBe("a".repeat(64));
    expect(screen.queryByText("Fictional purchase 1")).toBeNull();
    expect(screen.queryByText(/total spending/i)).toBeNull();
  });
  it("walks a 3000-row server pagination fixture with pinned snapshot and no duplicate composite rows", async () => {
    const h = collectionFixture(),
      keys = new Set<string>();
    let q = collectionQuery(),
      pages = 0;
    for (;;) {
      const page = await h.port.query(q);
      expect(validPage(page, q, collectionWorkspace, collectionCatalog)).toBe(
        true,
      );
      page.items.forEach((row) => {
        expect(keys.has(selectionKey(row))).toBe(false);
        keys.add(selectionKey(row));
      });
      pages++;
      if (page.next_cursor === null) break;
      q = { ...q, snapshot: page.snapshot, cursor: page.next_cursor };
    }
    expect(pages).toBe(120);
    expect(keys.size).toBe(3000);
    expect(h.calls.queries.every((q) => q.limit === 25)).toBe(true);
  });
  it("applies explicit filters only on submit and destroys old page selection", async () => {
    const h = await start();
    await h.user.click(screen.getAllByRole("checkbox")[0]);
    await h.user.selectOptions(
      screen.getByLabelText("Account"),
      "fictional-card",
    );
    expect(h.calls.queries).toHaveLength(1);
    await h.user.click(screen.getByRole("button", { name: "Apply filters" }));
    await screen.findByText("1–25 of 1500 matching transactions");
    expect(h.calls.queries[1].source_account_id).toBe("fictional-card");
    expect(h.calls.queries[1].cursor).toBeNull();
    expect(h.calls.queries[1].snapshot).toBeNull();
    expect(screen.getByText("0 selected on this page")).toBeTruthy();
  });
  it("rejects echo mismatches rather than showing foreign or differently filtered rows", async () => {
    const fixture = collectionFixture({ rows: [collectionRow()] });
    fixture.port.query = async (q) => ({
      ...collectionPage(q),
      workspace_id: "foreign-workspace",
    });
    render(
      <CollectionReview
        workspace={collectionWorkspace}
        sources={collectionSources}
        catalog={collectionCatalog}
        port={fixture.port}
        onDenied={() => {}}
      />,
    );
    await screen.findByRole("alert");
    expect(screen.queryByText("Fictional purchase 1")).toBeNull();
  });
  it("discards a late old-scope response after workspace changes", async () => {
    const a = collectionFixture(),
      b = collectionFixture({
        rows: [
          collectionRow(1, {
            workspace_id: "fictional-workspace-b",
            description: "Fictional new scope",
          }),
        ],
      });
    let release!: (page: Page) => void;
    let oldQuery!: Query;
    a.port.query = (q) => {
      oldQuery = q;
      return new Promise((resolve) => {
        release = resolve;
      });
    };
    const denied = vi.fn(),
      view = render(
        <CollectionReview
          workspace={collectionWorkspace}
          sources={collectionSources}
          catalog={collectionCatalog}
          port={a.port}
          onDenied={denied}
        />,
      );
    await waitFor(() => expect(release).toBeTypeOf("function"));
    view.rerender(
      <CollectionReview
        workspace={{
          ...collectionWorkspace,
          workspace_id: "fictional-workspace-b",
        }}
        sources={collectionSources}
        catalog={collectionCatalog}
        port={b.port}
        onDenied={denied}
      />,
    );
    await screen.findByText("Fictional new scope");
    await act(async () => {
      release(collectionPage(oldQuery));
    });
    expect(screen.queryByText("Fictional purchase 1")).toBeNull();
    expect(screen.getByText("Fictional new scope")).toBeTruthy();
  });
  it("destroys retained rows after current access is revoked", async () => {
    const h = await start({ rows: [collectionRow()] });
    h.port.query = async () => {
      throw new CollectionFailure("denied");
    };
    await h.user.click(
      screen.getByRole("button", { name: "Refresh transactions" }),
    );
    await screen.findByText(
      "Access to these transactions is no longer available.",
    );
    expect(screen.queryByText("Fictional purchase 1")).toBeNull();
    expect(h.denied).toHaveBeenCalledOnce();
    expect(screen.queryByText(/0 of 0/)).toBeNull();
  });
  it("distinguishes stale and empty reads and renders partial history honestly", async () => {
    const h = await start({ rows: [heldCollectionRow("invalid_precision")] });
    expect(screen.getByText(/Some source history is unavailable/)).toBeTruthy();
    h.port.query = async () => {
      throw new CollectionFailure("stale");
    };
    await h.user.click(
      screen.getByRole("button", { name: "Refresh transactions" }),
    );
    await screen.findByText(/Transactions or categories changed/);
    expect(screen.queryByText(/No transactions match/)).toBeNull();
    h.port.query = async (q) => collectionPage(q, []);
    await h.user.click(
      screen.getByRole("button", { name: "Refresh transactions" }),
    );
    await screen.findByText(/No transactions match these filters/);
  });
  it("previews only the exact selected source revisions and requires explicit save, execute and acknowledgement", async () => {
    const h = await start();
    await previewOne(h);
    expect(h.calls.previews[0].selection).toEqual([
      {
        workspace_id: collectionWorkspace.workspace_id,
        source_id: "fictional-checking",
        observation_id: "fictional-observation-0",
        observation_revision: "1",
      },
    ]);
    expect(h.calls.previews[0].action).toEqual({
      kind: "topic",
      topic_id: "dining",
    });
    expect(h.calls.execute).toBe(0);
    await h.user.click(
      screen.getByRole("button", { name: "Save label review" }),
    );
    await screen.findByText("This saved review has not been applied.");
    expect(h.calls.execute).toBe(0);
    await h.user.click(
      screen.getByRole("button", { name: "Confirm labels for 1 selected" }),
    );
    await screen.findByText("Labels applied. No journal was created.");
    expect(h.calls.execute).toBe(1);
    expect(h.calls.acknowledge).toBe(0);
    await h.user.click(
      screen.getByRole("button", { name: "Acknowledge result" }),
    );
    await screen.findByText("1–25 of 3000 matching transactions");
    expect(h.calls.acknowledge).toBe(1);
  });
  it("keeps explicit Uncategorized distinct from an unclassified filter or missing choice", async () => {
    const h = await start();
    await previewOne(h, "uncategorized");
    expect(h.calls.previews[0].action).toEqual({ kind: "uncategorized" });
    expect(screen.getByRole("region", { name: "Label preview" })).toBeTruthy();
  });
  it("recovers a lost prepare acknowledgement without executing or issuing a new key", async () => {
    const h = await start({ scenario: "lost_prepare" });
    await previewOne(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save label review" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    expect(h.calls.prepares).toHaveLength(1);
    const key = h.calls.prepares[0].prepare_key;
    await h.user.click(
      screen.getByRole("button", { name: "Recover saved review" }),
    );
    await screen.findByText("This saved review has not been applied.");
    expect(h.calls.prepares).toHaveLength(1);
    expect(h.calls.prepares[0].prepare_key).toBe(key);
    expect(h.calls.execute).toBe(0);
    expect(h.calls.acknowledge).toBe(0);
  });
  it("an absent discovery after uncertain prepare never permits new intent and retries the same key", async () => {
    const h = await start({ scenario: "lost_prepare" });
    await previewOne(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save label review" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    h.port.pending = async () => null;
    await h.user.click(
      screen.getByRole("button", { name: "Recover saved review" }),
    );
    await screen.findByText(/same saved review is not available yet/);
    await h.user.click(
      screen.getByRole("button", { name: "Retry saving same review" }),
    );
    await screen.findByText("This saved review has not been applied.");
    expect(h.calls.prepares).toHaveLength(2);
    expect(h.calls.prepares[0]).toEqual(h.calls.prepares[1]);
    expect(h.calls.execute).toBe(0);
  });
  it("reload discovery displays saved intent without automatically executing or acknowledging", async () => {
    const h = collectionFixture({ scenario: "pending" });
    render(
      <CollectionReview
        workspace={collectionWorkspace}
        sources={collectionSources}
        catalog={collectionCatalog}
        port={h.port}
        onDenied={() => {}}
      />,
    );
    await screen.findByText("This saved review has not been applied.");
    expect(document.activeElement).toBe(
      screen.getByRole("heading", { name: "Saved label review" }),
    );
    expect(h.calls.queries).toHaveLength(0);
    expect(h.calls.execute).toBe(0);
    expect(h.calls.acknowledge).toBe(0);
  });
  it("rejects a changed immutable preview in a command outcome", async () => {
    const h = await start();
    await previewOne(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save label review" }),
    );
    await screen.findByText("This saved review has not been applied.");
    const execute = h.port.execute;
    h.port.execute = async (identity) => {
      const result = await execute(identity);
      result.preview.review_rows[0].description = "Changed intent";
      return result;
    };
    await h.user.click(
      screen.getByRole("button", { name: "Confirm labels for 1 selected" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    expect(
      screen.queryByText("Labels applied. No journal was created."),
    ).toBeNull();
    expect(h.calls.acknowledge).toBe(0);
  });
  it("a lost terminal acknowledgement retries the same command rather than starting new intent", async () => {
    const h = await start();
    await previewOne(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save label review" }),
    );
    await screen.findByText("This saved review has not been applied.");
    await h.user.click(
      screen.getByRole("button", { name: "Confirm labels for 1 selected" }),
    );
    await screen.findByText("Labels applied. No journal was created.");
    const acknowledge = h.port.acknowledge;
    let dropped = false;
    h.port.acknowledge = async (identity) => {
      const result = await acknowledge(identity);
      if (!dropped) {
        dropped = true;
        throw new CollectionFailure("uncertain");
      }
      return result;
    };
    await h.user.click(
      screen.getByRole("button", { name: "Acknowledge result" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    await h.user.click(
      screen.getByRole("button", { name: "Recover saved review" }),
    );
    await screen.findByText(/same saved review is not available yet/);
    expect(h.calls.queries).toHaveLength(1);
    await h.user.click(
      screen.getByRole("button", { name: "Retry same acknowledgement" }),
    );
    await screen.findByText("1–25 of 3000 matching transactions");
    expect(h.calls.prepares).toHaveLength(1);
    expect(h.calls.execute).toBe(1);
    expect(h.calls.acknowledge).toBe(2);
  });
});

describe("authoritative terminal command outcomes", () => {
  it("never replaces a known committed outcome with cancelled during uncertain acknowledgement recovery", async () => {
    const h = await save();
    await h.user.click(
      screen.getByRole("button", { name: "Confirm labels for 1 selected" }),
    );
    await screen.findByText("Labels applied. No journal was created.");
    h.port.acknowledge = async () => {
      throw new CollectionFailure("uncertain");
    };
    const pending = h.port.pending;
    h.port.pending = async (workspace) => {
      const command = await pending(workspace);
      return command ? { ...command, state: "cancelled" } : null;
    };
    await h.user.click(
      screen.getByRole("button", { name: "Acknowledge result" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    await h.user.click(
      screen.getByRole("button", { name: "Recover saved review" }),
    );
    await screen.findByText(/same saved review is not available yet/);
    expect(screen.queryByText("Saved label review cancelled.")).toBeNull();
    expect(
      screen.getByText("Labels applied. No journal was created."),
    ).toBeTruthy();
    expect(h.calls.queries).toHaveLength(1);
  });
  // Each synthetic fixture starts from the same exact reviewed immutable selection.
  async function save() {
    const h = await start();
    await previewOne(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save label review" }),
    );
    await screen.findByText("This saved review has not been applied.");
    expect(document.activeElement).toBe(
      screen.getByRole("heading", { name: "Saved label review" }),
    );
    return h;
  }
  it("shows a commit winning cancellation without inferring cancellation or issuing another execute", async () => {
    const h = await save(),
      serverCommit = h.port.execute;
    h.port.execute = vi.fn();
    h.port.cancel = vi.fn((identity) => serverCommit(identity));
    await h.user.click(
      screen.getByRole("button", { name: "Cancel saved review" }),
    );
    await screen.findByText("Labels applied. No journal was created.");
    expect(h.port.cancel).toHaveBeenCalledOnce();
    expect(h.port.execute).not.toHaveBeenCalled();
    expect(h.calls.acknowledge).toBe(0);
  });
  it("shows a cancellation winning execution as cancelled and still requires acknowledgement", async () => {
    const h = await save(),
      serverCancel = h.port.cancel;
    h.port.execute = vi.fn((identity) => serverCancel(identity));
    await h.user.click(
      screen.getByRole("button", { name: "Confirm labels for 1 selected" }),
    );
    await screen.findByText("Saved label review cancelled.");
    expect(
      screen.queryByText("Labels applied. No journal was created."),
    ).toBeNull();
    expect(h.calls.acknowledge).toBe(0);
  });
  it("rejects acknowledgement that switches an already-known terminal state", async () => {
    const h = await save();
    await h.user.click(
      screen.getByRole("button", { name: "Confirm labels for 1 selected" }),
    );
    await screen.findByText("Labels applied. No journal was created.");
    const ack = h.port.acknowledge;
    h.port.acknowledge = async (identity) => ({
      ...(await ack(identity)),
      state: "cancelled",
    });
    await h.user.click(
      screen.getByRole("button", { name: "Acknowledge result" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    expect(h.calls.queries).toHaveLength(1);
    expect(h.calls.acknowledge).toBe(1);
    expect(screen.queryByText("Saved label review cancelled.")).toBeNull();
  });
});
