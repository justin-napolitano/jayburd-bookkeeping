# Automatic private categorization —488

## Decision and scope

Owner2026-11-11 explicitly requests automatic categorization. This supersedes
243/414's confirm-every-category default for analytic labels, retaining existing
artifact-backed accounting and explicit source-post confirmation. No automatic
ledger posting, financial-purpose/tax/ownership inference, private training,
third-party model or real/provider/cloud activation is introduced.

Reuse the current Go rule matcher and private SQL authority. Automatic effects
have their own durable policy/run/receipt, never a fabricated human command or
automatic acknowledgement of interactive work. Queries remain read-only and
filters/counts use the complete persisted effective projection before pagination.

## Reviewed sources and alternatives

Reviewed source/code paths and selected official Plaid sync PFC/version/confidence
and migration sections are recorded in [source review](../reviews/automatic-categorization-source.md)
and [existing PFC record](../research/pfc-normalization-sources.md). The public
taxonomy downloaded2026-11-11 is22003bytes, SHA256
c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc.
Only231 valid same-version code pairs and18 application-authored short labels
are packaged as interoperability metadata. Vendor descriptions, CSV prose and a
merchant dictionary are not copied. One composite v1 annotation and blank pairs
are excluded. No OSS/public-domain or ISO-taxonomy claim is made.

HIGH/VERY_HIGH is our conservative automatic-use policy, not a calibrated
Jayburd accuracy score. The exact reported v1/v2 pair must be in the reviewed
map. Missing/versionless/unknown/unmapped/lower-confidence data becomes an
exception; a private rule conflict never falls through to the provider.

Alternatives rejected: client-only page labels break global counts; GET writes
make reads grant effects; auto-clicking manual commands falsifies review/ack;
guessing purpose from merchants risks invented accounting; a second matcher
duplicates ranking semantics. The API process may run the same bounded durable
classification service in its existing lifecycle, gated explicitly and without
a new database/microservice. Deployment must supply background CPU/lifecycle or
an explicitly reviewed invocation schedule; source tests alone do not prove
continuous hosted operation when the process is idle/scaled to zero.

Standards applicability: this is an internal analytic label operation, not an
ISO20022 financial exchange or ISO21378 export. Original dated ISO4217 money and
dates are unchanged. Existing source/standards records retain their access limits;
no accounting framework, normative certification or financial policy is added.

## Source attribution publication

Default decoder/runner/provider constructors and wire remain unchanged. Explicit
categorized constructors request PFCv2 and use DecodePageWithCategories both on
first acquisition and retained active-attempt resume. Financial Page is unchanged;
optional top-level category_attribution is the existing CategoryPage profile,
source_page_sha256 and exact account/transaction/change attribution array.

Forward48, sole writer root, adds category-specific stage/context/complete
functions with the same e,b,stream,set,fence selectors:

- integration.stage_sync_category_page(...,raw text) validates closed sidecar,
  exact whole added/modified identity set and source-page hash, then delegates
  financial staging. Same page replay requires exact category/body equality.
- integration.sync_category_context(...) returns the ordinary closed context
  with byte_count covering the full categorized wire. Independently enforce
  2MiB cumulative financial+sidecar,10pages/1000changes/500perpage. Old financial
  limits/fences stay. Mixed legacy/categorized staged sets fail closed.
- integration.complete_sync_categories(...) calls existing complete_sync and
  publishes retained category-attribution revisions in the SAME transaction,
  bound to the resulting observation/source revision and exact set/page/hash.
  Same completion replay cannot duplicate category revisions. Category-only
  changes advance analytic fences, never financial observation revision, source
  correction exception, journal, money/date/currency or financial changed_count.

Only restricted sync login has these worker entrypoints. API/observer/foreign
workers cannot publish category facts. Category metadata is separate from the
financial provider_fields equality. New native-provider acquisition240 remains
separate; native source facts can still be categorized by approved private rules.
No historical raw scan, source refetch, inferred version or fabricated metadata.
Malformed category objects retain the strict opt-in decoder rejection, mapped
to existing fixed invalid_field diagnostic; valid unknown/low metadata abstains.

## Policy and starter categories

The owner of a currently readable/writable private workspace controls an
immutable versioned automatic policy. Existing workspaces start disabled until
one explicit enable request. Newly created workspaces default enabled under the
new feature; application setup explains automatic analytic labels. No labels are
created by migration or by a read. Background processing requires current owner
read/classify grants and matching runtime composition, not provider fetch consent.

The first enabled run installs the finite starter catalog under a distinct
Automatic categories parent, preserving all existing owner topic IDs/names and
manual decisions. Fixed app UUIDs and the231 pair map identify18 child topics.
An ID/name collision, retired required target or taxonomy limit is an actionable
hold, never an overwrite, resurrection or truncation. Current same-edition
installation is idempotent. Source/taxonomy/map revisions remain explicit.

## Durable automatic execution

All entrypoints use restricted API sessions, no table privilege. Trusted Go obtains
candidate owner/workspace metadata from the SQL work list; it never accepts owner
identity or computed results from a public caller. SQL rechecks current active
principal, workspace capabilities and every visible source/book grant per phase.
Current-row locks reuse collection_label_authorize's existing order: principal,
legacy entity/membership/book scopes in UUID order, workspace, private item/source
and legacy-source read fences, then label state. Policy/run locks follow those.
Do not upgrade an earlier weaker lock or invent a competing order.

Any existing shared interactive reservation defers classification. Automation
never executes/cancels/acknowledges it, and never hijacks stored v1/v2/v3 previews.
Inventory rotation commits independently of later phase failure, so an uncertain workspace cannot monopolize the next ten-workspace round. SQL-only private helpers inherit the secure entrypoint search_path with fully qualified table names to allow bounded query planning; they remain unexecutable by runtime roles. All worker phases are bounded to100 facts and4seconds per transaction, known
40001/40P01 whole-transaction retries at most3; cancellation returns no partial
accepted result. The same evaluator supplies private rule results.

A private per-workspace cache (at most2MiB) stores only the SQL-created current
context, selected fingerprints and policy revision. It avoids reconstructing the
same history twice in one bounded preparation transaction. Runtime roles cannot
read or write this table; there is no public cache/results route. Prepare requires
exact cached JSON equality, current authorization/locks, matching policy and full
query snapshot pins, then existing SQL result/manual/output validation. Source,
PFC, taxonomy/tag/rule and decision changes invalidate those pins. Execution still
recomputes fresh per-row fingerprints and reauthorizes before effects. Cache
survives exact inert prepare replay and is cleared by policy changes or terminal
execution. Public GET does not populate it.

| SQL entrypoint | Frozen meaning |
| --- | --- |
| banking.get_collection_automation(actor,w) | Read-only current-authorized status; profile private-collection-automation-v1, workspace_id, revision string, enabled bool, processing_state disabled/idle/pending/blocked, pending_count/attention_count strings, blocked_reason null/saved_work/category_catalog_unavailable/permission_changed, last_run nullable bounded metadata. Unknown is not zero. |
| banking.set_collection_automation(actor,w,raw) | Exact closed {key:string,expected_revision:string,enabled:boolean}, preserving original raw bytes for duplicate/unknown-key validation; immutable config receipt keyed owner/workspace/key. Replay same body under current authority; changed body conflicts. No interactive fake command. |
| banking.collection_automation_work() | Trusted service only: private-collection-auto-work-v1 with at most10 current-authorized enabled owner/workspace metadata pairs. No raw source payload or financial effect. |
| banking.collection_automation_context(actor,w) | private-collection-auto-context-v1: principal_id/workspace_id, state ready/pending/idle/blocked, run_id nullable, context nullable existing private-rule-context-v1, last_run always present and nullable full run metadata. Pending run wins over new selection. Select at most100 input-fingerprint mismatches. Starter initialization is a worker mutation, never a query effect. |
| banking.prepare_collection_automatic_run(actor,w,context_text,results_text) | Reauthorize, require exact equality with the SQL-created context cache and current policy/snapshot pins, and validate complete trusted evaluator results; persist immutable planned input/result with deterministic work_key and one durable run_id. Return private-collection-auto-run-v1 metadata. Inert preparation, no category effects. |
| banking.execute_collection_automatic_run(actor,w,run_id) | Reauthorize and revalidate policy/source/PFC/rule/map/taxonomy/tag/manual fingerprints; atomically persist decisions/projection/audit plus committed same-run receipt. A stale plan terminates stale with zero category effects, then fresh work may be planned. Exact committed replay returns original receipt, never duplicate effects. |

Run metadata includes profile, workspace_id, run_id, state prepared/committed/stale,
processed_count and attention_count decimal strings; no user acknowledgement is
required for authorized automatic results. Unknown prepare reply is recovered
through the durable pending run. Unknown execute reply recovers/replays that SAME
run, never a new operation inferred from an empty queue. A crash after commit is
safe; receipt access still requires current owner/source authority.

Per-row input fingerprints include current observation revision, category
metadata revision, fixed policy profile/map/rule/taxonomy/tag editions and actual manual/latest
reviewed-rule decisions. They exclude enable-toggle revision (separately fenced at execution) and unrelated automatic progress so one batch
does not invalidate every completed row forever. Edition/source changes yield
bounded reevaluation. Historical decisions and financial posted anchors remain.

## Effective category and UI wire

Current manual fields win independently; stale manual choices abstain until
explicitly resolved. Approved rule outputs precede provider fallback. Conflicts
abstain. Provider fallback uses an exact HIGH/VERY_HIGH mapped pair only when no
current rule resolves it. Missing/low/unknown facts are retained as exceptions.
No financial entry, receipt, book revision or posting outbox is created.

Current v3 query/category filters consume effective owner_topic/owner_tags across
the whole authorized result. Extra categorization metadata is additive to read
rows; old stored command bodies/profiles retain their interpretation. Snapshot
source hash covers automatic policy/projection progress so stale cursors/drills
fail closed. No category filter is implemented only on a displayed page.

New row categorization has state automatic/manual/needs_attention/pending/
unavailable, origin rule/provider/manual/none, reason nullable fixed code,
rule_names array, provider nullable {name,version,primary,detailed,confidence,
map_edition}, and decision_id nullable UUID. Names are retained at the actual
winning edition; do not label historical provenance with a renamed current rule.
Display category and readable origin in closed rows. Technical IDs remain out of
the ordinary flow. Needs attention adds the finite v3 topic selector kind needs_attention, evaluated against booked persisted needs_attention rows before count/pagination; old selectors and saved previews retain their interpretation. Do not count
intentionally Uncategorized, pending or removed records as routine manual tasks.

Explicit API composition advertises catalog.automatic_categorization={state:
available}. Add authenticated GET/POST /v1/collections/{workspace}/automation;
POST body is {key,expected_revision,enabled}. Reads have no effects. Unknown
configuration outcome preserves same request/key until authoritative replay.
No public endpoint accepts computed automatic results or another actor ID.
API configuration -automatic-categories defaults false, requires private
collection composition and48 readiness. Existing worker consumer category flag
of the same name selects both category-aware provider/runner together; old
profiles/defaults stay unchanged. No feature/deployment activation follows here.

## Acceptance

[AUTO01–16 inventory](../reviews/automatic-categorization-acceptance.md) names
independent corpus, current-authority/primed-race, source/edition/manual fences,
durable unknown replies, complete3000-row filters/counts, zero journals and real
browser/service/restricted SQL requirements. Existing source-post/recovery/money
and role-isolation proofs remain. Low-confidence/unknown/unsupported examples
must not be disguised as automatic success. Engineering fictional review is not
owner approval; deployment/provider/real-data gates remain separate.
