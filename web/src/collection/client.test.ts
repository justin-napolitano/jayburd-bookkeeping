import { describe, expect, it, vi } from "vitest";
import { APIError, type API } from "../bookkeeping/api";
import { createCollectionPort, discoverCollection } from "./client";
import {
  collectionCatalog,
  collectionPreview,
  collectionQuery,
  collectionRow,
  collectionSnapshot,
} from "../../tests/collection-fixture";
import { selectionOf } from "./types";

const workspace = "10000000-0000-4000-8000-000000000001";
const source = "20000000-0000-4000-8000-000000000001";
const access = {
  profile: "private-source-workspace-v1",
  workspace_id: workspace,
  workspace_revision: "1",
  capabilities: ["read", "classify"],
};
const catalog = {
  profile: "private-collection-catalog-v2",
  workspace_id: workspace,
  authority_revision: "1",
  ...collectionCatalog,
  sources: [{ source_id: source, name: "Fictional checking" }],
};
function apiFixture() {
  const request = vi.fn(async (path: string) =>
    path === "/v1/collection" ? access : catalog,
  );
  const api = request as unknown as API;
  api.query = vi.fn();
  return { api, request };
}

describe("authenticated private collection port", () => {
  it("transports a full4096-character cursor with100 astral search codepoints without truncation", async () => {
    const { api } = apiFixture();
    const query = vi.mocked(api.query!);
    query.mockResolvedValueOnce({});
    const cursor = "a".repeat(4096);
    const search = "🍎".repeat(100);
    await createCollectionPort(api).query({
      ...collectionQuery(),
      workspace_id: workspace,
      cursor,
      search,
      snapshot: collectionSnapshot,
    });
    const body = JSON.parse(query.mock.calls[0][1]);
    expect(body.cursor).toBe(cursor);
    expect(body.search).toBe(search);
    expect(
      new TextEncoder().encode(query.mock.calls[0][1]).byteLength,
    ).toBeLessThanOrEqual(8192);
  });
  it("discovers only current authorized scope and complete real catalog, mapping source42 revision", async () => {
    const { api, request } = apiFixture();
    expect(await discoverCollection(api)).toEqual({
      workspace: {
        workspace_id: workspace,
        authority_revision: "1",
        name: "Private transactions",
      },
      catalog: collectionCatalog,
      sources: catalog.sources,
      canClassify: true,
    });
    expect(request.mock.calls).toEqual([
      ["/v1/collection"],
      [`/v2/collections/${workspace}/taxonomy`],
    ]);
  });
  it("does not selfgrant or fabricate a catalog for explicit feature-off404", async () => {
    const { api, request } = apiFixture();
    request.mockRejectedValueOnce(new APIError(404, false, "read"));
    expect(await discoverCollection(api)).toBeNull();
    expect(request).toHaveBeenCalledTimes(1);
  });
  it("keeps unknown recovery unavailable on network,5xx, malformed discovery or mismatched catalog", async () => {
    for (const value of [
      new APIError(503, true, "read"),
      new Error("fictional network"),
    ]) {
      const { api, request } = apiFixture();
      request.mockRejectedValueOnce(value);
      await expect(discoverCollection(api)).rejects.toMatchObject({
        kind: "unavailable",
      });
    }
    for (const value of [
      { ...catalog, workspace_id: source },
      { ...catalog, authority_revision: "2" },
      { ...catalog, topics: [...catalog.topics, catalog.topics[0]] },
      { ...catalog, sources: [...catalog.sources, ...catalog.sources] },
    ]) {
      const { api, request } = apiFixture();
      request
        .mockResolvedValueOnce(access)
        .mockResolvedValueOnce(value as typeof catalog);
      await expect(discoverCollection(api)).rejects.toMatchObject({
        kind: "unavailable",
      });
    }
  });
  it("denies a revoked taxonomy read and never promotes a read capability to classify", async () => {
    const { api, request } = apiFixture();
    request.mockResolvedValueOnce({ ...access, capabilities: ["read"] });
    expect((await discoverCollection(api))?.canClassify).toBe(false);
    request
      .mockResolvedValueOnce(access)
      .mockRejectedValueOnce(new APIError(404, false, "read"));
    await expect(discoverCollection(api)).rejects.toMatchObject({
      kind: "denied",
    });
  });
  it("uses all seven frozen operations, exact bodies and wrappers without actor or Idempotency-Key", async () => {
    const { api, request } = apiFixture();
    const port = createCollectionPort(api);
    const q = { ...collectionQuery(), workspace_id: workspace };
    const query = vi.mocked(api.query!);
    query.mockResolvedValueOnce({});
    await port.query(q);
    const { workspace_id: _, ...queryBody } = q;
    expect(query.mock.calls[0]).toEqual([
      `/v2/collections/${workspace}/review/query`,
      JSON.stringify(queryBody),
    ]);
    const previewRequest = {
      workspace_id: workspace,
      snapshot: collectionSnapshot,
      selection: [selectionOf(collectionRow())],
      action: { kind: "uncategorized" as const },
    };
    const preview = collectionPreview(previewRequest, [collectionRow()]);
    query.mockResolvedValueOnce(preview);
    expect(await port.preview(previewRequest)).toEqual(preview);
    const { workspace_id: __, ...previewBody } = previewRequest;
    expect(query.mock.calls[1]).toEqual([
      `/v2/collections/${workspace}/labels/preview`,
      JSON.stringify(previewBody),
    ]);
    const command = {
      workspace_id: workspace,
      command_id: "fictional-command",
      prepare_key: "fictional-key",
      state: "prepared",
      acknowledged: false,
      preview,
    };
    request.mockResolvedValue({
      profile: "private-collection-command-v2",
      workspace_id: workspace,
      command,
    } as never);
    expect(
      await port.prepare({
        workspace_id: workspace,
        preview_id: preview.preview_id,
        prepare_key: "fictional-key",
      }),
    ).toEqual(command);
    await port.pending(workspace);
    for (const operation of ["execute", "cancel", "acknowledge"] as const)
      await port[operation]({
        workspace_id: workspace,
        command_id: "fictional-command",
      });
    expect(request.mock.calls).toEqual([
      [
        `/v2/collections/${workspace}/label-commands`,
        JSON.stringify({
          preview_id: preview.preview_id,
          prepare_key: "fictional-key",
        }),
      ],
      [`/v2/collections/${workspace}/label-commands/pending`, undefined],
      ...["execute", "cancel", "acknowledge"].map((operation) => [
        `/v2/collections/${workspace}/label-commands/fictional-command/${operation}`,
        "{}",
      ]),
    ]);
  });
  it("rejects mismatched wrapper/null mutation replies and distinguishes stale, denied and uncertain errors", async () => {
    const { api, request } = apiFixture();
    const port = createCollectionPort(api);
    request.mockResolvedValue({
      profile: "private-collection-command-v2",
      workspace_id: workspace,
      command: null,
    } as never);
    expect(await port.pending(workspace)).toBeNull();
    await expect(
      port.execute({ workspace_id: workspace, command_id: "x" }),
    ).rejects.toMatchObject({ kind: "uncertain" });
    request.mockResolvedValue({
      profile: "private-collection-command-v2",
      workspace_id: source,
      command: null,
    } as never);
    await expect(port.pending(workspace)).rejects.toMatchObject({
      kind: "unavailable",
    });
    request.mockRejectedValue(new APIError(503, true));
    await expect(
      port.cancel({ workspace_id: workspace, command_id: "x" }),
    ).rejects.toMatchObject({ kind: "uncertain" });
    vi.mocked(api.query!).mockRejectedValueOnce(
      new APIError(409, false, "read"),
    );
    await expect(port.query(collectionQuery())).rejects.toMatchObject({
      kind: "stale",
    });
    request.mockRejectedValue(new APIError(404, false));
    await expect(port.pending(workspace)).rejects.toMatchObject({
      kind: "denied",
    });
  });
});
