# Background work source review

## Scope

Selected public sections reviewed 2026-10-13 for #63, starting with bibliography
S13–S15 and G03. Supports the [proposed worker contract](../contracts/background-work.md).
No live provider calls, deployment verification or normative certification.

## BW01 — Transactional outbox

Chris Richardson, [Transactional outbox pattern](https://microservices.io/patterns/data/transactional-outbox.html),
Problem/Solution/Result context reviewed. Durable business changes and message
intent share a transaction; a relay can send twice after a crash, so consumers
need idempotency. This is engineering pattern guidance, not accounting authority.
Our separate sidecars are an inference from this repository's immutable outbox;
the pattern does not approve a particular DDL, fence or retry budget. Alternative:
database-only polling; revisit when measured transport overhead exceeds benefit.

## BW02 — Actual Cloud Tasks behavior

Google Cloud, [Issues and limitations](https://docs.cloud.google.com/tasks/docs/common-pitfalls),
Execution order, Duplicate execution, Backoff errors, Resource limitations and
long-running task ramp-up reviewed (page updated 2026-10-09). Delivery order and
absence of duplicate execution cannot be relied upon; overload changes throughput.
**Inference:** operation identity lives in SQL and completion is not queue depth.
No provider statistical percentage establishes our correctness or latency SLO.

Google Cloud REST **v2**, [RetryConfig](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/RetryConfig),
`maxAttempts`, `maxRetryDuration`, backoff fields reviewed. Attempt/duration retry
stopping conditions are documented jointly; queue configuration is not our OR
business limit. **Inference:** enforce attempt/age cutoff durably in the database.
Defaults and mutable service docs must be rechecked when Terraform is implemented.

Google Cloud REST **v2**, [Task resource](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/projects.locations.queues.tasks),
`dispatchDeadline`, schedule/attempt fields reviewed. HTTP deadlines have bounded
supported values; request timeout does not guarantee worker processing stops.
**Inference:** resource fence and commit predicate must survive duplicate handlers;
our60/90-second budgets are project proposals, not recommended universal settings.

## BW03 — Enqueue identity and authenticated delivery

Google Cloud REST **v2**, [tasks.create](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/projects.locations.queues.tasks/create),
Request body/task deduplication and response view reviewed. Explicit names can
reject duplicates including recently deleted/executed tasks; hashed IDs are advised.
**Inference:** `ALREADY_EXISTS` is not durable work completion or proof of a live
pending task. Persist generation and recover independently of name retention.
No fixed deduplication window is used as a correctness premise.

Google Cloud, [Create HTTP target tasks](https://docs.cloud.google.com/tasks/docs/creating-http-target-tasks),
Set up service account and Use HTTP target tasks with authentication tokens reviewed.
OIDC service identity supports authenticated Cloud Run invocation; queue overrides
may affect task authentication configuration. **Inference:** pin handler audience
and allowed invocation identity, then separately check current business authority.
No IAM resources or credentials created/read; exact Terraform grants remain a
reviewed deployment deliverable, not a copied sample's assumed suitability.

## BW04 — Database claim semantics

PostgreSQL Global Development Group, PostgreSQL **17**,
[SELECT, Locking Clause](https://www.postgresql.org/docs/17/sql-select.html#SQL-FOR-UPDATE-SHARE),
reviewed `FOR UPDATE`, `SKIP LOCKED`, queue-like use and inconsistent-view warning.
Short queue claims can skip contended rows; this does not provide fairness,
business ordering or a fence across external calls. **Inference:** deterministic
scan key, durable deadlines, compare-and-set ownership and recovery scans are
separate application requirements. Existing bookkeeping transaction isolation and
authorization locks remain authority for financial operations. Expiring a lease
does not terminate the holder process; verify stale commits with actual SQL tests.

## BW05 — Financial message applicability

ISO 20022 Registration Authority, [FAQ](https://www.iso20022.org/faq), About and
Compliance sections, reviewed 2026-10-13 in the adjacent reporting research.
The methodology/registered messages and community usage requirements must not be
confused with arbitrary JSON field naming. **Inference:** the internal
journal-fact/wakeup pair has no selected external financial-message profile;
owner reviews this bounded applicability assessment before implementation.
Individual ISO message definitions were not exhaustively reviewed. Bibliography
R03–R05 remain currency/audit research authority; no normative audit fields or
new currency recognition policy derived here. Financial provider adapters must
resolve their own required profile/mapping gates under the standards contract.
