# Sign-in bootstrap availability delivery
<!-- stage-runbook: true -->

## Status and boundary

#289, AUTH289-20261024T0530Z, fix/289-auth-bootstrap from c1cc86e.
No identity/controller/Firebase verification rule, SQL/API wire, provider policy,
cloud or secret change. [Decision](../contracts/auth-bootstrap.md). Production
memory-only verified identity and MFA remain authoritative.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Current issue/lease, browser-auth/identity-setup and rehearsal runbook reviewed |
| Implementation | Distinct loading, retryable unavailable, invalid and ready bootstrap; source/attempt and abort fences |
| Component/unit | 333 frontend tests PASS, including delayed valid configuration,503/network/timeout retry, invalid fetched/supplied config, unmount/source-change and old-session teardown |
| Browser |25 browser regressions PASS; new desktop/mobile keyboard retry case uses actual production bootstrap/SDK initialization with fictional public identifiers, no credentials, blocked external requests and zero attempted external calls |
| Core/web build | make check/test/build and web-check/web-build PASS; tracked Go1.26.10 and all dependencies unchanged; cached local Go1.26.5 temporary modfile only |
| Full stack | make app-integration PASS: real browser -> production Next bridge -> Go -> restricted disposable PostgreSQL, setup, explicit posting and dropped-commit recovery |
| CI follow-up | PR309 frontend unit/browser stages PASS, but exact-image smoke retained the old pre-hydration unconfigured heading assertion. Updated it to require loading/no premature credential/invalid state; all config503/no-store, non-root/read-only/no-finance/no-external-network controls retained |
| Exact image local limit | Docker build attempted with unchanged image/package pins, blocked by local Alpine registry TLS trust before runtime proof. No TLS/package/security gate weakened. Exact x86 image proof remains canonical CI; standalone local shell assertions are separate partial evidence |
| Corrected smoke local proof | Production standalone shell checks PASS: loading/no credential form, no-store/frame denial and missing-runtime-config503; shell syntax/whitespace PASS. This does not claim non-root/read-only image acceptance |

## Current evidence

Actual Tartan flow exposed a transient missing-configuration warning despite
working public configuration. This slice does not establish why the original
session remounted or prove provider logout. No real credential/session observed.
Existing controller/Firebase test suites remain intact; no auth adapter or route
bypass was introduced. Required exact-head canonical CI is separate from local
checks. ISO/currency/financial contracts and migration history are unchanged.

## Remaining work

Staged runbook/whitespace gate, review and PR/CI. Managed deployed acceptance
remains separate; no claim that synthetic identity proves managed MFA.

## PR handoff

Refs #289/#88/#90. No migration, OpenAPI or dependency change. Sensitive auth
composition requires review before release; no real-account activation follows.
