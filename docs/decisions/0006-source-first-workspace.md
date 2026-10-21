# ADR 0006: Source-first account workspace

## Status and question

Proposed for #238 review, 2026-10-21. The owner accepted the product direction:
connect accounts before creating books, import each provider transaction once under
its source account, review all authorized transactions, and organize accounting
into entity books. This ADR proposes the authority boundary. It does not approve
physical SQL, live-account ownership, intercompany policy or automatic posting.
Reviewers: owner for product/consent, identity and database owners for authority,
accounting owner/accountant for any future cross-entity posting treatment.

Question: how can a verified creator connect a Plaid Item containing accounts for
different entities, acquire available history before any book exists, and later
review/label/account for the observations without changing posted history?

## Decision proposed

1. A **private collection workspace** is scoped to a verified, explicitly enrolled
   principal. A Plaid Item, its credential reference, consent state, verified
   account inventory, filtered cursors and acquired observations belong to that
   workspace. Current identity is issuer+subject, never email, browser actor ID or
   Plaid account metadata. Sharing this collection with another principal is
   deferred until an explicit grant and revocation contract is reviewed.
2. Each institution account has a stable local source identity. Plaid account IDs
   and transaction IDs are external identifiers with recorded Item/environment and
   revision provenance; reconnect or ID churn cannot silently reuse an old cursor
   or duplicate posted evidence. Every acquired transaction is recorded once
   under its local source account, including before entity/book setup.
3. The source account's **legal owner** is a separate, explicit, effective-dated
   assertion with reviewer and evidence reference. Plaid type, mask, name and
   `holder_category` do not prove LLC ownership. Unknown ownership is an allowed
   review state, not a reason to invent a holding book. Owner assertion grants
   no source or book access and is not independent verification for a live account.
4. Each book belongs to one entity. A book's ledger account, period, journal,
   posting authority and reports retain that entity/book scope. Reviewed
   dispositions may link one source observation to accounting work in one or more
   books without moving or copying the observation. Once an owning book is
   created and approved, its account must remain reconcilable. Cross-entity
   entries require #244 and #1 policy, separate current permissions and balanced
   entries in each book; until then,
   mixed-entity items stay in an explicit review hold.
5. The workspace has **All Transactions** across its authorized source accounts,
   including unassigned history and provider pending/modified/removed states.
   Provider categories are inert source suggestions. Owner topics and strategies
   are versioned analytic labels. Owner-authored rules may propose topics and
   bookkeeping categories, but neither a provider category, rule nor model can
   draft or post automatically. A reviewer chooses book-specific accounting
   treatment; explicit posting remains the only journal mutation path.
6. Existing book-scoped Items, sources, observations, cursors and posted links
   remain valid and immutable. #239 must specify an additive migration/coexistence
   path and compatibility read behavior, with no journal or source-anchor rewrite.
   Credentials remain opaque references under the existing secret authority.

The detailed proposed authority matrix and acceptance cases are in
[source-first workspace](../contracts/source-first-workspace.md). This ADR does
not replace the currently implemented [connection SQL](../contracts/connection-sql.md)
or [provider sync SQL](../contracts/provider-sync-sql.md) until the new contract and
forward migration have been reviewed, implemented and proved.

## Evidence, interpretation and alternatives

| Source reviewed | Fact from source | Project inference / limit |
| --- | --- | --- |
| Plaid [Transactions API](https://plaid.com/docs/api/products/transactions/), `/transactions/sync` request, pagination, response and readiness sections; public documentation reviewed 2026-10-21 | Sync returns Item changes through a cursor, supports depository/credit/loan accounts, requires pagination restart from the original cursor on a mutation, and can return empty arrays before history is ready. | Preserve complete-set, account-filtered cursor/revision evidence before book mapping; an empty first response cannot mean zero history. Provider documentation does not authorize posting or prove coverage. |
| Plaid [Accounts API](https://plaid.com/docs/api/accounts/), `/accounts/get` and `account_id` description; reviewed 2026-10-21 | One Item lists accounts; `account_id` can change after provider reconciliation failure or token recreation. | Use local source identity and explicit reconnect matching; name/mask/holder metadata cannot establish legal owner. |
| Plaid [Link API](https://plaid.com/docs/api/link/), `/link/token/create` `transactions.days_requested`; reviewed 2026-10-21 | Initial Transactions history request can specify days, default 90 and maximum 730; it cannot be changed after initialization. | Request up to 90 days by default for new Items, retain the configured value, and report actual coverage separately. Existing Items are not silently reset. |
| [PI01–PI13 source record](../research/plaid-ingestion-sources.md) | Records exact selected sections, prior source review, versions and access limits. | Reuse existing provider decoder/version decisions; no private payload or institution was inspected for this ADR. |
| [Database foundations](../research/database-foundations.md), S02/S03/S07–S12 | Imported bank facts and book adjustments are distinct; normalized design and current SQL authority require reviewed keys, constraints and isolation. | Keep a collection layer within the existing PostgreSQL database; #239 owns physical columns, grants and lock order. |
| [Current connection](../contracts/connection-sql.md) and [sync](../contracts/provider-sync-sql.md) contracts | The implemented Item/source/stream scope is one entity/book and current grants are checked on replay. | This is a material compatibility change. A frontend reorder cannot deliver it. |

Alternatives considered:

| Alternative | Why it is insufficient for the accepted journey |
| --- | --- |
| Create an automatic holding book before Link | Invents an accounting entity and produces misleading balances/permissions; later moves conflict with immutable posted history. |
| Bind each Item to one book and ask the user to reconnect for every LLC | Preserves current code but cannot organize multiple legal owners from one Item, and duplicates consent/reconnect work. |
| Keep only account metadata before book setup | Enables connect-first screens but does not deliver the accepted automatic history and All Transactions review before assignment. |
| Copy transactions into each target book | Loses one source identity and invites duplicate review/posting; cross-entity treatment must be linked, not copied. |

## Open policy and implementation boundaries

- #239 decides physical schema, restricted functions, retention/deletion and
  migration proof. No schema-per-LLC, extra database or event store is implied.
- #240 decides Link/reconnect and acquisition operation details under the reviewed
  contract. Initial 90-day request is a target, not guaranteed coverage.
- #241 decides legal owner attestation/review and book setup; assertion evidence
  required for live use remains unresolved.
- #242/#243 decide global query, taxonomy, private rule evaluation and model
  quality/privacy gates. A future model needs a separate reviewed data-use decision.
- #244 and #1 decide intercompany/owner accounting treatment. No posting route is
  authorized by this ADR. Reversal/correction remains explicit and posted journals
  remain immutable.

## Acceptance before promotion

Review the matrix and synthetic fixtures in the linked contract. Require
`TestConnectWithoutBook`, `TestOneItemTwoLegalOwners`,
`TestUnassignedHistoryNoJournal`, `TestCrossPrincipalDeny`,
`TestItemRevokeFencesAllAccounts`, `TestAccountRevokeKeepsSibling`,
`TestReconnectCursorIdentity`, `TestProviderRevisionPreservesPostedAnchor`,
`TestExistingBoundMigrationNoRewrite`, `TestTopicNeverPosts`, and
`TestCrossEntityHeldWithoutPolicy`. These are **planned names**, not passing tests.
Owner/identity/database review of this decision is the gate for #239; the existing
synthetic posting proof does not establish live bookkeeping approval.
