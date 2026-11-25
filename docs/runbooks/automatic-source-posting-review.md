<!-- stage-runbook: true -->
# Automatic accounting customer boundary —334

## Status and boundary

Source-only implementation from rewritten main88fce16 in dedicated
feat/334-automation-review-api. Root coordinates334/65; core lane owns migration49
and durable execution. No cloud, provider, managed SQL, real financial records,
credential, runtime activation or official-book policy approval occurs here.
The owner requested automatic source-backed routine posting without per-entry
confirmation; the reviewed one-time mapping policy is the consent boundary.

## Stage ledger

| Stage | Evidence and limits |
| --- | --- |
| Contract review | Read AGENTS, issue334/65, source posting/correction/recovery contracts and fixed posting corpus. Accepted bounded source-identity-usd-v1; FX, paired transfers and split loans remain explicit zero-effect holds. |
| HTTP/composition | Authenticated closed policy/status routes,128KiB body, safe error codes, no public execution route. Default-off API and runner flags independent; runner requires API. SQL49 readiness requires seven public restricted functions and denies twelve private helpers. Both loops cancel/drain before pool closure. |
| Customer flow | Compact mapping inventory, local bounded YAML import/export, exact source/account references, one-time policy review/save, scope-safe async file handling, stable rule IDs and immutable historical editions. Separate runtime/policy states, closed snapshots joined by policy_revision, actionable source exceptions and posted-source link. |
| Compatibility | API48/manual drafts keep old wire shape. Only automatic-owned draft has versioned profile/origin/operation ID; new UI inspects it without creating normal interactive posting commands. Existing ReviewInbox cases continue passing. |
| Local checks | Targeted frontend66 tests passed, then full774 frontend tests passed and production Next build passed. Added final strict enum/NUL regression afterwards; final rerun67 targeted tests, TypeScript and generated OpenAPI check all passed. Combined autopost HTTP tests pass with unchanged copied core package. |
| Environment boundaries | Local installed Go1.26.5 differs from tracked1.26.11. External modfile changes only directive; no tracked dependency change. Exact x/net0.62.0 absent and proxy.golang.org DNS blocked server/cmd compile. Browser component bundle builds, but both1280/390 cases fail before execution at Vite listen EPERM ::1:3100. Canonical Linux CI must prove those checks. |

## Current evidence

New tests cover all policy/runtime combinations, unavailable status without invented
zeroes, closed status/operation shapes, mismatched policy revisions, stable YAML
IDs, unsafe YAML/resource bounds/ambiguous references, uncertain write retry with
identical body and key, actual saved-policy discovery after remount, deferred file
read invalidation, inspect-only automatic drafts and authorized source drill.
Proxy tests preserve exact settings bytes, no execution endpoints, no automatic
write retry, strict framing and128KiB cap. Backend tests cover default-off routes,
missing dependencies, truthful runtime signal and draining both background loops.

## Composed accounting browser proof

Added `bash scripts/test-app-integration.sh --accounting`, integration-tagged
server fixture and `playwright-accounting.config.ts`. The fixture seeds retained
fictional statement observations, their review evidence links and a protected
manual draft created under the restricted API login in a serializable transaction.
It seeds no automatic policy or journal. This proves use of retained evidence;
it does not claim to exercise statement upload/import acquisition.

The real Bookkeeping component uploads YAML, reviews/enables one policy, recovers
a lost committed settings response with identical bytes/key, waits for the real
background runner, follows the unmapped exception action, then opens the posted
source and reads its immutable journal. API assertions verify original source
identity/revision/units, protected draft and foreign/origin denials. Remount reads
saved policy and the same journal. No API response or financial outcome is mocked.
Independent database assertions require exactly one12.50USD debit/credit entry,
one automatic source link, receipt/outbox/settings receipt, preserved3.75/4.25/12.50
source lexemes, protected manual draft, manual/unmapped holds and zero interactive
posting commands/reservations. The test does not assume the latest operation is
committed: later held items can be latest while posted source access remains.

Authored checks: shell syntax, TypeScript, Playwright discovery (one real journey)
and standalone vet of the fixed fixture format arguments passed. Actual composed
invocation stops at the existing port admission check with `Synthetic proof port
unavailable` before Go/SQL/browser setup. This is NOT an execution pass and the
port, auth, toolchain and network boundaries were preserved. `make check`,
`make test` and `make build` were attempted with `GOTOOLCHAIN=local`: the230
existing Node release tests pass, then Go gates stop because the tracked1.26.11
minimum exceeds installed1.26.5. No dependency/toolchain workaround is committed.

## Remaining work

Root must integrate exact core checkpoint, run canonical Go1.26.11 race/vet/build,
real restricted SQL49 synthetic proof and composed browser/API/SQL acceptance.
Local browser socket denial is not a browser pass. Changed posted evidence stays
held pending separately reviewed333 correction work. No assertion that all334
accounting treatments or deployed runtime are operational is made.

## PR handoff

Commit only this lane's HTTP/composition/OpenAPI/proxy/UI/tests/docs. Copied
internal/autopost/{service,result,runner}.go are temporary compilation inputs owned
by core and excluded. Financial/security human review remains required before
merge. Repository visibility, auth, integration and rollout belong to root.
