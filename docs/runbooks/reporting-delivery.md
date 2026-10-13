# Reporting delivery integration

<!-- stage-runbook: true -->

## Status and boundary

Build the first authenticated reporting HTTP and browser slice on merged #75
(`30cb139`), coordinating #51–53. Audit #4's remaining identity-deletion evidence
in parallel. No migration, worker runtime, durable import, export or live data.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Dispatch | #75 merged; no open PRs; three isolated worktrees and bounded leases in #65 | Review wire contract before implementation |
| Wire review | Additive GET reports, existing SQL authority, opaque version/layout-bound cursor, exact strings; root required strict query/base64 handling | Implement transport and screens |
| Integration review | Existing bridge requires report allowlist and report-specific 2 MiB response cap; 12-second bridge budget covers bounded service | Verify actual browser/Next/Go/PostgreSQL flow |
| Posting audit | Agent `05b22a2` integrated as `43b7c90`; physical unused deletion, historical FK retention and disabled-actor denial have actual PostgreSQL evidence | Keep erasure-lifecycle decision distinct |
| Independent HTTP review | No blocking authorization/scope or exact-money defect found; reviewer requested both response-bound negatives and precise Unicode handling | Verify final fixes |
| Visual review | Root inspected synthetic desktop GL screenshot: readable controls/tables, original EUR and functional USD shown separately without clipping | Full application proof remains pending |

| Integration | HTTP `d45e9d6` and screens `603689d` integrated; root corrected ambiguous legacy date selectors and expected revision 5 (setup plus posting) | Review PR |
| Validation | Combined default/ledger/web gates and full-stack proof passed | Release leases into REVIEW |

## Current evidence

Existing foundation evidence is in [build foundations](build-foundations.md).
Root reviewed RFC 4648 sections
3.2, 3.3, 3.5 and 5 plus Go encoding/base64 documentation on 2026-10-13:
the internal cursor profile explicitly omits padding and rejects noncanonical
alphabet/encoding. Base64 conveys no authorization. SQL rechecks scope and revision.

On 2026-10-13, `make check test build ledger-test` and `make web-check web-test
web-build web-browser` passed: 130 web tests, eight Chromium journeys, exact
PostgreSQL HTTP totals, scope/revocation, cursor/response bounds and both new
principal-deletion tests. The existing 1,024-posting proof also passed. Docker
helper and browser socket restrictions required outside-sandbox reruns.

`make app-integration` passed with test-only port relocation: replace 39081,
39082, 39083 with 39181, 39182, 39183 in `scripts/test-app-integration.sh`,
`internal/server/app_integration_test.go` and `web/playwright-integration.config.ts`,
run the target, then restore those files byte-for-byte (verified no diff).
Default ports were occupied by the existing review server, which was left running.
No product code changed for relocation. The real flow reads the 125.50 capital
posting: TB debits 125.50, BS assets 125.50, P&L income 0.00 and GL closing 125.50.
It verifies one journal/receipt/outbox after lost-response recovery. Identity
remains test-only; these are not hosted or live-provider acceptance results.

## Remaining work

Review/merge this increment. Comparison controls, exports, full drill-through,
worker runtime and durable imports remain separate work. Existing report caps and
synthetic functional-currency policy remain unchanged. The broader #4 deletion/
retention lifecycle still needs explicit scope review.

## PR handoff

Implementation and local validation complete; GitHub review/CI remain separate.
No migrations or cloud/provider actions. Agent writers stopped; leases released
into REVIEW in #65. Broader issues remain open; this PR does not claim live use.
