# Identity and setup delivery

<!-- stage-runbook: true -->

## Status and boundary

Implemented offline contribution to #6: managed identity adapter, restricted
principal enrollment and atomic synthetic entity/book/account setup. Broader
#2/#3/#5 deployment gates remain open. No real identities, secrets or cloud changes.

## Stage ledger

| Stage | Evidence/result | Next action |
| --- | --- | --- |
| Source/contract review | I01–I05; identity/setup contract | Review PR |
| Identity/API | `TestIdentityBoundary`, `TestBrowserBoundary` pass | Live provider/session proof |
| SQL/setup | `TestControlledSetup`, `TestSetupHTTP` pass under actual runtime/operator roles | Review grants |
| Failure/concurrency | Invalid rollback, concurrent replay, lost acknowledgement and revocation race pass | Preserve regression tests |
| Portable harness | Explicit inputs copied to disposable volumes; both suites pass | CI verification |
| Dependencies | Go 1.26.9, Firebase 4.22.0 and patched transitive pins; scan passes | CI vulnerability gate |

## Current evidence

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

PR/CI review. Live project, workload identity/IAM, trusted non-emulator SDK client
composition, MFA enrollment/recovery, UI sessions, quotas, public server/container
and GCP/Infisical deployment remain release gates. Provider revocation is checked
before an operation, not atomically with SQL commit; local disabling is locked.
Setup adds one development book per new entity; existing-entity administration,
additional books/accounts/periods and real-LLC policy are later work.

## PR handoff

Review [identity/setup contract](../contracts/identity-setup.md), enrollment grants,
atomic setup/replay and migration 000003. Keep #6 open for its remaining gates.
Merged 000001/000002 SQL is unchanged; down migration deliberately fails to protect
history. Next delivery: trusted server composition and login/MFA flow, then the
first bookkeeping UI. Run `bash scripts/check-stage-runbook.sh --staged` before commit.
