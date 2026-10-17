# Development foundation

## Implemented boundary

Foundation source defines APIs, ten separated identities, five private
versioned state buckets, immutable image storage, independently disabled build and
deployment federation and project-scoped USD 50 budget alerts. The dedicated
workers-state bucket is owner-only; worker-runtime-state is isolated to the new
worker-deployer identity. Worker federation is separately disabled by default in
an unconfigured project and accepts only the exact owner-dispatched deploy-workers
workflow. See the [worker release contract](../../../docs/contracts/worker-release.md)
before owner bootstrap or state migration. Development foundation was initially
applied on 2026-10-12; [bootstrap evidence](../../../docs/runbooks/gcp-delivery.md)
records scope and limits. The worker identity and runtime bucket now exist in
`jayburd-bookkeeping-dev`; the [paused deployment record](../../../docs/runbooks/bank-token-vault.md)
records current control-plane evidence. No application, database, queue, secret
value or service-account key is created by this root.

Identity Toolkit activation now belongs only to the separate
[identity root](../identity/README.md). Its state uses the owner-only foundation
bucket with prefix `identity`; never grant application deployment identities access.

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
4. Future foundation changes remain owner-operated. In the deployed
   `jayburd-bookkeeping-dev` project, pass the checked-in
   `deployed-dev-identities.tfvars` after private `owner.tfvars` on both commands:
   `terraform plan -var-file=owner.tfvars -var-file=deployed-dev-identities.tfvars`
   and `terraform apply -var-file=owner.tfvars -var-file=deployed-dev-identities.tfvars`.
   The overlay contains only the three reviewed `true` identity flags; the private
   file still supplies project and billing inputs. Variable validation rejects a
   plan that would accidentally disable any of the deployed identities even if
   the overlay is omitted. Review and apply the full foundation plan privately
   to reconcile the worker flag after its owner-authorized direct GCP enablement.
   An intentional emergency disablement requires a reviewed change to this
   live-project guard. Application state belongs to the
   owner-operated database root; runtime bucket holds separate runtime/migration
   prefixes. The new source moves deployer object access from application to runtime
   state. Review that grant replacement; do not move existing database state.

## Promotion gate

The provider and pool are disabled by default. Before opting in, #27 must implement
and review `.github/workflows/build-images.yml`: owner-triggered workflow_dispatch,
main only, exact audience, isolated tests/build/scan before authentication, immutable
commit-to-digest receipt, no secret access. Federation independently restricts numeric
repository/owner/actor IDs, main ref and exact workflow identity/event.

The development build, application deployment, and isolated worker deployment
providers are now enabled after owner-authorized releases. The worker workflow
deployed two services while Scheduler remained paused; enabled federation alone
does not approve provider traffic or live bookkeeping. Read the
[rollout contract](../../../docs/contracts/cloud-run-rollout.md) and release
manifest gate before any further promotion. Current private main protection
returned HTTP 403 and no environments exist; do not describe manual review
conventions as enforced branch protection.
