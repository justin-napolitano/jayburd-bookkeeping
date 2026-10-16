# Project context

## Decision

Build actual bookkeeping software, learning through small, testable slices.
The owner selected Go, PostgreSQL, Next.js, GCP, transactions and outbox-driven
background work. The two predecessor repos are reference material only.

## Read by task

| Question | Authoritative document |
| --- | --- |
| How are per-account bank credentials encrypted? | [Encrypted credential contract](contracts/bank-token-vault.md) and [delivery ledger](runbooks/bank-token-vault.md) |
| How does Connect account prepare a new creator-owned book? | [Self-service onboarding](contracts/self-service-connections.md) and [delivery evidence](runbooks/self-service-connections.md) |
| How does the operator provision the first Sandbox principal and book? | [First Sandbox book bootstrap](runbooks/sandbox-book-bootstrap.md) |
| What is the next product/data-collection sequence? | [Delivery milestones](plans/product-and-ingestion.md) |
| What must pass before live sync, and how will scaling be measured? | [Sync quality gates](plans/sync-quality.md) and [validation evidence](runbooks/sync-quality.md) |
| What works now, and how do I review it? | [Project status and frontend guide](project-status.md) |
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
| How does the frontend authenticate to the private API? | [Runtime identity and private bridge](contracts/private-api-bridge.md) |
| How are application images packaged and published? | [Image delivery contract](contracts/image-delivery.md) |
| How does Cloud SQL connect and migrate safely? | [Cloud SQL contract](contracts/cloud-sql.md) |
| How do we prove actual API database permissions? | [Bounded permission probe and operator procedure](runbooks/database-permission-probe.md) |
| Is the development database provisioned? | [Cloud SQL provisioning evidence and next gates](runbooks/cloud-sql-bootstrap.md) |
| Has Cloud SQL role bootstrap succeeded? | [Cloud Shell bootstrap and current operator ownership](runbooks/cloud-shell-bootstrap.md) |
| How is the GCP foundation validated and bootstrapped? | [Foundation operations](gcp-foundation.md) |
| How do we start, review and finish tasks? | [Delivery workflow](delivery.md) |
| What are the proposed reporting queries and exports? | [Financial reporting contract](contracts/financial-reporting.md) |
| How will the financial review workspace look and behave? | [Workspace design](plans/financial-workspace.md) and [synthetic prototype](prototypes/financial-workspace.html) |
| What implements durable import confirmation and its Go storage boundary? | [Import SQL](contracts/import-sql.md), [Go orchestration](contracts/import-service.md), and [delivery evidence](runbooks/durable-import-delivery.md) |
| What persists synthetic source identities and private evidence? | [Source registry](contracts/source-registry.md) and [import staging](contracts/import-staging.md) |
| What is the first proposed statement-import format? | [Synthetic statement-import contract](contracts/statement-import.md) |
| How will Plaid pulls and webhooks feed the same ingestion core? | [Provider ingestion proposal](contracts/provider-ingestion.md) |
| How will outbox delivery, leases and sync recovery be proved? | [Worker contract review](runbooks/outbox-contract-review.md) and [provider/worker integration review](runbooks/provider-worker-review.md) |
| What is the reporting SQL/service implementation boundary? | [Reporting SQL contract](contracts/reporting-sql.md) and [foundation evidence](runbooks/reporting-foundation.md) |
| How will source evidence become a reviewed draft? | [Durable import proposal](contracts/durable-import.md), [review journey](plans/import-review.md), and [prototype](prototypes/import-review.html) |
| How does Go drive observer SQL without a queue transaction? | [Runtime adapter contract](contracts/observer-runtime.md) and [delivery evidence](runbooks/observer-runtime-delivery.md) |
| What proves the first receipt-only worker? | [Observer contract](contracts/worker-sql.md) and [integration evidence](runbooks/ingestion-foundations.md) |
| What can the synthetic CSV decoder validate? | [Decoder contract](contracts/statement-decoder.md) and [test evidence](runbooks/statement-decoder.md) |
| What tables and privileges would the first worker need? | [Proposed worker SQL contract](contracts/worker-sql.md) |
| What did the parallel foundation build verify? | [Integration evidence and remaining gates](runbooks/build-foundations.md) |
| What connects financial reports to HTTP and browser review? | [Reporting delivery evidence](runbooks/reporting-delivery.md) |
| What adds comparisons, journal detail and internal CSV? | [Review tools delivery](runbooks/report-review-tools.md), [pinned journal contract](contracts/report-drill.md), and [CSV wire contract](contracts/report-export.md) |
| What is the report wire format? | [HTTP contract](contracts/reporting-http.md) and [transport evidence](runbooks/reporting-http.md) |
| How are deleted and disabled identities tested? | [Posting identity evidence](runbooks/posting-gate.md) |
| What did the parallel design sessions deliver? | [Integration review and remaining gates](runbooks/parallel-contract-review.md) |
| What worked in Brainblast? | [Tooling assessment](research/brainblast.md) |
| How are identities enrolled and books created? | [Identity/setup contract](contracts/identity-setup.md) and [delivery evidence](runbooks/identity-setup.md) |
| How does the first application run? | [Server contract](contracts/server.md), [browser sign-in](contracts/browser-auth.md), [first UI](contracts/first-ui.md) and [application evidence](runbooks/first-app.md) |
| How do postings recover after losing a browser session? | [Durable command contract](contracts/command-recovery.md), [source review](research/recovery-sources.md) and [integration evidence](runbooks/command-recovery.md) |
| What has actually passed? | [PostgreSQL posting proof](runbooks/posting-proof.md) |
| How can I try generated books locally? | [Synthetic demo and cent checks](runbooks/synthetic-demo.md) |
| What was checked for the contract package? | [Ledger contract runbook](runbooks/ledger-contracts.md) |

For the generated 3,000-transaction review workflow, see the
[populated demo delivery](runbooks/populated-demo.md).

## Synthetic test data

Use the [corpus and execution guide](../tests/corpus/README.md),
[annotated test sources](research/testing-sources.md), and
[corpus evidence runbook](runbooks/synthetic-corpus.md).

## Next action

Hosted acquisition is implemented for review on top of merged PR #109. Read the [hosted delivery runbook](runbooks/hosted-sandbox.md) and
[platform contract](contracts/hosted-platform.md) for tested behavior and remaining
activation steps. Worker/IAM and provider adapters have synthetic evidence;
Infisical account configuration, real hosted acceptance and full Plaid Sandbox
qualification remain open in #7/#11/#88. [Link and controlled worker delivery](runbooks/link-worker-delivery.md)
adds the actual browser SDK adapter and paused worker promotion workflow for review.
Qualified reconnect and hosted acceptance remain follow-ups. No production account activation.

Review the [first application evidence](runbooks/first-app.md), then complete the
managed identity and deployment gates before connecting live accounts. JB-001 remains open for a real
LLC policy, opening-balance source and accountant cutover review. Development
accrual/USD defaults do not resolve those live-use decisions.
Entity/account counts, bank providers and application authentication configuration
remain unknown. Development project `jayburd-bookkeeping-dev` (`9088794002`)
and its billing link/foundation are established; see the delivery evidence ledger. The owner accepted an under-$50/month synthetic development target;
`us-central1` remains provisional. See [GCP delivery](gcp-delivery.md),
[epic #25](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/25) and
the [delivery evidence ledger](runbooks/gcp-delivery.md).

- [Cloud Run rollout contract](contracts/cloud-run-rollout.md) and [release review instructions](../deploy/README.md).

Acquisition delivery: [SQL ownership](contracts/acquisition-dispatch-sql.md), [Go runtime](contracts/acquisition-dispatch-runtime.md), [transport profile](contracts/acquisition-transport.md), [delivery runbook](runbooks/acquisition-dispatch.md), and [bounded HTTP host](runbooks/worker-host.md).

Hosted integration: [delivery evidence](runbooks/hosted-sandbox.md),
[worker composition](contracts/hosted-workers.md), [durable evidence](contracts/hosted-evidence.md),
[provider vault](contracts/provider-vault.md), [optional API composition](contracts/hosted-connections.md),
and [Sandbox diagnostic boundary](contracts/plaid-sandbox-qualification.md).

Application promotion now requires [explicit connection configuration](contracts/connection-release.md);
[delivery evidence](runbooks/connection-release.md) distinguishes reviewed metadata
from actual provider qualification.

- [Hosted Sandbox probe](runbooks/hosted-sandbox-probe.md): credential delivery,
  disposable API-created Item test, and explicit hosted evidence limits.

- [Sandbox Hosted Link enrollment](runbooks/hosted-link-enrollment.md): explicit
  session completion, account binding and synthetic application journey evidence.
