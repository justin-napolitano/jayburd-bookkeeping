# Source-first workspace contract proposal

Status: proposed under #238, 2026-10-21. [ADR 0006](../decisions/0006-source-first-workspace.md)
owns the decision and evidence. This document defines authority and acceptance
for later scoped SQL/API contracts; it is not an implemented endpoint, migration,
permission grant or posting policy.

## Identity and record grains

| Concept | Grain and authority | Financial effect |
| --- | --- | --- |
| Verified principal | Exact trusted issuer+subject enrolled by an operator; current owner-private workspace access | Can begin private collection only through an explicit permission; email, caller actor ID and Plaid metadata grant nothing. |
| Plaid Item/consent | One provider/environment/Item identity with one credential reference and lifecycle generation, owned by the private collection workspace | Authorizes provider acquisition only while current principal/consent/service authority holds; one Item may expose accounts of different legal owners. |
| Source account | Stable local identity for one institution account and its provider aliases/reconnect evidence | Holds source transactions and cursor lineage; legal owner is separately asserted, never inferred from name/mask/type. |
| Source observation | One provider transaction identity with immutable acquired revisions, original amount/currency/dates/status and source account | Remains private evidence; pending, modified or removed provider facts do not mutate a posted journal. A new identity after reconnect is held until overlap review. |
| Legal owner assertion | Effective-dated claim for a source account, reviewer, evidence reference and revision | Informs reconciliation proposals; grants no source, entity or book access. Does not post or certify live ownership. Conflicts/overlaps hold review. |
| Topic/strategy label | Versioned owner taxonomy and audited observation label | Analytic only; can span entity views authorized to the principal; changes do not move a book or journal. |
| Suggestion | Provider category or owner rule output with origin/version/confidence or abstention and reviewer decision | Inert until reviewed; no automatic draft, account mapping or posting. Future models have a separate privacy and quality gate. |
| Book disposition | Reviewed link from source observation revision to one or more proposed book treatments with current grants, currency and policy pins | May create a draft only under an approved scoped accounting contract; cross-entity cases hold until #244/#1. |
| Posted journal/source link | Existing balanced, immutable entity/book journal and provenance | Only explicit authorized posting; corrections by linked reversal/new entries. |

No local user account is auto-created from Plaid, no source account is an entity,
and no book is a folder that can change legal entity after posting. A source account
with unknown legal owner remains visible to its authorized collector but cannot be
used to present an entity cash balance.

## Scope and operation matrix

| Operation | Required current authority and behavior | Denial/replay rule |
| --- | --- | --- |
| Begin/complete Link | Verified enrolled principal with explicit private collection permission; durable attempt, one-shot exchange and secret reference | Expired/revoked identity or consent denies. Unknown remote result remains unresolved; retry cannot exchange twice. |
| Acquire/list source transactions | Current owner collection permission, active Item consent, scoped worker identity and account capability; bounded complete-set publication | Account revocation stops that stream; Item revoke/rotate fences all siblings. Replay rechecks current authority and generation. No book or journal write. |
| All Transactions | Verified principal with current access to each returned source account; server-scoped query and snapshot/cursor pagination | Never use caller-supplied scope as authority. Revocation removes rows on next request; stale cursors fail. No unbounded cross-workspace scan or browser persistence. |
| Assert legal owner | Verified principal authorized for both collection and target entity setup/review; reviewed evidence reference and effective date | Conflicting or overlapping claims hold; no ownership derived from provider fields. Later changes preserve assertion history. |
| Label/accept suggestion | Current source-review permission; exact observation and taxonomy/rule revisions | Immutable decision history; label edit never changes books, posted journals or source revision. |
| Create/select book disposition | Current source read/review plus current target entity/book prepare rights, policy/currency/period checks | Target-denied or stale pins reject. Cross-entity treatment needs independent current rights in each book and #244 policy. |
| Explicit post/correct | Existing book-scoped post/correction authority and reviewed immutable draft/command | No automatic post from acquisition, topic, rule or provider category. Posted journal remains in its original entity/book. |

The new private collection permission is a proposed capability, not a self-grant.
#239 must define restricted SQL functions and grants, exact lock/retry order,
retention and deletion. The browser/API may render only safe projections; credentials,
provider payloads and financial requests never enter browser persistence or logs.

## Source acquisition and migration invariants

- Request up to 90 days on new Transactions initialization, store the requested
  window and expose actual history readiness/coverage separately. An empty early
  sync page is not complete history. Preserve original units with ISO 4217 code
  and applicable dated currency reference; never use floating point for money.
- Use provider/environment/Item/account/transaction plus local identity and
  revision provenance to deduplicate within a provider identity. A changed
  account ID, Item or filter cannot inherit a cursor or silently identify an
  older transaction. New IDs stay distinct and held until explicit overlap
  review. Page mutation restarts the complete filtered pagination loop from its
  original cursor; publication is atomic and replay-safe.
- Current book-scoped Items and posted source anchors remain accessible through
  compatibility reads. Adoption of the same provider/environment/Item identity
  uses one audited authority transition, not a second Item row. The forward
  migration does not reparent, delete or silently copy existing observations,
  journals or source links. Before adoption succeeds, legacy streams keep their
  current behavior; any newly linked Item requires overlap detection and current
  permission checks before combining its history with legacy evidence.
- An account disconnect is a local stream fence; it does not call Plaid Item
  removal or imply provider-side consent revocation. Item removal, rotation and
  consent revocation fence all sibling streams. Credential cleanup and retained
  evidence follow a reviewed retention policy, not a UI assumption.
- A missing dated ISO 4217 reference or unofficial/unsupported currency cannot
  be silently converted or omitted while advancing a cursor. Publish the whole
  acquired change set into private evidence or quarantine with a fixed hold
  reason; mark affected observations visible as held in All Transactions, with
  original units only where validated. They cannot enter book review, draft or
  journal until currency mapping is reviewed. #239 must define atomic cursor/
  hold storage and bounded retention without weakening existing checks.

## Global review and classification boundaries

All Transactions must include unassigned, reviewable, held, draft and posted
observations from every source the principal may currently read. Display source
account, original amount/currency/date, provider pending/modified/removed/held status,
acquisition freshness, legal-owner assertion, topic labels, suggestion provenance,
book disposition and journal status as separate fields. Support bounded server
filters and stable pagination; no empty-state claim of complete history before
provider history readiness is known.

Plaid `personal_finance_category`, merchant and enrichment are untrusted source
metadata. An owner topic can be attached to many observations, and an observation
can have multiple analytic labels. An owner rule may propose a topic or ledger
category, with version/confidence or abstention, but acceptance is a distinct
audited decision. Bookkeeping category is the selected book's ledger account and
policy, not a topic or Plaid category. Any split/allocation or FX treatment needs
its own reviewed accounting contract.

## Standards applicability and planned evidence

Plaid Link/Accounts/Transactions JSON is a proprietary provider exchange. The
proposed ISO 20022 assessment is `not applicable` to this Plaid wire profile;
#240 must record the exact purpose, reviewed registered definitions and final
applicability/mapping before implementation. Do not invent a `camt` designation.
Internal collection commands/events require their own applicability entry,
version and semantic mapping. ISO 4217 currency code and
dated minor-unit reference apply to every monetary observation. ISO 21378 audit
export applicability remains unresolved at field level because only public
abstract/metadata were reviewed. See [standards](standards.md); no conformance
claim follows from a label or XSD.

Planned synthetic evidence, independent of implementation: connection without a
book; one Item/two legal account owners; 90-day request with shorter available
history; empty early response; account and Item revocation; lost exchange and
unknown commit; account-ID churn; pending→posted/modified/removed transaction;
existing bound Item migration; duplicate source suppression; USD/JPY/KWD original
unit preservation; cross-principal denial; global pagination across 3,000 items;
topic/rule override without journal mutation; cross-entity expense held with no
journal before policy; immutable correction after posting. #244 and #1 later own
policy-approved coordinated-entry fixtures. #239–#244 own executable tests and
reviewed results. STD-01–STD-05 in
the standards contract remain required for the applicable implemented slice.
