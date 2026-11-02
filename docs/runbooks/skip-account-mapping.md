<!-- stage-runbook: true -->
# Clear skipped account mappings —387

## Status and boundary

SKIP387, fix/387-skip-account-mapping fromf8ab1e4. Bounded browser selection fix;
no SQL/API contract, source limit, profile, consent, money or provider changes.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live387/391/392 and65 reviewed; exclusive Connections lease; dedicated worktree |
| Source | Existing binding/automatic-account-binding/command recovery contracts, bibliography and391 source record; strict blocker and ledger onChange reviewed |
| Decision | Explicit skip removes the entire local proposal, resets consent, and requires full mapping if selected again |
| Regression | New test failed before fix on retained currency; passes after full proposal removal.343 unit tests PASS |
| Validation | Locked web-check/web-test/web-build/web-browser PASS,26 component browser cases. Core check/test/build and app-integration PASS; required loopback/Docker gates rerun with escalation after sandbox denial |

## Current evidence

Synthetic regression starts with three proposed supported accounts, skips two,
checks consent invalidation, reselects a cleared row to prove partial validation,
then confirms only the intended checking account. No real financial records used.
Existing server authority and exact currency/profile validation remain unchanged.
Existing proprietary setup/ISO4217 assessment applies; no new financial message,
classification, standard mapping or accounting practice introduced.

## Remaining work

Exact-head CI/review pending.391–395 migration/API/picker work remains separate; no hosted
release or additional-account authorization from this fix.

## PR handoff

Refs387/391/104. Frontend only; no migration/generated API changes. Never clear
an unknown remote mutation result through local selection reset.
Local Go uses the existing external temporary1.26.5 adapter; tracked1.26.10
unchanged. Locked frontend dependencies installed, no dependency changes.
