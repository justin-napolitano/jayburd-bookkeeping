# Selection registration HTTP and recovery —393

Engineering review2026-11-02, following owner-reviewed399/44b8a53 and
[registration SQL](registration-sql.md). Source-only; no managed migration,
additional account, provider call or consent renewal. Existing SQL36 is immutable.

## Transport

POST `/v1/entities/{entity}/books/{book}/connection-attempts/{attempt}/registration`
requires verified bearer identity, current origin/admission controls, canonical
nonzero scoped UUIDs, one Idempotency-Key and application/json with no encoding or
media parameters. Closed body `{account_ids:[string,...]}`1–25 unique printable
immutable provider identifiers,1–256 bytes each;16KiB maximum. Reject extra actor,
environment, currency/date/code/mapping fields, duplicate JSON keys and queries.
Success201 returns the existing DurableCommand with operation register_accounts.
Exact same-key/selection replay returns the same plan. No request/result is stored
in browser persistence or logs. SQL owns selection authority and proposals.

The private Next bridge allowlists this exact canonical POST only, requires the
same idempotency key/16KiB bound and rejects media parameters/encoding before
service-token or backend access. It forwards unchanged JSON once, with separate
user and service identities; no automatic write retry or alternate target.

Only the explicitly injected connection service exposes preparation. The service
requires current finite runtime approval and current private SQL attempt/profile/
key preflight before preparation, without invoking provider/token/crypto I/O.
The server generates the audit request UUID. SQL's single pending-command gate
persists the exact resolved mappings before any execution; preparation never binds.
Generic POST commands continues rejecting register_accounts input, so clients
cannot forge the server-resolved body through a second route.

GET commands/pending and existing execute/cancel/acknowledge/clear routes recover
this operation. Every execute uses the saved command ID/body and current SQL
authority. Exact committed replay bypasses new-work expiry checks as in36, while
still requiring current readable scope. Unknown commit never becomes a fresh key
or implicit success/acknowledgement. Cancel prepared then acknowledge explicitly;
committed cannot be canceled. Revoked scope uses payload-free clear. No new command
transition or automatic execution is introduced by discovery/remount.

OpenAPI describes the selection and immutable RegistrationCommandBody (selection,
attempt/Item/generation, worker/scope/limit/book revisions, policy/date/expiry,
manifest/proposal digests, initial flag and exact bindings). Registration result is
the existing initial ConnectionAttempt or minimal append result with scope/Item/
streams. Decimal revisions remain strings. RegisterCommand input does not gain a
register_accounts variant; only DurableCommand output does.

Safe errors: invalid input400, current denial404/403 under existing operation
policy, pending/conflict/stale/registration_not_ready409, capacity422 and unknown
commit/temporary failure503. Never return raw SQL/provider/crypto errors. These
codes guide refresh/cancel/review, never retry with a different key after ambiguity.
Managed readiness requires the installed restricted36 preparer before serving.

##290 metadata/privacy prerequisite assessment

This slice adds no bank name/mask, full account number, balance, holder, transaction
description or raw provider field. It returns only the existing verified account
ID/type/subtype/currency projection and server-proposed ledger code/name/date plus
owned book IDs already present in the reviewed contracts. Names in proposals are
neutral SQL profile labels, not new provider payload collection. Immutable IDs
remain selection identity, not primary UI labels. SQL still determines eligibility/
capacity and existing connection discovery identifies registered streams.

This resolves the field-expansion review prerequisite for this transport slice;
290 stays open for recognizable private presentation/fallbacks in394. Any later
provider display-name/mask storage needs its own reviewed contract/SQL and named
privacy fixtures. Do not imply these fields are unavailable at the bank or invent
masked digits. Advanced mappings retain the existing reviewed binding path; this
preparer is the automatic new-ledger-account path implemented in36. No separate
new product/type/currency policy is implemented in Go or the browser.

## Browser recovery boundary

Add typed prepareRegistration to the shared memory-only recovery hook, retaining
its exact endpoint/body/key after an uncertain prepare. Remount discovers the
same server-owned command. The recovered panel displays every proposed ledger
account, currency and binding date plus book/expiry before an explicit Register
selected accounts action. A committed result says registration confirmed, not
book creation or posting; acknowledgement says Done. No automatic command execute
or acknowledge on load.394 owns the account picker and one-action normal journey.

## Evidence, standards and controls

Sources: [SR01–04](../research/select-register-sources.md), SQL36 and existing
connection HTTP/command recovery contracts. Reuse reviewed public sections; no
new vendor behavior/full normative ISO text or real records reviewed. Internal
setup/access/recovery has no applicable ISO20022 registered message; proprietary
Plaid API2020-09-14, ISO4217 dated currency/minor units and original amounts/dates
unchanged. No ISO21378 export or new conformance claim. STD01 applicability here;
STD02 closed HTTP/type fixtures; STD03 unchanged currency/date mapping in36;
STD04 same-plan prepare/execute/recovery; STD05 migration-first typed compatibility.

STRIDE: spoof/escalation(high)→verified identity/current SQL/profile/finite preflight;
tamper/replay(high)→closed selection, server-generated plan/audit ID, saved IDs;
disclosure(high)→existing minimal projection/no-store/redacted errors/no browser
persistence; DoS(medium)→16KiB/25 bounds/deadlines; repudiation(high)→existing
command/connection audit. Human exact security review before merge/deploy.

Acceptance: service tests prove syntax/profile/expired gates before any prepare,
zero provider/crypto effects, exact args/retry identity; HTTP tests auth/origin/
headers/duplicates/spoofed fields/closed route and generated audit UUID; restricted
SQL+Go integration initial→explicit ack→append with no journals; stale/cap/revoked
errors; browser prepare-response loss/remount/explicit execute/ack. Core, locked
web, app-integration and applicable SQL/ledger gates plus exact-head CI. Proposed
tests are not passes until recorded in the stage runbook.395 owns hosted proof.
