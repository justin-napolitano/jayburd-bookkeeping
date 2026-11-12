import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CollectionReview } from "./CollectionReview";
import { CollectionFailure } from "./types";
import type { AutomationStatus } from "./automation";
import {
  collectionCatalog,
  collectionFixture,
  collectionRow,
  collectionSources,
  collectionWorkspace,
} from "../../tests/collection-fixture";
afterEach(cleanup);
const workspace = {
  ...collectionWorkspace,
  workspace_id: "88000000-0000-4000-8000-000000000001",
};
const status: AutomationStatus = {
  profile: "private-collection-automation-v1",
  workspace_id: workspace.workspace_id,
  revision: "0",
  enabled: false,
  processing_state: "disabled",
  pending_count: "3000",
  attention_count: "17",
  blocked_reason: null,
  last_run: null,
};
function fixture() {
  return collectionFixture({
    rows: [collectionRow(0, { workspace_id: workspace.workspace_id })],
  });
}
it("does not call any new automation API when the advertised capability is absent", async () => {
  const f = fixture(),
    read = vi.fn(),
    configure = vi.fn();
  render(
    <CollectionReview
      workspace={workspace}
      sources={collectionSources}
      catalog={collectionCatalog}
      port={{ ...f.port, automation: { read, configure } }}
      onDenied={vi.fn()}
    />,
  );
  await screen.findByText(/matching transactions/);
  expect(read).not.toHaveBeenCalled();
  expect(configure).not.toHaveBeenCalled();
  expect(
    screen.queryByRole("region", { name: "Automatic categories" }),
  ).toBeNull();
});
it("requests the complete booked needs-attention selector without client page effects", async () => {
  const f = fixture();
  render(
    <CollectionReview
      workspace={workspace}
      sources={collectionSources}
      catalog={{
        ...collectionCatalog,
        automatic_categorization: { state: "available" },
      }}
      port={{
        ...f.port,
        automation: { read: async () => status, configure: vi.fn() },
      }}
      onDenied={vi.fn()}
    />,
  );
  await screen.findByText(/matching transactions/);
  await userEvent.click(
    await screen.findByRole("button", { name: "Needs attention" }),
  );
  await waitFor(() =>
    expect(f.calls.queries.at(-1)).toMatchObject({
      workspace_id: workspace.workspace_id,
      topic: { kind: "needs_attention" },
      status: { source: "booked", journal: "all" },
      snapshot: null,
      cursor: null,
      limit: 25,
    }),
  );
  expect(f.calls.previews).toHaveLength(0);
  expect(f.calls.prepares).toHaveLength(0);
  expect(f.calls.execute).toBe(0);
  expect(f.calls.acknowledge).toBe(0);
});
it("unknown configuration keeps scoped label actions and refresh locked while identical settings retry stays usable", async () => {
  const f = fixture(),
    locks = vi.fn(),
    configure = vi.fn().mockRejectedValue(new CollectionFailure("uncertain"));
  render(
    <CollectionReview
      workspace={workspace}
      sources={collectionSources}
      catalog={{
        ...collectionCatalog,
        automatic_categorization: { state: "available" },
      }}
      port={{ ...f.port, automation: { read: async () => status, configure } }}
      onDenied={vi.fn()}
      onLockChange={locks}
    />,
  );
  await screen.findByText(/matching transactions/);
  await userEvent.click(
    await screen.findByRole("button", { name: "Enable automatic categories" }),
  );
  await screen.findByText(/settings result is unknown/);
  expect(
    (screen.getAllByRole("checkbox")[0] as HTMLInputElement).disabled,
  ).toBe(true);
  expect(
    (
      screen.getByRole("button", {
        name: "Refresh transactions",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
  expect(
    (
      screen.getByRole("button", {
        name: "Retry same settings change",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(false);
  expect(locks).toHaveBeenLastCalledWith(true);
  await userEvent.click(
    screen.getByRole("button", { name: "Retry same settings change" }),
  );
  expect(configure.mock.calls[1]).toEqual(configure.mock.calls[0]);
  expect(f.calls.prepares).toHaveLength(0);
});

it("keeps automatic pending rows out of the explicit whole-result exceptions view", async () => {
  const rows = [
    collectionRow(0, {
      workspace_id: workspace.workspace_id,
      description: "Fictional retained exception",
      owner_topic: { state: "unclassified" },
      categorization: {
        state: "needs_attention",
        origin: "none",
        reason: "missing_provider_category",
        rule_names: [],
        provider: null,
        decision_id: "88000000-0000-4000-8000-000000000002",
      },
    }),
    collectionRow(1, {
      workspace_id: workspace.workspace_id,
      description: "Fictional automatic work still pending",
      owner_topic: { state: "unclassified" },
      categorization: {
        state: "pending",
        origin: "none",
        reason: "unavailable",
        rule_names: [],
        provider: null,
        decision_id: null,
      },
    }),
    collectionRow(2, {
      workspace_id: workspace.workspace_id,
      description: "Fictional source still pending",
      source_state: "pending",
      owner_topic: { state: "stale" },
      categorization: {
        state: "needs_attention",
        origin: "manual",
        reason: "stale_override",
        rule_names: [],
        provider: null,
        decision_id: null,
      },
    }),
  ];
  const f = collectionFixture({ rows });
  render(
    <CollectionReview
      workspace={workspace}
      sources={collectionSources}
      catalog={{
        ...collectionCatalog,
        automatic_categorization: { state: "available" },
      }}
      port={{
        ...f.port,
        automation: {
          read: async () => ({
            ...status,
            enabled: true,
            processing_state: "pending",
            pending_count: "100",
            attention_count: "1",
          }),
          configure: vi.fn(),
        },
      }}
      onDenied={vi.fn()}
    />,
  );
  await screen.findByText(/3 matching transactions/);
  await userEvent.click(
    await screen.findByRole("button", { name: "Needs attention" }),
  );
  await screen.findByText(/1 matching transactions/);
  const list = screen.getByRole("list", { name: "Transaction rows" });
  expect(list.textContent).toContain("Fictional retained exception");
  expect(list.textContent).not.toContain(
    "Fictional automatic work still pending",
  );
  expect(list.textContent).not.toContain("Fictional source still pending");
  expect(
    (screen.getByLabelText("Topic filter") as HTMLSelectElement).value,
  ).toBe("needs_attention");
  expect(f.calls.queries.at(-1)).toMatchObject({
    topic: { kind: "needs_attention" },
    status: { source: "booked", journal: "all" },
  });
  expect(f.calls.execute).toBe(0);
});
