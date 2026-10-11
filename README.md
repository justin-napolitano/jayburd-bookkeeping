# Jayburd Bookkeeping

A planned bookkeeping system for separate LLC and personal books, with a Go
backend, PostgreSQL ledger, Next.js interface, and GCP delivery.

## Current stage

The first synthetic PostgreSQL posting slice is implemented: balanced manual
journals, spot conversion, exact reversals, scoped reads and trial balance, with
atomic audit/receipt/outbox records. Go supplies transaction and HTTP adapters.
There is no public server, production identity verifier, bank connection, frontend,
cloud deployment or approval to use these books as a system of record.

## Start here

1. Read [project context](docs/index.md) and [product scope](docs/product.md).
2. Read the [architecture decision](docs/decisions/0001-architecture.md).
3. Follow the [delivery workflow](docs/delivery.md).
4. Review [Infisical setup](docs/infisical.md).
5. Resume from the [posting proof and remaining gates](docs/runbooks/posting-proof.md).

For source material and reading priorities, use the
[annotated bibliography](docs/research/bibliography.md).

## Local commands

Requires Go 1.26 and Git. These commands need no cloud account or credentials.

```sh
make check
make test
make build
make backlog
make packet ID=JB-001
make issue-drafts
```

Database work is SQL-first, using `golang-migrate` and pgx, without an ORM. Start with the [database contract](docs/contracts/database.md) and
[migration commands](db/migrations/README.md). `make migration-test` separately
requires Docker and tests disposable PostgreSQL migrations. `make ledger-test`
runs the synthetic accounting, permission, concurrency and recovery proof on an
internal Docker network with no published database ports.

The backlog is a one-time issue seed, not a mirror of GitHub status. After
publication, GitHub Issues own execution state and native dependency links.
Markdown in this repository owns design and evidence. No separate vault,
vector database, or autonomous agent scheduler is required.

## Repository map

| Location | Authority |
| --- | --- |
| `docs/` | Product, decisions, contracts, research, and evidence |
| `planning/backlog.json` | Reviewed initial issue proposals and stable IDs |
| `planning/issues.json` | Published issue identities, never execution status |
| `cmd/project/` | Credential-free validation, packets, and issue drafts |
| `internal/ledger/` | Go transaction and HTTP adapters; unit/integration tests |
| `db/migrations/` | Restricted SQL accounting authority and immutable migration history |
| `tests/corpus/` | Original synthetic fixtures and independent expected balances |
| `.github/` | Issue forms, PR template, CI and dependency update configuration |

The future application layout is documented in the architecture decision;
empty application and infrastructure directories are intentionally not seeded.
