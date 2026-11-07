import { APIError, type API } from "../bookkeeping/api";
import {
  CollectionFailure,
  validTopics,
  type Catalog,
  type CollectionPort,
  type LabelCommand,
  type SourceChoice,
  type Workspace,
} from "./types";

export type CollectionBootstrap = {
  workspace: Workspace;
  catalog: Catalog;
  sources: SourceChoice[];
  canClassify: boolean;
};
const record = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
const revision = (value: unknown): value is string =>
  typeof value === "string" && /^(0|[1-9][0-9]{0,18})$/.test(value);
const id = (value: unknown): value is string =>
  typeof value === "string" &&
  /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(value) &&
  value !== "00000000-0000-0000-0000-000000000000";

// Only explicit discovery404 means absent/unauthorized; no request enrolls or grants access.
export async function discoverCollection(
  api: API,
): Promise<CollectionBootstrap | null> {
  let workspace: unknown;
  try {
    workspace = await api<unknown>("/v1/collection");
  } catch (reason) {
    if (reason instanceof APIError && reason.status === 404) return null;
    throw new CollectionFailure("unavailable");
  }
  if (
    !record(workspace) ||
    workspace.profile !== "private-source-workspace-v1" ||
    !id(workspace.workspace_id) ||
    !revision(workspace.workspace_revision) ||
    !Array.isArray(workspace.capabilities) ||
    workspace.capabilities.some(
      (capability) =>
        !["read", "acquire", "classify"].includes(String(capability)),
    ) ||
    new Set(workspace.capabilities).size !== workspace.capabilities.length ||
    !workspace.capabilities.includes("read")
  )
    throw new CollectionFailure("unavailable");
  const context: Workspace = {
    workspace_id: workspace.workspace_id,
    authority_revision: workspace.workspace_revision,
    name: "Private transactions",
  };
  let catalog: unknown;
  try {
    catalog = await api<unknown>(
      `/v1/collections/${encodeURIComponent(context.workspace_id)}/taxonomy`,
    );
  } catch (reason) {
    throw mapFailure(reason, false);
  }
  if (
    !record(catalog) ||
    catalog.profile !== "private-collection-catalog-v1" ||
    catalog.workspace_id !== context.workspace_id ||
    catalog.authority_revision !== context.authority_revision ||
    !revision(catalog.taxonomy_revision) ||
    !revision(catalog.ruleset_revision) ||
    !revision(catalog.map_revision) ||
    !validTopics(catalog.topics) ||
    !Array.isArray(catalog.sources) ||
    catalog.sources.length > 100 ||
    !catalog.sources.every(
      (source) =>
        record(source) &&
        id(source.source_id) &&
        typeof source.name === "string" &&
        source.name.length > 0 &&
        source.name.length <= 500,
    ) ||
    new Set(catalog.sources.map((source) => (source as SourceChoice).source_id))
      .size !== catalog.sources.length
  )
    throw new CollectionFailure("unavailable");
  return {
    workspace: context,
    catalog: {
      taxonomy_revision: catalog.taxonomy_revision,
      ruleset_revision: catalog.ruleset_revision,
      map_revision: catalog.map_revision,
      topics: catalog.topics,
    },
    sources: catalog.sources as SourceChoice[],
    canClassify: workspace.capabilities.includes("classify"),
  };
}

// Fixed failure classes discard raw service diagnostics while retaining mutation uncertainty.
function mapFailure(reason: unknown, mutation: boolean): CollectionFailure {
  if (reason instanceof CollectionFailure) return reason;
  if (reason instanceof APIError) {
    if ([401, 403, 404].includes(reason.status))
      return new CollectionFailure("denied");
    if (reason.status === 409) return new CollectionFailure("stale");
    if (mutation && !reason.uncertain && [400, 422].includes(reason.status))
      return new CollectionFailure("rejected");
    if (mutation && reason.uncertain) return new CollectionFailure("uncertain");
  }
  return new CollectionFailure(mutation ? "uncertain" : "unavailable");
}

// The injected existing API owns fresh authentication/no-store transport; no actor or browser store is used.
export function createCollectionPort(api: API): CollectionPort {
  const base = (workspace: string) =>
    `/v1/collections/${encodeURIComponent(workspace)}`;
  const body = (value: unknown, maximum: number) => {
    const json = JSON.stringify(value);
    if (new TextEncoder().encode(json).byteLength > maximum)
      throw new CollectionFailure("unavailable");
    return json;
  };
  const read = async <T>(
    path: string,
    value: unknown,
    maximum: number,
  ): Promise<T> => {
    try {
      if (!api.query) throw new CollectionFailure("unavailable");
      return await api.query<T>(path, body(value, maximum));
    } catch (reason) {
      throw mapFailure(reason, false);
    }
  };
  const command = async (
    workspace: string,
    path: string,
    value?: unknown,
    maximum = 1024,
  ): Promise<LabelCommand | null> => {
    const mutation = value !== undefined;
    try {
      const result = await api<unknown>(
        path,
        mutation ? body(value, maximum) : undefined,
      );
      if (
        !record(result) ||
        result.profile !== "private-collection-command-v1" ||
        result.workspace_id !== workspace ||
        !Object.hasOwn(result, "command") ||
        (result.command === null ? mutation : !record(result.command))
      )
        throw new CollectionFailure(mutation ? "uncertain" : "unavailable");
      return result.command as LabelCommand | null;
    } catch (reason) {
      throw mapFailure(reason, mutation);
    }
  };
  const transition =
    (operation: "execute" | "cancel" | "acknowledge") =>
    async (identity: { workspace_id: string; command_id: string }) =>
      (await command(
        identity.workspace_id,
        `${base(identity.workspace_id)}/label-commands/${encodeURIComponent(identity.command_id)}/${operation}`,
        {},
      ))!;
  return {
    query: ({ workspace_id, ...request }) =>
      read(`${base(workspace_id)}/review/query`, request, 8192),
    preview: ({ workspace_id, ...request }) =>
      read(`${base(workspace_id)}/labels/preview`, request, 524288),
    prepare: ({ workspace_id, ...request }) =>
      command(
        workspace_id,
        `${base(workspace_id)}/label-commands`,
        request,
        4096,
      ).then((result) => result!),
    pending: (workspace) =>
      command(workspace, `${base(workspace)}/label-commands/pending`),
    execute: transition("execute"),
    cancel: transition("cancel"),
    acknowledge: transition("acknowledge"),
  };
}
