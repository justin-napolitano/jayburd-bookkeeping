# Ledger and API contract review

<!-- stage-runbook: true -->

## Status and boundary

Contract-only stage for JB-001/JB-002 from merged PR #15 (`72d30bf`). Owner
accepted named domain schemas and requested SQL/API contract drafting. Policy
choices and implementation acceptance remain distinct. No product DDL, database
connection, application code or live financial data.

## Stage ledger

| Stage | Result | Next action |
| --- | --- | --- |
| Check base and issue dependencies | PR #15 merged; JB-001 policy remains open | Draft contracts without claiming dependency completion |
| Source review | PostgreSQL namespaces, OpenAPI, HTTP problems and public AASB currency guidance located | Record exact review scope and contract implications |
| Draft contracts | Namespace ADR, scenarios, money rules, SQL dictionary/operations and OpenAPI written | Cross-check representations and validate |
| Review consistency | Corrected worked-month cash totals and distinguished row-locking reads from READ ONLY transactions | Check examples, schema and document links |
| Validate review artifacts | Go checks and OpenAPI validation passed; independent scenario arithmetic passed | Owner/schema/API review |

## Current evidence

Working tree was clean before creating `issue/2-ledger-api-contracts` from
origin/main. Source access is public only.

- `GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache make check test build`: PASS, including local links, existing race tests, migration-file checks and tooling build.
- `npx --yes @redocly/cli@2.8.0 lint api/openapi.json`: PASS, including the embedded synthetic request; one documented missing-license metadata warning. No license grant selected. Initial conditional-schema lint warnings were corrected. Validator was downloaded after sandbox DNS failure; no runtime dependency added.
- Reviewed arithmetic with Python standard-library Decimal in an ephemeral command: all 14 golden-month journals balance, checking 1,045 plus savings 100, trial-balance sides 1,300, income 70. Full/partial FX settlement and conversion-fee arithmetic PASS. No Python file or application dependency added.
- `git diff --check`: PASS before handoff; stage guard is required before commit.

These checks validate documents, schema shape and arithmetic, not a PostgreSQL
ledger implementation, real HTTP server, accounting compliance or ISO conformance.
Named product acceptance tests in the contracts remain unimplemented.

## Remaining work

Owner/accountant must select reporting basis and approve functional-currency,
rounding, rate-source and opening-balance policy. Schema reviewer must accept
function/identity/privilege and transaction proposals before product DDL. Close
evidence, subledger settlement/revaluation, reconciliation and import/export
physical contracts remain additional slices. These gaps are explicit in the
SQL/API documents; neither JB-001 nor JB-002 is closed by this draft.

## PR handoff

Review schemas, posting boundary, money/FX representation, HTTP interface and
acceptance scenarios together. Merge of a design draft is not ledger approval.
Start with ADR 0004, the accounting scenario/policy register, the SQL dictionary
and `api/openapi.json`. Related to #1 and #2; closes none. No migration or deployed
API change. The next implementable slice is manual identity/spot posting and
authorized reads after its policy and enforcement gates are accepted.
