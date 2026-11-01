# Paused Production API/frontend

Candidate370; human review before apply. Uses existing shared foundation and
bookkeeping_prod, separate Production runtime state bucket/prefix and existing
Production identities. Never inspect/copy state, plans or owner credential files.
Default runtime=null creates no service, job or runtime IAM. Inputs contain only
public metadata: scanned exact api/web/worker digests, public Firebase browser
identifier, acknowledged numeric static versions and finite readback deadline.

API is private to Production frontend; web exposes the existing sign-in shell.
Apply foundation's exact firebaseauth.users.get role for revocation verification;
no user administration. Existing controlled principal enrollment is still required
in the Production database; sharing Identity Platform does not copy ledger access.
Both services min0/max1, CPU1/512Mi, idle billing/no boost. This is capacity control,
not an enforced USD25 cap. No SQL/schema/instance, worker, queue or scheduler here.

All connection metadata explicitly selects Production, but provider expiry is
fixed to2020-01-01T00:00:00Z. Existing cold expired-recovery composition avoids
secret/KMS/provider startup and preserves authorized local bookkeeping/recovery.
There is no provider-activation input. Legacy Infisical fields identify static
delivery lineage; they grant no API access to Infisical. Live pilot needs its own
reviewed activation package, receiver, worker SQL/runtime and qualification gates.

The one-shot /credential-check uses API identity and existing read grants to check
the two numeric GCP versions with CRC32C/name validation. One task, no retries,
no scheduler or new invoker/impersonation grant. It prints fixed status only; no
source lookup, value/length/hash diagnostic, provider or SQL call. CLI rejects
expired or more-than-seven-day authority before metadata access. Successful
readback is not source equality or working Plaid authentication. Run only after
exact worker image containing this new command passes trusted runtime/scan gates.

Reviewed apply must bind exact input bytes/hash and verify service image/identity/
origin, private API denial, schema/SQL role readiness, public auth configuration
and redacted readback result. API/web images built before370 can run the paused
profile, but a pre370 worker does not contain /credential-check. Do not deploy
mixed-source images or claim the full release bundle passed from component receipts.
See docs/contracts/paused-production-runtime.md and the rollout runbook.
