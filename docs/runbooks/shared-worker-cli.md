<!-- stage-runbook: true -->
# Shared Production worker CLI — #368

## Status and boundary

PROD368-CLI; fix/368-shared-worker fromed9a2fc. Source repair of the last consumer
command guard that unconditionally rejects the reviewed shared project number.
No SQL, IAM, deployment, dependency, financial contract or provider call change.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Discovery | cmd/worker parse rejects9088794002 although reviewed DB/runtime adapters accept the exact shared instance |
| Decision | Align CLI with existing matched project-ID/number profile; preserve all resource/caller guards |
| Regression | Before patch: valid shared profile rejected and shared-ID/foreign-number accepted; after patch all cases PASS |
| Validation | Focused cmd/worker and make check/test/build PASS |

## Current evidence

Before implementation: inspected internal/clouddb Config.ValidateEnvironment/
sharedProductionRuntime, connectionruntime.validateProduction, cmd/deliver
config.valid and the failing CLI guard against the reviewed PR361/362 context.
Before handoff, cross-checked production-runtime.md and the shared-production-sql.md
runtime/provider follow-ups. Existing
bibliography/Production source register applies; selected repository sections,
not a new vendor/source audit. These define the owner-selected shared instance
and distinct bookkeeping_prod/Production identity profile. The CLI's unconditional
numeric-project denial conflicts with that reviewed profile. Inference: require
the same matched known project-ID/number pair before startup. Alternative: remove
the denial entirely; rejected because a shared-ID/foreign-number configuration
could then reach inconsistent secret resources. Dedicated fictional Production
projects must remain compatible. Existing exact-instance guard, key policy,
secret numeric pins, caller and expiry behavior remain mandatory.

ISO20022 does not apply to operational CLI metadata. No financial-message, ISO4217
currency or ledger policy change. Acceptance: exact shared consumer configuration
accepted, both internally consistent mixed project-ID/number profiles rejected,
wrong shared instance and Sandbox names rejected; existing dedicated and Sandbox
controls retained. Tests call parse only and cannot reach metadata/SQL/provider I/O.

## Remaining work

Human security review and newly scanned exact worker build before consumer
rollout remain open. Existing scanned worker supports actual
static delivery, but does not establish corrected shared consumer startup.
Independent credential payload/runtime verification and hosted SQL/application
qualification remain separate #365/#316 work.

## PR handoff

Refs #368/#316/#365. No migration. Actual build/delivery evidence from the current
operations is recorded in production-build-operations.md without claiming this
unmerged CLI fix was present in those images.
Local gates used installed Go1.26.5/external temporary modfile; tracked1.26.10
unchanged. Synthetic tests used authorized local loopback; no cloud/provider call.
