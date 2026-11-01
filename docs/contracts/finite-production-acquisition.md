# Finite Production acquisition delivery — #375

Decision candidate2026-11-01 before implementation. Reuse existing Go publisher,
consumer and signed receiver, approved shared instance/bookkeeping_prod, dedicated
Production identities/keys/static versions and existing private buckets. No new
application/schema/financial-message semantics. Selected source review: existing
production-runtime/source-sql/webhooks contracts, command flags, worker-runtime/
workers/webhook Terraform, strict webhook-login.sql and source35 ingress. PP13
records selected refreshed primary operational sources and access limits.

## Ownership, inputs and controls

Separate production-acquisition root with runtime=null by default. Foundation
retains existing identities/SQL users/buckets/secrets/KMS and their grants. New root
owns private publisher/consumer, exact queue/caller grants, always-paused scheduler,
dedicated signed receiver and explicitly selected receiver invocation binding.
Existing Production worker-runtime backend bucket/prefix stays separate from app
state. Never rename/import Sandbox resources. Both roots pin the known project
jayburd-bookkeeping-dev/9088794002/us-central1 and existing shared instance.

Inputs contain public metadata only: matching immutable Production worker image,
nonzero single binding UUID, canonical positive int63 static versions, explicit UTC
approval deadline and public_webhook_approved (default false). Reject expired or
beyond7days at plan validation; existing Go independently checks actual expiry
before startup/outbound calls and cached-key handling. Seven days is project pilot
policy, not a vendor rule. No secret/environment/state/plan files enter tests.

Publisher uses binding-v1 and fixed scheduler subject/email/audience; consumer uses
exact task-invoker subject/email/audience, Production SQL/keys/static versions and
evidence/registry prefixes. Only publisher gets queue enqueuer and actAs on the
one task identity. Queue dispatch/concurrency1 with bounded retries. Workers have
min0/max1, CPU1/512Mi, idle billing/no boost, timeout90s/concurrency1. Scheduler is
unconditionally paused, retry0, exact OIDC identity and private publisher route.
No input resumes it. Manual dispatch is a separately reviewed operation; reference
API documentation alone does not prove dispatch while paused.

Receiver overrides command /webhook, with explicit Production profile/static pins,
finite authority and existing signature/profile/request limits. Dedicated identity
has existing SQL/static-secret capability only. Private unless explicitly selected
public; allUsers then gets only this receiver's run.invoker. Cloud IAM cannot
identify Plaid: application signature verification remains the public boundary.
Actual source35 ingress and runtime role safety are mandatory before qualification.

## Explicit API pilot and operator bootstrap

Existing app root keeps null/inert and expired2020 bank defaults when pilot=null.
Separate explicit pilot object accepts only finite <=7day deadline and exactly
matching canonical Production receiver URL. No destination/identity/key/static
substitution or Hosted repair opt-in. Readback expiry alone never activates bank
access. Operator binds both roots to the same reviewed source/image/static/date
metadata; input validation does not establish owner approval or test completion.

Production webhook SQL adapts the strict existing bootstrap to exact shared project,
number/database and dedicated login. Retain all transitive/direct ACL, predefined
role, grant-option and unsafe capability checks; additionally deny Sandbox CONNECT.
One serializable transaction grants existing webhook capability/Production CONNECT,
sets bounded login/search_path, calls source35 ingress with immutable production
profile, and verifies actual desired enabled state on replay. No new SQL role/user,
migration or domain helper. Existing worker-logins.sql is a separately reviewed
operator stage; independently verify its exact targets and cross-DB denial.

## Evidence, alternatives and boundaries

Vendor facts: mocks simulate providers; plan-time clock supports time-sensitive
validation; manual Scheduler dispatch exists. Repository facts: explicit Production
Go/SQL profiles already implemented. Inference: distinct delivery roots can reuse
those controls; Terraform1.15.7/Google8.6.0 tests must validate the implementation.
Rejected Sandbox substitutions/default-active rollout because of mixed state and
authority. No second database/instance or extra standing architecture; capacity
controls do not guarantee USD25 total. STD-01: operational deployment/IAM metadata
has no applicable ISO20022 financial message; Plaid2020-09-14/ISO4217/original-date/
decimal semantics and existing audit/conformance access limits remain unchanged.

STRIDE: elevation/confusion(high) -> fixed profiles/resources/identities/current SQL
scope; spoofing/replay(high) -> signed receiver, current ingress and receipt fences;
disclosure(high) -> no raw logs/keys, private evidence and metadata identity;
availability/cost(medium/high) -> finite window, bounded capacity/retries and paused
scheduling. Actual signed delivery/reconnect/credential authentication/recovery/
retention/restore and one-account mandate still require observed dispositions.
Human source/security review before actual public/runtime access or operator grants.

Acceptance: mocked inert/private/public/paused/caller/profile/image/version/expiry/
capacity cases; real synthetic PG bootstrap safety/replay/revocation/login denials
plus Sandbox regression; core/infra/release and Production browser recovery gates.
Then current scanned images/static readback, actual caller/SQL/readiness/signature/
demand/review and stop/recovery proofs before owner-private bank link. No source
test or deployment alone completes the personal-account pilot.
