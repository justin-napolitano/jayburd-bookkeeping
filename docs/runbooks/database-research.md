# Database research review

<!-- stage-runbook: true -->

## Status and boundary

Research-only branch from main. No application, SQL migration, dependency or CI
changes. Migration PR #14 remains separate and unmerged. Public sources only;
no financial data or credentials accessed.

## Stage ledger

| Stage | Result | Next action |
| --- | --- | --- |
| Discover and read source sections | 22 source entries with review depth/limits | Compare alternatives |
| Critique proposed paradigms | Decision brief and unresolved choices recorded | Owner/schema review |
| Establish evidence standard | Source-to-inference-to-test record specified | Apply to subsequent ADRs |

## Current evidence

Targeted crawl covered OpenStax, BCcampus, original relational/SSI/accounting
papers, PostgreSQL 17 manuals, author pattern articles, provider documentation
and tool alternatives. DDIA is explicitly publisher-preview-only. Old arXiv HTML
was unavailable; PDF sections were readable. PDF title-page metadata was checked
where extraction gave an inconsistent title. No source text archive created.

`GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache make check test build`: PASS, including local document links,
runbook structure, existing Go race tests, vet and tooling build. These checks
validate the repository, not the literature's conclusions. DOI resolver returned
403 for Codd; the university-hosted paper remained readable. No accounting
correctness, concurrency benchmark or operational approval is claimed.

## Remaining work

Resolve JB-001 accounting policy, then review JB-002 logical/physical/transaction
contracts. Posting enforcement and isolation remain proposals pending evidence.

## PR handoff

Review source interpretation and the next design gate. Keep implementation paused
until the relevant slice's contract and evidence plan are accepted.
