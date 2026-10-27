<!-- stage-runbook: true -->
# Accurate Plaid handoff copy — #349

## Status and boundary

COPY349-HANDOFF; fix/349-plaid-handoff-copy from6be6c48.
Sandbox-only instructions/link/setup-group/restart copy becomes environment-neutral.
No routing/profile/auth/protocol/API/SQL or financial-message change. No actual
provider/cloud/secret/private data, spend or Production activation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |349/315/65 checked;315 proof merged, dedicated bounded UI-copy lease |
| Change |Instructions/link now Open Plaid; existing unit and Sandbox integration selectors updated |
| Validation |check/test/build, web-check/test/build/browser, app-integration, connection-browser-test and production-journey-test PASS;342 unit and26 browser tests |

## Current evidence

315 found misleading Sandbox copy with the Production-capable backend. Runtime
still owns provider environment; the browser gains no selector/authority.
Local Go1.26.5/external temporary modfile/cache used after pinned1.26.10 download
failed403; tracked pins unchanged, canonical exact-head CI required. Locked npm
install reports0 advisories and4 pending lifecycle scripts; no approval added.
No new source/version/currency mapping or ISO message designation applies to
operational UI wording; existing financial contracts and original units unchanged.

## Remaining work

Exact-head CI and merge. No live release readiness follows from
wording;337 decision and316 delivery work remain separate.

## PR handoff

Closes349 after reviewed merge. Text-only product change, no migration/dependency
impact; preserve link security attributes, completion/recovery and existing flows.
