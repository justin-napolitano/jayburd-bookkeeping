# Hosted acquisition evidence

## Decision

Cloud Run acquisition uses a private GCS bucket, never its ephemeral filesystem.
Application writes create a bounded raw page once using a generation precondition; its name is a SHA-256 digest of
the complete server-owned evidence key. Reads pin the returned GCS generation
and validate CRC32C. SQL remains scope/publication authority. Storage has no
account-discovery or journal-posting capability.

An independent create-only tombstone wins over concurrent writes and cannot be
removed by application code. Delete writes the tombstone before attempting raw
object deletion. A racing late raw upload remains unreadable and expires through
the bucket's seven-day raw-object lifecycle. This is logical deletion, not proof
that all cloud-held copies were immediately erased. Tombstones have no automatic
expiry; physical cleanup/retention needs hosted evidence before live data.

The initial bucket is development/Sandbox-only: uniform access, public access
prevention, no versioning, soft delete disabled, encryption at rest supplied by GCS.
Only the sync identity gets raw-prefix object get/create/delete permissions;
tombstones get create/read only. Raw immutability is an application CAS invariant,
not storage-enforced WORM: a compromised raw-delete identity can replace raw objects.
No public ACL, arbitrary bucket, redirect, proxy configuration or ambient credential
discovery is accepted. Each HTTP call has a five-second deadline and fixed errors.

## Sources and alternatives

Reviewed selected official GCS JSON v1 sections on 2026-10-14:
[request preconditions](https://docs.cloud.google.com/storage/docs/request-preconditions),
[insert](https://docs.cloud.google.com/storage/docs/json_api/v1/objects/insert),
[get](https://docs.cloud.google.com/storage/docs/json_api/v1/objects/get), and
[object metadata](https://docs.cloud.google.com/storage/docs/json_api/v1/objects).
Generation-match zero provides conditional creation; reads can select a generation;
CRC32C is transport integrity evidence. Our immutable-key/tombstone design is an
application inference, not a GCS promise of business authorization.

Local staging cannot survive Cloud Run replacement. Overwriting raw objects would
break replay equality. Separate tombstones avoid overwrite authority for deletion
state and preserve denial after late uploads. ISO20022 does not apply to this
internal opaque storage protocol; source currency/date contracts are unchanged.

## Acceptance

Named tests cover exact replay/conflict, scoped keys, generation-pinned reads,
checksum failures, lost upload responses, concurrent tombstone/write, deletion,
redirect denial, bounds and cancellation. Synthetic HTTP tests establish protocol
behavior; they do not prove deployed bucket IAM or provider qualification.
