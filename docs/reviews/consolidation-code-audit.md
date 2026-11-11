# Consolidation code audit — 486

## Scope and method

Audit against main `a9561e2`, 2026-11-11. Initial scope was this document only. Root independently verified the caller
graph and approved the exact C1/C2 paths before deletion.
No private records, credentials, provider calls or cloud state were inspected.

Reviewed AGENTS.md, the consolidation plan, research bibliography and decision
standard, artifact-backed accounting and command recovery contracts. This is
reachability maintenance, not a financial contract, accounting-policy or dependency
change. Existing standards applicability is unchanged; no external reference or
conformance claim is added.

Evidence uses repository-wide symbol/import searches plus the actual Next,
Vite/demo, package and synthetic-runner entry points. Absence from an import
search alone is insufficient where Go interfaces or persisted protocol versions
are involved. The bounded removal candidates below were also inspected for
their purpose, request construction and storage effects. Existing financial evidence and historical
Git records remain intact.

## Approved removals

### C1: Retired manual demo viewers and their exclusive dependency cluster

Exact candidate paths:

- `web/demo/FlowViewer.tsx` (387 lines).
- `web/demo/VolumeViewer.tsx` (406 lines).
- `web/demo/volume.ts` (18 lines).
- `web/demo/VolumeViewer.test.tsx` (65 lines).
- `web/demo/volume.test.ts` (21 lines).

The five files total897 lines removed. This is an isolated dead demo
cluster, not the supported source-backed demo or production financial code.

Caller/reachability evidence:

- `rg -n 'FlowViewer|VolumeViewer' web` finds FlowViewer's declaration only;
  VolumeViewer is imported only by its own component unit test.
- `rg -n 'depositCents|expectedCents|parseCents|from.*volume' web` finds the
  volume helper declarations, VolumeViewer and their own unit test only.
- `web/demo/main.tsx` imports `LoadDemo`, production `Bookkeeping` and the real
  API transport. It mounts LoadDemo and Bookkeeping; neither manual viewer is
  imported or conditionally reachable. `web/demo/index.html` selects main.tsx.
- `web/demo/vite.config.ts` supplies that separate loopback bundle. The package's
  actual Next entry is `web/src/app/page.tsx` → Application; production image
  copies remain separate from the demo. No dynamic viewer loader was found.
- `scripts/test-app-integration.sh` explicitly requires zero Volume Lab entries,
  3,000 source observations and one original linked posting. Its comment labels
  the old viewers internal primitive proofs, not current demo actions.
- The viewers still construct new `operation: 'post'` manual requests, which the
  accepted artifact-only public application rejects. They cannot serve as a
  valid alternative source-backed workflow.

Why removal is safe: they are unmounted, hold only transient in-memory synthetic
state, and own no persisted protocol, token format, source identifier or migration.
Removing them does not remove recovery for real old commands. The two associated
test files test the removed runner/formula only; they are not current source,
auth, exact-money or recovery acceptance tests. Keep every shared amount helper,
LoadDemo, source-backed browser test, low-level ledger invariant and the new real
1,000-source-posting proof. Their expectations must not be reduced to compensate.

Validation required: locked frontend type/unit/build checks, current demo
bundle build and existing demo/real-application journeys; confirm repository
search has no remaining code imports. Historical documentation references can
remain explicitly historical; do not rewrite old acceptance into current behavior.

### C2: Uninstantiated provider-flow credential fake

Exact candidate: only the `secretStore` type and its Put/Read/Resolve methods in
`internal/providerflow/flow_integration_test.go` (the block immediately before
`type transport`; approximately36 lines).

Caller/reachability evidence:

- `rg -n '\bsecretStore\b' internal/providerflow` returns its type/method
  declarations only. No variable, literal, constructor or receiver call uses it
  in the flow, restore or volume integration tests.
- The actual flow constructs `bankcredential.NewAPIStore`, wraps it in
  `lostCredentialAck`, and constructs `bankcredential.NewResolver` for acquisition.
  Those real encrypted-repository and restricted-SQL dependencies remain used.
- Other packages' `s.secretStore(scope)` methods are unrelated production
  connection-service methods; this candidate does not include those symbols.

Why removal is safe: this is an unused test double with no production reachability,
registration, reflection or durable state. Removing it does not replace encrypted
storage with a mock or weaken unknown-commit evidence. Keep flowWrapper,
lostCredentialAck/lostScopedAck and all existing proof assertions unchanged.

Validation required: integration-tag compile/vet and the unchanged actual
provider-flow runner, including source restore and volume. A compile-only skip
is not restricted-SQL execution acceptance.

## Reviewed and deliberately retained

| Area | Caller/compatibility evidence | Decision |
| --- | --- | --- |
| `NewPlaidHTTP` versus `NewPlaidHTTPForProfile` | Omitted-profile constructor is called by cmd/sandbox-webhook, sandbox qualification, provider-flow and dispatch tests. Explicit-profile constructor is called by managed worker composition and actual connection browser fixtures. Profile tests distinguish Sandbox-only legacy omission from explicit Production routing. | Retain both. Similar implementation is not dead code, and collapsing environment defaults could change authority. |
| Private preview versions1/2/3 | `web/src/collection/types.ts` dispatches historical saved preview profiles in validCommand; current client accepts v3 envelope containing the historical union. Legacy rule/version and replay tests exercise these branches. | Retain historical validators, immutable body comparison and display/recovery paths. |
| Saved manual command handling | Artifact policy blocks new/manual execution but explicitly preserves old committed receipts, cancel and acknowledgement under current authority. CommandPanel and ledger service guards remain in the application recovery path. | Retain. Removing a form does not make persisted commands disappear. |
| Shared amount/currency helpers | formatAmount and currencyAt are used in actual bookkeeping and source/report views, not just retired demos. Dated-unit and unexpected precision behavior is deliberate. | Retain exact string/currency behavior; do not replace with float or locale assumptions. |
| Source demo and independent load evidence | LoadDemo is mounted by demo/main; actual demo proves3,000 imports plus one posting. `TestSourceBackedPostingVolume1000` separately proves1,000 explicit source postings with fixed money/count oracles. | Retain both distinct workloads and original acceptance assertions. |
| Connection credential-store interface methods | Production uses optional scoped interfaces and explicit failure methods; a plain text symbol count cannot establish interface unreachability. | No runtime deletion recommended from this audit. |

## Decision and limits

Root approved and the lane removed C1/C2; local verification is recorded below.
Leaving their unused implementation indefinitely preserves misleading manual
examples and duplicate test-only credential logic. Bulk removal of every legacy,
v1/v2 or similarly named helper is rejected: persisted receipts and environment
fences have real compatibility obligations.

This is a bounded source audit, not an exhaustive whole-program dead-code proof
or security certification. No additional safe runtime/client deletion was proven.
The expected frontend unit-suite reduction is exactly four tests across two
files: two VolumeViewer lifecycle tests and two retired cents-generator tests.
These test only deleted runners/helpers; active recovery, source, money, browser
and independent source-backed load tests remain unchanged. The sole runner edit
is its obsolete comment, preserving the zero-Volume-books assertion and all
existing source/journal counts. Verification receipts follow below.


## Verification and further CSS candidate

- Integration-tag providerflow compile (`-run '^$'`) PASS0.832s and Go vet PASS
  with the existing external local compatibility module file. No SQL test was
  executed; no canonical Go1.26.11 or vulnerability-scanner result is inferred.
- Standard locked `npm ci` completed; package.json/package-lock.json unchanged.
  The initial missing-vitest preflight was resolved by installation, not a test
  exclusion or dependency change.
- `make web-check web-test web-build` PASS: API generation check, TypeScript,
  all669 remaining units across37 files and production Next build. The removal
  accounts for exactly four retired tests/two files, not a skipped active suite.
- Separate current demo Vite bundle build PASS; no retired viewer import remains.
- `make web-browser` PASS: all98 unchanged component journeys in6.2s. This local
  run could bind its listener; no alternate host, timeout or guard was used.
- Actual restricted-SQL provider-flow execution and full composed application
  acceptance remain the unchanged canonical CI gates. Compile/vet and component
  browsers do not substitute for those results.
- Scoped/source diff whitespace check PASS. The runner change is comment-only.
- Root approved and the lane removed only the two `.demo-nav button` selectors in
  `web/src/app/style.css`. `rg -n 'demo-nav' web` returns only those CSS rules;
  no current markup produces the class. `.demo-guide` is still mounted and must
  remain; its rules were preserved. No other CSS changed.
