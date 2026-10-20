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
2. **Database command — pending.** Migration 22 must extend the restricted binding
   function and prove forward/replay/down, role isolation, conflict and failure
   atomicity using disposable synthetic PostgreSQL only.
3. **Service and browser review — pending.** Go/OpenAPI/browser changes must
   preserve old existing-account requests, propose distinct accounts for every
   eligible account and keep unsupported types visible and unbound.
4. **Integrated evidence — pending.** Focused SQL/Go/UI tests and a synthetic
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

## Remaining work

Implement stages 2–5, run the issue-specific and repository gates, and record
bounded results here. Hosted success must omit provider identifiers and payloads.
Live institutions remain out of scope.

## PR handoff

The implementation PR must link #209, list validation and migration impact, state
remaining unsupported account types, and require migration-before-runtime release.
Deployment success alone does not close #88 or approve live bookkeeping use.
