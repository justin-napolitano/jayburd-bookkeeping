<!-- stage-runbook: true -->
# Connection service delivery

## Status and boundary

Stage: approved contract and interface implementation, #101.

Owned paths: internal/connections and this contract/runbook. Migration13 belongs
to the SQL lane; server/OpenAPI and integration commands belong to coordinator.

## Stage ledger

Reviewed provider Link/Items/Accounts sections and existing runtime/ingestion
contracts. Defined exact scope, explicit account binding, one-shot remote effects,
manifest-before-secret sequencing and fixed diagnostic boundaries.

## Current evidence

Interface and contract reviewed by coordinator; `make check test build` PASS for early interface stage. Service files remain unstaged while lifecycle proof is built.

## Remaining work

Validation pending implementation. No provider calls, secrets, cloud mutation or
live bookkeeping approval. Infisical adapter and Hosted Link qualification remain
separate gates; synthetic injected stores do not prove hosted secret durability.

## PR handoff

Issue #101; coordinator integrates staged interface and later service commits.
