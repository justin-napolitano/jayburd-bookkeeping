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
| How do we manage work and context? | [Knowledge and issue ownership](decisions/0002-project-memory.md) |
| What must the ledger guarantee? | [Accounting contract](contracts/accounting.md) |
| What must retries and workers guarantee? | [Background work contract](contracts/background-work.md) |
| How are secrets managed? | [Infisical setup](infisical.md) |
| How will builds reach GCP? | [GCP delivery contract](gcp-delivery.md) |
| How do we start, review and finish tasks? | [Delivery workflow](delivery.md) |
| What worked in Brainblast? | [Tooling assessment](research/brainblast.md) |
| What has actually passed? | [Seed runbook](runbooks/seed.md) |

## Next action

Resolve JB-001: choose the first representative LLC workflow, accounting basis,
initial currency, opening-balance source, and accountant review criteria.
Entity/account counts, providers, GCP project, billing budget and region are
unknown. `us-central1` is a provisional discussion default, not a deployed choice.
