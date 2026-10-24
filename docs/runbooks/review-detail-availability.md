# Review detail availability delivery
<!-- stage-runbook: true -->

## Status and boundary

#293, DETAIL293-20261024T0535Z, fix/293-preserve-review-detail from c1cc86e.
UI only; no SQL/API, accounting/posting/date policy or provider/cloud change.
[Decision](../contracts/review-detail-availability.md). Current permission loss
still removes sensitive snapshots; categorization failure cannot authorize drafts.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Current #293/#65, source-review-sql and command-recovery contracts reviewed; no competing writer |
| Diagnosis | Detail was set before categories, but shared catch erased it; policy-at-booked-date lookup observed in immutable SQL, actual hosted error remains unverified |
| Implementation | Preserve authorized detail on non-auth category failure, clear draft controls and offer explicit retry; existing denial/generation paths preserved |
| Validation | make check/test/build and web-check/test/build/browser PASS;335 unit tests,25 browser cases; delayed/error/retry and401/403/404/changed-book tests included |
| Full stack | make app-integration PASS against real browser/Next/Go/restricted disposable PostgreSQL; existing exact explicit posting/lost-commit recovery remains intact |

## Current evidence

No change to stored observations, booked/accounting dates, posted journals,
money/currency rendering or persisted commands. No real financial data read.
Synthetic tests do not establish actual managed historical-row acceptance.
Desktop/mobile browser fixture preserves the historical date/exact amount and
read-only explanation through keyboard retry, with GET-only requests and no draft
or command creation. Local Go1.26.5 cached modfile override; tracked1.26.10 and
all dependencies unchanged. Canonical CI remains a separate exact-head gate.

## Remaining work

Staged runbook check and PR/CI. Managed historical-case acceptance remains separate.

## PR handoff

Refs #293/#104/#88/#90. No migration, dependency or generated API change. Preserve
independent reviewed accounting authority and existing current-scope denial.
