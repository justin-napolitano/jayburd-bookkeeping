# Project context

## Decision

Build actual bookkeeping software, learning through small, testable slices.
The owner selected Go, PostgreSQL, Next.js, GCP, transactions and outbox-driven
background work. The two predecessor repos are reference material only.

## Read by task

| Question | Authoritative document |
| --- | --- |
| What are we building? | [Product scope](product.md) |
| Why this stack and boundaries? | [Architecture](decisions/0001-architecture.md) |
| What research supports the database direction? | [Research review](research/database-foundations.md) |
| Which source sections were reviewed? | [Source register](research/database-sources.md) |
| What should owners and agents read first? | [Annotated bibliography and reading guide](research/bibliography.md) |
| What evidence must future decisions include? | [Decision evidence standard](research/decision-evidence.md) |
| How do we manage work and context? | [Knowledge and issue ownership](decisions/0002-project-memory.md) |
| What must the ledger guarantee? | [Accounting contract](contracts/accounting.md) |
| Which ISO standards must messages and practices follow? | [Standards applicability contract](contracts/standards.md) |
| How do we design the database? | [Database contract](contracts/database.md) |
| Which schema owns each responsibility? | [Schema ownership ADR](decisions/0004-schema-ownership.md) |
| What examples and policy decisions drive posting? | [Accounting scenarios](contracts/accounting-scenarios.md) |
| What policy drives the first development book? | [Development policy ADR](decisions/0005-development-accounting-policy.md) |
| What exactly must the posting implementation do? | [Posting boundary and acceptance matrix](contracts/posting-boundary.md) |
| How are money and FX represented? | [Multicurrency contract](contracts/multicurrency.md) |
| What are the proposed columns, roles and transactions? | [First ledger SQL contract](contracts/ledger-sql.md) |
| What does the proposed HTTP interface accept? | [API contract](contracts/api.md) |
| Do we need an ORM? | [SQL-first decision](decisions/0003-database-first.md) |
| How do migrations run? | [Migration workflow](../db/migrations/README.md) |
| What must retries and workers guarantee? | [Background work contract](contracts/background-work.md) |
| How are secrets managed? | [Infisical setup](infisical.md) |
| How will builds reach GCP? | [GCP delivery contract](gcp-delivery.md) |
| How are application images packaged and published? | [Image delivery contract](contracts/image-delivery.md) |
| How does Cloud SQL connect and migrate safely? | [Cloud SQL contract](contracts/cloud-sql.md) |
| How is the GCP foundation validated and bootstrapped? | [Foundation operations](gcp-foundation.md) |
| How do we start, review and finish tasks? | [Delivery workflow](delivery.md) |
| What worked in Brainblast? | [Tooling assessment](research/brainblast.md) |
| How are identities enrolled and books created? | [Identity/setup contract](contracts/identity-setup.md) and [delivery evidence](runbooks/identity-setup.md) |
| How does the first application run? | [Server contract](contracts/server.md), [browser sign-in](contracts/browser-auth.md), [first UI](contracts/first-ui.md) and [application evidence](runbooks/first-app.md) |
| What has actually passed? | [PostgreSQL posting proof](runbooks/posting-proof.md) |
| What was checked for the contract package? | [Ledger contract runbook](runbooks/ledger-contracts.md) |

## Synthetic test data

Use the [corpus and execution guide](../tests/corpus/README.md),
[annotated test sources](research/testing-sources.md), and
[corpus evidence runbook](runbooks/synthetic-corpus.md).

## Next action

Review the [first application evidence](runbooks/first-app.md), then complete the
managed identity and deployment gates before connecting live accounts. JB-001 remains open for a real
LLC policy, opening-balance source and accountant cutover review. Development
accrual/USD defaults do not resolve those live-use decisions.
Entity/account counts, providers and GCP project/billing/auth configuration remain
unknown. The owner accepted an under-$50/month synthetic development target;
`us-central1` remains provisional. See [GCP delivery](gcp-delivery.md),
[epic #25](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/25) and
the [delivery evidence ledger](runbooks/gcp-delivery.md).
