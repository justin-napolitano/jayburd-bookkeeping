# Connection and acquisition delivery

<!-- stage-runbook: true -->

## Status and boundary

Implement #101 enrollment and #102 acquisition from merged #105 `9fd0669`.
Three isolated agents own SQL, enrollment service and acquisition runner; root owns
HTTP, OpenAPI, CI and combined proof. Only original synthetic fixtures and injected
transports run. No real credentials, provider calls, cloud changes or deployment.

## Stage ledger

- Migration 13 adds explicit connection authority, owner-configured worker mapping,
  one-shot lifecycle claims, reference-only credential publication and verified
  account/source binding. Merged migrations 1–12 are unchanged.
- Go enrollment and optional authenticated HTTP routes implement memory-only handoff,
  replay/reconciliation, explicit binding, account disconnect and Item-wide fencing.
- Runner captures private immutable evidence, performs bounded account-filtered sync,
  decodes exact amounts and publishes complete sets without posting journals.
- Independent review found a stale private credential-context gap. Fresh generation,
  expiry and sibling checks plus exclusive Item lifecycle claims now prevent stale
  external effects; SQL and Go regression tests cover the fix.
- Combined proof fixture corrections enforce private directory mode and the complete
  required provider schema. The decoder correctly rejected the incomplete fixture.

## Current evidence

- `make check test build`: PASS, including Go race tests and migration checksums.
- Migration, restricted Cloud SQL roles, ledger, observer, source registry, durable
  import and sync-state proofs: PASS. Connection SQL: 12 groups plus 5 subtests PASS;
  acquisition: 13 groups plus 9 subtests PASS against final migration 13.
- Final migration, enrollment service PostgreSQL/unit proof and combined
  `TestProviderEnrollmentAcquisitionFlow`: PASS on the integrated implementation.
- `make web-check web-test web-build web-browser`: PASS; 184 unit tests and 13 browser
  journeys. Generated OpenAPI types and final TypeScript check PASS.
- `make app-integration demo-test`: PASS; four demo journeys include 3,000 imports,
  explicit source-linked posting and 1,000 exact-cent postings. Alternate test ports
  preserved the owner's existing preview; committed defaults restored.
- Workflow actionlint: PASS. Independent SQL/service/HTTP/runner reviews completed.

`TestProviderEnrollmentAcquisitionFlow` composes authenticated HTTP → enrollment →
immutable synthetic secret store → explicit binding → actual sync adapter → SQL
review. Its independent expected sum is 0.30; replay fetches once, journal count
stays zero, disconnect blocks acquisition and revoked connect authority blocks reads.
`TestConnectionStaleContextStopsIO` checks four external-I/O boundaries. Detailed
proofs: [SQL](connection-sql.md), [service](connection-service.md),
[runner](acquisition-runner.md).

## Remaining work

Connection UI (#104), verified webhooks (#103), hosted dispatch/provider composition,
scoped Infisical writer and immutable-version propagation remain separate. Concrete
HTTP reconnect is unsupported until update-session proof is qualified; injected
provider tests establish only the owned lifecycle. Depository binding only; broader
account types require their source policy. Sandbox qualification (#88), independent
fault campaigns and measured scaling remain open. No official-books approval.

## PR handoff

Review migration 13, optional API wiring and the four new isolated CI proof targets.
Refs #101 and #102; do not close provider qualification or live-cutover gates.
