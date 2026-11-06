# Private collection review interface —305

Root-reviewed for isolated frontend implementation2026-11-06. User design approval
lives in GitHub238/242; implementation/dependencies/status live in305/304/239/65.
This interface is a typed component port, not an implemented HTTP API or SQL grant.
No production route/nav imports the component. Registry and service integration
remain mandatory before activation; synthetic callbacks are test harnesses only.

## Reviewed read profile

`private-collection-review-v1` queries one exact workspace. No actor/email is sent;
the future trusted service resolves verified identity and rechecks current grants
before counts, rows or cursor diagnostics. Fields: workspace_id, source_account_id
nullable, assignment all/assigned/unassigned, starts_on/through_date nullable,
literal search≤100, topic filter all/unclassified/uncategorized/topic(topic_id),
sort date_desc/date_asc, snapshot nullable, opaque cursor nullable and limit25.
The closed UI port permits page limits1–100; selection contains at most100 rows.

Snapshot is server-owned authority_revision/source_snapshot/taxonomy_revision/
ruleset_revision. `source_snapshot` is64lowercase hexadecimal and binds every selected source/revision; it is
not one source revision, a JS sum, a timestamp or a capability. Response echoes
the exact workspace/selector/snapshot and returns matching_count/start_index as
canonical decimal strings, at mostlimit items and opaque next_cursor. The UI
validates that context before rendering; client validation never grants access.
An echoed account filter is insufficient alone: every row must also match a
non-null selected source-account identity before rendering or selection.

Rows use composite workspace+source_id+observation_id identity, original source
revision/date and a discriminated money projection. Available money preserves
exact amount/direction, alphabetic/nullable numeric currency identifiers, dated reference
valid_from/valid_to and minor-unit scale. Held unsupported_currency/invalid_precision
facts expose null money/direction/currency/reference values and an explicit reason;
never fabricate0, a quantity or currency for a hold. A null numeric currency code
in the existing reference table does not invalidate otherwise known dated money;
render numeric-code unavailable explicitly rather than inventing one.
Rows also expose safe source label,
nullable merchant and description. Amounts remain decimal strings; the UI neither
aggregates nor converts currency. Source status, explicit assigned/unassigned/
unavailable book projection, journal status, current owner topic and attributed
provider suggestion are separate. A denied/absent book projection never implies
that the source is unassigned. Initial registry-only adapters must use assignment
and journal unavailable until304's separate current book/journal projection is proved.
Only already-authorized safe book labels/references
may be supplied; provider IDs, credentials and raw payloads have no port field.
Removed source observations may retain prior held/null money; show Removed and
Money unavailable separately. Removal must not invent a zero or newly usable value.

## Reviewed label and recovery profile

Desired labels are discriminated `{kind:topic,topic_id}` or
`{kind:uncategorized}`. Null never conflates clear/missing/Uncategorized.
Preview takes the exact selected composite observation IDs/revisions, current
snapshot and action. Response supplies a server preview ID/expiry, exact echoed
selection/action/snapshot and count. There is no implicit all-results selection.
The UI cannot expand a reviewed selection by paging or changing a filter.

Prepare references only the reviewed preview and a stable client key in memory.
Unknown prepare results use server pending discovery and may retry the same
preview/key; never synthesize a new key/body. Commands expose current workspace,
command ID, preview, state prepared/committed/cancelled, selected_count and explicit
acknowledgement state. Pending discovery precedes enabling label work after mount.
Only explicit user actions execute/cancel/acknowledge. Unknown execute/cancel
results rediscover the same command or explicitly retry the same exact operation.
An absent pending read after an uncertain mutation is not treated as proof of a
non-effect; the stable key/intent stays locked. Lost terminal acknowledgement may
explicitly retry that acknowledgement on the same command. Command ID/key and
the complete immutable preview are compared on every transition; acknowledged
prepared commands are invalid. Terminal acknowledgement is never automatic.
Cancel/execute may return either definitive terminal state when another context
wins the race. Show that authoritative committed/cancelled result and still require
acknowledgement; do not turn a known commit into an assumed cancellation or an
unknown outcome. Acknowledgement must preserve the previously known terminal state.
Recovery of an uncertain acknowledgement must preserve that known terminal state
as well; a matching command/key/preview cannot change committed into cancelled.
Discovery/rendering posts no journal and executes no command.

All props, rows, filters, previews, keys and commands are memory-only. Scope,
authority/taxonomy or lock changes unmount the collection session. Query/filter/
page changes clear rows and selections; stale async replies are discarded.
Denied/revoked responses destroy retained private state. Stale snapshots require
an explicit refresh, and mutation uncertainty remains distinct from read failure.
Preview and restored/saved command headings receive focus in a post-commit React
effect with current request epoch/workspace checks, never in an animation frame
that could beat the heading ref. Initial idle discovery does not steal focus.

## Design and standards

Existing Jayburd system sans typography, ink#202124/muted#555b63/line#dce0e3/
paper#ffffff/accent#286044/negative#9b3e38 are retained. A single compact review
list prioritizes date, account, description and original quantity; supporting
status lines remain quieter and separate. On mobile each row becomes one readable
stack with a44px selection target, not a horizontally clipped finance table.
Account/search stay visible; native disclosure holds book/topic/date filters so
an actual row is visible on the390px first screen even with a partial-history notice.
No new font, asset, animation, dependency, global CSS or unrelated redesign.

Internal analytic query/label review is not an ISO20022 financial exchange.
ISO4217 dated currency identity/minor units are preserved; no monetary arithmetic,
FX or audit-export wire is introduced. Existing standards review limits apply;
no ISO21378 or accounting conformance claim. The source and recovery authority
comes from the approved source-first/private-categorization and command-recovery
contracts, not this frontend. STRIDE: private disclosure/stale response→context
pins, state destruction and server grants; tampering→exact explicit selection and
server preview; replay→stable prepare key and server command discovery; DoS→finite
page/selection/text bounds. SQL locking/permission/replay proof remains304/239.

## Acceptance and alternatives

Unit tests cover scope reset/stale replies, echo mismatch, exact original strings,
server paging over3000 rows, denied/stale/loading/empty/partial, explicit preview
selection/action and uncertain recovery without automatic execution/acknowledgement.
Browser checks cover390px/keyboard/no-overflow when local port binding permits.
Tests use independently authored fictional accounts/merchants/amounts only.

Alternatives rejected: client read-all/aggregate (incorrect counts and exposure),
implicit select-all-matches (unreviewed expansion), null label semantics (ambiguous
owner intent), direct fetch before304 (invented runtime authority), persistent
browser recovery (financial data exposure), automatically acknowledging terminal
commands (loss of explicit recovery boundary). Exact API/OpenAPI/SQL integration
will be root-owned after239/304 evidence; this port may evolve only through review.
