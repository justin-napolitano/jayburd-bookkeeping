# Automatic analytic categorization source review —488

Reviewed2026-11-11 at frozen consolidation basebea304ec. Scope: source tracing and
proposal only. No source/profile/schema activation, provider/private-data/model
call, credential access or runtime/cloud mutation. Read AGENTS, current capability
map/index,488 stage, bibliography, PFC/private categorization and query/rule
contracts plus their annotated source records. GitHub488/243/413 read attempts
failed network; root owns current issue authority. SQL1–47 remain immutable;
48 is unallocated. Proposed names below are not implemented interfaces.

## Finding and recommendation

The app already has a bounded deterministic matcher and durable rule-label
projection. It does not have an automatic application mode or a built-in provider
map. The PFC decoder/map are pure, uncalled preparation. Turning on existing
suggestions alone would neither persist labels nor fix whole-query counts.

Use the reviewed finite official provider taxonomy below as the default analytic starter; allow
existing account/workspace rules to refine it and preserve manual decisions.
Users should not author a merchant dictionary to get initial categories. This
is a provider-attributed analytic default, never a business-purpose, deductibility,
legal-owner or accounting-account assertion. Explicit source posting stays intact.

## Inspected integration boundaries

| Existing path | Actual behavior / reusable boundary |
| --- | --- |
| `internal/acquisition/plaid_http.go`, Fetch | Account-filtered sync; original description requested, no explicit PFC request version. API header version is not taxonomy version. |
| `internal/acquisition/runner.go`, prefix/acquire | Both use legacy `plaid.DecodePage`; acquisition marshals `pageWire` and stages the whole financial page. Retained raw bytes support the same active attempt's resume. No category sidecar caller. |
| `internal/ingestion/plaid/category.go` | Opt-in `DecodePageWithCategories` returns unchanged financial Page and bounded category sidecar keyed by account+transaction/change and raw-page SHA256. Exact response version/code/confidence or fixed unavailable reason. Malformed metadata fails enhanced decode atomically. |
| `internal/ingestion/plaid/category_map.go` | `CompileCategoryMap` accepts at most1000 explicit version/primary/detailed pairs and caller-owned map edition. `Suggest` is inert and accepts every recognized confidence level; it does NOT implement a HIGH-only policy. No shipped mapping corpus, authority or SQL persistence. |
| SQL25/29 normalized sync publication | Closed transaction fields exclude categories. Provider-field equality controls financial revision behavior. Extending financial Transaction JSON silently would break compatibility/change revision semantics. |
| SQL42 private source path | `publish_collection_batch` accepts closed normalized changes; `collection_facts` unifies native and adopted legacy evidence. Native publication is exercised in restricted tests; current Go acquisition does not call it. New automation must cover both source origins, not assume all imports use this one function. |
| `internal/collectionlabels/rules.go` | `RuleProcessor.Process` loads SQL-authorized facts within service transaction; shared `internal/categorization` handles ranking/conditions/context budgets. Query/detail suggestions are inert; preview evaluation is persisted only through reviewed command flow. No automatic worker evaluator exists. |
| SQL46 `collection_rule_decision`, `collection_query_rows` | Immutable command-linked applied decisions overlay fields without manual history; current query filters/counts consume the persisted overlay. Manual topic/tags are independently protected; stale manual state abstains whole-row. Existing decisions require a command FK and cannot honestly be reused by inventing human commands for background work. |

Statement CSV currently supplies financial/source facts, not verified PFC. Existing
retained normalized rows cannot acquire missing PFC merely by installing a map.
No legacy raw scanning or provider refetch is authorized by this proposal.

## Official public source review and access limits

Refreshed these selected official sections via web on2026-11-11; not the entire
API, no customer settings/payloads examined. Supplements existing CR04/CR07 and
[PFC source record](../research/pfc-normalization-sources.md); root owns source-index
updates before implementation.

- [Transactions API](https://plaid.com/docs/api/products/transactions/): sync
  `options.personal_finance_category_version` and added/modified PFC object fields
  primary/detailed/confidence_level/version. Nullable PFC/confidence and explicit
  v1/v2 response version are documented. Confidence has five named levels, including
  UNKNOWN; vendor HIGH/VERY_HIGH claims are not measured Jayburd accuracy.
- [Migration guide](https://plaid.com/docs/transactions/pfc-migration/): opt-in/default,
  historical taxonomy retrieval and taxonomy link sections. Defaults vary by
  enablement date; changing taxonomy does not backfill prediction improvements.
  The official full taxonomy is linked as a CSV.
- [Official taxonomy CSV](https://plaid.com/documents/pfc-taxonomy-all.csv): root
  fetched public HTTP200 bytes to task-local temporary storage after this lane's
  browser CSV/DNS limitations. Independently recomputed22,003bytes and SHA256
  `c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc`.
  Read the entire CSV, including all code/description columns and explanatory row.
  It has128 DictReader rows: one explanatory note and127 category rows, six
  v2/v1 primary/detailed/description columns. All127 v2 pairs are unique valid
  codes (18 primaries). v1 has104 unique valid pairs (16 primaries),22 empty
  pairs and one composite mapping annotation. No duplicate valid pair in either.

### Finite starter map recommendation

Use each explicitly valid same-version pair from that exact corpus, mapped to its
OWN version's primary analytic bucket; do not translate v1 to v2 using adjacent
CSV columns. This yields231 versioned pair entries, safely below the existing1000
map cap. The final OTHER_OTHER row's v1 cells are slash-separated mapping prose,
not valid provider codes; exclude that composite pseudo-pair (the actual two v1
other-transfer pairs are separately listed). Blank v1 cells are v2-only categories,
not permission to copy their v2 identifiers into v1. Retain v2 OTHER_OTHER as an
explicit broad Other category, not proof of meaningful transaction purpose.

The18 candidate starter buckets are Bank fees, Entertainment, Food and drink,
General merchandise, General services, Government and nonprofit, Home improvement,
Income, Loan disbursements, Loan payments, Medical, Other, Personal care, Rent and
utilities, Transfers in, Transfers out, Transportation and Travel. These readable
labels are proposed rendering of official primary codes, not account/tax mappings.
Use stable application topic IDs tied to the approved map edition. Keep direction
and original units separate: loan proceeds/transfers are NOT silently income or
expense, and provider “income” does not prove taxable revenue. Detailed source
attribution remains visible without requiring users to manage127 categories.

CSV descriptions are proprietary vendor documentation, not an OSS merchant corpus
or ISO taxonomy. No explicit reuse license was found in the inspected CSV/sections;
do not claim public-domain/OSS status or redistribute descriptions wholesale.
Record source URL/digest and minimal exact interoperability code pairs with
app-authored short display labels; root must resolve repository distribution
policy before packaging a copied corpus. Do not silently install over an owner's
edited taxonomy or resurrect retired IDs. Fresh workspaces can receive the approved
starter catalog; existing ones need explicit deterministic map targets/unavailable
state. This mapping is now a reviewed finite proposal, not an activated profile.

## Proposed bounded architecture for root review

1. **Pin sources and activation semantics.** Freeze the immutable starter map,
   taxonomy IDs, automatic policy edition and confidence policy. Suggested initial
   provider auto-apply: HIGH/VERY_HIGH only; lower/unknown/missing/versionless/
   unsupported/unmapped values yield explicit exceptions. This is our conservative
   product threshold, not already implemented by CategoryMap. Rules retain existing
   account-before-workspace ranking; equal-rank conflict never falls through to PFC.
2. **Add category evidence alongside financial publication.** Explicit additive
   versioned stage/publish interface carries sidecar metadata, page hash and exact
   composite identity. Persist immutable category-attribution revisions separately
   from financial source revision; bind to the resulting observation and exact
   source revision. Category-only changes advance attribution/projection fences,
   not financial revision, draft invalidation, posted anchor or journal. Missing
   metadata remains explicit, never silently assigned a request-derived version.
   Pin v2 requests only after exact additive profile/source review; keep legacy
   profile decoding supported. Choose reviewed whole-page hold for malformed
   enhanced metadata; never advance past a discarded transaction.
3. **Record durable bounded work in the source transaction.** Import/publication,
   new financial/category revision, adoption and policy/rules/map changes schedule
   source-scoped dirty work. Existing normalized history can be evaluated by rules;
   PFC-less history stays unavailable. No unbounded read-triggered backfill. A unique
   work identity includes workspace/source/observation plus relevant revision pins.
4. **One shared evaluator, restricted persistence.** Dedicated Go orchestration
   claims at most100 authorized facts and evaluates the same categorization engine,
   then falls back to explicit provider map for eligible unmatched fields. Current
   manual fields win independently; any stale manual field abstains the row. Persist
   an immutable automatic decision with origin rule/provider, exact source/category,
   taxonomy/tag/ruleset/map/policy editions and rule provenance/unavailable reason.
   A separate automatic receipt/run identity replaces the human-command FK; never
   reserve/ack a fabricated interactive command or block the user's saved work.
5. **Current authority and atomic apply.** SQL owns same-transaction current private
   classify/read/source grants, active principal, pins and immutable effect equality.
   Initially an authenticated app orchestration action can process bounded dirty
   work without per-row confirmation; background execution requires an explicit
   scoped automation grant/delegation, not widening the sync login into a classifier.
   Revoke disables new decisions immediately; old source access remains current.
   SERIALIZABLE fences protect source/manual/catalog updates and exact retries.
   Unknown commit reconciles the same work receipt, never guesses success. No
   provider, model or network call occurs inside the transaction.
6. **One effective read projection.** A direct SQL join from immutable normalized
   PFC attribution plus pinned map/policy to a category is a valid durable-input
   projection: it need not create one derived decision row per provider label.
   Unlike today's inert sidecar suggestion, this category must feed whole-result
   selection/counts before pagination. Record applicable attribution/map revisions
   in the snapshot and explanation; a map/metadata update invalidates cursors.
   Configured Go rules still require actual automatic bounded execution/persistence,
   not simulated preview/prepare/execute clicks. Extend effective rows so manual→
   ranked rules→mapped provider→unclassified/exception precedence feeds query filtering,
   counts and every page. Reads remain inert. Keep provenance and processing state
   distinct from labels; avoid claiming full categorization while dirty work remains.
   Pending/removed/held/unsupported-unit evidence never becomes booked spending.

Do not start with new regex dictionaries for arbitrary merchants, an external
model, browser learning or category-only read-time overlays. None supplies verified
purpose, durable replay or global count correctness. Auto-learned merchant rules
from a single correction would also be a new scope; keep existing explicit rule
approval unless the owner separately accepts bounded learning semantics.

## Necessary scope and acceptance before code

Root alone allocates/owns any forward migration after review. Likely scope:
additive attribution store/publication APIs, auto policy/map catalog and immutable
auto decision/work/receipt tables, current-authority claim/commit functions and
effective query/count integration. Reuse immutable source history/label/rule tables;
never alter merged1–47 or retrofit human command provenance.

Go scope: ingestion/plaid sidecar/map and corpus tests; acquisition types/runner/
repository additive profile; shared categorization orchestration; restricted auto
service/job ports. CSV/import and native/adopted source hooks must be explicit.
HTTP/server/OpenAPI need bounded processing/status/exception contracts if app-driven;
background host/IAM activation is a later separately approved boundary, not implied.
UI consumes accepted projection/status and offers corrections/exceptions, not100
mandatory starter-rule forms. Root freezes exact wire/API scope before these edits.

Required synthetic proof: fresh imported response→durable category→all-page filters
and independent counts; identical replay; category-only correction leaves financial
revision/journal bytes fixed; manual field protection and stale whole-row abstention;
account/workspace conflict; confidence/version/malformed/unknown corpus pairs; current
source/principal/grant revoke and primed snapshot races; native plus adopted/CSV
coverage; 100-fact bounded work/cancellation; unknown commit receipt recovery;
0journal/posting-outbox changes from categorization. Existing source_post explicit
confirmation, saved legacy recovery and old API versions remain unchanged.

Internal analytic labels introduce no ISO20022 message; proprietary PFC is not an
ISO taxonomy. ISO4217/exact original currency mapping remains unchanged. Existing
standards/source access limits apply; no business/tax/financial-report conformance
or real provider coverage claim follows. This proposal needs root contract review,
source distribution-policy resolution and current task authority before implementation.

## Allocated Go implementation evidence (488 follow-up)

Root approved the minimal231 factual same-version pairs and18 app-authored labels.
`internal/ingestion/plaid/category_catalog.json` is the finite authoritative seed,
with edition `plaid-pfc-primary-2026-11-11` and the reviewed public source hash.
No CSV/descriptions or merchant dictionary is copied. `StarterCategories` returns
fresh copies; `AutoEligibleCategory` expresses only HIGH/VERY_HIGH eligibility,
not map existence/current authority. Actual SQL map seed is root-owned and must
prove exact equality against this catalog independently.

Explicit `acquisition.NewWithCategories` and
`NewPlaidHTTPForProfileWithCategories` select the enhanced decoder/additive SQL
path and fixedv2 request respectively. Existing constructors and financial JSON
are unchanged. Workerruntime `Config.Categories` is consumer-only; Managed selects
both opt-in constructors together. CLI flag `-automatic-categories` is separately
owned. No provider request or runtime activation was performed by this lane.

Categorized Context uses `integration.sync_category_context`, Stage uses
`integration.stage_sync_category_page`, Complete uses
`integration.complete_sync_categories`; existing role admission, SERIALIZABLE,
retry/unknown-commit rules remain. Root48 owns current fences, full categorized
wire byte_count/caps and paired-stage equality. A legacy staged attempt must not
silently resume as a categorized attempt; explicit drain/hold belongs to release
qualification. Same active-attempt raw resume is still allowed, not a general
historical raw extraction/backfill capability.

Malformed category metadata in opt-in mode atomically rejects the whole page
before storage/staging; its fixed existing SQL diagnostic is `invalid_field`.
Missing/unknown/versionless/lower-confidence metadata stays valid attribution and
is not a reason to discard financial evidence. This strict malformed-only
availability limit is intentional, not a blanket claim of automatic coverage.

Local compatibility-modfile Go1.26.5 evidence: all Plaid decoder tests PASS;
focused category acquisition/runtime tests and selected existing SQL/profile
checks PASS under race; integration-tag acquisition/runtime compile PASS;
scoped vet and `make check build` PASS. New tests cover lost stage reply/resume,
completed replay, raw-page digest/composite identity, exact cents, malformed
atomicity versus inert legacy behavior, explicit request options, three SQL
function dispatches, cumulative wire-byte cap, constructor inertness, consumer-only
admission, corpus/version traps and confidence policy. These are synthetic unit/
transport tests, not actual restricted-SQL48 acceptance, canonical Go1.26.11 or
security/provider validation. Root owns current aggregate/canonical evidence.

Full local `make test` was attempted once:18 release guards passed and all
non-socket packages completed, but the gate is BLOCKED/FAIL because existing
acquisition, webhookhost, workerhost and workerruntime listener tests cannot bind
loopback sockets (EPERM). No host/privilege/timeout workaround or weakened test
was introduced. Focused non-listener category/race tests above passed; canonical
full-stack/restricted SQL remains required. This is not a full local test PASS.

Root subsequently allocated forward48 to the sole root SQL writer; the unallocated
statements above describe proposal-time scope, not the later implementation claim.
`internal/acquisition/category_integration_test.go` adds executable restricted-SQL
fixtures for categorized stage/complete replay, lost committed replies, immutable
financial history across category-only updates, full-wire context accounting,
legacy-mode hold, forged/malformed sidecar rollback, unavailable metadata and exact
SQL catalog equality. Integration-tag compile/vet passed locally. Execution is
pending root's migration seal; compile alone is not SQL acceptance. Read-only SQL
review identified null-profile three-valued validation and scalar source-state
projection errors; root corrected them, plus normalized malformed helper errors
to the existing fixed acquisition SQLSTATE. No migration was edited by this lane.

### Executed restricted-SQL48 evidence

Root-owned isolated `jayburd-auto48-proof-20261111-2131` supplied sealed48.
Direct Docker per-step commands succeeded after the shell runner's public pull
hit the keychain helper; no credentials/auth configuration or privilege changed.
Created only an empty `acquisition_template` clone of root's empty rules template,
test-owned `acq_*` clones, a uniquely named acquisition binary and the standard
synthetic verifier login. Template/root resources remain under root cleanup.
Cross-compiled Linux arm64 with the declared local compatibility modfile; this
is actual PostgreSQL17/restricted-login execution, not canonical Go1.26.11 proof.

All seven initial category48 groups PASS (about1.8s combined): committed stage/
completion response loss and replay; category-only correction preserving exact
financial/review rows and0.10 original input; forged sidecar rollback; unavailable/
legacy cases; malformed whole-page hold; exact SQL/Go corpus equality; explicit
stage/complete replay, full-wire context bytes and legacy-mode hold. All remaining
existing acquisition unit/integration groups also PASS in the same isolated runner,
including exact original money, precision holds, historical currencies, mutation
restart, revocation/role isolation, lifecycle and sandbox verification races.
The suites were partitioned only to avoid duplicate fixture database names; all
compiled named tests ran, and the unchanged240s timeout remained.

Added `TestCategory48PrivateProviderFallback`, PASS0.42s: real categorized
acquisition → explicit legacy adoption → actual AutomaticService inert prepare/
execute → v3 effective category/provider provenance and filtered matching_count1.
Then a real manual Uncategorized preview/prepare/execute/ack remains manual after
further automatic steps. Source financial JSON remains identical; journals,
posting outbox and review drafts stay zero. Initial test-only count assertion used
an obsolete total_count name; corrected to the actual matching_count contract
before the successful rerun. No product code/SQL change accommodated that error.
This adds the missing legacy-provider→private-auto integration proof without
claiming native feeds/CSV acquire invented PFC or real provider accuracy.
