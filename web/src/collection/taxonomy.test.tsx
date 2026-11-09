import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CollectionReview } from "./CollectionReview";
import {
  CollectionFailure,
  validCommand,
  validTopics,
  sameAction,
  samePreview,
  validPreview,
  validPage,
  type LabelCommand,
} from "./types";
import {
  collectionCatalog,
  collectionFixture,
  collectionPreview,
  collectionSnapshot,
  collectionSources,
  collectionWorkspace,
  collectionPage,
  collectionQuery,
  collectionRow,
} from "../../tests/collection-fixture";

afterEach(cleanup);
async function editor(scenario?: "lost_prepare") {
  const fixture = collectionFixture({ scenario });
  const lock = vi.fn();
  const user = userEvent.setup();
  render(
    <CollectionReview
      workspace={collectionWorkspace}
      sources={collectionSources}
      catalog={collectionCatalog}
      port={fixture.port}
      onDenied={() => {}}
      onLockChange={lock}
    />,
  );
  await screen.findByText(/matching transactions/);
  await user.click(screen.getByText("Manage topics"));
  return { ...fixture, user, lock };
}

describe("private topic editing and shared command recovery", () => {
  async function draftTopic(h: Awaited<ReturnType<typeof editor>>) {
    await h.user.clear(screen.getByLabelText("Topic 1 name"));
    await h.user.type(screen.getByLabelText("Topic 1 name"), "Food");
    await h.user.click(
      screen.getByRole("button", { name: "Preview topic changes" }),
    );
    await screen.findByRole("heading", { name: "Review topic changes" });
  }
  function completedReceipt(
    h: Awaited<ReturnType<typeof editor>>,
  ): LabelCommand {
    return {
      workspace_id: collectionWorkspace.workspace_id,
      command_id: "fictional-command",
      prepare_key: h.calls.prepares[0].prepare_key,
      state: "committed",
      acknowledged: true,
      preview: collectionPreview(h.calls.previews[0], []),
    };
  }
  it("shows an exact acknowledged terminal prepare replay and requires explicit local acknowledgement", async () => {
    const h = await editor("lost_prepare");
    await draftTopic(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save topic review" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    const receipt = completedReceipt(h);
    h.port.prepare = vi.fn().mockResolvedValue(receipt);
    h.port.acknowledge = vi.fn().mockResolvedValue(receipt);
    await h.user.click(
      screen.getByRole("button", { name: "Retry saving same review" }),
    );
    await screen.findByText("Topics updated. No journal was created.");
    expect(screen.getByText(/acknowledged in another session/)).toBeTruthy();
    expect(h.port.acknowledge).not.toHaveBeenCalled();
    expect(h.lock).toHaveBeenLastCalledWith(true);
    await h.user.click(
      screen.getByRole("button", { name: "Acknowledge result" }),
    );
    await waitFor(() =>
      expect(h.port.acknowledge).toHaveBeenCalledWith({
        workspace_id: collectionWorkspace.workspace_id,
        command_id: "fictional-command",
      }),
    );
    await waitFor(() => expect(h.lock).toHaveBeenLastCalledWith(false));
  });
  it("shows an exact acknowledged terminal execute replay after a lost response without auto-acknowledgement", async () => {
    const h = await editor();
    await draftTopic(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save topic review" }),
    );
    await screen.findByRole("heading", { name: "Saved topic review" });
    const receipt = completedReceipt(h);
    h.port.execute = vi
      .fn()
      .mockRejectedValueOnce(new CollectionFailure("uncertain"))
      .mockResolvedValueOnce(receipt);
    h.port.acknowledge = vi.fn().mockResolvedValue(receipt);
    await h.user.click(
      screen.getByRole("button", { name: "Confirm topic changes" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    await h.user.click(
      screen.getByRole("button", { name: "Retry same confirmation" }),
    );
    await screen.findByText("Topics updated. No journal was created.");
    expect(h.port.acknowledge).not.toHaveBeenCalled();
    expect(h.lock).toHaveBeenLastCalledWith(true);
    await h.user.click(
      screen.getByRole("button", { name: "Acknowledge result" }),
    );
    await waitFor(() => expect(h.port.acknowledge).toHaveBeenCalledTimes(1));
  });
  it("keeps the shared gate closed after a first definite rejection until fresh pending:null enables a new review", async () => {
    const h = await editor();
    await draftTopic(h);
    let finish!: (command: LabelCommand | null) => void;
    h.port.prepare = vi.fn().mockRejectedValue(new CollectionFailure("stale"));
    h.port.pending = vi.fn().mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    await h.user.click(
      screen.getByRole("button", { name: "Save topic review" }),
    );
    await waitFor(() => expect(h.port.pending).toHaveBeenCalledTimes(1));
    expect(h.lock).toHaveBeenLastCalledWith(true);
    expect(
      screen.queryByRole("button", { name: "Save topic review" }),
    ).toBeNull();
    await act(async () => finish(null));
    await screen.findByText(/matching transactions/);
    await waitFor(() => expect(h.lock).toHaveBeenLastCalledWith(false));
    expect(h.calls.execute).toBe(0);
    expect(h.calls.acknowledge).toBe(0);
  });
  it("restores another tab's authoritative pending command after a first definite409 without executing it", async () => {
    const h = await editor();
    await draftTopic(h);
    const other: LabelCommand = {
      workspace_id: collectionWorkspace.workspace_id,
      command_id: "other-tab-command",
      prepare_key: "other-tab-key",
      state: "prepared",
      acknowledged: false,
      preview: collectionPreview(
        {
          ...h.calls.previews[0],
          action: {
            kind: "taxonomy",
            topics: [
              { ...collectionCatalog.topics[0], name: "Other reviewed topic" },
              collectionCatalog.topics[1],
            ],
          },
        },
        [],
      ),
    };
    h.port.prepare = vi.fn().mockRejectedValue(new CollectionFailure("stale"));
    h.port.pending = vi.fn().mockResolvedValue(other);
    await h.user.click(
      screen.getByRole("button", { name: "Save topic review" }),
    );
    await screen.findByRole("heading", { name: "Saved topic review" });
    expect(screen.getByText("Other reviewed topic")).toBeTruthy();
    expect(h.lock).toHaveBeenLastCalledWith(true);
    expect(h.calls.execute).toBe(0);
    expect(h.calls.acknowledge).toBe(0);
  });
  it("does not forget a previously uncertain prepare key after a definite retry rejection and pending:null", async () => {
    const h = await editor("lost_prepare");
    await draftTopic(h);
    await h.user.click(
      screen.getByRole("button", { name: "Save topic review" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    const initial = h.calls.prepares[0];
    h.port.prepare = vi.fn().mockRejectedValue(new CollectionFailure("stale"));
    await h.user.click(
      screen.getByRole("button", { name: "Retry saving same review" }),
    );
    await waitFor(() => expect(h.port.prepare).toHaveBeenCalledWith(initial));
    h.port.pending = vi.fn().mockResolvedValue(null);
    await h.user.click(
      screen.getByRole("button", { name: "Recover saved review" }),
    );
    await screen.findByText(/same saved review is not available yet/);
    expect(h.lock).toHaveBeenLastCalledWith(true);
    expect(
      screen.getByRole("button", { name: "Retry saving same review" }),
    ).toBeTruthy();
    expect(h.calls.execute).toBe(0);
  });
  it("retains stable IDs on rename,parent and retirement, then previews exact complete edition with no selection", async () => {
    const h = await editor();
    await h.user.clear(screen.getByLabelText("Topic 1 name"));
    await h.user.type(screen.getByLabelText("Topic 1 name"), "Food");
    await h.user.selectOptions(
      screen.getByLabelText("Topic 2 parent"),
      "dining",
    );
    await h.user.selectOptions(
      screen.getByLabelText("Topic 2 status"),
      "retired",
    );
    await h.user.click(
      screen.getByRole("button", { name: "Preview topic changes" }),
    );
    expect(
      await screen.findByRole("heading", { name: "Review topic changes" }),
    ).toBeTruthy();
    expect(h.calls.previews).toEqual([
      {
        workspace_id: collectionWorkspace.workspace_id,
        snapshot: collectionSnapshot,
        selection: [],
        action: {
          kind: "taxonomy",
          topics: [
            {
              topic_id: "dining",
              name: "Food",
              parent_id: null,
              status: "active",
            },
            {
              topic_id: "groceries",
              name: "Groceries",
              parent_id: "dining",
              status: "retired",
            },
          ],
        },
      },
    ]);
    expect(h.calls.prepares).toHaveLength(0);
    await waitFor(() => expect(h.lock).toHaveBeenLastCalledWith(true));
    await h.user.click(
      screen.getByRole("button", { name: "Save topic review" }),
    );
    await screen.findByRole("heading", { name: "Saved topic review" });
    expect(h.calls.execute).toBe(0);
    await h.user.click(
      screen.getByRole("button", { name: "Confirm topic changes" }),
    );
    await screen.findByText("Topics updated. No journal was created.");
    expect(h.calls.acknowledge).toBe(0);
    await h.user.click(
      screen.getByRole("button", { name: "Acknowledge result" }),
    );
    await waitFor(() => expect(h.calls.acknowledge).toBe(1));
    expect(h.calls.execute).toBe(1);
  });
  it("shares exact prepare-key recovery for a lost taxonomy prepare without automatic execution or acknowledgement", async () => {
    const h = await editor("lost_prepare");
    await h.user.clear(screen.getByLabelText("Topic 1 name"));
    await h.user.type(screen.getByLabelText("Topic 1 name"), "Food");
    await h.user.click(
      screen.getByRole("button", { name: "Preview topic changes" }),
    );
    await screen.findByRole("heading", { name: "Review topic changes" });
    await h.user.click(
      screen.getByRole("button", { name: "Save topic review" }),
    );
    await screen.findByText(/saved review outcome is unknown/);
    const key = h.calls.prepares[0].prepare_key;
    await h.user.click(
      screen.getByRole("button", { name: "Recover saved review" }),
    );
    await screen.findByRole("heading", { name: "Saved topic review" });
    expect(h.calls.prepares).toHaveLength(1);
    expect(h.calls.prepares[0].prepare_key).toBe(key);
    expect(h.calls.execute).toBe(0);
    expect(h.calls.acknowledge).toBe(0);
    await h.user.click(
      screen.getByRole("button", { name: "Cancel saved review" }),
    );
    await screen.findByText("Saved topic review cancelled.");
    expect(h.calls.acknowledge).toBe(0);
  });
  it("recovers a committed historic taxonomy preview after the current catalog and authority editions changed", async () => {
    const fixture = collectionFixture();
    const request = {
      workspace_id: collectionWorkspace.workspace_id,
      snapshot: collectionSnapshot,
      selection: [],
      action: { kind: "taxonomy" as const, topics: collectionCatalog.topics },
    };
    const preview = collectionPreview(request, []);
    const command = {
      workspace_id: collectionWorkspace.workspace_id,
      command_id: "fictional-old-command",
      prepare_key: "fictional-old-key",
      state: "committed" as const,
      acknowledged: false,
      preview,
    };
    const workspace = { ...collectionWorkspace, authority_revision: "2" };
    const catalog = { ...collectionCatalog, taxonomy_revision: "4" };
    expect(validCommand(command, workspace, catalog)).toBe(true);
    fixture.port.pending = vi.fn().mockResolvedValue(command);
    render(
      <CollectionReview
        workspace={workspace}
        catalog={catalog}
        sources={collectionSources}
        port={fixture.port}
        onDenied={() => {}}
      />,
    );
    await screen.findByRole("heading", { name: "Saved topic review" });
    expect(
      screen.getByText("Topics updated. No journal was created."),
    ).toBeTruthy();
    expect(fixture.calls.execute).toBe(0);
    expect(fixture.calls.acknowledge).toBe(0);
  });
  it("rejects invalid hierarchies, active children of retired parents and changed immutable topic intent", () => {
    const topics = collectionCatalog.topics;
    expect(validTopics(topics)).toBe(true);
    expect(
      validTopics([
        { ...topics[0], parent_id: "groceries" },
        { ...topics[1], parent_id: "dining" },
      ]),
    ).toBe(false);
    expect(
      validTopics([
        { ...topics[0], status: "retired" },
        { ...topics[1], parent_id: "dining" },
      ]),
    ).toBe(false);
    expect(validTopics([{ ...topics[0], name: "x".repeat(81) }])).toBe(false);
    expect(
      sameAction(
        { kind: "taxonomy", topics },
        {
          kind: "taxonomy",
          topics: [{ ...topics[0], name: "Changed" }, topics[1]],
        },
      ),
    ).toBe(false);
    const request = {
      workspace_id: collectionWorkspace.workspace_id,
      snapshot: collectionSnapshot,
      selection: [],
      action: { kind: "taxonomy" as const, topics },
    };
    expect(validPreview(collectionPreview(request, []), request)).toBe(true);
    const reordered = collectionPreview(
      {
        ...request,
        action: { kind: "taxonomy", topics: [...topics].reverse() },
      },
      [],
    );
    expect(validPreview(reordered, request)).toBe(true);
    expect(samePreview(reordered, collectionPreview(request, []))).toBe(false);
    expect(
      validPreview(
        { ...collectionPreview(request, []), selected_count: "1" },
        request,
      ),
    ).toBe(false);
  });
  it("keeps retired history readonly and excludes it from new assignments", async () => {
    const fixture = collectionFixture();
    render(
      <CollectionReview
        workspace={collectionWorkspace}
        sources={collectionSources}
        catalog={{
          ...collectionCatalog,
          topics: [
            { ...collectionCatalog.topics[0], status: "retired" },
            collectionCatalog.topics[1],
          ],
        }}
        port={fixture.port}
        onDenied={() => {}}
      />,
    );
    await screen.findByText(/matching transactions/);
    await userEvent.click(screen.getByText("Manage topics"));
    expect(screen.getByText("Dining · Retired (history only)")).toBeTruthy();
    expect(screen.queryByLabelText("Topic 1 name")).toBeNull();
    await userEvent.click(
      screen.getByRole("checkbox", {
        name: "Select Fictional purchase 1 on 2026-10-01 from Fictional checking",
      }),
    );
    const label = screen.getByLabelText("Label selected transactions");
    expect(
      [...label.querySelectorAll("option")].map((option) => option.value),
    ).not.toContain("topic:dining");
    await act(async () => {});
  });
  it("preserves80 Unicode codepoints in catalog and owner labels and rejects changed map/decision pins", () => {
    const name = "🍎".repeat(80);
    expect(validTopics([{ ...collectionCatalog.topics[0], name }])).toBe(true);
    expect(
      validTopics([{ ...collectionCatalog.topics[0], name: " Food " }]),
    ).toBe(true);
    expect(validTopics([{ ...collectionCatalog.topics[0], name: "   " }])).toBe(
      false,
    );
    expect(
      validTopics([{ ...collectionCatalog.topics[0], name: "Food\n" }]),
    ).toBe(false);
    expect(
      validTopics([{ ...collectionCatalog.topics[0], name: `${name}🍎` }]),
    ).toBe(false);
    const q = { ...collectionQuery(), snapshot: collectionSnapshot };
    const page = collectionPage(q, [
      collectionRow(0, {
        owner_topic: { state: "topic", topic_id: "dining", name },
      }),
    ]);
    expect(validPage(page, q, collectionWorkspace, collectionCatalog)).toBe(
      true,
    );
    for (const key of ["map_revision", "decision_revision"] as const) {
      expect(
        validPage(
          { ...page, snapshot: { ...page.snapshot, [key]: "99" } },
          q,
          collectionWorkspace,
          collectionCatalog,
        ),
      ).toBe(false);
    }
  });
});
