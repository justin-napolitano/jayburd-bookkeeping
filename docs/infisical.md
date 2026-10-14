# Infisical setup and delivery

## Decision

Infisical owns application secrets. Target delivery is a scoped, one-way sync to
GCP Secret Manager; Cloud Run reads selected versions through its service account.
GitHub deploys through GCP workload identity and need not read application secrets.
This keeps Infisical availability out of each application startup. Sync freshness,
rotation and rollback must be proved before live use (JB-013).

## Seed now

Create an empty Secret Management project named `jayburd-bookkeeping` in the
owner's Infisical organization. Record its actual ID, generated slug and cloud
region after creation. GitHub-linked login authenticates the human, not workflows.

Create environments `dev`, `staging`, `prod`, with component folders `/api`,
`/bank-sync`, `/import-export`. Keep root empty; create no dummy credentials.
Use owner-only access initially. Runtime access is read-only per component and
environment. If the account cannot enforce this scope, split projects; do not
compensate with broad access. Confirm plan capabilities before enabling identities.

Current status: private GitHub repository exists; Infisical account was reported
created by the owner. No project, identity, folder, sync or secret has been verified
or created by this seed. The session browser is unavailable. Local CLI is installed;
its authentication store has not been accessed.

## Cloud integration: JB-005 and JB-013

Use Infisical Cloud's GCP service-account impersonation connection, with a dedicated
sync service account per environment. Resolve the region-specific Infisical
principal from official setup instructions; grant impersonation only on that
account. Respect the documented organization-ID suffix for its name. Review exact
sync permissions against pre-created destination secrets; never grant project
Owner or distribute service-account keys. Terraform owns containers/IAM, while
Infisical owns values and versions. No secret values in Terraform inputs/state.

Scope each sync to one component folder and environment. Use a unique destination
prefix such as `jayburd-dev-api-`; configure manual initial sync and disable
destination deletion. Review the initial-sync behavior before enabling it: the
overwrite option can delete unrelated destination secrets. Never import existing
unrelated GCP secrets. After a synthetic canary proof, define freshness alerts and
controlled rotation, version selection, revocation and recovery procedures.

Cloud Run gets access only to its required destination secrets. A compromised
bank-sync worker must not read another component or production environment.
Direct Infisical GCP ID Token Auth is an alternative only through a reviewed
decision; do not implement both delivery paths by default.

## Optional future GitHub access

Seed CI has no Infisical access or OIDC permission. If a later job truly needs
Infisical, use a separate read-only machine identity and OIDC, remove default
Universal Auth, set TTL/max TTL to 900 seconds and grant only that job access.
Verified repository trust metadata (2026-10-11):

| Field | Value |
| --- | --- |
| Issuer/discovery | `https://token.actions.githubusercontent.com` |
| Repository ID | `1414515318` |
| Owner ID | `55423093` |
| Subject prefix | `repo:justin-napolitano@55423093/jayburd-bookkeeping@1414515318` |
| Main-only subject suffix | `:ref:refs/heads/main` |
| Environment subject suffix | `:environment:dev` (only for a job using that environment) |
| Audience | `https://github.com/justin-napolitano` (request and validate exactly) |

Require exact repository/owner IDs and `ref=refs/heads/main` claims alongside the
selected subject. No wildcard trust or fork/PR access. Environment protection
must actually be available before relying on it. Never print JWTs to debug claims.

## Verification and sources

JB-013 requires synthetic allow/deny, cross-environment denial, rotation/revocation,
sync-failure recovery and redacted receipts. Owner enters values separately; agents
receive names and PASS/FAIL evidence only. Default local and PR checks stay offline.

- [GCP impersonation connection](https://infisical.com/docs/integrations/app-connections/gcp)
- [GCP sync options](https://infisical.com/docs/integrations/secret-syncs/gcp-secret-manager)
- [GitHub OIDC and immutable subjects](https://infisical.com/docs/documentation/platform/identities/oidc-auth/github)

## Dynamic provider credentials

The [provider vault contract](contracts/provider-vault.md) now specifies and
implements a separately scoped API writer through Infisical GCP ID Token Auth.
This is the reviewed exception for connection creation/recovery, not an acquisition
reader fallback. Acquisition still reads only exact Google Secret Manager versions.
Eight precreated, never-reused development slots and immutable private GCS mappings
bind logical credential identity to independently verified destination versions.

The [hosted API composition](contracts/hosted-connections.md) remains explicitly
opt-in and Sandbox-only. Synthetic tests prove one-shot writes, uncertain-response
recovery and mismatch denial; no Infisical project, IAM, sync or hosted secret
acceptance is claimed. See the [delivery runbook](runbooks/hosted-sandbox.md) for
current evidence and owner configuration prerequisites.
