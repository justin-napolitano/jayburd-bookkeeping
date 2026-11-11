# Jayburd Bookkeeping

A bookkeeping system for separate LLC and personal books, with a Go
backend, PostgreSQL ledger, Next.js interface, and GCP delivery.

## Current stage

Implemented: private workspace setup and retained-history adoption, transaction
filters/detail, topics/tags and reviewed categorization rules, source-backed
accounting entries, financial reports/CSV and durable saved-work recovery.
Every new application entry starts from retained bank/provider or statement
evidence. Standalone manual posting and unbound reversals are rejected.

The [current capability map](docs/current-capabilities.md) separates working code,
synthetic proof, dated deployment evidence, owner review and future capability.
Latest [synthetic acceptance482](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/482#issuecomment-6805089153)
includes browser/API/restricted PostgreSQL, source-backed restore, 1,000 reviewed
postings and Go 1.26.11 security checks. Hosted provider qualification, complete
statement reconciliation and official-books acceptance remain distinct gates.

## Start here

1. Read [project context](docs/index.md) and [product scope](docs/product.md).
2. Read the [architecture decision](docs/decisions/0001-architecture.md).
3. Follow the [delivery workflow](docs/delivery.md).
4. Review [Infisical setup](docs/infisical.md).
5. Resume from the current GitHub issue and its exact remaining acceptance;
   [historical stages](docs/implementation-history.md) preserve earlier evidence.

For source material and reading priorities, use the
[annotated bibliography](docs/research/bibliography.md).

## Local commands

For an interactive review with fictional generated data, run `make demo` and open
[the synthetic demo](http://localhost:39083). It loads 3,000 source transactions across
three fictional LLCs into
disposable PostgreSQL and serves the real bookkeeping components and Go API.
Stop with Ctrl+C; restart `make demo` to reset. See the
[demo runbook](docs/runbooks/synthetic-demo.md) for prerequisites and data limits.
Loading imports source evidence; review and explicitly post a source item to
update accounting. Retired manual stepping/volume runners are not demo actions.
`make provider-flow-test` separately proves 1,000 source-backed postings and restore
in isolated databases; `make demo-test` verifies the current browser demo.

Requires Go 1.26.11 and Git. These commands need no cloud account or credentials.

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
