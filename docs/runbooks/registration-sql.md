<!-- stage-runbook: true -->
# Registration SQL delivery —392

## Status and boundary

REG392-SQL; feat/392-registration-sql from391 plan d0c009b. Sole SQL writer;
forward36 reserved under65. Source/synthetic tests only, no managed apply or
increase to one-account Production mandate. Existing first checking untouched.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live392/391/290/387/65 and PR397 reviewed; no competing SQL lease; migrations1–35 immutable |
| Source review | Database foundations/decision evidence/standards,34 bind/scope,11 command lifecycle,16 inaccessible recovery, selected SR01–04 |
| Physical proposal | registration-sql contract; reuse immutable interactive command instead of parallel plan store; shared binding loop; default1 limit/explicit append |
| Implementation | Forward36 adds operator-only default1 source maximum and server-resolved register_accounts commands using the existing pending/audit/ack gate; shared binding loop preserves legacy receipts |
| Focused validation | All32 connection-state tests including8 new registration cases PASS; restricted Production-profile logins, concurrent replay, first published source unchanged, injected partial failure rollback, current scope/expiry, paused/disconnected and multicurrency cases |
| Final combined main | Rebased onto398/306aab1 after396/397 merged; core/doc checks,32 connection tests with explicit API table/private executor/forged registration/limit-history denials and343 frontend unit tests PASS |
| Broad validation | make check/test/build, migration-test (fresh/replay/down-up, failed DDL and retained-history guards, exact bootstrap isolation), ledger-test, locked web-check/test/build/browser and app-integration PASS; no hosted proof claimed |

## Current evidence

Prior375 verified one checking stream/sync1/1/187 review rows/0 journals, separate
from this source-only registration operation. No real financial records
or credentials used in source/testing.

## Remaining work

Exact-head CI and exact source human security review before merge/apply.393/394/
395 remain dependent work;290 metadata/privacy is separate. No source
readiness claim permits second live account or expired Sandbox provider calls.

## PR handoff

Refs392/391; migration36 sealed with1–35 unchanged; no dependency/runtime source
or generated HTTP type change. Existing API cannot create register_accounts through
generic registration;393 must expose only the reviewed preparer and typed recovery.
SQL must be deployed before that API; never roll back to command-unaware runtime
after new commands exist. Default1 preserved; no managed apply or scope increase.
Local tests used installed Go1.26.5/external temporary modfile; tracked1.26.10
unchanged. Docker/loopback gates ran with required sandbox escalation. App identity
is synthetic; browser/SQL success is not actual managed/Plaid/reconciliation proof.
