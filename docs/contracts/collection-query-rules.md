# Broader collection queries and private rules — 242/414

## Frozen query45 and proposed rule46 ownership

2026-11-07 query45 frozen by coordinator after merged ca1b37e schema inspection.
Approved product design remains GitHub238/242; GitHub65 owns live scheduling.
Forward45 is allocated solely to query/detail; forward46 rule storage is reserved
to the same sole SQL writer. The rule46 wire below remains proposed until freeze.
Migrations1–44 remain immutable. No owner activation, source adoption, provider,
cloud, consent, journal or financial recognition change follows from this work.

## Additive query and detail

Add v3 catalog/query and a separate source-detail POST, keeping existing
v1/v2 shapes and command recovery intact. Snapshot initially retains v2 fields:
authority_revision, source_snapshot, taxonomy_revision, tag_revision,
ruleset_revision, map_revision and decision_revision. Revisions are decimal
strings and source_snapshot is lowercase SHA256 hex. Every page/cursor pins the
complete selector and snapshot. All filters precede count/rank/limit; stable
source/observation UUID tie-breaks retain one row per observation.

Existing account/date/description/topic/tag selectors remain. Proposed additions:

| Selector | Closed meaning |
| --- | --- |
| merchant | all, missing, exact literal or contains literal; no wildcard/regex/fuzzy identity; at most100 codepoints |
| amount | null or one dated currency_version_id with inclusive nullable minimum/maximum nonnegative decimal magnitude and all/inflow/outflow/zero direction; held values never interpreted or compared |
| institution | Exactly {kind:all} or {kind:unavailable}; no provenance-bearing institution metadata exists, and display aliases or Item IDs cannot replace it. |
| legal_owner | Exactly {kind:all} or {kind:unknown}; future known-owner filtering requires the separate effective-dated reviewed assertion, never principal custody or book entity. |
| book | all, unassigned, unavailable, or exact entity_id/book_id; exact targets require current independent book read, without revealing denied names/IDs/counts |
| status | independent source and journal enums; unavailable journal metadata is distinct from unposted |
| sort | closed field/order across the full matching set; original amount grouping uses ISO code and dated version before numeric magnitude, never FX-equivalent comparison |

Currency bounds validate canonical decimal grammar/length, ledger numeric limits
and dated minor-unit precision before numeric casts. No JS/Go float, rounding,
conversion, sum or unknown-currency fallback. Source-projected dates retain their
meaning. Native unassigned sources and incomplete-history flags remain visible.
Complete authorized catalogs must not truncate into apparent completeness.

Fixed functions are get_collection_taxonomy_v3(uuid,uuid),
query_collection_review_v3(uuid,uuid,text) and get_collection_detail(uuid,uuid,text),
all banking SECURITY DEFINER entrypoints granted only to the restricted API role.
Reads require repeatable read or serializable isolation and current authorization.
HTTP retains existing bounds/deadlines/errors: query/detail8192 bytes and result1MiB.
V2 manual commands and every v1/v2 closed response/snapshot remain unchanged.

Catalog profile private-collection-catalog-v3 retains v2 fields and adds complete
books:[{entity_id,book_id,name,entity_name}] for current-authorized books associated
with visible legacy sources (at most100), and currencies:[{currency_version_id,
code,numeric_code,minor_unit,valid_from,valid_to}] for distinct available-money
versions observed in current visible facts (at most1000). Numeric code and
exclusive valid_to can be null. It adds institution:{state:unavailable} and
legal_owner:{state:unknown}; every row adds exactly those same two objects.
No compatibility book entity is called an asserted legal owner.

Query retains all v2 keys, changes profile to private-collection-review-v3 and
sort to {field,order}, and adds required merchant,amount,institution,legal_owner,
book,status keys. Merchant objects use kind all/missing, or exact/contains plus
value1–100 Unicode codepoints. Matching is C-locale case-folded literal; missing
includes null or empty merchant. Amount is null or {currency_version_id,minimum,
maximum,direction}; bounds are null or strings matching
`^(0|[1-9][0-9]{0,28})([.][0-9]{1,9})?$`, at most39 characters, less than1e29,
and nonzero fractional precision no greater than the dated minor unit. Trailing
zeros are accepted without rounding. Direction is all/inflow/outflow/zero.
Book objects use kind all/unassigned/unavailable, or book with entity_id/book_id.
Existing assignment remains a separate intersecting all/assigned/unassigned filter.
Status is {source:all|booked|pending|removed|held,
journal:all|unposted|posted|correction_required|unavailable}.
Sort fields are date/account/description/merchant/amount/currency/book/source_state/
journal_state/topic/tags/institution/legal_owner; order asc/desc. Unknown sort
values remain last in either direction. Currency and amount group ISO code then
dated version ID, with exact magnitude only within that version; amount sorting
does not require an amount filter. All ties use source_id then observation_id.

Detail input contains profile:private-collection-detail-v1, snapshot, source_id, observation_id and
observation_revision; workspace is a separately verified path selector. The
result is {profile:private-collection-detail-v1,workspace_id,snapshot,row,links}.
Links is {state:available|unassigned|unavailable,journals:[...]}. Authorized anchors contain
exact entity_id/book_id/journal_id and linked source revision. They are anchors
for the existing freshly authorized journal endpoint, not copied journal rows.
Do not expose raw provider payloads, tokens, unsupported evidence extraction or
book metadata after its current authorization is lost. Stale source/snapshot
rejects rather than silently jumping to another revision.
Anchor source_revision is a positive decimal historical linked revision, not
necessarily the current corrected source revision. At most100 unique anchors;
available can be empty for a readable unposted legacy row, unassigned denotes
native, and unavailable hides every denied legacy anchor/name. Each requires a
freshly authorized read at the existing journal endpoint before disclosure.

## Versioned private rule contract proposal

Reuse internal/categorization's bounded case-sensitive Go matcher as the only
regex/ranking implementation. SQL selects/locks current authorized facts and
taxonomy/rule editions; Go compiles and evaluates inside that transaction. SQL
owns current grants, exact source/selector/result identity, immutable state,
atomic effects and replay. HTTP never accepts evaluator results from a caller.
The application matcher is explicitly trusted for match/rank correctness, not
misrepresented as a SQL-verified computation; SQL still rejects foreign outputs,
selection drift, stale pins and protected manual-field replacement.

Propose complete immutable rule editions, initially revision0 empty. At most100
active rules and1000 retained stable IDs, finite UTF8 name, terminal retirement,
local nullable account scope, integer priority,1–8 closed conjunctive conditions,
one active topic or explicit Uncategorized and0–20 active tag IDs. All existing
pattern/regex-program/source-string/batch bounds remain. No text cleanup mutates
source evidence. Account scope precedes workspace, then priority/specificity;
equal-winning-rank disagreement abstains with stable provenance.

Proposed new durable actions are rule_catalog (full replacement, empty selection)
and rules_apply (1–100 explicit composite observation/revision identities).
Preview returns immutable per-row result/provenance and exact apply/abstain
counts, not a promise to expand a selector at execution. Source/taxonomy/tag/
ruleset/map/decision pins, current capabilities and expiry are checked at preview,
prepare and execute. Execute writes the stored eligible results atomically;
replay cannot re-evaluate changed rules or append duplicate decisions.

Current manual topic and strategic tags live in separate stores. Preserve each
current manual field independently. A stale manual source/taxonomy/tag decision
abstains for that observation until explicit manual confirmation, without rule
fallback. Preserve immutable manual/rule provenance and original posted journal
anchors. Rule application never writes journals, outbox posting intent or raw
source facts; canceled and conflict-only work has zero analytic effects.

Future matches may display an inert page-bounded suggestion after current SQL
authorization; reads/acquisition do not accept it. Query counts and persisted
topic/tag filters use stored decisions, not client-side evaluation. A query
service can evaluate at most100 picked rows within the same RR transaction.
Explicit selected preview/apply remains the only historical rule effect path.
PFC fallback stays unavailable with map_revision0 until its separate reviewed
source-normalization/map integration; no inferred provider taxonomy.

V3 durable commands reuse collection_label_preview/command, the exact shared
principal reservation, immutable result, explicit terminal acknowledgement and
payload-free inaccessible recovery. V3 recovery accepts historical v1/v2/v3
preview unions; earlier profiles fail closed on unsupported saved work. Current
grants are checked before payload/replay; stale preview pins do not block an
otherwise authorized cancel/acknowledge. Unknown commit remains503, never an
automatic retry; known40001/40P01 whole-transaction retries remain capped at3.

## Sources, standards, alternatives and acceptance

Reviewed2026-11-07: PostgreSQL17 isolation13.2.2–3, locks13.3.2–4 and
numeric8.1.2 relevant sections; current Go regexp overview was go1.27.3, so
existing CR08 pinned1.26.5 review plus tracked1.26.10 CI remains the implementation
evidence. URLs: https://www.postgresql.org/docs/17/transaction-iso.html,
https://www.postgresql.org/docs/17/explicit-locking.html,
https://www.postgresql.org/docs/17/datatype-numeric.html,
https://pkg.go.dev/regexp. Existing categorization-rules-sources.md owns CR01–08;
no code/license or universal merchant dictionary is imported.

Facts: exact numeric arithmetic and locking snapshots are available mechanisms;
the matcher is bounded Go code. Inference: strict typed selectors, explicit
current-row fences and stored reviewed results suit these product constraints.
Alternatives rejected: client-side whole-set filtering, inferred ownership,
SQL/JS duplicate regex semantics, direct rule writes, or read-time automatic apply.

ISO applicability follows standards.md: internal private analytic query/rule
commands are not ISO20022 financial exchanges or ISO21378 audit exports. Original
dated ISO4217 references and ISO calendar dates remain exact. No FX, valuation,
accounting-purpose or normative conformance claim is introduced.

Threats: cross-owner disclosure/elevation requires verified actor plus current
source/book grants and guarded metadata counts; tampering requires closed raw
JSON, current pins, exact selected results and immutable history; stale execution
requires real revocation/source/ruleset races; DoS requires finite regex/input/
catalog/result bounds and existing4-second context; unknown outcome requires
durable same-identity recovery and cannot be resolved from absence.

Named proofs must cover TestAllTransactionsScope, TestUnassignedVisibility,
TestCrossPrincipalDeny, global3000-row filter/sort/count/paging, exact dated
currency/holds, independently authorized detail/journal anchors and primed
revocation. Rule proofs add actual TestRulePreviewApplyEquality,
TestRuleVersionConflict, TestOverridePreserved, TestRuleRevisionReplay,
TestPostedTagChangeNoJournalMutation, conflict abstention, unknown commit,
cross-engine reservation races and real browser/API/SQL recovery. Pure matcher
unit results are not storage/authority acceptance. Concrete source review and
canonical required gates precede merge; live rollout remains separately scoped.
