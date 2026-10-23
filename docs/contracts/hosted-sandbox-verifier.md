# Hosted Sandbox verifier execution

## Scope and decision evidence

Issue272, owner request2026-10-23 to provision the restricted verifier and execute
a fresh independently authored Sandbox proof. Engineering profile for review;
new sensitive runtime/IAM source requires human review before release. No new
product migration, public endpoint, provider secret access or posting permission.
The existing [comparison contract](sandbox-verification.md) remains authoritative.

Use one private Cloud Run job with the trusted worker image containing the inert
`/sandbox-verify` binary. Its separate IAM identity gets only Cloud SQL connector/
IAM-login roles and read access to a dedicated synthetic-input bucket. Owner SQL
bootstrap grants only the preexisting NOLOGIN capability; membership has INHERIT
true and ADMIN/SET false. No credential key or service-account impersonation is
created. One task, parallelism1, retries0, timeout120s; comparator budgets remain
the existing15s transport/4s transaction/3s statement/1s lock. No scheduler.

Only a reviewed run ID directory is mounted read-only at `/verification`. Inputs
are exactly manifest.json, fixture.json and identity-map.json. Existing CLI hashes
both financial fixture and identity-map bytes against a finite owner mandate
before connecting. Bucket is private, uniform-access, no versioning or soft-delete,
with one-day fallback lifecycle. Operator uploads each object create-only and
deletes the three exact generation-pinned objects immediately after comparison;
lifecycle is not a claim of prompt deletion. No secrets or raw provider responses
may enter this bucket, Terraform variables/state, logs or public evidence. Bucket
viewer permits reading other directories in that dedicated synthetic-only bucket;
`only-dir` is a filesystem view, not an IAM boundary. Remove inputs promptly and
grant no access to application evidence/token buckets. Disable runtime when idle.

## Acceptance sequence and boundaries

Current hosted application configuration expired2026-10-22. The concrete release
candidate uses trusted build39898191502 attempt1/source edb778e, release expiry
2026-10-23T18:00Z and new-session Sandbox consent deadline2026-10-23T20:00Z.
This is an explicitly proposed fresh-run interval, not permission to extend an
existing Item. Application configuration supplies new enrollment metadata; only
the new synthetic source is enrolled. Its persisted Item consent remains SQL
authority. Existing sources keep their prior consent and no live connection is
authorized. Exact-hash owner dispatch under deploy/README.md remains required.

1. Validate safe platform identity/SQL capability without a business mandate.
2. Author a complete small fixture before provider injection; pin exact dates and
   original lexemes independently. Start with one new synthetic checking source.
   Equal-valued purchases retain distinct logical identities. USD is this case's
   test unit, not a reduction of required multicurrency scope.
3. Use a new bounded Sandbox source consent; expired prior consent is not renewed
   implicitly. Use real managed browser identity and reviewed app book setup.
4. Obtain a separate identity-only map using the reviewed projection below;
   do not derive expected financial fields from adapter/observations/provider output.
5. Complete actual acquisition and idle stream; owner-enroll exact current source/
   binding/Item/generation plus both digests under a fresh <=60-minute mandate.
6. Upload exact bounded files outside Terraform; run one explicit job. Validate
   execution identity/image/command/volume/action and bounded result. Preserve
   both equality and absolute-zero journal/outbox evidence; no test posting.
7. Revoke the mandate, delete exact input generations and disable runtime. Retain
   only redacted outcomes and metadata receipts. Unknown commit or mismatch is
   not PASS; no automatic retry, source mutation or scope widening.

## Reviewed sources and standards

### Identity readback correction, engineering profile for review

The [Sandbox create response](https://plaid.com/docs/api/sandbox/#sandboxtransactionscreate)
documents only request_id, not transaction IDs (selected request/response sections
reviewed2026-10-23). The earlier comparison contract's assumption that injection
response metadata supplies the map cannot implement this endpoint's actual wire
shape. This correction permits a separate bounded readback solely for identity.

For the first fresh posted-only proof, independently author all financial fixture
rows before creating/injecting any provider data. Give each logical row a unique
synthetic description tag, including purchases with equal amounts. Owner-private
transport pipes one complete initial account-filtered sync page to the offline
identity projector. It requires HISTORICAL_UPDATE_COMPLETE, no further pages and
no modified/removed records, exact expected row count and account, unique matching
tags and unique provider IDs. Larger/dynamic histories require a separately
reviewed pagination/history protocol; this helper cannot certify them.

Projection copies transaction_id only. It consults name only to match pre-authored
identity tags; it never reads amounts/dates/currencies/states into the expectation
or compares them to decide which identity matches. Those fields are tested later
by SQL against the unchanged authored fixture. Wrong/missing/duplicate tags and
unexpected identities reject the map rather than adapting expectations. Original
financial fixture bytes remain hash-pinned; modifying provider financial fields
cannot change the projected ID map. Raw readback stays ephemeral/unlogged in the
owner-private pipe; only the map is written create-only to the private run directory.
The agent sees fixed outcomes/counts, not raw readback. No token/DSN/network flag
or credential discovery belongs in the projector.

The local operator command is `bin/sandbox-identity-map -action=project
-fixture=<absolute reviewed directory>/fixture.json -fixture-sha256=<approved hash>
-account-id=<exact synthetic account ID> -input-fd=3`. The owner supplies only the
complete account-filtered response through a private pipe on descriptor3. Saved
payload files/stdin/stderr are rejected. It writes the fixed sibling identity-map.json
create-only/0600 and emits only a readiness Boolean/count. This is an offline
identity association tool, not a provider transport, acquisition or evidence of
financial equality. No raw response is sent to the agent or retained in artifacts.

For this first case, directly configure the custom user with the independently
authored transactions before Link; no transaction-create endpoint call is needed.
Use a separate complete initial provider readback for identity association. If the
owner cannot supply the authorized private pipe, leave the hosted proof blocked;
do not substitute observed SQL rows, browser UI counts or a general data dump.

Alternative matching by amount/date or regenerating expectations from observed
rows would destroy oracle independence. Trusting creation-array order is not
documented and cannot identify returned IDs anyway. A tagged identity-only
projection narrows the correction to provider identity association. Named tests
must prove financial-field changes leave the map identical and incorrect identity,
account, incomplete/duplicate/modified input cannot produce a map. Hosted proof
still requires actual acquisition and complete comparison, not projector success.

Reviewed2026-10-23 selected official sections, not full product manuals:

- [Cloud Run job storage mounts](https://docs.cloud.google.com/run/docs/configuring/jobs/cloud-storage-volume-mounts),
  Required roles, Terraform, Limitations and mount-specific-directory sections:
  storage viewer supplies reads, read-only GCS mounts expose files without image
  changes; FUSE lacks full POSIX/concurrent-write semantics. Our create-only inputs,
  digest checks and dedicated bucket are design controls, not FUSE guarantees.
- [Plaid custom users](https://plaid.com/docs/sandbox/user-custom/), configuration,
  transactions and date handling: custom account overrides support explicit data;
  default fields may generate data and date rolling changes expected dates. Use
  explicit history and roll_dates_forward=false for exact dated fixtures.
- Existing PostgreSQL17 role/function/locking sources in the comparison contract
  govern SQL authority; no new financial comparison algorithm or DDL is introduced.

Alternative: general operator SQL or reused sync identity broadens financial and
credential authority. Baking dynamic inputs into an image retains them in build
artifacts; secret volumes misuse credential storage for ephemeral test inputs.
The isolated bucket adds a narrow private input channel while preserving the
existing CLI and image. Actual FUSE compatibility and IAM denial require hosted
evidence, not a mocked Terraform PASS.

ISO20022 registered messages do not apply to this proprietary operational
execution/input transport. ISO4217 dated identity and original units remain
required by the unchanged comparison profile; ISO21378:2019 audit relevance has
only the previously reviewed abstract, with no export/conformance claim.

Named gates: verifier bootstrap replay/unsafe-role/ACL rejection against real
disposable PostgreSQL30; mocked infrastructure deny-by-default/exact job/image/
identity/read-only bucket/no secret access; existing independent comparison tests;
core/migration/ledger gates as applicable. Actual hosted receipt/cleanup remain
separate acceptance. Follow-up272/88; no production qualification.

## Focused threat review

| STRIDE boundary / risk | Control and named evidence |
| --- | --- |
| Spoofed scope/login or elevated SQL access (high) | Derived verifier login, canonical SQL effective-ACL predicate, finite mandate pins; real bootstrap and existing verifier authorization tests |
| Changed expectations or substituted identity input (high) | Fixture authored before provider action, owner-private identity-only projection, exact fixture/map hashes and complete comparison; financial-independence and ambiguous-identity negatives |
| Unauthorized disclosure via storage/output (high) | Dedicated private bucket, read-only runtime, no provider-secret rights, private create-only map, fixed CLI outcomes; mocked IAM and CLI redaction/file-mode tests; actual hosted IAM still pending |
| Replay/unknown commit falsely recorded as PASS (high) | Existing one-attempt comparator drops uncertain results; zero job retries, validated execution plus bounded result; existing unknown-commit tests and operator receipt review |
| Unbounded job/input or lingering data (medium) | One task/pool1/120s job, bounded files/pipe/transaction, exact generation cleanup and idle job removal; limit tests and actual cleanup readback required |

This is a bounded engineering threat review; it is not independent AppSec signoff.
