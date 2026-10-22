<!-- stage-runbook: true -->
# Position contract delivery — #248

## Status and boundary

Lease POSITION-CONTRACT-248-20261021T2325Z on branch `issue/248-position-contract`.
Contract/research only; no SQL, OpenAPI, browser, migration, live data or cloud
mutation. Owner approved first-screen scope/visual direction on 2026-10-22. Cash
membership, real-book policy and consolidation remain separate unapproved scope.

## Stage ledger

| Stage | Evidence | Status |
| --- | --- | --- |
| Dependency review | Read #247/#248, #1/#51/#53 status, docs index, bibliography, decision-evidence, database foundations and reporting contracts/runbooks. | Complete |
| Contract choice | Coordinator agreed to existing balance-sheet authority, explicit classified cash IDs, point comparison, and separate attention states. | Complete |
| Fixed examples | Hand-fixed two-date and reversal totals, original EUR and functional USD, empty/denied/stale/source-only cases. | Complete |
| Validation | Local link/whitespace review completed. `make check test build` attempted; all blocked before execution by Go 1.26.10 toolchain download/permissions. | Blocked by environment |
| Handoff | Contract, source record and navigation links prepared for draft PR; unresolved decisions identified. | In review |

## Current evidence

The [Position contract](../contracts/position-read-model.md) now specifies the
owner-approved All books overview grouped by entity with scope narrowing and
direct Position/Ledger/Reports navigation. Each row uses the existing single-book
balance-sheet comparison for assets/liabilities/book equity/equity change. Browser
orchestration permits at most four in-flight reads, current membership discovery,
per-request authorization, independent row snapshots and partial unavailable
states. There is no cross-book sum, FX or common snapshot claim. Unsupported
policies beyond today's synthetic USD profile remain unavailable.

The [fixed examples](../research/position-sources.md) now include independent
entity/book/revision/date metadata, exact A/B/empty results, unsupported EUR,
partial failure, revocation, stale response, five-book scheduling and late-result
rejection. These are acceptance constants and planned cases, not present runtime
coverage. Owner approval concerns design/scope only; cash, attention and real-book
policy remain deferred. Clean sans-serif text, green/red signed equity changes
and alternating faint gray/unshaded rows supersede earlier column-tint proposals.

## Validation record

On 2026-10-21, `make check`, `make test`, and `make build` each stopped at the
pinned Go 1.26.10 toolchain download before any project check/test/build ran.
`make check` was retried outside the sandbox and through `goproxy.cn`; the
alternate download reset its TCP connection. This is an environment gate, not
passing product evidence. Local Markdown relative links and whitespace were
reviewed separately; no SQL/API/browser files changed, so migration, ledger,
web and app-integration gates do not apply to this documentation-only commit.
The first PR CI check reached Go and found that the annotated bibliography had
crossed its 32 KiB context limit. Its index entry was condensed below the cap;
the follow-up CI run is the validation source for this correction.

## Remaining work

- Owner/accountant must approve real-book cash classification; #1 remains open.
  Position vocabulary/scope is visually approved, with implementation evidence pending.
- #249 can render existing statement totals and honest unavailable states;
  server cash membership and attention feed need separate approved contracts,
  authorization and named tests before showing computed cash or `All clear`.
- Full subtotal-to-journal navigation is not provided by the pinned single-line
  reader. Show account contributors first or design an additional bounded API.
- Existing reporting caps, synthetic USD functional policy and revision refresh
  remain visible limitations. No ISO/GAAP/audit-export conformance claim.

## All-books stage — 2026-10-22

Live #248/#249 and their scope comments were reviewed. Owner conversation approved
implementation of the latest all-books layout. Updated only contract, source
record and this marked stage runbook; no API/SQL/UI or shared navigation files.
Coordinator owns integration and browser acceptance.

`make check test build` passed locally for this documentation change with Go
1.26.5 using the existing temporary modfile
`/private/tmp/jayburd-investigate-255-go.mod` and cache
`/private/tmp/jayburd-investigate-go-cache` (`GOTOOLCHAIN=local`). Repository
`go.mod`/`go.sum` were unchanged. This workaround validates project doc links,
context budget, migration file integrity, stage structure, vet, race tests and
builds; it does not claim execution under the pinned Go 1.26.10 toolchain or
browser acceptance for #249. `git diff --check` also passed.

## PR handoff

Coordinator owns any PR and integration. This branch changes contract, source,
runbook and navigation files. No migration impact. Required Go gates
must be rerun when the pinned toolchain becomes available; owner vocabulary
approval covers visual/scope only.
