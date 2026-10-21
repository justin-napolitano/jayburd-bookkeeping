<!-- stage-runbook: true -->
# Plaid liability binding delivery

## Status and boundary

Issue #210, branch `fix/plaid-liability-bindings`. Synthetic development
enrollment of reviewed credit-card and student/mortgage loan accounts. No live
institution, opening balance or automatic posting.

## Stage ledger

1. **Issue and sources — complete.** #210 is open and claimed; #209 is closed.
   Reviewed Plaid Accounts, Transactions and Liabilities product boundaries.
2. **Decision and contract — complete.** The liability profile and fail-closed
   source-review boundary are recorded in the contract and source register.
3. **SQL, Go and browser implementation — complete.** Migration 26 extends the
   serializable binding command, preserves a trusted account-type decoder pin
   and blocks liability-source posting drafts. The browser proposes one account
   for each supported verified type/subtype.
4. **Synthetic validation — complete with toolchain limit.** Migration fresh
   install/down/up, connection-service SQL integration, `make ledger-test`,
   `make check`, `make test`,
   `make build`, `make app-integration`, `make connection-browser-test` and all
   web gates pass. Go ran with locally installed 1.26.5 and a temporary modfile
   because the repository-pinned 1.26.9 download returned HTTP 403. CI must
   repeat with the exact pinned toolchain.
5. **Hosted qualification — pending.** Migration before runtime, then fresh
   fictional Sandbox account enrollment and sync evidence under #88.

## Current evidence

The focused decoder and connection-service suites, disposable SQL integration
suite, migration fresh install/down/up, 286 browser unit tests, 17 component
browser tests, one end-to-end connection journey and repository check/test/build
gates pass. Hosted qualification and exact-toolchain CI remain pending.

## Remaining work

Plaid product availability differs by institution. Liability observations are
not authorized for posting until a separate accounting-policy review.

## PR handoff

Link #210, report validation and migration impact, and require migration 26
before matching API/worker deployment. Hosted Sandbox acceptance remains #88.
