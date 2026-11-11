# Synthetic application acceptance matrix — 482

## Scope and method

Audit baseline main `0a7d37317ff89cab0df75d872095ee2a6f3dd770` on
2026-11-11. This is source review plus named fresh CI evidence, not a new
accounting policy, live-bank qualification or financial-framework certification.
Only fictional identities, provider adapters and isolated PostgreSQL fixtures are
permitted. No financial records, credentials, provider calls or managed mutations
were used by this review. GitHub482 and the marked
[stage ledger](../runbooks/synthetic-readiness.md) own delivery acceptance.

Grounding: [research index](../research/bibliography.md),
[corpus/source register](../../tests/corpus/README.md),
[independent activity oracle](../../tests/corpus/spending/README.md),
[artifact policy](../contracts/artifact-backed-accounting.md),
[source review](../contracts/source-review-sql.md),
[command recovery](../contracts/command-recovery.md),
[identity](../contracts/identity-setup.md),
[reports](../contracts/financial-reporting.md),
[CSV](../contracts/report-export.md) and
[journal drill](../contracts/report-drill.md). These contracts distinguish
implemented boundaries from proposed or privileged low-level capabilities.
Named tests below were found in current source; naming alone does not establish
a fresh passing run. CI acceptance is recorded separately below.

## Implemented chain and fixed oracles

| Claim | Named current evidence | Independent expectation and boundary |
| --- | --- | --- |
| Strict source decode retains decimal units/dates/identity | `TestPlaidExactMoney`, `TestPlaidLosslessAndDates`, `TestPlaidStrictJSON`, `TestPlaidAccountMoneyMatrix`, `TestDecoderMoneyOracle`; statement `TestStatementMoney`, `TestStatementExpectations` | Literal decimal/reference cases and deliberate invalid forms; decoder acceptance is not bank-profile qualification. No float valuation or assumed undocumented currency. |
| Staged evidence reaches durable import atomically | `TestImportServicePostgresStages`, `TestImportServicePostgresRejection`, `TestDurableImportIdentity`, `TestDurableImportAtomicity`, `TestDurableImportConcurrency` | Retained observation/revision/assignment identities; failed publication has zero financial effects; replay must not add observations. Statement import client/service is not a new public upload UI. |
| Acquisition handles page faults, changed source and remote uncertainty | `TestAcquisitionPageFaultPreservesPublication`, `TestAcquisitionMutationRestart`, `TestAcquisitionCommittedReplyLoss`, `TestAcquisitionLifecycleOracle`, `TestSyncPostedEvidenceRemainsOriginal` | Original source evidence remains fixed while current source revisions can change; posted evidence is not rewritten by later sync. Test-only remote adapter does not prove live cursor/history behavior. |
| Review/category draft is inert until explicit confirmation | `TestSourceReviewPosting`, `TestSourceReviewSnapshots`, `TestSourceReviewCommandIntegrity` | Two independently authored 12.50 inflow/outflow observations; draft creates zero journals. Source asset leg remains debit/credit 12.5 in both transaction and functional units. Two executed reviews yield exactly two entries, links, receipts and outbox intents, with original two revisions retained. |
| Public application only creates artifact-linked new entries | `TestArtifactPublicBoundary`, `TestArtifactExecutionTransaction`, `TestArtifactHTTPExecution`, `TestArtifactRegistrationAndSafeTransitions`, `TestArtifactRetriesAndReplay` | Auth first; public direct manual post/reversal and new unlinked post registration reject before posting SQL. Prepared legacy manual work cannot execute publicly; cancel/ack and committed receipt recovery remain. This is an application boundary, not a database-wide ban on privileged ledger primitives. |
| Real application posts one exact reviewed source item | `TestSyncReviewPostJourney` in `web/tests/integration/connections.spec.tsx`; `scripts/test-app-integration.sh --connections` | Literal source0.10 -> two exact0.1 USD lines; one journal, one review_posting, two imported observations, one outbox. Source detail links to the same immutable journal. Import/sync/draft alone do not post. |
| Actual report, journal/account drill and downloaded CSV agree | Same `TestSyncReviewPostJourney`/`verifySourcePostingReports`; `TestFinancialReportsGoldenMonth`, `TestReportCSVGoldenSQL`, `TestReportCSVAllPagesSQL`, `TestReportJournalPinned`, `TestReportReviewHTTPPostgres` | Source0.10 trial balance and ledger closing0.10; balance sheet/net income-0.10; comparison0.10/0.00/0.10; CSV total_debits exact0.1/0/0.1 with CRLF. Shared revision pins and same journal ID on two-line drill. Corpus golden month independently fixes checking1045, savings100, income70, capital1100; bookB opening70 stays separate. |
| Source activity totals cover whole result, never a bank balance | `TestSpendingCurrenciesSeparate`, `TestAggregateDrillEquality`, `TestWholeResultNotPage`, `TestReadNeverPosts`, `TestSourceRevisionSnapshot`, `TestFreshnessUnknownStale` | Hand-authored activity fixture: outgoing940, incoming2017.50, net1077.50 in one USD version; separate2-unit version never merged. 3000 rows total60, first50 only0.99. Browser acquisition0.10+0.20=0.30, two exact drills; balance/freshness may remain unknown. PhaseB allocation/spending157.50 is an illustrative future oracle, not installed spending classification. |
| Private topics/tags/rules remain categorization, not accounting | `TestBulkTopicAtomicity`, `TestPostedRetagNoJournalMutation`, `TestIndependentStrategicTagSets`, `TestRule46ManualAndHistorical`, `TestRule46AtomicFailure`; actual `rules.spec.tsx` | Manual topic/tags protected fieldwise; equal-rank disagreement abstains; rules preview is inert and application explicitly confirmed. Existing rule actor has three facts/two rule decisions. Posted journal/source history remains unchanged after retagging. These choices are not category-account posting or auto-post334. |
| Existing history can be adopted without native duplication | `TestExistingBookMigration`, `TestOnboarding47AdoptionAndFences`, `TestOnboarding47HTTPAdoptionAndRecovery`, actual `onboarding.spec.tsx` | Two original legacy observations retain0.10 and9007199254740993.01; no native observation copies/journals. One canonical adoption, original cursor retained, Item paused:generation2; two acknowledged setup/adoption commands, reservation count0. API uses verified enrolled actor, not caller UUID/email inference. |

## Failure, authority and money boundaries

| Risk | Named current evidence | Required outcome / limit |
| --- | --- | --- |
| Lost execute/commit or acknowledgement reply | `TestSyncReviewPostJourney`, `TestCommandDatabaseUnknownCommit`, `TestCommandLostAcknowledgement`, `TestCommitAcknowledgementLoss`, `TestImportServicePostgresUnknownCommit`, `TestRule46UnknownReplyAndRevocation` | Recover same body/key/command/receipt; explicit acknowledgement; no replacement key or inferred completion from empty pending. Source browser replays acknowledged execute and compares same journal, while acquisition replay retains journal_count1. |
| Cancel/execute race and late failure | `TestCommandConcurrency`, `TestSourceReviewAtomicity`, `TestLabelExecuteCancelRace`, `TestLabelExecutionAtomicRollback`, `TestIndependentTagAtomicExecution` | Whole transaction rollback includes journal/lines/link/receipt/outbox/revision; cancel never reverses a committed effect or posts anything. |
| Current authority, tenant/book isolation and restricted roles | `TestSourceReviewAuthorization`, `TestSourceRegistryMixedScope`, `TestSecondBookIsolation`, `TestSchemaPrivileges`, `TestSourceRegistryPrivileges`, `TestCollectionPrivileges`, `TestQuery45DetailAuthorization`, `TestReportCSVAuthoritySQL` | Foreign/inactive/revoked actor/source/book denied even on retained receipt replay; actual restricted roles cannot directly change tables or execute owner/helpers. Reused account IDs across scopes do not share balances. Synthetic actor injection is test-only. |
| Primed snapshots and revocation races | `TestSourceRegistryRevocationRace`, `TestCurrentSourceReadLocks`, `TestReportRevocationRace`, `TestReportJournalRevocationRace`, `TestLabelPrimedSnapshotFences`, `TestRule46PrimedSnapshotFences`, `TestOnboarding47PrimedWorkspace` | Observed row-lock races force retry/deny rather than disclose stale authority. Cursor/snapshot mismatch fails closed; no stale pins silently accepted. |
| Shared pending work / inaccessible recovery | `TestSharedInteractiveReservation`, `TestTagRecSharedGate`, `TestIndependentRecoveryOrdinaryAckReceipt`, `TestIndependentRecoveryPrimedAdmission`, `TestTagRecRecoveryRestoredAndNewClaim`, onboarding recovery groups | One principal gate across engines, explicit terminal acknowledgement. Payload-free clear cannot execute/reverse, cannot clear newly authorized work, and exact old receipt cannot release a newer claim. Discovery alone changes no decisions/journals. |
| FX, original units and precision | `TestSourceReviewFXGate`, `TestCurrencyScalesAndRounding`, `TestHistoricalCurrencyAcquisition`, `TestSourcePrecisionHold`, `TestPrecisionEligibilityForgery`, `TestReviewCoverageAndLegacyPrecision`, `TestQuery45OriginalUnits` | Source JPY12/KWD12.501 remain exact but foreign-category drafts return fx_required and create zero draft/journal. Low-level synthetic spot examples assert half-even1.005→1,1.015→1.02,2.675→2.68 and reject forged99. USD1.001 remains evidence under a hold, not rounded into eligible money. Low-level FX tests do not implement source-review FX posting. |
| Unsupported, overlapping, stale or closed-period facts | `TestSourceReviewBoundaries`, `TestSourceReviewSnapshots`, `TestCurrencyHoldAtomicity`, `TestQuery45ClosedInputs`, `TestReportBoundaryAndRevision` | Overlap cannot draft; stale source/book pins reject; closed period leaves command prepared with zero financial effect; unsupported currency is held/unavailable rather than guessed. No resolution/matching UI is implied. |
| Worker demand, lease/generation fencing and webhook replay | `TestSyncFenceAndWakeup`, `TestDeliveryPermanentFencesLatePublication`, `TestDeliveryExpiredLeaseNewClaim`, `TestDeliveryUnknownFailureQuarantines`, `TestDispatchRealUnknownClaim`, `TestWebhookPostgresReceipt`, `TestWebhookPostgresRoleIsolation` | Signed synthetic receipt/replay and durable demand/claim rules, zero unreviewed journals; revoked/expired workers cannot publish a new current effect. Does not qualify actual webhook delivery, KMS/IAM, institution OAuth or hosted retries. |

## Volume and restore: keep claims separate

- Existing imported/read volume: `TestCollectionPagination3000`,
  `TestAllTransactionsPagination3000`, `TestTagRecPagination3000`,
  `TestQuery45ThreeThousand`, activity `TestWholeResultNotPage`, and demo
  acquisition retain bounded page/count and independent original-unit expectations.
  `TestIndependentHundredTagHTTP` covers100 selected of3000 under the actual
  bounded service context, not unlimited batches.
- `TestReportThousandPostings` covers ledger/report materialization, not1000
  source-backed application posts. The public demo imports3000 observations and
  explicitly posts one linked entry. These workloads do not prove a universal
  throughput/latency SLA, arbitrary-volume ingestion or accountant acceptance.
- Baseline `TestLedgerRestoreReplay` is a low-level ledger receipt restore test.
  The new482 `TestProviderSourceRestore/{prepared,committed}` uses actual
  restricted `importer.Service` plus public Handler: two retained CSV rows0.10
  and0.20USD, explicit reviewed import/ack, draft and source_post, then isolated
  dump/restore. It compares pre/post hashes of selected evidence, review, draft,
  commands, link, journal, receipts and outbox identities. Independent oracles
  require two source rows, one draft, original lexemes0.10/0.20, source sum0.3;
  before execute zero journals/links/receipts/outbox, after execute one of each,
  two0.1 journal lines and trial-balance debit/credit0.1. Prepared or committed
  unacknowledged body/result recovery, repeated execute and post-ack replay must
  not duplicate effects. Raw object bytes are an explicitly recovered synthetic
  input beside their retained reference/hash; pg_dump alone does not restore
  object storage. Compile/vet passed; first actual execution FAILED as recorded
  below, so restore acceptance remains open.
- New `TestSourceBackedPostingVolume1000` uses1000 original0.10USD CSV outflows,
  actual restricted import and HTTP detail/categories/draft/prepare/execute/ack.
  Fixed independent targets are1000 unique journals/links/drafts/receipts/outbox,
  2000 lines,100USD per side and report total100; every execute replays before
  acknowledgement and four historical acknowledged commands replay again after
  all later revisions. This is sequential bounded source-backed volume, not an
  arbitrary-concurrency or availability SLA. Compile/vet passed; actual execution
  FAILED on the first actual CI run as recorded below, not yet an accepted
  workload result.
- Both new tests remain synthetic-only. The standard local proof attempt was
  blocked before fixture resources by Docker credential-helper/socket permissions;
  no credentials, network/permission bypass, alternative SQL authority or skipped
  assertion was used. Source-level readiness is distinct from restricted SQL
  execution/restore acceptance.

## Fresh baseline acceptance

Exact baseline source: `0a7d37317ff89cab0df75d872095ee2a6f3dd770`.

| Gate | Exact run/job | Observed result |
| --- | --- | --- |
| Seed frontend | [43078080938 /130723126216](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/43078080938/job/130723126216) | SUCCESS;673 units across39 files,98 component browser journeys23.4s, two identity-rehearsal browsers2.0s; generated types/build/review/image checks also passed |
| Seed actual application | [43078080938 /130723126038](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/43078080938/job/130723126038) | SUCCESS, completed2026-11-11T18:35:28Z; selected results below |
| Seed backend validate | [43078080938 /130723125517](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/43078080938/job/130723125517) | In progress; listed source tests are not all claimed freshly passed yet |
| Infrastructure | [43078080924](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/43078080924) attempt1 | SUCCESS on exact baseline; not application/business acceptance |

Actual source-stage success must include its mandatory post-browser SQL guards:
journal_entry1, review_posting1, source_observation2 and outbox_message1.
Production-shaped fixture guards separately require original provider lexemes
0.10,0.20, exact sum0.3, one journal/outbox and one production-shaped Item;
the adapter still performs no external bank calls. Ordinary setup/recovery guards
require journals0, operation_receipt0, setup_receipt1, one acknowledged committed
command/execute audit and outbox0. Private label/rule flows require books0/journals0;
onboarding requires no native-copy observations or journals. A browser-only pass
does not replace these independently evaluated SQL comparisons.

Selected public actual application output (no full logs retained here):

- Private labels/tags/rules and inaccessible recovery: four passed8.9s; final
  SQL guard reports zero books/journals.
- Setup/adoption: one passed2.9s, with retained-history/no-copy SQL guards.
- Ordinary API/PostgreSQL recovery: one passed1.9s.
- `TestSyncReviewPostJourney`:6.2s, suite7.3s at18:34:15Z. Mandatory source
  posting counts passed after the browser; source0.10, same command/journal
  replay, report/CSV/drill and empty browser persistence assertions were retained.
- `TestProductionProfileJourney`:3.6s, suite4.2s at18:34:54Z. Final guard states
  fictional Production through real browser/API/SQL, no external bank calls.
- Four demos passed12.3s: acquisition9.2s, no-manual-runner387ms,
  historical-cent oracle913ms and large-import/no-auto-journal980ms.
  Final log confirms3000 generated imports, one atomic source-linked posting,
  exact cents and recovery.

## Explicit remaining acceptance

Synthetic identity/provider adapters do not prove Firebase enrollment/MFA,
OAuth/institution behavior, real consent renewal, Plaid source completeness,
bank reconciliation, cloud retention/restore, live retry characteristics or
accountant-reviewed policy. Anonymous health/auth transport and immutable deployed
revision metadata are separate bounded observations, not private-use approval.

Document receipt/invoice ingestion and matching476, auto-post334, source-linked
corrections333, transfer-pair matching, source FX/split drafting and complete
reconciliation remain NOT IMPLEMENTED/EXTERNAL VALIDATION as their contracts
specify. Low-level reversal/FX primitives and the illustrative corpus do not
expose these as customer workflows. PhaseB spending/allocation examples remain
proposed. No placeholder Documents UI or manual entry is an acceptable substitute.

Several older index/corpus paragraphs still describe already implemented rules
as unintegrated or list runtime gates generically as future work. This matrix
uses current named implementations for positive claims and retains the narrower
contract limits; documentation cleanup alone must not broaden feature acceptance.

## Extension run tracking

PR483 source `41155b77160a2d85db449ebfd337e1234c7e7dec` adds the test-only
restore/volume extension and moves the existing provider-flow proof earlier in
Seed so failures surface sooner. No gate, permission, image or timeout is removed
or weakened. Initial run43080136527 is superseded and is not final acceptance.
Exact-head Seed [43080241136](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/43080241136)
has validate130730726857, actual application130730726304 and
frontend130730726874. Frontend and actual application are SUCCESS.
The new restricted source-restore/posting-volume phase FAILED. Foundation is
[43080241199/job130730729377](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/43080241199/job/130730729377).
Record actual named test results and final gate metadata before promoting the
pending claims above.

Selected first actual restricted provider-flow output,2026-11-11:

- Existing `TestProviderEnrollmentAcquisitionFlow` PASS0.29s.
- `TestProviderSourceRestore/prepared` failed1.41s and `/committed` failed1.37s
  at `restore_integration_test.go:393`: restored immutable journal differs from
  receipt. Overall restore2.77s. Do not infer corrupted source/product history
  from this assertion until its actual projection comparison is diagnosed.
- `TestSourceBackedPostingVolume1000` failed18.80s at
  `volume_integration_test.go:134`: volume source projection or book revision
  changed. Posting throughput/count acceptance is not granted by reaching that
  late assertion; inspect actual count and revision assumptions first.
- Both test owners and coordinator received exact failures. No runtime edit,
  relaxed assertion, timeout increase, skipped gate or substitute authority is
  part of this matrix review. Corrected source requires a fresh exact-head run.

Existing actual application acceptance on that same extension head remained
green: labels/rules four8.5s; onboarding one2.6s; recovery one1.6s;
`TestSyncReviewPostJourney`6.2s/suite7.1s, Production3.1s/suite3.7s,
four demos10.3s. Final log again confirms3000 imports, one atomic source-linked
posting, exact cents and recovery. This preserves baseline workflow acceptance
but does not override the two new failing test groups.
