# Connected account presentation —290 increment

## Decision and boundary

2026-11-03 source proposal; review/merge pending. Existing account projections
only. [Display labels](account-display-labels.md) remain current-view ordinals,
with their fixed-set/reorder limitation. Bank names/masks need a later reviewed
private source/API contract; they are neither invented nor fetched here.290 stays
open for that work.104/255 authority/status behavior is unchanged.

Each connection retains its heading and actual state. Registered accounts become
a semantic list with one readable heading/state and compact primary actions:
Transactions and Sync status. Accessible action names retain the account label
and immutable source/stream handoffs. An explicit Manage disclosure contains
disconnect and the existing nested technical-ID disclosure. Opening either
disclosure is presentation only, never a provider call, disconnect or permission.
The existing separate Confirm disconnect fieldset, exact body/key, same-request
recovery, disabled/denied/stale/scope fencing and retained-history explanation
remain unchanged. Current import/readiness information is retained where already
available; active state is never relabeled as successful sync, complete history
or bank balance. Empty connection account lists explain the current book scope.
The Item-level technical disclosure follows the account list, leaving the primary
actions ahead of technical metadata. Synthetic proof outputs are visually hidden
in the test-only harness; no production DOM access/identity adapter is added.

## Sources, alternatives, standards and design

Reviewed repository sections2026-11-03: existing label contract's decision/privacy,
registration-http privacy and command-recovery browser boundary; selected SR01 in
[selection source record](../research/select-register-sources.md) reused only for
the existing names/masks access limit. No new vendor behavior/normative text review.
Fact: the current listing supplies account/source/stream IDs and state, without
bank display names/masks. Inference: structured headings and shorter visible
actions improve scanning without changing that projection. Native details keeps
secondary controls keyboard accessible. Alternatives: prominent disconnect on
every row increases visual competition; a modal adds focus/recovery complexity;
fake bank names/masks misrepresent evidence. A future safe metadata contract is
separate, not silently completed by this presentation.

Retain paper white #fff, ink #202124, muted #555b63, divider #dce0e3 and green
#286044, system sans. Left-aligned connection heading and simple account rows with
dividers; readable state at the opposite edge, primary actions below. Avoid nested
cards and repeated long button labels.44px controls, wrapped mobile actions,
visible keyboard focus and no horizontal overflow. Existing technical disclosure
and explicit confirmation copy preserve context for destructive actions.

STD01: internal presentation has no applicable registered ISO20022 exchange;
existing ISO4217/date/money/source mapping unchanged. No new data contract/export,
ISO21378 mapping or conformance claim. STD02–05 unchanged transport and authority.
Restricted-data controls: synthetic fixtures only, no new collection/URL/log/
persistence; IDs remain exact internal keys and escaped private technical text.
No authentication, authorization, accounting or API/SQL changes.

## Acceptance

Browser desktop/390px keyboard tests: registered list has distinct current labels,
primary review action hands over the original source; Manage reveals controls
without a request; disconnect selection opens the existing explicit confirmation,
cancel sends nothing; locked/disconnected actions stay disabled. IDs remain hidden
until both disclosures open. Existing unit lost-response/confirmed-rejection tests
must explicitly open Manage and retain all exact-key assertions. Real fictional
Production journey still proves consent/authorization/source isolation, preserved
history and explicit disconnect. Core/locked web/type/build/browser and actual app/
connection journey gates; screenshots inspected. No hosted/live qualification.
