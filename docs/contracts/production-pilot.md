# Proposed Production profile for one read-only account

## Status and purpose

#308 under #90, 2026-10-24. **Proposed; not activation or security approval.**
Owner goal: complete prerequisites, prove Sandbox/recovery, then connect one
personal account and reconcile imported transactions. The approved broader
connect-first redesign is outside this pilot's critical path. Current book-first
setup can be used after its applicable gates pass. No live account is connected
by this document, and no closed issue/deployment is live-bookkeeping approval.

[PP01–05 source review](../research/production-pilot-sources.md) separates vendor
facts from the engineering proposals below. Current adapters, encrypted-token
identity, source/Item SQL and signed webhook scope intentionally require Sandbox.
A Production secret alone cannot enable real accounts. PP01 also rules out moving
an existing test Item to Production: a new owned enrollment is required.

## Proposed pilot boundary

One verified owner, one explicitly named supported source account, one personal
entity/book, one US Transactions Item and desktop browser as the proposed first
client. Exact type/institution, statement currency/interval, deployment targets,
source consent lifetime, retention/deletion, incident contact, cost ceiling and
owner review must be recorded before actual activation. The current pending
account-type question is not an approval. No loan/card accounting policy is
inferred if the selected type changes; unsupported account types stay unbound.
Enforce one reviewed active source binding under the pilot grant. Other accounts
may be discovered within the owner's provider-approved inventory, but remain
unbound; their transactions are not acquired. Reject a second active binding or
foreign-account page atomically rather than treating “one account” as a UI hint.

Collect source observations for review only. Do not automatically classify,
prepare or post journals; no payment/money-movement product, Investments,
Liabilities or Transactions Refresh add-on is enabled by default. Intended initial
lookback is90 days, with supplied coverage explicitly recorded. Preserve exact
decimal strings and original currencies; no unrelated-currency totals. A history
hold/overflow/partial response is not complete acquisition and cannot advance a
cursor over discarded data. Posted book balances are not bank balances.

This is source-to-statement reconciliation, not an official close or accountant
approval. Relevant #1/#10 criteria and policy implications must be explicitly
dispositioned under #90; this proposal neither closes those issues nor silently
waives their existing gates. Full official-books/cutover remains #13/#58.

## Deployment alternatives and pending choice

Recommended proposal: isolated Production deployment using the existing Go/
PostgreSQL/Next and restricted-worker architecture, with separate workload/SQL
identities, credential/evidence resources, runtime approval and operator receipts.
Separate project/database provisioning and costs require reviewed acceptance;
this is not authority to add a database, service or Terraform root now.

Alternative: explicitly reviewed isolation inside existing infrastructure. It
must demonstrate equivalent source/credential/log/evidence/backup/role and
cleanup boundaries, address real-data classification and retain development
synthetic-only guarantees. Reusing the dev release with a changed secret/URL is
rejected. No architecture decision is accepted by this recommendation alone.

Do not rename or mutate existing Sandbox Items, ciphertext, environment values,
synthetic flags or financial evidence. In particular, never label real records as
synthetic to satisfy existing SQL. Real-data retention and deletion cannot inherit
test reset/cleanup assumptions. No Sandbox diagnostic/verifier can reach Production.

## Authority and integration matrix

| Operation / boundary | Required Production authority and denial |
| --- | --- |
| Runtime composition | Explicit reviewed environment enum/profile and exact deployment target; no ambient discovery, arbitrary endpoint or cross-environment fallback. One provider environment per runtime |
| User setup/Link | Existing verified issuer/subject/MFA/current principal and book authority; explicit finite source consent. Browser cannot select environment, provider endpoint, secret version, service identity or target callback |
| SQL enrollment/source/binding | Operator-approved runtime/provider scope plus existing owner/read/review grants. Immutable environment included in namespace/Item identity; current functions must deny a caller outside their environment capability. No direct table grants |
| Token storage | Infisical-owned static secret injection with reviewed numeric destination pins; encrypted per-Item tokens with correct environment in AAD and current SQL scope. Cross-environment unwrap/reference/read/write fails before returning plaintext |
| API/provider transport | Closed host/path allowlist and fixed reviewed API-Version; authorized credentials must match runtime environment. No redirect, Sandbox test endpoint or surprise optional product in Production |
| Worker acquisition | Current scoped source consent, generation/binding/lease/cursor checks and exact-money decoder; execution context, credential and runtime environments must agree. Acquisition commits no journal |
| Signed webhook | Environment-matched key service, cryptographic verification and durable SQL receipt; no unverified body/environment can select a credential or tenant. Owned Item/session repair remains #275; no generic login timestamp as proof |
| Status/disconnect/revoke | Current SQL-authorized reads and local pause/disconnect remain distinct from explicit Item-wide provider removal. Revoke fences queued/inflight authority, retains immutable provenance and records cleanup truthfully |
| Release/rollback | Exact source/images/config/approval/targets and accepted schema floor; separate finite activation, deny expired/mixed/fallback configurations. Old Sandbox image cannot be a compatible Production rollback merely because it starts |
| Comparison/statement | Owner supplies/reviews private records outside agent-visible output. Complete interval, opening/closing balance and exact transaction set under independent expectations; only redacted counts/outcomes retained. Pending/timing differences separately explained |

Each SQL/transport signature, role capability and field propagation must be
reviewed in its bounded child before implementation. Do not solve this with a
global boolean that weakens every Sandbox check. Preserve financial profile
version/provenance; explicitly decide whether the current usage profile is
compatible or needs a new version before DDL. Same semantic money fields do not
by themselves approve changed data provenance or authority.

## Standards applicability

Production remains a proprietary Plaid JSON financial exchange, not ISO20022.
Retain API-Version2020-09-14, exact amount/sign/date/state/identity mapping and
dated ISO4217 references from provider-ingestion, acquisition and liability
contracts. Required multicurrency scope is unchanged; a single-account fixture
is not a one-currency ledger. No new bank message or currency-reference version
is selected here. Profile compatibility/data-origin changes require the reviewed
mapping/version decision in the SQL/transport children. Internal release/IAM/AAD
routing is an operational authority boundary without a forced ISO message.
Audit/export ISO21378 applicability retains its existing review limits; no
normative conformance or accountant approval is claimed.

## Threat review and named acceptance

This bounded STRIDE review is engineering evidence, not independent security
signoff. Auth/secrets/restricted-data/exposure/runtime changes require human
security review before release.

| Threat | Control / proposed acceptance case |
| --- | --- |
| Environment or tenant spoofing (high) | Explicit runtime/SQL capability/current owner scope; TestProductionProfileCrossEnvironmentDeny and TestProductionProfilePrincipalDeny |
| Ciphertext/reference substitution (high) | Environment-bound AAD and addressed SQL authority; TestProductionCredentialEnvironmentBinding, no plaintext/secret in failure output |
| Tampered/replayed Item or event (high) | Signed matched-environment event, owned session proof, durable dedup/revision fences; TestProductionWebhookEnvironmentDeny and existing exact replay/repair cases |
| Financial disclosure or synthetic cleanup of real data (high) | Private Production resources/retention/least privilege, immutable origin, no fixture reset route; TestProductionRecordsNotSynthetic and reviewed backup/deletion receipts |
| Duplicate exchange/journal or false readiness (high) | One durable attempt, uncertain-result recovery, atomic publication and independent complete-set comparison; TestProductionProfileJourney, zero automatic journal/outbox assertions |
| Outage/overload/cost amplification (medium/high) | Accepted bounded workload/cost/consent limits, no automatic refresh add-on, pause/revoke/restore/kill switch; #64/#89/#58 exact fault evidence |

Proposed names are acceptance obligations, not existing tests. Synthetic fixtures
must drive both profiles with injected transports and real restricted PostgreSQL,
prove foreign credentials/cursors/Items/events/evidence never cross environments,
and retain current Sandbox regression expectations independently of the new code.
Run applicable migration/ledger/core/web/full-stack gates. No default CI makes
Production calls or injects real data. Hosted activation requires a separate
owner-reviewed exact manifest after applicable #88/#274/#275/#276 evidence.

## Decisions still needed before implementation/activation

Owner/security reviewer: accept deployment/data isolation; exact permitted
profile and role/retention threat controls. SQL/transport review: signatures,
capabilities and financial usage-profile compatibility. Qualification coordinator:
all applicable matrix rows and existing #90 policy/reconciliation/scale gates
have explicit evidence/disposition, never suite-name inference. Owner pilot:
account/type/institution/platform, private credentials, complete statement
interval, cost/duration/retention and final approval of the concrete activation.

The implementation children cover SQL authority, profile/transport, credential
and runtime composition, signed webhook/environment isolation, release preparation
and independent synthetic integration. Their IDs and current blockers live in
#308; claim no migration version or writer until the relevant decisions are accepted.

| Issue | Bounded delivery | Hard source prerequisites |
| --- | --- | --- |
| [#311](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/311) | SQL environment/data-origin/current authority and restricted functions | Accepted #308, exact reviewed SQL contract; sole SQL writer |
| [#312](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/312) | Closed provider profile, enrollment and sync transports | Accepted #308 |
| [#313](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/313) | Credential AAD/addressed storage and managed API/worker composition | #311/#312 |
| [#314](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/314) | Matched-environment key/event/receipt/wakeup | #311/#312; coordinate #271/#275 interfaces and exclusive files |
| [#315](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/315) | Independent offline Production-profile journey and cross-environment denials | #311/#312/#313/#314 |
| [#316](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/316) | Exact finite release/rollback/isolation package | #308/#311–#315; actual activation stays separately scoped in #90 |

These are native children/blockers of #90, not competing epics. All remain
unclaimed/blocked for implementation until their reviewed prerequisites hold.
SQL and Go profile work can be separate future leases after contract acceptance;
one SQL writer and coordinator ownership of shared profile/CLI remain required.
