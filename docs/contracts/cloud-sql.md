# Cloud SQL connection and migration contract

## Decision

Implement #26 without changing merged SQL. Terraform owns a PostgreSQL 17 Enterprise
single-zone instance and API/migrator IAM users. Owner bootstrap creates reviewed
NOLOGIN roles and a `deployment` schema; a distinct migration binary applies embedded,
checksummed SQL. Runtime startup never migrates. Sources and limits are in the
[GCP source register](../research/gcp-sources.md).

This is infrastructure/internal configuration, not an ISO 20022 financial message.
ISO currency and accounting contracts remain unchanged. `deployment` holds only
migration-control metadata, never journal or provider payloads.

## Configuration and authority

API selects `-database-mode=cloudsql -cloud-sql-instance=PROJECT:us-central1:INSTANCE`
alongside existing managed-mode flags. Project must equal the identity project;
JAYBURD_DATABASE_URL presence is rejected in connector mode. URL mode remains
compatible. Health mode does not construct clients or read a database URL.

Connector 1.25.3 uses metadata-only API and IAM-login token sources, lazy refresh,
public-IP TLS tunnel and automatic IAM authentication. No ADC discovery, key file,
password, private network or host override is accepted. pgx TLS is disabled only
inside this authenticated encrypted tunnel; no fallback to direct TCP exists.

API identity is derived as `jayburd-dev-api@PROJECT.iam`, database `bookkeeping`.
Pool cap is 5, minimum 0; connections age out at 30 minutes plus up to five minutes
jitter and idle out at five minutes. Dial/refresh bounds are 15 seconds; startup
is bounded to 20 seconds. Existing shorter request/transaction deadlines still
apply; cold connection failure returns unavailable rather than extending posting.
Migrator derives its separate login, uses at most two connections and permits
60-second SQL statements with a 10-second SQL lock timeout.

Terraform enables IAM authentication, encrypted connector enforcement, no authorized
networks, max_connections 50, initial 10 GiB SSD with growth capped at 20 GiB,
daily backups with seven retained copies and seven-day PITR. SQL statement/error
statement logging is suppressed; redacted application metrics remain separate.
No SLA is claimed for shared-core capacity. Cloud IAM client/login roles are
project-scoped for two accounts; SQL role grants restrict authority within the
selected instance. Broader inherited project grants require deployment review.

## Bootstrap and migration

Migration 8 requires the owner-operated bootstrap to precreate the safe NOLOGIN
`bookkeeping_publisher` and `bookkeeping_observer` capability roles. Rerun the
reviewed bootstrap before applying migration 8 to an existing hosted database;
the restricted migrator still has no CREATEROLE privilege. This does not provision
worker IAM logins, observer bindings or service permissions. Those remain a
separate hosted-worker enrollment gate. API readiness rejects worker-role membership.

Owner uses [bootstrap SQL](../../db/bootstrap/cloud-sql.sql) with existing API and
migrator IAM role identifiers. Administrator must own the database/schema and have
CREATEROLE plus sufficient authority to alter the two logins; this authority is
not granted to the migrator. Bootstrap rejects unsafe attributes/inherited roles,
grants only CONNECT to API, grants migrator database CREATE and explicit SET ROLE
bookkeeping_owner, and gives migrator access to deployment. API has no DDL/table
or enrollment grant. Connection limits are 20 API and 2 migrator across revisions.
Actual Cloud SQL administrator compatibility remains a live gate.

An explicit `bootstrap_operator_email` application-root input may provision one
human IAM SQL operator with `cloudsqlsuperuser` membership and Cloud SQL
client/login IAM bindings; default null creates none. API/migrator resources
remain unchanged. This operator is a development administration identity, never
a runtime identity. The owner authorized evaluating this G24 path after #42.
Membership does not prove CREATEROLE, ownership or authority to alter other IAM
logins; verify those before bootstrap. Password administration is the alternative,
with owner-only credential injection if required.

For this owner-operated check only, the pinned Cloud SQL Auth Proxy v2.26.0 may
consume existing ADC internally with automatic IAM authentication. Use a private
Unix-socket directory (0700), not a network listener; psql ignores startup files
and never prompts for a password. Do not expose tokens, credential files or raw
provider errors. The application connector remains metadata-only. Stop the proxy
after the check. Failed checks must not leave an unnecessary administrator: remove
the optional operator through Terraform after confirming it owns no objects.
After successful bootstrap, role/schema ownership must be transferred or retained
explicitly before removing the login; never silently abandon a privileged user.
`database_roles` is write-only: zero drift is not proof of SQL membership. Verify
SQL privileges separately, and retain the explicit input on later applies while
the operator is needed. See [IAM bootstrap evidence](../runbooks/iam-bootstrap.md).

`cmd/migrate` defaults to offline `-action=verify`. Explicit `-action=up -instance=...`
uses IAM transport and golang-migrate 4.20.1; no down/force/target command exists.
It rejects invalid artifacts before network access and unsafe migrator privileges,
serializes runners with a database advisory lock, checks stored artifact hashes,
registers attempted files before execution and uses golang-migrate's separate
version/dirty-state lock. `deployment.migration_artifact` stores filename, SHA256
and registration time; it is not a claim that every registered version succeeded.
The library owns `deployment.schema_migrations` as the applied/dirty authority.

`cmd/migrate -action=status -instance=...` uses the same restricted IAM connector
but only a read-only repeatable-read transaction to check current clean version
and exact registered hashes. It does not create metadata or run migrations. The
[status release contract](migration-status.md) defines its distinct reviewed
`verify` operation and receipt; the human operator receives no additional access.

Resealed registered SQL is rejected. Crash after artifact registration is replayable
with the identical package; dirty SQL execution blocks retry until owner-reviewed
repair. Migration SQL remains immutable. A signal requests stop between migrations;
a running statement is bounded by its timeout. Errors are fixed/redacted, never
raw provider errors or SQL text. Digest promotion and migration Job invocation
belong to #27; this slice does not invoke GCP.

## Acceptance and recovery

`TestConnectorConfiguration`, `TestConnectorRejectsAmbiguity` and
`TestCloudSQLSelection` prove transport selection and bounds without credentials.
`TestArtifactIntegrity`, `TestRestrictedMigration` and `TestMigrationContention`
cover local artifacts, non-superuser migration/replay, runtime denial, durable
checksum drift and overlapping execution. Terraform's application tests assert
recovery/capacity/transport configuration. Existing migration and ledger suites
remain required. Passing these tests does not establish IAM token/TLS acceptance.

Denial evidence must match the expected SQLSTATE, not merely any failure:
`42501` for forbidden DDL/table reads/owner role, and `P2101` for an unknown
principal at the workspace entrypoint under its required repeatable-read isolation.
Connection failures, missing objects and the `P2100` isolation precondition do
not demonstrate authorization. Direct-table probes use zero-row reads.

Maximum planned demand including overlapping API/publisher/consumer revisions is
28 runtime connections, plus two migrator and five administrative = 35 below the
configured 50. Publisher/consumer pools are future #7 work. Live shared-core login
latency, actual pool recovery, backup restore/PITR, exact restored journal totals
and replay remain #26/#12 cloud evidence. Restore into a separate temporary instance;
never replace/delete the original automatically. Record recovery time/data-loss
window and retire only the drill instance after review. No backup artifact or
Terraform plan/state enters agent context.
