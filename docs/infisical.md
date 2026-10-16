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
`/bank-sync`, `/import-export`. Owner override (2026-10-14): existing Plaid keys remain at root.
The [allowlisted delivery contract](contracts/scoped-delivery.md) maps only the
two selected root keys; do not enable a root-wide native sync.
Use owner-only access initially. Runtime access is read-only per component and
environment. If the account cannot enforce this scope, split projects; do not
compensate with broad access. Confirm plan capabilities before enabling identities.

Current setup and resource-adoption evidence is in the
[delivery runbook](runbooks/scoped-delivery.md). The GCP connection worked per owner
report; secret propagation is not yet qualified. Repository is now public.
Native sync is blocked by its upstream destination-read requirements.

## Cloud integration: JB-005 and JB-013

Historical native-sync candidate: Infisical Cloud's GCP service-account impersonation connection, with a dedicated
sync service account per environment. Resolve the region-specific Infisical
principal from official setup instructions; grant impersonation only on that
account. Respect the documented organization-ID suffix for its name. Review exact
sync permissions against pre-created destination secrets; never grant project
Owner or distribute service-account keys. Terraform owns containers/IAM, while
Infisical owns values and versions. No secret values in Terraform inputs/state.

Native sync remains disabled. The following are historical qualification requirements,
not instructions to enable the incompatible native adapter. Scope each sync to one
component folder and environment. Use a unique destination
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

## Per-account bank tokens

Owner selected [encrypted PostgreSQL persistence](contracts/bank-token-vault.md)
under #162–#164. New API/worker binaries use Cloud KMS envelopes in the database,
not dynamic Infisical keys or delivery slots. Infisical still owns static Plaid
client credentials; their existing controlled delivery remains separate.

The [legacy provider vault](contracts/provider-vault.md) and its eight-slot pool
are retained only for history and explicit legacy recovery, with no automatic
fallback, import or resource deletion. Existing legacy release identifier fields
remain validated for schemaVersion2 compatibility but do not select a dynamic
writer in the new binaries. Retiring these fields requires a versioned release
contract change. Do not paste tokens into SQL or create slot placeholders.
