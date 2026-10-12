# First bookkeeping application

<!-- stage-runbook: true -->

## Status and boundary

Implemented for review under #3/#6/#9, following the owner's parallel-agent request.
PR #22 targets `main` after confirmed merge of #21 (`aaebd92`), which retains
the `e4e33e7` baseline. No live credentials, financial data or cloud mutation.
Manual development bookkeeping is the scope; import/reconciliation remain open.

## Stage ledger

| Stage / owner | Scope | Evidence / next action |
| --- | --- | --- |
| Server agent | Go executable, workload identity composition, bounded workspace discovery, migration 000004 | Unit/race, workspace and runtime-role database proofs pass; isolated runtime image proof passes |
| Login agent | In-memory managed sign-in, TOTP enrollment/challenge, safe logout/session invalidation | 17 synthetic auth tests pass; concurrent logout regression fixed |
| Frontend agent | Next.js setup, split journal review/post, trial balance and retry UI | Types, 33 unit tests, build, audit and three Chromium journeys pass |
| Coordinator | Shared API, dependency integration, CI, docs, review and PR | Local proof completed; merged-main hosted CI subsequently passed |

Files have exclusive ownership while agents work. Agents do not commit or change
branches. The coordinator integrates into one reviewable branch and updates this
same runbook before every commit. GitHub remains task authority; no external vault
or additional orchestration service is required.

## Current evidence

PASS: migration fresh/replay, rollback/dirty-state and ledger forward-upgrade proof;
complete ledger suite and synthetic restore replay; `TestWorkspaceAuthorization`,
`TestWorkspaceRevocation`, `TestWorkspaceBounds`; `TestRuntimeReadiness` rejects
table access, enrollment authority, schema DDL, owner/operator roles and missing
workspace permission. Initial migration-version assertion corrected from3 to4.

PASS: OpenAPI (Redocly2.62.1, existing missing-license warning), actionlint1.7.7,
govulncheck1.1.4 (zero reachable/imported vulnerabilities; one unimported module-only
finding remains from the baseline). `make check test build` passes, including
race tests, vet, context links, migration integrity and executable builds.

PASS: web `api:check`, `check`, `test` (33 tests), `build` and `test:browser`
(three Chromium journeys: setup, split/review/unchanged retry, denied scope).
Compatible TypeScript5.9.3
and openapi-typescript7.13.0; patched gRPC override removes npm advisories.
Review fixed concurrent logout, inherited PG overrides, stale report completion,
uncertain-denied retries, user-facing recovery keys and RFC9457 admission errors.
Clean-copy `check` now generates Next route types before TypeScript; it passes
without `.next` output or a previous build.

GitGuardian incident38110406 flagged adjacent environment-variable names in
`TestManagedConfiguration`, not credential values. The fixture now uses an explicit
set of names with empty values; all presence-denial assertions remain. No detector
or path exclusion was added. PR rescanning retained only the original historical
occurrence; the PR discussion records its false-positive disposition requirement.

PASS: `scripts/test-api-container.sh --local-build` builds the static executable
with the pinned host toolchain, proves non-root/read-only/internal-network runtime,
health200/readiness503/absent-ledger404 and graceful stop. The canonical Docker
source stage then PASSed on merged main56d7c38 in
[hosted run38170006206](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38170006206).
That run's backend and frontend jobs passed. The local Docker proxy CA limitation
remains environment-specific; verification was never disabled.

## Remaining work

PR #22 is merged. Continue with [durable recovery and integration](command-recovery.md).
Live provider verification,
MFA enrollment/recovery, GCP project/IAM/ingress, Infisical injection, real-LLC policy
and official-bookkeeping approval are not supplied by passing synthetic tests.

## PR handoff

Review [identity/setup](../contracts/identity-setup.md) and the new server/browser/UI
contracts when delivered. Preserve merged migration history; use 000004 for new
workspace reads. Record exact tests, deployment boundaries and outstanding gates.
Unknown-commit keys/bodies live only in browser memory; logout/tab loss requires
operator verification before re-entry. This blocks live-use readiness. Multiple
TOTP factors need operator recovery. No imports, persisted drafts, journal list,
reversal UI or cloud deployment is supplied by this slice.
Historical limitations above describe #22; the next slice owns their resolution.
