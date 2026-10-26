<!-- stage-runbook: true -->
# Initial empty sync receipt — #346

## Status and boundary

EMPTY346-RECEIPT; fix/346-empty-sync-receipt from3a2365c. Source-only receipt
validation fix discovered by315. No SQL/cloud/provider/secret or real records.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Reproduce |315 actual Production-profile browser reaches first empty page; one fictional fetch, invalid_metadata |
| Review |Current complete_sync preserves source_revision0 when changed=0; Go requires positive revision for Complete and Resume |
| Decision |Accept canonical nonnegative int64 source revisions only; keep all generations, fences, scope and currency controls |
| Validation |TestInitialEmptyCompletionAndResume failed before the fix, then check/test/build/acquisition-test all PASS |

## Current evidence

Existing bibliography/PP04, provider-sync SQL and acquisition-runner contracts
reviewed. Operational source revision grammar is not a financial message; no new
ISO20022 mapping, money/currency semantics or ISO4217/ISO21378 applicability change.
Alternative of fabricating a source change would misrepresent evidence and alter
immutable SQL unnecessarily. Complete/Resume must recognize the same SQL receipt.

## Remaining work

315 composed real-SQL browser proof and exact-head CI/human review.
Local gates use installed Go1.26.5 and an external temporary modfile/cache after
the pinned1.26.10 download returned403; tracked pins remain unchanged.
Test validates both Complete/Resume plus canonical int64 bounds and retained
positive generation. No observed monetary output is used as a generated oracle.

## PR handoff

Refs346/315. No migration or dependency impact. Risk is receipt grammar expansion
only for valid0, with invalid/negative/noncanonical/overflow and zero generation
denials retained. No new provider work or journal posting authorized by receipt.
