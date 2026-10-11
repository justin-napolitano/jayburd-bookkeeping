# Development application infrastructure

## Database slice

This root owns the Cloud SQL instance, database, API/migrator IAM logins and their
Cloud SQL connect/login grants. It references foundation account names directly;
no remote state, passwords or secret values are read. No runtime, queue or Job is
deployed yet. See [database contract](../../../docs/contracts/cloud-sql.md).

`make infra-test` validates both roots using mocked providers. Owner provisioning
requires merged foundation, verified project/billing metadata, accepted costs and
explicit apply scope. The role bootstrap and actual managed-service restore remain
unverified in GCP.

## Owner setup (separate terminal)

Before the first apply, install the example as `backend.tf`, then initialize with
`terraform init -backend-config="bucket=OWNER_APPLICATION_STATE_BUCKET" -backend-config="prefix=application"`.
Use the separate application bucket created by foundation, never its foundation
bucket. Owner reviews plans/state privately; no credentials or state are supplied
to agents. Provide only project_id in ignored owner.tfvars. Region is us-central1.

After reviewed apply, owner runs `db/bootstrap/cloud-sql.sql` against bookkeeping
as its administrator with psql variables api_user and migrator_user set to the
created IAM login names. No passwords belong in flags, Git or logs. Run the
reviewed migration image as the migrator identity only after bootstrap; #27 owns
the Job and digest promotion. Runtime readiness rejects unbootstrapped privileges.

Deletion protection is enabled in Terraform and the Cloud SQL settings. Do not
remove protection or automatically destroy for rollback. Use compatible images
and forward database repair; restore drills must use a separately named instance.
