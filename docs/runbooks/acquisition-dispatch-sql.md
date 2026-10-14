<!-- stage-runbook: true -->
# Acquisition dispatch SQL proof

## Status and boundary

Migration 15 implements the scoped delivery sidecars in
[the SQL contract](../contracts/acquisition-dispatch-sql.md). This is synthetic
foundation evidence, not a Cloud Tasks deployment or live provider qualification.
Migrations 1–14 remain unchanged. No provider, secret or cloud call was performed.

## Stage ledger

- Contract reviewed with coordinator, Go adapter author and independent reviewer.
- Implemented current authorization, generation discovery, deterministic queue
  identity, original/derived lease inspection, receipt recovery and quarantine.
- Fixed independent review findings: one timestamp for the exact 24-hour deadline;
  due-only candidate selection; next dispatch created with committed retry handoff.

## Current evidence

- `bash scripts/test-sync-dispatch.sh`: PASS, 7 actual PostgreSQL test groups:
  initial generation without webhook, false completion hint, derived live lease and
  committed receipt, unknown failure quarantine, eight failures with one-hour
  Retry-After, permanent late-publication fence, expired lease/new claim, current
  permission revocation and immutable receipts. Tests use real restricted sessions.
- Temporary copied runner is coordinator-owned and excluded from this commit.
- Test scheduling advances use owner-only synthetic fixture updates; no product
  clock override exists. No financial journal is written by delivery.
- `make check`, `make test`, `make build`: PASS. The first sandboxed test run could not bind a synthetic loopback listener; the approved rerun passed.
- `bash scripts/check-stage-runbook.sh --staged`: required before commit.

## Remaining work

Coordinator integration, HTTP target profile 2, fairness and dependent database
regressions passed; see the [delivery evidence](acquisition-dispatch.md).
Cloud provisioning, publisher scheduling and live adapter qualification are not
proven here. Quarantine reset requires a separate explicit operator contract.

## PR handoff

SQL-only lane for #7; follow the contract's current-authority and unknown-outcome
rules. Do not treat successful task creation as completed acquisition. Preserve
separate acquisition and queue generations and inspect both restart leases.

The age bound prevents new dispatch/execution; it does not timestamp publication.
An already committed completion remains authoritative when discovered after cutoff.

Follow-up review tightens SQL CHECK constraints to reject partially null execution,
age and terminal metadata; the named null-state negative test verifies seven invalid
owner fixture updates cannot bypass PostgreSQL's three-valued CHECK semantics.
Follow-up evidence: all 8 actual PostgreSQL groups PASS, including the new seven-case
null-state constraint test; `make check` PASS. Prior default test/build evidence
remains applicable (this follow-up changes only DDL and its synthetic proof).
