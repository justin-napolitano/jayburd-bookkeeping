# Project context

Start with the [current capability map](current-capabilities.md): what is
implemented, what has passed, and what still needs owner or hosted acceptance.
GitHub owns task state; an open qualification issue is not an instruction to
rebuild its delivered implementation. The [consolidation stage](runbooks/consolidation.md)
records486. No external project-management vault is required.

## Decision

Go owns backend transport and bounded workers; PostgreSQL owns durable accounting
and atomic receipts/outbox; Next.js owns the browser. Every new application entry
requires retained source evidence. Use [AGENTS](../AGENTS.md),
[architecture](decisions/0001-architecture.md), [artifact policy](contracts/artifact-backed-accounting.md)
and [delivery workflow](delivery.md). Preserve exact money, current authorization,
immutable posted history and saved-work recovery.

## Read by task

| Task | Current contract and evidence |
| --- | --- |
| What works and what remains? | [Capability map](current-capabilities.md); current GitHub acceptance, not old narrative status |
| Latest synthetic acceptance | [482 exact-head receipt](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/482#issuecomment-6805089153), [source matrix](reviews/synthetic-application-matrix.md), [boundaries](reviews/synthetic-boundaries-and-gaps.md) |
| Source-backed posting | [Artifact policy](contracts/artifact-backed-accounting.md), [source review SQL](contracts/source-review-sql.md), [command recovery](contracts/command-recovery.md) |
| Private setup and retained-history adoption | [Onboarding](contracts/collection-onboarding.md), [source stages](runbooks/collection-onboarding.md), [controlled release](runbooks/onboarding-sandbox-rollout.md) |
| Transactions, labels, tags and rules | [Query/rules](contracts/collection-query-rules.md), [labels](contracts/collection-labels.md), [tags/recovery](contracts/collection-tags-recovery.md), [composed delivery](runbooks/query-rules-build.md) |
| Reports and exports | [Reporting](contracts/financial-reporting.md), [CSV](contracts/report-export.md), [journal drill](contracts/report-drill.md) |
| Synthetic automatic category rollout | [Current scoped release](runbooks/automatic-category-rollout.md); GitHub485 owns execution and actual results |
| Current banking UI/release evidence | [Customer UI](runbooks/customer-banking-ui.md), [web rollout](runbooks/customer-ui-web-release.md); owner visual review stays distinct |
| Local fictional review | [Demo guide](runbooks/synthetic-demo.md), [source acquisition contract](contracts/demo-acquisition.md) |
| Provider and worker boundaries | [Runtime boundaries](contracts/provider-runtime-boundaries.md), [dispatch](contracts/acquisition-dispatch-runtime.md), [worker operator guide](../deploy/WORKERS.md) |
| Hosted Plaid qualification | [88 matrix](plans/plaid-88-test-matrix.md), [source register](research/plaid-88-test-sources.md), [audit runbook](runbooks/plaid-88-test-matrix.md) |
| Identity and secret ownership | [Identity contract](contracts/identity-setup.md), [Infisical](infisical.md); agents never retrieve application secrets |
| Financial/database design or migrations | [Standards](contracts/standards.md), [database](contracts/database.md), [decision evidence](research/decision-evidence.md), [research index](research/bibliography.md) |
| Fixed independent test expectations | [Corpus](../tests/corpus/README.md), [test source register](research/testing-sources.md) |

## Historical context

The [historical implementation index](implementation-history.md) preserves the
former entry index and its wider contract/release links. [Project snapshots](project-status.md)
and dated runbooks record what was known at each delivery stage. They are useful
evidence, not a second current backlog. Corrections append or supersede history;
they do not erase original evidence.

## Synthetic test data

Use the [fixed corpus](../tests/corpus/README.md) and its independent source
register. Current source-backed acceptance is linked above; historical manual
primitive proofs do not authorize new public manual entries.

## Next action

Resume from the current GitHub issue, its exact remaining acceptance and the
capability map. Read its contracts/research and claim a bounded delivery in65
before editing. Reuse completed code and passing proof. Do not treat generated
issue seeds, an old “pending” paragraph, green CI, or a closed predecessor as
permission for deployment, provider access, real data or official-books cutover.
