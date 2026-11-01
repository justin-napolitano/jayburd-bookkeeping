# Production personal-book bootstrap — #374

Decision candidate2026-11-01. Existing owner-approved shared instance and isolated
bookkeeping_prod; no new database, migration, runtime authority or provider call.
Source review: identity-setup enrollment/setup boundary; sandbox-book-bootstrap
reference and creator proof; production-source-sql operator scope; migration34
configure_provider_scope; sandbox-book.sql and worker-logins.sql. Reviewed their
selected SQL bodies/guards, not an independent audit of all dependencies.

## Reference and standards

Use the same reviewed SIX ISO4217 List One2026-09-17 USD record and source hash
33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff
recorded in sandbox-book-bootstrap's Standards and source review. No new external
source or current-list claim. Mapping: USD/840/minor-unit2; preserve exact source
release and the existing local policy window[2026-09-17,2027-01-01). This is a
bounded project validity policy, not USD inception or ISO effective dates. Existing
historical reference installation is unchanged; the new reference stage does not
call a Sandbox-named history installer. ISO20022 has no registered message for
this operator setup. ISO21378 field-level conformance is not claimed. No FX,
currency redefinition or accounting recognition change.

production-reference.sql accepts only exact project/number, source/window and a
nonzero canonical currency UUID, checks bookkeeping_prod and safe existing owner
SET capability before mutation. Under owner role, serialize USD provisioning and
insert only the missing exact reference; changed replay or overlapping ID rejects.
It does not enroll identities or create books. Facts: existing constraints reject
currency overlaps and setup requires a contained policy window. Inference: a Q4
book can use this reference while90-day transaction review keeps original dates;
historical review does not authorize posting outside the chosen open period.

## Historical USD coverage — #380

Actual metadata2026-11-01 found the Production history interval absent. Migration25
ran before this bootstrap reference existed. Missing dated metadata rejects the
whole acquisition set; a Q4 book therefore does not alone support a90-day import.
Source review: [dated currency decision](plaid-history-currency.md), migration25
installer/context/SQL validation and existing acquisition history tests. Re-reviewed
MA115 primary PDF page1 text on2026-11-01: El Salvador USD/840/2 effective2001-01-01.
Web screenshot cache failed; native rendering had a transparency limitation. This
is historical corroboration, not USD inception or a full amendment-series audit.
The existing pinned SIX2026-09-17 USD record/hash remains unchanged.

Decision: the same guarded Production reference transaction also appends exact
USD/840/2 local coverage[2001-01-01,2026-09-17), fixed public-reference ID
3a184659-cf9b-4b80-b140-3f7a7796cbbd and provenance
ISO4217-MA115:2002-05-13+SIX-list-one:2026-09-17;local-coverage-v1.
This reuses the reviewed public ISO metadata identity, never a Sandbox bank Item,
credential or financial record. Both current and historical rows are insert-only;
exact replay checks every field, conflict/overlap aborts both inserts atomically.
Existing advisory lock and interval constraint remain authoritative. Do not invoke
the Sandbox-named helper in Production or install reference data at runtime.

Alternatives of shortening history, changing transaction dates/current provenance,
or ignoring reference dates lose source fidelity or rewrite evidence. ISO4217
identifiers/minor units and the existing local supported-coverage profile apply;
no new ISO20022 operation or ISO21378 conformance claim. No currency/FX/recognition
policy, book period, source observation, cursor or journal changes. Human review
of this concrete reference-only change precedes actual apply. Acceptance adds
independent fixed history/current metadata, immutable replay, late rollback,
conflict/overlap denials and SQL dated validation at the interval boundary; existing
historical acquisition/ledger regressions and core/CI remain required.

## Book and acquisition scope

Owner privately signs in and prepares/reviews/acknowledges the ordinary durable
setup command; no direct insertion of books, accounts, balances or setup receipts.
Book name/currency/period/current supported policy require owner selection.
production-book.sql takes exact known project/number, actor/entity/book IDs,
Production sync/publisher logins and owner-observed scheduler_paused=true. It
requires safe existing operator owner SET, safe API/sync/publisher roles and actual
SQL CONNECT denial to Sandbox. SQL cannot verify a Cloud Scheduler attestation;
operator must independently observe paused/absent scheduling before invoking it.
No SQL/cloud role is created or granted by either new script.

Creator setup receipt, active actor/read/post grants and safe worker predicates
must pass. Reject revoked grants, disabled principals and changed bindings.
Create only a distinct per-book service principal with a Production-specific
issuer suffix and narrow actor import/connect plus service read/import grants.
Call existing configure_provider_scope for production, fixed API/sync logins and
the already approved jayburd-prod bank-credentials/hosted-link KMS resources, then
existing connection-worker and delivery functions. Closed scope has provider
origin, never relabels Sandbox state; its initial-empty-book guard is SQL-owned.
All scope/grants/service principal/receipts/bindings commit in one serializable
transaction. Replay cannot reactivate revoked state. Output contains only receipt
IDs/revisions and PASS; real IDs stay in private operator metadata, not public logs.

Alternatives: runtime auto-provisioning widens authority; ad hoc grants omit
replay/creator/current-state fences. Reusing sandbox-book.sql would choose the
wrong database/logins and omit Production scope. Explicit scripts reuse existing
domain SQL instead of adding another business-rule implementation or DDL.

STRIDE: elevation/confusion(high) -> exact targets/logins/keys, creator proof,
current permissions and cross-DB denial; replay(high) -> existing receipts and
revocation checks; disclosure(high) -> public metadata only/fixed errors; partial
setup(high) -> atomic transaction and late-failure rollback test. No bank consent,
receiver exposure, provider calls, scheduling, credential values or journal effects.
Human source/security review before real reference/scope provisioning. Activation
and the exact finite pilot package remain375/337/316.

Acceptance: test-production-book-bootstrap.sh exercises actual synthetic PG roles,
reference initial/exact replay/conflict/overlap, wrong target/operator/logins/keys,
creator/current-state denials, initial Production scope/delivery/exact replay,
late rollback, revoked-state denials, no Items/journals and safe diagnostics.
Sandbox bootstrap regression, core/release/ledger/app gates and exact-head CI.
