<!-- stage-runbook: true -->
# Read error copy —291

## Status and boundary

UI291 from49d7989/PR423; dedicated issue/291-read-error-copy. Parent source review
and merge remain pending. Fixed browser messages only; no API-wire, SQL, auth,
provider, cloud or live authority change.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live291/416/65 and parent CI checked; exclusive UI291lease09Z; JB-001packet |
| Design | RFC9110 selected sections reviewed; contract preserves uncertainty/authority |
| Implementation | GET/CSV fixed availability wording, unchanged uncertainty and POST recovery; no raw error-body reads | 
| Validation | Core make check/test/build PASS; locked web API/types/check382unit/build PASS;43browser PASS; actual browser/Next/Go/disposable PostgreSQL PASS | 

## Current evidence

[Contract](../contracts/read-error-copy.md) records facts, inference, alternatives,
unchanged standards applicability and named acceptance.

## Remaining work

Dependent PR, exact-head CI and human review remain pending.
416 categorization recovery and spending/rules remain separate.

## PR handoff

Refs291/412; stacked on423/422/421. No migration. Human review and exact-head CI
required before merge; synthetic identity does not prove hosted/provider behavior.
