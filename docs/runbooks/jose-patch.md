<!-- stage-runbook: true -->
# API dependency patch — #358

## Status and boundary

fix/358-jose; minimal indirect go-jose/v4 update4.1.4→4.1.6. No migration,
financial message, auth policy or credential change. Owner requested this fix
before remaining shared Production runtime wiring under316.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Finding | API image scan40806816655 reports CVE-2026-116587/HIGH |
| Investigation | Independent source trace confirms Firebase/storage→gRPC/xDS→SPIFFE bundle→go-jose; public attacker route unproven |
| Patch | Exact existing dependency patch; retain upstream RSA limits and private-key rejection defaults |
| Validation | Focused identity/observer/webhook/server and core checks running; fresh API scan pending |

## Current evidence

See identity-sources I07. Existing synthetic auth tests preserve ordinary RSA
verification, invalid signatures/claims rejection, revocation and TOTP boundaries.
Dependency presence alone is not evidence of an exploited deployed endpoint.
Local validation uses Go1.26.5/external temporary modfile; tracked1.26.10 retained.

## Remaining work

Finish core and focused verification, independent candidate review, then exact
API image runtime/scan. Human security review applies before release of sensitive
changes. Shared SQL runtime configuration and live qualification remain separate.

## PR handoff

Refs #358/#316. No SQL migration impact. Do not close358 before patched image
scan evidence. No real provider requests, secrets or personal financial data.
