# Jayburd agent instructions

## Resume

Read `docs/index.md`, the relevant GitHub Issue, its dependencies, linked
decision/contract, and current runbook. Generate `make packet ID=JB-001`
for a seed issue. Recheck GitHub state before starting work; packet seeds are
not live scheduling authority. Claim one bounded issue and use a dedicated
branch/worktree. Never treat a missing or canceled predecessor as completed.

## Authority and scope

GitHub owns task state and discussion. Repository docs own design and evidence.
Generated packets, issue drafts, and dashboards are disposable views.
No application behavior exists at seed stage. Do not equate a passing tooling
build, closed issue, or successful deployment with validated accounting.
No external Brainblast vault is required to build or resume this project.

## Data and secrets

Use synthetic data in tests and CI. Do not read, print, copy, commit, or upload
credential files, private keys, process environments, bank statements, provider
payloads, database dumps, Terraform state/plans, or personal financial records.
Use secret names and redacted outcomes. Owners retrieve/inject real secrets.
Infisical owns application secrets; see `docs/infisical.md`.
Workload identity is the intended GCP authentication mechanism.
No source-provider calls or cloud mutation from default local/CI commands.

## Architecture

Go owns the backend; PostgreSQL owns durable ledger state. Next.js/TypeScript
is the planned frontend. Do not introduce Python, Scala, an event store,
additional databases, or microservices without a reviewed decision.
Posted journals are balanced, atomic, replay-safe and immutable. Corrections
are explicit. Outbox intent commits with its business mutation. Worker retries
never authorize duplicate effects or bypass current entity permissions.
No business rules duplicated across API and jobs. Money never uses float.
Review database contracts before product DDL. Use golang-migrate SQL migrations;
no ORM auto-migration or application-startup migration. Merged SQL is immutable.
Run `make migration-test` for migration/tooling changes; never use a live DSN.

## Changes and evidence

Before material design decisions or implementation, follow
`docs/research/decision-evidence.md`: cite reviewed source sections, distinguish
facts from inference, compare alternatives and name acceptance evidence.
Start database work with `docs/research/database-foundations.md`. Research
recommendations do not themselves approve a schema or accounting policy.

Preserve unrelated changes. Keep one marked stage runbook per delivery branch;
update it with meaningful stages and include it in every implementation commit.
Run `make check`, `make test`, and `make build` before handoff. Additional
integration gates belong to each issue; tooling tests are not product tests.
Before every commit run `bash scripts/check-stage-runbook.sh --staged`.
Add concise purpose/boundary comments for new or changed functions and types.
Acceptance criteria must name tests or explicit reviewed evidence.
Every code-changing PR links an issue, records validation and migration impact,
and identifies remaining risks. Never close canceled work as completed.
Do not deploy, connect live accounts, or publish data without explicit scope.
