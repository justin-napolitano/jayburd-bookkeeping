# Recovery and integration sources

Reviewed 2026-10-11 before implementation. Relevant sections only; these are
engineering constraints, not accounting-policy or ISO conformance evidence.

| Source / review depth | Established fact | Project inference |
| --- | --- | --- |
| [PostgreSQL17 isolation §13.2.2–3](https://www.postgresql.org/docs/17/transaction-iso.html), relevant sections reviewed | Serializable transactions can fail and must be retried in full; committed effects have serial semantics. | Commit command completion and financial mutation together; reuse existing full-transaction retry boundary. |
| [PostgreSQL17 locking §13.3.2–4](https://www.postgresql.org/docs/17/explicit-locking.html), row locks/deadlocks reviewed | Conflicting row locks wait; ordering matters and deadlocks can abort transactions. | Serialize the principal/command workflow and reuse accounting lock order; prove execute/cancel and revocation races independently. |
| [OWASP HTML5 Security, Local Storage](https://cheatsheetseries.owasp.org/cheatsheets/HTML5_Security_Cheat_Sheet.html#local-storage), full subsection reviewed | Browser storage is script-accessible and cannot stand in for authorization; sensitive data/session IDs should not be stored there. | Keep tokens and financial payloads out of persistent browser storage; discover server-owned recovery only after authentication. |
| [Playwright network, modify responses](https://playwright.dev/docs/network#modify-responses), relevant section reviewed | Routing can fetch the actual response before altering delivery. | Drop an acknowledged real API response to prove recovery against committed SQL state rather than a fabricated success. This does not test live identity providers. |

Alternatives: storing request bodies in localStorage survives refresh but leaves
financial data on shared browsers and cannot recover on another device. Receipt
lookup alone needs a retained key and cannot safely distinguish an in-flight write
from one never received. Server-owned prepare/execute/acknowledge adds round trips
and a narrow persistent command record, but makes unresolved work discoverable.
One unacknowledged command per principal is an intentional initial concurrency
limit. Revisit for multiple concurrent operators/devices with an explicit workflow
contract; do not silently auto-acknowledge outcomes. Revoked scope recovery remains
operator-mediated. Contract and named proof: [command recovery](../contracts/command-recovery.md).
