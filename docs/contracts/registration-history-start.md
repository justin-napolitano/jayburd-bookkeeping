# Historical ledger availability during registration —394

2026-11-02 proposed394 correction, exact human source review before merge/apply.
Actual normal browser→Next→Go→SQL registration/import succeeded, then draft of
fixed original2026-10-13 transaction0.10USD failed P2500:36 preparation makes the
new ledger account active on today's binding date;11 source draft requires both
ledger accounts active on the unchanged accounting date. Fixture transactions
and financial assertions must not be moved forward merely to hide this failure.

Reviewed repository primary sections:36 prepare_registration and
apply_registration_bindings (new-account.active_from=d and INSERT true,d);11
prepare_review_draft active-account/policy/period checks;1 immutable book_policy,
setup_change policy overlap/revision triggers; existing posting-boundary, standards,
registration SQL/options and database foundations. Bibliography/source register
SR01–04 and S08/S12 access limits unchanged; no new vendor/normative ISO text read.
This is an internal account-availability policy interpretation, not an ISO claim.

Decision proposal: new automatic ledger accounts are available from the earliest
approved book-policy effective_from on/before binding day. This describes when
the ledger can represent entries, not when a bank opened, an opening balance,
source consent or permission to post. Keep the actual source binding date today,
every original transaction/accounting date, original units and dated currency
resolution. Draft/post still requires the actual applicable approved policy,
open accounting period, active category/source accounts, current permissions and
explicit owner confirmation. Missing historical policy/currency/period still holds.
No account, source, grant or journal is created by reading a proposal.
Existing accounts and their accepted availability dates are not rewritten.395 must
check older imported transactions on the already-bound checking account and plan
an explicit preserving repair if that older setup needs one; no live outcome is
inferred from this new-account fix.

Alternatives: automatically backdate binding by90 days guesses coverage/consent;
move transaction dates corrupts provenance; bypass active-date/closed-period checks
weakens accounting; require Advanced dates preserves the broken normal path.
Using the already approved policy start gives a defined ledger boundary while
preserving independent draft/post review. Multiple policy ranges/gaps do not grant
posting in a gap: the existing transaction-date policy lookup still governs it.

Forward38 only,37/earlier immutable. Owner-only registration_account_start(e,b,d)
requires a unique approved policy effective on d, then returns minimum approved
effective_from<=d.38 automatic preparation persists that exact date in the existing
new_ledger_account.active_from; current book revision/policy/date/proposal digest
remain pinned. Policy insertion advances the book revision, so stale work cannot
silently pick a different start. Saved preview shows the proposed ledger start.

Shared binding helper accepts a new account's active_from equal to binding date
(unchanged legacy behavior), or exactly the derived approved ledger start. It
inserts the reviewed active_from, not binding date. Arbitrary earlier/later dates
are rejected. Legacy direct initial mapping can choose only those two existing
approved boundaries; no arbitrary backdating, new runtime privilege or endpoint.
Binding/current authority and final execution validation remain unchanged.

Down38 refuses if any saved registration command or bind receipt uses a distinct
historical new-account start, including cancelled/acknowledged intent; never erase
reviewed history. Empty down restores exact37 preparer and36 binding helper, then
removes only the new private helper. Runtime readiness requires installed private38
helper before serving. No startup/managed migration or live retroactive mutation.

Standards: internal ledger activation has no applicable ISO20022 message/version;
ISO4217 transaction-date references and proprietary Plaid API2020-09-14 mapping
unchanged. No new ISO21378 export/conformance assertion. STD01 applicability here;
STD02 accepted date boundaries/closed payloads; STD03 fixed original dates/units;
STD04 saved-date/revision/down guards; STD05 old mapping compatibility/forward38.
STRIDE: owned scope/SQL-only derivation; exact immutable proposal/current revision;
no new provider fields/logging; grants/period/post authority unchanged. This new
SQL/accounting-boundary code requires human review of the concrete patch.

Acceptance: TestRegistrationHistoricalAccountStart proves original dates retained,
approved start persisted and arbitrary start denied; stale policy revision and
down guard remain fail-closed. Existing RegistrationCurrencyAndProfile/profile/
atomicity/authority/replay tests and ledger/migration gates. The real normal
browser→Next→Go→SQL one→append→historical review→explicit exact0.10USD posting/
lost-response recovery must pass without replacing fixture transaction dates.
This is source evidence, not real-LLC/accountant cutover or live reconciliation.
