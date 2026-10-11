# GCP build and deployment contract

## Status

Planned; no project, billing, federation, infrastructure, application image or
deployment workflow exists. The seed CI validates project tooling and docs only.
Implement JB-003/JB-005/JB-011 before any cloud proof. Region and budget need owner
selection. Do not add a workflow that reports a placeholder deployment as success.

## Build once, promote by digest

PR: credential-free tests, real disposable PostgreSQL integration tests, UI
checks, migration tests and Terraform validation as components become available.
Main: build tested component images once, scan, publish to Artifact Registry,
record commit and image digests, provenance and dependency inventory.
Deployment: promote exact digests, migrate through a dedicated identity, verify
health/authorization, then record the revision and reviewed evidence.

## Infrastructure boundary

Terraform owns infrastructure and configured service/job revisions; workflows
invoke it rather than independently changing the same fields with gcloud.
Separate bootstrap/foundation and application state. Bootstrap must establish
the restricted remote state bucket and GitHub federation before workflows can
use them: there is an explicit owner-operated first step, not circular automation.
Infrastructure plans/state may contain sensitive data; never publish them in
Issues or agent context. Owner review and approved apply happen in the deployment
boundary. Use minimal metadata receipts for project context.

## Application secrets

Follow [Infisical delivery](infisical.md): scoped sync to GCP Secret Manager,
runtime service-account access, no values in Terraform or build workflows.
JB-013 must pass before the synthetic deployment milestone.

## Access and operations

- Restrict OIDC trust to repository identity and approved branch/environment;
  fork PRs receive no cloud credentials. No `pull_request_target` code execution.
- Separate deployer, migrator, API, publisher and worker identities. Runtime
  database roles cannot perform migrations or bypass entity authorization.
- Cloud Run API/workers remain private until an explicit user-authenticated
  ingress contract exists. No public demo shares real ledger data or credentials.
- Cloud SQL backups/PITR, restore drills, deletion protection and bounded
  connection pools; workload max instances respect the total connection budget.
- Redacted structured logs, task/operation IDs, freshness/backlog alerts, budget
  alerts, failure quarantine and separate development/production targets.
- Serialize deployments per environment. Use expand/contract migrations;
  application rollback does not mean automatically reversing database changes.

## GitHub protection caveat

Private branch protection returned HTTP 403 on 2026-10-11 (Pro upgrade required).
Verify environment/reviewer capabilities separately before cloud releases.
Do not assume `environment: production` creates an approval gate. If unavailable,
choose a documented owner-controlled promotion boundary before live releases.

## Sources checked 2026-10-11

- [Immutable action pins](https://docs.github.com/en/actions/reference/security/secure-use)
- [GCP OIDC](https://docs.github.com/en/actions/how-tos/secure-your-work/security-harden-deployments/oidc-in-google-cloud-platform)
- [Private environment restrictions](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/review-deployments)
