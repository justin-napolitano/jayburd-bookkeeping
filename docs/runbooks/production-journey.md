<!-- stage-runbook: true -->
# Production journey proof — #315

## Status and boundary

PRODUCTION315-JOURNEY; test/315-production-journey from3a2365c.
Synthetic-only Production profile, no real authentication/provider/cloud/secret
access or managed mutation. #337/#316 and hosted qualification remain separate.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live315/65 read;311–314 closed,345 merged, bounded test lease |
| Decision | production-journey contract before test changes; existing PP01–09/standards/fixtures reviewed |
| Implementation | Explicit Production service/profile/bank+Link key adapters with actual restricted operator/API/worker logins; independent empty/history decimal fixtures |
| Finding | Initial empty sync returned invalid_metadata after SQL completion; linked346/PR347 fixes valid source revision0 without changing SQL |
| Browser | TestProductionProfileJourney PASS: one binding, lost completion result, empty/replay, two history pages/replay, pause/resume/disconnect denial, explicit posting/lost-result recovery and exact0.10 report change |
| Regression | check/test/build/ledger-test PASS; existing connection-state/webhook suites PASS, including Production scope/credential/posting/reconnect negatives |
| Web | app-integration, unchanged connection-browser-test, web-check/test/build/browser PASS;342 unit and26 browser tests |

## Current evidence

314 CI all five passed before merge3a2365c.315 composed proof includes the exact
346 fix patch (9447ea5); that fix is not treated as merged by local testing.
Independent shell assertions retain original0.10/0.20 lexemes and exact0.3 sum,
one provider-origin Production Item, two observations and one explicitly reviewed
journal/outbox after posting. Empty/catchup acquisition itself creates no journals.

Test-only identities do not prove managed identity/MFA. Fictional wrapping keys
exercise actual AEAD/AAD but not KMS IAM. Local synthetic evidence proves scoped
retention/replay, not actual GCS isolation/backup/deletion. Production intentionally
uses controlled owner/worker bootstrap; bookless/self-service remains237. UI still
says Sandbox in Hosted Link instructions: correct before Production release.

Local Go1.26.5/external temporary modfile/cache used after pinned1.26.10 download
failed403; tracked toolchain unchanged, canonical exact-head CI required. Locked
npm install reports0 advisories and4 pending lifecycle scripts; no approval added.
Initial fixture failures used the wrong constructor and superuser operator session;
corrected to the explicit Production service and actual restricted operator login.
Existing Sandbox data/expected results were not relabeled or rewritten.

## Remaining work

PR/exact-head CI and reviewed dependency346 merge; then337/316 decisions/release.
Owned signed reconnect uses passing TestProductionOwnedSessionSQL from314 with
same addressed encrypted credential; browser test does not claim actual provider
reconnect, OAuth/mobile or hosted recovery320/276. Existing Production revocation,
missing/revoked/stale/foreign posting and cross-profile tests remain separate
component evidence, not one browser scenario. No missing gate silently waived.

## PR handoff

Refs315. No SQL/product feature/dependency change. Record limitations including
test-only identity and external fake KMS/provider; no pilot activation implied.
