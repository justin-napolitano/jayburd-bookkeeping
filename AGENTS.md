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
The synthetic posting proof is implemented; see `docs/runbooks/posting-proof.md`
for exact evidence and limits. Identity and controlled setup follow
`docs/contracts/identity-setup.md` and `docs/runbooks/identity-setup.md`; never
substitute an auth emulator, email or caller actor ID for verified identity. Do not equate a passing build, closed issue or
successful deployment with live bookkeeping approval.
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
owns the frontend. Follow `docs/runbooks/first-app.md` for its evidence and limits.
Interactive setup/post recovery follows `docs/contracts/command-recovery.md`:
persist reviewed work before execution and require explicit terminal acknowledgement.
Do not store financial requests or tokens in browser persistence. Run
`make app-integration` for changes crossing this browser/API/SQL boundary;
`docs/runbooks/command-recovery.md` records its synthetic identity limitation.
Do not introduce Python, Scala, an event store,
additional databases, or microservices without a reviewed decision.
Posted journals are balanced, atomic, replay-safe and immutable. Corrections
are explicit. Outbox intent commits with its business mutation. Worker retries
never authorize duplicate effects or bypass current entity permissions.
No business rules duplicated across API and jobs. Money never uses float.
Use the domain schemas and lowercase singular naming in
`docs/decisions/0004-schema-ownership.md`; no application objects in `public`
or a schema per LLC by default. Follow the scoped SQL/API contracts linked from
`docs/contracts/database.md`; draft policy and test names are not implemented guarantees.
For the first synthetic posting slice, follow ADR 0005 and
`docs/contracts/posting-boundary.md`: SQL owns exact valuation and normalized
receipt equality; Go owns transport, trusted identity and transaction retries.
Development policy is not live-LLC approval. Do not mount the deferred close route.
Review database contracts before product DDL. Use golang-migrate SQL migrations;
no ORM auto-migration or application-startup migration. Merged SQL is immutable.
Run `make migration-test` for migration/tooling changes and `make ledger-test`
for accounting/transaction changes; both use disposable synthetic databases,
never a live DSN.

## Standards requirement

Owner direction: use applicable ISO standards for every financial message,
data contract and practice. Before designing or changing one, follow
`docs/contracts/standards.md`: record applicability, authoritative source,
edition/message version, usage profile, mapping and named validation evidence.
Use ISO 20022 definitions for covered financial exchanges and ISO 4217 currency
identifiers. Review applicable audit-data standards for exports and semantics.
Internal APIs/events also require an applicability assessment; do not invent
an ISO designation or force an unrelated message onto a business operation.
Document gaps and deviations with rationale and a reviewed decision before
implementation. ISO does not replace the selected accounting framework or
jurisdictional requirements. Never claim conformance from naming or XSD alone.
Multicurrency is required scope; do not silently reduce the ledger to one currency.

## Changes and evidence

Start source review with `docs/research/bibliography.md`. Keep its annotated
resource index and the linked canonical source records current when adding or
using references; record review depth and access limits, not just URLs.

For synthetic accounting checks, follow `tests/corpus/README.md` and its source
register. Keep fixed expected results independent of generators and posting code.
External engines are optional test tools; default CI must not download or invoke
them. Never treat generator self-validation or XSD acceptance as ledger or ISO
conformance. Preserve original units alongside functional-currency assertions.

Before material design decisions or implementation, follow
`docs/research/decision-evidence.md`: cite reviewed source sections, distinguish
facts from inference, compare alternatives and name acceptance evidence.
Start database work with `docs/research/database-foundations.md`. Research
recommendations do not themselves approve a schema or accounting policy.

Preserve unrelated changes. Keep one marked stage runbook per delivery branch;
update it with meaningful stages and include it in every implementation commit.
Run `make check`, `make test`, and `make build` before handoff. Additional
integration gates belong to each issue; tooling tests are not product tests.
For browser or API-contract changes run `make web-check web-test web-build
web-browser` after installing the locked dependencies and test browser. Generated
OpenAPI types must match the reviewed contract; test-only auth adapters never
belong in application routes. Money remains decimal strings through the UI.
Before every commit run `bash scripts/check-stage-runbook.sh --staged`.
Add concise purpose/boundary comments for new or changed functions and types.
Acceptance criteria must name tests or explicit reviewed evidence.
Every code-changing PR links an issue, records validation and migration impact,
and identifies remaining risks. Never close canceled work as completed.
Do not deploy, connect live accounts, or publish data without explicit scope.
