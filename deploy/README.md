# Review a development release

A migration-only candidate is committed in [dev-release.json](dev-release.json).
It does not execute on merge. Review its source, digests and expiry using the
[candidate runbook](../docs/runbooks/migration-candidate.md); deployment federation
and exact-hash owner dispatch are separate steps. Follow the
[rollout contract](../docs/contracts/cloud-run-rollout.md).

## Migration-only bootstrap release

To gather #26 evidence before full application promotion, follow the
[migration-only contract](../docs/contracts/migration-release.md). Use the same
`deploy/dev-release.json` path and exact-hash `deploy-dev.yml` dispatch. Set
`operation` to `migrate`; omit `firebaseApiKey` and `previousDeployment`, and keep
only bootstrap/compatibility evidence. All three same-build image receipts remain
required; only the migrator image executes. Foundation #5 must be closed and #44
merged and ancestral to both the workflow and build commits.

Owner reviews the exact digest, source, bootstrap and compatibility evidence
before dispatch. This operation applies only the migration root, validates a
successful immutable execution, emits a migration receipt and exits. No runtime
root, frontend configuration or service smoke runs. It does not close #26 or
approve a later promote/rollback. Provider readback failure requires investigation
because SQL may already have committed. Never force dirty state or down-migrate.

## Read-only migration verification

Read-only database acceptance uses the same manifest shape as migration-only,
but `operation: verify` and a fresh image supporting `-action=status`. See the
[status contract](../docs/contracts/migration-status.md). It still requires all
three successful build receipts and an exact reviewed hash; it updates the existing
job action and never deploys services. The currently committed old `migrate`
manifest is not a verification release and must not be used as one.

## Full application promotion and rollback

Before dispatch: merge recovery #23 and synthetic tooling #24; evidence and close
identity #6, secrets #11 and database #26. Owner applies reviewed foundation changes
with deployment federation enabled, provisions the database and runs reviewed SQL
bootstrap. Configure Identity Platform email/password, verified invitations, TOTP,
authorized deterministic frontend domain and the public Firebase API identifier.
Account enrollment follows the identity contract; no fake identity adapter deploys.

Publish a complete successful build via `build-images.yml`. Create a PR adding
`deploy/dev-release.json`, with exactly these fields:

| Field | Value |
| --- | --- |
| `schemaVersion` | `1` |
| `project`, `projectNumber` | Explicit project matching GitHub public variables |
| `operation` | `promote` or `rollback` |
| `build` | Object: `run`, `attempt` as decimal strings; `commit` as 40-character SHA |
| `images` | Object: `api`, `migrate`, `web`; exact component repository URLs with SHA256 digests from that build |
| `firebaseApiKey` | Public browser identifier; never Admin credentials |
| `evidence` | Object: `bootstrap`, `identity`, `secrets`, `compatibility`; each a reviewed issue-comment URL in this repository |
| `expiresAt` | UTC date-time later than dispatch, at most seven days ahead |
| `previousDeployment` | `null` for promote; successful prior deployment run ID string for rollback |

Evidence comments record named checks, reviewer and redacted outcomes. The owner
reviews the exact digests, current infrastructure diff, cost scope and schema/API
compatibility. The workflow verifies references structurally; it does not interpret
prose as proof. Merge the release PR, compute `shasum -a 256 deploy/dev-release.json`,
then dispatch `deploy-dev.yml` on main with `release_sha256`. Never paste credentials.
An updated manifest or expired authorization requires a new review/hash.

The workflow authenticates only after metadata/dependency checks and synthetic
release/infra tests. Plans/state/provider diagnostics are not uploaded. On failure,
owner diagnoses using private operator access; do not paste raw provider output.
If migration failed, services have not been updated. If service rollout/smoke failed,
inspect both revisions: API and frontend updates are not atomic.

For rollback, select a successful earlier deployment receipt and its matching build
artifacts (build retention 30 days, deployment receipt 90 days). Review compatibility
with the current schema, put the prior images/build in a new manifest, set rollback
and prior deployment ID, then dispatch its new hash. Workflow skips migrations and
changes services through Terraform. It never rolls back SQL or silently selects an
image. Missing artifacts block automated rollback; review a separate recovery path.

Before closing #6, complete the separate managed-identity rehearsal (#39),
email-verification implementation (#38), and restricted SQL identity evidence.
The rehearsal has no finance routes and does not require a Cloud Run deployment;
it does not replace hosted acceptance or relax the current closure gates.

After transport smoke, complete real sign-in/MFA, entity/revocation denials and
synthetic posting/recovery with owner-controlled access. A green transport receipt
is not bookkeeping or identity acceptance. See the [living ledger](../docs/runbooks/gcp-delivery.md).
