<!-- stage-runbook: true -->
# Connection service delivery

## Status and boundary

Stage: bounded implementation and synthetic lifecycle proof, #101.

Owned paths: internal/connections and this contract/runbook. Migration13 belongs
to the SQL lane; server/OpenAPI and integration commands belong to coordinator.

## Stage ledger

Reviewed provider Link/Items/Accounts sections and existing runtime/ingestion
contracts. Defined exact scope, explicit account binding, one-shot remote effects,
manifest-before-secret sequencing and fixed diagnostic boundaries.

- Interface commit `6cbf03b` froze requests before coordinator HTTP implementation.
- Implemented service, restricted SQL adapter and opt-in fixed Sandbox HTTP adapter.
- Added independent HTTP wire/association/consent bounds and lifecycle fault tests.
- Added stale-context regression at four pre-I/O boundaries: exchange, manifest,
  secret write and secret readback. Each asserts exact prior operation counts and
  no subsequent external operation or active reference publication.
- Composed real SQL with service for activation, revoked actor, lost secret reply,
  lost actual claim COMMIT reply, two-account reconnect/rotation/removal and local
  account disconnect. Provider and secret store remain original synthetic adapters.

## Current evidence

`make check test build` PASS and focused race suite PASS. The disposable proof
passed 18 top-level groups: six service/PostgreSQL groups and twelve unit/HTTP
groups, plus their named subtests. Disposable PostgreSQL
proof log: `/private/tmp/jayburd-connection-service-proof.log`; focused unit/HTTP
log: `/private/tmp/jayburd-connections-unit.log`; default gates:
`/private/tmp/jayburd-connection-default.log`. No provider network was used.
Follow-up `TestConnectionStaleContextStopsIO` passes all four injected stale
context cases under the race detector; focused suite now has thirteen unit/HTTP
groups. Its log is `/private/tmp/jayburd-connections-stale.log`; follow-up default
gates are `/private/tmp/jayburd-connection-followup-default.log`. The earlier
eighteen-group PostgreSQL proof predates this added unit group; coordinator reruns
the combined proof against the final SQL context/Item-generation guard.
The coordinator's `test-connection-service.sh` harness uses
`connection_service_template` and `-connection-service-proof`; migration13 is
integrated separately from the SQL lane, never committed from this branch.

## Remaining work

No real provider calls, secrets, cloud mutation or live bookkeeping approval.
Concrete reconnect is rejected before effects pending verified update-session
qualification. Infisical dynamic writer permissions, synchronization acknowledgement
and version visibility remain separate design/qualification gates. Unknown exchange,
rotation, missing secret and ambiguous removal deliberately remain unresolved.
Synthetic injected stores do not prove hosted secret durability. Coordinator is
integrating the SQL lane's additional current Item-generation context guard.

## PR handoff

Issue #101; coordinator integrates staged interface and later service commits.
