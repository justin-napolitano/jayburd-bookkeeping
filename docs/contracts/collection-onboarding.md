# In-app private workspace onboarding — 460

## Decision and authority

Owner requested2026-11-08: stop manual/ad hoc SQL and finish the application
flow. This supersedes routine operator setup under453; admin-only historical
scripts and optional inline459 do not become normal onboarding. Existing active,
enrolled, Firebase/TOTP verified users can explicitly create their own absent
private workspace. Identity invitation/enrollment remains unchanged. No UID,
email, principal, capabilities, operator proof or evidence is accepted in HTTP.
The trusted authenticated actor is the sole workspace owner, independent of
book membership or legal-owner assertions. Fixed capabilities read/acquire/classify
are disclosed in review and created only by explicit execute. Existing workspace
capabilities are never elevated/restored by this workflow.

Adoption is explicit for one currently authorized existing connection and its
complete active-source set. Discovery and preview are inert. Existing42 adoption
owns exact consent principal, current book/entity/source authority, Item generation,
complete-set, unresolved connection/staging checks, pause/generation advance and
unchanged original anchors/credential AAD/cursors/history. No provider fetch,
consent renewal, acquisition resume, source copying, journal or posting occurs.
New normal book-scoped registrations become eligible for this same mounted flow;
connect-first provider enrollment without a book remains240, not a claimed feature.

## Runtime and storage

Forward47 only; merged1–46 immutable. Add principal-scoped immutable preview and
command records, a typed third arm in existing interactive_command_reservation,
and exact recovery closure receipts. Reuse persistent44 principal recovery fence;
no second independent user gate. Scope new managed transport to existing Sandbox
project/private-enabled configuration; preserve current authenticator/browser
origin boundary. No new IAM/resource/owner-role membership or API EXECUTE on the
general owner-only provision helper. A narrow SECURITY DEFINER wrapper may call
it internally with its own trusted actor, fixed caps, null expected revision and
command-derived key/evidence after checking actual API login/current principal.
Closed wrapper requires SERIALIZABLE and current permissions on every operation.

First lock principal, follow existing book/workspace/Item lock order, then44
recovery fence last. Preview/prepare/execute pin current workspace absence or
revision and adoption generation/complete sources/authoritative binding revisions.
Prepare stores immutable exact server preview under one principal-scoped key;
same-key replay precedes expiry/staleness rejection after current authorization.
Setup execute requires workspace still absent. A committed exact command replay
returns only under current disclosure authority. Current disabled/revoked state
denies all payload disclosure. Cancellation/acknowledgement use current disclosure
authority without requiring old generation pins that execution intentionally
advanced. Known-aborted transactions can retry; unknown commit returns503 with
same-command explicit recovery. No automatic execution/acknowledgement.

## Closed wire v1

Routes under /v1/collection-onboarding, all authenticated/no-store:

- GET root: Catalog profile private-onboarding-catalog-v1. workspace is
  {state:absent} or {state:ready,workspace_id,authority_revision,capabilities} or
  {state:unavailable}. can_setup/can_adopt booleans; candidates array, max100,
  complete or explicit422, no silent truncation. Each candidate contains
  candidate_id (existing opaque local collection_item UUID), entity_id, book_id,
  entity_name, book_name, snapshot (64lowercasehex), and sources array of
  {source_id,display_name}, max25 complete active sources. No provider IDs,
  credentials, account numbers, cursors, financial observations or legal-owner claims.
- POST /previews: exact {operation:setup} or
  {operation:adopt,candidate_id,snapshot}. Server resolves full tuple and safe
  complete source labels; it does not accept arbitrary target/source/name mapping.
  Return Preview profile private-onboarding-preview-v1, preview_id, operation,
  expires_at, workspace_id null|UUID, snapshot, review object. Setup review is
  {capabilities:[read,acquire,classify]}; adopt review contains candidate fields
  plus expected_generation and impact pause_legacy_acquisition. Adopt review is exactly the catalog candidate fields plus expected_generation
  (positive decimal string) and impact=pause_legacy_acquisition.
- POST /commands: exact {preview_id,prepare_key}; return Command.
- GET /commands/pending: global Pending, works before workspace exists.
- POST /commands/{command_id}/execute, /cancel, /acknowledge: exact {}.
- POST /recovery/clear: exact {recovery_id}, inaccessible own onboarding only.

Command profile private-onboarding-command-v1: command_id, recovery_id (stable exact claim/closure UUID),
prepare_key (exact saved key), state
prepared|committed|cancelled, preview, result nullable, acknowledged boolean.
Setup result retains existing workspace_id/workspace_revision/capabilities;
adoption result retains42 collection_item_id/item_generation/sources.
Pending profile private-onboarding-pending-v1: state none|available|inaccessible|
blocked, command null|Command, recovery_id null|UUID. blocked means another
engine owns the gate and discloses no payload; never treat it as none or clear it
through onboarding. Inaccessible clear returns profile
private-onboarding-clear-v1, recovery_id, cleared boolean and writes durable
exact closure; stale retry cannot release a newer command. Ordinary terminal ack
also closes its exact claim. The command persists/echoes its own immutable
recovery_id before transitions, and the reservation uses that exact identity.
Pending available must match envelope and command recovery IDs. If an actual ack
reply is lost and authority is revoked before replay, the retained opaque ID can
prove its typed durable closure through clear without exposing saved payload. Reads use SERIALIZABLE fence to prevent stale-none
discovery/admission. Other engines retain their historical wire semantics and
recognize busy onboarding without decoding unsupported payloads.

Bounds: preview request8192, prepare4096, transition/clear1024 bytes; all persisted
and HTTP responses <=1MiB with wrapper headroom. Preview15min; a prepared command
remains recoverable after expiry. Closed duplicate/unknown/case-alias JSON; UUID,
key and snapshot validation server-side. Fixed400invalid,401auth,404denied,
409conflict/stale/pending,422limit,503unavailable; never raw SQL/plan diagnostics.

## Mounted user experience

Always-mounted onboarding pending discovery joins existing book/label/opaque
recovery locks before ANY new interactive work enables, even before /collection
exists. Catalog reports actual setup eligibility; 404 is never permission proof.
All transactions offers Set up private transactions, fixed capability review,
Save review, explicit Create private workspace, then Acknowledge result. An empty
workspace offers authorized existing connections, server-owned full-set review,
explicit adoption confirmation explaining legacy sync pause, and acknowledgement.
Then refresh collection/taxonomy/rows. New connection registration refreshes the
same candidate discovery; it does not silently adopt, reconnect, fetch or post.

Persist review only server-side. Browser memory holds exact uncertain preview/key
or command ID, never local/session storage. Lost prepare retries same key; remount
discovers actual principal gate. Lost execute never creates a new command; remount
renders saved prepared/terminal work and requires explicit execute/ack. Lost ack
retries exact command and checks exact terminal acknowledged reply, not pending:none
alone. API/session changes synchronously hide old scope/seed and discard late replies.
Conflicts refresh all engines while keeping locks until current discovery succeeds.
Keyboard/focus/mobile and error/retry states must be usable, not dead-end SQL advice.

## Evidence, standards and alternatives

Builds on ADR0006, private-registry42, command-recovery, self-service-connections,
collection-tags-recovery44 and actual API-role adoption tests. PostgreSQL17
CREATE FUNCTION, Writing SECURITY DEFINER Functions Safely (database-sources S12)
and Transaction Isolation, Serializable Isolation Level establish safe search_path,
explicit function grants and whole-transaction retries. RFC9110 safe-method and
idempotency distinctions (recovery-sources) support inert discovery and durable
same-intent retries. Re-reviewed2026-11-08 the official PG17 SECURITY DEFINER safety section in full,
Serializable section and RFC9110 sections9.2.1–2. Existing source registers/relevant sections are the source
record; this own-workspace self-service policy and fixed capabilities are a new
owner-authorized project decision, not a claim that PostgreSQL grants user rights.
Alternative manual SQL leaves the observed onboarding gap; runtime owner membership
or caller actor/capabilities widens authority; silent adoption hides a sync transition.

Internal authority/recovery metadata is not an ISO20022 financial exchange or
ISO21378 audit export. Original currency/dated ISO4217 evidence and accounting
recognition remain unchanged. No conformance or live bookkeeping approval inferred.

Security review flags: restricted metadata, authenticated public transport and
bounded authorization change require review. STRIDE: spoofing/high mitigated by
unchanged actual Firebase/TOTP and actor-only SQL; tampering/high by closed server
previews/current pins; repudiation/medium by immutable reviewed commands/audits;
disclosure/high by current per-target authority and payload-free inaccessible
recovery; denial/medium by count/byte/deadline bounds; elevation/high by no generic
provision EXECUTE, no self-restoration and explicit actual API login. Synthetic
cross-principal/revocation/race/unknown-outcome tests plus independent code review
are required; no organizational security certification is claimed.

Acceptance: fresh existing enrolled actor without workspace through actual
Next/Go/restricted SQL setup/review/prepare/execute/ack; eligible legacy adoption
with original facts/journal anchors intact; no workspace/adoption/journal effects on GET/preview/prepare;
setup-vs-book-vs-label-vs-adoption races and primed stale snapshots; foreign/disabled/
revoked/book/source/consent/generation/complete-set/busy denials; lost prepare,
execute and ack plus remount/exact replay; stale clear cannot close newer claim;
bounded pathological metadata/responses; actual UI navigation/mounted locks and
no browser financial persistence. Run full canonical checks and trusted scanned
Sandbox rollout before owner review. No manual operational SQL in delivery.
