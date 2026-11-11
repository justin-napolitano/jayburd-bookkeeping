<!-- stage-runbook: true -->
# Synthetic readiness proof — 482

## Status and boundary

Owner direction2026-11-11: prove the implemented system with synthetic data first,
then revisit real data. GitHub482 owns acceptance;65 owns the source/test claim
through23:00Z. Dedicated test/482-synthetic-readiness from main0a7d3731, which
merges the accepted customer UI release documentation. No real data, secrets or
provider calls; no consent renewal, managed service/job/SQL mutation, cloud
restore, IAM/new resources or deployment. All database fixtures are disposable
and isolated. Existing product SQL1–47 immutable;48 unallocated.

## Proof design before implementation

Reuse the reviewed source/posting/recovery/reports/worker/identity contracts,
research index, fixed corpus and standard test runners. Test-only evidence does
not change a financial message, valuation/currency mapping or accounting policy;
existing standards applicability is unchanged. Expected money/counts come from
independently authored fixtures, never decoded/provider/generated output.
Identity equality may compare retained pre/post receipts; exact money still
requires independent expectations. Never revive manual customer posting.

The existing ledger restore check proves a low-level post receipt cannot repost.
Audit and extend proof of the retained source→review/draft→saved command→linked
entry→report chain across a synthetic restore. Exercise prepared and committed
unacknowledged recovery if existing supported APIs permit it. Reuse real restricted
SQL and source-backed posting rather than a mocked ledger or privileged manual
journal shortcut. Preserve existing tests and rollback/cleanup guards.

Audit bounded volume honestly: 3,000 imported observations and one linked entry
do not prove1,000 source-backed postings or an availability SLA. If missing,
add meaningful bounded source-backed volume/replay evidence with fixed oracles,
without timeout inflation or reintroducing the retired manual browser runner.

## Acceptance matrix

| Area | Required evidence | Current state |
| --- | --- | --- |
| Metadata/transport | Current deployed immutable revisions/health and anonymous200/401/403, no financial bodies | Selected metadata observed healthy; four fresh anonymous transport checks pass; not private-data acceptance |
| Source through reports | Real browser→Next→Go→restricted synthetic PG, original units/source identity, one linked posting, exact reports/CSV/drill | Fresh baseline main CI running |
| Replay/unknown outcomes | Same request/body/key/receipt, no duplicate journal/outbox after lost replies/retries/reordering | Fresh named proofs pending |
| Restore | Synthetic source/evidence/commands/journal/report identity and totals survive isolated dump/restore; replay safe | Audit/extension pending |
| Isolation/permissions | Actual restricted roles, tenant/book scope, revocation and current-authority fences | Fresh named proofs pending |
| Worker/webhook | Synthetic signed delivery/replay, leases/demand/generation fences, no unreviewed journal | Fresh named proofs pending |
| Money/currency | Fixed exact cents, precision/FX/original units and unsupported/stale/closed-period holds | Fresh named proofs pending |
| Volume | Explicit workload/count/oracles/replay and measured duration; no universal scaling claim | Audit/extension pending |
| Browser/recovery |98 existing journeys, five-width layout, explicit confirm/cancel/ack and no idle debugging | Fresh frontend baseline running |
| External/unfinished | Plaid/Firebase/OAuth/provider history and real-bank reconciliation; documents476 and auto-post334 | Not cleared by synthetic tests; feature gaps stay explicit |

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Fresh main/register read.481 all five checks green, merged0a7d3731. Seed43078080938 and Infrastructure43078080924 automatically started on exact main. |
| Authority | Owner explicitly selects synthetic-only. No real-account/Plaid consent request is needed for this phase; earlier expired consent stays unchanged. |
| Dispatch | Root coordinates runbook/CI/transport and gap decisions; disjoint restore, application coverage and security/volume lanes. No product/runtime edits without coordinator review. |
| Local baseline | `make check build` passed using local Go1.26.5 and an untracked exact-dependency modfile; canonical Go1.26.10 remains unchanged. `make test` passed18 release guards and most race packages, but socket-dependent acquisition/webhookhost/workerhost/workerruntime tests hit sandbox bind EPERM. Overall local test gate is BLOCKED, not PASS; use unchanged canonical CI. |
| Anonymous transport | Fresh2026-11-11 root/auth-config200, proxied workspace401 and direct API workspace403. IPv4 curl supplied empty Authorization/Cookie/serverless-auth headers, followed no redirects and discarded all bodies. No private application/provider acceptance inferred. |
| Implementation | Added `TestProviderSourceRestore/{prepared,committed}` and `TestSourceBackedPostingVolume1000`. Runner creates a pristine migrated template; each new test clones its own disposable database. No runtime/schema/dependency edits. Root independently reviewed restricted API effects, fixed oracles, cleanup and dump boundary. Integration-tag compile/vet and shell syntax passed. |
| Local integration boundary | Standard provider-flow runner stopped before resource creation: Docker public-image credential-helper error; an explicitly empty task Docker config still could not access the daemon. No resource/dump created, no restricted-SQL PASS claimed. No privilege/host/guard/timeout changes; canonical CI required. |
| Object recovery scope | PostgreSQL dump/restore retains original source object reference/hash and the complete source/review/command/entry graph. Original fictional bytes are a separate explicit recovery input, reinstalled through the storage API and compared byte-for-byte/hash-for-hash. This does not prove managed object backup/PITR or key recovery. |

## Current evidence

Baseline code matches deployed source9b2a656; only merged documentation differs.
Prior successful releases are historical evidence, not this fresh acceptance.
Some local socket/port proofs were previously restricted; use standard runners
and canonical CI, never hosts/guards/credentials workarounds.

## Remaining work

Complete named fresh proofs, fill the matrix with exact source/run/assertions,
execute the new restore/volume coverage in canonical CI. Fix actual
defects without weakening expectations. Publish redacted receipts and preserve
FAIL/NOT IMPLEMENTED/EXTERNAL VALIDATION as distinct outcomes. No production,
real-bank or full financial-framework conformance claim follows from these tests.

## PR handoff

Pending synthetic evidence. GitHub58/84/88/89/90 keep their actual scope and external
acceptance. No old issue is closed simply because a generic CI job is green.
