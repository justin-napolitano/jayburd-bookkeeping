<!-- stage-runbook: true -->
# Owned Hosted repair service — #319

## Status and boundary

`LINK319-HOSTED-SERVICE`, dedicated `feat/319-owned-hosted-repair` worktree.
Original base325/a92adda; final branch rebases on33/PR327 after preserving this
bounded Go source.324/325 are merged.326/327 is an explicit pending source
dependency until CI/source acceptance; never treat it as completed beforehand.
No new SQL, API/OpenAPI/UI, CLI/runtime/release or infrastructure change in this
branch. `HTTPConfig.HostedRepair` is separate/default false; existing runtime
composition does not set it. No deployment/managed apply/provider/real data access.

## Stage ledger

| Stage | Outcome / limit |
| --- | --- |
| Resume | Current65/319/318/326, protocol32, bibliography/LR07, standards/database/decision evidence and vault profile reviewed |
| Initial prototype | Units pass; concrete canonical SQL/AEAD/SDK Begin failed because32 reconnect handoff copied19 numeric-only KMS grammar; failure preserved |
| Separate prerequisite | Released service lease; sole-writer326 forward33 restores reviewed20 grammar without editing32 or substituting an easier key fixture; PR327 pending |
| Composite service proof | Same Go source plus exact sealed33 pair/checksum PASS actual signed receipt/SQL/AEAD/concrete SDK injected transport cases |
| Capability decision | Separate explicit HostedRepair false plus exact existing receiver; preserve prior webhook-only disabled behavior, reject implicit permission expansion |
| Required gates | Final rebased core and connection/ops/provider/browser recovery gates pending |

## Current evidence

`TestHostedRepairPendingNoIO`: pending/exited/ambiguous/invalid proof reaches no
handoff, key, bank operation or claim through either completion entrypoint.
`TestHostedRepairCompletion`: one existing credential read, one read-only manifest,
original consent cap, zero generic read/write/preflight or new-mode verification.
`TestHostedRepairUnknownClaim`: lost committed claim with no manifest recovers
the exact attempt after Link expiry, with no new handoff/read/exchange.
`TestHostedRepairRequestAndScopeDenial`: changed nonce/Link/public token and
revoked owned context deny before bank work. `TestOwnedHostedRepairWire`: exact
fixed API version/host, Hosted900seconds, existing credential/receiver, no
products or account-selection widening; constructor inert and separate opt-in.
`TestHostedRepairNoAutomaticRestart`: expired-Link unclaimed cancellation uses
SQL only and never removes/restarts the provider Item. Existing global approval
denial regression remains unchanged and passed the unit check.

Actual restricted SQL + signed HTTP + production AEAD + pinned SDK fixture
(fictional in-memory KEK, injected provider transport; no real provider):

- `TestHostedRepairServicePublication`: pending repeated completion performs no
  additional key/provider work; signed success publishes one same-version repair
  cause, new discovered account stays unbound, original ciphertext/consent unchanged.
- `TestHostedRepairServiceUnknownReply`: actual committed claim, manifest and
  confirmation reply losses each recover the exact attempt, including a missing
  manifest and expired unclaimed Link window; one Link/claim/resume, no exchange.
- `TestHostedRepairServiceRejectedScope`: exited/two-session proof, current
  revoked/expired source or Link, missing approved account, changed type/currency
  cannot publish a resume; denied preclaim work reaches no handoff/key/provider.
- `TestHostedRepairServicePaused`: explicit pause survives successful repair,
  requested generation1 remains, no wakeup from the repair.

Fixed observations: one original encrypted bank version/publication, one original
source/stream, zero observations/journals/`journal.posted` outbox, exactly one
proof-pinned claim and one stable resume only after successful active-stream
confirmation. The tests do not claim actual signed delivery or bookkeeping equality.
Link handoff encryption legitimately wraps its distinct scoped ephemeral key during
Begin; completion/recovery never creates another bank or Link envelope.

## Remaining work

Finish final current-head gates and review; merge327 only after its canonical CI.
Explicitly track CLI/runtime/finite release capability propagation before319
closure or managed UI flow.320 owns browser waiting/reload/recover/cancel and
actual qualification;303 owns dedicated signed delivery. Human security review,
image/schema33/bootstrap compatibility and fresh exact owner release scope are
required before managed repair. Production308–316, independent Sandbox equality
and the one-account90 statement reconciliation remain unproved.

The existing global enrollment approval remains required for provider entries,
including the current abandon route. Expired global approval never permits a
bank request or renews source consent; local SQL bookkeeping recovery remains
the separate281 profile. Prior expired31-only Sandbox manifest was not dispatched.

## PR handoff

Refs319/275; partial source phase, neither closes. Depends on merged32/323,
324/325 and forward33/327. No new migration or credential crypto/metadata policy.
Tracked Go1.26.10/dependencies unchanged; local cached Go1.26.5 override. Record
final exact-head core/integration/frontend evidence and remaining risks before PR.
Default runtime cannot enable repair from this source alone; no actual hosted or
Production/personal-account clearance follows from source tests or a merge.
