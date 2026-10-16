<!-- stage-runbook: true -->
# Worker binding lookup

## Status and boundary

Issue #164: the creator's first **Connect account** action now prepares the
per-book delivery binding automatically under [self-service onboarding](../contracts/self-service-connections.md).
`sandbox-book.sql` remains an administrative recovery path, not the routine way
to obtain the binding ID. This helper reads one active binding ID only; it does
not modify SQL, contact Plaid, deploy workers, or enable scheduling.

## Stage ledger

- Reviewed the current self-service connection contract, migration 17, the
  delivery-binding table in migration 15, and the worker release inputs.
- Added a bounded `psql` lookup through `bookkeeping_owner` and a shell wrapper
  that validates two UUIDs, suppresses connection diagnostics and prints only
  one binding UUID.
- Tested the SQL on a disposable synthetic database: active scoped lookup,
  wrong-entity and wrong-book queries passed. Wrapper tests cover successful,
  missing, failed-connection and malformed-ID outcomes without network access.

## Current evidence

`make sandbox-bootstrap-test` passed against disposable PostgreSQL and the
wrapper passed its no-network outcome checks. `bash -n` and `git diff --check`
passed. The repository-wide `make check test build` attempt was blocked before
checks because this machine has Go 1.26.5 while `go.mod` pins 1.26.9; the
official toolchain download returned HTTP 403 even after an unrestricted retry.

## Remaining work

The owner still needs to run this metadata lookup through their existing
development Cloud SQL connection and review the returned UUID. Worker federation
and paused runtime release remain separately gated.

## PR handoff

The branch adds a read-only SQL file, a small validated wrapper, synthetic SQL
and wrapper checks, and this operator instruction. It changes no migration,
application API, cloud resource or credential handling.

## Operator use

Use an existing **development** Cloud SQL proxy and operator IAM `psql` setup,
as documented in [Cloud Shell bootstrap](cloud-shell-bootstrap.md). Verify that
the proxy targets `jayburd-bookkeeping-dev:us-central1:jayburd-dev`; the script
cannot infer the proxy's target from a PostgreSQL session. From the repository:

```sh
scripts/show-worker-binding.sh ENTITY_UUID BOOK_UUID
```

For the book previously shown in the owner's Sandbox UI, the arguments are:

```sh
scripts/show-worker-binding.sh 9781d120-b826-4f77-949e-51ceda2c5e0f 8b09e985-e348-4497-9bca-2b10eaf61417
```

The script prints only the active binding UUID. A missing row means the chosen
book has no active delivery binding with the expected dev worker logins; do not
invent an ID or dispatch the worker release. The operator must have `SET` on
`bookkeeping_owner`. No token, password, DSN or provider value is an input.

## Evidence and limits

The lookup proves only the selected row's metadata at query time. It does not
prove current consent, successful token persistence, hosted worker IAM, or Plaid
transactions. The owner reviews the returned ID before pinning a paused worker
release. Scheduler activation and controlled Sandbox acquisition stay separate.
