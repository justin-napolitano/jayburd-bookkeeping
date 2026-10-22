<!-- stage-runbook: true -->
# Plaid #88 swarm integration

## Status and boundary

Owner authorized three parallel lanes plus coordinator on2026-10-22. Lease
`PLAID-88-SWARM-20261022T1540Z`, expiry20:00Z, implementation only. Existing
17:00Z source consent is not extended. Baseline52ef702 includes merged #269;
its validate/frontend/application-integration/foundation/secret checks passed.
No real account, provider call, new IAM grant or cloud exposure in this stage.

Dedicated coordinator branch `issue/88-swarm-integration` in
`/private/tmp/jayburd-plaid-fixes`. Agents own isolated worktrees/reservations
recorded in #65. Root owns shared connector/SQL/API/deployment/index integration.

## Stage ledger

1. **Dispatch complete:** WEBHOOK103 host, CONNECTION104 recovery and VERIFY85
   independent lifecycle/fault proof. Existing issue bodies/dependencies exported
   from fresh GitHub reads; no canceled predecessor treated as complete.
2. **Design review:** injected exact-route webhook host approved; existing-wire
   strict Hosted exit/recovery fix approved. Do not relax reconnect/session gates.
   [Managed webhook contract](../contracts/webhook-managed-runtime.md) records
   dedicated connector identity and future release/security conditions.
3. **Coordinator connector complete:** allow only the dedicated webhook
   connector component with existing narrow pool settings; no user/grant creation.
   `go test -count=1 ./internal/clouddb` and `make check test build` pass using
   the documented external local Go1.26.5 modfile. Initial core run was blocked
   only by a synthetic listener sandbox denial; approved rerun passed. No
   foundation or hosted test was run. Diff and staged runbook guard before commit.
4. **Validation/integration pending:** each lane supplies named results and exact
   commit. Root reviews diffs, combines changes and runs required integrated gates.

## Current evidence

Plaid Dashboard is a separate sign-in; Launch Center review is pending owner
authentication in the opened Dashboard tab. Independent local work continues.
No secret/key page or credential was read. The existing bookkeeping tab's
authenticated state was not mistaken for Dashboard access.

## Remaining work

Integrate passing bounded changes, review managed construction, prepare the
concrete release scope and execute authorized hosted cases only after prerequisites.
Full #88 qualification remains open, including the live signed chain and exact
hosted comparator; #90 is separately gated. New public exposure and sensitive
runtime changes require human security review under the applied security skill.

## PR handoff

Refs #88/#103/#104/#85/#87. No product migration yet, no changes to merged SQL.
Do not infer deployed behavior from synthetic tests or this implementation plan.
