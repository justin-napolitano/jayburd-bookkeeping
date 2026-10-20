<!-- stage-runbook: true -->
# Automatic account binding delivery

## Status and boundary

Issue #209 owns automatic account creation for the exact cash-like depository
allowlist. Issue #210 separates credit/loan liability policy. AB01 records Plaid's
six top-level types and depository subtypes. No provider call, journal, opening
balance or live-account approval is part of this delivery.

## Stage ledger

1. **Issue and decision — complete.** The accepted contract selects the explicit
   subtype allowlist and atomic account creation inside the existing idempotent
   binding command over pre-creation or a two-command workflow.
2. **Database command — complete.** Migration 22 extends the restricted manifest
   and binding functions. Fresh install, replay and latest down/up passed with the
   exact pinned locally cached images. Focused SQL tests prove three-account
   creation, same-key replay, unsupported-subtype rollback and zero journals.
3. **Service and browser review — complete.** Go/OpenAPI/browser changes preserve
   existing-account requests, carry bounded subtype metadata, propose distinct
   accounts for every eligible account and keep unsupported types visible.
4. **Integrated evidence — in progress.** Focused SQL/Go/UI tests pass. A synthetic
   browser→API→PostgreSQL journey must prove three accounts, response-loss
   recovery, distinct active streams and zero automatic journals.
5. **Hosted acceptance — pending.** Migration must precede matching runtime. The
   owner will repeat a fresh fictional Plaid Sandbox enrollment after deployment.

## Current evidence

- #209 and #210 record the separated implementation and accounting-policy scopes.
- `docs/research/automatic-account-binding-sources.md` records the selected Plaid
  type/subtype schema, product-matrix review, project interpretation and limits.
- `docs/contracts/automatic-account-binding.md` records the accepted request,
  classification, atomicity, recovery, standards and acceptance boundaries.
- `TestConnectionAutoAccountBinding` and `TestConnectionAutoAccountAtomicity`
  pass against disposable PostgreSQL with migration 22.
- The connection-service suite passes, including legacy credential, lifecycle,
  cancellation and HTTP adapter cases. Frontend unit/type/OpenAPI checks pass with
  284 tests, including three distinct automatic proposals and an unbound credit.

## Remaining work

Complete the browser/API/PostgreSQL journey and repository gates, then implement
stage 5. Hosted success must omit provider identifiers and payloads. Live
institutions remain out of scope. Docker Hub's redundant pull returned EOF during
local validation; the same exact pinned digests were already cached and executed.

## PR handoff

The implementation PR must link #209, list validation and migration impact, state
remaining unsupported account types, and require migration-before-runtime release.
Deployment success alone does not close #88 or approve live bookkeeping use.
