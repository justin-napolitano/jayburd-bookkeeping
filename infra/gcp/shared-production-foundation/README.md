# Shared Production foundation

Candidate source for316/337. No cloud apply in default commands. Exact project
jayburd-bookkeeping-dev/9088794002, existing instance jayburd-dev, separate
bookkeeping_prod database. This root contains no SQL instance resource and no
operator elevation. It creates no application service, scheduler or public route.

Validate with `bash scripts/test-infra.sh shared-production-foundation`.
Use an independent private backend; never read/copy Sandbox Terraform state.
Before first apply, import already-created prod API/migrator service accounts,
their SQL IAM users, the existing bookkeeping_prod database and their four
connector/login IAM memberships into the matching resource addresses. Import
does not approve a plan or deployment. Keep state/provider logs private.
Do not destroy/recreate these resources or manage the existing SQL instance from
this root. Retain separate lifecycle ownership of the existing migration job.

The root creates empty secret containers only. Infisical remains the value owner;
no Terraform-managed secret version or credential is permitted. Secret delivery
identity/federation and freshness still require scoped configuration and proof.
Production evidence/registry buckets and bank/link keys are separate from Sandbox.
Build federation defaults disabled. Review enablement and the exact finite build
target before enabling it. Default provisioning cannot publish or deploy images.

Resource privacy and IAM mocks do not establish billing, hosted authority, SQL
grants, secret propagation, restore coverage or approval for personal accounts.
