# Automatic categorization acceptance inventory —488

## Decision and evidence status

Owner2026-11-11 supersedes suggestion-only/confirm-every-label as the default for
routine **private analytic categorization**. Reuse existing rules/matcher/source
lineage. Current owner corrections win; unambiguous private rules precede a
supported explicitly versioned provider fallback. Unknown/conflicting/stale or
unsupported facts become explainable exceptions. No universal merchant/business
purpose/tax/ownership inference, private training or external model calls.

Accounting still requires retained artifact→review/category treatment→explicit
source_post confirmation→linked immutable entry. Automatic posting334 is not
authorized by automatic analytic categories; optional owner clarification is still
pending. Documents/matching476, evidence-linked corrections333, source FX/splits,
bank reconciliation and hosted identity/provider qualification remain separate.

This inventory is proposed acceptance, not implemented API/SQL or a passing test
receipt. Root owns authority/execution/schema/API design; no migration number,
SQL signature, role, schedule, wire profile or managed activation is allocated
here. Consolidation487/bea304ec remains a frozen unmerged prerequisite; externally
queued CI is not green. Every new outcome needs actual restricted SQL/service/
browser proof on the final exact head. No data/provider/cloud read was performed.

Grounding: [automatic stage](../runbooks/automatic-categorization.md),
[private design](../contracts/private-categorization.md),
[query/rules](../contracts/collection-query-rules.md),
[matcher](../contracts/rule-evaluator.md),
[PFC preparation](../contracts/pfc-normalization.md),
[command recovery](../contracts/command-recovery.md),
[artifact policy](../contracts/artifact-backed-accounting.md),
[standards](../contracts/standards.md),
[independent corpus](../../tests/corpus/README.md) and
[existing synthetic matrix](synthetic-application-matrix.md).
Existing source registers record selected review depth/limits. No new vendor,
ISO designation or accounting-standard conformity is asserted by this inventory.

## Acceptance matrix

AUTO identifiers are proposed proof obligations. Test names under “New proof” are
suggested integration-test groups, not claims that functions or cases exist yet.
Fixed expected decisions/counts must be authored independently of the evaluator,
provider decoder, generator or returned preview; never generate the oracle by
calling the implementation twice.

| ID / property | New proof required | Independent expected outcome |
| --- | --- | --- |
| AUTO01 Durable automatic effect | `TestAutomaticCategoryPublication` through actual ingest/publication and approved execution boundary | A fictional eligible fact receives a durable analytic decision without an owner confirming each row. Refresh/remount returns it. Import, GET/catalog/query and the UI do not secretly execute a manual command or auto-click confirmation. Explicitly prove where execution is triggered; an inert suggestion labeled Automatic is insufficient. |
| AUTO02 Current authority | `TestAutomaticCategoryAuthority` using actual restricted runtime/worker roles | Current active verified owner/private classify authority and every exact source capability remain required. Book membership, import grant, caller UUID/source alias or worker identity alone grants no private category authority. Foreign/disabled/revoked/read-only owners receive no write or private disclosure; helpers/tables remain inaccessible. Freeze explicit delegation/enablement identity and revocation semantics before DDL. |
| AUTO03 Owner precedence | `TestAutomaticCategoryManualPrecedence` | Manual topic only is preserved while unprotected tags follow the reviewed fieldwise contract; manual tags only are preserved while an eligible unprotected topic may change. Both-manual and explicit Uncategorized are owner decisions, not missing data. Automatic reruns/rule edits/provider fallback must not silently replace them. Manual stale-on-source-correction behavior must be explicit and visible, never inferred away. |
| AUTO04 Rule precedence/conflict | `TestAutomaticCategoryRulePrecedence` using the existing Go matcher | Distinct priority/specificity and source-scoped/global cases produce hand-fixed winners. Equal-rank full-output disagreement abstains under existing conservative rule semantics, even when a differing field is protected. Identical rerun yields no extra decision. Do not introduce a second SQL/JS matcher or change tie policy merely to reduce exceptions. |
| AUTO05 Provider fallback | `TestAutomaticCategoryProviderFallback` | Only approved observed taxonomy/version, exact primary/detailed pair, confidence semantics and finite map edition can supply a fallback when private rules do not decide. Omitted/null/versionless/unknown version/code/confidence/map produce fixed distinct abstention reasons. A confidence enum is attribution, not a calibrated percentage. No retrospective refetch/raw-payload extraction or invented metadata for old rows. |
| AUTO06 Source identity/correction | `TestAutomaticCategorySourceFences` | Decision provenance binds workspace/source/observation/current revision plus exact source inventory snapshot. A correction/removed/pending/unsupported-money fact cannot receive stale prepared automatic output. Preserve old decision/source/journal anchors. Current eligible revised source may be reevaluated according to frozen policy; historical source100/journal100 is not rewritten when current source becomes15/removed. |
| AUTO07 Rule/map/taxonomy revisions | `TestAutomaticCategoryEditionFences` | Every automatic result retains exact rule, taxonomy, map and relevant manual-decision revisions. Rename changes display metadata, not identity. Retired/foreign output topics/tags cannot acquire new assignments; unknown IDs fail closed. Rule/map changes have a bounded explicit reevaluation policy, not silent historical mutation or read-time effect. |
| AUTO08 Races and atomicity | `TestAutomaticCategoryAtomicRaces` with observed waiting transactions | Primed snapshots versus revoke/source correction/manual override/ruleset edit cannot commit stale output. Two workers/batches for one identity do not duplicate decisions. Failure injected after any decision/pointer/audit write rolls back the whole contracted atomic unit. No partial count/result announced as complete. Define lock order and generation/claim fences before implementation. |
| AUTO09 Durable retries/unknown outcomes | `TestAutomaticCategoryUnknownCommit` | Lose the actual committed reply; recover the same execution identity/input/result under current authority. Known whole-transaction abort may use bounded retries, unknown commit must not mint a replacement batch/key or issue blind effects. Expired/stale lease completion cannot overwrite a newer generation. Durable progress survives process loss; in-memory product state is insufficient. |
| AUTO10 Global filter/count correctness | `TestAutomaticCategoryWholeResult` | More than one page with independent exact category/uncategorized/exception/provenance counts; filter/search/date/source/topic/tag/sort/count/pagination apply to the entire authorized inventory. No browser filtering only the visible page or counting suggestion as persisted. Changed selector/snapshot/current source or category revision rejects stale cursor/drill consistently. Unknown/unavailable is not zero. |
| AUTO11 Exceptions and explainability | `TestAutomaticCategoryExceptions` plus real component/browser proof | Finite readable reasons for no match, conflict, missing/versionless provider evidence, retired mapping, held source and stale owner choice. Show actual retained rule/provider/map provenance and allowed next step. Correct a merchant pattern through the existing reviewed rule authority without rewriting unrelated/manual decisions. No invented merchant/purpose labels, raw payload/ID inspector, or normal-flow debug inventory. |
| AUTO12 Pending manual work compatibility | `TestAutomaticCategoryRecoveryCompatibility` | Background categorization must not strand or implicitly execute/ack/cancel existing private/book/onboarding commands. Preserve current shared interactive gate and exact saved v1/v2/v3 previews/receipts/recovery IDs. Determine explicitly whether automation is deferred or invalidates a prepared preview; terminal replay/current disclosure and payload-free old receipt/new-claim isolation remain. Never auto-ack to unblock a worker. |
| AUTO13 Bounded work/cancellation | `TestAutomaticCategoryBounds` | Validate caps before effects; one existing matcher with deadline checks, bounded regex/compiled rules/fact batches and response size. Prove resume/remaining-work semantics on3000 independent source rows, exact no-missing/no-duplicate identities, worst admitted rules and cancelled context. Record measured workload/hardware/timing; do not promise universal accuracy, throughput or an SLA. Do not increase timeouts to conceal quadratic selection/evaluation. |
| AUTO14 Zero accounting effects | `TestAutomaticCategoryNoJournal` on every success/replay/failure path | Exact before/after ledger entries/lines/source-post links/posting receipts/book revisions/posting outbox unchanged; source money/dates/currency/Item/cursor/history anchors unchanged. Analytic decision/audit changes only as contracted. Source review/draft/post remains a distinct explicit action; account-add297/298 is not inferred from a label. |
| AUTO15 Privacy and transport | `TestAutomaticCategoryClosedTransport` and actual Handler→service→SQL browser flow | Trusted actor separate from closed raw input; unknown/duplicate fields, caller-supplied computed results or arbitrary target owner cannot supply authority. No financial values/provider metadata/cursors in URLs/logs/browser persistence; no raw diagnostics. Client has no local matcher, arithmetic or fallback store establishing authoritative categories. No body before commit/unknown-outcome resolution. |
| AUTO16 Product review | Real fictional-data journey using the production composition | Start with a mixture of automatically categorized and true exception rows. Normal path reviews exceptions, explains/corrects one pattern, verifies current whole-result filters/counts and retained manual choice after remount. At390/768/1024/1280/1440 and long labels, controls/focus remain reachable. Hold/denial/error/uncertain states preserve existing global locks. Screenshot/mock callback proof is distinct from actual restricted SQL application acceptance and from owner visual sign-off. |

## Reuse existing regression evidence

These are current implemented tests to retain, not new automatic-mode acceptance:

- `TestRule46Lifecycle`, `TestRule46ManualAndHistorical`,
  `TestRule46AtomicFailure`, `TestRule46UnknownReplyAndRevocation` and
  `TestRule46PrimedSnapshotFences`: exact explicit rule application/manual
  precedence/current authority and original history; the approved matcher remains
  the sole matching implementation.
- `TestPFCVersionMapping`, `TestPFCOmittedUnknown`, `TestPFCRevisionCorrection`,
  `TestPFCSourceMoneyAndScope`: explicit sidecar/current-page attribution;
  these do not prove normalized durable automatic fallback is installed.
- `TestQuery45ThreeThousand`, `TestQuery45OriginalUnits`,
  `TestQuery45DetailAuthorization`, `TestReviewCoverageAndLegacyPrecision`:
  bounded whole-result reads, exact original units and held/unavailable separation.
- `TestTagRecSharedGate`, `TestIndependentRecoveryOrdinaryAckReceipt`,
  `TestIndependentRecoveryPrimedAdmission`, `TestOnboarding47SharedGateRaces`:
  durable interactive recovery and current gate/receipt fences.
- `TestProviderSourceRestore/{prepared,committed}` and
  `TestSourceBackedPostingVolume1000`: independently fixed evidence→source_post
  restore/replay and1000 linked entries/2000 lines/USD100 per side; analytic
  automation must leave these accounting semantics unchanged.
- Actual `TestSyncReviewPostJourney`, Production-shaped and demo journeys retain
  exact source0.10/posting/ack/replay/report/CSV/drill and original units. Retain
  current frontend recovery/readonly/long-label and authority assertions.

## Required design freeze before positive claims

Root must resolve these in the actual contract; this document does not choose an
unreviewed mechanism:

1. Where current authority is attested for autonomous work, its disable/revoke
   behavior and exact permitted source set. Never inherit it from a page view,
   book membership or an arbitrary worker-supplied owner UUID.
2. Durable execution/claim/idempotency/recovery identity, atomic unit, retry class,
   finite backfill/reevaluation policy and stalled-work exception reporting.
3. Decision provenance/revision keys, current-source versus historical rendering,
   manual protection/staleness, retired outputs and rule/map/taxonomy updates.
4. Compatibility with existing interactive command/recovery snapshots and old
   immutable receipt profiles, plus category projection/query/count/drill pins.
5. Finite verified PFC profile/map/fallback policy. A prepared decoder sidecar is
   not evidence that stored historical normalized rows contain categories.

## Handoff and limits

This is a reviewable inventory of proposed proof obligations, not test execution
or owner sign-off. New work is stacked on487; required exact-head CI still applies
to every dependency and final source. Applicable standards/source mapping changes
belong in the reviewed financial/data contracts before implementation, without
inventing an ISO name for this internal analytic operation.

Only this allocated review file and the separate
[offline patched release plan](../runbooks/patched-runtime-release-plan.md) were
edited by this lane. No source/API/SQL/dependency/cloud/provider action occurred.
Managed deployment, feature enablement, real-data application, owner visual use,
live identity/provider validation and accounting auto-post approval remain open.

## Executed implementation evidence —2026-11-12

The inventory above remains a set of obligations; this paragraph does not mark
every proposed race or hosted scenario accepted. Actual restricted PostgreSQL48
now passes nine TestAuto48 groups in collectionlabels: durable unknown prepare
and execute replies,3000facts/30runs with complete1500/1500category/exception
counts and bounded service read, source/policy staleness, saved human work,
later-row atomic rollback, exact attention exclusion, manual correction,
11workspace inventory fairness and retained downgrade refusal. The whole-history
case completed in90.77s without changing the4second transaction limit.

Eight TestCategory48 groups pass actual attributed source stage/complete and
replay, metadata-only financial equality, malformed/forged/unavailable evidence,
231pair/18label catalog equality and the adopted-private provider fallback with
protected manual correction. Actual server TestAuto48BrowserBootstrap and
TestAuto48ServerReadinessSettings pass, including six privilege revoke/restore
cases and mounted HTTP/auth/settings over the restricted SQL login. Existing
TestRule46 lifecycle, compatibility, primed snapshot and fieldwise tests also
passed against48. No categorization journal/outbox effect is accepted.

Local make check/test/build, fresh migration/replay/empty downgrade-upgrade,
707frontend units,100component browser journeys, API types and optimized Next
build pass. Go runs used local1.26.5 with an untracked exact-dependency modfile;
source/CI remain pinned1.26.11. The new full Next/Go/restricted-SQL browser runner
is implemented and wired into CI but blocked locally by EPERM at its unchanged
port prerequisite, before fixture resources. It is not counted among the100
component passes. Canonical exact-head CI and prerequisite487 remain pending;
no provider/cloud/deployment/real-data or owner usability acceptance is implied.
