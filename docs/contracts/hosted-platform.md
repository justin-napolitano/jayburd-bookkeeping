# Hosted Sandbox platform composition

## Decision and boundary

This slice composes the existing acquisition protocol with real dependency adapters.
Cloud SQL remains authority; private Cloud Run publisher/sync services use Scheduler
and Cloud Tasks. [Worker configuration](hosted-workers.md), [evidence storage](hosted-evidence.md),
[provider vault](provider-vault.md) and [API composition](hosted-connections.md) own
their interfaces. No new accounting message or migration is introduced. Existing
ISO mappings remain unchanged; infrastructure identifiers are not financial messages.

The new workers Terraform root includes privileged bootstrap IAM and optional
runtime resources. It is owner-operated until bootstrap/runtime promotion are
separated and reviewed. Foundation adds a dedicated private workers-state bucket;
application deployer access remains confined to runtime state. Build publication
adds a tested worker candidate, not permission to apply bootstrap infrastructure.
No new deployment federation or broad secret administrator grant is introduced.

## SQL setup and activation order

Existing migrations through15 and safe capability roles must be present. Owner
provisions the two IAM logins through Terraform, then runs
[worker-logins.sql](../../db/bootstrap/worker-logins.sql) through the established
private Cloud SQL operator procedure with identifier-only `publisher_user` and
`sync_user` variables. Expected usernames are `jayburd-dev-publisher@PROJECT.iam`
and `jayburd-dev-sync@PROJECT.iam`. The script validates both before mutation,
rejects privileged/cross-capability membership, grants inherited execution without
SET ROLE or role administration, sets search_path to pg_catalog and caps each
login at four connections across overlapping revisions. It grants no table access,
DDL, ledger ownership or business enrollment. No password or runtime migration.

Owner separately enrolls the current service principal/import membership,
`integration.set_connection_worker`, then `integration.enroll_sync_delivery`
using the reviewed actor/entity/book and idempotency contracts. Never insert directly
into their tables. Keep Scheduler paused until exact binding, private OIDC callers,
Infisical propagation and Sandbox execution readiness have been verified. One runtime
publisher binding serves one selected book in this first release; multi-book fan-out
and linear scalability remain #89 work.

Pools: API five per instance, worker two each. With overlapping configured API
revisions (20), two overlapping worker revisions each (8), migrator (2) and five
operator connections, modeled demand is35 against max_connections50. This is a
configuration budget, not a measured capacity claim.

## Acceptance and missing evidence

`test-worker-bootstrap.sh` uses PostgreSQL17 and a non-superuser administrator to
prove replay, inherited capability, SET/DDL denial and unsafe membership rejection.
Mocked Terraform tests check exact IAM destinations, paused/default-disabled
services, digest-pinned configuration and state separation. These do not prove
Cloud SQL's administrator behavior, caller JWTs, Cloud Tasks delivery or secret sync.

The [Sandbox diagnostic](plaid-sandbox-qualification.md) is opt-in and partial.
Actual Link UI/association, dynamic pending-to-posted, webhook delivery, persisted
cursor-to-review consistency and zero automatic journals still require hosted
qualification. Frontend Link loading, reconnect support and automated worker
promotion remain open; no production institution or official-book approval follows.
