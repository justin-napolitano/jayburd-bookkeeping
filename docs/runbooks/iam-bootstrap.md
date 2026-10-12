# Development IAM administrator bootstrap
<!-- stage-runbook: true -->

## Status and boundary

Owner authorized continuation after #42 merged at `ef2675a` on 2026-10-12.
Issue #26, branch `issue/26-iam-bootstrap`. The existing database has no product
schema yet. No financial data or credential retrieval is authorized by this work.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Source review | G24 documents IAM role assignment but not full bootstrap authority | Test an explicit human operator, isolated from application logins |
| Implementation | Optional Terraform operator; default off; six application mocked runs pass | Apply only for a bounded check |
| Cloud attempt | Operator creation succeeded; local SQL connection timed out; TCP 3307 reachability failed | Resume from a network with connector access |
| Cleanup | Operator and its bindings removed; API readback confirms absence; private drift exit 0 | Resume with no elevated identity left behind |

## Current evidence

- PASS: `make infra-test`, 22 mocked runs across five roots. Named additions:
  `bootstrap_disabled_by_default`, `separate_bootstrap_operator` and
  `reject_service_account_operator`. Provider 8.6.0 accepts `database_roles`.
- PASS: optional IAM operator apply; selected Cloud SQL user-type readback.
  API and migrator received no new role membership.
- BLOCKED: psql connection timed out before SQL execution. Proxy v2.26.0
  private Unix listener started, but its upstream connection test exited 1.
  Independent five-second TCP check to the instance connector port 3307 failed.
  This establishes local reachability failure, not its firewall/VPN cause and
  not validity of IAM authentication or SQL administrator privileges.
- PASS: `make check test build` and `make cloud-sql-test migration-test`.
  Isolated restricted migration/replay, API denial, contention, checksum and dirty
  failure tests passed. Docker helper sandbox failure was resolved by approved
  rerun; these tests do not prove Cloud SQL compatibility.
- Private raw operational logs stayed unread. Proxy errors were reduced to fixed
  categories. No SQL bootstrap, migration, identity enrollment or finance occurred.
- PASS: cleanup apply exited 0, human IAM SQL users read back absent and both
  application logins present. Final private Terraform drift check exited 0.
  Both temporary proxy processes exited; no proxy listener remains.

## Remaining work

Resume from an owner-controlled network that permits outbound connector traffic,
or implement a separately reviewed GCP operator runner. Do not open authorized
networks, disable connector enforcement or substitute a privileged application
login. From that environment, recreate the optional operator, establish IAM login,
then check CREATEROLE, database/schema ownership and authority to alter the IAM
logins. Execute existing bootstrap only after those checks. Migrations wait for
successful bootstrap. The owner-only password alternative remains unneeded until
the IAM authority test actually runs.

## Resume commands and cleanup

Use the existing application GCS backend from the [provisioning runbook](cloud-sql-bootstrap.md),
with the source on this branch. Explicitly set `bootstrap_operator_email` to the
owner's lowercase IAM email during the bounded operator apply. The default is null;
keep it null after this unsuccessful check. No administrator input is saved in Git.

Install the reviewed tool with Go 1.26.9:
`go install github.com/GoogleCloudPlatform/cloud-sql-proxy/v2@v2.26.0`.
The operator supplies existing ADC privately. In a dedicated terminal, use a short
private socket path to stay under the operating system path-length limit:

```sh
umask 077
sql_socket_dir=$(mktemp -d /tmp/jb-sql.XXXXXX)
cloud-sql-proxy --auto-iam-authn --unix-socket="$sql_socket_dir" \
  --quota-project=jayburd-bookkeeping-dev \
  jayburd-bookkeeping-dev:us-central1:jayburd-dev
```

Keep raw logs private. In a second owner terminal, set `sql_socket_dir` to that
directory and `sql_operator` to the same IAM email. Query only capability metadata:

```sh
PGPASSFILE=/dev/null psql -X -w -h "$sql_socket_dir/jayburd-bookkeeping-dev:us-central1:jayburd-dev" \
  -U "$sql_operator" -d bookkeeping -v ON_ERROR_STOP=1 \
  -c 'SELECT rolcreaterole FROM pg_roles WHERE rolname=current_user;'
```

This is only the first capability check, not permission to run migrations. Follow
the [bootstrap contract](../contracts/cloud-sql.md#bootstrap-and-migration) for
the remaining authority checks and `db/bootstrap/cloud-sql.sql` invocation.
Stop the proxy after use. Remove the optional operator and connector/login bindings
through Terraform if no SQL objects were created. After a successful bootstrap,
review ownership dependencies before deleting or downgrading it. Never use
ABANDON or leave an unnecessary privileged login to hide a failed cleanup.

## PR handoff

Keep #26 open for managed-service acceptance, pool recovery and restore proof.
Include this runbook in every commit. Private operational logs/state stay ignored.
