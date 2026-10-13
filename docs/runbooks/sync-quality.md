# Sync quality planning and parallel validation

<!-- stage-runbook: true -->

## Status and boundary

Owner requested quality issues and parallel runnable checks. Base d22b748 from
PR #83; GitHub still reported OPEN at dispatch, despite the owner's merge message.
This branch plans acceptance and reruns existing synthetic foundations. New
provider, end-to-end ingestion and scaling suites are not implemented here.

## Stage ledger

- Read repo instructions, ADR0002, existing epics and provider/corpus boundaries.
- Created #84–#90 with named acceptance, dependencies, candidate scopes and
  estimates. Reused #64 for operational target design instead of duplicating it.
- Reviewed official Plaid and Google Cloud documentation; recorded source sections
  and distinguished proposed scaling thresholds from measurements.
- Dispatched three read-only leases in #65 on isolated d22b748 worktrees:
  ledger/corpus (#84), observer/registry (#85), staging/decoder/fuzz (#86).
  No tracked agent edits, provider requests or cloud mutations authorized.

## Current evidence

At exact runtime SHA d22b748, separate worktrees ran:

- `make ledger-test`: PASS, 76 top-level executions (75 unique names; restore
  replay repeated), 174 subtests, no skips. Includes 1,000-posting reports and
  actual dump/restore. Corpus checks passed for two golden-month and three FX
  books. Log `/private/tmp/jayburd-quality-ledger.log`.
- `go test -race -count=1 -v ./internal/ingestion/... ./cmd/corpus`: PASS,
  26 tests, 94 subcases and seven fuzz seeds; no race reports or skips.
- `go test ./cmd/corpus -run '^$' -fuzz '^FuzzDecimal$' -fuzztime=60s -parallel=2`:
  PASS, 7,585,439 executions, seven initial seeds, 159 new interesting inputs
  (166 total), 60.564s package duration. Fresh task cache; no regression failure.
  Log `/private/tmp/jayburd-quality-fuzz.log`. This parser fuzzer shares its
  lexical predicate with the implementation; it is not an independent arithmetic
  oracle and does not satisfy all #86 properties.
- `make observer-test source-registry-test`: PASS, 32 observer groups
  (19 SQL, seven adapter unit, six PostgreSQL integration) and 13 registry
  groups, no skips/failures. Log `/private/tmp/jayburd-quality-observer.log`.
  Fake transport only; hosted queue/OIDC/provider qualification remains open.
- Root `make check test build`: PASS after the documentation-heading correction.
  Log `/private/tmp/jayburd-quality-default.log`. No application code changed.

Initial sandbox Docker helper failures were environmental; normal escalated
reruns used disposable synthetic databases. No runtime test failure found.
Initial root documentation check identified a missing required PR-handoff heading;
corrected before rerunning. Historical full frontend/application evidence lives in
[product delivery](product-workspace-delivery.md); no UI code changed here.

## Remaining work

Review the plan and proposed thresholds; native parent/dependency links are
recorded on #84–#90. All three validation leases are released at handoff. Implement #8 before claiming full statement-to-ledger proof;
#70/#7 before provider sync proof. See [quality gates](../plans/sync-quality.md).
Live pilot and hosted drills retain their explicit source/budget/identity gates.

## PR handoff

Planning and evidence only; no runtime, migration or cloud changes. All quality
issues remain open for their full acceptance. Required local checks passed;
GitHub review and merge remain separate. PR stacks on #83 if still unmerged.
