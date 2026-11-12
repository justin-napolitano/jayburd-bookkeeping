# Current capability map

Reviewed2026-11-11 against maina9561e2 and the exact tested f84a359 tree merged
by483. This is the current product entry point; GitHub owns live execution and
acceptance state. [Historical delivery index](implementation-history.md) retains
the original narratives. An open issue can mean remaining owner review or hosted
qualification rather than missing code. No synthetic result approves official books.

## Implemented and proven with synthetic data

Latest [exact acceptance receipt482](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/482#issuecomment-6805089153):
all five checks passed, including actual browser→Next→Go→restricted PostgreSQL,
673 frontend units/98 component-browser journeys, source-backed restore and
1,000 source-backed postings. Those are the baseline before this consolidation;
retiring unused-code-only tests changes counts, not the required active behavior.

| Capability | Implemented boundary and evidence | Distinct remaining work |
| --- | --- | --- |
| Private workspace setup/history adoption | Durable in-app setup and adoption preserve original history without native copies; [contract](contracts/collection-onboarding.md), [release evidence](runbooks/onboarding-sandbox-rollout.md) | Actual authenticated owner review/adoption acceptance460/453/436 |
| Transaction review | Current scoped queries, sorting/filters, original units, detail and authorized journal links; [query contract](contracts/collection-query-rules.md), [UI stages](runbooks/collection-filters-rules-ui.md) | Owner usability review; broader product proposals are separate |
| Private categorization | Persistent topics/tags and reviewed rule apply; candidate488 adds automatic private rule/provider categories, protected corrections and exact exceptions filter; [automatic contract](contracts/automatic-categorization.md), [delivery evidence](runbooks/automatic-categorization.md) | Automatic runtime flags remain off; final composed browser/CI acceptance and deployment are pending. Automatic accounting334 remains separate. |
| Source-backed accounting | Retained bank/provider or statement row→review/category→immutable draft→saved source_post→confirm/execute→explicit acknowledgement; [artifact policy](contracts/artifact-backed-accounting.md) | Receipt/invoice ingestion476 and artifact-linked correction333 are not implemented |
| Reports/CSV/detail | Exact ledger/trial balance, financial reports, pinned comparison/export and same-journal drill; [reports](contracts/financial-reporting.md), [CSV](contracts/report-export.md), [drill](contracts/report-drill.md) | Full statement-to-ledger golden-month reconciliation84; real policy/accountant acceptance |
| Recovery and authority | Same-command replay, unknown outcomes, explicit cancel/ack, revocation/current-scope/isolation; [recovery](contracts/command-recovery.md), [matrix](reviews/synthetic-application-matrix.md) | Hosted identity/MFA/provider behavior is separately qualified |
| Provider/worker mechanisms | Scoped enrollment/acquisition, signed synthetic webhook receipts, durable demand/lease/generation/retry fences; [boundaries](contracts/provider-runtime-boundaries.md) | Actual applicable Plaid/OAuth/reconnect/webhook cases88 and live pilot90 |
| Bounded restore/load | Prepared and committed-unacknowledged source graph restore; 1,000 linked entries/2,000 lines/100USD per side with exact replay; [proof stage](runbooks/synthetic-readiness.md) | Managed backup/PITR/key recovery, multi-book/worker ingestion benchmark89, cost/soak58/64 |
| Banking UI | Responsive navigation, compact account actions, contextual recovery and removed idle debugging; [UI stages](runbooks/customer-banking-ui.md), [web release](runbooks/customer-ui-web-release.md) | Owner visual/private-use acceptance479 and related review issues |

## Source/build and deployed acceptance

Source/build Go pins are1.26.11 after484; the required scanner passed with zero
affecting findings. One uncalled required-module finding remains recorded in482.
No blanket dependency-clean or exploit-remediation claim follows.

The accepted prior web-only release is web00027 with preserved API00026. The
source/build patch is not a deployment: [485](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/485)
tracks patched immutable artifacts and separately scoped rollout. Release
runbooks record dated settings/health evidence; this map does not perform a fresh
cloud inspection or infer live provider/financial acceptance.

## Real remaining scope

| Type | Existing work item | Completion requires |
| --- | --- | --- |
| Financial acceptance | [84](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/84) | Complete independent statement golden month, balances/timing and mutation probes; reuse existing ledger/report/source code |
| New capability | [333](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/333), [476](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/476) | Evidence-linked corrections; retained documents and reviewed event matching without double-posting |
| Optional new automation | [334](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/334) | Owner-approved versioned accounting rules, holds and current authorization; not the already implemented categorization rules |
| Other scoped capability gaps | [297/298](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/297), [413](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/413) | Reviewed existing-book account addition; normalized/integrated PFC suggestions beyond the delivered current-response sidecar |
| Operational acceptance | [89](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/89), [58](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/58), [485](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/485) | Defined ingestion/load/fault/cost/managed restore evidence and separately scoped patched deployment |
| Hosted/external qualification | [88](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/88), [90](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/90) | Actual provider/identity cases, then a bounded statement-reconciled real-account pilot |

These are separate acceptance conditions, not a plan to repeat delivered features.
The [backlog disposition audit](reviews/consolidation-backlog.md) records evidence
and precise issue narrowing/closures for486. The [code audit](reviews/consolidation-code-audit.md)
records safe removals and retained compatibility. Only current GitHub state decides
which issue is active; preserve owner/hosted gates when narrowing old issues.
