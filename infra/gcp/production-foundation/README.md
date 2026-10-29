# Production foundation candidate

Source only; no apply approved. See [contract](../../../docs/contracts/production-foundation.md).
Use `bash scripts/test-infra.sh production-foundation` for clean mock-provider tests.
Owner must accept337, verify project/number and billing/cost, configure a separate
backend and review exact resources before any apply. No owner tfvars/backend file
is committed. Bootstrap the state backend separately; do not read/copy existing dev
state. Build federation and optional human SQL operator default disabled/absent.
This root creates no runtime services, deployment authority or live provider scope.
