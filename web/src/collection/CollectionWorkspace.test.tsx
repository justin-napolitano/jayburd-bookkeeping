import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { APIError, type API } from "../bookkeeping/api";
import { CollectionWorkspace } from "./CollectionWorkspace";
import {
  collectionCatalog,
  collectionPage,
  collectionPreview,
  collectionQuery,
  collectionSnapshot,
} from "../../tests/collection-fixture";

afterEach(cleanup);
const workspace = "10000000-0000-4000-8000-000000000001";
const access = {
  profile: "private-source-workspace-v1",
  workspace_id: workspace,
  workspace_revision: "1",
  capabilities: ["read", "classify"],
};
const catalog = {
  profile: "private-collection-catalog-v1",
  workspace_id: workspace,
  authority_revision: "1",
  ...collectionCatalog,
  sources: [],
};
const preview = collectionPreview(
  {
    workspace_id: workspace,
    snapshot: collectionSnapshot,
    selection: [],
    action: { kind: "taxonomy", topics: collectionCatalog.topics },
  },
  [],
);
function fixture(pending = false) {
  const command = {
    workspace_id: workspace,
    command_id: "saved-topic-command",
    prepare_key: "saved-topic-key",
    state: "prepared",
    acknowledged: false,
    preview,
  };
  const request = vi.fn(async (path: string) => {
    if (path === "/v1/collection") return access;
    if (path.endsWith("/taxonomy")) return catalog;
    if (path.endsWith("/pending"))
      return {
        profile: "private-collection-command-v1",
        workspace_id: workspace,
        command: pending ? command : null,
      };
    throw new Error("Unexpected fixture operation");
  });
  const api = request as unknown as API;
  api.query = vi.fn(async (_path, body) =>
    collectionPage(
      { ...collectionQuery(), ...JSON.parse(body), workspace_id: workspace },
      [],
    ),
  ) as API["query"];
  return { api, request };
}
describe("always-mounted private recovery coordinator", () => {
  it("discovers pending taxonomy work while another tool is active without querying rows or executing it", async () => {
    const { api, request } = fixture(true);
    const lock = vi.fn();
    render(
      <CollectionWorkspace
        api={api}
        active={false}
        bookLocked={false}
        onLockChange={lock}
      />,
    );
    await screen.findByRole("heading", { name: "Saved topic review" });
    expect(api.query).not.toHaveBeenCalled();
    expect(request.mock.calls.map(([path]) => path)).toEqual([
      "/v1/collection",
      `/v1/collections/${workspace}/taxonomy`,
      `/v1/collections/${workspace}/label-commands/pending`,
    ]);
    expect(lock).toHaveBeenLastCalledWith(true);
  });
  it("keeps book work locked on discovery uncertainty until an explicit successful feature-off read", async () => {
    const { api, request } = fixture();
    request.mockRejectedValueOnce(new APIError(503, true, "read"));
    const lock = vi.fn();
    render(
      <CollectionWorkspace
        api={api}
        active={false}
        bookLocked={false}
        onLockChange={lock}
      />,
    );
    await screen.findByRole("alert");
    expect(lock).toHaveBeenLastCalledWith(true);
    request.mockRejectedValueOnce(new APIError(404, false, "read"));
    await userEvent.click(
      screen.getByRole("button", {
        name: "Retry private transaction recovery",
      }),
    );
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
    expect(api.query).not.toHaveBeenCalled();
  });
  it("unlocks idle discovery without reading private rows until the private tool opens", async () => {
    const { api } = fixture();
    const lock = vi.fn();
    const view = render(
      <CollectionWorkspace
        api={api}
        active={false}
        bookLocked={false}
        onLockChange={lock}
      />,
    );
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
    expect(api.query).not.toHaveBeenCalled();
    view.rerender(
      <CollectionWorkspace
        api={api}
        active
        bookLocked={false}
        onLockChange={lock}
      />,
    );
    await screen.findByText("0 of 0 matching transactions");
    expect(api.query).toHaveBeenCalledTimes(1);
  });
  it("preserves the external book gate while discovering and never executes private work", async () => {
    const { api, request } = fixture(true);
    const lock = vi.fn();
    const view = render(
      <CollectionWorkspace api={api} active bookLocked onLockChange={lock} />,
    );
    await waitFor(() => expect(request).toHaveBeenCalledTimes(3));
    expect(
      screen.queryByRole("button", { name: "Confirm topic changes" }),
    ).toBeNull();
    expect(lock).toHaveBeenLastCalledWith(true);
    view.rerender(
      <CollectionWorkspace
        api={api}
        active
        bookLocked={false}
        onLockChange={lock}
      />,
    );
    await screen.findByRole("button", { name: "Confirm topic changes" });
    expect(request.mock.calls.some(([path]) => path.endsWith("/execute"))).toBe(
      false,
    );
  });
});
