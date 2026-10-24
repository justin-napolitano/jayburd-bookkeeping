# Bookkeeping readiness planning delivery
<!-- stage-runbook: true -->

## Status and boundary

Planning only under #50/#65, branch plan/bookkeeping-readiness based on
c1cc86eea6a6f05c913ca529a14da6c9f98bd68e. No product, SQL, provider or cloud
mutation. No implementation agents dispatched. The repository plan and GitHub
issues define work; they do not approve real-account access or posting.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Reviewed index, bibliography, decision evidence, command recovery, Tartan runbook, current open issues and latest released #65 lease |
| Audit | Existing golden-month/report/reversal/full-stack journey tests found; their presence is not new acceptance clearance |
| Decompose | Created #296–#305 with native parents/blockers for chart contract/backend/UI, private verifier transport/actual proof, hosted bookkeeping loop, webhook release/actual signed proof and All Transactions API/UI |
| Schedule | Existing epics retained, hard dependencies separated from SQL-writer serialization, unclaimed future lanes and exact lease requirements recorded |
| GitHub readback | All ten new issues' native parent/blocker relationships verified; 27 existing issues refined while preserving original acceptance. #301 additionally blocked by #293 |
| Validation | `make check test build` PASS using cached local Go1.26.5/modfile override; tracked Go1.26.10 pins unchanged, canonical CI remains separate. Initial sandbox attempt could not bind synthetic localhost sockets; rerun outside sandbox passed. No long foundation suite or product/cloud qualification rerun |
| Staged gate | `bash scripts/check-stage-runbook.sh --staged` PASS before commit; final documentation link/structure check and whitespace check PASS |

## Current evidence

[Delivery plan](../plans/bookkeeping-readiness.md) links the issue-backed outcomes,
baseline limits, future tests and authority gates. It preserves #88 matrix
statuses and existing source-review limits. No new accounting policy or external
financial-source claim is introduced. GitHub native relationships use the
documented [sub-issue](https://docs.github.com/en/rest/issues/sub-issues) and
[dependency](https://docs.github.com/en/rest/issues/issue-dependencies) API fields;
these are scheduling tooling references, not financial design sources. Existing
issue bodies retain their historical acceptance after a new execution-plan prefix;
no issue was closed, assigned an implementation lease or relabeled PASS.

## Remaining work

Publish/review the plan PR and release the bounded planning lease. All
implementation and qualification work remains open until the linked acceptance
evidence is actually produced. GitHub remains the current scheduling authority.

## PR handoff

Refs #50, #65, #88, #104, #237, #242, #271, #272, #294.
Documentation/issue planning only; no migration, runtime contract or release
configuration change. Hosted test evidence remains the existing dated record.
