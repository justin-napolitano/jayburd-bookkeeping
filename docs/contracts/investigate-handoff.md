# Investigate: owned status and handoff proposal

## Decision record

**Question and scope:** Issue #255 asks for one focused place to resume unresolved
bookkeeping work. This document proposes a read and handoff boundary, not a new
queue, posting authority, navigation position or aggregate status API. **Status:**
proposed for owner/coordinator review, 2026-10-21. The existing [command recovery](command-recovery.md),
[source review](source-review-sql.md), [connection status](connection-status-ui.md)
and [reporting HTTP](reporting-http.md) contracts own their respective facts.
The selected entity/book must come from verified, currently authorized scope.

The alternative is a single cross-source `attention_count` assembled in the
browser. Reject it for the first slice: source review requires a source selector
and pagination, command recovery is principal-wide, connection status is read
per stream, and report failures are transient. Adding those values would silently
change their scopes and would make a missing source look like zero. A future
server-owned index needs a separate reviewed API/SQL contract and proof that each
source is complete, current and authorized.

## Source inventory

| Work kind | Authoritative read and scope | Action owner and resolved state | First-slice treatment / gap |
| --- | --- | --- | --- |
| Source review | `GET /v1/entities/{e}/books/{b}/sources`, then `GET .../review-items?source_account_id=...&state=...&limit=...&cursor=...`. Source and review permissions are checked on each read. A review page pins `source_revision`; `needs_review`, `overlap_hold`, `draft_ready`, `correction_hold` and `posted` are distinct. | Existing review detail/category/draft and recoverable `source_post` flow. Only explicit reviewed command execution posts; `posted` has immutable journal evidence. Holds are not category-ready work. | Show a chosen, authorized source's unresolved rows, with its own filter/revision and pagination. Source-list overflow, a failed page, or unread sources prevent any book-wide empty claim. `correction_hold` requires the current source-review implementation contract and must not be mapped to an ordinary draft action. |
| Unacknowledged command | `GET /v1/commands/pending` is principal-wide and reauthorizes operation-specific scope. `null` means no command on that principal gate; a returned command may target another book. | Existing execute/cancel/acknowledge; terminal `committed` or `cancelled` remains pending until explicit acknowledgement. Inaccessible work uses payload-free `/pending/clear` only under its contract. | Show only a command whose returned target matches the selected book and whose current payload is authorized. A command for another book is not this book’s item; its principal-wide gate may still block new work. Do not expose its body through a cross-book summary. |
| Enrollment attempt | `GET .../connection-attempts?limit=100&before=...` is actor/book-scoped and gives server-owned `unfinished_count` for attempts. Exact attempt detail rechecks authority. Expired unclaimed, claimed uncertainty and expired binding review differ. | Connections owns completion, inspection, reconciliation, binding and guarded abandonment. Active/disconnected/abandoned or server-qualified expiry changes the attempt state; a UI read cannot infer a provider result. | Display its own status and count only if the optional connection service is composed and discovery completes. Do not call `unfinished_count` an Investigate total. A link-ready attempt is not a connected account. |
| Stream acquisition | `GET .../connections` selects an authorized stream; `GET .../connections/streams/{stream}/status` reads one exact stream. Requested/completed generations, readiness, local publication time, queue/lease and fixed `last_error` are distinct. `provider_updated_at` can be null. | Connections owns pause/resume/request, with current pins, confirmation and exact-key uncertain-result recovery. A request only queues intent; complete publication is not a posted balance. | A selected stream can be investigated. No book-wide health count or generic retry is derivable from one-stream reads. `last_error` null does not mean complete history. |
| Report/read failure | The corresponding scoped report or pinned journal GET owns revision, authorization and error. No durable report-error list is defined. | Retry the same scoped read after a 503, refresh/reselect after 409, or leave the view on denial. Reports and Ledger retain their own direct tool access. | Contextual handoff only while the failed operation is in memory. Do not manufacture a persistent issue, count or resolved record. |
| All Transactions | Issue #242 remains open and blocked by #238/#239 plus a scoped API contract. | No owner taxonomy or cross-book queue action is approved here. | No cross-book or unassigned history in the first Investigate slice. |

The current UI does not receive a reliable book-wide status snapshot. In
particular, `No known items` is truthful only for a named, fully read scope (for
example, the selected source and filter after its page chain reaches the end).
It is not a default screen heading or a synonym for successful Position loading.

## Handoff and presentation boundary

Position or Ledger may pass an **in-memory intent**: canonical selected entity and
book IDs, observation date, optional book/report revision, optional account ID or
source ID, and the originating tool. A display label, monetary amount or badge
from the caller is never status authority. Investigate validates identifiers and
reads its owning source after entry; it may discard an unsupported hint. A stale
pin triggers a fresh scoped read and an explicit `Changed since you opened it`
message, never an action against the old pin. Returning to the origin restores its
selected book/date/selection where still authorized, then that tool refreshes its
own evidence. No financial request, token or intent enters URL, browser storage
or telemetry.

Investigate uses one source and one task at a time. The user chooses **Review
source**, **Recover command**, or **Check connection**; contextual report errors
stay with the originating tool. These are candidate tasks, not a required top
navigation arrangement. Ledger and Reports remain directly available as focused
tools. The selected book/date/currency and source freshness remain visible while
working; amounts stay exact decimal strings with source currency and no client FX.

Each read surface has four separate outcomes: `Still loading` before an answer;
`No known items` only after a complete authorized read for that named scope;
`Status unavailable` for absent optional service, transport failure, truncated
or malformed/incomplete data; and `Permission denied` when current authorization
fails. A stale 409 is `Changed since you opened it` and requires refresh before
action. A 404 that can hide another principal's resource never reveals whether
that resource exists. Late responses from an old book/source selection are
discarded. An item without a currently supported action remains inspectable
with its owner and reason; it gets no speculative button.

## Standards, sources and limits

This is an internal status/navigation contract, not a bank financial exchange or
an ISO 20022 message. Existing Plaid exchange mappings and ISO 4217 currency
version/source-date validation stay with their owners. No new monetary field,
audit export, ISO message version or accounting policy is selected. ISO 21378:2019
and ISO 21926:2026 are relevant when audit export/semantic mapping is designed;
their public abstracts/previews do not establish this screen's field-level
conformance. See [standards](standards.md) and the [source review](../research/investigate-sources.md).

Facts above are from the reviewed local contracts and route implementation.
Inference: composition must preserve those scopes and cannot derive a global
empty/ready conclusion. Remaining decisions: whether a server-owned index is
worth adding, its completeness/freshness contract, and owner review of the exact
task vocabulary. #104 owns connection behavior; #242 owns cross-book transaction
taxonomy; #248 owns Position snapshot semantics. None is completed by this design.

## Acceptance evidence for an implementation issue

Use fixed synthetic examples in the [runbook](../runbooks/investigate-contract.md).
Named browser cases should prove one actionable source item, empty selected
source, unavailable connection runtime, denied book, stale source/pin, cross-book
command and handoff/back navigation at 390px and keyboard width. API/SQL tests
must prove current permissions, pagination/completeness and no leaked payload.
For any changed browser/API boundary run `make web-check web-test web-build
web-browser app-integration` plus `make check test build`; for migrations or
ledger mutations add their required database gates. This proposal itself does
not satisfy those implementation tests.
