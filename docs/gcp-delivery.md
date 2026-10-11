# GCP build and deployment contract

## Status

Cloud delivery remains planned; no project, billing, federation, infrastructure or
deployment workflow exists. CI now checks the Go server, synthetic PostgreSQL
ledger and Next.js application. An API Dockerfile is available for local proof;
it has not been published or deployed. See [application evidence](runbooks/first-app.md).
Delivery work is tracked by [epic #25](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/25).
The owner accepted the synthetic architecture below; project/billing/auth metadata
and apply authorization remain provisioning gates. Do not report placeholder deployment success.

## Accepted synthetic deployment — 2026-10-11

Owner direction: retain transactional Go/PostgreSQL bookkeeping, use managed GCP
services and target **under $50/month** at low development usage. This target is
not a guaranteed cap. `us-central1` is the provisional region, not a deployed choice.

| Component | Selected development default |
| --- | --- |
| API and Next.js frontend | Cloud Run, request billing, min 0/max 2 instances each |
| Ledger | Cloud SQL PostgreSQL 17 Enterprise, `db-g1-small`, single zone, initial 10 GiB SSD |
| Directed background work | Cloud Tasks to one private Go HTTP consumer, OIDC service identity |
| Outbox publisher | Private bounded Cloud Run HTTP handler, Cloud Scheduler every minute |
| Publisher and consumer capacity | Min 0/max 1 instance each; queue limits and DB pools budgeted together |
| Secrets | Infisical scoped sync to Secret Manager; values never enter Terraform/Git |

Use the Cloud SQL Go connector with automatic IAM database authentication and
lazy refresh. Initial connectivity uses encrypted public-IP connector access
without authorized networks, a dedicated VPC connector or NAT; validate project
policy first. Separate IAM login identities from SQL privileges and prove managed
role compatibility. Include overlapping revisions and administrative reserve in
the connection budget. Test IAM login latency/recovery under shared-core CPU pressure
before accepting this tier. Require daily backups, PITR, bounded storage growth,
deletion protection and an isolated restore drill. This tier is not an HA promise.

Posting commits the whole journal and outbox intent atomically. The first consumer
checks a `journal.posted` fact against its immutable journal and records a durable
receipt; it never creates another posting. Define the versioned envelope, directed
command wrapper and ISO applicability before implementing #7. Public APIs remain
compatible. Preserve durable failures independently of queue retention.

No Kubernetes/self-hosted broker or Pub/Sub in this milestone. Reconsider Pub/Sub
when independent subscribers need the same event; reconsider Kubernetes only with
measured sustained usage or shared-capacity economics. Managed queues and request
billing fit intermittent development traffic; this is our inference, not a priced
benchmark. See the reviewed [GCP sources](research/gcp-sources.md).

Before provisioning, estimate all services, including identity, scheduler, storage,
backups, logging, images, network and the temporary restore instance. Configure
alerts/resource bounds and review actual spend after seven days. Owner-supplied
project, billing and auth configuration cannot be inferred from another project.

## Delivery slices and evidence

GitHub owns current status and native dependencies; this table is a navigation map.
The [delivery runbook](runbooks/gcp-delivery.md) records review and release evidence.

| Issue | Owned outcome |
| --- | --- |
| [#5](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/5) | Terraform foundation, IAM, cost and promotion boundary |
| [#26](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26) | Cloud SQL, migrations, pool limits and restore |
| [#27](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/27) | Authenticated application deployment and compatible rollback |
| [#7](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/7) | Outbox lifecycle and directed consumer proof |
| [#11](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/11) | Infisical isolation, rotation and sync recovery |
| [#12](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/12) | 1,000 synthetic postings, delivery faults, restore and seven-day spend |

Infrastructure preparation and local outbox work can proceed independently after
their existing prerequisites are evidenced. Database follows foundation;
application acceptance requires database, identity and secrets evidence. Final
cloud acceptance requires merged PR #23 recovery and PR #24 synthetic capabilities,
without deploying their test identity adapters. Inspect current PR state first.

The early infrastructure proof no longer depends on a full accounting month #10.
Live-pilot #13 still requires #10 and #12. Older open issues are not automatically
completed by this planning change. No bank integration, accountant approval or
live-bookkeeping readiness follows from synthetic cloud success.

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
