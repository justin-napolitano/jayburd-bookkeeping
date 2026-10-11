import type { components } from "../lib/api.generated";
// API-owned shapes come only from the checked OpenAPI generator output.
export type Currency = components["schemas"]["WorkspaceCurrency"];
export type Account = components["schemas"]["WorkspaceAccount"];
export type Book = components["schemas"]["WorkspaceBook"];
export type Entity = components["schemas"]["WorkspaceEntity"];
export type Workspace = components["schemas"]["Workspace"];
export type TrialBalance = components["schemas"]["TrialBalance"];
export type Journal = components["schemas"]["Journal"];

export type DurableCommand = components["schemas"]["DurableCommand"];
export type RegisterCommand = components["schemas"]["RegisterCommand"];
export type PendingCommand = components["schemas"]["PendingCommand"];
