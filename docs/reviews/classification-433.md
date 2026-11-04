# Independent categorization review cases —433

Fixed fictional examples written before runtime implementation. These are manual
contract oracles and future test specifications, **not executed acceptance tests**.
No generator, provider payload, real merchant/account or ledger calculation is
used. Currency IDsUSD-A/JPY-A are distinct dated installed references. All money
below is exact decimal text; no cross-unit total is defined.

| Planned evidence | Fixed input | Expected result |
| --- | --- | --- |
| TestPFCVersionMapping | Current booked USD-A24.01; explicit supported PFCv2 pair and approved mapM1→Dining | Attributed Dining suggestion with v2/M1/source revision; no reviewed treatment, draft or journal |
| TestPFCOmittedUnknown | Same transaction, PFC absent/null/versionless/v9/unknown pair, each separate case | Fixed abstention reason; original24.01/source date retained; no invented historical v1/v2 |
| TestPFCConfidenceNotCalibration | Supported mapped pair, UNKNOWN confidence | Display attributed metadata, abstain from fallback suggestion; no invented probability |
| TestPFCRevisionCorrection | Source revision1 PFC Dining; revision2 PFC Groceries, same24.01 | Current provider attribution changes; old version retained; no second purchase or journal |
| TestProviderCategoryIsInert / TestNoJournalFromPFCSuggestion | Supported category on an otherwise reviewable observation | Query/preview leaves command/journal/outbox counts unchanged |
| TestPFCNormalizedCompatibility | Old normalized profile lacks PFC; new version adds explicit nullable PFC | Old replay remains byte/semantic-compatible; new SQL/profile installed before new wire; no edit to merged SQL |
| TestPFCPrincipalDeny | PrincipalB requests principalA source or guesses its observation UUID | Deny before count/category/confidence/cursor disclosure |
| TestOwnerRuleScope | A's account-specific rule for Cafe; B has same exact merchant label; A has another account | B never sees A's topic/rule; A's account rule does not match A's second account |
| TestTaxonomyRevision | Preview uses taxonomy3/topicT; owner retiresT in taxonomy4 | Apply conflicts; historical topicT decisions remain readable only to current authorized owner |
| TestTaxonomyCycle | TopicsT1→T2 and proposedT2→T1 | Reject whole taxonomy edit, retain previous edition |
| TestRulePrecedence | Explicit Groceries override; matching approved Cafe→Dining rule and PFC Shopping | Groceries remains current; all lower suggestions cannot overwrite it |
| TestAccountBeforeWorkspaceRule | Account rule→Travel, workspace rule→Dining, same exact merchant | Account proposal Travel wins with account-rule provenance |
| TestRuleSpecificity | Equal-priority account exact Cafe→Dining and account contains Cafe→Shopping | Exact Dining wins; broad match remains explainable but not selected |
| TestRuleConflictAbstains | Equal-priority/scope/exact Cafe rules→Dining and→Travel | Conflict with both IDs/revisions; no UUID winner, no PFC fallback |
| TestRuleAgreement | Equal-winning-rank rules both output Dining and the same tag set | One Dining suggestion with both provenance IDs; count transaction once |
| TestRulePreviewApplyEquality / TestBulkLabelAtomicity | Preview exactly observationsO1:r2,O2:r1; apply exactly those at unchanged pins | One atomic override receipt for those2; no O3 or later page silently included |
| TestRuleVersionConflict | Same preview, ruleset changes4→5 or O2:r1→r2 | Entire apply conflicts; neitherO1 norO2 changes; new preview required |
| TestOverridePreserved | Manual Dining onO1:r1, later provider/rule revisions propose another topic | Manual decision retained; changed source revision marks it stale rather than overwriting |
| TestRegexWorkBound | Pattern with backreference,129 UTF8 bytes, compiled program>1024,101 rules, or501-codepoint input | Each unsupported/oversize case rejects/unavailable by reviewed bound; never false no-match or unbounded evaluation |
| TestRuleRevisionReplay | Apply commandK/bodyV commits, reply lost; retryK/V; then retryK/differentV | Exactly one analytic decision receipt; same authorized receipt on equal retry; mismatch conflict |
| TestLabelRevocationRace | Owner's classify/source grant is revoked while apply waits for lock | Current authority check denies whole write; old preview grants nothing |
| TestPostedTagChangeDoesNotMutateJournal | Posted source100 has journalJ; owner retags topicT1→T2 | New analytic audit decision; journalJ,100 original posting and receipt unchanged |
| TestCorrectionInvalidatesReviewedTreatment | Posted100 corrected source to15; prior treatment pinsr1 | Source activity15; immutable journal100; reviewed spending contribution unavailable until r2 confirmed |
| TestPendingRemovedNoSpending | Pending10, removed20 and valid booked purchase30 | Activity/spending eligibility excludes10/20; current reviewed spending30 only; no implicit refund20 |
| TestSpendingAllocationEquality | Booked loan payment220, reviewed allocations principal200+interest20 | Outflow220; gross spending20; contribution drill20 alongside original220; principal200 excluded |
| TestUnallocatedLoanAbstains | Booked loan payment220, topic Loans, no reviewed allocation | Outflow220, unallocated220; no guessed interest or220 expense |
| TestRepaymentNotDoublePurchase | Card purchase50; checking repayment50; card repayment inflow50 | Source records stay separate; reviewed spending50 purchase once, repayment contributions0 |
| TestRefundOwnDate | September purchase30; linked October refund10 | September gross30; October refund10/net−10; no historical September rewrite |
| TestMixedCurrencyNoGrandTotal | Reviewed USD-A12.50 and JPY-A1000 | Separate unit totals12.50 and1000; no1012.50 or browser FX |
| TestCategoryVersusTreatment | Topic Dining on transfer40 with reviewed transfer treatment | Topic visible; contribution0; topic does not override economic treatment |
| TestMetricDrillContext | Summary sourceS/unitUSD-A/taxonomy3/rules4/allocation8; allocation changes9 | Drill conflicts/refreshes, never new contributions under old total |
| TestThreeThousandItemsPagination |3000 eligible observations, bound100 | All unique rows/counts over30 pinned pages; aggregate applies before page limit; no read-all client sum |
| TestOpportunityAbstains | Only one matching subscription-like purchase or unknown history completeness | Insufficient evidence; no unused-service/savings claim |
| TestOpportunityContributionBasis | Three monthly50 purchases and one50 card repayment | Candidate cites purchases only; repayment cannot double measured recurring spend |
| TestScenarioIsNotSavingsPromise | Owner selects monthly100 reviewed category baseline and10percent reduction | Exact scenario10 in same unit, clearly hypothetical; no cancellation, transfer or guaranteed outcome |

Map these specifications to actual unit/SQL/runtime-role/concurrency/browser tests
at implementation. Core/docs validation cannot turn this table into passing product
evidence. Future regex timing/work and3000-row cases need actual bounded load proof.

## Review outcomes required

- Identity/database reviewers accept238's private workspace/source authority and
  the239 adoption/grant/revocation/retention contract. Sharing and cross-entity
  posting remain outside this analytic feature.
- Maintainer reviews the proposed single primary topic plus tags, case-sensitive
  matching, account-before-workspace precedence, equal-rank abstention, stale
  override confirmation and inert future rule suggestions.
- Source/security reviewer accepts413's additive normalized compatibility design
  and the distinction between provider confidence, suggestions and reviewed
  treatment. No historical backfill or private source read is included.
- Implementers then fill exact physical/API/lock profiles and run the named
  independent cases. Nobody closes238/242/413/414 from these planned examples.
