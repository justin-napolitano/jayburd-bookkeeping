# Strategic tags and payload-free recovery SQL —450/451

## Frozen scope and evidence

Root froze TAGREC-20261107T0340Z before44 DDL after explicit owner authorization.
GitHub450/451/65 own approval/task state. Existing1–43 source is immutable;
44 adds separate strategic tags and a payload-free inaccessible-private-command
escape path. Root owns shared transport/types; this appendix owns SQL semantics.
Read AGENTS, index/bibliography, decision evidence/database foundations, standards,
accepted private categorization,43 physical/recovery and command contracts.

Reviewed PostgreSQL17 isolation13.2.2–3, row locking/deadlocks13.3.2–4,
INSERT ON CONFLICT/RETURNING and FK/unique constraints in the preceding source
review; relevant full sections establish mechanisms, not our tag/authority policy.
Sources: [isolation](https://www.postgresql.org/docs/17/transaction-iso.html),
[locks](https://www.postgresql.org/docs/17/explicit-locking.html),
[INSERT](https://www.postgresql.org/docs/17/sql-insert.html),
[constraints](https://www.postgresql.org/docs/17/ddl-constraints.html).
Internal analytic tags/recovery are not ISO20022 exchanges or ISO taxonomies.
ISO4217 dated unit identities remain unchanged; no arithmetic/FX/export or ISO21378
conformance introduced. Retain existing standards access limits and source policy.

Alternative extending v1 response objects rejected: closed old readers must keep
working. V2 makes new pins/projections explicit; no invented historic tag pins.
Alternative granting access from command ownership rejected: recovery can release
only inaccessible intent and cannot expose its payload or reinstate grants.

## Tag catalog and decisions

Separate workspace-scoped stable tag UUIDs; complete immutable editions, initial
revision1 empty. Full replacement retains every old ID; retirement is terminal.
At most500 active/1000total, nonblank name1–80 Unicode codepoints/no controls,
case-sensitive duplicate names forbidden including retired entries. No hierarchy.
Precreated label state gains tag_revision; initializer/backfill prevents empty-state
primed snapshots. Catalog edits advance tag_revision; tag decisions advance the
existing decision_revision once per atomic selection. Old v1 source digest also
binds tag_revision without changing its six-key snapshot shape.

New action {kind:tag_catalog,tags:[{tag_id,name,status}]} has empty selection.
New {kind:tags,operation:add|remove|replace,tag_ids:[UUID]} has1–100 exact current
composite observation/revision selections. IDs unique/canonical sorted; add/remove
1–25 IDs, replace0–25, each resulting set at most25. Add is union, remove difference,
replace exact set; never changes the primary topic. Add/replace require active
catalog IDs. Remove may reference retired retained IDs. Foreign/unknown IDs denied.
A prior tag decision pinned to an older source revision is stale; add/remove refuse
it and require explicit replace to revalidate. No automatic copying onto corrected
or different observation identities. Every accepted command appends immutable full
resulting sets and original source/catalog/command provenance; no source/journal DML.

## Versioned wire and operations

Six banking functions with suffix _v2 keep43 signatures: get_collection_taxonomy,
query_collection_review,preview_collection_labels,prepare_collection_label_command,
pending_collection_label_command,transition_collection_label_command. Matching
private-collection-{catalog,review,preview,command}-v2 profiles. New catalog adds
tag_revision and complete tags. V2 snapshot adds tag_revision to the six old pins.
Query requires orthogonal tag filter {kind:all|untagged} or {kind:tag,tag_id}.
Tag filtering never duplicates observations/counts; untagged includes absent/stale/
current-empty sets, tag matches only current-source-pinned sets.

Row v2 is unchanged base v1 row plus owner_tags:{state:unclassified|current|stale,
tags:[{tag_id,name,status}]}. Stale projections retain prior set IDs with current
catalog names/status. Preview v2 always contains tag_results (empty for non-tags).
Entries {workspace_id,source_id,observation_id,observation_revision,tags:[{tag_id,
name,status}]} contain authoritative resulting sets, sorted source/observation UUID
then tag UUID. Exact selected rows/action/results remain immutable through recovery.

V2 command wrappers may contain an unchanged historic v1 preview or a v2 preview.
Historic v1 keeps six pins/no tag_results and unchanged rows; never manufacture
missing tag metadata. V1 outputs stay closed; its command operations encountering
v2 intent fail fixed P2804 before disclosing unsupported payload. Current authority,
shared principal reservation, whole-transaction retries and explicit terminal ack
remain43's boundaries. Query8192/preview524288/prepare4096/transition1024bytes;
response1MiB, cursor4096hexchars, current private/book locks and source holds reused.

## Payload-free private recovery

banking.collection_recovery_status(actor uuid), SERIALIZABLE, returns exactly
{profile:private-collection-recovery-v1,state:none|available|inaccessible,
recovery_id:UUID|null}. Current verified active API principal is checked before
lookup. Only its current private shared reservation is considered; book claims
return none under their existing separate recovery. No workspace/engine/command
name, financial quantity, source description, topic/tag, preview or old payload.
A persistent perprincipal recovery_revision row is precreated/backfilled and added
on future principal insert. Reservation insert/delete touches it; lock it last
under exclusive principal-first order so primed discovery cannot miss new claims.

banking.clear_collection_recovery(actor uuid,raw text), SERIALIZABLE, accepts exact
{recovery_id:UUID}, max1024bytes. Current active verified principal owns that exact
opaque reservation token. Eligibility reuses normal private capability/source/book
reauthorization, catching only its explicit denial states. Stale pins alone do not
make work inaccessible. Current-authorized intent refuses P2802. Prepared work is
cancelled then acknowledged through two guarded updates; terminal work only
acknowledged. The exact shared-reservation release guard remains mandatory.

Response exactly {profile:private-collection-recovery-clear-v1,recovery_id,
cleared:true}. Immutable payload-free receipt keyed principal+token permits exact
lost-reply replay even after a newer claim exists, without touching that new claim.
Disabled/foreign principals stay denied on replay. Existing cancel/ack audit and
immutable decisions/history remain; no grant inferred or restored and no execute.
No automatic clear from discovery or absence/transport failure. Runtime has only
the closed functions, no new helper/table access. New down is empty-only and refuses
retained catalogs/decisions/previews/recovery receipts; no CASCADE or history reset.

## Required independent evidence

Planned: cross-principal/foreign/retired tag cases; name/size/set bounds; add/remove/
replace and stale-source override; catalog/source/decision conflicts; atomic late
bulk rollback; old v1 closed reads and historic preview recovery; both command
engine races and no journal/source changes. Recovery: all workspace/classify/source/
book revocations, restored authority refusal, stale-only ordinary path, disabled/
foreign denial, minimal output keys, prepared/committed/cancelled preservation,
lost-reply replay while a newer claim exists, primed absent/new-claim wait and
exact missing-release retry. Actual SQL/API/browser evidence belongs in runbooks;
these planned names are not passing claims.

Frozen refinements: unknown/foreign tag IDs P2801; retired add/replace P2803;
retired removal allowed; deleted old catalog IDs/resurrection P2802; malformed or
duplicate IDs/names P2800; bounds P2805. Every explicit PRIVATE acknowledgement
release, ordinary or inaccessible clear, writes a payload-free closure receipt.
Therefore unknown clear A followed by ordinary Ack A elsewhere is resolved by
same-ID clear A from durable acknowledgement proof, including when a new private
or book claim exists. No receipt or success is inferred from discovery:none.
Book releases do not create private closure receipts. All catalog/read/preview/
command responses enforce the1MiB bound before returning.

## Bounded selection implementation

V2 preview materializes the authorized selected safe-row cache once, then validates
each canonical composite identity/revision and rejects duplicates before storing
anything. V2 disclosure/execute similarly validate the complete selection against
one current fact inventory under the same locks/snapshot. This avoids full3000-row
rendering100 times without widening selection or moving authority into Go. Named
deadline proofs measure preview/prepare/atomic execute under3500ms statement and4s
context; exact observations/limits are in the SQL runbook. Immutable historic v1
previews keep their original version and content; old1–43 source remains unchanged.
