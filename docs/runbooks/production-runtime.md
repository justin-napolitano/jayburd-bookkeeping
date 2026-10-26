<!-- stage-runbook: true -->
# Production credential/runtime delivery — #313

## Status and boundary

PRODUCTION313-RUNTIME; feat/313-production-credentials from main775a913.
Owner delegated implementation and reports Infisical secret entry complete.
No value read, provider/cloud/resource/deployment action or pilot activation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Fresh313/311/312/65 state; dependencies closed; dedicated worktree and exclusive lease |
| Decision | production-credentials contract reviewed before source edits; current SQL preflight/profile and legacy encrypted handoff inventory read |
| Implementation | Closed explicit Production metadata, current SQL key/profile preflight, bank/Link AEAD environment, dedicated IAM/evidence composition and finite worker/API authority implemented |
| Current scoped SQL | TestProductionCredentialRuntimeSQL passed on actual restricted API/worker logins in disposable PG17: wrong key creates no attempt; Hosted handoff roundtrip; bank credential read; one exchange across replay; exact worker reference; environment/Item/version/reference/key substitution denied before unwrap/static; scope revocation denies read; zero journals |
| Regression | make check/test/build, migration-test, ledger-test, connection-state-test, connection-service-test, acquisition-test and provider-flow-test PASS with local toolchain qualification below |
| Browser/API/SQL | make app-integration PASS: real browser -> production Next bridge -> Go -> disposable restricted PostgreSQL; expired optional enrollment preserves setup/posting/recovery without provider access |
| Frontend | Locked npm ci; web-check/web-test/web-build/web-browser PASS, including342 unit and26 browser cases; no frontend/API wire contract changed |
| Review cleanup | Legacy service rejects Production private projections; explicit invalid worker environment denies dependency transport; comments/CLI metadata descriptions clarified. Focused affected unit tests rerun |

## Current evidence

Source-only implementation with synthetic credentials. Owner-reported secret entry
is not independent delivery/IAM/version verification. Existing schema34 apply/status
is complete per issue26 comment6393474965; no migration belongs to this issue.

Pinned Go1.26.10 download failed connection reset, then403 on the approved retry.
Local validation uses installed Go1.26.5 and an external modfile/cache under
/private/tmp; tracked go.mod/go.sum and image/toolchain pins remain unchanged.
Canonical CI must validate the pinned toolchain. Synthetic loopback/socket and
Docker tests needed approved execution outside the filesystem sandbox. Locked npm
installation succeeded on the approved network retry; zero reported advisories,
four existing package lifecycle scripts remain unapproved and were not enabled.

Initial integration fixture had no HostedURL, then an invalid /link URL instead
of the strict /hl URL, and its final zero-journal assertion named ledger.journal
instead of ledger.journal_entry. Corrected the original fictional fixture and
reran the entire restricted suite; no product check or expected outcome weakened.
The new public docs use production-runtime filenames because the context loader
correctly rejects credential-shaped paths. Bibliography stays under32KiB.

## Remaining work

Exact-head canonical CI and human sensitive-source review remain before merge.
Keep314 signed receiver/reconnect,315 independent journey and337/316 resource,
static secret delivery/version mapping and release acceptance separate. Production
receiver URLs and Hosted repair remain rejected until their reviewed capability
exists. No Infisical value was opened or copied, and no live provider was called.

## PR handoff

Issue313; no SQL/migration/frontend/infra/release changes. Human security review
required for concrete auth/crypto/secret source changes before merge. No claim of
actual secret delivery, live-account readiness or Production activation.

The bundled Citizen Developer policy requires human review for sensitive generated
source; agent self-review and passing tests do not substitute that review. Existing
owner direction retains Infisical/GCP/platform KMS rather than the bundled Azure
defaults. No new secret provider or policy exception is introduced here.
