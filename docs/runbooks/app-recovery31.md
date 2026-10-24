# Sandbox app recovery after migration31
<!-- stage-runbook: true -->

## Status and boundary

Issue272/281; lease VERIFY272-APP31-20261023T1945Z. Owner requested recovery and
controlled browser testing after actual migration31/status/login bootstrap passed.
Dedicated branch from736b703. This is a finite forward Sandbox app release, not
the durable281 expiry-availability fix or permission to connect live accounts.
Exact manifest owner review remains required under deploy/README.md.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Resume |31 actual apply/status and complete login predicate PASS; current GitHub lease checked | Preserve original saved command |
| Browser | Original signed-in tab still shows same UNKNOWN setup registration; no reload/input | Reconcile unchanged command after recovery |
| Candidate | Trusted31-aware build40004622277, exact app manifest and local core/build/release gates PASS | Exact owner review |
| Recovery/testing | Not dispatched/tested | Transport acceptance then controlled synthetic UI flow |

## Current evidence

Actual release40014820903 passed transport with exact images/config digest,
API00020-zpd/web00020-zb4; PR288 all five CI checks SUCCESS. Owner signed in and
the subsequent [Tartan browser run](tartan-browser-qualification.md) created and
acknowledged a fresh book, recovered the original provider-success attempt, bound
nine supported accounts, and verified scheduled historical-ready publication for
all nine through scoped read-only metadata. Transaction review remains blocked
by a browser socket failure; independent R02/R03 proof remains open. This evidence
supersedes the preparation-only stage below; no future expiry281 fix is implied.

Trusted owner build40004622277 attempt1/sourcefb4ac04 has all seven jobs SUCCESS,
including canonical Go1.26.10, full browser/API/SQL and exact four-image runtime/
security gates. API/web/runtime source paths show no changes from previously
reviewed edb778e. SQL31 only changes verifier permissions. Existing public browser
identifier, Infisical metadata and static secret version1 pins retained; no secret
contents accessed. Compatibility evidence is explicit in the release manifest.
No webhook opt-in, worker/scheduler root or existing Item consent update.

Candidate dispatch expiry2026-10-24T02:00Z; new-enrollment Sandbox consent ends
2026-10-24T23:00Z. A finite interval is required to recover API startup currently
coupled to expired optional configuration. It does not fix future expiry handling;
281 remains open. Existing expired Items remain expired. This release executes
schema31 status (no up), updates existing API/web services, then transport smoke.
Existing development platform/cost boundaries remain; no new service topology.

Exact manifest SHA256:
bda64a77c06913077893a2c25e08157202d1cc3247ee602541962588cea252c0.
make check/test/build/release-test PASS using documented cached Go1.26.5 temporary
modfile, tracked1.26.10 unchanged. Approved local HTTP test-port access used;
trusted image build supplies canonical pinned evidence. Existing independent
fixture/custom-user file digests rechecked unchanged (794d555b/a2aba7ed), authored
before provider action. API config validates future consent up to30 days; this
27-hour proposed window fits the reviewed profile. No provider test action yet.

## Remaining work

Exact metadata review, native merge/guarded
dispatch, validate transport receipt and actual revisions. Reconcile the same
saved registration, review/execute/acknowledge authorized synthetic setup, then
controlled four-row custom-user Link and sync. No duplicate registration to escape
UNKNOWN. Provider identity-only map/finite mandate and actual comparator remain
separate required proof. Do not infer financial equality from UI counts.

## PR handoff

Refs272/281/88. No new SQL/migration or product source changes. Preserve money as
decimal strings, no browser persistence of financial requests/tokens. No live
accounts, Production, automatic posting, or full Plaid qualification claim. Add
actual testing receipts/screenshots only after observing the outcome.
