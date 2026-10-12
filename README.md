# Jayburd Bookkeeping

A planned bookkeeping system for separate LLC and personal books, with a Go
backend, PostgreSQL ledger, Next.js interface, and GCP delivery.

## Current stage

The first synthetic PostgreSQL posting slice is implemented: balanced manual
journals, spot conversion, exact reversals, scoped reads and trial balance, with
atomic audit/receipt/outbox records. Go supplies transaction and HTTP adapters.
An identity adapter requires managed TOTP sign-in and checks revocation; controlled
setup creates synthetic entity/book/account bundles atomically. See the
[identity/setup evidence](docs/runbooks/identity-setup.md).
The first application adds a bounded Go server, managed sign-in/TOTP screens,
book setup, split journal review/posting and trial balance. Default server mode
exposes health only; the UI remains unconfigured until the owner supplies public
identity configuration. See [application evidence](docs/runbooks/first-app.md).
Reviewed setup and posting requests now survive a lost response or browser restart:
prepare, execute, recover the saved result, then explicitly acknowledge it.
See [command recovery evidence](docs/runbooks/command-recovery.md).
There is no live identity configuration, bank connection, cloud deployment or
approval to use these books as a system of record.

## Start here

1. Read [project context](docs/index.md) and [product scope](docs/product.md).
2. Read the [architecture decision](docs/decisions/0001-architecture.md).
3. Follow the [delivery workflow](docs/delivery.md).
4. Review [Infisical setup](docs/infisical.md).
5. Resume from the [recovery proof and remaining gates](docs/runbooks/command-recovery.md).

For source material and reading priorities, use the
[annotated bibliography](docs/research/bibliography.md).

## Local commands

Requires Go 1.26.9 and Git. These commands need no cloud account or credentials.

```sh
make check
make test
make build
make backlog
make packet ID=JB-001
make issue-drafts
```

For the browser package, use Node 24.18.0 and `npm --prefix web ci`, then
`make web-check web-test web-build`. Install the pinned test browser with
`cd web && npx --no-install playwright install chromium`, then run
`make web-browser` from the repository root. Browser tests use synthetic adapters;
they never connect to a managed identity project.

`bin/api` starts loopback health-only mode. See the [server contract](docs/contracts/server.md)
for explicit managed composition and the [browser contract](docs/contracts/browser-auth.md)
for the separate provider configuration and recovery gates.

Database work is SQL-first, using `golang-migrate` and pgx, without an ORM. Start with the [database contract](docs/contracts/database.md) and
[migration commands](db/migrations/README.md). `make migration-test` separately
requires Docker and tests disposable PostgreSQL migrations. `make ledger-test`
runs the synthetic accounting, permission, concurrency and recovery proof on an
internal Docker network with no published database ports.
`make app-integration` requires those tools plus the installed browser and tests
the real Next rewrite, Go HTTP service and PostgreSQL together. It drops a response
after commit and proves recovery without a duplicate journal. Identity is synthetic;
this does not test the managed sign-in provider. Ports 39081–39083 must be free.

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
| `cmd/api/`, `internal/server/` | Bounded HTTP lifecycle and explicit managed composition |
| `web/` | Next.js application, memory-only sign-in and synthetic browser proof |
| `internal/identity/` | Managed authentication adapter; offline boundary tests |
| `internal/ledger/` | Go transaction and HTTP adapters; unit/integration tests |
| `db/migrations/` | Restricted SQL accounting authority and immutable migration history |
| `tests/corpus/` | Original synthetic fixtures and independent expected balances |
| `.github/` | Issue forms, PR template, CI and dependency update configuration |

The future application layout is documented in the architecture decision;
empty application and infrastructure directories are intentionally not seeded.
