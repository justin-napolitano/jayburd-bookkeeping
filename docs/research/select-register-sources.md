# Select-and-register source review

Reviewed2026-11-02 for391–395. Public official documentation, selected sections
only; no actual Item payload, institution configuration or financial record read.
Planning scope only; current hosted behavior must be independently qualified.

| Record | Exact reviewed source/sections | Facts and project interpretation |
| --- | --- | --- |
| SR01 | [Plaid Accounts API](https://plaid.com/docs/api/accounts/), accounts/get introduction; account name/official_name/mask/type/subtype and currency fields | Cached active accounts associated with an Item; newly unshared accounts require update mode. Name is user/institution assigned; official_name/mask nullable; mask2–4 characters and can collide. Interpret as minimal private display projection, never identity or accounting authority. No balances needed in picker |
| SR02 | [Plaid update mode](https://plaid.com/docs/link/update-mode/), Using update mode, Using update mode to request new accounts | Existing Item token unchanged/no repeated public-token exchange; account selection explicitly enabled; selected accounts shared and deselected accounts cease being shared. US scope here; UK/EU limitation is not addressed by new Item creation in this slice. Interpret already-shared local registration separately from new bank consent; reconcile current permissions without erasing immutable posted provenance |
| SR03 | [Hosted Link](https://plaid.com/docs/link/hosted-link/), Overview, Obtaining a public token, completion redirect | Hosted flow uses verified server completion evidence from SESSION_FINISHED/link-token-get, not frontend success callback; six-hour detail availability. Official guide prefers SDK when feasible. Keep current owned Hosted integration for this bounded pass, with bounded refresh/recovery; full SDK replacement is a separate decision, not necessary for a checkbox picker |
| SR04 | PostgreSQL17 [transaction isolation](https://www.postgresql.org/docs/17/transaction-iso.html), Serializable Isolation Level; [function privileges](https://www.postgresql.org/docs/17/perm-functions.html), registered S08/S12 | Reused selected review from database-foundations/database-sources; no new full-page review. Locks/serializable retries/restricted entrypoints are mechanisms; project must prove capacity/append/current-authority invariants with synthetic tests |

Repository evidence: migration34 bind_connection known receipt precedes the
active-attempt prohibition, literal Production one-binding guard and current
manifest/currency/account validation; begin_connection separately limits Items.
Existing automatic-account-binding/production-source-sql/command-recovery
contracts establish atomic creation, dated currency, permissions and durable
review/execute/acknowledge. These are reviewed implementation facts, not a claim
the proposed append/plan functions already exist.

Alternatives: removing all limits is rejected (unreviewed authority expansion);
configuration-only change is insufficient (active bind is terminal); relinking
the bank is rejected as ordinary local registration (duplicate Items/history risk);
an explicit additive plan/append command is proposed. Keep advanced mapping while
removing technical fields from normal workflow.390's Sandbox callback/count proof
and the first Production import do not prove these future registration semantics.

Access limits: public web text only; no full normative ISO text reviewed, SDK
vendor promise not a bank coverage guarantee. No inference that Plaid names/masks
are unique, all accounts are shared, or90 days guaranteed. See plan for proposed
named validation and remaining physical-contract review.
