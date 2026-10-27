# One-account Production pilot mandate — #337

Status: PROPOSED2026-10-27; owner selections and security review pending. This is
a concrete decision proposal, not a provisioning, spend, credential or live-data
approval. #315/PR348 merged as6be6c48 after all five CI checks passed. #316 offline source preparation may proceed with fictional targets; actual
provisioning/activation requires this accepted decision. Original review source maineaab6ee.

## Proposed scope and owner selections

| Decision | Proposed value | Acceptance still required |
| --- | --- | --- |
| Isolation | Separate Production GCP project, same Go/PostgreSQL/Next architecture | Owner accept additional project/SQL resource cost; no creation yet |
| Project | `jayburd-bookkeeping-prod`, region `us-central1` | Actual availability, billing account and numeric project number; proposed name is not a provisioned target |
| Account | One supported US Transactions account, one personal entity/book | Owner selects privately in Plaid; verify returned type/currency and supported capability before import |
| Client | Desktop browser only | Exact browser/platform and institution/OAuth applicability before live use |
| History | Request90 days; reconcile all actually returned dates and gaps | Owner selects reconciliation interval; request does not guarantee90-day availability |
| Connection window | Seven days from separately approved activation | Exact UTC expiry in manifest; no automatic extension |
| Bookkeeping | Owner-reviewed manual categories/journals; no automatic rules initially | Owner accepts book accounting policy, currency, period and post permission; no default Sandbox policy |
| Bank authority | Transactions read access; no transfers/payments/money movement | Actual Plaid product/institution/account capability verified |
| Cost | Owner target approximately USD25/month for GCP only, recorded2026-10-27 | Costed candidate below; target is not an enforced cap or new-resource purchase approval |
| Incident owner | Repository owner Justin Napolitano | Owner accepts response responsibility and private contact channel |

User direction to complete the prerequisites authorizes source/testing/planning;
it does not settle missing deployment/policy decisions. Bank selection is not
a prerequisite for offline release software. Actual account capability and book
policy are verified in the private connection/setup flow. Do not copy secrets or financial
records into an issue or this document.

## Budget candidate and assumptions

Owner target: approximately USD25/month for GCP only; owner explicitly excludes
Plaid costs. For planning, include ongoing Sandbox hosting in that GCP total.
Actual cloud usage and provider fees remain unverified.
PP11 records public us-central1 prices, reviewed2026-10-27; 730 hours/month,
USD list prices, no discounts/taxes/network assumptions:

| SQL candidate | Monthly arithmetic | Consequence |
| --- | --- | --- |
| Existing checked-in dev db-g1-small | 730 ×0.035 =25.55 compute alone | Already above target before storage; source is not actual running-tier evidence |
| One db-f1-micro | 730 ×0.0105 =7.665 compute; 10GiB SSD≈1.70; total backup-used allowance10GiB≈0.80 | Approximately10.17 before other services; shared CPU has no Cloud SQL SLA |
| Two micro instances | Twice the preceding allowance≈20.33 | Leaves≈4.67 for all other usage; too little margin to promise25 total |

Preferred candidate for review: isolated Production micro, scale-to-zero Cloud
Run services, and Sandbox SQL operated only during bounded test sessions. This
requires explicit review of dev availability and real workload capacity; no
resource has been changed. Stopped Sandbox storage/backups still cost money.
Keeping the current dev tier always on does not meet the planning target.
Alternatively keep both micro databases running, accepting less headroom and
shared-CPU limitations. Do not remove backups or IAM isolation to force a price.

Cloud Run's billing-account-wide free allowance can help low usage but is not a
per-project guarantee. Reserve headroom for Run, KMS, secrets, storage, images,
builds, logging, restore rehearsals and network. Proposed alert thresholds15/20/25
notify the owner; they do not impose a hard cap. Verify actual service quantities,
backup usage and total bill before accepting the candidate. The read-only sizing
attempt could not run because gcloud was unavailable; no live tier/bill verified.
Plaid plan and actual fees require private Dashboard confirmation: free Trial
eligibility is not assumed, especially for an existing Production-approved app.

Acceptance evidence: reviewed itemized estimate including retained Sandbox costs,
actual nonsecret sizing readback, bounded synthetic capacity/recovery results,
and owner decision on test-environment availability. Confirm provider pricing
separately; Plaid fees do not consume the USD25 GCP target.
This source-only proposal authorizes no spend, resize, shutdown or deployment.

## Proposed target and capability separation

One Production Cloud SQL PostgreSQL17 instance `jayburd-prod`, database
`bookkeeping`, owns the existing domain schemas. No microservices, extra application
database technology or schema-per-entity design. Separate Terraform state from
development; no dev service, backup, raw evidence, secret or ciphertext reused.
Cloud SQL tier/availability/storage and restore target are selected within the
accepted cost ceiling before apply; a restore rehearsal resource is temporary
and requires its own bounded cost/duration, not an undisclosed standing database.

| Capability | Proposed identity/resource boundary |
| --- | --- |
| Frontend | `jayburd-prod-web`; calls private `jayburd-prod-api` only; separate Identity Platform project/verified owner/MFA and canonical origin |
| API | `jayburd-prod-api` IAM SQL login; owner/book scope before provider calls; explicit bank/Link keys and no worker/owner/migrator role |
| Acquisition | `jayburd-prod-sync` IAM SQL login; one authorized stream, exact Item/credential/lease context and bank unwrap; no post/owner role |
| Publisher/callers | Separate `jayburd-prod-publisher`, `jayburd-prod-scheduler`, `jayburd-prod-task-invoker`; exact service audiences/queue, scheduling paused until explicit activation |
| Receiver | `jayburd-prod-webhook`, canonical project-number URL; signed metadata receipt/wakeup only, no bank unwrap/claim/post authority |
| Operator/migration | Separate restricted operator and migration identities; reviewed forward migration/status and actual role bootstrap; no app-start migrations |
| Secrets | Infisical remains owner. Reported `plaid_production_secret` entry is already done; owner delivers to exact numeric `jayburd-prod-bank-sync-client-*` GSM versions with redacted mapping evidence |
| Wrapping | Proposed KMS keyring `jayburd-prod`, distinct `bank-credentials` and `hosted-link` keys in Production project; exact versions and environment AAD, no dev or mutable alias |
| Evidence | Private `${project}-jayburd-prod-acquisition-evidence`; separate `${project}-jayburd-prod-credential-registry` compatibility metadata; no public ACL, raw artifacts or dev fallback |

Role permissions must be resource-specific, not project-wide Editor/default
service identities. Workload identity/metadata tokens only; no static service
account keys. Names above are proposed configuration, not effective IAM evidence.
Production scope and controlled owner/worker setup precede registration. The
bookless/self-service redesign remains237; initial registration does not wait for
the entire rules/corrections roadmap333–336.

## Proposed private retention and recovery

Proposal for owner/security/accounting review, not a legal retention opinion:
keep imported records, immutable journals and their required provenance through
the pilot and reconciliation; set the final ledger/provenance retention before
real posting. No auto-delete rule may break source-to-journal audit links.
Raw acquisition evidence: proposed30-day minimum for reconciliation/replay,
extended when required by retained posted provenance. Sensitive payloads/tokens
never enter CI/public evidence/diagnostic logs. Proposed operational log retention
30 days with identifiers/metadata minimized; public engineering receipts contain
only redacted outcomes and immutable source/image/config hashes.

Proposed standard daily SQL backups: eight retained; PITR logs seven days, exact
US location explicit. Restore target RPO<=24h/RTO<=4h is a rehearsal target,
not an achieved guarantee. Independently verify restore, ledger balance, imported
set and replay fences before activation; restore into isolated private scope,
with queues paused and provider credentials disabled until exact comparison.

Deletion request: first pause acquisition/disable scope, then explicitly remove
the owned Plaid Item, retain its redacted result, and review which private raw,
credential, primary and backup copies can be removed under the accepted policy.
Local disconnect alone does not remove the provider Item or erase ciphertext.
Retained backups and immutable bookkeeping may outlive active access. Record
actual deletion/expiry dates, never promise instant erasure from all copies.

## Activation, stop and rollback

Before activation316 supplies exact source/build/schema/image/IAM/SQL/secret/
origin/receiver receipts, finite hashes, reviewer, cost estimate and tests. Default
no profile/null/unknown/mixed/expired state denies provider execution. API/web
can preserve currently authorized local bookkeeping/recovery after expiry.

Stop sequence: owner pauses scheduler and delivery; disables provider scope and
current stream acquisition/ingress; denies new credentials/provider work; checks
inflight lease/queue outcomes and current SQL status. Revoking local fetch does
not destroy posted journals or silently prohibit authorized local bookkeeping.
For a suspected credential compromise, owner performs scoped secret/key/provider
revocation and private incident handling; no public diagnostic payloads.

Rollback changes only to an exact previously accepted compatible Production
source/image/schema/config profile. Initial pilot has no previous Production
receipt: rollback is unavailable; use stop/forward repair. Never down-migrate
retained Production provenance or restore/relabel Sandbox ciphertext. Backup
restore is a distinct reviewed operation, not an ordinary image rollback.

## Required disposition before activation

| Gate | Current disposition | Required next evidence |
| --- | --- | --- |
|315 composed offline path | PASS, merged348/6be6c48, all five exact-head CI checks PASS | Retain exact proof; includes346 fix, not live approval |
|302/303 signed hosted delivery | OPEN | Exact Sandbox receiver identity/bootstrap/signature → durable demand → publication/review evidence |
|275/320 owned reconnect | Source implemented, actual hosted proof OPEN | Owned reset/reconnect/reload/expiry with unchanged credential and one catchup cause |
|274 Launch Center | Personalized review PARTIAL | Current app-specific Production requirements explicitly mapped and cleared |
|276 OAuth/platform | Desktop conditional; mobile/native excluded | Chosen bank determines desktop OAuth cases; no mobile support claim |
|88 equality/recovery | Applicable hosted rows incomplete | Independent complete-set Sandbox comparison and no unintended journals, exact versions/receipts |
|1/10/90 policy/reconciliation | Real-account book policy and private interval pending | Owner accepts recognition/currency/period and complete transaction/balance comparison |
|62/89 controls | Live scoped evidence/disposition pending | Backup/restore/kill/cost and bounded-load/recovery evidence appropriate to one account |
|316 finite package | Not implemented/approved | Compatible Production delivery, reviewed exact bytes and actual metadata |

No OPEN row is waived by this proposal. N/A requires a reviewed scope rationale;
implementation/synthetic PASS is not hosted acceptance. Record exact evidence and
owner/security acceptance in337 before provisioning/activation. See
[316 execution plan](../plans/production-release.md).

## Sources, standards and alternatives

PP01–11 in production-pilot-sources.md distinguish vendor mechanisms
from this project's proposed resource, retention, cost and authority choices.
Cloud Run identity and Cloud SQL backup/billing selections reviewed2026-10-26;
PP11 adds public list-price sizing, not an account-specific quote. No account,
actual project, ISO normative or jurisdictional retention
rule reviewed. No ISO20022 message fits operational release/IAM metadata;
existing Plaid API2020-09-14, original-unit ISO4217 and ISO21378 limits remain.

Separate Production resources improve reviewable origin/IAM/evidence boundaries
but add standing SQL costs. Reusing development project/resources avoids some
cost but needs equivalent separation proof and risks shared IAM/data/reset paths;
not selected by this proposal. New database technologies/services are rejected.
Alerts-only budgets notify rather than enforce a spending ceiling; combine
bounded capacity, finite activation, alerts and owner stop controls. Confirm any
service-specific spend-cap support before claiming it, rather than assuming it.

STRIDE: spoofing/escalation(high) → verified owner, dedicated IAM/SQL/keys/current
scope; tampering/replay(high) → hashes, provenance, immutable receipts/journals;
disclosure(high) → isolated private evidence/backups and no raw logs; denial/cost
(medium/high) → one stream, finite approval, paused defaults and bounded capacity;
repudiation(high) → approved policy and redacted action/restore/stop evidence.
Residual actual security/configuration/restore/retention risks require human review.
