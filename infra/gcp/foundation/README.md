# Development foundation

## Implemented boundary

Foundation source defines APIs, nine separated identities, three private
versioned state buckets, immutable image storage, independently disabled build and
deployment federation and project-scoped USD 50 budget alerts. The runtime bucket,
deployment grants and identity lookup role are new unapplied #27 additions. No application, database, queue, secret
value or service-account key is created. Development foundation was applied on
2026-10-12; [bootstrap evidence](../../../docs/runbooks/gcp-delivery.md) records scope and limits.

Pinned tools: Terraform 1.15.7 and Google provider 8.6.0; checksums cover Linux AMD64
and macOS ARM64. `make infra-test` copies only allowlisted source/lock/test files to
a temporary directory, clears ambient environment/configuration, disables backend
initialization and uses a mocked provider. Package installation needs registry
access; validation needs no cloud account. Mocks prove configuration, not GCP IAM
acceptance or service availability.

## Owner bootstrap (separate terminal)

Before any apply, owner verifies project ID, project number and linked billing
account, confirms us-central1 and the cost/recovery assumptions in
[foundation operations](../../../docs/gcp-foundation.md), and authenticates outside
agent context. Enabling Service Usage and permission to enable APIs are bootstrap
prerequisites. Owner needs resource-specific administrative permissions plus billing
budget management; do not give these to a runtime or builder account.

1. Create ignored `owner.tfvars` with only project_id, project_number and
   billing_account_id; leave enable_github_build_identity false. Never include values
   for application secrets. Owner reviews Terraform plans privately; agents do not
   retrieve plan/state artifacts.
2. For user ADC, set `USER_PROJECT_OVERRIDE=true` and
   `GOOGLE_BILLING_PROJECT` to the selected project for Terraform commands. The
   budget API requires an explicit quota project; this does not change credentials.
   Run from this directory: `terraform init`, then owner-reviewed
   `terraform plan -var-file=owner.tfvars` and `terraform apply -var-file=owner.tfvars`.
   The first execution deliberately uses local state; keep it private and protected.
3. After successful creation, install the GCS backend using
   `cp backend.tf.example backend.tf`. Run
   `terraform init -migrate-state -backend-config="bucket=OWNER_FOUNDATION_BUCKET" -backend-config="prefix=foundation"`
   with the actual newly created foundation bucket name. Verify migration privately;
   retain a secure recovery copy according to owner retention policy. Never commit
   or send local state/backups to agents. Never use the application bucket here.
4. Future foundation changes remain owner-operated. Preserve existing explicit
   build enablement when updating this project. Application state belongs to the
   owner-operated database root; runtime bucket holds separate runtime/migration
   prefixes. The new source moves deployer object access from application to runtime
   state. Review that grant replacement; do not move existing database state.

## Promotion gate

The provider and pool are disabled by default. Before opting in, #27 must implement
and review `.github/workflows/build-images.yml`: owner-triggered workflow_dispatch,
main only, exact audience, isolated tests/build/scan before authentication, immutable
commit-to-digest receipt, no secret access. Federation independently restricts numeric
repository/owner/actor IDs, main ref and exact workflow identity/event.

The merged build-images workflow is now available; the development provider was
explicitly enabled during owner-authorized bootstrap. Deployment federation remains disabled until the owner reviews and applies
`enable_github_deploy_identity=true`. Read the [rollout contract](../../../docs/contracts/cloud-run-rollout.md)
and release manifest gate before enabling its privileged release identity. Current private main protection returned HTTP 403 and no environments
exist; do not describe manual review conventions as enforced branch protection.
