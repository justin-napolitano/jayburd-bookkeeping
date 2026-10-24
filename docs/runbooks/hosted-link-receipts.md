<!-- stage-runbook: true -->
# Signed Hosted Link receipts — #319 phase1

## Status and boundary

Branch `feat/319-owned-link-receipts`, base `e764786` (merged SQL32/PR323).
Lease `LINK319-SIGNED-RECEIPTS` in #65; one bounded signed-receiver phase of319.
Synthetic source work only. No managed migration, bootstrap, deploy, provider
request or secret/financial data access. `SupportsMode(reconnect)` remains false.
SQL32 is merged but not applied to the managed Sandbox database (currently31).

## Stage ledger

| Stage | Result / limit |
| --- | --- |
| Resume | Current65/318/319, index/bibliography, LR07, protocol32, standards, database foundations and decision evidence reviewed;318 merged after all five canonical checks |
| Source refresh | Official Link API SESSION_FINISHED properties/default enablement rechecked2026-10-24; selected full section only, no actual provider event |
| Design | Private event-kind projection after common signature/raw-byte verification, fixed closed SQL dispatch, private kind-specific acknowledgement; contract records bounds and alternatives |
| Implementation | Hash/discard tokens/user, retain existing Transactions path, preserve commit uncertainty and actual-login checks; no service/credential/UI enablement |
| Signed proof | `make webhook-test` PASS on disposable PostgreSQL17 with32 and actual separate runtime/owner/webhook logins |
| Core gates | `make check test build` PASS, including all Go race tests/vet, migration checksums, links/context/stage checks and executable builds |

## Current evidence

`TestHostedRepairSignature` denies invalid signatures, stale/future issuance,
raw-byte alteration, duplicate/unknown JSON, wrong environment/kind/code/outcome,
empty/oversize/non-ASCII identifiers and trailing JSON before business persistence.
Existing algorithm/JWK/expiry/cache/admission/header/size/cancellation negatives
remain in `TestWebhookRejectedInputs`, `TestWebhookKeyLifecycle` and companions.
`TestHostedRepairOptionalProfile` covers closed optional/null/array limits;
`TestHostedRepairReceiptKind` denies acknowledgement substitution between kinds.
`TestHostedRepairNoSecretAccess` checks the hashed operational projection and
discarded optional tokens/user; public receiver composition has no bank or KMS client.

Actual signed HTTP→restricted SQL evidence:

- `TestHostedRepairSignedReplay`: commit succeeded but reply lost→503; same and
  equivalent ECDSA signatures→204 with no additional receipt; a newly signed
  same session retains a second delivery without ambiguity. Stored rows contain
  hashes/outcome only, no raw tokens/JWT/user. Actual webhook login cannot call
  the API proof reader or read the bank credential table.
- `TestHostedRepairSignedForeignScope`: unknown/new-mode token→ignored204,
  no retained repair proof, original attempt stays pending.
- `TestHostedRepairSignedAmbiguous`: exited, conflicting outcomes and two
  sessions retain the expected blocked projection without any claim.
- `TestHostedRepairSignedRevoked`: current connect/source-worker permission,
  local deadline and disabled ingress each deny both replay and new session403,
  without a new row or effect.

Independent fixed assertions: zero repair claims, wakeups, observations,
journals, `journal.posted` outbox and bank-credential writes; existing credential
version1/publication count1 and requested acquisition generation1 unchanged.
Old Transactions receipt/commit-loss/catchup tests also pass, retaining their
distinct authorized wakeup behavior. These checks qualify the local receiver
boundary, not a successful reconnect or source transaction equality.

Local environment: cached Go1.26.5 override with temporary modfile/cache; tracked
Go1.26.10/dependencies unchanged. Runner uses exact pinned cached PG17/migrate
images on a disposable internal Docker network, no inherited DSN or TLS bypass.
The first sandboxed core run could not bind localhost test listeners. The required
run was repeated with that permission and passed; no source change or weakened
test was used to avoid the listener gate.

## Remaining work

Complete319 existing-credential purpose read/Hosted update/explicit proof-gated
completion and same-claim manifest recovery. Then320 UI/reload and actual303
dedicated signed delivery/275 Sandbox recovery qualification. Human security
review is required before managed receiver authority/crypto release; migration32,
binary and exact closed bootstrap must be promoted compatibly under fresh scope.
Production308–316 and independent Sandbox equality remain open before90 pilot.
No real account has been connected or personal transactions reconciled here.

PR322 Sandbox manifest dispatch window ended20:00UTC2026-10-24 without owner
approval; it was not dispatched. Do not reuse that expired manifest or infer a
consent extension. A fresh exact-byte finite release needs owner review per
`deploy/README.md`; source merging alone is not deployment permission.

## PR handoff

Refs319 and parent275; partial phase only, neither closes. Depends on merged
PR323/32. No new SQL or API/OpenAPI/browser contract change; runtime requires
32 and its restricted receiver bootstrap before accepting Link receipts.
Record final validation and exact head at review. Actual provider update shape,
managed IAM, signature delivery, reconnect and financial equality remain unproved.
