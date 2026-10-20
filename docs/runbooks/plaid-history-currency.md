<!-- stage-runbook: true -->
# Plaid history currency delivery

## Status and boundary

Issue #225; branch fix/plaid-currency-history. Follow-up #226 owns history
settings/capacity and out-of-period review messages. Development Sandbox scope.

## Stage ledger

1. **Diagnosis/source review — complete.** Owner supplied three fixed currency
   rejections; no private page inspected. Singleton USD metadata rejects synthetic
   pre-2026-09-17 dates. Official Plaid history/pagination sections and ISO 4217 MA
   amendment 115 page 1 reviewed; see the [contract](../contracts/plaid-history-currency.md).
2. **Contract/issues — complete.** #225 claims dated-currency acquisition and
   additive reference history. #226 preserves separate product/capacity acceptance.
3. **Implementation — complete.** Migration 25 and compatible worker preserve
   immutable reference provenance and existing source/posting authorization.
4. **Validation — local passed, pinned-toolchain CI pending.** Named synthetic
   regressions, repository/SQL gates and app integration; see evidence below.
5. **Promotion/hosted acceptance — pending.** Preserve enrollment; paused dispatch,
   matching artifacts, explicit app retry and observed publication/review required.

No live-account approval, private records, provider payloads or credential retrieval.
Do not widen bookkeeping policy dates to make acquisition pass.

## Current evidence

`make check test build`, `make migration-test ledger-test sync-state-test`,
`make acquisition-test`, `scripts/test-sandbox-bootstrap.sh` and
`make app-integration` pass. The final migration/acquisition rerun includes the
bounded 33-row discovery probe, populated-version-24 upgrade, protected rollback,
owner-only history installer, exact replay and ID-conflict preservation.

TestHistoricalCurrencyAcquisition proves two dated versions, exact sum 3.03,
one committed cursor, completed replay without provider calls, old-date policy
denial, usable current-date draft and zero journals. Its old enrollment fixture
models migration-22 new-source overlap metadata before observations exist;
fixture owner authority is not a product recovery path. Initial review-fixture
failures (assignment join and legacy overlap state) were corrected and rerun.
TestCurrencyVersionContext covers one mixed-version page, exact interval boundary,
legacy metadata, invalid contexts and missing-coverage/precision/identity denials.
TestSyncHistoricalCurrencyValidation independently checks SQL date/identity/precision.
Existing JPY/KWD and FX/ledger tests remain passing.

The browser proof crosses production Next API bridge, Go HTTP and disposable
PostgreSQL, with one explicit posting and lost-response recovery. Identity remains
test-only. No cloud or Plaid calls occurred. Local Go validation uses a cached
network-isolated Go 1.27.2 container/public modules because the pinned 1.26.9
download failed. Repository toolchain pin is unchanged; CI must verify 1.26.9.

## Remaining work

Pinned-toolchain CI, review and matching migration/worker promotion remain before
owner-session hosted retry. Current Sandbox scheduler is still paused; this patch
does not run workers or reset quarantined attempts.

## PR handoff

Link #225 and #226; migration 25 changes scoped worker metadata and appends only
reviewed reference history. No HTTP/browser shape change. Keep qualification open
until actual publication and review; synthetic success is not hosted acceptance.
