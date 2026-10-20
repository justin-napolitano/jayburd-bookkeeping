# Binding review recovery sources

Reviewed 2026-10-19 for issue #215.

## Plaid Items API

- Source: <https://plaid.com/docs/api/items/#itemremove>
- Review depth: endpoint purpose, request/response, token invalidation, and
  offboarding guidance.
- Fact: Plaid documents `/item/remove` as the endpoint for removing an Item and
  says the Item access token and associated processor/bank-account tokens become
  invalid. Plaid recommends it when a user disconnects and requires it to end
  billing for subscription products unless the user already revoked access.
- Project decision: a Jayburd binding-review cancellation is not complete until
  provider removal is confirmed. Local state alone cannot represent revoked
  provider access.

## Plaid Item errors

- Source: <https://plaid.com/docs/errors/item/#item-not-found>
- Review depth: `ITEM_NOT_FOUND` definition, causes, and recovery guidance.
- Fact: Plaid says `ITEM_NOT_FOUND` can mean the Item was previously removed by
  `/item/remove`, removed by the user, or removed by Plaid.
- Project decision: only on the exact already-fenced Item-removal retry,
  `ITEM_NOT_FOUND` is accepted as the terminal already-removed result. It is not
  treated as success for account reads, exchange, or any other operation.

## Access limits and standards

The public documentation was reviewed; no provider payload, dashboard record,
credential, or bank data was collected. Item removal is a proprietary lifecycle
operation rather than an ISO 20022 business message. Existing ISO 4217 currency
and accounting contracts remain applicable to binding completion.
