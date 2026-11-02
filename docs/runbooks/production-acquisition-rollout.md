<!-- stage-runbook: true -->
# Finite Production acquisition rollout — #375

## Status and boundary

PROD375-RUNTIME source merged377/1eec1f0; current operator branch
ops/375-qualification from379/9d75886. Production app, Personal USD/Q4 book,
reference/scope, workers and signed receiver now deployed. Bank access remains
paused (API pilot=null); no Production Item/import/provider call performed.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |372/PR373 merged796d41c with five checks; validate23m26s;374/PR376 pending |
| Source review | Existing Production flags/SQL guards and Sandbox delivery/bootstrap templates |
| Decision | finite-production-acquisition; distinct root, paused scheduling, explicit finite API pilot input |
| Implementation | Inert Production acquisition root, separate finite API pilot input, strict exact-target webhook bootstrap |
| CI isolation | Seed caller/ref namespacing retains cancellation of superseded runs while permitting independent same-source Sandbox/Production validation |
| Owner review | Owner explicitly approved reviewed376/377 source/setup;376 merged06465ec after five checks, integrated here;377 exact-head CI still required |
| Combined checks | Rechecked combined main376 +375 source: core/release and both Production book/webhook bootstrap PASS; context budget retained |
| Validation |41 mocked infra cases; migration/tooling, Sandbox+Production webhook/worker bootstrap, core/release/actionlint, ordinary+Production browser/API/SQL journeys PASS |
| Actual book |374 completed: ordinary durable owner setup committed/acknowledged; creator scope and worker capability/cross-DB readbacks true |
| Actual images | Same-source1eec1f0 builds41280038692 Production and41280040488 Sandbox completed; all four component receipts independently matched trusted run/source/attempt/scans; temporary Production builder disabled |
| Actual app/readback | API00002-t6t/web00002-ppm Ready; public200/private unauth403; credential-check9nxkl completed and exact fixed integrity PASS; provider authentication untested |
| Managed IAM repair |379 merged after five checks and owner review; exact source03a97ea18ffa30100406193d95fe2342eca380f48cda2605454b29cacddab5d8 applied; six read-only webhook permission/profile/isolation predicates true; Google marker unchanged |
| Fresh scope | Owner approved proposal v2 SHA2560ef481ed3a70083a224c604c27e44cce889a38d18897086d908d2c519fc3c068 through2026-11-02T04:00:00Z: Sandbox35/test receiver/one fictional Item; paused Production workers/receiver, no bank activation |
| Actual acquisition | Input2498476a5a2a49c8927ecef31c95dc5e33fc36b7fda2b8612af0e4d4d989857b applied; receiver00001-7cn Ready; private publisher/consumer403, unsigned receiver401, Production Scheduler PAUSED; signed chain still unproved |
| Sandbox schema/runtime | Trusted predecessor head34/status verified; single approved forward35 execution ch55c and independent status8xjkb succeeded, migration job left read-only. Dedicated receiver00001-hdj, API00021-r59 and web00021-dnb Ready; five SQL capability/profile/isolation predicates true; public web200/private API403/unsigned receiver401 |
| Sandbox automatic acquisition | Owner explicitly superseded Sandbox paused-only scheduling; exact scanned publisher, binding-v1 and single fictional-book binding verified before enabling existing per-minute Scheduler. Consent deadline unchanged; Production Scheduler remains PAUSED |
| Sandbox first import | Ordinary durable book setup reviewed/executed/acknowledged; one fictional checking stream bound. Generation1 completed16:49:04Z; read-only SQL18 observations/18 review items/0 journals. Other manifest accounts unbound |
| Sandbox browser recovery | Explicit repeat requested, component unmounted/remounted through Ledger/Connections; server recovered requested2/completed2, last complete17:01:13Z. This is component-remount proof, not an injected lost HTTP response |
| Sandbox transport recovery | Source paused and UI confirmed paused/no lease, Sandbox queue paused, source resumed to requested3/completed2/queued/no lease. Existing automatic publisher retained one task. Queue resumed; independent SQL3 completions/18 observations/18 review items/0 journals, with no duplicate imported rows |
| Historical USD | Approved381 merged3004d66; reference-only source7278a8afffa51646eff7a35292f3c8488529accf81e77b53b7c2f7183220ac7b applied to Production. Independent current_usd_unchanged/historical_usd_exact predicates true;380 completed. No shortened lookback or transaction date rewrite |
| Genuine webhook | Link/Transactions receipt counts0 before repeat; no genuine signed-chain pass claimed. Owner signed into official Dashboard; Item Debugger returned No item found for exact fictional Item ID independently matched in SQL. No keys, tokens or raw payload panes inspected |
| Signed Sandbox proof385 | Reviewed389/5ea9990, trusted build41376988868; existing receiver00003-zwq/worker6eb4704 Ready/current/100%. Supported callback204/0.365s, durable receipt1, normal consumer204 and generation4/4; observations/review/journals18/18/0 unchanged. See [exact stages](sandbox-webhook-probe.md). Production receiver still needs the matching release |

## Current evidence

PP13 selected official operational docs and existing reviewed contracts; no secret,
state/plan or real financial-record inspection. Null/inert/expired defaults remain;
no source test reaches cloud/provider. Human review before real access changes.
Owner approved Personal/USD/Q4 setup and the fresh qualification window. Exact
Production bank pilot/account/retention/recovery disposition remains explicit.
All tests use synthetic fixtures and disposable databases. Browser identity/fake
provider/KMS transport do not prove actual managed authentication or Plaid behavior.
No Go dependency, schema or financial recognition change. Local core/integration
used installed Go1.26.5/external temporary modfile; tracked1.26.10 unchanged.
Pinned Terraform1.15.7/Google8.6.0 exercised selected PP13 mechanisms directly.
Source validation proves rejection of stale/overlong/foreign activation metadata,
not actual deployment approval. Existing Sandbox runtime mock contracts remain green.

## Remaining work

Actual scoped signed Sandbox delivery passed385. Independent provider-to-review
equality proof remains open; source/review row-count agreement and historical-ready status alone
are insufficient. Account pause/resume and queued-transport recovery passed above;
other applicable recovery scenarios still require their named evidence.
Production caller/current-scope/stop/restore evidence and applicable qualification
dispositions, then separately bounded owner-private bank pilot.401/403/configuration
observations do not prove signed delivery, publication or reconciliation. Never
silently renew authority. Sandbox scheduling is explicitly authorized only for the
one fictional test book within the existing finite consent. Backup metadata shows seven retained daily
backups/PITR logs and successful recent backups; actual restore is not yet proven.

## PR handoff

Qualification evidence update rebased onto merged381/3004d66. Required
`make check test build` and staged runbook gate passed with the documented
temporary Go toolchain adapter. Documentation only; no migration/image change.

Refs #375/#316/#337/#90. No migration, financial semantics or provider calls from
default local/CI commands. Preserve separate resource/state ownership.
