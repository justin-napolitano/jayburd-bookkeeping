<!-- stage-runbook: true -->
# First Sandbox book operator setup

## Status and boundary

Issue #136 supplies two explicit operator stages and synthetic PostgreSQL tests.
No hosted execution, new SQL role grant, source connection or personal data was
performed. [Contract](../contracts/sandbox-book-bootstrap.md) owns exact inputs,
reference provenance and boundaries. Hosted execution belongs to #133.

## Stage ledger

- Reviewed existing enrollment/setup/connection/delivery SQL and operator bootstrap.
- Verified SIX List One's publication/hash and USD record; recorded a bounded local
  validity window, distinct from ISO historical dates.
- Added transactional identity/reference and creator-owned book bootstrap scripts.
- Added isolated PostgreSQL tests and the application-integration CI step.

## Current evidence

PASS: eight named PostgreSQL test groups, including initial/replay/conflict,
rollback after a deliberately failed delivery insert, disabled actor/service,
revoked grant, cross-entity, wrong/missing worker capability and API denial.
PASS: `make check test build` and pinned actionlint on `seed.yml`. Initial
sandbox execution lacked Docker/loopback permission; authorized isolated reruns
passed. Independent review strengthened expected-error assertions and disabled
binding cases; those use existing operator functions with fresh idempotency keys. Tests use fictional subjects,
no network, and disposable PostgreSQL; source provenance assertions use the reviewed
public source identifier, not fictional accounting data as ISO authority.

## Remaining work

Owner provides their Identity Platform UID privately (Users console), never email,
password, token or TOTP code. The local rehearsal does not expose UID. Confirm the
existing operator has SET on both owner and identity-operator roles. Scripts do
not self-escalate. Missing owner SET is a separate administration-review blocker.
If enrollment authority alone is missing, the administrator may explicitly review
`GRANT bookkeeping_identity_operator TO "<operator IAM login>" WITH ADMIN FALSE,
INHERIT FALSE, SET TRUE`; do not grant this to API/migrator/workers.

Use the retained [Cloud Shell proxy procedure](cloud-shell-bootstrap.md), verify
its dev instance, private socket and selected `bookkeeping` database first. Run
psql with `-X -qAt -v ON_ERROR_STOP=1`, no password prompt, and redirect stderr to
an owner-private file; never upload diagnostics. Do not use shell tracing/SQL echo.
Before each stage review the merged script hash and exact metadata inputs.

Identity stage: `-v project=jayburd-bookkeeping-dev -v subject=<owner UID>` plus a
new reviewed `currency_id` UUID, `valid_from=2026-09-17`, `valid_to=2027-01-01`, and
`source_release=SIX-list-one:2026-09-17:sha256:33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff`,
then `-f db/bootstrap/sandbox-identity.sql`. Save returned principal/currency IDs.
Use this currency in the UI to create a fictional book entirely within that window.
A changed UID deliberately identifies a different invited principal; review it as
new enrollment, never assume it is a retry of the first person.

Book stage: review creator `actor_id`, returned `entity_id`/`book_id`, exact
`worker_login=jayburd-dev-sync@jayburd-bookkeeping-dev.iam` and
`publisher_login=jayburd-dev-publisher@jayburd-bookkeeping-dev.iam`.
Workers must already pass `db/bootstrap/worker-logins.sql`. Observe the acquisition
scheduler is PAUSED, then attest `scheduler_paused=true` and invoke
`db/bootstrap/sandbox-book.sql` with these `-v` inputs and the fixed project.
Save returned service principal and binding IDs for the reviewed worker release.
The script does not contact or change Cloud Scheduler; recheck it before rollout.

## PR handoff

No migration change or runtime privilege widening. Only the exact creator's book
is eligible; no initial balance, journal, source account or provider payload is
created. Commit uncertainty permits identical-input replay; changed or revoked
state rejects. Live readiness, historical currency coverage and hosted acceptance
remain outside this PR. Owner/operator authority still requires actual readback.
