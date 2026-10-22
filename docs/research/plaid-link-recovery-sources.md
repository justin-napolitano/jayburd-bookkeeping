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

Also inspected pinned generated `link.go`: `LinkSessionResults` has Item-add and
CRA update fields; `LinkTokenGetSessionsResponse` contains modern exit and legacy
success fields. Model availability alone does not establish actual provider
semantics or account enablement. No fallback to deprecated fields is approved.

Design alternatives, inferences, uncertainty and required acceptance are in
[the recovery contract](../contracts/plaid-link-recovery-qualification.md).
