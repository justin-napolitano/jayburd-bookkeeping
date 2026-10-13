# Private synthetic import staging

## Decision and API

Implemented root-reviewed local foundation for #82 under #8/#55. This adapter stores synthetic
exact bytes only; no decoder, SQL publication, authorization, upload endpoint or
UI is added. Durable-import/statement-decoder contracts remain authority for
future integration. STD-01 synthetic internal CSV assessment is unchanged; no
financial exchange or ISO-conformance claim.

`Open(existingRoot)` requires an owner-created private 0700 directory on local
Linux/macOS storage. `NewReference(Scope{EntityID,BookID})` generates a random
object UUID; all three IDs must be canonical lowercase nonzero UUIDs. Only this
server reference reaches `Put(ctx,Ref,Source)`, `Read(ctx,Ref)` and `Delete(ctx,Ref)`.
Identifiers are not authorization: the future authenticated caller must check
current scope before every operation. There are no caller filenames or URLs.
Receipts contain the reference, byte count and server-computed SHA-256. Read
returns a bounded copy and receipt; it never exposes a path or long-lived handle.

Source owns `Read(context.Context, []byte)` and `Close(context.Context)`. Put
accepts ownership and closes it exactly once, including invalid/cancelled input.
Source implementations must honor the read context and bounded close context.
The adapter cannot cancel an arbitrary blocking io.Reader or uncooperative source;
it does not hide a blocked read in a goroutine. Operations use a 15s deadline,
1MiB plus one sentinel, zero-byte rejection and bounded cleanup. Regular local
filesystem calls/fsync themselves are not context-interruptible; deadline checks
bound admission/progression, not a hung kernel/filesystem call.

## Publication, replay and deletion

Flat names contain only validated scope/object IDs or generated temporary IDs.
os.Root confines operations; nofollow opens, regular-file/owner/mode checks and
0700/0600 permissions reject unsafe entries. Per-object nonblocking flock serializes
cooperating Store instances/processes with context-aware waiting. Uploaded temporary
bytes are private and not a published object. Fsync file then close, acquire lock,
check tombstone and existing exact bytes, hard-link without replacement, then fsync
directory. Existing bytes must compare in full even if digests match. Failed/unknown
publication never returns a receipt; retry reconciles the same reference.

Delete publishes and fsyncs a permanent zero-byte tombstone before removing raw
bytes and syncing the directory. It also tombstones missing references, preventing
late publication after cancellation. Repeated deletion converges; Put cannot reuse
a deleted identity. Reads under the same lock honor the tombstone. Already returned
byte copies cannot be revoked. Tombstones/lock metadata persist; this is not secure
erasure or hosted retention approval. Crashed temp files remain private orphans;
retention sweeps and encrypted Cloud Storage remain separate work.

A cancellation observed before publication leaves no visible object. Once link or
tombstone publication begins, later failure/cancellation is `storage_unknown`:
retry/reconcile, never assume nothing happened. All errors are fixed codes and
contain no paths, payload values, reader errors or raw filesystem diagnostics.

## Sources, alternatives and limits

Reviewed selected mechanisms on 2026-10-13:

- **ST01 — Go Authors, Go1.26.9 `src/os/root.go` and File.Sync implementation**:
  local pinned toolchain source, Root/OpenRoot/OpenFile/Link/Lstat and Sync comments.
  [Public os reference](https://pkg.go.dev/os#Root) corroborates confinement and
  warns Root follows in-root symlinks and does not block mounts/devices. The public
  page served a newer version; pinned local source owns implementation assumptions.
  We add nofollow/type checks and exclude hostile same-UID/admin mutation, mounts,
  network filesystems and unsupported platforms from this local adapter's boundary.
- **ST02 — Linux man-pages6.19**, [link(2), DESCRIPTION/ERRORS/BUGS](https://man7.org/linux/man-pages/man2/link.2.html),
  [fsync(2), DESCRIPTION](https://man7.org/linux/man-pages/man2/fsync.2.html):
  hard links do not replace existing names; syncing file data alone does not
  guarantee directory-entry persistence. Inference: same-directory no-replace
  publication plus directory sync, explicit uncertainty on failure. These calls
  are not tested hardware/power-loss guarantees; network filesystems are excluded.
- **ST03 — Linux man-pages6.19**, [flock(2), DESCRIPTION/NOTES](https://man7.org/linux/man-pages/man2/flock.2.html):
  advisory locks and nonblocking contention support cooperating-process exclusion.
  Inference: immutable lock files, never unlink locks while the store exists;
  unsafe writers/admins are not contained by advisory locking.

Rename can replace existing evidence; a check-then-rename race is unacceptable.
A database blob would simplify metadata atomicity but changes the reviewed storage
boundary. Hash-only replay cannot prove exact equality. A cross-platform unsafe
fallback would weaken pathname/locking guarantees, so unsupported systems fail
closed. Synthetic concurrency, process-exit/fault retry, cancellation, bounds, redaction
and tombstone evidence is recorded in the [living runbook](../runbooks/import-staging.md).
