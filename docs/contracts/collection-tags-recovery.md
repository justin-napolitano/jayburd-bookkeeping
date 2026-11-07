# Private strategic tags and payload-free recovery — 450/451

## Frozen decision

Owner requested all three next tasks on 2026-11-07: inaccessible private command
recovery, strategic tags and synthetic hands-on UI review. Base main is merged
449 / 1a1491a. GitHub65 TAGREC-20261107T0340Z owns scheduling; this contract owns
the profile. Sole forward migration44; merged1–43 remain immutable. No provider,
cloud, hosted activation or private records. Existing explicit feature flag
remains default off. Tags classify private observations, not journal recognition.

## Versioned tags

Preserve the six v1 SQL entrypoints and their closed output shapes. Add matching
`*_v2` entrypoints with the same signatures, and the same eight HTTP suffixes
under `/v2/collections/{workspace}`. Bounds, verified actor, exact JSON, fixed
errors, isolation, deadlines, retries and recovery follow collection-labels.md.
Output profiles are private-collection-{catalog,review,preview,command}-v2.

V2 snapshots add required decimal-string `tag_revision`. V1 snapshot source hashes
fold current tag revision so tag edits invalidate older previews without changing
their field shape. A tag edition starts at1 with an empty catalog. Tags are
{tag_id,name,status}; UUID identities, name at most80 Unicode codepoints, status
active/retired, at most1000 total and500 active. Full replacements retain all IDs;
retirement is terminal. Names are case-sensitive unique across the full catalog.
Tags have no hierarchy and remain independent of primary topics.

V2 query adds required `tag`: {kind:all}, {kind:untagged}, or
{kind:tag,tag_id}. The filter is orthogonal to topic. Untagged includes absent or
stale sets; a tag match uses only a current source-pinned set. Multiple tags never
duplicate source rows, counts, topic totals or journal amounts. Rows add
`owner_tags`:{state:unclassified|current|stale,tags:[{tag_id,name,status}]}.

New actions are {kind:tag_catalog,tags:[...]} for full catalog replacement, and
{kind:tags,operation:add|remove|replace,tag_ids:[UUID...]}. Tag IDs are unique;
add/remove require1–25 IDs, replace0–25; resulting sets have at most25 tags.
Add/replace require active local tags; removal accepts retired local tags.
Prior sets pinned to a changed source reject add/remove and require an explicit
replacement. Full resulting sets are recorded immutably per composite observation;
one bulk commit advances existing decision_revision once, never the primary topic.

Preview adds `tag_results` containing exact authoritative results, sorted by
composite identity and tag UUID: [{workspace_id,source_id,observation_id,
observation_revision,tags:[{tag_id,name,status}]}]. It is empty for non-tag actions.
Tag catalog actions have empty selection/review_rows and selected_count0. Other
tag actions select1–100 exact current source identities. Preview/execute compare
all authority/source/topic/tag/decision pins and preserve the shared reservation.
V2 command recovery accepts an immutable historical v1 or v2 preview union.
V1 encountering a v2 command fails with fixed P2804, never an unsupported payload.

## Payload-free private recovery

Global GET `/v1/collection-recovery` calls banking.collection_recovery_status(actor).
It returns exactly {profile:private-collection-recovery-v1,
state:none|available|inaccessible,recovery_id:UUID|null}. Only an active currently
verified owner may inspect their private shared reservation. No workspace, engine,
command ID, name, source/book metadata, preview or money is returned. A book claim
returns none here; existing book recovery remains independently gated.

POST `/v1/collection-recovery/clear` calls
banking.clear_collection_recovery(actor,raw). Body is exactly {recovery_id:UUID},
at most1024 bytes. Response is exactly
{profile:private-collection-recovery-clear-v1,recovery_id,cleared:true}.
Both operations use SERIALIZABLE, including GET. No query, encoding or header key.
Ordinary current private and original-book disclosure checks determine eligibility;
only expected authority denial qualifies. Stale pins alone do not qualify.
Currently authorized work returns409 and continues through ordinary recovery.

For the exact inaccessible owned private claim, explicit clear cancels prepared
work, then acknowledges it; terminal work is only acknowledged. Decisions,
originals, journals, previews and immutable receipts remain. Existing guarded
updates/audit and exact one-row reservation release remain enforced. An opaque
reservation recovery ID plus durable actor-bound closure receipt makes exact replay
safe after lost replies, including when newer work now owns the reservation.
Every explicit private acknowledgement writes that minimal closure receipt, including ordinary acknowledgement after a lost clear reply. Replay never releases newer work. Disabled/foreign principals remain denied.

A persistent precreated per-principal revision fence, touched on reservation
insert/delete, prevents a primed status snapshot from missing a new claim after
waiting. Lock current active principal first and fence last in compatible order.
Known40001/40P01 aborts retry; unknown commits never automatically retry.

The always-mounted UI discovers opaque recovery before ordinary private discovery
can establish absence. Inaccessible work offers an explicit clear action. An
uncertain clear retains its exact recovery ID and closed gate even if later
status says none; retry the same clear to obtain its receipt. After confirmation,
refresh both engines to discover newer work. Initial feature-off404 may disable
this optional feature; denial/failure after known work never establishes absence.
No financial requests or tokens enter browser storage, URLs or logs.

## Evidence and review

Reviewed 2026-11-07: PostgreSQL17 sections13.2.2–3 on snapshot/retry semantics and
13.3.2–4 on row locks/deadlocks, plus RFC9457 sections3/5 on fixed problem details
and disclosure. Sources: https://www.postgresql.org/docs/17/transaction-iso.html,
https://www.postgresql.org/docs/17/explicit-locking.html,
https://www.rfc-editor.org/rfc/rfc9457.html. Relevant sections, not whole manuals.
Facts: a waiting transaction may need retry to observe current state; diagnostic
details can leak internals. Inference: persistent fences and minimal receipts
provide a bounded recovery path. Existing bibliography links the parent source
review in collection-labels.md; this is its versioned continuation.

ISO applicability follows standards.md: private analytic tags/recovery are not an
ISO20022 financial exchange or ISO21378 export. Original dated ISO4217 units stay
unchanged. No invented conformance, FX or accounting-policy claim.
Alternatives rejected: changing closed v1 shapes, treating missing pending as
clearance, restoring old grants, exposing saved reviews after revocation, or
clearing by principal alone (could release newer work).

STRIDE: high disclosure/elevation mitigated by active verified principal, exact
owned claim, current authority and payload-free outputs; high tampering by exact
pins, immutable tag editions/decisions and atomic reviewed commands; repudiation
by retained audited lifecycle/actor-bound receipts; medium DoS by finite inputs,
deadlines/catalog/selection limits; high stale outcome by receipt replay and
closed uncertain gates. Sensitive data-access/API changes require human/AppSec
review under the applied security skill; automated proof is not that approval.

Named acceptance: actual restricted SQL/HTTP current/foreign/disabled denial,
prepared vs committed clear, authority restored before clear, old receipt/new
claim race, primed status retry, no payload disclosure, tag retirement/foreign
IDs/stale source/catalog pins/atomic sets, source/topic/journal preservation,
unchanged v1 outputs, real browser/API/PostgreSQL lost-reply recovery, and
synthetic mobile/desktop/keyboard inspection. GitHub records actual results.
