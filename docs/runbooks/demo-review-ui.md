<!-- stage-runbook: true -->
# Demo acquisition and review UI

## Status and boundary

Issue #98 implements the synthetic acquisition control and scoped review inbox.
Coordinator owns HTTP/OpenAPI, recovery integration, workspace navigation and
application composition. No CSV upload is required. Loading creates imported,
unposted review items; balances change only after an explicit reviewed source-post
command. This component delivery adds no SQL migration or dependency.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Contract/source review | Coordinator and SQL author approved source/list/detail/categories/draft projections and category-only identity-currency draft input before binding. Backend author approved bounded acquisition step/status protocol. |
| Acquisition | Mount discovers only; explicit Load/Resume reconciles GET then performs at most 64 bounded stages. Pause, blocked foreign command, lock or unmount stops subsequent writes. Uncertain reply triggers read reconciliation, not automatic additional mutations. |
| Review | Source-scoped opaque pagination; no browser float or persistence. SQL categories, fixed source booking date, editable journal memo, unchanged uncertain draft request/key, explicit source_post recovery handoff. |
| Peer corrections | Posted historical drafts have no posting action. Stale drafts require fresh category confirmation. Actual posted journal read validates entity/book/journal identity and displays immutable lines. |
| Verification | make check test build, web-check web-test web-build PASS with coordinator-owned schema/recovery dependencies temporarily copied for composition. 170 unit tests (17 new: 11 inbox/6 loader),12 browser tests (1 new) PASS. |

## Current evidence

Reviewed repository source records: FW01 Focus Visible (Success Criterion, Intent,
Benefits), FW02 Tables with One Header (Overview and header-row example), DI01
PostgreSQL constraints/serializable sections, and existing command-recovery,
standards and durable-import contracts. Reuses the recorded selected reviews;
no new normative ISO review or conformance claim. Internal UI data retains the
approved synthetic exception and ISO4217 currency-version semantics.

Interpretation: native labelled controls, semantic tables, exact decimal-string
display and separate imported/posted statuses. An automatically posting demo was
rejected because it removes explicit accounting review. Browser-only fabricated
books would not exercise durable imports; injected browser clients are fixtures.

Inbox tests cover exact 9007199254740993.01 display, draft request and handoff fields,
unchanged uncertain retry, cursor preservation/stale-page rejection, denied data,
hidden/locked requests, late old-scope replies, FX blocker, discovery retry,
posted draft suppression, stale draft refresh and actual scoped journal retrieval.
Loader tests cover mount read-only behavior, explicit server steps, pause/resume,
uncertain reconciliation, foreign pending commands, unmount and 64-step bound.
The new browser test uses keyboard activation, checks readonly booking date and
exact draft body/idempotency header, then proves handoff adds no execute request.
Desktop/mobile screenshots inspected; controls readable and table overflow stays
inside its scroll container. Mobile document width stays within 390px.

Logs: `/private/tmp/jayburd-demo-ui-gates.log`,
`/private/tmp/jayburd-demo-ui-final-web.log`, and
`/private/tmp/jayburd-demo-ui-browser.log`. Browser rerun initially encountered
sandbox localhost listen denial, then passed with approved local-server access.
Screenshots remain ignored under `web/test-results/review-inbox-source-review-64310-act-draft-on-desktop-mobile-chromium/`.

## Remaining work

Coordinator owns actual SQL/HTTP/demo integration and workspace composition.
Component browser mocks do not establish accounting correctness or real 3,000-row
loading; backend and full application evidence must prove those independently.
Provider acquisition, live policy, FX draft valuation and retention cleanup remain
outside this slice. No financial requests or tokens use browser persistence.

## PR handoff

Issue #98. Integrate after coordinator-generated API projections and source_post
recovery support. Borrowed generated/API/CommandPanel files are excluded from this
commit. Root owns combined real application gates and PR creation.
