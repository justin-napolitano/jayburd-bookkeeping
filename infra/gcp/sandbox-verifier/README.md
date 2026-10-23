# Finite Sandbox comparator

Owner-operated, default disabled. Contract:
`docs/contracts/hosted-sandbox-verifier.md`; delivery runbook:
`docs/runbooks/hosted-sandbox-verifier.md`. Applying this root never executes a job.

Explicit `enabled=true, run=null` provisions only the isolated identity, IAM SQL
login and private synthetic-input bucket. Operator then runs reviewed
`db/bootstrap/verifier-login.sql` with `project_id=jayburd-bookkeeping-dev`.
It grants no business mandate. Inputs, secrets and financial/provider data must
never be Terraform variables, resources, outputs or state.

For a separately reviewed finite run, supply `run={image=<exact trusted worker
digest>,run_id=verify-<bounded unique ID>}`. Upload only the three reviewed files
create-only under that directory after a valid <=60-minute mandate exists.
Execute the fixed job with no overrides. Capture only exact execution metadata
and the closed comparator result. Revoke the mandate, delete the exact uploaded
object generations, and set run=null to remove the idle executable job. Keep the
isolated platform identity without granting any app/provider/DDL/posting access.

Backend: existing `jayburd-bookkeeping-dev-jayburd-dev-runtime-state` bucket,
prefix `sandbox-verifier`; never reuse another root's prefix. Owner supplies
short-lived platform authentication privately. Review only selected outcomes;
do not print/read state, plans, credentials or raw provider diagnostics.

Synthetic gate: `bash scripts/test-infra.sh sandbox-verifier`.

The explicit guarded owner apply entry point is scripts/apply-sandbox-verifier.sh;
no arguments performs no file/network/platform work. It requires the reviewed canonical metadata hash,
clean reviewed source equal to current remote main, and the fixed backend prefix.
The bootstrap JSON also removes the idle job after a run while retaining the
protected isolated identity/bucket. Do not disable enabled to bypass protected
resource cleanup. No command in this root implicitly runs a comparison.
