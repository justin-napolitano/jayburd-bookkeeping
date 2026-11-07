# Private transaction labels —304/305

## Decision and boundary

Implementation profile frozen2026-11-07 by coordinator and SQL/API/UI lanes,
following accepted238/242 design and owner instruction to merge447 and build
the persistent backend/existing UI. GitHub304/305/65 own task state. Main460bad4
contains private authority42. This slice adds43, restricted SQL, verified HTTP
and actual browser composition; it does not authorize a hosted rollout.

Private analytic labels never alter original source facts, accounting category,
assignments, journals or spending recognition. Source corrections leave prior
decisions immutable and visibly stale. Provider/rule/map execution is separate:
ruleset/map revisions are explicitly0 until those stores are implemented.
No financial records or tokens enter browser persistence, URLs or logs.

## Frozen wire

Base `/v1/collections/{workspace}`; UUID paths are canonical nonzero UUIDs.
Existing GET `/v1/collection` returns42's private-source-workspace-v1; map
workspace_revision to UI authority_revision. No implicit owner provisioning.
Explicit404 means absent/unavailable capability; other discovery failures keep
the shared action gate closed until retry. Existing managed feature flag remains
default false; when enabled readiness requires matching43 functions/grants.

| HTTP suffix | SQL banking function | Isolation/body cap |
| --- | --- | --- |
| GET /taxonomy | get_collection_taxonomy(actor,w) | RR/no body |
| POST /review/query | query_collection_review(actor,w,raw) | RR/8192 bytes |
| POST /labels/preview | preview_collection_labels(actor,w,raw) | SERIALIZABLE/524288 bytes |
| POST /label-commands | prepare_collection_label_command(actor,w,raw) | SERIALIZABLE/4096 bytes |
| GET /label-commands/pending | pending_collection_label_command(actor,w) | RR/no body |
| POST /label-commands/{command}/{operation} | transition_collection_label_command(actor,w,id,operation,raw) | SERIALIZABLE/1024 bytes |

Operation is execute/cancel/acknowledge. Query is the existing private review
Query excluding workspace_id: profile, source_account_id, assignment,
starts_on, through_date, search, topic, sort, snapshot, cursor, limit.
Preview body is snapshot,selection,action; selection identities include workspace.
Prepare is exactly preview_id,prepare_key; transition is exactly an empty object.
SQL owns duplicate/unknown-key and business validation; Go preserves raw bytes.
No query strings, encodings or Idempotency-Key headers. Actor is independently
verified, never supplied by the request body. Responses are bounded1MiB JSON.

Catalog profile private-collection-catalog-v1 contains workspace_id,
authority_revision,taxonomy_revision,ruleset_revision,map_revision,topics,sources.
Sources are complete≤100 safe source_id/name pairs. Topics are complete≤1000
topic_id/name/parent_id/status objects,≤500 active; no truncation masquerades as
complete. Names≤80 Unicode codepoints; case-sensitive sibling uniqueness,
depth≤8, no cycles/foreign parents. Retain all existing IDs, retire instead of
delete, terminal retirement, active children require active parents. Initial
immutable edition1 is empty, not an invented universal classification.

Page profile private-collection-review-v1 retains the existing UI row and exact
server matching_count/start_index/selector/partial/next_cursor fields. Snapshot
contains authority_revision,source_snapshot,taxonomy_revision,ruleset_revision,
map_revision,decision_revision. Revisions are decimal strings; source_snapshot
is64 lowercase SHA256 hex characters. Source/assignment/book availability changes
and current decision versions invalidate relevant pages/previews. Money stays
original decimal strings with dated ISO4217 metadata or explicit nullable holds.
Unavailable assignment is distinct from unassigned; book/journal disclosures
require their own current book authorization in addition to private authority.

Preview profile private-collection-preview-v1 contains workspace_id,snapshot,
selection,action,preview_id,expires_at,selected_count,review_rows. Labels choose
topic or explicit uncategorized for1–100 exact composite observations/revisions.
Taxonomy action is kind:taxonomy with a complete topics array; its selection and
review_rows are empty and selected_count is0. It still needs explicit preview,
durable prepare, execute/cancel and terminal acknowledgement.

Prepare/pending/transitions return private-collection-command-v1 with
workspace_id and command. Only pending permits command:null. A command retains
command_id,workspace_id,prepare_key,state,acknowledged and its immutable preview;
state is prepared/committed/cancelled. At most one unacknowledged command per
workspace. Equal prepare-key replay returns the same command; changed intent
conflicts. Execute checks current authority, all pins and expiry before atomic
writes. Current-authorized pending/cancel/ack and committed replay preserve the
historical preview even when its source/taxonomy pins become stale; they must
not strand recovery merely because the command changed the taxonomy itself.
Stored book metadata cannot be returned after its disclosure authority is lost.

## Concurrency, recovery and boundaries

SQL precreates state for all existing workspaces and initializes future42
workspace inserts. Current projection fences also cover source-inventory inserts
and legacy grant/review changes, so a primed RR snapshot cannot silently retain
older authorization/projections after waiting. Lock order: current principal,
legacy entity/membership/book scope sorted, private workspace,42 source/Item
locks, then label state. Additive projection-fence triggers must follow compatible
source→state ordering. Prove actual concurrent schedules, not only static checks.

Reads use RR, durable preview/commands use SERIALIZABLE. Four-second request and
remaining statement/lock deadlines bound service calls. Retry only known
40001/40P01 transaction aborts,≤3 attempts; unknown commit is503 without an
automatic retry. Commit acknowledgement precedes returning a result. Existing
P2800–P2805 fixed RFC9457 errors are reused, without private diagnostics.

An always-mounted browser coordinator discovers pending private work even when
the collection view is inactive. Pending/uncertain collection commands gate book
actions/navigation; book recovery gates collection actions. Sign-out clears only
memory; remount discovers persistent work. Catalog refresh after explicit ack
must not replace a saved command's immutable review values.

## Evidence, standards, alternatives and threats

Research begins with bibliography.md/database-foundations.md and accepted
private-categorization.md. Reviewed2026-11-07: PostgreSQL17 sections13.2.2–3
([isolation](https://www.postgresql.org/docs/17/transaction-iso.html)) and13.3.2–4
([locks](https://www.postgresql.org/docs/17/explicit-locking.html)); RFC9457
sections3/5 ([problems](https://www.rfc-editor.org/rfc/rfc9457.html)). Relevant
sections reviewed, not whole manuals. Facts: RR snapshots and conflicting row
locks can require transaction retry; diagnostic detail can disclose internals.
Inference: explicit fences, lock ordering and fixed errors suit this workflow.

ISO applicability follows standards.md/private-categorization.md: this is an
internal analytic command, not an ISO20022 exchange or an ISO21378 export.
Original dated ISO4217 currency metadata is preserved without valuation/rounding.
No new ISO designation/conformance claim. No accounting-policy decision follows
from an owner choosing a topic. Strategic tags, rules/maps and recognition stay
separate stores/integration work, with no fabricated runtime revision.

Alternatives: reuse book-bound ledger commands (wrong private scope/result),
direct labels POST without durable review (uncertain replay), browser fake store
(no authority/persistence), read-all client paging (incorrect counts/exposure).
Choose separate private command persistence and exact bounded server queries.

STRIDE: high disclosure/elevation→verified actor/current private and independent
book grants before payloads/counts/replay; high tampering→exact revision pins and
atomic immutable decisions; high repudiation→audited commands/decisions/receipts;
medium DoS→finite input/page/topic/selection/deadline bounds; high stale intent→
explicit conflict/new preview, terminal recovery and acknowledgement. Existing
source/privacy contracts own retention; no deletion/expiry policy is invented.

Acceptance: real restricted-role principal/mixed-role/helper/DML denials,
3000-row count/pagination/filter/sort, topic cycles/retirement/revision conflicts,
bulk rollback/stale source, current authority and primed-RR races, exact replay,
lost responses/cancel races and unchanged journal/source history. Actual
browser→Next→Go→SQL recovery proves the connected UI; mocks alone do not.
