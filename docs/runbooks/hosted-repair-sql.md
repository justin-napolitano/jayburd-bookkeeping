# Owned Hosted repair SQL
<!-- stage-runbook: true -->

## Status and boundary

#318 SQL implementation prepared under `SQL318-OWNED-REPAIR`, sole SQL writer.
Base27062957592bef2128c85969eafa5b302968d39b; branch
`feat/318-hosted-repair-sql`. Forward32 reserved after current31/#65 check.
The [reviewed protocol](../contracts/hosted-reconnect.md), merged321 after all5 CI
checks, owns the design. No actual provider event, production, cloud mutation or
managed apply32. Source319/UI320 and dedicated signed delivery remain separate.

## Stage ledger

| Stage | Outcome |
| --- | --- |
| Resume | Read current318/275, merged protocol/source register, database foundations/standards and current13/14/19/23/28 functions |
| SQL review | Two immutable grains, exact restricted interfaces, full principal/source/Item/attempt lock order and compatibility reviewed below |
| Implementation | Forward32, two immutable receipt grains, scope pin, private credential read, ten compatibility wrappers and closed bootstrap allowlist |
| Independent SQL tests | All16 Hosted repair cases PASS on actual restricted-login disposable PostgreSQL, including forced duplicate snapshots and receipt/claim/cancel races |
| Required gates | Core/race/build, migration, ledger, connection state/service/ops, webhook/bootstrap,342 web units/26 browser cases and real synthetic browser/API/SQL command recovery PASS |
| Handoff | Pending PR; no hosted gate or issue closure |

## Current evidence

Existing source has no concrete HTTP reconnect, so SQL guards must land before
enabling it. Merged19 changes the default and record extension for both modes;
32 must cap only reconnect. Attempts have no created_at; the unique immutable
begin event provides the age origin. Current confirmation otherwise copies a
manifest expiry, requiring an explicit reconnect no-extension check.

SQL design: one immutable `link_session_receipt` row per verified delivery
(environment/receipt SHA key; exact attempt FK, hashed Link/session/body, closed
outcome and signed issuance time). One immutable `link_repair_claim` per claimed
attempt pins a successful receipt, claim UUID, existing credential identity and
old consent ceiling. API can read only owned proof projection and execute its
existing commands; dedicated webhook can only accept normalized signed receipts.
Neither runtime role gets table access or old-helper EXECUTE. No raw provider
body/JWT/token or financial amounts enter these tables.

The attempt's immutable-once scope digest pins source binding/stream revisions
and states only at fresh begin, excluding changing cursors/demands and labels.
Legacy missing snapshot denies; never infer it from current rows. This bounded
profile requires the original consent principal, with no delegated-owner switch.
The receiver has its own narrow SQL-selected-actor locker: global API/ledger
cross-role denial is unchanged, no SET ROLE or event-supplied actor. Refreshed S12
PG17 full SECURITY DEFINER safety section; qualified names, pg_temp last, exact
EXECUTE grants/revocations occur within the migration transaction.

All principal identities (actor, configured worker, existing consent principal,
stream service principals) lock in UUID order before current authority/book,
source accounts in order, Item, attempt, then receipt/claim. Source/Item generation
and complete stream set recheck after locks. Receipt pins no claim or wakeup;
confirmation uses stable repair:attempt signal per active approved stream and
preserves paused streams. Exact same effect key cannot advance again.

Compatibility: retain new/rotate/remove semantics and public API shape; add private
`item_consent_expires_at` only for reconnect service use. Old concrete provider
still rejects reconnect. Legacy unclaimed reconnect's mutable expiry is capped
once with an immutable policy-change event; immutable history is never rewritten.
Missing/ambiguous begin provenance expires eligibility rather than guessing.
Legacy claimed reconnect without signed proof cannot resume. Existing terminal
receipts remain scoped readable; no item/credential/journal alteration from this
backfill. Fresh attempts get the same explicit cap before returning a Link.

## Remaining work

Exact-head canonical CI and source review remain before merge; human security
review precedes managed release. The closed bootstrap allowlist has only the
two intended receipt functions; no broader role or IAM exception. #319 must cap
manifest consent at the SQL-projected live ceiling,
check signed proof before bank-token work and recover exact unknown claims.
The purpose-specific `get_repair_bank_credential` read derives the existing tuple
from this repair;319 must wire it into the scoped vault instead of18's generic
expiry-ordered attempt lookup. Reconnect never writes another encrypted version.
This SQL change alone cannot enable concrete reconnect or clear service recovery.

Review distinction:15minutes limits unclaimed Link/proof/claim. A committed
proof-pinned claim must remain explicitly reconcilable after Link expiry while
original source consent/current scope are live; otherwise a lost reply strands
the paused Item. No new Link, execution grant, exchange, credential version or
source deadline is created. This follows13's existing unclaimed-vs-claimed recovery
boundary; dedicated expiry-recovery test and current-authority denial are required.

Executed evidence: `TestHostedRepairSQLProofAndPublication`, `Ambiguous`,
`ScopeDrift`, `PausedSibling`, `RoleAndReceiptBoundary`, `ConcurrentClaimAndConfirm`,
`ReceiptClosure`, `ExpiredConsent`, `CredentialPurpose`, `LegacyMigration`,
`HandoffRecovery`, `KnownConsentExpiry`, `CancelRaces`, `OriginalOwner`,
`DuplicateRace`, `ClaimedExpiryRecovery` (all share the TestHostedRepairSQL prefix).
The duplicate case holds the book to establish two actual waiting snapshots,
then releases them; one stored receipt, accepted/duplicate outcomes and no effect.
Known narrower consent expires against the real recorded server-clock deadline
within a bounded5second fixture barrier; immutable manifest evidence is not edited.
The legacy case runs actual32 down/up around a fictional31-era unclaimed attempt,
preserving source identities and requiring explicit cancel rather than rescoping.

One-shot fresh Link response permission was accidentally masked by an earlier
projection merge; actual service integration caught it. The wrapper now preserves
the single returned permission; a fresh-vs-replayed SQL assertion prevents its
return. Generic credential lookup by expiry was not reused for claimed repair;
the exact-purpose read avoids selecting older active enrollment metadata. Existing
state/service fixtures now supply distinct synthetic token association and a
normalized receipt through a real dedicated login, not a crypto/provider claim.

Local `make check test build migration-test ledger-test webhook-test
connection-ops-test app-integration` PASS on final SQL; connection state/service
and webhook-bootstrap regressions PASS. `make web-check web-test web-build
web-browser` PASS:342 units,26 browser cases. Staged runbook/diff gate required
before commit. Cached Go1.26.5 temporary modfile used locally; tracked1.26.10
unchanged, exact-head canonical CI remains separate. Locked npm dependencies
installed, no new package or runtime architecture. The integration's identity is
test-only and explicitly proves no managed MFA/provider/cloud/live readiness.

PR322's selected app build expects schema31. Do not apply32 ahead of its exact
status-only release; later32 runtime/apply needs its own compatible images and
reviewed migration/release order. No silent renewal, source widening or fake
synthetic marker for real data. Production308 remains a separate contract.

## PR handoff

Refs #318/#275; forward migration32 only when fully tested/reviewed. Down is for
disposable testing, never a live recovery procedure. No financial/ISO message or
currency mapping changes; proprietary session state follows STD-01 and existing
API2020-09-14/SDK49.0.0, exact money/original units unchanged. Named tests will be
recorded only after execution; no conformance or hosted qualification claimed.
