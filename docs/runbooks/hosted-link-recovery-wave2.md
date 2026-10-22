<!-- stage-runbook: true -->
# Hosted Link fresh-enrollment recovery — wave 2

## Status and boundary

#187 / parent #104/#88. Lease `PLAID-88-WAVE2-20261022T1930Z`, expiry23:59Z,
base `24f263ef2cde697cf0f8cf51c1176d5c1521e679`. Implementation only: no provider,
cloud, secrets, actual financial records, consent renewal or live activation.
Existing migration19 retention is reused; no SQL edit or new route/profile.

## Stage ledger

- Read fresh #187/#104/#88 issue snapshots, index, bibliography, contracts,
  migration19/current claim code and LR05 official sections.
- Found the actual remaining post-claim duplicate association lookup and the
  UI409 expired-unclaimed loop; coordinator approved bounded correction before
  runtime edits. Private typed server evidence retains every new-mode verifier
  binding check. Normal caller public tokens still require verification.
- Added aggregate/recovery read deadlines and explicit UI distinction between
  Hosted URL expiry and the server recovery window; no automatic restart.
- Corrected the earlier source description of migration19's read versus write
  guards. Physical encrypted-history deletion and ordinary repair remain open.
- Installed the fresh locked Next16.4.2 dependency graph with npm ci; no old
  node_modules symlink or dependency change.
- Final full-unit rerun exposed a pre-existing reviewed-date race: the submitted
  date changed from2026-01-02 to today. Coordinator approved moving only the
  existing synchronous initializer to a layout effect before form interactivity;
  no date/proposal policy or scope reset changed. LR06 records the mechanism.

## Current evidence

PASS: focused Go race tests, `make check test build`, final generated API/type
checks,326 unit tests across19 files, production web build,24 Chromium tests,
and both `make app-integration connection-browser-test` journeys. Final web and
full-stack runs include the approved initialization correction. Browser/API/SQL
proofs use real disposable PostgreSQL and synthetic provider/identity adapters;
no real institution, OAuth, cloud identity or hosted delivery is established.

The new `TestHostedRecoverySingleAssociation` was also run against untouched
baseline24f263e in an isolated detached test checkout: expected FAIL,
`credential_recovery_required`, caused by its second post-claim provider lookup.
The same actual-HTTP-adapter fixture passes with this correction and proves one
lookup, one exchange, one credential publication and no repeated provider work.
Pending/ambiguous/outage/lost-claim and caller-token mismatch remain fail-closed.

The first final full-unit run caught the reviewed-date defect; it was not counted
as passing or hidden by retry. The approved initializer change then passed the
focused date assertion, full326-test suite and both final browser/SQL journeys.

Local logs: `/private/tmp/jayburd-187-core.log`, `jayburd-187-focused-go.log`,
`jayburd-187-baseline-regression.log`, `jayburd-187-date-regression.log`,
`jayburd-187-final-web.log`, `jayburd-187-final-browser.log` and
`jayburd-187-final-integration.log`. No provider payloads or credentials captured.

## Remaining work

Hosted L02–L04 and #187 acceptance need a separately authorized fresh deployment
and real Sandbox completion/reload/expiry evidence. Ordinary reconnect, signed
LINK completion receipts, OAuth return, Item lifecycle/consent and physical
retention/deletion remain distinct requirements; this patch does not close them.

## PR handoff

Refs #187/#104/#88. No migration, OpenAPI/generated type or release-profile change.
No deployed outcome inferred from synthetic tests. Canonical
[contract](../contracts/hosted-link-enrollment.md) and
[source review](../research/plaid-link-recovery-sources.md) record exact limits.
