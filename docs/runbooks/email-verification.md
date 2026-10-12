# Invited-account email verification

<!-- stage-runbook: true -->

## Status and boundary

Owner requested contract/issue definition before implementation, 2026-10-12.
Claimed #38 on branch `issue/6-email-verification`; parent #6. Provider configuration is
applied and the owner reports creating a user. No email delivery, verified email,
TOTP or SQL enrollment evidence exists yet. No real credentials or mail accessed.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Research and contract | BA03; browser verification state and acceptance defined before code | Review #38 |
| Implementation | Explicit send/reload, phase isolation, cancellation and resend guard | Review PR |
| Dependency review | #39 records pre-deployment identity rehearsal; #6/#27 updated | Preserve release gate |

## Current evidence

PASS: `make web-check web-test web-build` (103 tests); `make web-browser`
(six existing synthetic bookkeeping browser journeys); `make check test build`
(Go vet/race/build, migration integrity and documentation checks).
Verification UI itself is exercised by SDK/controller and rendered-component
tests; no real provider proof is implied. Initial UI test found an ambiguous
status-role query in its fixture; corrected. Browser port sandbox denial resolved
with approved execution. Review also fixed cancellation during failed-refresh
cleanup; a named regression test proves late errors stay cleared.

## Remaining work

Merge/review #38 implementation. Then prepare an owner-operated
loopback managed-identity rehearsal with the production auth component and public
Firebase settings, no database/finance routes, no emulator. Owner enters credentials,
requests mail and enrolls TOTP privately. Do not run it from default CI or claim
actual delivery from mocked SDK success. Rehearsal tooling is a separate delivery, #39;
provider domain/key restrictions must be reviewed before use.

After provider rehearsal, provision #26 and complete restricted principal
onboarding/revocation checks, record the no-application-secret assessment in #11,
then prepare #27 release. Keep all current workflow closure gates enforced;
full hosted identity/finance acceptance remains after deployment.

## PR handoff

No SQL migration, financial API or cloud configuration change. Sources and
contract updates preceded code; validation is recorded above. This does
not close #6 or assert email delivery to the owner's new account.
