# Provider ingestion and Plaid adapter

## Decision and scope

**Proposed contract**, owner project maintainer, 2026-10-13, [#70](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/70).
The [runtime implementation boundaries](provider-runtime-boundaries.md) reconcile
this proposal with merged migrations9–11 and define the current synthetic build.
Proposal capabilities below must not be read as implemented provider behavior.
Plan Plaid server-to-server Transactions alongside [CSV](statement-import.md),
feeding a shared source-observation/review layer. CSV is one fixture adapter, not
the architecture. No provider runtime, source tables, general draft API or automated
posting exists by virtue of this document. [PI01–PI08](../research/plaid-ingestion-sources.md)
record official source review. #8 owns importer implementation; #13 owns provider
qualification/cutover; #63/#7 own generic work delivery. No live calls authorized.

Propose profile `jayburd-plaid-transactions/1`, explicit `Plaid-Version: 2020-09-14`,
Transactions-only product access, initially depository accounts bound to monetary
asset accounts. Countries, institution coverage, fees/consent and selected SDK are
unresolved implementation gates. Credit/loan/investment support needs distinct
account mapping; do not silently treat it as deposits. Native bank CSV, camt and
Plaid remain distinct protocols with shared internal semantics.

## Standards applicability

Plaid JSON is a proprietary financial exchange, not ISO 20022/camt. The mapping
below is the proposed non-ISO-adapter decision required by [standards](standards.md);
owner review is required before implementation. ISO 4217 applies to currency-code/
validity references; ISO 8601 date semantics use the existing RFC3339 full-date
profile. Preserve ISO currency metadata versions; do not call unofficial assets
ISO currencies. Audit exports under ISO 21378/21926 remain separate unresolved
mapping work. No IBAN/BIC/LEI interpretation is introduced. #17's official schema/
usage-profile gap remains open; SDK/schema acceptance alone is not conformance.

## Connection lifecycle and authorization

A connection is one account-specific acquisition stream owning provider environment,
Item, provider account, entity and book. Multiple streams may share an Item credential
reference; that reference grants no bookkeeping authority. Each approved
provider account maps explicitly to one stable internal source-account ID and ledger
account, with binding revision and consent provenance. Never infer ownership from
mask, name, institution or public-token contents. One Item cannot fan out into
multiple LLCs/books in this first profile. A separate reviewed Link/connection is
required for another ownership scope; detect suspected duplicate Items and require
an explicit overlap decision before activation, not silent coalescing.

An authenticated authorized user begins a durable Link attempt naming the intended
scope. Server creates the restricted Link token; browser holds temporary Link/public
tokens in memory only. Bind the return to the initiating principal, attempt, scope,
expiry and anti-replay nonce; reauthorize before server token exchange and activation.
Backend validates actual Item/account metadata and the approved binding manifest;
client Link metadata is not ownership evidence. Unmapped/unsupported accounts remain outside acquisition; they cannot activate a
stream until separately bound under an accepted profile. Account metadata may be
used for owner selection, but its transaction payload is not collected.

Application DB stores only opaque credential references/version, never access tokens.
Infisical owns provider client credentials and per-connection token material; workload
identity gets least-privilege access to the needed references. No token enters URLs,
queue bodies, logs, browser storage, fixtures or issues. Provisioning and atomic
reference-publication need a separate secret-store consistency design. A failed
exchange/rotation acknowledgement leaves `credential_recovery_required`; do not
blindly repeat a potentially non-idempotent provider operation or claim connected.

Proposed states: `link_pending`, `binding_review`, `active`, `reauthorization_required`,
`paused`, `credential_recovery_required`, `disconnect_pending`, `disconnected`. Update mode repairs the existing Item
where possible; verify identity/account manifest before resuming its checkpoint.
New Item means a new acquisition generation and overlap review, never assumed ID
continuity. Item credentials have a shared lifecycle distinct from account streams:
rotation atomically pauses/fences **all sibling streams**, writes the replacement
securely, then publishes one consistent credential reference/version to the group.
Resume each authorized stream only after replacement verification. A credential-only
change preserves provider observation identities; checkpoint adoption requires
verified same Item/account/filter, never token identity alone. An uncertain rotation
keeps all siblings in credential recovery.

Disconnecting one account disables/fences only its stream and does not call
`/item/remove`. Removing the entire Item requires explicit approval covering all
affected bindings, atomically disables/fences every sibling locally, and separately
tracks provider removal until confirmed. Lost removal responses stay unresolved,
not falsely deleted. There is no cross-account surprise revocation from disconnect.
Removing provider access never deletes posted journals or their evidence links.

Current SQL grants are read/prepare/post/close; connect/import/sync capabilities
are **proposed**, not implemented permissions. Separate grants/consent are required
for connection management and source acquisition. Each call and publication rechecks
its account-specific connection, consent, entity/book/source-account grant and
stream generation. Revoking one binding pauses/fences that stream; revoking Item
consent or its controlling principal pauses all associated streams. Other authorized
account streams need not stop for a single-account binding revocation.

New accounts remain unbound and are not automatically synced. A filtered response
containing any different/unapproved account fails closed: discard that payload and
staging, do not advance cursor, expose only a safe remediation code. The worker
cannot accept a different account filter from a queue payload or caller parameter.

## Acquisition, staging and complete-update publication

Use `/transactions/sync` with fixed `options.account_id` for each approved stream.
Choose this over Item-wide sync because permission revocation and cursor publication
can stay within one account. Cost: more requests/checkpoints for multiple accounts;
measure during qualification. PI01 documents separate filtered cursors. Stream
identity includes provider, environment, Item, exact account filter, connection
ID and generation. Never implicitly reuse a cursor across changed identity or
configuration. Credential-only rotation may explicitly adopt the existing checkpoint
after same Item/account/filter verification under a new fence; switching between
Item-wide and filtered modes or changing Item/account starts a new generation with
a null cursor and explicit overlap review. The selected implementation supports filtered mode only.

Start historical sync without a cursor; no migration-only `now` shortcut. Proposed
requested history is 90 days at Item initialization, recorded once; availability
still depends on product/institution. Empty initialization remains `initializing`,
not “zero transactions” or “all history complete.” Track initial/historical readiness
and provider last-successful-update separately from our per-stream publication time.

Align with [background work](background-work.md): a durable operation captures
`connection_id`, `connection_generation`, `target_generation` and current checkpoint.
`requested_generation` increments for verified webhook, scheduled or manual wakeups;
`completed_generation` advances only with complete acquisition publication. The
connection `fence_token` is separate from publisher `dispatch_generation`. Queue
messages contain operation/delivery IDs only; scope, cursors and credential references
are loaded internally. These are **new proposed work records**, not an extension of
`ledger.interactive_command`'s setup/post gate or immutable `journal.posted` payloads.

1. Claim connection lease/fence in a short transaction. Capture committed starting
   cursor `C0`, binding manifest and reference versions; create a private staging set.
2. Call sync outside SQL transactions. Preserve raw number tokens and page order,
   protected response provenance/digest, next cursor and `has_more`. Stage each valid
   page durably under current fence; staging is invisible to review/accounting APIs.
3. A successful bounded run may checkpoint **staging continuation**, not publish it.
   A new lease may adopt a valid incomplete set after rechecking generation/C0 and
   acquire a new fence; old worker cannot stage or commit. On page failure, including
   pagination mutation or ambiguous page delivery, abandon that attempt and restart
   the complete update from C0. Never restart only the failed page or mix attempts.
4. Only after `has_more=false`, validate the whole update, including cross-page
   pending lineage and the exact approved account. In one SQL transaction, compare-and-
   swap C0/generation/fence/lease and current authorization; publish immutable source
   revisions/tombstones and review exceptions, final cursor, completion receipt and
   `completed_generation=target_generation`. Update `source_revision`, **not** the
   ledger book revision. Incomplete/invalid sets advance neither public data nor cursor.
5. If requested_generation exceeds completed_generation, persist another bounded
   continuation/wakeup intent atomically. Repeated delivery resolves the receipt;
   commit-ack loss must not replay source effects. New notification during commit
   cannot be erased by an unqualified “sync done” flag.

Proposed engineering bounds (not provider guarantees): sync page size `count=500`; response
2 MiB; call 20s; run 60s/at most 10 pages; lease 90s renewed at 30s; at most 100 pages,
50,000 changes and 50 MiB per complete set; staging lifetime 24h. The #63 handler owns
its tighter transaction/deadline limits. Cap/expiry produces durable `limit_exceeded`
or `staging_expired`, no partial publication; authorized operator review is required
to change bounds. A huge history cannot be “solved” by advancing an incomplete cursor.
Unknown enum/schema or financially material unmapped field holds the set for adapter
review. Additive ignorable metadata must be explicitly allowlisted by parser version.
Do not interpret arbitrary provider URLs or fetch merchant images/websites.

## Observation identity, values and source changes

Durable provider identity is case-sensitive `(provider, environment, item_id,
account_id, transaction_id)`, linked to the stable source-account binding. It excludes
lease, run, filter, credential and connection generation: rotating credentials or
reacquiring history must not manufacture new purchases. A new Item retains distinct
provider identity and needs explicit overlap/continuity review. An
observation revision also names acquisition set/position, parser/API versions and
original value provenance. Same completed acquisition replay is idempotent; same
provider transaction later modified creates a new revision, not a conflicting second
purchase. Preserve all material changes. Ambiguous repeated IDs/change combinations
within a complete set require review rather than invented precedence.

| Provider value | Proposed internal meaning |
| --- | --- |
| JSON `amount` | Decode the original numeric token losslessly before any SDK float conversion. Positive = outflow, negative = inflow; store unsigned exact magnitude and original signed lexeme. Decimal/exponent notation converts by exact coefficient/scale arithmetic, never floating point. Reject >128-byte numeric tokens and exponent outside[-100,100] before expansion; then enforce numeric(38,9) bounds and currency scale. Zero/negative zero retained as review-only observations, not ordinary positive journal lines. |
| `iso_currency_code` / unofficial code | Bind valid ISO code/date to installed currency version. Missing/unsupported/unofficial currency prevents this profile's complete-set publication and records a safe exception; never default USD or sum currencies. |
| `date`, `authorized_date`, optional datetimes | Preserve source posted/pending date separately from authorization date and any supplied timestamp. No local-midnight conversion or substitution of authorized date for accounting date. User later chooses accounting date under book policy. |
| `pending`, `pending_transaction_id` | Source state and explicit lineage only. Pending is ineligible for posting; posted replacement is a distinct ID and must be reviewed at its final amount. A null link on an ordinary directly posted transaction is valid and remains review-eligible. Only suspected but unresolved replacement creates a lineage exception; never guess linkage from equal amounts/names. |
| added / modified / removed | Append new source revision or tombstone, maintain current-source projection and invalidate stale unposted review suggestions. Existing journal and source-to-journal linkage remain immutable; changed/removed evidence on a posted journal creates an exception for explicit correction/reversal review. |
| description/category/enrichment | Inert source text and suggestions, never authority to choose expense/account or execute code. No automatic classification-to-post action. |

Fictional fixed expectations: `12.50` USD → outflow magnitude`12.50`;
`-1.255e2` USD → inflow`125.5` (display125.50); JPY`125` and KWD`1.005` preserve
units. USD`0.001` is unsupported precision, never rounded to a cent. A pending
P 10.00 removed on page 1 and posted Q 12.00 linked to P on page 2 becomes one pending
history plus one eligible posted source, not two purchases. If Q later changes in the same USD currency to
13.00 after a 12.00 USD journal, retain the 12.00 journal and open a 1.00 discrepancy for
review; this does not authorize an automatic 1.00 adjusting entry.

## CSV overlap and accounting boundary

CSV and Plaid observations keep distinct adapter identities even for the same
stable source account. The shared-core key is proposed as
`(entity_id, stable_source_account_id, adapter_namespace, external_entry_id)`; this is a required reviewed
extension of CSV's existing three-part proposal, not a silent change to its file
format. CSV maps to its fixed profile/source namespace and original entry ID;
Plaid maps to provider/environment/Item/account identity and transaction ID, excluding
run/filter/credential generations.
Changing namespaces cannot bypass overlap review or reset dedup tombstones. Record explicit match decisions with actor/time, evidence,
source revisions and one bookkeeping-event group. Amount/date/description equality
is only a candidate, never a dedup key. Two identical coffee purchases with distinct
provider IDs stay two events. A CSV row C and Plaid row P proven to represent one
purchase link to one group without discarding either provenance chain. If C already
has a journal, matching P links evidence to that event and creates no second journal.
Contradictions or uncertain matching block automatic event linkage; separate real
purchases remain separate. Reconnection IDs get the same explicit overlap treatment.

The proposed core owns source revisions, matching decisions, durable drafts and
accounting links; adapters own acquisition and mapping. Source imports never invoke
posting. Draft approval executes reviewed ledger commands later with current ledger
revision/permissions; source revision changes invalidate stale source-based approval.
API/SQL contracts for that boundary are a prerequisite, not implied implemented behavior.

## Verified ingress, retries and privacy

Propose a separately deployed public webhook ingress with signature verification
and only durable wakeup/lifecycle-write authority. The [private API bridge](private-api-bridge.md)
and application Cloud Run API remain private; do not expose the full API to receive
Plaid callbacks. No webhook endpoint or public ingress is created in this slice.
Use a dedicated Plaid webhook verifier, distinct from end-user auth. Require a
single bounded Plaid-Verification JWT, ES256, trusted Plaid-fetched kid/JWK and
valid signature; validate iat age<=5min, proposed future skew<=60s, key metadata and
exact raw-body SHA256 in constant time. Do not deserialize/reserialize before hashing.
Fetch keys only from fixed provider endpoint, never JWT-supplied URLs; bound/cache
keys by kid/environment, reject expired keys, and rate-limit cache misses.
Proposed ingress bounds: 64 KiB body / 8 KiB JWT, strict JSON and approved event schemas.

After verification, resolve environment/Item to its approved active account streams
and persist a scoped delivery receipt plus requested-generation increments/outbox
intents for those streams before 2xx. No unbound account is activated by notification. Use a verified signed-delivery fingerprint for identical replay; different
valid notifications may coalesce into the same outstanding work without losing the
generation signal. No assumption of a provider event-ID field. Unknown/disabled scope
fails with a uniform denial (including an Item with no eligible stream); bad signature/body/age never schedules work. Persistence
failure returns 503. A webhook means “check source,” never a journal or guaranteed
real-time balance. Ignore legacy transaction notifications for sync business logic;
verified lifecycle notifications may mark reauthorization without granting access.

Proposed missed-notification catch-up every 6h and user retry both request the same
worker; neither forces paid refresh. Provider 429/transient failures use bounded
exponential backoff/jitter, Retry-After where supplied, and #63 attempt ceilings.
Consent/login errors pause for user action; unsupported data/limits quarantine for
operator review. Do not hammer a failed connection. UI separately shows last provider
update, last complete publication, initializing, stale (>24h proposed), retrying,
reauthorization-required, disconnected and review exceptions; never “live/synced”
solely because a webhook/task was acknowledged.

Raw authorized staging is private encrypted storage, no public URL/log/body telemetry;
retain no unauthorized account payload. Proposed expiry 24h for abandoned staging,
30 days published raw evidence; normalized revisions, dedup receipts and journal
provenance persist for the approved dataset lifetime. Raw deletion is explicit and
does not free identities for reimport. Provider disconnect, token deletion, source
retention and ledger retention are different actions. Live owner/audience/retention,
legal holds, backup erasure, residency and incident policy remain approval gates;
synthetic proposals are not statutory accounting-retention advice.

## Planned evidence and implementation handoff

All test names are **future requirements**, not executed PASS claims.

| Proposed evidence | Required outcome |
| --- | --- |
| `TestPlaidConnectionLifecycle` | Wrong actor/scope, expired/replayed Link return, mixed ownership and unsupported account reject; reconnect/rotation uncertainty pauses; local disconnect fences calls before provider removal completes. |
| `TestPlaidExactMoney` | Fixed examples above preserve cents, exponent lexemes and 0/2/3 decimals; SDK float path prohibited; overflow/unofficial currency/unknown material field blocks whole set. |
| `TestPlaidCompleteUpdate` | Two-page pending replacement publishes once; mutation or failed page restarts C0; staging pause/crash resumes safely; each cap+1 and expiry leave public checkpoint unchanged. |
| `TestPlaidPublicationAtomicity` | Failure after each source/tombstone/receipt/cursor write rolls back; lost commit acknowledgement resolves same set; source revision advances but ledger book revision/journal count do not. |
| `TestPlaidStreamAndCredentials` | Separate filtered cursors never cross accounts; mode/Item changes start null; credential rotation preserves observation identity and fences every sibling; single-account disconnect never removes the shared Item; Item removal requires all-affected approval. |
| `TestPlaidFenceAndWakeup` | Lease expiry, old connection generation and account revocation reject old calls/commits; webhook during completion causes another requested generation; no stranded catch-up. |
| `TestPlaidWebhook` | Valid signature/body passes; tampered body, wrong algorithm/kid, stale/future token, expired key, size overflow and unknown Item deny without work; only dedicated webhook ingress is public, app API remains private; duplicate/reordered/missed notifications converge via worker/catch-up. |
| `TestPlaidSourceLifecycle` | Ordinary posted row with null pending link remains review-eligible; unlinked pending hold removal creates no journal; modification/removal after posting creates exception, preserves journal; identical separate purchases stay separate. |
| `TestCrossAdapterOverlap` | CSV/Plaid and reconnect overlap need explicit scoped match; existing posted event never gets a duplicate journal; contradictory evidence remains unresolved. |
| `TestProviderPrivacyRetention` | No tokens/payloads in queue/log/error; denied scope returns no evidence; raw deletion retains dedup/link history; disconnect is not history deletion. |

Default future tests use original synthetic responses and locally generated test
signatures, no provider credentials. Separately authorized Plaid Sandbox qualification
may test provider behavior but is neither default CI nor a live-bank proof. Acceptance
requires owner profile/standards/consent/retention decisions; #63/#7 work records,
lease/outbox and handler implementation; #56/#62 credential/identity interfaces;
#8 source/draft/matching SQL/API; #50/#52 connection/review UI; #13 provider/cutover
approval. No DDL or adapter should begin by assuming these proposed interfaces exist.

## Implemented follow-on contracts

The proposal above is historical design context. Migrations 12–14 and owned
adapters implement bounded subsets; current authority lives in
[sync SQL](provider-sync-sql.md), [connection SQL](connection-sql.md),
[operations SQL](connection-operations-sql.md), [acquisition](acquisition-runner.md)
and [verified webhooks](provider-webhooks.md). Use their named executed evidence,
not this proposal's future-test table, to assess readiness. The #103 verifier
normalizes equivalent ECDSA signatures for receipt identity after standard
signature verification; exact original body bytes still determine its digest.
Transport dispatch, timed scheduling and actual Sandbox/live acceptance remain
separate gates. No posted journal follows from a notification or sync request.
