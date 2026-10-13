# Import review journey

## Decision

Use explicit source account selection, preview, import confirmation, review and
separate draft preparation. The [interactive prototype](../prototypes/import-review.html)
is synthetic and in-memory; it is not an application route. The
[durable import contract](../contracts/durable-import.md) owns proposed persistence
and wire semantics. Existing [decoder](../contracts/statement-decoder.md) owns
file lexical rules; no upload/parser implementation is implied here.

## Work

Select entity/book and a source account before preview. Display original amount,
currency, source identity and booking date separately from later accounting choices.
Confirmation creates review items, never journals. `needs_review` may become
`draft_ready` after explicit classification and valuation; a separately reviewed
source-linked posting command must atomically reauthorize and link its journal.
No generic “approved” status implies that posting occurred.

Repeated identities with equal canonical content create no duplicate review items.
Different IDs preserve identical purchases. New adapter overlap remains `overlap_hold`. Changed fields under an existing CSV
entry identity reject the whole batch as `source_correction_required`; future
provider changes require `correction_hold`, not an invented automatic resolution.
A digest alone cannot prove business identity. Every scope/scenario change clears
preview, selection and proposed draft. Denial hides all evidence; malformed input
rejects the complete fixture without partial acceptance.

Use native labels, visible keyboard focus, semantic table headers/caption and an
announced status region following existing FW01–02 source review. Original decimal
strings include USD 12.50, JPY 125 and KWD 1.005. Foreign-currency draft preparation
is blocked until explicit functional valuation exists; no browser conversion.

## Done when

The prototype journey proves separate identical purchases, replay without new
items, hold gating, denied/malformed clearing, scope invalidation, exact original
units and explicit USD draft preparation. Evidence belongs to the
[living runbook](../runbooks/import-review.md). Real imports, retention, permission
checks, source-linked posting, production accessibility and mobile acceptance remain
implementation gates. This design introduces no new ISO financial message.
