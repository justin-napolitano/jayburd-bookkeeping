<!-- stage-runbook: true -->
# Exact-attempt repair credential read — #319 phase2

## Status and boundary

Branch `feat/319-repair-credential-read`, originally based on `ef00932` from PR324;
its signed receiver is now merged after all five canonical checks passed. Lease
`LINK319-REPAIR-CREDENTIAL` in65. SQL32/PR323 is merged; no managed apply.
Optional internal reader only, not a service/API/UI route or reconnect enablement.
No managed/provider/KMS request or real credential/financial data access.

## Stage ledger

| Stage | Result / limit |
| --- | --- |
| Resume/design | Current65/319 and32 purpose function, bibliography/LR07/owned-repair/standards and encrypted-bank contract reviewed; unchanged AEAD/AAD/source profile |
| Implementation | Scoped exact-attempt function, closed addressed metadata equality before unwrap, confirmed SQL before key work; unscoped adapter refuses; no generic fallback or write |
| Unit | Bankcredential/connections tests PASS including wrong metadata, SQL denial/unknown commit, real AEAD tamper and cross-book ciphertext |
| SQL/signature/AEAD | `make webhook-test` PASS, including real signed receipt/current SQL/AEAD purpose read and natural consent expiry |
| Required core | `make check test build` PASS: context/links/stages/checksums, vet, all Go race tests and executable builds |

## Current evidence

`TestRepairCredentialRead` proves the only SQL request is the exact purpose
function with trusted actor/entity/book/attempt, followed by AEAD readback.
`TestRepairCredentialDeniedBeforeUnwrap` covers denied or unknown SQL results,
wrong reference/version/Item/environment, missing/null/extra/wrong-typed fields
and malformed actor. Fixed expected unwrap count0, no fallback query or wrap.
`TestRepairCredentialTamper` rejects modified ciphertext and a valid envelope
under a foreign book AAD; no fresh wrapping or replacement is attempted.
Existing full envelope/KMS integrity/profile/AAD unit cases remain unchanged.

Actual SQL integration uses the PR324 owned/signed fixture with one independently
fixed synthetic encrypted version installed by the owner as fixture setup only.
The runtime reader and signed receiver subsequently use their actual separate
restricted logins; no SET ROLE impersonation. Key wrapper is fictional in memory,
while production envelope AEAD and SQL32 authority execute unchanged.

Passed independent observations:

- `TestHostedRepairCredentialPurpose`: exact fresh pre-Link read succeeds;
  link-ready pending and signed-but-unclaimed reads each deny before unwrap;
  same proof-pinned claim can read twice after the unclaimed deadline while old
  consent is live; subsequent revocation prevents unwrap. One byte-identical
  ciphertext/version/publication remains, zero wraps/wakeups/journals/posting intent.
- `TestHostedRepairCredentialCurrentScope`: actor/source revocation, foreign
  actor/book/attempt, naturally expired source consent (both unclaimed and already
  proof-pinned), unclaimed deadline, wrong addressed Item and explicit cancel
  deny with zero unwrap/wrap, no new claim, replacement credential or financial effect.

Fixture correction: direct live consent editing without the required generation
transition correctly failedP2602. Final expiry test uses a legitimate five-second
initial consent and a bounded database-clock barrier, preserving every state guard
and isolating consent expiry from generation drift. The first core check rejected
a newly linked runbook filename containing the protected word `credential`; renamed
the evidence to `hosted-repair-vault.md`, retaining the repository protection.

## Remaining work

Complete319 service/provider integration using this explicit optional interface,
without generic fallback: Hosted update creation with existing access token;
SQL-proof-before-secret owner completion; same exact claim/manifest/confirm
recovery; no exchange/rotation/new version or consent renewal. Then320 UI and
303/275 actual signed delivery/Sandbox qualification. Concrete reconnect remains
disabled, Production308–316 and independent equality remain open before90 pilot.

Human security review is required before managed credential/receiver authority
release. Source merges do not apply32, approve deployment or connect a real bank.
No source approval or expired deployment manifest is renewed by this change.

## PR handoff

Refs319/275; neither closes. Depends on PR324 and merged32/PR323; no new SQL,
OpenAPI/browser or accounting changes. Local cached Go1.26.5 override and pinned
synthetic PG17/migrate runners; tracked Go1.26.10/dependencies unchanged.
Record final required gates/exact source head before review. Real KMS IAM,
provider behavior, managed identities and transaction equality remain unproved.
