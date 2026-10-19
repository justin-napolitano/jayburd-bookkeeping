# Encrypted bank credentials

## Decision and sources

Owner selected encrypted PostgreSQL writes on 2026-10-15 (#162–#164) to replace
per-account Infisical keys, finite slots and source-to-runtime delivery. Static
Plaid client credentials retain the existing Infisical/Secret Manager path.

Reviewed selected official sections on 2026-10-15:
[Plaid persistent tokens](https://plaid.com/docs/identity/add-to-app/#get-a-persistent-access_token)
requires secure server-side persistence, not Infisical specifically;
[Cloud KMS envelope encryption](https://docs.cloud.google.com/kms/docs/envelope-encryption)
describes local data encryption keys protected by a remotely held wrapping key;
[KMS integrity](https://docs.cloud.google.com/kms/docs/data-integrity-guidelines)
describes request/response CRC32C and verification flags. These sources support the
mechanisms, not certification or hosted qualification. Our scoped SQL boundary,
immutable version policy and retry rules are application design decisions.

## Persistence and authority

### Canonical KMS project identifier correction (#194)

Reviewed 2026-10-19: Cloud KMS [resource hierarchy](https://docs.cloud.google.com/kms/docs/resource-hierarchy)
and [symmetric encryption](https://docs.cloud.google.com/kms/docs/encrypt-decrypt)
use `projects/PROJECT_ID/.../cryptoKeys/KEY_NAME` for a key resource. Review covers
the resource-name table and encrypt/decrypt request examples, not all API behavior.
An operator-controlled probe encrypted only a fixed synthetic marker: the numeric
project resource returned permission denied, while the project-ID form succeeded
with the same principal and fixed key. The attempted API-account impersonation
was denied; this probe does not independently qualify the runtime identity.

Decision: compose API and acquisition KMS resources from their validated project
ID. Secret Manager references continue using their separately validated numeric
project number. Go checks the exact configured key and numeric immutable version;
it does not rewrite a returned resource name because that name participates in AAD.
Forward migration 20 permits canonical project IDs in both ciphertext-write
functions while preserving existing numeric metadata for compatibility. Runtime
key equality still rejects foreign projects/keys. No ciphertext or AAD is rewritten.
The down migration refuses when canonical-ID envelopes exist, before restoring
numeric-only write validation. This avoids claiming a compatible rollback after
new envelopes have been written.

Acceptance: canonical-ID KMS request/CRC/readback tests, numeric-key configuration
rejection, SQL canonical and malformed-name cases, migration forward/down guards,
and synthetic connection/binding/acquisition. A fresh Hosted Sandbox run remains
required after migration and runtime promotion. Official Plaid SDK adoption is
tracked separately in #195 and does not repair a KMS resource-name failure.

Migration 18 adds integration.bank_credential, owned by bookkeeping_owner. Store
only AES-256-GCM ciphertext, a random 96-bit nonce, KMS-wrapped random 256-bit data
key, format version and wrapping-key-version resource. No token, unwrapped data
key, or plaintext envelope crosses SQL. PostgreSQL encryption at rest alone is
not this protection. KMS permissions and SQL permissions are independently required.

Canonical UTF-8 JSON array AAD binds profile, entity, book, credential reference,
logical version, provider Item and environment. Both local AEAD and KMS wrapping
use that binding; local AEAD additionally authenticates the returned wrapping-key
version. Reject foreign scope/reference/version/Item/environment, altered
ciphertext/nonce/wrapped key/key resource, malformed metadata and invalid CRCs.

API SQL wrappers recheck current connection authority and the initiating actor's
matching claimed attempt before write/read. Writes require a recorded manifest
and exact SQL-assigned reference/version. A worker reads only through current
sync_context lease/fence authority; the function obtains the credential reference
from SQL, never a caller-chosen lookup. No application gets direct table access.
No key or token belongs in API responses, receipts, logs, migrations or Terraform.

Rows are immutable per scope/reference/version. Put first reads an existing row
and verifies its decrypted token equals the supplied memory-only token. Concurrent
writers return the existing winner; decrypt/compare rejects a different token.
Confirmed same-version retries and lost commit recovery do not require identical
new randomness, and never repeat Plaid exchange. New logical versions append rows.
No plaintext fallback or automatic legacy-secret import. Legacy resources remain
untouched; an encrypted runtime rejects missing rows instead of guessing storage.

## Failure and deployment

The exchange claim still commits before Plaid mutation. The encrypted store checks SQL function permissions and performs a fresh
KMS wrap/unwrap preflight before consuming a public token; SQL write/readback and
credential confirmation precede binding review. Exchange and DB are not one atomic
transaction: a crash after exchange but before durable ciphertext may lose the
token. Such an attempt cannot be repaired by repeating a consumed public token.
Existing #160 has no observed source token; preserve its audit trail and use an
explicit abandonment/new-enrollment path after classification, not manual reset.

Provision the fixed development KMS key in workers infrastructure; API encrypts
and decrypts, acquisition only decrypts. No publisher decrypt permission. Forward
migration must precede matching API/worker rollout. The tables are additive, but the new abandoned state requires updated UI/API
clients. Old binaries must not handle abandoned attempts. Old/new credential formats
are not silently interchangeable; complete the coordinated application rollout
before using the abandonment action.

## Standards and acceptance

This internal credential-storage protocol is not an ISO 20022 financial message.
It does not change currency/accounting contracts or imply ISO cryptographic
certification. Named evidence must cover fresh/replay migrations, restricted API
and worker SQL, current revocation/lease checks, immutable conflict/lost commit,
tamper/wrong AAD/KMS key/integrity denial, redaction and synthetic end-to-end
connection/binding/acquisition. Hosted Sandbox acceptance remains separate.


## Explicit lost-token abandonment

POST `connection-attempts/{attempt}/abandon` requires
`{"confirm_unrecoverable":true}`. Current initiating-actor connection authority is
rechecked. An unclaimed new setup, or a new unpublished credential-recovery
attempt with no stored DB ciphertext, can move to immutable `abandoned`. Preserve nonce/hash/claim/manifest
and append an audit event; do not delete or recycle anything. Discovery excludes
this terminal state from unfinished_count. It never removes the provider Item;
known orphaned Sandbox Item cleanup remains explicit. **Try connecting again** is
the user's affirmative request to resolve and restart a failed new enrollment.
For claimed credential recovery the UI first reconciles the exact attempt; only a confirmed recovery conflict
followed by an exact read of the still-unresolved attempt can invoke abandon.
The server checks that no ciphertext or credential publication exists before
accepting the command. An uncertain response or another unfinished attempt
stops the restart. The old provider exchange is never repeated.
This route is an internal operational command, not an ISO 20022 message.

Unclaimed new-setup cancellation additionally follows [#200](hosted-link-enrollment.md#explicit-cancellation-and-retry-200); it cannot discard a claimed exchange or published credential.
