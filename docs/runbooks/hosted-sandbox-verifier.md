# Hosted Sandbox verifier delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue272; owner authorized the proposed fresh hosted synthetic verification.
Lease VERIFY272-HOSTED-20261023T1439Z, dedicated delivery branch/worktree from
978725f. Migration30 applied and independently verified. This branch supplies
missing one-shot execution/bootstrap preparation; actual fixture/mandate proof
must be recorded separately. No existing expired source consent is reused.

## Stage ledger

Renewal delivery: owner approved PR282, merged68612cb after all five exact-head
checks passed. Execution lease VERIFY272-EXEC-20261023T1802Z covers issue272.
The fresh clock check was18:02Z, after the original app candidate expired; it was
not dispatched. This branch changes only the two app authorization deadlines:
dispatch21:00Z and new-source consent23:00Z on2026-10-23. Existing Item consent,
images, build, secrets and infrastructure are unchanged. Exact renewed hash is
955166d1e8850ffa236c62ab3bd74d0f4cb2c406c7ec075e5df8594824bb4ea3;
owner exact-hash review remains required by deploy/README.md before dispatch.
This temporary forward release does not resolve issue281's expiry/startup bug.
The approved bootstrap hash730b079c was applied successfully from clean merged
68612cb using the guarded helper: exact metadata validated, private Terraform
init/plan/apply completed, and no comparison executed. No private provider logs,
plans or state were inspected. Cloud Shell currently has no active operator
account; SQL login grants and actual comparison remain pending owner sign-in.
Renewal checks: exact manifest plus all three retained trusted image receipts,
release-test and make check/test/build PASS. The first sandboxed core test run
could not bind local httptest ports; the approved unrestricted rerun passed.
Local cached Go1.26.5 temporary modfile remains separate from tracked1.26.10;
canonical PR CI is still required. No product code or SQL changed.

| Stage | Evidence | Next action |
| --- | --- | --- |
| Resume | GitHub/main/register checked; predecessor lease released | Prepare bounded runtime |
| Design | hosted-sandbox-verifier contract records sources, alternatives and boundaries | Implement/test exact profile |
| Identity/runtime | Safe bootstrap and isolated runtime implemented; real SQL +7 mocked infrastructure scenarios PASS; not provisioned | Human source review before platform apply |
| Fixture/sync | Four-row fixture authored/hash-pinned; no provider test action; identity-only projector and negatives PASS | New bounded app release, reconcile saved setup, then one fresh custom-user source |
| Comparison/cleanup | Not executed | Require finite mandate and redacted actual receipts |

## Current evidence

Existing signed-in browser workspace is reachable. Existing deployed consent
expired2026-10-22; a new scoped interval is required. Existing worker build39898191502
contains /sandbox-verify; new product SQL is unnecessary. Review scope separates
synthetic tests, actual IAM/bootstrap, provider acquisition and comparison outcomes.

Initial PR282 head2c08b81 Infrastructure39994561810 failed in the new fake-executable
shell test because rg was unavailable on the runner. Both new shell proofs now use
standard grep with the same strict assertions; no test outcome was waived. The
source/app integration jobs from that superseded head are not final evidence.
The incomplete-response test also preserves JSON raw numeric lexemes while
removing an identity, avoiding float decoding even in synthetic mutation setup.

Core check/test/build, migration-test, ledger-test, sandbox-verification-test,
verifier-bootstrap-test, release-test and focused mocked Terraform tests PASS.
Pinned actionlint1.7.7 on the modified Seed workflow also PASS after the approved
network retry for registry metadata. Final canonical manifest/build validation PASS.
Go local uses cached1.26.5/temporary modfile, tracked1.26.10 unchanged; canonical
exact-head CI still required. Initial Terraform DNS and Docker sandbox denials
were rerun with approved access. A synthetic operator initially lacked database
CONNECT grant authority; its setup was corrected, a positive actual login assertion
was added before denials, and rejected first-time bootstrap rollback is now proved.
Initial context-budget failures were resolved by concise bibliography annotations;
no evidence/authority requirement was removed.

The original injection-ID assumption was corrected from official API sections:
the create endpoint returns request_id only. New cmd/sandbox-identity-map defaults
offline, accepts only an explicit owner-private pipe and pre-pinned fixture, writes
the fixed private create-only map, and never prints IDs/tags/financial fields.
Its financial-independence/shape/duplicate/scope/size/file/redaction tests PASS.
Only a complete initial posted-only page is supported; no pagination/history claim.

Independent actual fixture (private temporary directory, never provider-derived):
`794d555b11ca1cbc94b59c5d944d14105b2cce76d1d7691b1988f314a5a62964`;
separately authored custom-user configuration:
`a2aba7ed7daab88e70d09affb75b162eacecf6287d3f168be9670b8e1d449d36`.
Four dated USD rows include distinct equal-valued purchases, refund and a cent.
Installed current USD currency metadata was selected from the visible setup form.
Other currencies remain required scope; this first case cannot qualify them.

### Hosted UI availability finding

An owner-authorized fresh synthetic setup was prepared through the real managed
browser. Registration/reconciliation and one unchanged retry remained UNKNOWN;
no execution was requested. Preserve the same pending registration, do not create
another command. Selected request metadata showed API500/web502. Fixed API stderr
on deployed00019-k8v says server configuration rejected; selected argument readback
confirmed consent deadline2026-10-22T17:00Z. Config.Validate rejects expired metadata
before startup. Issue281 records this availability coupling and its missing durable
fix; new-source consent metadata below can restore qualification startup but does
not fix future expiry handling. No private diagnostics/payloads/credentials exported.

## Remaining work

Human-review new sensitive IAM/runtime/projection source and exact metadata below.
After all source CI passes and native merge, perform the guarded app release and
verifier platform bootstrap. Reconcile the saved setup registration before any
setup execution/terminal acknowledgement; create only its authorized fresh book.
Then controlled custom-user Link/acquisition, owner-private identity projection,
finite mandate enrollment, one comparison, revocation and exact input cleanup.
No synthetic PASS closes hosted272/88. Actual pipe/mandate/IDs/runtime results are
not available yet. Human review applies under the Citizen Developer security skill.

## PR handoff

Links272/88 and docs/contracts/hosted-sandbox-verifier.md. No new migration expected;
merged SQL immutable. Keep default tests offline and synthetic. Owner handles
credentials privately; no credential files, raw provider payloads or financial
observations in output, artifacts or Git.

Exact candidates, not dispatched:

| Operation | SHA256 |
| --- | --- |
| Application Sandbox forward release | `121ebe1f3e3f53f3b8c003f194450b8a7a93a84623bfb6d795f27bdf45653bc8` |
| Verifier platform bootstrap / idle cleanup | `730b079ca00af12d11a8022bf3c20ba70fc7b860af9525b7455f5caf095ede1a` |
| Verifier one-shot runtime configuration | `653b95634100df485bd15db1f2bf92f05f6317e53c58f95864df296e25b20e8f` |

Application expiry2026-10-23T18:00Z; new-session consent2026-10-23T20:00Z.
Trusted owner build39898191502 attempt1/source edb778e; original all-component
receipts validated. Worker digest238782abc1c7fe3667ba92ec2f29c1fbdb6a00cfd6455264c0ef2b9290cf35a0.
No caller-selected image/build discovery or consent renewal. Expired/changed
candidates require fresh exact review; do not silently extend them.

Owner-operated platform command after approved source merges: check out current
main, then `bash scripts/apply-sandbox-verifier.sh --apply
deploy/verifier-bootstrap.json <bootstrap-hash> <reviewed-main-commit>`.
The runtime candidate uses the same command with deploy/verifier-runtime.json and
its hash only after reviewed finite inputs exist. Default is inert. This copies
explicit public Terraform source to a private directory, uses a separate fixed
backend prefix, keeps plans/state/diagnostics private and never executes the job.
Short-lived platform authentication remains owner-injected; no credential files
or GitHub tokens are copied. Remote main read uses public HTTPS with credential
helper disabled. SQL bootstrap follows the existing Cloud Shell proxy procedure;
never run product migration or generic financial queries as administrator.

For a manually created fresh book, use the existing reviewed
db/bootstrap/sandbox-book.sql procedure after actual Scheduler PAUSED readback;
it grants the scoped current service/import binding, not provider credentials.
Use the trusted managed setup result/current creator metadata, never an invented
actor identity. Preserve the Scheduler's prior state and use selected queue-name/
lease evidence without task payloads or a universal drain claim. Existing creator
self-service enrollment may supply this setup; do not duplicate enrollment when
the authoritative current binding already exists.
