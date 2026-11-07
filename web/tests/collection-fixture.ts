import {
  CollectionFailure,
  selectionKey,
  selectionOf,
  type Catalog,
  type CollectionPort,
  type LabelCommand,
  type Page,
  type Preview,
  type PreviewRequest,
  type Query,
  type Row,
  type Selector,
  type Workspace,
} from "../src/collection/types";

export const collectionWorkspace: Workspace = {
  workspace_id: "fictional-workspace-a",
  authority_revision: "1",
  name: "Fictional private workspace",
};
export const collectionCatalog: Catalog = {
  taxonomy_revision: "3",
  ruleset_revision: "4",
  map_revision: "0",
  tag_revision: "1",
  tags: [
    {
      tag_id: "41000000-0000-4000-8000-000000000001",
      name: "Travel",
      status: "active",
    },
    {
      tag_id: "41000000-0000-4000-8000-000000000002",
      name: "Tax review",
      status: "active",
    },
  ],
  topics: [
    { topic_id: "dining", name: "Dining", parent_id: null, status: "active" },
    {
      topic_id: "groceries",
      name: "Groceries",
      parent_id: null,
      status: "active",
    },
  ],
};
export const collectionSources = [
  { source_id: "fictional-checking", name: "Fictional checking" },
  { source_id: "fictional-card", name: "Fictional card" },
];
export const collectionSnapshot = {
  authority_revision: "1",
  source_snapshot: "a".repeat(64),
  taxonomy_revision: "3",
  ruleset_revision: "4",
  map_revision: "0",
  decision_revision: "0",
  tag_revision: "1",
};

// These independently authored financial examples are test data, never a runtime data source.
export function collectionRow(index = 0, changes: Partial<Row> = {}): Row {
  return {
    workspace_id: collectionWorkspace.workspace_id,
    source_id: index % 2 === 0 ? "fictional-checking" : "fictional-card",
    observation_id: `fictional-observation-${index}`,
    observation_revision: "1",
    account_name: index % 2 === 0 ? "Fictional checking" : "Fictional card",
    booked_date: "2026-10-01",
    description: `Fictional purchase ${index + 1}`,
    merchant: "Fictional Cafe",
    money: {
      state: "available",
      amount: "12.50",
      direction: "outflow",
      currency: "USD",
      currency_version_id: "fictional-usd-version",
      numeric_code: "840",
      minor_unit: 2,
      valid_from: "2000-01-01",
      valid_to: null,
    },
    source_state: "booked",
    assignment: { state: "unavailable" },
    journal_state: "unavailable",
    owner_topic: { state: "unclassified" },
    owner_tags: { state: "unclassified", tags: [] },
    provider_suggestion: {
      state: "attributed",
      label: "Food and drink",
      taxonomy_version: "v2",
      confidence: "HIGH",
    },
    ...changes,
  };
}

// Held quantities intentionally carry no invented zero, direction, currency or dated reference.
export function heldCollectionRow(
  reason: "unsupported_currency" | "invalid_precision" = "unsupported_currency",
): Row {
  return collectionRow(9000, {
    description: `Fictional ${reason} hold`,
    source_state: "held",
    money: {
      state: "held",
      reason,
      amount: null,
      direction: null,
      currency: null,
      currency_version_id: null,
      numeric_code: null,
      minor_unit: null,
      valid_from: null,
      valid_to: null,
    },
    provider_suggestion: { state: "unavailable" },
  });
}

// Existing dated currency references may omit a numeric code; this does not invalidate known money.
export function nullNumericCollectionRow(): Row {
  const row = collectionRow(9100, {
    description: "Fictional known currency without numeric code",
  });
  if (row.money.state === "available")
    row.money = {
      ...row.money,
      currency_version_id: "fictional-null-numeric-version",
      numeric_code: null,
    };
  return row;
}

// A provider removal retains the earlier unresolved currency hold without inventing money.
export function removedHeldCollectionRow(): Row {
  return {
    ...heldCollectionRow(),
    source_state: "removed",
    description: "Fictional removed currency hold",
  };
}

// Fixture selectors echo every actual filter independently from opaque pagination state.
export function collectionSelector(q: Query): Selector {
  return {
    source_account_id: q.source_account_id,
    assignment: q.assignment,
    starts_on: q.starts_on,
    through_date: q.through_date,
    search: q.search,
    topic: q.topic,
    tag: q.tag,
    sort: q.sort,
  };
}
export function collectionQuery(changes: Partial<Query> = {}): Query {
  return {
    profile: "private-collection-review-v2",
    workspace_id: collectionWorkspace.workspace_id,
    source_account_id: null,
    assignment: "all",
    starts_on: null,
    through_date: null,
    search: "",
    topic: { kind: "all" },
    tag: { kind: "all" },
    sort: "date_desc",
    snapshot: null,
    cursor: null,
    limit: 25,
    ...changes,
  };
}
export function collectionPage(
  q = collectionQuery(),
  rows = [collectionRow()],
): Page {
  return {
    profile: "private-collection-review-v2",
    workspace_id: q.workspace_id,
    selector: collectionSelector(q),
    snapshot: q.snapshot ?? collectionSnapshot,
    matching_count: String(rows.length),
    start_index: rows.length ? "1" : "0",
    items: rows,
    next_cursor: null,
    partial: rows.some((row) => row.source_state === "held"),
  };
}
export function collectionPreview(
  request: PreviewRequest,
  rows: Row[],
): Preview {
  return {
    ...request,
    preview_id: "fictional-preview",
    profile: "private-collection-preview-v2",
    expires_at: "2099-01-01T00:00:00Z",
    selected_count: String(request.selection.length),
    review_rows: rows,
    tag_results:
      request.action.kind !== "tags"
        ? []
        : rows
            .map((row) => {
              const action = request.action;
              if (action.kind !== "tags")
                throw Error("Invalid fictional tag action");
              const previous = row.owner_tags.tags.map((tag) => tag.tag_id);
              const ids =
                action.operation === "replace"
                  ? action.tag_ids
                  : action.operation === "add"
                    ? [...new Set([...previous, ...action.tag_ids])]
                    : previous.filter((id) => !action.tag_ids.includes(id));
              return {
                ...selectionOf(row),
                tags: [...ids]
                  .sort()
                  .map(
                    (id) =>
                      collectionCatalog.tags.find((tag) => tag.tag_id === id)!,
                  ),
              };
            })
            .sort((a, b) =>
              selectionKey(a) < selectionKey(b)
                ? -1
                : selectionKey(a) > selectionKey(b)
                  ? 1
                  : 0,
            ),
  };
}

// This test-only fake server exposes calls for UI assertions; no application imports it.
export function collectionFixture(
  options: {
    rows?: Row[];
    scenario?: "normal" | "lost_prepare" | "denied" | "pending";
    onCall?: (name: string) => void;
  } = {},
) {
  const rows =
    options.rows ??
    Array.from({ length: 3000 }, (_, index) => collectionRow(index));
  const calls = {
    queries: [] as Query[],
    previews: [] as PreviewRequest[],
    prepares: [] as {
      workspace_id: string;
      preview_id: string;
      prepare_key: string;
    }[],
    execute: 0,
    cancel: 0,
    acknowledge: 0,
    pending: 0,
  };
  const firstPreview = collectionPreview(
    {
      workspace_id: collectionWorkspace.workspace_id,
      snapshot: collectionSnapshot,
      selection: rows.slice(0, 1).map(selectionOf),
      action: { kind: "topic", topic_id: "dining" },
    },
    rows.slice(0, 1),
  );
  let latestPreview: Preview | null = null,
    saved: LabelCommand | null =
      options.scenario === "pending"
        ? {
            workspace_id: collectionWorkspace.workspace_id,
            command_id: "fictional-command",
            prepare_key: "fictional-key",
            state: "prepared",
            acknowledged: false,
            preview: firstPreview,
          }
        : null,
    dropPrepare = options.scenario === "lost_prepare";
  const port: CollectionPort = {
    async query(q) {
      calls.queries.push(structuredClone(q));
      options.onCall?.("query");
      if (options.scenario === "denied") throw new CollectionFailure("denied");
      const matching = rows.filter(
        (row) =>
          (!q.source_account_id || row.source_id === q.source_account_id) &&
          (!q.search || row.description.includes(q.search)) &&
          (q.assignment === "all" || row.assignment.state === q.assignment) &&
          (!q.starts_on || row.booked_date >= q.starts_on) &&
          (!q.through_date || row.booked_date <= q.through_date) &&
          (q.topic.kind === "all" ||
            (row.owner_topic.state === q.topic.kind &&
              (q.topic.kind !== "topic" ||
                (row.owner_topic.state === "topic" &&
                  row.owner_topic.topic_id === q.topic.topic_id)))),
      );
      const offset = q.cursor === null ? 0 : Number(q.cursor.slice(5)),
        items = matching.slice(offset, offset + q.limit);
      return {
        ...collectionPage(q, items),
        matching_count: String(matching.length),
        start_index: items.length ? String(offset + 1) : "0",
        next_cursor:
          offset + q.limit < matching.length
            ? `page:${offset + q.limit}`
            : null,
      };
    },
    async preview(request) {
      calls.previews.push(structuredClone(request));
      options.onCall?.("preview");
      latestPreview = collectionPreview(
        request,
        request.selection.map(
          (item) =>
            rows.find((row) => selectionKey(row) === selectionKey(item))!,
        ),
      );
      return structuredClone(latestPreview);
    },
    async prepare(request) {
      calls.prepares.push(structuredClone(request));
      options.onCall?.("prepare");
      if (!latestPreview) throw new CollectionFailure("unavailable");
      if (!saved)
        saved = {
          workspace_id: request.workspace_id,
          command_id: "fictional-command",
          prepare_key: request.prepare_key,
          state: "prepared",
          acknowledged: false,
          preview: structuredClone(latestPreview),
        };
      if (dropPrepare) {
        dropPrepare = false;
        throw new CollectionFailure("uncertain");
      }
      return structuredClone(saved);
    },
    async pending() {
      calls.pending++;
      options.onCall?.("pending");
      return saved && !saved.acknowledged ? structuredClone(saved) : null;
    },
    async execute() {
      calls.execute++;
      options.onCall?.("execute");
      if (!saved) throw new CollectionFailure("unavailable");
      saved = { ...saved, state: "committed" };
      return structuredClone(saved);
    },
    async cancel() {
      calls.cancel++;
      options.onCall?.("cancel");
      if (!saved) throw new CollectionFailure("unavailable");
      saved = { ...saved, state: "cancelled" };
      return structuredClone(saved);
    },
    async acknowledge() {
      calls.acknowledge++;
      options.onCall?.("acknowledge");
      if (!saved) throw new CollectionFailure("unavailable");
      saved = { ...saved, acknowledged: true };
      return structuredClone(saved);
    },
  };
  return { port, calls };
}
