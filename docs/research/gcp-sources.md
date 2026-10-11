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
resources. The Cloud SQL pricing page returned an error in this review; no new
verified database price quotation is claimed here. Measure seven-day spend in #12.
