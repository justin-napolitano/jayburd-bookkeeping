<!-- stage-runbook: true -->
# Hosted API database permission proof

## Status and boundary

Implements the bounded runner for #130, child of #26. No hosted execution or
migration replay is claimed. The API image contains a separate explicit probe;
the normal API command and health default are unchanged. No source SQL changes,
new grants, public endpoint, business-data reads or credentials are involved.

## Decision and evidence

Follow the existing [Cloud SQL contract](../contracts/cloud-sql.md), particularly
exact SQLSTATE denial and metadata-only authentication. Its linked
[GCP source register](../research/gcp-sources.md) and
[database source register](../research/database-sources.md) remain authoritative:
PostgreSQL transaction rollback provides cleanup; IAM login proves the real
runtime identity. Reusing an operator's SET ROLE would only simulate that identity.
This operational probe is not an ISO financial message; no accounting contract changes.

`databaseprobe.Run` checks exact session/current user and safe role capabilities,
then requires 42501 for DDL, zero-row direct-table reads and forbidden owner/worker
roles. Workspace unknown-principal rejection must be P2101 at repeatable-read.
Each statement has its own transaction, always rolled back, including unexpected
success. Read-only mode is deliberately not substituted: 25006 is not an ACL proof.
The reserved synthetic unknown-principal UUID must not be enrolled in hosted data.

## Operator procedure

After review/merge and a successful Build images run containing this binary,
validate its API build receipt and immutable digest against that exact commit.
Record run, commit and digest in #130 before execution. Do not reuse the pre-probe
API image. The commands below use that reviewed public digest as `probe_image`.

Initialize `infra/gcp/database-probe` with the existing approved GCS backend bucket
and independent prefix `database-probe` (see its README). Using the existing
authorized deployer/operator, apply only this root with `project_id` set to
`jayburd-bookkeeping-dev` and `image` set to that digest. Keep state, plan and
provider output private. No environment or secrets are mounted. This operation
does not require closing #26 and cannot grant itself additional permissions.

Execute once, record the returned execution name as `probe_execution`, and read
only the allowlisted metadata:

```sh
gcloud run jobs execute jayburd-dev-database-probe --project=jayburd-bookkeeping-dev --region=us-central1 --format='value(metadata.name)'
gcloud run jobs executions describe "$probe_execution" --project=jayburd-bookkeeping-dev --region=us-central1 --format='json(metadata.name,spec.taskCount,spec.template.spec.containers[].image,spec.template.spec.containers[].command,spec.template.spec.containers[].args,spec.template.spec.serviceAccountName,spec.template.spec.maxRetries,spec.template.spec.timeoutSeconds,status.conditions[].type,status.conditions[].status,status.succeededCount,status.failedCount,status.cancelledCount,status.runningCount,status.retriedCount,status.completionTime)' > /tmp/jayburd-api-probe-metadata.json
node scripts/database-probe-receipt.mjs "$probe_image" /tmp/jayburd-api-probe-metadata.json /tmp/jayburd-api-probe-receipt.json "$probe_execution"
```

Poll selected metadata until terminal; do not treat acceptance of the execution
request as success. Retain only the validated receipt. On failure stop; do not
inspect raw logs, widen permissions or substitute another identity. No automatic
retry or destructive cleanup is configured. A successful receipt proves the
actual API IAM SQL boundary, not end-user enrollment or restore/load acceptance.

Replay remains a separate reviewed `operation=migrate` release with the same v15
image, followed by `operation=verify`; retain both receipts. Keep #26 open.

## Current evidence

PASS: `TestProbeOutcomes`, `TestConnectionFailure`,
`TestOfflineAndRejectedArguments`, `TestRestrictedMigration` through
`make cloud-sql-test`, and `make check test build release-test`.
All three mocked Terraform tests passed, including `bounded_api_permission_probe`.
`scripts/test-api-container.sh --local-build` passed offline probe/default rejection
and normal API health behavior. This proves the runtime image only; canonical
source-builder verification remains CI evidence, not a local claim.

## Stage ledger

- Issue #130 published before implementation. Added rollback-only probe and mock
  receipt/Terraform tests. Disposable PostgreSQL exact-probe acceptance passed.
- Initial documentation validation identified missing required runbook headings;
  added them before rerunning repository checks.
- Root review added identity-operator and Cloud SQL superuser membership denial.
  Full checks and exact disposable probe passed afterward. Sandbox blocked initial
  localhost/Docker/provider access; approved reruns passed without source changes.

- Integrated main after PRs #134/#135 merged; resolved the Makefile conflict by
  retaining both Sandbox release-policy and database-probe receipt test suites.
  `make check test build release-test` passed again on the merged branch.

## Remaining work

Review and build immutable API image, then run the separate
hosted proof and migration replay/status releases. No hosted PASS is inferred.

## PR handoff

No cloud resources changed in this branch. Hosted acceptance remains outstanding.
Record exact tested outcomes here before handoff; never close #130 based on local tests.
