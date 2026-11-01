<!-- stage-runbook: true -->
# Production USD history — #380

## Status and boundary

USD380-HISTORY; fix/380-production-usd-history from9d75886. Production read-only
metadata found historical USD coverage absent; current reference installed374.
Source-only reference bootstrap repair; no actual reference apply, bank call,
financial-record inspection or migration change. Qualification375 proceeds separately.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live380/375 and completed374/225 reviewed; actual history absence verified |
| Sources | Existing dated-currency/source register and MA115 page1 text re-reviewed; access limits recorded in contract |
| Decision | Atomic exact-target current+historical public reference insertion, preserving all existing metadata |
| Implementation | Exact guarded current+historical insert/check in one transaction; no existing row rewritten |
| Validation | Core check/test/build; Production+Sandbox bootstrap, ledger and acquisition suites PASS on disposable PG17 |

## Current evidence

Independent fixed current/history metadata, synthetic late-insert rollback and conflict/overlap denials passed. Installed dates accepted Aug3/Sep16/history and Sep17/current while swapped versions and excess precision rejected; no Item/journal effect. Existing ledger/acquisition regression and core gates passed. No actual reference apply or new cloud image. Local installed Go1.26.5/external temporary modfile; tracked1.26.10 unchanged.

Pinned existing USD/840/2 and historical local coverage/provenance are reused;
no new financial message or ISO conformance claim. The current Q4 book period
does not constrain source acquisition dates or authorize out-of-period posting.

## Remaining work

Exact-head CI and human review,
then separately scoped reference-only apply/readback. No silent90-day reduction.

## PR handoff

Refs380/375/225/226/337/90. Merged migrations immutable; no image/API/runtime
change or automatic DML. Actual personal-account pilot/reconciliation remains open.
