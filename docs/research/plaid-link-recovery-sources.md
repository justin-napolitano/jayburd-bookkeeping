# Link recovery source register — 2026-10-22

Selected official sections read in browser; mutable guides, not certification.
Pinned API version remains 2020-09-14, Go SDK v49.0.0. No credentials or provider
payloads were accessed; Dashboard capability/Launch Center was not reviewed.

| ID | Source / reviewed section | Established fact and limit |
| --- | --- | --- |
| LR01 | [Hosted Link](https://plaid.com/docs/link/hosted-link/), Overview/Who should use it, obtaining token and testing | SDK recommended for owned frontend; Hosted supports update mode. This does not prove Jayburd's update-session association. |
| LR02 | [Update mode](https://plaid.com/docs/link/update-mode/), Using update mode, requesting accounts, refreshing expiration | Existing access token initializes repair and remains unchanged; account-selection widening is explicit. It does not authorize local consent extension. |
| LR03 | [Link API](https://plaid.com/docs/api/link/), token-get link_sessions/results/exit/on_success/on_exit | Modern Item-add results differ from deprecated callback fields; modern exit reports unsuccessful exit. Ordinary update success association remains unresolved; CRA update is a different product. |
| LR04 | [Web SDK](https://plaid.com/docs/link/web/), create/onSuccess/onExit/destroy/OAuth navigation | SDK callbacks and lifecycle are browser mechanisms. They do not replace server authority/session proof. Redirect details must be reviewed again before implementation. |
| LR05 | [Hosted Link](https://plaid.com/docs/link/hosted-link/), storing associated token/obtaining public token; [Link API](https://plaid.com/docs/api/link/), token-get session results; [invalid Link token](https://plaid.com/docs/link/handle-invalid-link-token/), opening recovery sections; refreshed 2026-10-22 | Retain the user-associated Link token and retrieve bounded completed-session results; modern results preferred. URL expiration does not prove no completed session. Explicit restart must preserve Jayburd's one-shot claim and cancellation policy. |
| LR06 | [React19.3 useLayoutEffect](https://react.dev/reference/react/useLayoutEffect), reference/caveats and before-paint state updates, 2026-10-22 | Client layout effects finish state updates before repaint. Apply only to the existing small synchronous review initializer; keep the reviewed date/edit semantics and scope reset. No server effect or accounting rule is introduced. |

Also inspected pinned generated `link.go`: `LinkSessionResults` has Item-add and
CRA update fields; `LinkTokenGetSessionsResponse` contains modern exit and legacy
success fields. Model availability alone does not establish actual provider
semantics or account enablement. No fallback to deprecated fields is approved.

Design alternatives, inferences, uncertainty and required acceptance are in
[the recovery contract](../contracts/plaid-link-recovery-qualification.md).

## LR07 — owned Hosted repair review, 2026-10-24

Read full relevant official sections: [Link API](https://plaid.com/docs/api/link/),
SESSION_FINISHED properties/default enablement; [Hosted Link](https://plaid.com/docs/link/hosted-link/),
supported flows, obtaining session results, testing; and [update mode](https://plaid.com/docs/link/update-mode/),
Using update mode with access token, requesting accounts and consent revocation.
Mutable documentation; pinned API2020-09-14/SDK49.0.0 unchanged.

The modern event supplies outcome, Link token/session and environment, without
Item or completion time; Hosted gets this event by default. Update uses the old
credential without exchange; deselection can remove previously shared accounts.
Sandbox Hosted update can expire after30minutes; our shorter local deadline must
remain authoritative. Model fields alone do not qualify actual provider behavior.

The retained server-created hash→owned attempt→preselected Item/streams join and
proof-gated owner claim are Jayburd's inference, not a provider guarantee. No
ordinary update response, Dashboard setting, actual event, credentials or financial
payload was accessed. Existing verifier checks in `internal/webhook/verifier.go`,
new-only handoff19, connection fencing13 and exact bootstrap ACL were inspected.
The [proposed protocol](../contracts/hosted-reconnect.md) defines alternatives,
ambiguity/expiry/lock boundaries and named future evidence. No runtime acceptance
is claimed, and no deprecated callback or CRA result is substituted.

Wave 2 additionally inspected `HTTPProvider.CompletedPublicToken`, `VerifySession`,
service completion and merged migrations 13/19/23: the new-mode verifier repeats
the same provider association check after a claim. Reusing private verified
hash evidence is our engineering inference, not a Plaid recommendation or a
relaxation of current SQL authority. Physical ciphertext deletion is not
implemented. Read depth is the named sections/model functions only; no Dashboard,
provider payload, credentials or full API review.
