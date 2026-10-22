<!-- stage-runbook: true -->
# Plaid #88 coverage audit

## Status and boundary

Documentation and issue clarification, not full Sandbox clearance. Owner asked
for every applicable Plaid test to be explicit before a Production request.
Lease `PLAID-88-MATRIX-20261022T1500Z`, expiry17:00Z, docs-only branch
`docs/plaid-88-qualification-matrix`, worktree `/private/tmp/jayburd-plaid-fixes`,
base `8af3a1937bdfc3bf781e41dfd5d1edd0fddba09c`.

No provider request, cloud mutation, secret retrieval, consent renewal, personal
account, financial contract, SQL or application change. GitHub #88 remains open.

## Stage ledger

1. **Resume complete:** read index, quality plan/contracts/runbooks, research
   bibliography, #88 history, #85/#87 dependencies and #65; generated JB-001 seed
   packet without treating it as live scheduling authority.
2. **Source review complete:** selected official sections recorded in the
   [source register](../research/plaid-88-test-sources.md). Launch Center remains
   uninspected; no account-specific exhaustive-clearance claim.
3. **Coverage audit complete:** [matrix](../plans/plaid-88-test-matrix.md) separates
   actual hosted passes, historical synthetic tests, missing evidence, current
   implementation gaps and product/platform exclusions. Original #88 acceptance
   retained; latest #258/#268 evidence supersedes stale checking/unsupported notes.
4. **Validation complete:** `make check test build` passed. Fresh
   `go test -race -count=1 ./internal/ingestion/plaid ./internal/acquisition
   ./internal/connections ./internal/webhook ./internal/syncdispatch` passed
   all five packages. No foundation run or automatic provider UI tests.
   Existing full-stack/SQL results remain historical A/B evidence, not fresh
   runtime qualification.
5. **Issue publication complete / PR review next:** #88 body now links the
   49 case groups and17 official source records with current verified evidence,
   explicit missing cases and owners; original acceptance/history retained.
   Documentation PR records the exact commit; release lease to review afterward.

## Current evidence

Source inspection identified concrete gaps: CreateLink omits a webhook; only
Transactions/SYNC_UPDATES_AVAILABLE is accepted by webhook ingress; provider
reconnect does not have a qualified Hosted Link/expected-Item completion path.
Those facts explain BLOCKED/OPEN rows; they do not authorize speculative fixes.

Local execution used installed Go1.26.5 and the previously documented external
`/private/tmp/jayburd-88-go.mod`; tracked Go1.26.10 files remain unchanged.
The first uncached acquisition unit run was blocked by the sandbox's local-port
restriction; the approved outside-sandbox rerun passed. Initial document check
exceeded the existing32KiB bibliography budget; condensed index prose without
dropping source links or raising the limit, then reran core gates successfully.
No test failures were relabeled as provider acceptance. Diff validation and the
staged runbook guard are required before this documentation commit.

Counterexamples to overclaiming: stable repeated UI counts do not establish
exact source equality, SQL journal counts or cursor persistence; historical
readiness does not establish ninety-day completeness; public documentation
review does not establish the user's personalized Launch Center requirements.

## Remaining work

Execute the matrix in the named issue scopes with fresh #65 reservations and
bounded provider/cloud authorization where required. Publish only redacted
outcomes; keep #88/#90 open until their own acceptance evidence exists.

## PR handoff

Refs #88, #85, #87 and #104/#187; does not close any acceptance issue. Migration
impact none; rollback is documentation-only. No new ISO message mapping or
accounting policy is introduced. Passing doc/unit gates is not Plaid approval.
