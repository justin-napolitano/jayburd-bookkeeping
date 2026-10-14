# Connection bridge delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #104, lease UI104-BRIDGE. The private frontend bridge now forwards the seven
merged [connection HTTP operations](../contracts/connection-http.md). This is a
transport allowlist extension, not provider enablement or a working Link screen.
The existing [private bridge](../contracts/private-api-bridge.md) remains the
identity/header/deadline authority; Go and SQL still authorize each operation.

## Stage ledger

- Read AGENTS, project index, connection HTTP and private bridge contracts.
- Added case-sensitive canonical connection routes with exact GET/POST methods.
- Rejected query strings including bare `?`, encoded alternative paths, unknown
  provider routes and wrong methods before metadata or backend access.
- Preserved required idempotency keys on begin, bindings and disconnect. Execute
  and reconcile use their persisted attempt identity and need no additional key.
- Bounded connection request bodies to32KiB. Transient Link/public-token handoff
  bytes are forwarded once, with no new persistence, logging, caching or retry.

## Current evidence

- Focused `npm --prefix web test -- src/server/bridge.test.ts`:91 tests passed.
- `npm --prefix web run check`: Next route type generation and TypeScript passed.
- Full `npm --prefix web test`:206 tests in15 files passed.
- `npm --prefix web run build`: production build passed.
- `npm --prefix web run test:browser`:13 existing component scenarios passed.
- `make check test build`: vet, race tests, migration file checks and builds passed.
- Named connection bridge tests exercise all seven routes, required keys, method
  and query denials, encoded paths, bounds, identity rejection, exact transient
  bytes, separate service identity and safe uncertain upstream responses.

All transport fixtures are synthetic; no provider or cloud calls occurred. Browser
scenarios validate existing components, not the new connection UI being built in
the coordinator lane. Existing proprietary provider/internal-command standards
assessment applies; this transport change introduces no ISO message claim.

## Remaining work

Coordinator integrates the connection navigation/workflow and runs its combined
proof. Hosted service configuration, real identity/provider qualification and
Infisical credential writes remain separate approved deployment gates.

## PR handoff

Only bridge implementation, bridge tests and this living runbook changed. No SQL,
API schema, financial behavior, navigation or migration changes in this lane.
