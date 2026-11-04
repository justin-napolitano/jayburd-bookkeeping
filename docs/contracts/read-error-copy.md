# Read availability and command uncertainty —291

## Decision and boundary

Source implementation for291,2026-11-03; human review pending. The browser API
client distinguishes GET availability copy from POST commit uncertainty. It keeps
the existing `uncertain` flag, HTTP status, authority rejection and retry behavior
unchanged. Failed pending-command discovery still blocks all new work until an
authoritative result arrives. Reads never assert a zero balance or successful sync.

The discriminator is the body-presence check already used to select GET or POST;
no URL parsing, automatic replay or persistent state. Two-argument injected
APIError fixtures retain mutation wording by default. GET network/timeout/5xx/
408/429/invalid JSON explain unavailable data and an explicit read retry. Other
GET rejection/CSV format errors use fixed read failure copy.401/403/404 retain
identity/scope wording first. Raw error bodies remain unread; transport details
are never exposed.

## Evidence, alternatives and standards

[RFC9110](https://www.rfc-editor.org/rfc/rfc9110.html), June2022,
§9.2.1–2 and §9.3.1 reviewed in full2026-11-03: GET requests a representation,
safe methods express read-only intent, and non-idempotent retries require known
semantics or proof the original was not applied. These are HTTP facts. Our
inference is separate fixed wording with conservative command discovery. The
[recovery contract](command-recovery.md) retains authority for saved work and
explicit acknowledgement.

Alternative: clear uncertainty for every GET. Rejected because pending discovery
would lose its authority signal. Endpoint-specific copies duplicate transport
knowledge and leave ordinary consumers inconsistent. No new financial message,
amount/currency/status mapping or export: ISO20022/4217/21378 applicability remains
unchanged from existing contracts; no new message designation/conformance claim.
No DDL or API-wire migration. No expired-provider diagnosis or bank-sync claim.

## Acceptance

API client tests cover GET network/timeout/5xx/408/429/invalid JSON and CSV failure;
equivalent POST outcomes preserve unchanged-request copy and key. Authority tests
retain401 session rejection and403/404 scope fencing. Browser `failed pending
discovery remains locked until explicit successful read` proves network/503/
unreadable200 keep navigation disabled, no writes, then explicit successful read
unlocks only after `{command:null}`. Existing dropped-response real Next/Go/SQL
integration must pass. Hosted/provider qualification remains separate.
