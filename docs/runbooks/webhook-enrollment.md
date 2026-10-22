<!-- stage-runbook: true -->
# Sandbox webhook enrollment — #271

## Status and boundary

Implementation-only lease `PLAID-271-PREP-20261022T1725Z`, expiry22:00Z.
Dedicated branch `issue/271-webhook-enrollment`, baseline
`a2c9dee793f81ed483055b0ee916876b59265d27`. No provider/cloud calls, secret access,
deployment, consent renewal or existing Item mutation. Prior consent expired;
implementation authorization does not reactivate it.

## Stage ledger

- Reviewed live #271 export, current index, bibliography, merged runtime and
  connection contracts; refreshed WE01–WE03 official sections.
- Coordinator approved exact optional `HTTPConfig.WebhookURL` and dedicated
  canonical endpoint before implementation. Empty preserves the old wire.
- Added strict pre-construction configuration validation and generated Link
  request propagation for new enrollment only. No allowlist or mode widening.
- Extended the exclusive lane at coordinator request to runtime composition and
  optional API flag. Nonempty configuration pins both fixed development project
  ID/number before dependency access; disabled configuration remains wholly inert.
- Documented existing Item update authority/vault/uncertainty proposal; no
  executable route, token flag, SQL or provider mutation was added.

## Current evidence

PASS: targeted `go test -race -count=1 ./internal/connections
./internal/connectionruntime ./cmd/api` using local Go1.26.5 and the existing
external modfile/cache; tracked toolchain requirement unchanged. Wire tests
independently assert the literal destination and omission for legacy config;
runtime tests reject substituted/disabled/cross-project config before I/O and
prove startup performs only the two existing synthetic static-version reads.
All tests intercept synthetic responses; none contacts Plaid or obtains secrets.

PASS: `make check test build` with approved local listener permission; no
browser/SQL contract changed. Evidence logs:
`/private/tmp/jayburd-271-enrollment-tests.log` and
`/private/tmp/jayburd-271-enrollment-core.log`.

## Remaining work

Coordinator owns server/config/release composition and independent integration
review. Dedicated host public exposure/IAM, fresh consent, actual signed chain,
retry/recovery and exact/no-journal evidence remain #271/#87/#88 gates. Existing
Item update proposal needs a separate reviewed command and authority boundary.

## PR handoff

Refs #271/#103/#87/#88. No migration, browser/OpenAPI schema or accounting change.
Contract: [webhook enrollment](../contracts/webhook-enrollment.md).
Sources: [WE01–WE03](../research/webhook-enrollment-sources.md).
