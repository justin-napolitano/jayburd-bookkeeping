# Connection service boundary

Approved synthetic implementation design for #101, 2026-10-13. SQL migration13
owns current connect permission, attempts, one-shot claims, explicit bindings and
shared Item fencing. This package supplies injected provider and immutable secret
storage boundaries. No real provider, Infisical, cloud or credential access occurs.

`Begin`, `Execute`, `Reconcile`, `Attempt`, `List`, `Bind`, and `Disconnect` accept
verified actor/entity/book scope. Begin modes are new/reconnect/rotate/remove.
Existing Item operations require the reviewed generation and complete approved
stream IDs. Tokens returned to the browser remain memory-only. The browser returns
the original Link token and nonce; SQL receives hashes only, while the provider
verifies the completed session association before exchange. Client metadata never
authorizes account ownership. Missing provider association fails closed.

SQL commits each external-operation claim before I/O. Only the first confirmed
claim may exchange/rotate/remove. A lost SQL acknowledgement performs no provider
operation. A lost provider acknowledgement remains credential recovery and is never
blindly retried. Successful exchange verifies Item/accounts, durably records that
manifest with a preallocated credential reference, then writes the secret version.
Exact reference/version/Item and token readback precede reference publication.
Recovery only reads a deterministic version and revalidates its manifest; it never
repeats exchange or rotation. An absent secret remains unresolved.

Reconnect verifies the same Item and account scope; changed Item identity requires
a new connection and explicit overlap review. Rotation fences every sibling before
external mutation. Single-account disconnect never calls Item removal. Full Item
removal requires all affected bindings, fences locally first, and confirms remotely
before reporting removal complete. Posted journals and source history are retained.

Selected depository accounts map explicitly to ledger/source/currency/evidence.
SQL permits a first provider namespace without an overlap hold only where it proves
there is no prior namespace/evidence; existing sources retain overlap review.
Unsupported accounts are metadata only. No balances, journals or transactions are
invented by connection management.

## Sources and limits

Reviewed selected official sections on 2026-10-13, extending PI05–PI08:

- [Plaid Link API](https://plaid.com/docs/api/link/): token creation/expiration and
  token-get session results. Complete non-Hosted session events require provider
  enablement. The injected strict session verifier is a Hosted-Link-compatible
  qualification profile, not a claim that arbitrary callbacks prove association.
- [Plaid Items API](https://plaid.com/docs/api/items/): exchange identifies an Item;
  rotation replaces its credential; removal invalidates Item access. Remote
  operations are not assumed transactional with local SQL.
- [Plaid Accounts API](https://plaid.com/docs/api/accounts/): server account metadata,
  case-sensitive identities and account-type/currency mapping. Names/masks are not
  ownership proof; balance data is not used by this service.

These are selected public documentation sections, not Sandbox qualification.
Proprietary Plaid JSON retains the existing STD-01 assessment; ISO4217 currency
references and date rules apply. No ISO20022 or legal-consent conformance is claimed.
The concrete HTTP adapter uses injected transport and fixed sandbox endpoints;
Infisical production adapter and hosted Link/event availability remain #88 gates.

## Validation requirements

Synthetic lifecycle tests must prove denial before I/O, wrong scope/session, current
revocation, duplicate claims, unknown provider/secret/SQL acknowledgement, exact
secret readback, same-Item reconnect, sibling rotation fencing, explicit binding,
account disconnect versus Item removal, and no credential serialization. Actual SQL
tests must compose the service with migration13 and verify no financial mutations.
