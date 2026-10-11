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
| What evidence must future decisions include? | [Decision evidence standard](research/decision-evidence.md) |
| How do we manage work and context? | [Knowledge and issue ownership](decisions/0002-project-memory.md) |
| What must the ledger guarantee? | [Accounting contract](contracts/accounting.md) |
| Which ISO standards must messages and practices follow? | [Standards applicability contract](contracts/standards.md) |
| How do we design the database? | [Database contract](contracts/database.md) |
| Do we need an ORM? | [SQL-first decision](decisions/0003-database-first.md) |
| How do migrations run? | [Migration workflow](../db/migrations/README.md) |
| What must retries and workers guarantee? | [Background work contract](contracts/background-work.md) |
| How are secrets managed? | [Infisical setup](infisical.md) |
| How will builds reach GCP? | [GCP delivery contract](gcp-delivery.md) |
| How do we start, review and finish tasks? | [Delivery workflow](delivery.md) |
| What worked in Brainblast? | [Tooling assessment](research/brainblast.md) |
| What has actually passed? | [Seed runbook](runbooks/seed.md) |

## Next action

Resolve JB-001: choose the first representative LLC workflow, accounting basis,
functional currencies and FX policy, opening-balance source, and accountant review criteria.
Entity/account counts, providers, GCP project, billing budget and region are
unknown. `us-central1` is a provisional discussion default, not a deployed choice.
