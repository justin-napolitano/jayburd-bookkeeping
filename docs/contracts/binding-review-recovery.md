# Binding review recovery contract

## Scope

This contract covers a new Plaid Item after public-token exchange and encrypted
credential publication, while its attempt is still `binding_review`. Completion
remains the normal path: the owner reviews distinct ledger-account proposals and
submits one atomic binding command. Cancellation is an explicit fallback that
removes the provider Item before another enrollment becomes available.

Issue #215 owns this slice. Automatic supported-account creation remains defined
by [automatic account binding](automatic-account-binding.md); credit and loan
policy remains #210.

## Binding readiness

The browser derives proposals only from the verified attempt manifest, current
book snapshot, installed ISO 4217 currency versions, and selected binding date.
It must name each unmet requirement beside the disabled confirmation: binding
date, at least one complete selection, matching ledger/currency fields, unique
ledger accounts, reviewed new-account code/name, or owner acknowledgement.
Unsupported provider accounts remain visible and unbound.

## Cancel state machine

Only an authorized, new-mode `binding_review` attempt with its exact paused Item,
published encrypted credential, and zero streams may begin cancellation. The
database locks the attempt and Item, changes both to `disconnect_pending`, and
increments the Item generation before provider I/O. This serializes against the
binding command: whichever transition commits first wins.

The service reauthorizes the exact attempt, reads only its addressed encrypted
credential, and calls Plaid `/item/remove`. A confirmed response, or Plaid
`ITEM_NOT_FOUND` on a retry, permits finalization. Finalization rechecks the
claim, Item, generation, credential publication, and absence of streams; it then
marks the Item `disconnected` and attempt immutable `abandoned`. The encrypted
credential and audit events remain under the existing retention contract, but
the removed access token is no longer usable.

A timeout, malformed response, provider failure, lost prepare response, or lost
final response never releases the enrollment gate by inference. The attempt
stays `disconnect_pending`; repeating the same cancel action resumes removal for
that exact Item. No path recreates Link or repeats public-token exchange.

Cancellation creates no ledger account, source account, namespace, stream,
grant, balance, transaction, or journal. It is a Plaid lifecycle operation and
not an ISO 20022 financial message. Existing ISO 4217 mapping and accounting
policy are unchanged.

## Evidence

Synthetic SQL tests must prove prepare/finalize guards, bind-versus-cancel
serialization, replay, current authority, zero streams, and zero journals.
Service tests must prove exact credential addressing, `/item/remove`,
already-removed recovery, provider failure, and lost-response resumption.
Browser tests must prove named readiness blockers, explicit confirmation, and
fresh enrollment only after server-confirmed abandonment.
