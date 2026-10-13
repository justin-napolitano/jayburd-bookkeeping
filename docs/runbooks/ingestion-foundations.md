# Ingestion foundations

<!-- stage-runbook: true -->

## Status and boundary

Build on merged #77 (`380bb0d`). Implement the local receipt-only observer; specify
durable import boundaries and illustrate the review queue. No hosted worker,
provider connection, import persistence or live accounting authority.

## Stage ledger

- Reviewed #7 dependencies: #4/#5/#63 closed completed with merged evidence.
- Two agents own observer SQL and durable import contract. Root owns synthetic
  review prototype after the session agent-thread limit prevented a third worker.
- Corrected existing browser screenshot path to Playwright-owned outputPath;
  Linux CI could not write the hardcoded macOS `/private/tmp` path.

- Root extended owner-operated bootstrap to create two NOLOGIN worker capability
  roles before restricted migrations; no worker login/binding or remote action.
  API readiness rejects either worker role, with named regression cases.

- Independent observer review corrected terminal retry status and commit-time
  invariant-check privileges. Coordinator approved a liveness correction: fixed
  dispatch creation age governs redispatch, so repeated uncertain/ALREADY_EXISTS
  replies cannot postpone recovery indefinitely; active leases stay protected.

## Current evidence

`make web-check web-test web-build web-browser` passed (148 unit tests, nine
browser journeys); screenshot now uses the test-owned output directory. Reviewed
the captured general-ledger screen: amounts, original units and actions are visible
at desktop width. The standalone import prototype also passed browser interaction
checks and desktop/mobile visual inspection; [evidence](import-review.md).

Combined `make check test build migration-test ledger-test cloud-sql-test
observer-test` passed with migration8, bootstrap and schema-version fixtures.
`TestRestrictedMigration` proves deployment without CREATEROLE; runtime readiness
rejects both worker-role memberships. All 19 observer tests passed.
`make app-integration` passed through production Next bridge, Go and PostgreSQL;
temporary ports39181–39183 preserved the existing review server, then all three
harness files were restored exactly. Identity remains a synthetic fixture. #77 full application integration passed; its browser screenshot failed
on Linux. Existing auth fixture synchronization passed that CI run.

## Remaining work

Publish PR and release leases. Next implement the bounded Go publisher/consumer
transaction adapter with fake transport, then separately qualify hosted identity
and queue delivery. Import storage and posting need their own implementation
acceptance; this contract/prototype does not persist imported data.

## PR handoff

Combined validation passed. Migration8 is forward-only; migrations1–7 are unchanged.
Before hosted migration, owner reruns reviewed bootstrap for the two safe NOLOGIN
roles. No cloud command, worker login, service binding or provider was activated.
Completed foundation/design issues #72/#73/#74 reconciled with merge evidence;
#7/#8 and their parent epics remain open.
