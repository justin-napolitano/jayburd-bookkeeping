# Private categorization design —433

## Decision status and purpose

Owner design accepted2026-11-06 in
[GitHub238](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/238)
and [242](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/242).
GitHub owns the approval and task state. Concrete source/security review and
implementation evidence remain required. This is a design contract for413/414/242,
not a runtime endpoint, migration, provider request or accounting policy. Existing
239→304→305 implementation gates remain. One-book delivery is desirable, but a frontend
filter or book membership cannot substitute for the proposed private authority.

Purpose: make imported activity understandable using attributed provider topics,
owner corrections and explainable rules, while preserving original evidence.
Keep three independent dimensions: analytic topic, economic spending treatment,
and accounting/journal state. A restaurant label alone proves neither business
purpose nor a deductible expense; a payment label proves no principal/interest
split. This contract never drafts, posts or corrects a journal.

Review owners: maintainer for product behavior; identity/database owners for
principal-private grants, scoped keys, concurrency and adoption; security reviewer
for restricted reads and commands. Accounting owner is needed only for later
recognition/cross-entity policies, not to choose a personal analytic topic.

## Evidence and existing-path investigation

The [source register](../research/categorization-rules-sources.md) records exact
public sections and depth. Facts versus project choices:

| Evidence | Established fact | Consequence proposed here |
| --- | --- | --- |
| Plaid Transactions sync options and PFC response fields, CR04/07 | Taxonomy versions and confidence are provider metadata; version defaults depend on enablement date. | Carry response version explicitly; never infer historical version from deployment date or the API header. |
| Plaid migration guide, CR04/07 | Historical taxonomy backfill requires provider retrieval; version conversion does not retroactively improve the old prediction's confidence. | No automatic refetch or claim that old normalized rows contain PFC. |
| Actual rules stages/conditions/management, CR01 | Supports condition-based changes, specificity and preview; its conflict behavior differs from this proposal. | Reuse concepts only; equal-rank disagreement abstains. No copied code or runtime dependency. |
| Go regexp/syntax, CR08 | Standard implementation provides a bounded-language alternative to backtracking engines. | Go owns rule matching; SQL owns authorized selection, revision checks and durable effects. |
| Existing identity/source contracts | Verified identity and current scoped grants are required; private collection authority is still proposed. | No current read/post grant implicitly becomes a private taxonomy grant. |

Inspected tracked application source only, at base49ca8c4:

1. `internal/acquisition/plaid_http.go` builds sync options with account filter
   and original description. It does not pin a PFC taxonomy request option.
2. `internal/ingestion/plaid/decoder.go:transaction` allows the outer
   `personal_finance_category` key, but does not project/validate its nested
   values. `types.go:Transaction` has no PFC field. Raw bytes are private evidence.
3. `internal/acquisition/runner.go` decodes then JSON-marshals the normalized
   `pageWire`. `Transaction.Raw` has `json:"-"`; the normalized wire loses PFC.
4. Merged25/29 SQL validates a closed normalized transaction shape and stores it
   as `provider_fields`. Merely adding a Go field would break that SQL contract.
   The existing provider-fields equality also controls observation revision replay.
5. Unmerged41 projects only named normalized `merchant_name`, not raw PFC. The
   source-activity API explicitly returns categorized_spending unavailable.

Thus historical normalized PFC availability is **unproved and absent from the
current dedicated projection**. No private payload, database row, secret, bank
statement or provider endpoint was inspected. Full raw evidence is not a backdoor
for this feature. Any historical extraction/backfill needs a separately reviewed
source-read, retention, migration and consent strategy under413.

413 must introduce an additive versioned normalized profile, matching SQL
validation and compatibility proof before activation; never edit merged SQL.
Old profile remains supported and reports unavailable. New PFC-only revisions
preserve original amounts/dates/journal anchors. A new nullable field must not
manufacture an observed financial change or duplicate review/posting; its exact
revision equality rule is a required implementation-contract decision.

## Authority, identities and taxonomy

Collection workspace is owned by an enrolled verified issuer+subject principal
under238. A caller-supplied principal/workspace/source/book ID grants nothing.
No private topic or rule is shared with another book member by default. A worker
or replay must reauthorize the same current principal/workspace/source capability.
New source read/classify capabilities require239's restricted SQL contract; they
are not available merely because a principal may post in a book.

| Operation | Required current authority and pins | Effect allowed |
| --- | --- | --- |
| List categories/suggestions | Private workspace read plus current read on every source observation | Bounded safe projection only; no mutation/audit write merely from reading |
| Edit taxonomy/rule | Private classify capability, expected taxonomy/ruleset revision | New immutable version and audited current pointer; no source/journal update |
| Preview a rule/past label selection | Current source read/classify, exact source revision and current taxonomy/ruleset | Inert matches/conflicts with full selector and count; no auto-apply |
| Apply reviewed overrides | Same current authority and all preview pins, bounded explicit selection | Atomic analytic decisions, reviewed durable command and terminal acknowledgement |
| Revoke/disconnect |239 lifecycle contract, not a category command | Deny subsequent reads/replays; no cleanup/deletion policy invented by this proposal |

Logical keys include workspace and source identity everywhere, including
observation UUID/revision. UUID appearance is not a global uniqueness proof.
After239 adoption, legacy entity/book keys stay in compatibility references;
no topic edit rehomes a source observation, book, Item or journal. A different
principal cannot inherit private rules from the same merchant/provider account.
Sharing/export, legal-owner assertions and cross-entity treatments stay deferred.

Taxonomy v1 proposal: stable opaque topic IDs within one private workspace;
immutable edition revisions, display name, optional parent and active/retired
state. Maximum500 active topics, depth8, label80 Unicode codepoints; reject cycles,
foreign parents and duplicate sibling names under the explicit case-sensitive
profile. No rename changes topic identity. Retired topics remain in historical
decisions but cannot receive new assignments. A new taxonomy edition invalidates
pending previews. Strategic tags are separate optional many-to-many labels;
the primary analytic category is single-valued to avoid double-counted totals.

An explicit Uncategorized choice is an owner decision, distinct from missing
metadata and abstention. The initial seed is an editable template, not a universal
or ISO taxonomy.413 must pin the approved provider→topic map edition; unknown or
new provider codes have no invented mapping. No public merchant dictionary is
treated as proof of an individual's transaction purpose.

## Provider attribution and correction

Safe normalized projection: origin `plaid`, source observation/revision,
response taxonomy version, primary/detailed code and confidence enum or explicit
unavailable reason. API2020-09-14 is not the PFC taxonomy version. Preserve known
provider values as attribution; a recognized confidence word is not calibrated
Jayburd accuracy. Never turn it into an unsourced numeric probability.

Known mapping requires a supported version plus an approved primary/detailed
pair. Missing/null/versionless/unknown-version/unknown-code/confidence cases
abstain with a fixed reason; malformed type/oversize behavior must preserve the
complete financial acquisition or hold it atomically under the reviewed profile,
never skip one transaction while advancing a cursor. Do not silently relabel
legacy rows v1 or v2. No icons, URLs or remote images are loaded from metadata.

Provider correction creates new attributed evidence. An explicit owner override
is preserved in audit and is never overwritten by PFC or a rule. If its pinned
source revision changes, show stale/needs confirmation and exclude it from
reviewed spending treatment until revalidated. A removed observation contributes
no current spending; its original override and posted journal remain historical.
Pending data may show an attributed suggestion, but is excluded from booked
spending. Pending→booked lineage does not silently copy an accepted override onto
an unrelated new identity. Explicit lineage/review contract governs that choice.

## Deterministic rule proposal

Precedence: current explicit transaction override; approved account-scoped rules;
approved workspace-wide rules; supported mapped PFC suggestion; unclassified.
Within each rule scope, explicit owner-reviewed priority wins first; equal
priority ranks exact merchant above merchant contains above description contains
above regex. More conditions do not imply more truth. Equal winning rank with
different topic/tag outputs **abstains**, returns every conflicting rule/version,
and does not fall through to a lower-ranked PFC or arbitrary ID winner. Equal
rank agreeing outputs return one suggestion with all provenance IDs. Stable IDs
order explanations only, never settle a semantic conflict.

All conditions are conjunctive. Initial fields: original exact merchant label,
original description, source account, source direction, inclusive source date
range and exact currency-version ID. Text operators: exact, literal contains,
Go regexp. Literal matching is case-sensitive; regex inline flags are explicit
reviewed pattern syntax. No hidden normalization, inferred merchant
identity, hidden payee rewrite or locale dependence in v1. Actual-style cleanup
is a future separately versioned alias stage, not silent source modification.
Rules propose a topic/tag only; no amount/date/source/account/journal mutation,
FX, web request, script/template evaluation or economic allocation action.

Engineering work bounds proposed:100 active rules/workspace,8 conditions/rule,
pattern128 UTF8 bytes, matching input500 codepoints/2000 UTF8 bytes; no unbounded
read-all evaluation. Compile Go regexp once per immutable rule version, reject
unsupported syntax and compiled instruction count over1024. At most100 explicit
observations per synchronous preview/apply; server-side pagination never expands
an accepted selection. Exhaustion/error returns unavailable, not no matches.
Go evaluator is shared by API and future jobs; no JS/PostgreSQL regex duplicate.
No external regex/model library is required. Implementation must prove bounds
with adversarial patterns and the actual pinned Go version, not a timing slogan.

Approved rules may yield **inert future suggestions** on new evidence. Opening
a screen never persists a label or creates a rule. After an owner correction,
offer Save rule with visible scope/conditions/output/priority, preview/conflicts
and explicit approval. Applying to past records is a separate reviewed selection;
no all-history overwrite, no automatic majority-vote learning. An accepted manual
override remains higher priority, even when an owner later approves a broad rule.

## Preview, apply, recovery and audit

Preview pins workspace/principal authority revision, exact source/filter/range,
source observation revisions, taxonomy/ruleset/map versions and intended label
action. Response includes exact matching count, each selected ID/revision and
result/provenance, explicit conflicts/skips, and a short-lived server-owned preview
reference. Page cursors retain those pins and reauthorize. A digest is equality
metadata, not a grant. No private labels, filters or cursors in URLs/logs/storage.

Apply requires an explicit selected-ID set≤100, matching preview and reviewed
durable command per [command recovery](command-recovery.md). SQL locks current
authority/version pointers in239/304's reviewed order, compares exact pins and
selection, and atomically writes override decision history plus command receipt.
Go supplies a deterministic proposed result, not caller-authorized SQL effects.
No partially applied bulk selection. Source/grant/rule/taxonomy changes conflict
and require a new preview; an old preview cannot authorize a new result.

Idempotency is principal+operation+immutable command identity/body. Equal replay
reauthorizes and returns the same receipt; mismatched payload conflicts. Lost
commit/acknowledgement recovers through server discovery, with explicit terminal
acknowledgement before another financial action. Reversal of a label change is a
new audited analytic decision, not erasure of history. Retagging a posted item
never changes its journal, posting receipt or original source revision.

Initial delivery adds no asynchronous consumer. Future background suggestions
need business/outbox intent in the same transaction and execution-time grants;
this document does not create that lane. Retention/deletion must reuse the
reviewed private-source policy under238/239, including audit/posted-anchor holds.
No legal duration, user-sharing grant or credential cleanup is inferred here.

## Spending and opportunities depend on separate evidence

[Spending metrics](spending-metrics.md) remains the recognition authority proposal.
A topic is not an economic treatment. Positive purchase/fee/interest allocation,
linked refund, transfer/card repayment exclusion, loan principal and unallocated
quantity have independent reviewed annotation/allocation revisions. SQL enforces
exact original-unit allocation equality and aggregates complete matching sets.
The220 payment with20 interest drills to20 contribution alongside220 original;
the200 principal is excluded, not lost. Never sum the entire source payment as
interest because its topic says Loans. Pending/removed/held and unresolved purpose
remain explicit; bank balance and import freshness still need their own projection.

418 Spending needs exact category/contribution/merchant drills and comparable
explicit periods; a category chart must disclose suggestion versus reviewed basis
and incomplete recognition.419 recurrence/fees/rising-category candidates must
exclude non-spending contributions, show the matched evidence and abstain for
insufficient/comparability gaps. Recurrence proves neither unused service nor
guaranteed savings. Scenarios are owner-selected exact reductions, never a bank
action, cancellation or promised result. No feature reads a new provider product.

## Standards, threats, alternatives and acceptance

Purpose/direction: private internal analytical reads and reviewed label commands
between verified browser, Go and PostgreSQL. Local profile `private-topic-v1`
is a proposed internal identifier, not an ISO message. No appropriate ISO20022
exchange is introduced by topic/rule editing; Plaid PFC remains proprietary,
with explicit version/provenance mapping above. ISO4217 applies to untouched
original currency/version/minor-unit references; no FX or monetary rounding.
ISO21378:2019 export field mapping is unresolved and **out of this non-export
slice**; no audit-export conformance claim. Reuse the exact public overview/FAQ
review limits in [standards](standards.md), not unread normative sections.
STD-01–STD-05 evidence maps to version/unknown fixtures, original-unit preservation,
correction/replay and compatibility cases below. No conformance from naming.

STRIDE: high disclosure/elevation→current principal-private source grants before
query/count/preview/replay; high tampering→immutable versions, exact scope/pins,
atomic apply and no journal authority; medium denial of service→bounded RE2-style
Go matching, batch/page/cursor limits and existing deadlines; high repudiation→
audited decisions/receipts and durable recovery; high stale-suggestion confusion→
separate attribution/review/treatment statuses and explicit invalidation.
No tokens/raw provider fields/browser financial persistence or sensitive logging.
Human authority/source/security review is required before integrated deployment.

Alternatives: provider-only is faster but cannot preserve personalized overrides;
last-rule-wins is simpler but silently resolves ambiguity; global merchant rules
leak preferences and confuse purpose; model-first adds data-use/evaluation costs;
one-book private tables before authority review hard-code the wrong ownership.
Choose explicit private scope, conflict abstention and no model. These are
proposed product/engineering choices, not facts established by the research.

[Fixed cases](../reviews/classification-433.md) specify independent expected
results and [delivery gates](../plans/classification-delivery.md) map the remaining
issues. Named tests there are **planned**, not passing runtime evidence. This
design was accepted in238/242. Registry42/source reads merged447; label43 and
connected UI implementation/evidence are in449. Provider/rule runtime413/414
and managed activation retain their separate gates; no deployment follows from
these source proofs.
