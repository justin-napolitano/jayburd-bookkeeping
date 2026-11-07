import {
  canonicalRules,
  validRuleCatalog,
  validRuleSuggestion,
  validRuleApplication,
  type RuleDefinition,
  type RuleResult,
  type RuleSuggestion,
  type RuleApplication,
} from "./rules";
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
export type Tag = {
  tag_id: string;
  name: string;
  status: "active" | "retired";
};
export type BookChoice = {
  entity_id: string;
  book_id: string;
  name: string;
  entity_name: string;
};
export type CurrencyChoice = {
  currency_version_id: string;
  code: string;
  numeric_code: string | null;
  minor_unit: number;
  valid_from: string;
  valid_to: string | null;
};
export type Catalog = {
  rules: RuleDefinition[];
  books: BookChoice[];
  currencies: CurrencyChoice[];
  institution: { state: "unavailable" };
  legal_owner: { state: "unknown" };
  taxonomy_revision: string;
  ruleset_revision: string;
  map_revision: string;
  topics: Topic[];
  tag_revision: string;
  tags: Tag[];
};
// Opaque server state binds every source revision selected by the query, never a JS sum.
export type Snapshot = {
  authority_revision: string;
  source_snapshot: string;
  taxonomy_revision: string;
  ruleset_revision: string;
  map_revision: string;
  decision_revision: string;
  tag_revision: string;
};
export type LegacySnapshot = Omit<Snapshot, "tag_revision">;
export type TopicFilter =
  | { kind: "all" }
  | { kind: "unclassified" }
  | { kind: "uncategorized" }
  | { kind: "topic"; topic_id: string };
export type LabelAction =
  | { kind: "rule_catalog"; rules: RuleDefinition[] }
  | { kind: "rules_apply" }
  | { kind: "topic"; topic_id: string }
  | { kind: "uncategorized" }
  | { kind: "taxonomy"; topics: Topic[] }
  | { kind: "tag_catalog"; tags: Tag[] }
  | {
      kind: "tags";
      operation: "add" | "remove" | "replace";
      tag_ids: string[];
    };
export type TagFilter =
  | { kind: "all" }
  | { kind: "untagged" }
  | { kind: "tag"; tag_id: string };
export type MerchantFilter =
  | { kind: "all" | "missing" }
  | { kind: "exact" | "contains"; value: string };
export type AmountFilter = null | {
  currency_version_id: string;
  minimum: string | null;
  maximum: string | null;
  direction: "all" | "inflow" | "outflow" | "zero";
};
export type BookFilter =
  | { kind: "all" | "unassigned" | "unavailable" }
  | { kind: "book"; entity_id: string; book_id: string };
export type SortField =
  | "date"
  | "account"
  | "description"
  | "merchant"
  | "amount"
  | "currency"
  | "book"
  | "source_state"
  | "journal_state"
  | "topic"
  | "tags"
  | "institution"
  | "legal_owner";
export type Selector = {
  merchant: MerchantFilter;
  amount: AmountFilter;
  institution: { kind: "all" | "unavailable" };
  legal_owner: { kind: "all" | "unknown" };
  book: BookFilter;
  status: {
    source: "all" | "booked" | "pending" | "removed" | "held";
    journal:
      | "all"
      | "unposted"
      | "posted"
      | "correction_required"
      | "unavailable";
  };
  source_account_id: string | null;
  assignment: "all" | "assigned" | "unassigned";
  starts_on: string | null;
  through_date: string | null;
  search: string;
  topic: TopicFilter;
  tag: TagFilter;
  sort: { field: SortField; order: "asc" | "desc" };
};
export type Query = Selector & {
  profile: "private-collection-review-v3";
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
  owner_tags: { state: "unclassified" | "current" | "stale"; tags: Tag[] };
  provider_suggestion:
    | { state: "unavailable" }
    | {
        state: "attributed";
        label: string;
        taxonomy_version: string;
        confidence: string;
      };
};
export type LegacyRow = Omit<Row, "owner_tags">;
export type ReviewRow = Row & {
  institution: { state: "unavailable" };
  legal_owner: { state: "unknown" };
  rule_suggestion: RuleSuggestion;
  rule_application: RuleApplication;
};
export type DetailRequest = Selection & {
  profile: "private-collection-detail-v1";
  snapshot: Snapshot;
};
export type JournalAnchor = {
  entity_id: string;
  book_id: string;
  journal_id: string;
  source_revision: string;
};
export type SourceDetail = {
  profile: "private-collection-detail-v1";
  workspace_id: string;
  snapshot: Snapshot;
  row: ReviewRow;
  links: {
    state: "available" | "unassigned" | "unavailable";
    journals: JournalAnchor[];
  };
};
export type Page = {
  profile: "private-collection-review-v3";
  workspace_id: string;
  selector: Selector;
  snapshot: Snapshot;
  matching_count: string;
  start_index: string;
  items: ReviewRow[];
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
export type ManualLabelAction = Exclude<
  LabelAction,
  { kind: "rule_catalog" | "rules_apply" }
>;
export type V2Preview = Omit<PreviewRequest, "action"> & {
  action: ManualLabelAction;
  profile: "private-collection-preview-v2";
  preview_id: string;
  expires_at: string;
  selected_count: string;
  review_rows: Row[];
  tag_results: TagResult[];
};
export type Preview = PreviewRequest & {
  profile: "private-collection-preview-v3";
  preview_id: string;
  expires_at: string;
  selected_count: string;
  review_rows: Row[];
  tag_results: TagResult[];
  rule_results: RuleResult[];
  apply_count: string;
  abstain_count: string;
};
export type TagResult = Selection & { tags: Tag[] };
export type LegacyPreview = Omit<
  V2Preview,
  "profile" | "snapshot" | "review_rows" | "tag_results" | "action"
> & {
  profile: "private-collection-preview-v1";
  snapshot: LegacySnapshot;
  review_rows: LegacyRow[];
  action: Exclude<ManualLabelAction, { kind: "tags" | "tag_catalog" }>;
};
export type StoredPreview = Preview | V2Preview | LegacyPreview;
export type LabelCommand = {
  workspace_id: string;
  command_id: string;
  prepare_key: string;
  state: "prepared" | "committed" | "cancelled";
  acknowledged: boolean;
  preview: StoredPreview;
};
export type CommandIdentity = { workspace_id: string; command_id: string };
// This injected port declares future service behavior; it supplies no fetch, auth bypass or store.
export type CollectionPort = {
  query: (query: Query) => Promise<Page>;
  detail: (request: DetailRequest) => Promise<SourceDetail>;
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
export function sameSnapshot(
  a: Snapshot | LegacySnapshot,
  b: Snapshot | LegacySnapshot,
): boolean {
  return (
    a.authority_revision === b.authority_revision &&
    a.source_snapshot === b.source_snapshot &&
    a.taxonomy_revision === b.taxonomy_revision &&
    a.ruleset_revision === b.ruleset_revision &&
    a.map_revision === b.map_revision &&
    a.decision_revision === b.decision_revision &&
    ("tag_revision" in a
      ? "tag_revision" in b && a.tag_revision === b.tag_revision
      : !("tag_revision" in b))
  );
}
function validSnapshot(value: unknown, legacy = false): value is Snapshot {
  return (
    object(value) &&
    decimal(value.authority_revision) &&
    text(value.source_snapshot, 64) &&
    /^[a-f0-9]{64}$/.test(value.source_snapshot) &&
    decimal(value.taxonomy_revision) &&
    decimal(value.ruleset_revision) &&
    decimal(value.map_revision) &&
    decimal(value.decision_revision) &&
    (legacy
      ? !Object.hasOwn(value, "tag_revision")
      : decimal(value.tag_revision))
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
      : action.kind === "rule_catalog"
        ? { ...action, rules: canonicalRules(action.rules) }
        : action.kind === "tag_catalog"
          ? {
              ...action,
              tags: [...action.tags].sort((a, b) =>
                a.tag_id.localeCompare(b.tag_id),
              ),
            }
          : action.kind === "tags"
            ? { ...action, tag_ids: [...action.tag_ids].sort() }
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
const tagID = (value: unknown): value is string =>
  typeof value === "string" &&
  /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(value) &&
  value !== "00000000-0000-0000-0000-000000000000";
// Catalog and result limits are separate; result sets cannot silently expand past25 identities.
function validTagList(value: unknown, maximum: number): value is Tag[] {
  return (
    Array.isArray(value) &&
    value.length <= maximum &&
    value.every(
      (tag) =>
        object(tag) &&
        Object.keys(tag).length === 3 &&
        tagID(tag.tag_id) &&
        text(tag.name, 80) &&
        !/^ *$/.test(tag.name) &&
        !/[\u0000-\u001f\u007f-\u009f]/u.test(tag.name) &&
        ["active", "retired"].includes(String(tag.status)),
    ) &&
    new Set(value.map((tag) => tag.tag_id)).size === value.length &&
    new Set(value.map((tag) => tag.name)).size === value.length
  );
}
export function validTags(value: unknown): value is Tag[] {
  return (
    validTagList(value, 1000) &&
    value.filter((tag) => tag.status === "active").length <= 500
  );
}
function validAction(
  value: unknown,
  version: 1 | 2 | 3 = 3,
): value is LabelAction {
  return (
    object(value) &&
    ((version === 3 &&
      value.kind === "rule_catalog" &&
      Object.keys(value).length === 2 &&
      validRuleCatalog(value.rules)) ||
      (version === 3 &&
        value.kind === "rules_apply" &&
        Object.keys(value).length === 1) ||
      (value.kind === "uncategorized" && Object.keys(value).length === 1) ||
      (value.kind === "topic" &&
        Object.keys(value).length === 2 &&
        opaque(value.topic_id)) ||
      (value.kind === "taxonomy" &&
        Object.keys(value).length === 2 &&
        validTopics(value.topics)) ||
      (version >= 2 &&
        value.kind === "tag_catalog" &&
        Object.keys(value).length === 2 &&
        validTags(value.tags)) ||
      (version >= 2 &&
        value.kind === "tags" &&
        Object.keys(value).length === 3 &&
        ["add", "remove", "replace"].includes(String(value.operation)) &&
        Array.isArray(value.tag_ids) &&
        value.tag_ids.length <= 25 &&
        (value.operation === "replace" || value.tag_ids.length >= 1) &&
        value.tag_ids.every(tagID) &&
        new Set(value.tag_ids).size === value.tag_ids.length))
  );
}
export function selectionKey(value: Selection): string {
  return JSON.stringify([
    value.workspace_id,
    value.source_id,
    value.observation_id,
  ]);
}
export function selectionOf(value: Row | LegacyRow): Selection {
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
function validRow(
  value: unknown,
  workspace: string,
  legacy = false,
): value is Row {
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
  if (
    legacy
      ? Object.hasOwn(r, "owner_tags")
      : !object(r.owner_tags) ||
        !["unclassified", "current", "stale"].includes(
          String(r.owner_tags.state),
        ) ||
        !validTagList(r.owner_tags.tags, 25) ||
        (r.owner_tags.state === "unclassified" &&
          r.owner_tags.tags.length !== 0)
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
    a.tag.kind === b.tag.kind &&
    (a.tag.kind !== "tag" ||
      (b.tag.kind === "tag" && a.tag.tag_id === b.tag.tag_id)) &&
    canonical(a.merchant) === canonical(b.merchant) &&
    canonical(a.amount) === canonical(b.amount) &&
    canonical(a.institution) === canonical(b.institution) &&
    canonical(a.legal_owner) === canonical(b.legal_owner) &&
    canonical(a.book) === canonical(b.book) &&
    canonical(a.status) === canonical(b.status) &&
    canonical(a.sort) === canonical(b.sort)
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
    !object(value.selector.tag) ||
    !sameSelector(value.selector as Selector, query) ||
    !validSnapshot(value.snapshot) ||
    value.snapshot.authority_revision !== workspace.authority_revision ||
    value.snapshot.taxonomy_revision !== catalog.taxonomy_revision ||
    value.snapshot.ruleset_revision !== catalog.ruleset_revision ||
    value.snapshot.map_revision !== catalog.map_revision ||
    value.snapshot.tag_revision !== catalog.tag_revision ||
    (query.snapshot && !sameSnapshot(value.snapshot, query.snapshot)) ||
    !decimal(value.matching_count) ||
    !decimal(value.start_index) ||
    !Array.isArray(value.items) ||
    value.items.length > query.limit ||
    !value.items.every(
      (row) =>
        validReviewRow(row, workspace.workspace_id) &&
        row.rule_suggestion.rules.every(
          (rule) => rule.rule_revision === catalog.ruleset_revision,
        ) &&
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
function validStoredPreview(
  value: unknown,
  request:
    | PreviewRequest
    | Pick<LegacyPreview, "workspace_id" | "snapshot" | "selection" | "action">,
  version: 1 | 2 | 3 = 3,
): value is StoredPreview {
  const legacy = version === 1;
  return (
    object(value) &&
    request.selection.length <= 100 &&
    value.profile === `private-collection-preview-v${version}` &&
    value.workspace_id === request.workspace_id &&
    opaque(value.preview_id) &&
    text(value.expires_at, 40) &&
    Number.isFinite(Date.parse(value.expires_at)) &&
    decimal(value.selected_count) &&
    value.selected_count === String(request.selection.length) &&
    (["taxonomy", "tag_catalog", "rule_catalog"].includes(request.action.kind)
      ? request.selection.length === 0
      : request.selection.length >= 1) &&
    validSnapshot(value.snapshot, legacy) &&
    sameSnapshot(value.snapshot, request.snapshot) &&
    validAction(value.action, version) &&
    sameAction(value.action, request.action) &&
    Array.isArray(value.selection) &&
    uniqueSelection(value.selection, request.workspace_id) &&
    sameSelection(value.selection, request.selection) &&
    Array.isArray(value.review_rows) &&
    value.review_rows.every(
      (row) =>
        validRow(row, request.workspace_id, legacy) &&
        ![
          "institution",
          "legal_owner",
          "rule_suggestion",
          "rule_application",
        ].some((key) => Object.hasOwn(row, key)),
    ) &&
    uniqueSelection(value.review_rows, request.workspace_id) &&
    sameSelection(value.review_rows, request.selection) &&
    (legacy
      ? !Object.hasOwn(value, "tag_results")
      : validTagResults(value as unknown as Preview | V2Preview)) &&
    (version === 3
      ? validRuleResults(value as unknown as Preview)
      : !Object.hasOwn(value, "rule_results") &&
        !Object.hasOwn(value, "apply_count") &&
        !Object.hasOwn(value, "abstain_count"))
  );
}
// SQL supplies exact resulting sets; validate their bounded identities and agreement with reviewed set intent.
function validTagResults(preview: Preview | V2Preview): boolean {
  if (!Array.isArray(preview.tag_results)) return false;
  if (preview.action.kind !== "tags") return preview.tag_results.length === 0;
  if (
    !preview.tag_results.every(
      (result) =>
        object(result) &&
        Object.keys(result).length === 5 &&
        validSelection(result, preview.workspace_id) &&
        validTagList(result.tags, 25),
    ) ||
    !uniqueSelection(preview.tag_results, preview.workspace_id) ||
    !sameSelection(preview.tag_results, preview.selection)
  )
    return false;
  const keys = preview.tag_results.map(selectionKey);
  if (canonical(keys) !== canonical([...keys].sort())) return false;
  const action = preview.action;
  return preview.tag_results.every((result) => {
    const row = preview.review_rows.find(
      (row) => selectionKey(row) === selectionKey(result),
    );
    if (
      !row ||
      (row.owner_tags.state === "stale" && action.operation !== "replace")
    )
      return false;
    const previous = row.owner_tags.tags.map((tag) => tag.tag_id);
    const expected =
      action.operation === "replace"
        ? action.tag_ids
        : action.operation === "add"
          ? [...new Set([...previous, ...action.tag_ids])]
          : previous.filter((id) => !action.tag_ids.includes(id));
    const actual = result.tags.map((tag) => tag.tag_id);
    return (
      canonical(actual) === canonical([...actual].sort()) &&
      canonical(actual) === canonical([...expected].sort()) &&
      (action.operation === "remove" ||
        result.tags.every(
          (tag) =>
            !action.tag_ids.includes(tag.tag_id) || tag.status === "active",
        ))
    );
  });
}
export function validPreview(
  value: unknown,
  request: PreviewRequest,
): value is Preview {
  return validStoredPreview(value, request);
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
    !object(value.preview)
  )
    return false;
  const version =
    value.preview.profile === "private-collection-preview-v1"
      ? 1
      : value.preview.profile === "private-collection-preview-v2"
        ? 2
        : value.preview.profile === "private-collection-preview-v3"
          ? 3
          : null;
  if (
    !version ||
    !validSnapshot(value.preview.snapshot, version === 1) ||
    !Array.isArray(value.preview.selection) ||
    !validAction(value.preview.action, version)
  )
    return false;
  return validStoredPreview(
    value.preview,
    {
      workspace_id: workspace.workspace_id,
      snapshot: value.preview.snapshot,
      selection: value.preview.selection as Selection[],
      action: value.preview.action,
    },
    version,
  );
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
export function samePreview(a: StoredPreview, b: StoredPreview): boolean {
  const rows = (value: StoredPreview) =>
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
    a.profile === b.profile &&
    a.preview_id === b.preview_id &&
    a.expires_at === b.expires_at &&
    a.selected_count === b.selected_count &&
    a.workspace_id === b.workspace_id &&
    sameSnapshot(a.snapshot, b.snapshot) &&
    canonical(a.action) === canonical(b.action) &&
    sameSelection(a.selection, b.selection) &&
    canonical(rows(a)) === canonical(rows(b)) &&
    (a.profile === "private-collection-preview-v3"
      ? b.profile === "private-collection-preview-v3" &&
        canonical(a.tag_results) === canonical(b.tag_results) &&
        canonical(a.rule_results) === canonical(b.rule_results) &&
        a.apply_count === b.apply_count &&
        a.abstain_count === b.abstain_count
      : a.profile === "private-collection-preview-v2"
        ? b.profile === "private-collection-preview-v2" &&
          canonical(a.tag_results) === canonical(b.tag_results)
        : b.profile === "private-collection-preview-v1")
  );
}

// New read metadata never fabricates an institution or legal owner from account/book text.
export function validReviewRow(
  value: unknown,
  workspace: string,
): value is ReviewRow {
  return (
    validRow(value, workspace) &&
    object((value as ReviewRow).institution) &&
    (value as ReviewRow).institution.state === "unavailable" &&
    object((value as ReviewRow).legal_owner) &&
    (value as ReviewRow).legal_owner.state === "unknown" &&
    validRuleSuggestion((value as ReviewRow).rule_suggestion, true) &&
    validRuleApplication((value as ReviewRow).rule_application)
  );
}
// Echo every immutable selector/pin and keep historical journal anchors distinct from current source revisions.
export function validSourceDetail(
  value: unknown,
  request: DetailRequest,
): value is SourceDetail {
  if (
    !object(value) ||
    value.profile !== request.profile ||
    value.workspace_id !== request.workspace_id ||
    !validSnapshot(value.snapshot) ||
    !sameSnapshot(value.snapshot, request.snapshot) ||
    !validReviewRow(value.row, request.workspace_id) ||
    !value.row.rule_suggestion.rules.every(
      (rule) => rule.rule_revision === request.snapshot.ruleset_revision,
    ) ||
    !sameSelection([value.row], [request]) ||
    !object(value.links) ||
    !["available", "unassigned", "unavailable"].includes(
      String(value.links.state),
    ) ||
    !Array.isArray(value.links.journals) ||
    value.links.journals.length > 100 ||
    (value.links.state !== "available" && value.links.journals.length !== 0)
  )
    return false;
  if (
    value.links.state === "available"
      ? value.row.assignment.state !== "assigned"
      : value.links.state === "unassigned"
        ? value.row.assignment.state !== "unassigned"
        : value.row.assignment.state !== "unavailable"
  )
    return false;
  if (
    (value.links.state === "unavailable") !==
    (value.row.journal_state === "unavailable")
  )
    return false;
  const detailRow = value.row;
  const ids = new Set<string>();
  return value.links.journals.every((anchor) => {
    if (
      !object(anchor) ||
      !tagID(anchor.entity_id) ||
      !tagID(anchor.book_id) ||
      !tagID(anchor.journal_id) ||
      !decimal(anchor.source_revision) ||
      anchor.source_revision === "0" ||
      detailRow.assignment.state !== "assigned" ||
      !detailRow.assignment.books.some(
        (book) => book.book_id === anchor.book_id,
      )
    )
      return false;
    const key = JSON.stringify([
      anchor.entity_id,
      anchor.book_id,
      anchor.journal_id,
    ]);
    if (ids.has(key)) return false;
    ids.add(key);
    return true;
  });
}

// Preview counts are checked only against the exact selected rows, never inferred from a current query page.
function validRuleResults(preview: Preview): boolean {
  if (
    !Array.isArray(preview.rule_results) ||
    !decimal(preview.apply_count) ||
    !decimal(preview.abstain_count)
  )
    return false;
  if (preview.action.kind !== "rules_apply")
    return (
      preview.rule_results.length === 0 &&
      preview.apply_count === "0" &&
      preview.abstain_count === "0"
    );
  if (
    !preview.rule_results.every(
      (result) =>
        validSelection(result, preview.workspace_id) &&
        validRuleSuggestion(result) &&
        result.rules.every(
          (rule) => rule.rule_revision === preview.snapshot.ruleset_revision,
        ),
    ) ||
    !uniqueSelection(preview.rule_results, preview.workspace_id) ||
    !sameSelection(preview.rule_results, preview.selection)
  )
    return false;
  const applies = BigInt(
    preview.rule_results.filter((result) => result.will_apply).length,
  );
  return (
    BigInt(preview.apply_count) === applies &&
    BigInt(preview.abstain_count) === BigInt(preview.selection.length) - applies
  );
}
