# First Sandbox book bootstrap

## Decision

Issue #136 authorizes two owner-operated SQL stages, never application startup.
The operator selects only `jayburd-bookkeeping-dev` and database `bookkeeping`.
Existing session must have SET authority for safe NOLOGIN `bookkeeping_owner`;
identity stage also requires SET for `bookkeeping_identity_operator`. Missing
capability fails before mutation. Scripts grant no roles and accept no credentials.
API/migrator/worker sessions are not approved operators. Merged migrations remain
unchanged; no persistent helper functions or alternate authentication path exist.

## Reference and identity stage

`sandbox-identity.sql` takes `project`, `subject`, `currency_id`, `source_release`,
`valid_from`, `valid_to`. Issuer is fixed; subject is the invited Identity Platform
UID copied by its owner, never email/token. Exact pinned USD metadata is installed
under owner role, then enrollment runs under the identity-operator role in the
same bounded transaction. Existing currency must match every field; overlapping
or conflicting IDs/versions fail. Existing disabled principals cannot reactivate.
Only currency/principal UUIDs and fixed PASS text are reported; never UID or email.

## Book stage

UI setup may use the custom form or the bounded Sandbox starter preset. The preset
only prepares the ordinary durable setup command with a personal entity, one
monetary checking account, and non-monetary equity/income/expense accounts. The
owner still reviews and explicitly executes it; it adds no alternate SQL or auth
path. After UI setup, `sandbox-book.sql` takes `project`, `actor_id`, `entity_id`,
`book_id`, `worker_login`, `publisher_login`, `scheduler_paused`. Exact fixed dev
IAM login names and a true owner-observed scheduler-paused attestation are required;
SQL cannot query cloud scheduler state. Existing SQL capability predicates must
validate both worker logins. A setup receipt must prove actor created this exact
book/entity, with active read/post authority. Shared read access alone is insufficient.

A distinct per-book service principal uses issuer `urn:jayburd:service:jayburd-bookkeeping-dev`
and subject `acquisition:<book UUID>`; it cannot map a Firebase user. Insert only
missing actor import and service read/import memberships; reject revoked grants or
disabled subjects. Existing owner functions grant connect and enroll connection /
delivery bindings with fixed idempotency keys. No sources, journals, financial
accounts, provider calls, cloud scheduling or database role grants are performed.
Atomic rollback covers new service principal, grants and both binding operations.
Replay never reactivates revoked bindings. Output is IDs/revisions only.

## Standards and source review

USD/840/2 was verified against SIX List One XML, publication `2026-09-17`, retrieved
2026-10-15, SHA256 `33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff`.
[Official maintenance page](https://www.six-group.com/en/products-services/financial-information/market-reference-data/data-standards.html)
and [List One XML](https://www.six-group.com/dam/download/financial-information/data-center/iso-currrency/lists/list-one.xml)
were reviewed for agency ownership, publication and the US Dollar record, not the
full ISO text. The local approved validity is `[2026-09-17,2027-01-01)`; publication
does not establish USD's historical inception. This narrow local window is policy,
not an ISO effective-date claim. Earlier books/import dates need separate reviewed
reference history. `source_release` retains publication/hash provenance, never a
synthetic-fixture claim. No exchange rates or financial messages are introduced;
ISO 20022 does not apply to these internal provisioning controls.

Alternative runtime auto-enrollment would widen authority. Ad hoc administrator
DML would omit replay/conflict checks. Neither is selected. Synthetic PostgreSQL
acceptance is `scripts/test-sandbox-bootstrap.sh`; hosted execution remains #133.
