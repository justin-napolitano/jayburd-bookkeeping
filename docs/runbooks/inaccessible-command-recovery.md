# Inaccessible saved-command recovery
<!-- stage-runbook: true -->

## Status and boundary

Issue #133 hosted acceptance found an authenticated owner blocked before first-book
setup by an older unacknowledged command whose current scope authorization fails.
Add one explicit recovery action for the owning principal. It returns only whether
work was cleared, never the command, scope, request, result, or financial data.

## Stage ledger

- PostgreSQL locks the active principal and pending command in a serializable
  transaction, reuses current command authorization, and permits clearing only
  the existing not-found/permission-denied cases.
- Prepared work is cancelled and then acknowledged; committed or cancelled work
  is acknowledged. Accounting functions are never called and immutable history
  plus cancel/acknowledge audit records remain.
- The API and same-origin bridge expose `POST /v1/commands/pending/clear` with an
  exact empty object. The UI requires an explicit click and reloads recovery and
  workspace state only after confirmed success.
- PR #145 merged as `cacc5a0`. Owner-dispatched build 38809806298 passed all
  validation and published matching API, migrator, web, and worker candidates.
- Migration-only release PR #146 merged and run 38813529384 completed
  successfully, applying migration 16 without changing application services.
- Matching application release PR #147 and run 38815405193 completed, but hosted
  acceptance exposed a stale-shell defect: the root response retained the prior
  release's asset map under `Cache-Control: s-maxage=31536000` even though the
  current content-hashed bundle contained the recovery control.
- The root route now renders dynamically and the candidate-container gate
  requires `no-store`; content-hashed static assets retain their normal cache
  policy. This prevents an authenticated browser from pinning obsolete UI code
  across releases.

## Current evidence

- `make migration-test` passed fresh, replay, down/up, and dirty-state proofs.
- `make ledger-test` passed the full PostgreSQL suite after advancing the schema
  compatibility sentinel to clean migration 16. Recovery tests also reject
  empty, malformed, and nonempty bodies through the owned error contract.
- `make check test build` passed. All 261 web unit tests, the production build,
  and all 17 browser tests passed, including explicit denied recovery and
  restored starter-book setup.
- The cache correction repeated all 261 web unit tests and all 17 browser tests.
  The production route manifest marks `/` dynamic, and a loopback production
  response returned `Cache-Control: private, no-cache, no-store, max-age=0,
  must-revalidate` while referencing content-hashed static assets.
- `make app-integration` passed the synthetic PostgreSQL, API, and web
  composition gate.
- Build 38809806298 passed its complete reusable validation plus per-image
  runtime, vulnerability, and publication gates.

## Remaining work

Hosted scope denial remains fail-closed; no private payload or log was inspected.
The non-cacheable web release and owner acceptance remain.
The local candidate-container rerun was blocked before application startup by
an Alpine mirror certificate/index refresh failure; CI remains the independent
candidate-image authority for that gate.

## PR handoff

Migration execution is complete. The next handoff is a web release containing
the shell cache correction. Hosted acceptance is the owner explicitly clearing
inaccessible work and preparing the starter book without exposing the old command.
