# Jayburd Bookkeeping

A planned bookkeeping system for separate LLC and personal books, with a Go
backend, PostgreSQL ledger, Next.js interface, and GCP delivery.

## Current stage

This is a planning and delivery seed. The only executable is project tooling.
There is no accounting implementation, bank connection, frontend, deployed
infrastructure, or approval to use these books as a system of record.

## Start here

1. Read [project context](docs/index.md) and [product scope](docs/product.md).
2. Read the [architecture decision](docs/decisions/0001-architecture.md).
3. Follow the [delivery workflow](docs/delivery.md).
4. Review [Infisical setup](docs/infisical.md).
5. Resume from the [seed runbook](docs/runbooks/seed.md).

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
| `.github/` | Issue forms, PR template, CI and dependency update configuration |

The future application layout is documented in the architecture decision;
empty application and infrastructure directories are intentionally not seeded.
