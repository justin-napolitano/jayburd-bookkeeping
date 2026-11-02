# Select accounts and register

Proposed delivery plan for [391](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/391),
2026-11-02. Owner direction: finish checking first and simplify the next database
pass to registration plus account selection. This plan does not apply a migration
or increase the current one-account Production mandate. Tests below are planned,
not existing acceptance results. GitHub391–395 and65 own live scheduling.

## User experience

Register accounts → private Plaid flow → select accounts → Register selected →
Importing/Ready → View transactions. Show the current Personal book, recognizable
account name/type and a masked suffix when legitimately available. Already
registered checking stays visibly Registered; a new account is unchecked until
selected. Unsupported accounts show a specific reason and stay unselected.
Do not require ledger codes, currency-version IDs or a binding date in the normal
supported path. Optional Advanced mapping presents the server's proposed ledger
account and permits an explicit reviewed override.

The Register selected action affirms the displayed scope and server-generated
mapping. Persist that reviewed plan before execution. Keep an explicit terminal
Done/View transactions acknowledgement, with plain recovery language when a
response is uncertain. A redirect is a prompt to inspect server-owned completion,
never proof of authentication. Bounded completion refresh/focus and manual retry
must reuse the same attempt; no repeat Item/token exchange after an unknown result.

### Layout and critique

Use a single left-aligned account list, with selection at the left and status at
the right; one primary registration action below. Keep existing visual tokens:
paper#ffffff, ink#202124, muted#555b63, divider#dce0e3, action#286044,
error#9b3e38. Existing system sans stays for this focused pass: 24px section title,
16px account name,14px supporting type/status; visible keyboard focus and ordinary
sentence case. No repeated form cards or raw identifier headings.

```text
Register accounts                         Personal book
Choose the accounts to import from this connection.

[✓] Example Checking ••1234       USD     Registered
[ ] Example Savings  ••5678       USD     Available
    Example Investment                    Not supported: policy needed

Request up to90 days; actual history depends on the bank.
Advanced mapping                          Register selected accounts
```

Registered rows are indicators, not editable checkboxes; the check mark above is
status. On mobile, place status under the account name and keep the action in
normal document flow. Compare alternatives: a wizard hides account comparison;
the current full mapping form overwhelms the choice. The list directly expresses
the owner's task. Keep technical details in a disclosure, and remove repeated
consent/date/currency controls once the server plan owns those decisions.
Synthetic screenshots and keyboard testing must critique the built result before
release; this text is a proposal, not a claim of browser validation.

## Why a migration is required

Migration34's bind function explicitly permits exactly one Production binding;
its active attempt rejects further binds. A separate begin_connection check also
limits non-disconnected Production Items to one. Merely deleting a conditional
would leave append/recovery/current authority and duplicate-source risks unresolved.
Merged migrations34/35 remain immutable.392 claims the next free migration number
only after checking live leases;36 is currently a candidate, not reserved here.

The proposed forward migration must:

1. Add an explicit scoped source maximum, default1 and bounded1–25, to the
   Production authority model. Only the reviewed owner/operator configuration
   command may change it; an app actor cannot increase its own access. Preserve
   the separate one-Item restriction in this slice. Changing the maximum does
   not grant access to an account or renew expired consent.
2. Add a durable registration-plan/receipt model keyed by current verified owner,
   entity/book and owned Item, selected provider IDs, manifest/Item generation,
   authority/profile revisions, expiry and normalized mapping digest. Selection,
   server-resolved mappings and reviewed command history are persisted before
   execution; browser storage is not a recovery mechanism.
3. Permit atomic addition of verified, already-shared accounts on an active Item
   through a new operation. Never reopen the original attempt or rewrite its bind
   receipt. Preserve checking's ledger/source/stream IDs, Item credentials, cursor,
   observations/review history, provenance and journals. Do not reset the Item to
   force another initial sync. The new stream receives its own acquisition demand.
4. Lock scope and Item consistently; count existing plus proposed non-disconnected
   sources including paused ones. Deduplicate provider-account identity across
   historical bindings and reject concurrent capacity/code conflicts atomically.
   Specify disconnected-source rebind/reactivation before implementation: reuse
   immutable source lineage with a new authorized revision, never manufacture a
   fresh history or use disconnect as a source-limit bypass.
5. Execute mappings/account creation/grants/streams/demand/receipt atomically with
   current permissions, consent, source limit and pinned references rechecked.
   Exact replay returns the committed result under current authorized reads;
   changed same-key content conflicts. New work after revocation/expiry fails.
   Keep immutable results discoverable through the approved payload-free recovery
   route when ordinary result access is no longer authorized.

## Server mapping and compatibility

Go transports selection and verified identity; SQL remains authoritative for
rules, locks, exact references and mutation. Server preparation resolves supported
type/subtype into the existing reviewed asset/liability profile, creates distinct
account proposals using the current chart, and resolves ISO4217 currency version
on the book-local binding date. Binding active date is not a transaction-date
rewrite or permission to exclude older returned transactions. Existing historical
currency/period review rules remain. Missing/ambiguous currency or a code conflict
returns an actionable review state; no silent default to USD/current period.

The registration plan is immutable once reviewed. If chart/reference/authority
changes invalidate it, return the new proposal for renewed review rather than
silently changing names/codes/account IDs. Older direct bind clients retain their
strict reviewed request and original receipt semantics; do not reinterpret them
as append commands. Final physical SQL, closed API schemas, maximum/expiry bounds
and reconnect/rebind policy must be reviewed in392/393 contracts before code.

Only safe bounded display name, subtype and optional mask are proposed under290.
Masks can collide; they are not identity. Preserve immutable provider IDs
internally, with stable account ordinals for duplicate/missing-name fallbacks.
Do not add balances/full numbers/holder details to this display projection or
logs. Names remain untrusted text; escape them and constrain size/control chars.

Already-shared accounts can be selected from the existing verified Item. Accounts
not actually shared require additional provider consent/update mode; do not
promise a login-free path for those. Update-mode account deselection can revoke
sharing of previously selected accounts: keep checking selected when adding
savings, detect the authoritative changed manifest and stop unauthorized work.
Do not delete posted provenance in response to provider deselection.

## Delivery and leases

| Issue | Bounded outcome | Dependency and ownership |
| --- | --- | --- |
|290 | Minimal recognizable metadata, unique fallback, privacy/API contract | Existing issue; metadata contract before schema/API field changes |
|387 | Clear complete skipped-row mapping | Existing issue; bounded regression can land first; serialize Connections edits |
|392 | Explicit scoped source limit, durable plan and atomic append | Reviewed physical contract/source evidence; sole SQL/migration writer |
|393 | Verified prepare/execute/inspect/ack API and generated types |392 and290 contract; one API authority owner |
|394 | Selection-only account picker and plain recovery |393+290; include387 behavior; sole Connections component writer |
|395 | Synthetic compatibility and fresh hosted qualification/release |All source slices/required CI; sole release/cloud writer |

391 owns this plan only. No agent is dispatched by a generated packet. Each slice
rechecks live dependencies/65, claims a finite bounded lease and dedicated branch,
then records exact evidence. No missing/canceled predecessor is treated as done.
SQL contract/migration numbering and managed apply are serialized. Offline fixture
or design review may be independent; shared source files require exclusive leases.

## Acceptance and release order

Proposed named SQL tests: TestRegistrationScopeLimit (default1 denies2, scoped
owner opt-in2, paused counted, concurrent append); TestRegistrationAppendPreservesExisting;
TestRegistrationPlanAtomicity; TestRegistrationPlanReplayAndUnknownCommit;
TestRegistrationPlanCurrentAuthority; TestRegistrationCurrencyAndProfile.
Use fixed independent synthetic expectations: existing checking, available
savings, foreign/unsupported accounts, duplicate names/masks, two currencies and
historical currency boundaries. Prove no partial accounts/grants/demand, no lost
history, no duplicate source/stream, no journal from registration/acquisition.

Browser/API/SQL evidence must cover first-one and first-many, then add-second;
skipped row cannot block selected row; registered account cannot be duplicated;
no required technical form on supported happy path; advanced override; keyboard/
mobile; cancel/reload/remount; lost execute response after commit; exact retry;
expired/stale/foreign/revoked/unsupported/cap-exceeded requests. Stop after unknown
mutation outcome until same-plan recovery resolves it. Import counts/readiness
do not substitute for independent complete-set value comparison.

Run core make check/test/build, migration-test, ledger-test, locked web-check/
web-test/web-build/web-browser and app-integration as applicable; exact-head CI,
human security review and reviewed migration-first rollout. Keep default1 during
deployment and verify existing first-checking identity/history predicates privately.
Then explicitly scoped fresh Sandbox one→two account proof with independent
fixtures/recovery, before a concrete reviewed finite second-account Production
proposal. Old Sandbox consent is expired; this issue grants no provider calls.
Never down-migrate live provenance or reset/relink checking as rollback. Forward
repair or pause acquisition retains books; a restore is separately reviewed.

## Sources, standards and limits

[Reviewed source record](../research/select-register-sources.md) distinguishes
Plaid facts from the proposed Jayburd UX/SQL policy. Internal setup/selection is
not an ISO20022 financial exchange and has no invented registered message ID.
Use proprietary Plaid API2020-09-14 mapping, installed ISO4217 dated identifiers/
minor units, decimal strings, original transaction units/dates and existing
immutable correction semantics. ISO21378:2019 audit-data relevance remains under
the existing mapping/access limits; no new export or conformance claim. Apply
[standards review](../contracts/standards.md) STD01–05 in392/393 physical contracts.
No normative ISO edition/text newly reviewed by this planning pass.

This is a bounded improvement to the existing book-scoped path, not the whole237
bookless registry/cross-entity design. No new account-type/product policy, extra
Items, paid Refresh, credentials migration, infrastructure, automatic bookkeeping
rule or money movement. The owner can review current checking transactions while
this is implemented; no need to reconnect that account.
