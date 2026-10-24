# Link recovery qualification and SDK transition

The bounded #275 [owned Hosted repair protocol](hosted-reconnect.md) proposes the
signed-session association for ordinary update mode. It does not enable repair or
supersede the eventual SDK direction. It supplies a separate protocol before code;
the historical requirements below must not be read as implemented guarantees.

## Decision record — #104 / #187, 2026-10-22

Status: proposed transition; current runtime remains Hosted Link. The owner
authorized completing the qualification plan. This record makes the remaining
security and recovery design reviewable; it does not enable SDK/update mode.

Recommended product direction: official Web SDK for the owned frontend, after
server association and OAuth return recovery are qualified. Retain the deployed
Hosted profile until that replacement passes. The current HTTP provider rejects
reconnect before attempt creation or exchange claim; preserve that guard.

Reviewed official sources and exact sections are recorded as LR01–LR04 in
[the source register](../research/plaid-link-recovery-sources.md). Hosted Link
supports update mode, but that capability alone does not establish which modern
server result proves successful ordinary Item update in this integration.

| Alternative | Benefit | Remaining cost / risk |
| --- | --- | --- |
| Official Web SDK | Vendor recommendation for owned frontend; direct lifecycle callbacks and OAuth support | Requires qualified server association, redirect resume and device evidence |
| Retain Hosted Link | Existing bounded encrypted handoff and observed non-OAuth enrollment | Extra tab; explicit completion; ordinary update association still unqualified |
| Accept callback or successful accounts lookup | Small implementation | Rejected: client metadata/current credentials do not prove this session's repair |

This is proprietary bootstrap/authorization behavior under the existing STD-01
assessment, not an ISO 20022 exchange. Currency/date/accounting mappings remain
unchanged. No new financial record or accounting policy is introduced.

## Minimal transition requirements for coordinator review

1. **Versioned profile:** explicitly select Hosted or SDK in reviewed release
   metadata. Do not infer SDK enablement from callback success. Qualify actual
   Link Events availability; no fabricated `link_events_qualified=true`.
2. **Server association:** define the exact current modern server response/event
   proving success for new and update sessions. Match retained Link-token hash,
   current actor/entity/book/attempt, unique session, completion time and expiry.
   For repair also match the preselected Item and approved streams. Deprecated
   `on_success`, browser account IDs and a healthy `/accounts/get` alone remain
   insufficient. Obtain provider clarification if ordinary update proof is absent.
3. **SQL:** migration 19 handoff write requires new/link_ready/unclaimed/unexpired;
   its read rechecks current authority, new mode and the recorded token hash.
   The completion service additionally requires link_ready and a live recovery
   window before using that read. Read alone does not impose state/expiry guards.
   A new reviewed migration must extend precisely the intended mode and current
   authority checks before reload recovery for reconnect. Never edit merged SQL.
   Preserve shared-Item generation/stream fences and one-shot claim. Repair must
   reuse the existing access credential and Item: no public-token exchange.
4. **OAuth return:** an authenticated same-origin resume route resolves a scoped
   unclaimed attempt and decrypts its retained Link token server-side; returns
   transient initialization material only after renewed authority. Browser
   persistence may hold no Link/public/access tokens or financial requests.
   If an opaque correlation handle is needed, bind it server-side to current
   actor/scope and expiry; a URL handle alone grants no authority. Treat OAuth
   return parameters as transient vendor input, suppress logs/analytics, and
   remove them from the displayed URL after initialization. Review exact SDK
   `receivedRedirectUri` requirements before implementation.
5. **UI:** show repair only when supported and authorized. Reconnect begins with
   explicit Item/generation/full approved stream confirmation; no new-product
   or new-account consent is implied. Account-selection changes require their
   own scope review. Cancellation never silently reconnects or creates a new Item.
6. **Duplicate prevention:** institution/mask/name are warnings, not ownership
   keys. Compare provider-confirmed selection before exchange where supported;
   distinguish same credentials from legitimate distinct logins and partial
   selections. SQL still prevents duplicate Item/account bindings after exchange.
   Resolve pre-exchange provider metadata and cross-book privacy rules first.
7. **Transport:** signed `SESSION_FINISHED` is notification, not authority to
   exchange blindly. Durable event receipt and user-driven completion share one
   claim. Do not execute completion from public webhook handler without a scoped
   durable operation and explicit reviewed execution contract.

## Synthetic evidence versus hosted execution

### Implemented bounded exit recovery

One matching token response with exactly one finished, fresh session, a bounded
session identifier, a modern `exit` object and no `results` object yields internal
`link_session_exited`. Both a null error (cancellation) and a non-null error
(failed exit) receive the same fixed recovery guidance; no vendor code/message
is exposed. Unfinished, multiple, missing-identifier or legacy-only exits stay
unresolved. Valid completed Item-add results retain their existing precedence.
This classification does not assert that a provider Item is absent.

Before any SQL claim, service maps that classification to existing HTTP 409
`connection_operation_blocked`. Both intact-tab and discovered-attempt UI paths
inspect the exact attempt and refresh authorized discovery before offering the
existing explicit **Cancel setup** / **Try again** actions. A competing completion
instead displays its actual durable state. No automatic abandon, replacement,
exchange, terminal state or API schema change is introduced. Read failures and
revoked scope keep existing fail-closed handling.

Acceptance: `TestHostedExitPreservesClaim`, modern exit/error/unfinished/legacy/
multiple cases in `TestHostedLinkCompletion`, and “offers explicit recovery after
a completion conflict” for same-tab and reload flows. These prove no claim,
exchange, secret publication or automatic cancellation/restart.

`TestHostedLinkCompletion` rejects unfinished/legacy-only/null/ambiguous/stale/
future completion evidence. `plaidLink.test.ts` enumerates 17 applicable Link
error codes and proves fixed diagnostic text, late-callback fencing, no browser
persistence, no automatic reopen and fresh explicit-session recovery. These are
synthetic protocol/privacy proofs; none clears the actual provider error rows.

Use [the #88 matrix](../plans/plaid-88-test-matrix.md) for clearance authority.
For every hosted case record source SHA, deployed revisions/profile, matrix ID,
public fixture name, browser/OS/device, timestamp, action, redacted terminal
status, expected/actual result, evidence location and defect/retest link. Do not
retain provider payloads, tokens, institution account identifiers or screenshots
containing financial records. Failed/blocked cases stay open.

| Rows | Fixture/action and acceptance |
| --- | --- |
| L02–L04 | Complete, cancel, close/reload and expire separate attempts; each resolves its exact durable attempt, at most one exchange/credential, no automatic replacement. Distinguish URL expiry from six-hour result recovery. |
| L05 | Same login twice, distinct login at same institution, overlapping and disjoint selected accounts; no silent duplicate imports and no institution-only blanket rejection. |
| L06 | Public simple-user savings/depository fixture; explicit account binding, first publication and unchanged repeat; reconcile expected scope. |
| O01 | Desktop OAuth happy path, cancel, missing return/expired session and wrong signed-in scope; return to original attempt without persistent tokens or duplicate exchange. |
| O02–O03 | Real supported iOS/Android browsers, institution app installed/absent; record versions and separate each outcome. Keep mobile unqualified until observed. |
| O04 / I01 | On an explicitly authorized fictional Item induce `reset_login`, observe actionable login-required state, repair in update mode and verify same Item/credential/account continuity and resumed sync. Blocked until server association implementation above. |
| O05 / I02–I03 | Revoke/reduce consent, expiration and new-account notices; no acquisition outside explicit current grants and no automatic widening or local expiry extension. |
| I04 | Account disconnect versus full Item removal; explicit confirmation, ambiguous reply inspection, stale-worker fencing and retained approved immutable history. |
| E01 | Each public OTP/questions/selections/multiple-round fixture: success, wrong response and cancel; no exchange before verified completion; inspect/recover exact attempt. |
| E02 | Owner-operated reCAPTCHA success/failure; agent stops at challenge; safe cancellation/retry. |
| E03–E04 | Each of the 17 named matrix codes separately at a supported non-OAuth test institution; verify fixed/redacted application message, retained attempt and explicit recovery; a fake SDK exit is insufficient. |
| E05 | OAuth insufficient-credentials/access-denied selector, missing institution/account and cancel; no scope widening, duplicate Item or automatic replacement. |

Native/webview and UK/EU variants remain excluded only by the current explicit
US web profile. Product/platform expansion reopens their applicability.
