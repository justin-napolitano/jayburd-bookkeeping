<!-- stage-runbook: true -->
# In-app private onboarding UI — 460

## Status and boundary

Implementation branch feat/460-onboarding-ui from b2e6621. Root owns the frozen
collection-onboarding contract, HTTP/SQL/shared transport and canonical CI. This
lane owns the onboarding client/coordinator, Bookkeeping mounting, focused tests
and new actual synthetic browser fixture/journey. No cloud, provider, real identity
or financial data was accessed. Setup is server-advertised explicit self-service;
existing capability restoration and provider enrollment without a book are excluded.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Contract review | Read frozen collection-onboarding v1. Identified missing command prepare_key echo; coordinator approved required echo for exact uncertainty binding. No caller actor/capability/plan fields accepted. |
| UI and recovery | Always-mounted third principal recovery gate, explicit server preview/save/execute/ack, source-set adoption and legacy-pause disclosure. Initial unsupported404 only before known work; other failures remain locked. Exact receipt/key/state/review pins, expiry-safe uncertain retry, current-session epoch fences and SHA256 pins after denied payload destruction. |
| Scope gates | Sibling busy errors cannot deadlock the authoritative own onboarding claim; all new book/private work stays gated. Setup eligibility is read from explicit catalog. Read-only workspace cannot self-elevate. Registration/other acknowledgements refresh discovery. |
| Focused checks | 11 onboarding unit tests PASS. Full existing+new 618 unit tests PASS. TypeScript noEmit PASS. No dependencies added; npm ci installed locked dependencies with existing cache (offline attempted first; missing cached package required ordinary registry install). |
| Actual proof authored | New isolated fictional actor094 owns a legacy Item/source and two exact original-unit observations, but no private workspace or journals. Actual Bookkeeping→Next→Go→SQL journey authors setup/adoption and injects only completed response loss, including full remount recovery and explicit acknowledgement. Awaiting root-composed service/47 for execution. |
| Independent wire review | Corrected actual47 microsecond timestamps, explicit cancelled:true terminal result and new private source aliases in adoption receipts. Added independent focused regression and kept complete original-source selection distinct from returned aliases. Later404 after any supported discovery stays locked; ready/no-candidate copy and normal Connections navigation stay honest. Imported reviewed shared1e3a4cf/fe60fc3/11c8656 only to align the actual transport. |
| Closure and asynchronous review | Required command recovery_id now remains with opaque saved identity and must match available pending/replayed transitions. A denied payloadful acknowledgement can explicitly resolve only that exact payload-free closure. Digest creation is inside guarded try/finally and pending-read generations release their own busy state after superseding a slow catalog. Own-claim sibling panels stay mounted but hidden until fresh post-ack recovery. Final focused12 onboarding units and TSC pass; composed browser proof still awaits actual47. |

## Current evidence

Unit tests exercise explicit mutation boundaries, complete source review, initial
feature absence, malformed/current authority failures, unknown prepare with exact
key after expiry, execute-loss/remount, ack-loss with pending:none, switched
terminal receipt rejection, API replacement and no read-only self-elevation.
The browser integration proof is authored, not yet execution evidence. Existing
operator-plan activation is not used or advertised by this UI.

## Remaining work

Finish independent mobile/keyboard/composed gate checks and execute the new
actual integration fixture after root imports SQL47/service/bridge. Run production
web build and canonical combined gates. Review all final source under current lease.

## PR handoff

Links issue460 and coordinator dispatch65. No migrations or shared HTTP contract
files edited in this lane. Do not merge independently; root imports bounded commits,
checks actual browser/SQL postconditions and owns GitHub delivery state.
