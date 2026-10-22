# Owner-operated Sandbox webhook bootstrap

## Decision and scope

Issue #271 preparation; coordinator approved this operational profile2026-10-22.
Reuse migration14's existing `bookkeeping_webhook` capability and
`integration.set_webhook_ingress`; no product DDL, rewritten migration or startup
migration. SQL role wiring and ingress enrollment are one serializable transaction.
Cloud IAM/user provisioning is a separate reviewed infrastructure operation.
Neither this contract nor the wrapper authorizes cloud execution or source consent.

`db/bootstrap/webhook-login.sql` is executed by an owner-authorized SQL operator
with existing CREATEROLE/ADMIN authority over the target/capability, database grant
rights and SET permission on `bookkeeping_owner`. It checks session=current user
before the explicit owner switch. The runtime never receives those capabilities.
No email/caller actor substitutes for authenticated SQL identity.

## Explicit interface and authority

`scripts/bootstrap-webhook.sh` has no default operation. `--check` validates metadata
only; `--apply` requires a separately owner-created authenticated local Unix socket.
Both require explicit `--project`, `--operator-user`, `--socket-dir`,
`--operation-key`, `--enabled`. Only project `jayburd-bookkeeping-dev` is supported;
SQL derives `jayburd-dev-webhook@jayburd-bookkeeping-dev.iam` and independently
requires database `bookkeeping`. There is no runtime-login override, DSN, password,
secret version, cloud CLI invocation, IAM token lookup or provider call. The operator
login is a connection identity, not an application actor. Socket setup and operator
authentication remain owner operations under the existing Cloud SQL procedure.

The wrapper clears inherited libpq configuration, uses `psql -X -w`, fixed port5432,
explicit database/login/socket, rejects host-list commas, five-second connection timeout and no password or
service file. It suppresses SQL/results/server diagnostics and emits only a fixed
success or failure. Metadata check is not authentication or permission evidence.

Before grants, SQL rejects unsafe LOGIN/capability attributes, owner/API/sync/other
bookkeeping, predefined `pg_*` and privileged transitive memberships, ADMIN or SET options from any
grantor, database CREATE/TEMP, schema CREATE, product table/sequence privileges,
and unintended function EXECUTE. The sole permitted inherited function is
`integration.accept_verified_webhook(text,text,text)`. Direct product function
ACLs and schema/database delegation options are rejected, even for the intended
function. Existing normal USAGE on the integration schema through the capability
is expected. Unknown extra roles with no elevated/app privilege are not represented
as approved application capabilities; actual Cloud SQL IAM role compatibility
remains a hosted acceptance gate.

Then grant only `bookkeeping_webhook` with INHERIT TRUE, ADMIN FALSE, SET FALSE;
CONNECT; search_path `pg_catalog`; connection cap4. This supports at most one old
and one new revision, each max1 instance × two pooled connections. Cap4 alone
does not enforce Cloud Run instance policy. No privilege is silently revoked to
make an unsafe account appear valid. Unexpected privilege causes atomic failure.
SQL uses statement10s/lock2s/idle-in-transaction15s limits; no automatic retry.

## Ingress enrollment, replay and revocation

After role validation/grants, `SET LOCAL ROLE bookkeeping_owner` invokes only:
`integration.set_webhook_ingress(actual_login,'sandbox',stable_key,enabled)`.
The function's durable key is `(session_user,idempotency_key)` and its normalized
body includes login, environment and enabled. Exact same-key replay returns its
existing receipt without incrementing ingress revision. Changing enabled with the
same key raises P2802 and rolls the entire bootstrap transaction back.

Therefore disable uses a **new stable operation key**, for example `release-271-disable-1`;
retry that exact disable with the same key until known. A later enable uses another
new key. A post-call owner check confirms the current exact login/enabled state; replaying an older enable after a later disable fails without re-enabling. Owner identity changes also change the receipt key, so the same textual
key through another operator is not an exact replay. Operators retain operation
identity and inspect only approved redacted outcomes. An uncertain COMMIT is not a
successful bootstrap; owner may replay identical inputs, never invent a new key to
resolve uncertainty. The wrapper does not retry automatically.

Disabling ingress revokes acceptance through the existing SQL mandate, leaving
role wiring and immutable historical receipts intact. It does not delete provider
Items, revoke source consent or change private API/workers. It requires the login
to remain safe; preexisting unsafe authority blocks bootstrap and needs a separately
reviewed owner repair, rather than executing a mixed-capability operation. For emergency acceptance stop,
an independently authorized operator can use the existing serializable owner
`integration.set_webhook_ingress(actual_login,'sandbox',NEW_STABLE_DISABLE_KEY,false)`
boundary directly: its disabled branch does not require the runtime login to pass
its enabled-role check. This is a separately scoped owner operation, not a wrapper
fallback. It does not repair unsafe grants; quarantine and repair remain separate.
No live emergency operation is authorized or executed by this preparation.

## Sources, standards and alternatives

Reviewed2026-10-22, selected full relevant PostgreSQL17 sections:
[GRANT](https://www.postgresql.org/docs/17/sql-grant.html), role membership
INHERIT/SET/ADMIN and grantor-specific grants;
[ALTER ROLE](https://www.postgresql.org/docs/17/sql-alterrole.html), connection
limits, role configuration and CREATEROLE/ADMIN constraints;
[psql](https://www.postgresql.org/docs/17/app-psql.html), -X/-w, variables and
connection parameters; [Predefined roles](https://www.postgresql.org/docs/17/predefined-roles.html),
Table21.1 and the following file/program-access paragraph. These predefined roles
can grant powerful file/program/data access without SUPERUSER or BYPASSRLS;
attribute checks alone do not establish safety. The bootstrap rejects their
transitive membership. The synthetic negative fixture grants
`pg_execute_server_program` and checks rejection without executing a server
program, file read or COPY operation. Inspected migration14's actual key/body/revision behavior,
`db/bootstrap/worker-logins.sql`, [Cloud SQL](cloud-sql.md) and
[managed webhook runtime](webhook-managed-runtime.md). Public manuals establish
mechanisms; project choice is fixed development metadata, fail-closed ACL checks,
bounded budgets and atomic enrollment. No hosted administrator compatibility was
established by public documentation or synthetic superuser setup.

This is internal operational provisioning, not an ISO20022 financial message or
an audit-data export. Existing proprietary notification and ISO4217/date mapping
remain unchanged; no amount, currency or accounting semantics are introduced.
No ISO conformance claim. The sources do not approve public exposure or live use.

Alternative: reuse worker bootstrap and borrow sync authority. Rejected because
webhook runtime forbids mixed capabilities. Alternative: startup auto-enrollment
or implicit psql environment. Rejected because it couples runtime to owner grants
or selects an unreviewed database. An explicit standalone owner command adds an
operator step but preserves the existing bootstrap/runtime boundary.

## Acceptance and limits

`bash scripts/test-webhook-bootstrap.sh` uses pinned PostgreSQL17/migrate images,
private disposable network and fictional roles. It proves exact replay/revocation,
actual login inheritance, SET/DDL/table/owner-function denial, unsafe role/direct
ACL/grant-option rejection, unchanged mandate after rejection and inert metadata
validation. Fixed SQLSTATE42501 catches forbidden operations; connection failures
are not accepted as permission proof. Existing migration/ledger permission gates
remain required. Production, actual Cloud SQL owner authority and hosted ingress
qualification remain open; tests do not renew expired source consent.
