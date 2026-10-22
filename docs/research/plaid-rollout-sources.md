# Plaid Sandbox rollout source review — #258

Reviewed 2026-10-22; public selected sections only. No provider payload,
credential, state or plan inspected.

## RQ01 — Cloud Scheduler v1 Job/RetryConfig/State

Google, [REST Job](https://docs.cloud.google.com/scheduler/docs/reference/rest/v1/projects.locations.jobs),
`schedule`, `attemptDeadline`, State and RetryConfig fields. PAUSED prevents new
scheduled execution. A deadline cancels an overdue request; this does not prove
the application stopped all work. Default retryCount is zero; zero count with
zero maxRetryDuration does not retry a failed attempt. Existing readback has a
90-second deadline, omitted/default-zero retryCount and maxRetryDuration `0s`;
source Terraform explicitly sets retry_count=0. This is a bounded metadata
interpretation, not inspection of every historic execution.

## Local application boundary and interpretation

Cloud Tasks [v2 tasks.list](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/projects.locations.queues.tasks/list),
responseView and pagination sections, reviewed on the same date: BASIC is the
default view and excludes sensitive payload details; FULL is separate authority.
List order is unspecified and fewer results may arrive than pageSize. The CLI
listing used here follows its normal BASIC mode, paginates and projects only
names into a count with an upper bound. Any nonzero count blocks the empty-queue
observation; no FULL view, task body or arbitrary request header is requested.

`internal/observer/http.go` bounds acquisition publisher/consumer handler context
at 80 seconds after separately bounded five-second caller verification;
`internal/acquisition/runner.go` bounds acquisition at 60 seconds. Context expiry
requires cooperative completion and is not a distributed-drain guarantee.

Rollout procedure: pause the existing Scheduler, verify PAUSED, wait beyond its
90-second request deadline with two observations separated by useful rollout
work, inspect only queue-name counts and fixed known-account lease/status fields,
and keep scheduling paused through worker readback. No task payload is retrieved.
An empty list and inactive known leases are point-in-time evidence, not an
universal absence-of-work proof. If work remains, wait/reinspect rather than purge
tasks or force SQL state. Migration28 also retains the old generic diagnostic,
exact amount rules and existing binding semantics, so residual authorized old
work remains compatible; new loan binding/checking retry wait until matching
services are ready. Restore only the previously enabled schedule afterward.

Alternative: replace workers while normal publication continues. Rejected for
this bounded qualification because it obscures version/diagnostic attribution.
Acceptance: retained migration/status/runtime receipts, PAUSED readbacks, selected
queue/status observations, matching immutable revisions, then actual per-account
completion. No live banking, automatic journals or ISO exchange is introduced.

## CI availability check — 2026-10-22

[npm's public status page](https://status.npmjs.org/), selected current component
and incident sections, reported Security Audit and package installation
operational with no incident for October 22. Public status text only was read;
this does not prove availability from a particular runner. PR #263's actual
audit request returned HTTP 503 from the bulk-advisory endpoint after five
minutes. Preserve that failed-gate evidence and rerun without dropping or
weakening the audit. No dependency or security-policy change follows from a
status page. Current source/SQL tests and actual hosted qualification remain
separate acceptance evidence.
