# Cloud Shell SQL bootstrap
<!-- stage-runbook: true -->

## Status and boundary

Issue #26 continues from merged #43 (`dc77d9a`), owner-authorized development
bootstrap. Cloud Shell network reachability was reported by the owner; remote
command access was independently verified. Neither proves SQL authorization.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Connectivity | Owner reports TCP 3307 PASS; remote command probe PASS | Test IAM and SQL authority |
| Operator | Existing reviewed Terraform optional operator apply exited 0; IAM login/capability query passed | Execute the reviewed SQL transaction |
| Bootstrap | Hash-verified merged SQL committed, then replayed successfully | Verify grants and ownership |
| Privilege readback | API has no schema creation or owner/operator membership; migrator has explicit owner SET, no ADMIN or inheritance | Execute the dedicated migration artifact in a later controlled stage |

## Current evidence

- PASS: Cloud Shell command access through `gcloud cloud-shell ssh`; existing
  session authorization used internally. No `--authorize-session`, token printing,
  credential-file inspection or credential transfer was used. SDK-created SSH
  keys remain outside Git and were not inspected.
- PASS: proxy v2.26.0 IAM authentication as the separate operator. Readback confirmed
  CREATEROLE, cloud administrator membership, database ownership authority and
  schema creation permission.
- PASS: `db/bootstrap/cloud-sql.sql` SHA256
  `44e7e971be93ba5a17cc9f0660b9797fa806422fc97298f2a617017b8c5b635d`
  verified after copying to Cloud Shell. Initial transaction and replay both
  completed with ON_ERROR_STOP. This proves the managed administrator can execute
  the existing login alterations and role grants; merged SQL is unchanged.
- PASS: SQL privilege metadata confirms API safe attributes, bookkeeping_api
  membership, no owner/identity-operator/cloud-admin membership, no database,
  deployment or public schema CREATE, and connection limit 20. Migrator has safe
  attributes, owner membership with SET true / INHERIT false / ADMIN false,
  database/deployment CREATE, no public CREATE or cloud administrator membership,
  and connection limit 2.
- PARTIAL: these are actual catalog/privilege readbacks through the administrator,
  not separate API/migrator IAM logins or attempted DDL denial under those sessions.
  The `ledger` schema is absent; no product migration or enrollment ran.
- Each proxy stopped through an exit trap. Only reviewed SQL and the official
  proxy binary were transferred/downloaded; operational logs stayed private.
- PASS: post-bootstrap private Terraform drift check exited 0 with the operator
  input preserved. This covers infrastructure, not write-only SQL memberships.
- PASS: `make check test build` with Go 1.26.9, including vet/race tests, migration
  artifact checks and binaries. No product code or merged migration was changed.

## Remaining work

Run the dedicated migrator artifact under its own workload identity, prove
forward migration/replay and API denial with actual runtime sessions, then perform
restore and pool/load acceptance. Do not use the administrator to run migrations
or relax metadata-only application authentication to run it under Cloud Shell's
human identity. Image/job provenance and reviewed deployment gates still apply.

## Operator retention and resume

The successful bootstrap created an ownership dependency: `deployment` is owned
by the operator, which also executed role grants. The operator is intentionally
retained for the next administration stage; cleanup is not claimed. Review schema
ownership and grantor dependencies before revoking its membership or deleting it.
Do not force-drop or ABANDON it. Complete this ownership review before runtime
release; it is part of #26's remaining work.

Preserve the explicit `bootstrap_operator_email` input on subsequent application
applies while this dependency exists. The selected human IAM login can be verified
by names/types through `gcloud sql users list`; never retrieve credentials or state.
Private Terraform directory remains
`/private/tmp/jayburd-iam-bootstrap/.local/iam-bootstrap/application`, using the
existing application GCS backend. Its source matches merged #43. Resume inputs
and quota settings are in [provisioning](cloud-sql-bootstrap.md#operational-resume).

Cloud Shell working directory: `/tmp/jb-bootstrap.lhcrBG`; reviewed SQL and proxy
are there with private logs. Treat the directory as ephemeral. Recopy/reverify
the exact SQL hash if it disappears. The [IAM bootstrap instructions](iam-bootstrap.md)
describe the pinned proxy and private socket boundary. `gcloud cloud-shell ssh`
with `--command` worked; the installed CLI's custom `--ssh-key-file` flag was
incorrectly boolean, so SDK default SSH handling was used without overwrites.

## PR handoff

Keep #26 open until named cloud acceptance passes. Source and implementation
authority remain [G24/G25](../research/gcp-sources.md) and the
[Cloud SQL contract](../contracts/cloud-sql.md).
