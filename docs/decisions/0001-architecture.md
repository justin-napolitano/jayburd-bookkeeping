# ADR 0001: transactional Go bookkeeping on GCP

## Decision

Accepted direction, 2026-10-11: Go backend, PostgreSQL authoritative ledger,
Next.js/TypeScript frontend, GCP Cloud Run/Cloud SQL, GitHub Actions and Terraform.
Python is not a runtime requirement. DuckDB is deferred until an analytical
workload justifies it. No full event sourcing or distributed ledger writes.

## Boundaries

Start as a modular application with independently runnable commands. The API,
outbox publisher, bank sync handler and import/export runner reuse the same
accounting rules. Separate executables do not imply separate ledgers or copied
posting logic. Only introduce services for permission, scaling or failure domains.

Proposed future layout (create directories with their first tested behavior):

```text
cmd/api/                  HTTP application composition
cmd/outbox-publisher/     bounded delivery of committed intent
cmd/bank-sync/            private task endpoint
cmd/import-export/        bounded batch job
internal/accounting/      posting and reconciliation rules
internal/ingestion/       provider-independent acquisition operations
internal/platform/        database, identity and transport adapters
db/migrations/            checksummed forward SQL migrations
web/                      Next.js UI
infra/                    Terraform roots with separate state ownership
```

## Consistency

A single database transaction validates authority/open period, posts the full
balanced journal, records audit/idempotency evidence, and writes outbox intent.
No independent debit/credit messages. Closing and posting coordinate locks.
Initially, authoritative reports read posted ledger rows directly.
See [accounting](../contracts/accounting.md) and
[background work](../contracts/background-work.md).

## Tradeoffs

Go favors explicit operations and small executables. PostgreSQL provides durable
transactions but cannot infer correct accounting. Concurrency remains bounded
by database connections, provider limits and per-connection ownership. The
monorepo shares review/build tooling, not an obligation to deploy everything
on every change. Library choices and version pins are a JB-003 decision.
