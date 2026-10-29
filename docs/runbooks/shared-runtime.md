<!-- stage-runbook: true -->
# Shared Production runtime transport — #316

## Status and boundary

build/316-shared-runtime, stacked on358 dependency patch. Owner requested final
wiring after the shared database migrated through35. This stage prepares runtime
transport; it does not claim a complete hosted application deployment.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Design | Exact existing instance, explicit Production environment, derived bookkeeping_prod and dedicated Production IAM identities |
| Implementation | API/sync/publisher/webhook derive isolated database together; Sandbox retains bookkeeping; migrator still requires explicit database flag |
| Validation | Core check/test/build and synthetic browser/API/SQL app-integration PASS; exact routing and mixed webhook project tests included |

## Current evidence

Shared Production migration-only transport is already deployed and verified.
Runtime uses the same fixed target with separate login names. No caller DSN,
database or user override is introduced. Adjacent projects/instances and verifier
remain denied. Financial schemas, valuation, API contracts and ISO message
applicability are unchanged; this is operational routing metadata.
Webhook configuration also binds the exact shared instance to its numeric
project9088794002 and dedicated Production secret names; mixed project pins
remain denied. Local checks use Go1.26.5/external temporary modfile; tracked
Go1.26.10 remains unchanged. Hosted credentials/provider access were not used.

## Remaining work

Review runtime source; validate integration and exact image scans. Prepare the
shared-target release validators and infrastructure, remaining worker/webhook
identities and SQL grants, Production key/secret name/version bindings, private
web/API route and signed webhook delivery; preserve paused scheduling. Obtain
concrete release review before cloud activation. No secret retrieval, provider
calls or changes to the already migrated database occur in this branch.

## PR handoff

Refs #316/#337. Shared administrator/capacity/restore scope remains the accepted
tradeoff documented in shared-production-sql.md. Live qualification remains open.
