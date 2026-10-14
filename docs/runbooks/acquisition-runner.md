# Acquisition runner delivery
<!-- stage-runbook: true -->

## Status and boundary

Reviewed implementation in progress for #102, lease ACQUISITION-GO-102. Scope is
[runner contract](../contracts/acquisition-runner.md), new internal/acquisition
files and tests. No live provider, credentials, cloud or migration changes.

## Stage ledger

- Read issue102, AGENTS, merged sync/decoder/runtime contracts and source index.
- Refreshed official Plaid account-filter/pagination sections.
- Coordinator approved bounded runner and durable raw evidence design.
- Agreed sync_context/sync_resume wire with the sole SQL author; connection lane
  confirmed opaque credential binding and memory-only resolver expectations.

## Current evidence

Implemented SQL transaction adapter, cursor runner, pinned HTTP transport and durable
local evidence. `go test ./internal/acquisition -count=1` passed with synthetic
loopback HTTP: exact large USD/JPY/KWD/exponent amounts, complete mutation restart,
commit-loss model, revoked authority, immutable concurrent evidence and redaction.
This first handoff does not claim actual PostgreSQL runner proof; that follows next.

## Remaining work

Prove the implemented adapters together through disposable PostgreSQL, including
actual committed response loss and revoked authority. Run default gates. Hosted encrypted storage, durable
retry scheduler, actual provider qualification and deployment remain separate gates.

## PR handoff

Initial implementation handoff is followed by the actual PostgreSQL proof. Coordinator owns shared composition/scripts/CI; sole SQL author owns
migration13 and context/resume functions. No journals from acquisition.
