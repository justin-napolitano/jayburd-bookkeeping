# Managed identity rehearsal

<!-- stage-runbook: true -->

## Status and boundary

Claimed #39 on `issue/39-identity-rehearsal` from merged #40. Owner approved building
a local managed sign-in rehearsal. No real user credentials, inbox, provider
payloads or authenticator setup material will be inspected by agents.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Contract | Separate entry, fixed dev project, loopback server and private owner flow | Review |
| Synthetic validation | 105 unit/component tests, two rehearsal browser tests, six existing browser journeys, static-server denials | Owner rehearsal |

## Current evidence

PASS: `make web-check web-test web-build`, `make identity-rehearsal-test web-browser`,
`make check test build`, pinned actionlint and document-link checks. The rehearsal tests intercept every external request;
no actual mail/provider acceptance is claimed. Server tests verify strict routing,
Host/origin/method denial and CSP. Initial validation caught test imports/React
prop typings, corrected before final checks. Registry sandbox stall resolved
through approved install; no credentials were inspected. Vite reports an expected
Next.js `use client` directive warning in the entirely client-side static bundle.
Identity Platform configuration is applied. User reports account creation only.
Owner review server started at `http://127.0.0.1:39091`; root returned 200 and
`/v1/entities` returned 404. This is transport evidence, not a user login.

## Remaining work

Owner supplies public web API key in the local screen,
then uses their own password/email/authenticator. Record only PASS/PARTIAL/BLOCKED
for mail delivery, verified email, TOTP enrollment, fresh challenge and logout.
Cloud database/Go revocation and hosted access remain separate gates.

## PR handoff

Contract: [identity rehearsal](../contracts/identity-rehearsal.md). No SQL,
financial API or cloud changes. Keep #39 open until owner evidence exists.

## Owner instructions

From this branch/worktree, run `npm --prefix web ci` once, then
`make identity-rehearsal`. Open **http://127.0.0.1:39091** (not localhost).

1. Open [Identity Platform providers](https://console.cloud.google.com/customer-identity/providers?project=jayburd-bookkeeping-dev)
   → **Application setup details**. Copy only `apiKey` into the local form.
2. Sign in with your invited account. Click **Send verification email**, open
   Google's link privately, then return and click **I have verified my email**.
3. Sign in again, select **Set up authenticator**, enter the private setup key
   in your authenticator, and verify its code. Enrollment signs you out.
4. Sign in once more with password and authenticator code. The completion screen
   confirms browser sign-in only. Sign out when finished; Ctrl-C stops the server.

If configuration/login fails, check the selected project and API-key restrictions
privately; report only the visible generic message and step. Do not send credentials,
links or setup-key screenshots. No automatic email is sent by starting this tool.
Missing mail: request once, check spam, wait at least 60 seconds before retrying;
provider quotas may require longer. Expired links need a new request.
