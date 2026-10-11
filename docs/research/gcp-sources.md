# GCP delivery source register

## Decision and review boundary

Owner accepted managed Cloud Run, Cloud SQL and Cloud Tasks for the synthetic
milestone. Google Cloud publishes all sources below; accessed 2026-10-11.
These are mutable provider manuals/pricing pages, not accounting standards.
Selected sections were reviewed, not entire documentation sets. Recheck at
implementation; acceptance belongs to the linked issues in [GCP delivery](../gcp-delivery.md).

| ID / source | Reviewed sections and established facts | Application and limits |
| --- | --- | --- |
| G01 — [Cloud Run pricing](https://cloud.google.com/run/pricing) | Request-based pricing and billable instance time; request work/startup/shutdown are billed, configured minimum instances incur idle charges; Jobs have a one-minute minimum per instance. | Choose zero minimums and bounded HTTP publisher calls. Actual cold-start/scheduler/load costs require measurement. |
| G02 — [Cloud Tasks pricing](https://cloud.google.com/tasks/pricing) | Billable operations and billing notes: first million operations/month/account free, then $0.40/million at the displayed tier; API calls and delivery attempts count in 32 KB chunks. | Transaction count is not operation count. Retries, payload size, shared allowance and network costs affect spend. |
| G03 — [Cloud Tasks issues and limitations](https://docs.cloud.google.com/tasks/docs/common-pitfalls) | Execution order, duplicate execution, backoff and target resource limits. Ordering is not guaranteed; duplicates occur; overloaded targets cause backlog and throttling. | Require durable deduplication, fault tests and bounded queue/pool concurrency in #7. Provider delivery does not prove business completion. |
| G04 — [Connect from Cloud Run](https://docs.cloud.google.com/sql/docs/postgres/connect-run) | Go connector example and connection-pooling guidance. Connector integrates with pgx; public/private connectivity and pool management are explicit choices. | #26 validates encrypted connector access and connection budget. Examples using passwords do not authorize secret retrieval or supersede IAM direction. |
| G05 — [Cloud SQL IAM authentication](https://docs.cloud.google.com/sql/docs/postgres/iam-authentication) | IAM authentication concepts, automatic authentication and performance on shared-core instances. Connector-managed authentication handles short-lived access tokens; database permissions remain separately managed. | Choose automatic IAM authentication; test SQL role grants and managed-service compatibility. Shared-core IAM login can slow or time out under CPU pressure; test bounded connection recovery. No production access has been exercised. |

## Alternatives and uncertainty

Self-hosted Kubernetes/broker adds operational responsibilities; managed request
billing is our preferred fit for intermittent development, not a demonstrated
universal cost win. Pub/Sub fan-out is deferred until independently owned consumers
exist. Neither choice replaces the atomic PostgreSQL ledger.

The under-$50 target excludes bank-provider subscriptions and is not a guaranteed
ceiling. A complete regional estimate remains required by #5, including Cloud SQL
compute/storage/backups, identity, network, scheduler, logs, artifacts and restore
resources. The initial browser extraction of SQL pricing failed; the foundation follow-up
review below retrieved official HTML and records the bounded rate check. Measure
seven-day spend in #12.

## Foundation implementation review — 2026-10-11

- **G06 — HashiCorp, [Terraform provider mocking](https://developer.hashicorp.com/terraform/language/tests/mocking)**: reviewed Mock Providers and Generated Data. Mocks use provider schemas without resource API calls; passing mocks do not prove real IAM or provider behavior. Implemented six named foundation tests with Terraform 1.15.7/provider 8.6.0. Selected sections only.
- **G07 — Google, [WIF for deployment pipelines](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines)**: reviewed GitHub attribute mapping/conditions and default audience. Numeric repository/owner IDs avoid name-reuse trust; mapped claims constrain federation. Our additional owner actor/event/workflow restrictions are a project decision. Actual token denial tests remain cloud evidence.
- **G08 — HashiCorp, [GCS backend](https://developer.hashicorp.com/terraform/language/backend/gcs)**: reviewed bucket prerequisite, locking, versioning and authentication guidance. Bootstrap must create its bucket first; owner then migrates local state privately. Separate application bucket narrows deployer access. No remote state was read or initialized.
- **G09 — Google, [Cloud Scheduler pricing](https://cloud.google.com/scheduler/pricing)**: reviewed Pricing overview/Free usage. Billing is per configured job, not per execution; first three jobs/account are free, then $0.10/job/31 days. Invocation compute is separate; account-wide usage remains unknown.

The checked-in lock file records signed provider package hashes for macOS ARM64
and Linux AMD64. Release identity was checked against the upstream Google provider
v8.6.0 release. Private branch-protection API still returned HTTP 403; the repository
environments API returned an empty list. The [foundation contract](../gcp-foundation.md)
therefore does not grant GitHub apply access or assume an enforced reviewer gate.

- **G10 — Google, [Cloud SQL pricing](https://cloud.google.com/sql/pricing?hl=en)**: reviewed Shared-core CPU/RAM and Iowa storage/backup tables in official HTML after the browser tool rejected its size. db-g1-small $0.035/hour; SSD $0.000232877/GiB-hour; used backups $0.000109589/GiB-hour. Shared-core types are excluded from the Cloud SQL SLA. The 730-hour example in the foundation contract is our calculation, not a full-stack quote or a billing-account-specific offer. No authenticated billing data was retrieved.
