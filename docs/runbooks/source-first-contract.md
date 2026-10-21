<!-- stage-runbook: true -->
# Source-first contract stage record

## Status and boundary

Branch `issue/238-source-first-contract`, issue #238, base
`6a921869f7af22e15ade0cd567c9114b34739075`. This is the single marked
stage runbook for this delivery branch. Product direction was accepted by the
owner; the material contract remains proposed until reviewed. No source-provider
call, private data, credential access, cloud mutation, SQL migration or posting
implementation is part of this branch.

## Stage ledger

- Read `docs/index.md`, `docs/research/bibliography.md`, the database foundations,
  decision-evidence and standards requirements, #238 and its #65 lease.
- Compared existing `connection-sql`, `provider-sync-sql`, `source-review-sql`,
  identity and accounting contracts. Current Item/source/review paths are
  entity/book-scoped, so bookless acquisition is a material contract change.
- Three parallel read-only reviews covered provider/storage, global review/
  taxonomy, and accounting/identity. They made no file or issue edits.
- A second read-only accounting/identity review found four wording risks;
  corrected grant, reconciliation, cross-entity and standards applicability
  language before handoff.
- Refreshed selected public Plaid Link, Accounts and Transactions sections in
  PI13. No provider payload or live account was inspected.

## Current evidence

- ADR 0006 defines private collection, stable source identity, explicit legal
  owner assertion, entity-owned books, inert topics/suggestions and immutable
  posted corrections. The companion contract names operation authority and
  planned synthetic acceptance scenarios.
- #239 remains blocked pending owner/identity/database review. #244/#1 remain
  the policy gate for cross-entity posting; this branch does not authorize it.
- Before a code-changing child starts, reconcile additive migration, consent/
  retention, current grants, standards mapping and existing bound Item behavior.
- `make check`, `make test` and `make build` passed with locally installed Go
  1.26.5, a temporary modfile outside the repository and temporary build cache.
  The pinned 1.26.10 download failed even after an authorized retry because the
  proxy socket was unavailable. The first sandboxed `make test` run failed when
  localhost test listeners could not bind; the authorized unsandboxed rerun
  passed all race tests and migration-file checks. No repository `go.mod` or
  `go.sum` was changed.

## Remaining work

Pending: staged-runbook check before commit and final diff review. A passing
build validates documentation branch mechanics only, not provider or accounting
acceptance.

## PR handoff

Pending commit/PR. Migration impact: none. Record validation results and
unresolved contract decisions before review.
