# Identity and setup delivery

<!-- stage-runbook: true -->

## Status and boundary

Branch `issue/6-managed-identity` adds the owner-operated development Identity
Platform root. Applied to `jayburd-bookkeeping-dev` on 2026-10-12; subsequent
Terraform plan returned zero (no configuration drift).
Existing managed adapter, browser flow and restricted SQL enrollment are merged;
live login/enrollment/revocation remain unproven. No real bookkeeping approval.

## Stage ledger

| Stage | Evidence/result | Next action |
| --- | --- | --- |
| Provider configuration | I03/I06 reviewed; 19 mocked infrastructure tests pass across five roots | PR review |
| Authorized dev apply | API enablement and identity config apply exit 0; subsequent plan exit 0 | Owner account/enrollment proof |
| Source/contract review | I01–I05; identity/setup contract | Review PR |
| Identity/API | `TestIdentityBoundary`, `TestBrowserBoundary` pass | Live provider/session proof |
| SQL/setup | `TestControlledSetup`, `TestSetupHTTP` pass under actual runtime/operator roles | Review grants |
| Failure/concurrency | Invalid rollback, concurrent replay, lost acknowledgement and revocation race pass | Preserve regression tests |
| Portable harness | Explicit inputs copied to disposable volumes; both suites pass | CI verification |
| Dependencies | Go 1.26.9, Firebase 4.22.0 and patched transitive pins; scan passes | CI vulnerability gate |

## Current evidence

PASS on 2026-10-12: `make infra-test` (19 mocked runs), `make check test build`
(including Go race tests), identity-only apply exit 0, post-apply
`plan -detailed-exitcode` exit 0. Read-only service listing confirms
`identitytoolkit.googleapis.com` is enabled. No users, credentials, provider
payloads, state or plans were inspected. All cloud command output went to ignored
private logs. State uses the existing foundation bucket with prefix `identity`;
application deployment identities receive no new grants.

Configured invited email/password, disabled anonymous/phone/public signup,
operator-managed deletion and TOTP enrollment with one adjacent interval.
Configuration convergence does not prove actual user sign-in or MFA claims.
Backend init initially failed in the sandbox and passed with network permission.
Default Go auto-download failed; checks passed with the existing pinned compiler.
Provider validation rejected documented `deletion_policy`; lifecycle
`prevent_destroy` remains. Historical evidence below is retained separately.

PASS on 2026-10-11:

```sh
make check test build
make migration-test
make ledger-test
bash scripts/migrate.sh check origin/main
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/seed.yml
npx --yes @redocly/cli@2.62.1 lint api/openapi.json
```

OpenAPI has only the existing missing-license warning. Conditional spot-line
branches now explicitly permit their surrounding fields; the parent still rejects
unknown fields. Vulnerability scan: zero reachable/imported-package findings;
one module-only advisory, GO-2026-5932, concerns unimported `x/crypto/openpgp`.
SDK tests use synthetic SDK outcomes; they do not prove live Google verification,
key rotation, MFA enrollment or provider revocation. PostgreSQL tests execute real
roles, transactions and first posting from returned setup IDs. Existing accounting,
isolation, close-race and synthetic restore proof remain passing. Integration binary
is static/non-race; ordinary Go tests use `-race`.

Local compiler: `/private/tmp/jayburd-toolchain-1.26.9/go/bin/go`, official archive
SHA256 `f9bb7c0a02506c5d9bf0d1eb1f7ee6c7684f844ae49558308a0427b830e022cc`.
Its bin directory was prepended to PATH; GOCACHE=/private/tmp/jayburd-go-cache and
GOMODCACHE=/private/tmp/jayburd-go-modcache. Proxy/toolchain and Docker bind-mount
failures were resolved with the verified official archive and disposable input
volumes. Docker credential-helper access needed execution outside the sandbox.
No credential values were inspected. Initial vulnerabilities were patched before
final checks; the workflow now runs the same vulnerability scan.

## Remaining work

PR/CI review, owner account creation/verified email, browser
configuration, MFA enrollment/recovery, SQL principal enrollment and GCP/Infisical
deployment remain release gates. Provider revocation is checked
before an operation, not atomically with SQL commit; local disabling is locked.
Setup adds one development book per new entity; existing-entity administration,
additional books/accounts/periods and real-LLC policy are later work.

Owner next action: open [development Identity Platform users](https://console.cloud.google.com/customer-identity/users?project=jayburd-bookkeeping-dev)
and create the intended invited account privately. Do not share its password,
verification links or authenticator codes with agents. Email verification must be
established before our browser permits TOTP enrollment; verification UI implementation
is tracked in #38 and the [verification runbook](email-verification.md). Review
and complete the live operator flow before
claiming the account is ready. Provider creation does not enroll a SQL principal.

Then supply public browser configuration through the reviewed release inputs,
complete TOTP enrollment in the deployed browser, sign out and sign in again,
and enroll the verified issuer/subject using the restricted SQL operator path.
These steps depend on the remaining database and application deployment gates;
the hosted bookkeeping application is not available from this identity apply.

## PR handoff

Review [identity/setup contract](../contracts/identity-setup.md), provider settings
and state isolation. Keep #6 open for its remaining gates. This delivery changes
no SQL or application routes. See the
[identity root](../../infra/gcp/identity/README.md) for state isolation and setup.
Run `bash scripts/check-stage-runbook.sh --staged` before commit.
