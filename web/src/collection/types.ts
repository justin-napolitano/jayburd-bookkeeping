// Scope comes from the future authorized service; its shape grants no capability.
export type Workspace = {
  workspace_id: string;
  authority_revision: string;
  name: string;
};
// Only current authorized source/topic display projections may be passed to the component.
export type SourceChoice = { source_id: string; name: string };
export type Topic = {
  topic_id: string;
  name: string;
  parent_id: string | null;
  status: "active" | "retired";
};
export type Catalog = {
  taxonomy_revision: string;
  ruleset_revision: string;
  map_revision: string;
  topics: Topic[];
};
// Opaque server state binds every source revision selected by the query, never a JS sum.
export type Snapshot = {
  authority_revision: string;
  source_snapshot: string;
  taxonomy_revision: string;
  ruleset_revision: string;
  map_revision: string;
  decision_revision: string;
};
export type TopicFilter =
  | { kind: "all" }
  | { kind: "unclassified" }
  | { kind: "uncategorized" }
  | { kind: "topic"; topic_id: string };
export type LabelAction =
  | { kind: "topic"; topic_id: string }
  | { kind: "uncategorized" }
  | { kind: "taxonomy"; topics: Topic[] };
export type Selector = {
  source_account_id: string | null;
  assignment: "all" | "assigned" | "unassigned";
  starts_on: string | null;
  through_date: string | null;
  search: string;
  topic: TopicFilter;
  sort: "date_desc" | "date_asc";
};
export type Query = Selector & {
  profile: "private-collection-review-v1";
  workspace_id: string;
  snapshot: Snapshot | null;
  cursor: string | null;
  limit: number;
};
// Composite identity is required even when an observation UUID happens to look globally unique.
export type Selection = {
  workspace_id: string;
  source_id: string;
  observation_id: string;
  observation_revision: string;
};
// Held facts preserve an explicit unavailable projection; zero or a guessed currency is never substituted.
export type Money =
  | {
      state: "available";
      amount: string;
      direction: "inflow" | "outflow" | "zero";
      currency: string;
      currency_version_id: string;
      numeric_code: string | null;
      minor_unit: number;
      valid_from: string;
      valid_to: string | null;
    }
  | {
      state: "held";
      reason: "unsupported_currency" | "invalid_precision";
      amount: null;
      direction: null;
      currency: null;
      currency_version_id: null;
      numeric_code: null;
      minor_unit: null;
      valid_from: null;
      valid_to: null;
    };
export type Row = Selection & {
  account_name: string;
  booked_date: string;
  description: string;
  merchant: string | null;
  money: Money;
  source_state: "booked" | "pending" | "removed" | "held";
  assignment:
    | { state: "unassigned" }
    | { state: "unavailable" }
    | { state: "assigned"; books: { book_id: string; name: string }[] };
  journal_state: "unposted" | "posted" | "correction_required" | "unavailable";
  owner_topic:
    | { state: "unclassified" }
    | { state: "uncategorized" }
    | { state: "stale" }
    | { state: "topic"; topic_id: string; name: string };
  provider_suggestion:
    | { state: "unavailable" }
    | {
        state: "attributed";
        label: string;
        taxonomy_version: string;
        confidence: string;
      };
};
export type Page = {
  profile: "private-collection-review-v1";
  workspace_id: string;
  selector: Selector;
  snapshot: Snapshot;
  matching_count: string;
  start_index: string;
  items: Row[];
  next_cursor: string | null;
  partial: boolean;
};
export type PreviewRequest = {
  workspace_id: string;
  snapshot: Snapshot;
  selection: Selection[];
  action: LabelAction;
};
// Review rows let restored commands show the exact persisted intent before explicit execution.
export type Preview = PreviewRequest & {
  profile: "private-collection-preview-v1";
  preview_id: string;
  expires_at: string;
  selected_count: string;
  review_rows: Row[];
};
export type LabelCommand = {
  workspace_id: string;
  command_id: string;
  prepare_key: string;
  state: "prepared" | "committed" | "cancelled";
  acknowledged: boolean;
  preview: Preview;
};
export type CommandIdentity = { workspace_id: string; command_id: string };
// This injected port declares future service behavior; it supplies no fetch, auth bypass or store.
export type CollectionPort = {
  query: (query: Query) => Promise<Page>;
  preview: (request: PreviewRequest) => Promise<Preview>;
  prepare: (request: {
    workspace_id: string;
    preview_id: string;
    prepare_key: string;
  }) => Promise<LabelCommand>;
  pending: (workspace_id: string) => Promise<LabelCommand | null>;
  execute: (identity: CommandIdentity) => Promise<LabelCommand>;
  cancel: (identity: CommandIdentity) => Promise<LabelCommand>;
  acknowledge: (identity: CommandIdentity) => Promise<LabelCommand>;
};
// Fixed failure classes never expose raw provider, SQL or sensitive diagnostics to users.
export class CollectionFailure extends Error {
  constructor(
    public readonly kind:
      | "denied"
      | "stale"
      | "rejected"
      | "unavailable"
      | "uncertain",
  ) {
    super("Collection action unavailable");
  }
}

const revision = /^(0|[1-9][0-9]{0,18})$/;
const amount = /^(0|[1-9][0-9]{0,28})(?:\.[0-9]{1,9})?$/;
// text bounds transport fields without interpreting text as HTML, URLs or code.
function text(value: unknown, max = 500): value is string {
  return typeof value === "string" && Array.from(value).length <= max;
}
// opaque tokens are consistency/identity metadata, not verified authorization.
function opaque(value: unknown): value is string {
  return text(value, 2048) && value.length > 0;
}
function decimal(value: unknown): value is string {
  return text(value, 19) && revision.test(value);
}
function object(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}
// Same-snapshot equality is meaningful only after the server has checked current authority.
export function sameSnapshot(a: Snapshot, b: Snapshot): boolean {
  return (
    a.authority_revision === b.authority_revision &&
    a.source_snapshot === b.source_snapshot &&
    a.taxonomy_revision === b.taxonomy_revision &&
    a.ruleset_revision === b.ruleset_revision &&
    a.map_revision === b.map_revision &&
    a.decision_revision === b.decision_revision
  );
}
function validSnapshot(value: unknown): value is Snapshot {
  return (
    object(value) &&
    decimal(value.authority_revision) &&
    text(value.source_snapshot, 64) &&
    /^[a-f0-9]{64}$/.test(value.source_snapshot) &&
    decimal(value.taxonomy_revision) &&
    decimal(value.ruleset_revision) &&
    decimal(value.map_revision) &&
    decimal(value.decision_revision)
  );
}
// Explicit discriminants keep no label, explicit Uncategorized and a selected topic distinct.
export function sameAction(a: LabelAction, b: LabelAction): boolean {
  const normalize = (action: LabelAction) =>
    action.kind === "taxonomy"
      ? {
          ...action,
          topics: [...action.topics].sort((left, right) =>
            left.topic_id < right.topic_id
              ? -1
              : left.topic_id > right.topic_id
                ? 1
                : 0,
          ),
        }
      : action;
  return canonical(normalize(a)) === canonical(normalize(b));
}
// Bounded complete editions retain history; semantic hierarchy checks are repeated by SQL.
export function validTopics(value: unknown): value is Topic[] {
  if (!Array.isArray(value) || value.length > 1000) return false;
  if (
    !value.every(
      (topic) =>
        object(topic) &&
        Object.keys(topic).length === 4 &&
        opaque(topic.topic_id) &&
        text(topic.name, 80) &&
        !/^ *$/.test(topic.name) &&
        !/[\u0000-\u001f\u007f-\u009f]/u.test(topic.name) &&
        (topic.parent_id === null || opaque(topic.parent_id)) &&
        ["active", "retired"].includes(String(topic.status)),
    )
  )
    return false;
  const topics = value as Topic[];
  const byId = new Map(topics.map((topic) => [topic.topic_id, topic]));
  if (
    byId.size !== topics.length ||
    topics.filter((topic) => topic.status === "active").length > 500
  )
    return false;
  const siblings = new Set<string>();
  for (const topic of topics) {
    const sibling = JSON.stringify([topic.parent_id, topic.name]);
    if (siblings.has(sibling)) return false;
    siblings.add(sibling);
    const seen = new Set([topic.topic_id]);
    let parent = topic.parent_id;
    let depth = 1;
    while (parent !== null) {
      const ancestor = byId.get(parent);
      if (
        !ancestor ||
        seen.has(parent) ||
        ++depth > 8 ||
        (topic.status === "active" && ancestor.status !== "active")
      )
        return false;
      seen.add(parent);
      parent = ancestor.parent_id;
    }
  }
  return true;
}
function validAction(value: unknown): value is LabelAction {
  return (
    object(value) &&
    ((value.kind === "uncategorized" && Object.keys(value).length === 1) ||
      (value.kind === "topic" &&
        Object.keys(value).length === 2 &&
        opaque(value.topic_id)) ||
      (value.kind === "taxonomy" &&
        Object.keys(value).length === 2 &&
        validTopics(value.topics)))
  );
}
export function selectionKey(value: Selection): string {
  return JSON.stringify([
    value.workspace_id,
    value.source_id,
    value.observation_id,
  ]);
}
export function selectionOf(value: Row): Selection {
  return {
    workspace_id: value.workspace_id,
    source_id: value.source_id,
    observation_id: value.observation_id,
    observation_revision: value.observation_revision,
  };
}
function validSelection(value: unknown, workspace: string): value is Selection {
  return (
    object(value) &&
    value.workspace_id === workspace &&
    opaque(value.source_id) &&
    opaque(value.observation_id) &&
    decimal(value.observation_revision)
  );
}
// Equality includes source/workspace/revision and is independent of list ordering.
export function sameSelection(a: Selection[], b: Selection[]): boolean {
  const sort = (items: Selection[]) =>
    items
      .map((item) =>
        JSON.stringify([
          item.workspace_id,
          item.source_id,
          item.observation_id,
          item.observation_revision,
        ]),
      )
      .sort();
  const left = sort(a),
    right = sort(b);
  return (
    left.length === right.length &&
    left.every((value, index) => value === right[index])
  );
}
function uniqueSelection(items: Selection[], workspace: string): boolean {
  return (
    items.length <= 100 &&
    items.every((item) => validSelection(item, workspace)) &&
    new Set(items.map(selectionKey)).size === items.length
  );
}
// Canonical source calendar dates are validated without locale or monetary conversion.
function date(value: unknown): value is string {
  return (
    text(value, 10) &&
    /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/.test(value) &&
    value >= "0001-01-01" &&
    value <= "9999-12-31" &&
    Number.isFinite(Date.parse(value)) &&
    new Date(value).toISOString().slice(0, 10) === value
  );
}
// Currency intervals and precision stay attached to original quantities; unavailable holds are not numeric facts.
function validMoney(
  value: unknown,
  bookedDate: string,
  sourceState: string,
): value is Money {
  if (!object(value)) return false;
  if (value.state === "held")
    return (
      (sourceState === "held" || sourceState === "removed") &&
      ["unsupported_currency", "invalid_precision"].includes(
        String(value.reason),
      ) &&
      [
        "amount",
        "direction",
        "currency",
        "currency_version_id",
        "numeric_code",
        "minor_unit",
        "valid_from",
        "valid_to",
      ].every((key) => value[key] === null)
    );
  if (
    value.state !== "available" ||
    !text(value.amount, 40) ||
    !amount.test(value.amount) ||
    !["inflow", "outflow", "zero"].includes(String(value.direction)) ||
    !text(value.currency, 3) ||
    !/^[A-Z]{3}$/.test(value.currency) ||
    !opaque(value.currency_version_id) ||
    !(
      value.numeric_code === null ||
      (text(value.numeric_code, 3) && /^[0-9]{3}$/.test(value.numeric_code))
    ) ||
    !Number.isInteger(value.minor_unit) ||
    (value.minor_unit as number) < 0 ||
    (value.minor_unit as number) > 9 ||
    !date(value.valid_from) ||
    !(value.valid_to === null || date(value.valid_to)) ||
    value.valid_from > bookedDate ||
    (value.valid_to !== null && (value.valid_to as string) <= bookedDate)
  )
    return false;
  return (
    (value.amount.split(".")[1]?.length ?? 0) <= (value.minor_unit as number) &&
    /^0(?:\.0+)?$/.test(value.amount) === (value.direction === "zero")
  );
}
function validRow(value: unknown, workspace: string): value is Row {
  if (!validSelection(value, workspace)) return false;
  const r = value as unknown as Record<string, unknown>;
  if (
    !text(r.account_name, 200) ||
    !date(r.booked_date) ||
    !text(r.description) ||
    !(r.merchant === null || text(r.merchant)) ||
    !["booked", "pending", "removed", "held"].includes(
      String(r.source_state),
    ) ||
    !validMoney(r.money, r.booked_date, r.source_state as string)
  )
    return false;
  if (r.source_state === "held" && r.money.state !== "held") return false;
  if (
    !object(r.assignment) ||
    !["unassigned", "unavailable", "assigned"].includes(
      String(r.assignment.state),
    )
  )
    return false;
  if (
    r.assignment.state === "assigned" &&
    (!Array.isArray(r.assignment.books) ||
      r.assignment.books.length < 1 ||
      r.assignment.books.length > 25 ||
      !r.assignment.books.every(
        (b) => object(b) && opaque(b.book_id) && text(b.name, 200),
      ))
  )
    return false;
  if (
    !["unposted", "posted", "correction_required", "unavailable"].includes(
      String(r.journal_state),
    ) ||
    !object(r.owner_topic) ||
    !["unclassified", "uncategorized", "stale", "topic"].includes(
      String(r.owner_topic.state),
    )
  )
    return false;
  if (
    r.owner_topic.state === "topic" &&
    (!opaque(r.owner_topic.topic_id) || !text(r.owner_topic.name, 80))
  )
    return false;
  return (
    object(r.provider_suggestion) &&
    (r.provider_suggestion.state === "unavailable" ||
      (r.provider_suggestion.state === "attributed" &&
        text(r.provider_suggestion.label, 160) &&
        text(r.provider_suggestion.taxonomy_version, 80) &&
        text(r.provider_suggestion.confidence, 80)))
  );
}
function sameTopic(a: TopicFilter, b: TopicFilter): boolean {
  return (
    a.kind === b.kind &&
    (a.kind !== "topic" || (b.kind === "topic" && a.topic_id === b.topic_id))
  );
}
// The selector comparison intentionally ignores pagination tokens and includes every actual filter.
export function sameSelector(a: Selector, b: Selector): boolean {
  return (
    a.source_account_id === b.source_account_id &&
    a.assignment === b.assignment &&
    a.starts_on === b.starts_on &&
    a.through_date === b.through_date &&
    a.search === b.search &&
    sameTopic(a.topic, b.topic) &&
    a.sort === b.sort
  );
}
// Reject a mismatched or oversized safe projection before any private row reaches rendering.
export function validPage(
  value: unknown,
  query: Query,
  workspace: Workspace,
  catalog: Catalog,
): value is Page {
  if (
    !object(value) ||
    value.profile !== query.profile ||
    value.workspace_id !== workspace.workspace_id ||
    !object(value.selector) ||
    !object(value.selector.topic) ||
    !sameSelector(value.selector as Selector, query) ||
    !validSnapshot(value.snapshot) ||
    value.snapshot.authority_revision !== workspace.authority_revision ||
    value.snapshot.taxonomy_revision !== catalog.taxonomy_revision ||
    value.snapshot.ruleset_revision !== catalog.ruleset_revision ||
    value.snapshot.map_revision !== catalog.map_revision ||
    (query.snapshot && !sameSnapshot(value.snapshot, query.snapshot)) ||
    !decimal(value.matching_count) ||
    !decimal(value.start_index) ||
    !Array.isArray(value.items) ||
    value.items.length > query.limit ||
    !value.items.every(
      (row) =>
        validRow(row, workspace.workspace_id) &&
        (query.source_account_id === null ||
          row.source_id === query.source_account_id),
    ) ||
    !uniqueSelection(value.items as Row[], workspace.workspace_id) ||
    !(
      value.next_cursor === null ||
      (text(value.next_cursor, 4096) && value.next_cursor.length > 0)
    ) ||
    typeof value.partial !== "boolean"
  )
    return false;
  if (
    BigInt(value.matching_count) < BigInt(value.items.length) ||
    (value.items.length === 0 && value.start_index !== "0") ||
    (value.items.length > 0 && BigInt(value.start_index) < 1n)
  )
    return false;
  if (
    (value.items.length > 0 &&
      BigInt(value.start_index) + BigInt(value.items.length) - 1n >
        BigInt(value.matching_count)) ||
    (value.items.length === 0 && value.next_cursor !== null)
  )
    return false;
  return true;
}
// Restored previews must include the exact immutable selection and safe review rows.
export function validPreview(
  value: unknown,
  request: PreviewRequest,
): value is Preview {
  return (
    object(value) &&
    request.selection.length <= 100 &&
    value.profile === "private-collection-preview-v1" &&
    value.workspace_id === request.workspace_id &&
    opaque(value.preview_id) &&
    text(value.expires_at, 40) &&
    Number.isFinite(Date.parse(value.expires_at)) &&
    decimal(value.selected_count) &&
    value.selected_count === String(request.selection.length) &&
    (request.action.kind === "taxonomy"
      ? request.selection.length === 0
      : request.selection.length >= 1) &&
    validSnapshot(value.snapshot) &&
    sameSnapshot(value.snapshot, request.snapshot) &&
    validAction(value.action) &&
    sameAction(value.action, request.action) &&
    Array.isArray(value.selection) &&
    uniqueSelection(value.selection, request.workspace_id) &&
    sameSelection(value.selection, request.selection) &&
    Array.isArray(value.review_rows) &&
    value.review_rows.every((row) => validRow(row, request.workspace_id)) &&
    uniqueSelection(value.review_rows, request.workspace_id) &&
    sameSelection(value.review_rows, request.selection)
  );
}
// Recovery retains historic immutable edition pins; the service rechecks current authority before disclosure.
export function validCommand(
  value: unknown,
  workspace: Workspace,
  _catalog: Catalog,
): value is LabelCommand {
  if (
    !object(value) ||
    value.workspace_id !== workspace.workspace_id ||
    !opaque(value.command_id) ||
    !opaque(value.prepare_key) ||
    !["prepared", "committed", "cancelled"].includes(String(value.state)) ||
    typeof value.acknowledged !== "boolean" ||
    (value.state === "prepared" && value.acknowledged) ||
    !object(value.preview) ||
    !validSnapshot(value.preview.snapshot) ||
    !Array.isArray(value.preview.selection) ||
    !validAction(value.preview.action)
  )
    return false;
  return validPreview(value.preview, {
    workspace_id: workspace.workspace_id,
    snapshot: value.preview.snapshot,
    selection: value.preview.selection as Selection[],
    action: value.preview.action,
  });
}
// Stable structural equality rejects changed saved review values without depending on JSON key order.
function canonical(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(canonical).join(",")}]`;
  if (object(value))
    return `{${Object.keys(value)
      .sort()
      .map((key) => `${JSON.stringify(key)}:${canonical(value[key])}`)
      .join(",")}}`;
  return JSON.stringify(value) ?? "null";
}
export function samePreview(a: Preview, b: Preview): boolean {
  const rows = (value: Preview) =>
    [...value.review_rows]
      .sort((left, right) =>
        selectionKey(left) < selectionKey(right)
          ? -1
          : selectionKey(left) > selectionKey(right)
            ? 1
            : 0,
      )
      .map(canonical);
  return (
    a.preview_id === b.preview_id &&
    a.expires_at === b.expires_at &&
    a.selected_count === b.selected_count &&
    a.workspace_id === b.workspace_id &&
    sameSnapshot(a.snapshot, b.snapshot) &&
    canonical(a.action) === canonical(b.action) &&
    sameSelection(a.selection, b.selection) &&
    canonical(rows(a)) === canonical(rows(b))
  );
}
