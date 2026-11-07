# Collection filters, source detail and private rule UI

## Read profile checkpoint — migration 45

The reviewed v3 catalog/query/detail contract is additive to existing v2 manual label commands. The seven-pin snapshot is unchanged. The query selector includes exact echoed merchant, original amount/currency, source/journal status, book, institution/legal-owner state and sort objects. All matching, counts, ordering and cursor continuation remain server-owned. The frontend keeps private selectors in bounded POST bodies and uses no financial browser persistence.

Merchant literal filters use case-insensitive matching, distinct from the rule evaluator's case-sensitive conditions. Amount bounds are nonnegative exact decimal strings, optionally inclusive minimum/maximum, within one selected dated currency. Feedback validates lexical bounds, nonzero fractional precision and ordered bounds with integer arithmetic; it never rounds or converts. Amount sorting groups original currency code/version before magnitude, with unavailable money last. No cross-currency FX comparison or financial total is fabricated. Institution remains unavailable and legal owner unknown until separately evidenced metadata exists; neither is inferred from account names or book entities.

Source detail pins the complete existing snapshot and composite source observation identity/revision. It echoes a current v3 row and distinct available/unassigned/unavailable book links. Available links may be empty for a readable unposted legacy source; unavailable never means unassigned. Every journal anchor is unique and canonical, belongs to an assigned readable book, and carries its actual positive historical source revision. The frontend rejects mismatched pins or forbidden anchors before rendering.

Clicking a row opens focused detail without replacing the list's server page or explicit selection. Back restores the row's keyboard focus and visible position. Explicit journal anchors use an injected existing authenticated book-journal read; its exact entity/book/journal identity is rechecked and original/functional decimal strings remain separate. No posting, reversal or draft controls are introduced. Denial destroys detail/page state; superseded responses cannot repopulate another scope. Read failures offer retry or Back without inventing empty results.

Reviewed evidence: current collection-labels/tags-recovery/live-label contracts, bounded Go evaluator source, compact-detail focus precedent, standards/decision-evidence requirements and actual existing book journal route. Internal filtering/rule management is not an ISO 20022 exchange or ISO 21378 export. Existing dated ISO 4217 references and original decimal semantics are preserved; no recognition/valuation or ISO conformance claim is added. Browser matching/direct writes were rejected because they duplicate authoritative logic or bypass recoverable review.

## Rule profile

Coordinator and SQL lane have frozen the next v3 rule catalog/actions/results. Implementation and named evidence follow in the next checkpoint; this document does not claim that the read-only checkpoint implements rules or managed activation. Current GitHub #242/#414/#65 owns execution and acceptance.
