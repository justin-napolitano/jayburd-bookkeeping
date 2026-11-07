# Private labels browser composition —305

## Decision and sources

Accepted source implementation under owner238/242 and coordinator
[frozen304/305 protocol](collection-labels.md). This appendix replaces the
isolated-only activation boundary of [the original component](collection-review-ui.md)
for this delivery branch. Current task/approval state remains in GitHub304/305/65;
managed feature activation and live-data use are outside this source change.

Reviewed local source sections: frozen wire/concurrency/recovery in
collection-labels.md; Authority/identities/taxonomy in
[private categorization](private-categorization.md); State/authority, HTTP and
Browser/acceptance in [command recovery](command-recovery.md); injected API and
Bookkeeping lifecycle/recovery implementation; original collection UI guards.
These establish the project's current authority/command contract. The inference
is to compose the existing verified transport and recovery UI, preserving SQL
authority rather than inventing a browser-owned catalog or command store.

Internal taxonomy/query commands are not an ISO20022 financial exchange.
ISO4217 dated identifiers, original decimal strings and minor units remain
unchanged. No ledger calculation, FX, audit export or conformance claim is added;
the existing [standards applicability](standards.md) review/access limits apply.

## Discovery and authenticated port

Always mount CollectionWorkspace beside the book recovery coordinator. Discover
GET/v1/collection without actor/email/provisioning input; map workspace_revision
to authority_revision and use the fixed display name Private transactions.
Explicit discovery404 means unavailable/unauthorized; other failed/malformed
discovery locks new work until explicit retry. Taxonomy/catalog discovery must
echo workspace/authority and provide complete bounded sources/topics and current
taxonomy/ruleset/map editions. No default topic seed is fabricated.

CollectionPort uses only the injected existing API function/query method, which
owns fresh memory-only authentication and safe no-store same-origin transport.
The frozen query/preview/prepare/pending/transition routes and body caps are
preserved; no Idempotency-Key is added to private commands. Path IDs are encoded,
private filters stay in POST bodies, and error messages discard raw diagnostics.
Only pending permits a null command in its matching profile/workspace wrapper.
Current metadata alone cannot grant private classification; read-only capability
keeps assignment/topic editing disabled and visibly read-only.

Snapshot includes authority/source/taxonomy/ruleset/map/decision pins. First-page
responses must match current catalog taxonomy/ruleset/map; subsequent replies
also match the entire opaque snapshot and exact selector. A selected account
pins every returned row. Original held money and unavailable book projections
retain their distinct states. Unicode topic bounds count codepoints, not UTF16
units, so80 astral characters remain valid evidence rather than being truncated.
Opaque continuation supports4096 characters separately from2048-character IDs;
the full cursor and100-codepoint Unicode search remain intact in the8KiB query.

## One review and recovery engine

Label actions still contain1–100 explicit composite observations/revisions.
Taxonomy actions contain a complete edition, empty selection/review_rows and
selected_count0. Topic IDs remain stable on rename/reparent/retire; all old IDs
are retained. Existing retired topics are read-only history and cannot receive
new assignments. Client validation mirrors finite/depth/cycle/parent/name bounds
for useful feedback; authoritative SQL validation still owns acceptance.

SQL canonicalizes topic arrays by topic_id. Request/preview comparison treats a
taxonomy array as an identity-keyed set with exact name/parent/status values;
the accepted server preview is then immutable in its actual stored order on
prepare/recovery/transitions. Changed saved fields or opposite known terminal
states are rejected. No edit bypasses inert preview, durable prepare, explicit
confirm/cancel and explicit terminal acknowledgement. No discovery executes or
acknowledges a command.

A first definite rejected prepare clears only unsaved intent and immediately
discovers current pending work under a continuously closed shared gate. This can
restore another tab's command or permit an explicit new preview after pending:null.
A previously unknown prepare retains its original key/preview even if a retry is
definitively rejected. Exact acknowledged terminal prepare/execute/cancel replay
receipts from another context are displayed; local explicit Acknowledge remains
required and sends only the same identity. Null alone never resolves an uncertain
mutation. Later denial destroys private payloads but keeps book work locked until
explicit current discovery/recovery succeeds; no private clear endpoint is invented.

Recovery permits historic immutable edition pins after current taxonomy/source
metadata changes; current service authorization remains mandatory before any
payload is disclosed. In particular a committed taxonomy command can be restored
after it advanced the current edition. Refresh catalog after explicit ack; never
replace saved intent with new current catalog values. Explicit top-level Refresh
also reloads catalog and authority for stale, unsaved reads.

After a confirmed explicit private acknowledgement, the coordinator opens All
transactions before reloading its catalog/pending state. This preserves the
private task after a force-visible recovery was restored over the default
Overview. Initial idle/pending discovery, execution and unresolved replies do not
change navigation; a failed acknowledgement keeps the current view until an
explicit same-identity retry confirms its terminal receipt.

The topic-choice select references a stable React-generated text-span ID through
aria-labelledby, while retaining its wrapping native label and layout. Its exact
accessible name stays Label selected transactions after option changes; option
text cannot become part of the label lookup. The real journey's exact-label
assertion remains unchanged, and component/browser checks use the same exact name.

Collection pending/unknown/preview work gates book actions and navigation even
when All transactions is not active. Book recovery independently gates collection
mutation. The parent passes only the book lock into the collection session; its
own reported lock cannot re-key/destroy its uncertain request/prepare identity.
Inactive idle collection views clear retained rows and skip row queries while
continuing pending discovery. Sign-out clears memory; persisted server discovery
supports later recovery. No financial data/key/token enters browser persistence.

## Reciprocal command discovery

The shared principal reservation is an authoritative SQL boundary coordinated
upstream; this browser does not assume that its initial two pending reads remain
current across other tabs. A first definite private prepare409 checks both its
own pending endpoint and idle book recovery. A first definite book registration409
checks its own endpoint and requests a private recovery refresh. The shared gate
closes synchronously before reciprocal refresh and remains closed until both
current reads settle. Each endpoint retains its own payload/identity; no private
command is returned through book recovery or vice versa. Neither read executes,
cancels or acknowledges work.

The private coordinator defers refresh while its current session is busy, reviewed,
pending or uncertain. Book idle-only refresh skips active commands and uncertain
registrations. Book workflow locks clear private rows/selection and disable
actions, but no longer key/unmount or automatically rediscover a private session:
this explicitly supersedes the isolated component's old lock-unmount behavior.
Authority/workspace/taxonomy changes still destroy that scope-owned session.

A previously uncertain book registration keeps the exact key/body/path after a
later definite409 and never triggers the first-rejection callback. After that
rejection, book pending cannot prove a command belongs to the retained key;
only explicit exact-key registration retry resolves it. Existing initial ambiguous
reply discovery still restores the server's current pending intent for explicit
review; it is not proof that the original candidate executed. No ledger key-echo
wire change or broader recovery policy is introduced. Idle-only
reciprocal refresh cannot replace a known active command. These controls preserve
intent while SQL reservation/replay proves the actual cross-context no-effect
boundary; browser mocks alone do not prove database mutual exclusion.

## Evidence and alternatives

Client unit tests name exact operations/bodies/wrappers, feature-off/denied/stale/
uncertain failures, complete bounded catalog and read-only classification.
Coordinator unit tests cover inactive pending discovery, failed-discovery lock,
explicit retry, no inactive row fetch and the external book gate. Taxonomy tests
cover exact full-edition intent, stable IDs, parent/retire validation, lost prepare,
historic committed recovery, retired history, Unicode and map/decision pins.

Browser collection-live checks use the actual Bookkeeping/client composition
with explicitly synthetic HTTP fixtures: no-book navigation,3000 count/paging,
exact selected observation, lost prepare, explicit execute/ack, remounted historic
taxonomy recovery, book/private uncertainty and390px empty-taxonomy editing.
Original390px default/larger-font fold, keyboard and control-size checks remain.
These fixtures prove browser transport/composition, not SQL permission/replay or
live feature availability; the coordinated restricted HTTP/SQL/app proof remains
required before complete304/305 acceptance.

Alternatives rejected: bootstrap only on navigation (misses pending private work),
feeding the combined parent lock back into the session key (destroys uncertain
intent), treating failed discovery as empty state (enables unconfirmed work),
requiring current taxonomy pins for restored commands (strands their own commits),
direct taxonomy mutation (bypasses reviewed recovery), dropping retired rows to
fit a catalog (deletes history), and browser-persisted recovery (exposes data).
