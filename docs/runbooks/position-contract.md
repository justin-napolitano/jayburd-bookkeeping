<!-- stage-runbook: true -->
# Position contract delivery — #248

## Status and boundary

Lease POSITION-CONTRACT-248-20261021T2325Z on branch `issue/248-position-contract`.
Contract/research only; no SQL, OpenAPI, browser, migration, live data or cloud
mutation. Owner review of vocabulary/cash membership remains a gate, not an
inferred approval.

## Stage ledger

| Stage | Evidence | Status |
| --- | --- | --- |
| Dependency review | Read #247/#248, #1/#51/#53 status, docs index, bibliography, decision-evidence, database foundations and reporting contracts/runbooks. | Complete |
| Contract choice | Coordinator agreed to existing balance-sheet authority, explicit classified cash IDs, point comparison, and separate attention states. | Complete |
| Fixed examples | Hand-fixed two-date and reversal totals, original EUR and functional USD, empty/denied/stale/source-only cases. | Complete |
| Validation | Local link/whitespace review completed. `make check test build` attempted; all blocked before execution by Go 1.26.10 toolchain download/permissions. | Blocked by environment |
| Handoff | Commit SHA and unresolved decisions reported to coordinator. | Pending |

## Current evidence

The [proposed Position contract](../contracts/position-read-model.md) can drive a
first UI slice **now** using the existing balance-sheet report: show total assets,
liabilities, direct equity, unclosed earnings and total equity for a chosen date,
plus signed point comparison in one report response. Show `Cash: Not configured`
until account membership is reviewed and a server-owned cash extension exists.
It does not require a fake cash estimate or a new accounting engine in the UI.
The [fixed examples](../research/position-sources.md) are proposed acceptance
constants; current reporting tests prove their own older synthetic fixtures,
not this new Position fixture.

## Validation record

On 2026-10-21, `make check`, `make test`, and `make build` each stopped at the
pinned Go 1.26.10 toolchain download before any project check/test/build ran.
`make check` was retried outside the sandbox and through `goproxy.cn`; the
alternate download reset its TCP connection. This is an environment gate, not
passing product evidence. Local Markdown relative links and whitespace were
reviewed separately; no SQL/API/browser files changed, so migration, ledger,
web and app-integration gates do not apply to this documentation-only commit.

## Remaining work

- Owner/accountant must approve real-book cash classification and Position
  vocabulary; #1 remains open. No live policy is established here.
- #249 can render existing statement totals and honest unavailable states;
  server cash membership and attention feed need separate approved contracts,
  authorization and named tests before showing computed cash or `All clear`.
- Full subtotal-to-journal navigation is not provided by the pinned single-line
  reader. Show account contributors first or design an additional bounded API.
- Existing reporting caps, synthetic USD functional policy and revision refresh
  remain visible limitations. No ISO/GAAP/audit-export conformance claim.

## PR handoff

Coordinator owns any PR and integration. This branch changes only the three
new contract/research/runbook files. No migration impact. Required Go gates
must be rerun when the pinned toolchain becomes available; owner vocabulary
review remains open.
