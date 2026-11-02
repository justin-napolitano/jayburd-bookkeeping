# Private account display labels —290

2026-11-02 implementation assessment, existing-projection increment only.
Read SR01 in [selection sources](../research/select-register-sources.md) and the
[registration HTTP privacy assessment](registration-http.md). No provider calls
or normative ISO text reviewed. Existing manifest exposes ID/type/subtype/currency;
connection listing exposes immutable IDs/state. Neither includes bank names/masks.
Do not fabricate them or expand collection as part of this presentation change.

Decision: use distinct `Bank account N` labels for the current visible account set,
with verified type/subtype/currency beside discovery rows. Deduplicate and sort
the union of discovered and registered IDs so response reordering cannot change
the labels for a fixed set. Use that same lookup for binding controls, connected
rows, sync status, disconnect confirmation and transaction-review buttons. Keep
IDs as React identity and exact API/review handoff values. Show identifiers only
in explicitly expanded private technical details, as escaped React text.

These are current-view labels, not permanent bank identifiers. A changed account
set can renumber them; the UI explains this. Reload without discovery metadata
can also change the visible set. Persistent recognizable names/masks and their
retention/provenance/storage need a separately reviewed290 contract.290 remains
open; this increment must not imply full394 picker delivery or expose mock masks.

Alternatives: raw IDs as headings are difficult to review; truncated IDs and hash
labels remain opaque and can collide; invented bank names/masks misrepresent the
source. Additional private provider display metadata would improve recognition,
but requires reviewed source/SQL/API work. Existing ledger names are not a bank
identity and listing has no reviewed ledger-name join. Current-view ordinals are
an interim display choice with an explicit limitation, never a binding authority.

Design follows391: existing white/ink/muted/divider/green/error tokens, system
sans, left-aligned account list, quiet technical disclosures. Retain existing
actions/forms and focus behavior; no new icon, font, palette or modal.

Standards: internal presentation has no ISO20022 registered message. Existing
ISO4217 currency values, decimal money, dates and proprietary Plaid API2020-09-14
projection are unchanged. STD01 applicability here; STD02–05 unchanged transport,
identity and evidence. No new financial contract or ISO conformance claim.

Acceptance: fixed-set reorder/dedup/duplicate-type/missing-metadata helper tests;
component proof that labels select/review/disconnect the original IDs; synthetic
desktop/mobile/keyboard screenshot review; skipped-mapping regression; locked web,
core and app-integration gates. No Production DOM, screenshots or records.
