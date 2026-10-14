<!-- stage-runbook: true -->
# Connection browser proof

## Status and boundary

Integration-tagged, disposable synthetic browser fixture for #104. No production
routes, credentials, provider calls, secret discovery or migration changes. Explicit
`-app-proof-connections` opt-in is mutually exclusive with the populated demo.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Contract | Existing connection HTTP/service/SQL and acquisition contracts reused. |
| Fixture | Dedicated scoped book, exact USD cents, explicit owner review mandate, actual API/sync database roles. |
| Compile | `go test -tags integration ./internal/server -run '^$'` PASS. |
| Defaults | `make check test build` PASS; local synthetic HTTP listener required sandbox escalation. |
| Combined browser | Coordinator runs real browser/Next bridge/Go/PostgreSQL journey after integration; not claimed here. |

## Current evidence

The fixture injects memory-only Link/exchange/manifest/secret dependencies and the
real acquisition HTTP adapter with a synthetic RoundTripper. It preserves original
0.10 and 0.20 lexemes for two purchases dated 2026-10-13. A single scoped,
origin/auth/body-bounded `POST /__proof/acquire` with `{}` runs the fixed acquisition
claim and returns status, changed_count, fetch_count and exchange_count. It accepts
no caller-selected entity, source, credentials or provider payload. Repeated trigger
calls reuse the claim and must not refetch. The separate explicit fixture-owner
source review grant precedes acquisition; product binding does not self-grant review.

Fixed entity/book IDs use prefix `10000000-0000-4000-8000-` and suffixes
000000000002/000000000003; bank 000000000011, expense 000000000012. Existing
synthetic browser principal and USD currency remain unchanged. UI adapter receives
`synthetic-link` and returns `synthetic-public`. This proves owned boundaries only,
not provider identity, live Link or Infisical qualification.

## Remaining work

Coordinator must exercise browser enrollment, explicit binding, acquisition, reviewed
category draft and explicit source-linked posting, with lost committed reply/reload
recovery. Database assertions require two source observations, one journal/link/outbox,
exact 0.10 posting and no automatic posting before approval. Fixture alone does not
claim that journey passed. Live connection deployment and provider qualification stay
separate.

## PR handoff

Only new test fixture, minimal integration-server injection and this runbook belong
to this lane. Root owns script, browser harness and acceptance assertions. Existing
non-connection app and demo paths remain selected by their existing flags.
