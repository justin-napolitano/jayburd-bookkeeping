<!-- stage-runbook: true -->
# Private registry delivery —239

## Status and boundary

REGISTRY239, sole SQL42 writer on feat/239-private-registry from2315cc4.
Owner design approval and task state live in238/242/239/65. One additive registry
migration and focused synthetic tests; existing1–41 immutable. No provider/cloud/
private records/secret/environment/state/plan access or runtime deployment.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |AGENTS/security/identity/database/source-first/provider-sync contracts and live239/238/65 reviewed; dedicated worktree/live sole-writer claim |
| Contract |[Physical/operation contract](../contracts/private-registry.md); root reviewed provision/read/query/snapshot/error wire beforeDDL; actual-login/role and canonical legacy/native identity boundaries explicit |
| Implementation |Forward42 and closed owner/API operations; global legacy/native reservation and legacy acquisition fencing; immutable native revisions/holds, original alias reads and exact SHA256 source snapshots. No taxonomy or automatic posting |
| Validation |Actual pinned PostgreSQL17.11 up1–42 PASS.14 restricted-session groups (two deterministic primed-RR subcases) PASS, including both blocked identity-race winner schedules, current revocation/replay, Item generation, actual posted legacy journal/source/cursor preservation, currency/precision holds and3000rows/30pages. Core make check/build and base migration immutability check PASS |
| Down safety |Fresh up1–42→empty down41→up42 PASS withoutCASCADE. Retained private workspace rejects42down with private_registry_retained_records and the original row remains. Mandatory latest-empty migration roundtrip gate retained unchanged |

## Current evidence

Owner approved the source/private classification direction; source239's new SQL
and its concrete test evidence still require review before merge. Docker server
available; use only the existing pinned public PostgreSQL/migrate images and
disposable internal trust network. No inherited DSN or financial fixtures.

Executed names: TestUnassignedScope, TestCrossPrincipalDeny, TestRevisionReplay,
TestCurrencyHoldAtomicity, TestRevokeAndReconnect, TestCollectionPagination3000,
TestCollectionPrivileges, TestCollectionRevocationRace, TestExistingBookMigration,
TestDuplicateItemAndAccount, TestPublicationGenerationFence,
TestLegacyNativeIdentityRace, TestCurrentReplayAuthority and
TestCurrentSourceReadLocks (legacy/native subcases). TestExistingBookMigration
also proves old-source revision invalidates a private cursor, source/import/book
permission revocation denies adoption replay, and raw private source access remains
separate from book permission. No books/journals/outbox are created by native source
storage; the old journal sentinel was independently posted before adoption.

The final complete14-group run passed with3000row publication/query in2.34seconds
on warmed local Docker Desktop arm64; an earlier cold run took24.44seconds.
These timings are disclosed observations, not a latency guarantee. Each page is
bounded100 and counted against the entire matching result, with30unique pages.
Root's independent actual Go HTTP/service/PostgreSQL proof also passed, including
huge original decimal text, dated nullable numeric currency code, denied foreign
principal, invalid/duplicate body, changed-source snapshot and revocation. That
root-only test joins the integration branch; this SQL branch does not claim it
as a committed local test file.

Local Go1.26.5/external modfile/cache adapter used because tracked1.26.10 cannot
download here; module/dependency pins unchanged. make test was attempted: existing
host socket-binding packages fail under local sandbox restrictions. No tests were
skipped or weakened. The ordinary shell Docker runner also inherits that socket
restriction and the host public-pull credential helper failed. Identical pinned
runner steps were therefore executed through permitted direct official Docker CLI
commands, with task-created empty public-auth config, observed local Unix socket,
internal network, disposable volume/database and no real credential/config reads.

Review fixed a provision argument ambiguity caught by actual SQL; replay target
authority, expected Item generation, NULL-sensitive CHECKs, finite timestamps and
current-pointer FK were checked before the final proof. Native Item/source and
adopted legacy source authority rows now lock FORSHARE before visibility, so a
primed RR snapshot after a lifecycle change must abort40001 and a fresh retry
omits/denies the revoked source; both schedules are observed with real row locks. A race-test connection
cleanup bug was corrected before notification and final rerun. No old SQL changed.
Root review identified that always-refusing down would violate the mandatory
latest-empty migration roundtrip. Replaced it with exact empty-only removal after
checking every new custody/audit/reservation table, and tested both successful
empty reapply and refusal with retained records. A rejected down marks only that
disposable fixture's migration metadata dirty; the fixture is discarded, never
force/reset or represented as a clean managed database.
The migration holds an explicit legacy Item admission lock through reservation
backfill/trigger installation and rejects preexisting duplicate canonical identity
groups with fixed P2802 before insert; no identity rewrite/dedupe policy. Existing
41 already enforces that unique identity, so no managed duplicate data was inspected
or manufactured and no broader compatibility claim is inferred from this guard.

## Remaining work

Root integration runs remaining canonical migration/ledger and combined boundary
gates. Publish this reviewable42 patch and exact-head CI; human source/security
review remains before merge. No migration or provider operation against managed
Sandbox/Production was performed.
240 owns actual private provider/credential/worker integration;304 owns taxonomy,
private annotation and book/journal projections. No source feature activation
or live acceptance inferred from the storage foundation.

## PR handoff

Refs239/238/242/412. New forward42 only; no old SQL edits. Sensitive source review,
exact-head CI and named restricted-role proofs required before merge. No cloud
apply/provider consent, accounting policy or live bookkeeping approval.
