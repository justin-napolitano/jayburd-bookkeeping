# Finite Sandbox release for readiness fixes
<!-- stage-runbook: true -->

## Status and boundary

Issues #281/#289/#293; candidate preparation only. Source changes merged in
PR307/309/310; [readiness roadmap](../plans/bookkeeping-readiness.md) merged306.
No new migration, endpoint, dependency, identity authority or financial policy.
No managed rollout, provider call, real account or automatic posting yet.

Lease `APP281-CANDIDATE-20261024T1618Z`, branch
`release/281-browser-acceptance`, base/source
`c96d5e94ea6e0762ff631d7bd0bbdeb7435ebad5`. Dedicated worktree; no competing
writer or SQL version reservation. Earlier queued06:40 lease text was corrected
in #65 before authoring these files; it did not authorize edits after expiry.
Lease yielded temporarily for the two-document reconnect review correction;
`APP281-CANDIDATE-RESUME` reclaimed these exact paths at16:33Z through18:03Z,
before manifest authoring. Current #65 comments own the precise timestamps.

## Stage ledger

| Stage | Evidence / limit |
| --- | --- |
| Resume | Current #65/#281, deployment/connection contracts and prior actual31 app receipt reviewed |
| Source review | Only three bounded source fixes since prior selected build; infra, workflows, dependencies, SQL and Dockerfiles unchanged |
| Trusted build | Owner/main build40110865260 attempt1 for c96d5e9 all7 jobs SUCCESS, including exact image runtime/security gates |
| Candidate | Three successful same-build candidate receipts matched; CLI manifest/hash/build validation PASS |
| Owner review | Not approved/dispatched; exact reviewed manifest hash required |
| Hosted acceptance | Not observed; source proof does not clear actual gates |

## Current evidence

Merged source: expired well-formed optional enrollment keeps SQL bookkeeping and
command recovery available without bank/secret I/O; malformed configuration still
rejects. Sign-in distinguishes bootstrap loading, retryable transport failure and
invalid config, fencing stale controllers. Authorized transaction detail survives
optional category failure; auth/not-found errors still clear sensitive state.

Each source PR's exact head passed all five CI checks. Combined current-main
core/race/build, web gates (26 browser cases), and actual synthetic
browser→Next→Go→PostgreSQL lost-response setup/post recovery passed in306. Local
Go1.26.5 used a temporary modfile; tracked1.26.10 unchanged. Trusted image build
supplies canonical pinned runtime/security evidence. Local Alpine registry TLS
failure is not waived or treated as exact-image success; controls were unchanged.

Previous actual application release40014820903 selected schema31 and API20-zpd /
web20-zb4. Candidate must run schema31 **status**, never an up/down migration.
No SQL change since that release. Only deploy metadata changed in infra/deploy/
workflow/Dockerfile comparison fromfb4ac04 to selectedc96d5e9. Source diff contains
16 bounded Go/TS/test files; no provider/acquisition/worker behavior replacement.

Retain existing public browser identifier, numeric static secret version names,
Infisical metadata, `link_events_qualified=false` and original Sandbox approval
deadline2026-10-24T23:00Z. Never retrieve secret contents. No webhook opt-in,
existing-Item consent update, worker/scheduler root or Production host change.
Proposed dispatch deadline20:00Z is finite and earlier than existing enrollment
deadline; it authorizes no execution until the exact candidate is reviewed.

Exact [manifest](../../deploy/dev-release.json) SHA256:
`707c3856ed0c8d29723a5baf1a2d66d4609db0dd8d9f2c0482985dd7dcaa2fe1`.
Selected build40110865260 attempt1/sourcec96d5e9 passed all7jobs. Exact API,
migrator and web digests are in that manifest and match retained candidate-only
receipts; publication alone is not deploy authorization. Compatibility review:
[#281 comment](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/281#issuecomment-6386746363).
No physical infrastructure plan/state or secret/provider logs were read or exported.
Local `make check test build release-test`, metadata/build/hash CLI and diff checks
PASS; staged runbook gate required before commit. New-head PR CI remains separate.

## Remaining work

Owner reviews the
concrete release under [deploy instructions](../../deploy/README.md), including
privileged release scope, compatibility and bounded cost. No automatic deploy on
merge. Updated/expired manifest requires a new hash/review.

After a separately approved guarded dispatch, validate transport receipt and exact
API/web revisions. Owner signs in/MFA privately; inspect only synthetic records.
Verify normal auth bootstrap reaches ready without transient invalid-config text;
review one authorized preperiod row with unavailable categories and confirm amount,
date/detail remain readable and no posting is enabled by a missing category.
Do not derive the earlier error's cause from this recovery alone.

At/after23:00Z, prove provider enrollment rejects while authenticated workspace,
SQL reads and exact saved-command recovery remain available. Distinguish warm
expiry from cold-start proof; an explicitly scoped restart/revision operation is
needed for actual cold-start evidence and must be separately reviewed. Never
shorten/renew consent or alter a live identity to manufacture a test. Record only
revision/time/redacted outcome and named case. Keep hosted281 open until observed.

Reconcile any existing UNKNOWN command by its original identity; no replacement
command to escape the recovery guard. New controlled setup/post tests require
explicit synthetic review/acknowledgement. Reconnect #275 and dedicated signed
delivery #271/#303 remain separate gates; this release implements neither.
Actual independent fixture equality/no-journal verification remains #272/#300;
46 visible Tartan rows and unchanged savings repeat do not prove full equality.

## PR handoff

Refs #281/#289/#293/#88/#90. Metadata-only release proposal; no migration or source
change. No production, real records, consent extension or worker scheduling.
API/web rollout is not atomic; on failure inspect both revisions, retain exact
receipts and prepare compatible forward repair or reviewed rollback. Prior
pre-fix images restore the expiry/UX bugs; rollback is not proof of readiness.
Human review precedes managed release. Passing build/transport does not close
hosted feature, Plaid qualification, personal pilot or reconciliation gates.
