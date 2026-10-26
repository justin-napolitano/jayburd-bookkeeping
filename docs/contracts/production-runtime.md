# Production credential/runtime composition — #313

Coordinator source decision, 2026-10-26, before implementation. Builds on merged
#311/#312; no deployment/resource/pilot decision is made for #337. No SQL, browser
or financial message changes. Proprietary Plaid2020-09-14 and existing ISO4217
money semantics remain unchanged; operational key/AAD/config metadata has no
applicable registered ISO20022 message. ISO21378 access limits remain unchanged.

## Closed metadata and authority

Legacy constructors/omitted environment remain Sandbox-only. Explicit Production
managed configuration uses `provider-environment=production`, numeric exact
static GSM versions named jayburd-prod-bank-sync-client-id/client-secret, prod
Infisical namespace and dedicated registry/evidence names. Infisical remains the
secret owner; this task neither reads nor delivers the reported
`plaid_production_secret`. Its source version/destination mapping and least-privilege
delivery must be owner-reviewed in #316. Secret contents cannot prove their own
environment; exact approved references and delivery evidence are required.

Production requires explicit bank/link KMS cryptoKey resource names in the runtime
project (us-central1); no dev keyring, mutable version alias or arbitrary URL.
API/worker Cloud SQL IAM logins use dedicated jayburd-prod-api/sync/publisher names;
existing development identities stay unchanged. These are source naming profiles,
not provisioned or approved resources. Production mode rejects development instance,
registry/evidence/key/static references before dependency I/O. Explicit finite
approval fences every outbound dependency request. Cold expired API startup retains
SQL-only status/pause/disconnect/receipt recovery with zero secret/KMS/provider I/O.

Before begin and every provider operation, current SQL
connection_provider_context authorizes actor/entity/book and must agree with the
immutable runtime environment, data origin and both exact configured key resources.
No context caching or browser-selected environment. Production uses existing
configured-book enrollment, not creator onboarding that could manufacture an
unconfigured Production scope. SQL owns current permissions, consent and revision
fences; comparison is runtime anti-confusion, not duplicated permission logic.
Each private attempt must match the runtime profile before provider/credential use.

Bank AAD retains existing fixed tuple/profile and now accepts the two closed
environments. Sandbox ciphertext/AAD stays byte-compatible; Production tuples differ
by environment. Production Link AAD appends the explicit environment to the existing
actor/entity/book/attempt tuple; Sandbox tuple stays unchanged. Bank and Link use
separate configured wrapping capabilities and exact numeric key-version checks.
Addressed storage and current SQL authority precede unwrap; worker expected
environment/Item/reference/version must match before KMS/static credentials.
No relabeling of existing envelopes, new cipher, plaintext SQL or credential logging.

Production notification/reconnect remains rejected here until #314's separate
matched receiver/key/receipt capability is implemented and qualified. No arbitrary
callback option, Sandbox receiver or hosted-repair opt-in in Production. Empty
receiver permits source construction testing only and is not release-ready.

## Evidence and alternatives

PP08 in production-pilot-sources records selected Google KMS AAD and Secret Manager
version-access sections and refreshed Plaid host/storage sections. Vendor facts:
different AAD cannot decrypt; version aliases exist; Items cannot move environments.
Our exact version pins, separate names, per-request preflight and immutable profile
are engineering controls, not vendor policy or security signoff. A secret/URL swap
was rejected because it leaves SQL/key/evidence identities mixed. A duplicate
credential implementation was rejected to preserve current exact recovery behavior.

STRIDE: profile/key/reference substitution (high) -> current SQL preflight, closed
runtime, exact immutable versions and AAD; unauthorized unwrap (high) -> current
addressed SQL scope and worker lease; disclosure (high) -> memory-only secrets,
fixed errors and no raw diagnostics; expired authority (high) -> outbound deadline,
cold inert startup; source mixing (high) -> dedicated IAM/evidence naming and
explicit transport environment. Residual actual IAM/delivery/retention/restore and
pilot acceptance remains #337/#316/#90. Human sensitive-source review before merge.

Acceptance: TestProductionCredentialEnvironmentBinding; Production API/worker
configuration mismatch/expiry/static-only startup tests; private-context-before-begin
and wrong-origin/key tests; actual restricted PostgreSQL Production credential
roundtrip/current-denial tests; unchanged Sandbox credential/recovery regressions.
Synthetic-only default tests, core/race/build, migration/ledger regression and
app-integration before handoff. No new dependencies or migration.
