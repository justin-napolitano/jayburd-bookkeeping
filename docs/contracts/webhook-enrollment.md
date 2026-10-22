# Sandbox webhook enrollment

## Decision and scope

Engineering profile for #271, coordinator reviewed 2026-10-22 under the owner's
implementation-only swarm authorization. No provider/cloud execution, deployment,
public exposure, credentials, consent renewal or Production activation.

`connections.HTTPConfig.WebhookURL` is optional and defaults empty. Empty keeps
the existing new-Link request unchanged, including omission of the `webhook`
field. A nonempty value must exactly equal `SandboxTransactionsWebhookURL`:

`https://jayburd-dev-webhook-9088794002.us-central1.run.app/webhooks/plaid`

This public canonical endpoint identifies only the dedicated development
Transactions receiver. Constructor validation rejects every other spelling,
scheme, project, host, port, path, userinfo, query or fragment before constructing
the provider. No request body, user choice, environment discovery or provider
response selects a receiver. Changing the approved endpoint requires source
review and a release-policy update, not a permissive URL parser.

For `Mode=new`, the fixed Sandbox `/link/token/create` request adds the reviewed
`webhook` value only when explicitly configured. It retains Transactions only,
US/English, Hosted Link, exact API version 2020-09-14, bounded no-retry transport
and the existing encrypted handoff/one-shot exchange rules. Reconnect remains
unqualified and disabled. Startup performs no provider call or Item mutation.
The API exposes optional `-plaid-webhook-url` and `connectionruntime.Config`
threads it to the provider. Before I/O, a nonempty value additionally requires
ProjectID `jayburd-bookkeeping-dev` and ProjectNumber `9088794002`. Disabled
composition still requires an entirely zero configuration, so a receiver flag
alone never enables enrollment. The coordinator owns release metadata and
infrastructure propagation; adding these fields changes no running deployment.

## Sources, alternatives and standards

[WE01–WE03](../research/webhook-enrollment-sources.md) records selected current
official sections and access limits. Plaid accepts `webhook` during new Link
creation; update-mode Link creation does not change an existing Item's webhook.
Initial sync activates `SYNC_UPDATES_AVAILABLE` delivery. Therefore opt-in
enrollment metadata is necessary configuration, not signed-chain evidence.

Alternative arbitrary HTTPS configuration is rejected: malformed or substituted
recipients could receive provider account notifications. Alternative enabling
the URL by default is rejected: public receiver/IAM readiness and owner scope
must precede activation. Exact optional configuration preserves old deployments.
Alternative treating an update acknowledgement as success is rejected: the
acknowledgement is a different Item event, without receipt-to-publication proof.

STD-01 applicability: this adds an operational destination to proprietary Plaid
Link JSON, not an ISO 20022 financial exchange. Mapping is internal reviewed URL
→ `/link/token/create.webhook`; there are no amounts, currencies, dates or
journal mutations. Existing ISO 4217/source-accounting mappings are unchanged.
No ISO conformance, privacy-policy or live-use approval is claimed. A registered
ISO financial message does not substitute for this provider-specific bootstrap.

## Existing Items — proposed, not implemented

Plaid requires `/item/webhook/update` with the existing access token; Link update
mode cannot perform this change. Do not add an unguarded method, route, token
CLI argument or provider allowlist entry. Required reviewed design:

1. An authenticated actor/entity/book command names one known Item, its expected
   generation, complete affected approved stream set, exact reviewed target and
   finite owner-authorized Sandbox interval. An Item may back multiple streams;
   account-level permission alone is not Item-wide mutation authority.
2. Current SQL authority and consent must be rechecked before any provider I/O.
   A deterministic durable operation and claim resolve concurrent requests and
   lost replies. The current schema has no approved webhook-update command;
   review a new SQL/API boundary rather than repurposing rotate/remove/reconnect.
3. A scoped credential resolver reads the immutable current vault reference and
   decrypts only in process. Match exact credential version/Item/generation;
   neither agents nor operator flags retrieve access tokens. No extra vault/KMS
   access is granted to the inbound webhook host.
4. The adapter uses the fixed Sandbox endpoint, fixed URL, bounded one-attempt
   transport and redacted errors. Validate the returned Item identity and URL.
   An ambiguous response remains uncertain; inspect authoritative Item webhook
   metadata under the same scope before deciding whether another action is safe.
   Never blindly retry a mutation or broaden a single Item into an enumeration.
5. Provider `ITEM/WEBHOOK_UPDATE_ACKNOWLEDGED` goes to the new receiver. The
   current verifier/handler accepts only Transactions notifications and must
   continue rejecting it. Do not call an HTTP acknowledgement, provider response
   or this Item event W01/W02 success. Supporting Item events needs its own
   reviewed profile; no silent broadening of this receiver.
6. Actual acceptance must traverse first sync/signed
   `SYNC_UPDATES_AVAILABLE` → durable receipt/wakeup → publisher/task/worker →
   correctly scoped source publication. Add duplicate/retry/missed-event cases,
   independent exact/no-journal evidence and cleanup/rollback procedures.

The proposal cannot run with expired consent. Existing Item activation remains
blocked until these authority, migration/API, release and owner-scope decisions
are reviewed separately.

## Validation and remaining evidence

`TestWebhookEnrollmentOptIn`: empty configuration omits the wire field; explicit
configuration emits the exact independently written destination; constructor is
inert, new enrollment retains Transactions-only scope and performs one request.
`TestWebhookEnrollmentRejectsDestinationSubstitution`: hostile/lookalike hosts,
scheme/project/service/path substitutions, userinfo, port, query, fragment,
case/whitespace variants fail before transport and reveal fixed errors only.
`TestWebhookConfigurationScope` rejects disabled, foreign-project and substituted
configuration before SQL, token minting or secret reads. `TestWebhookOptInStartup`
proves valid composition reads only the two existing exact static versions; no
provider/SQL/enrollment operation occurs at startup.

Run existing `internal/connections` race tests and `make check test build`.
No SQL or browser/API contract changes in this slice. Root integration must test
optional configuration/release propagation separately. Hosted W01–W05/W07 remain
open until exact-image deployment and actual signed delivery are observed under
fresh explicit scope. Merely configuring this URL is not qualification.
