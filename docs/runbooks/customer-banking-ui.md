<!-- stage-runbook: true -->
# Customer banking experience — 479

## Status and boundary

Owner screenshot and explicit request authorize removing debugging features and
polishing the existing banking/accounting workflow. Fresh main0fb98416 includes
the accepted artifact-policy release record. Dedicated feat/479-product-ui;
GitHub479 owns delivery,65 owns the source claim through2026-11-09T09:00Z.
No auth/API/SQL/financial-rule/dependency/provider/private-record/cloud change.
Existing Sandbox00026 remains unchanged; SQL1–47 immutable,48 unallocated.

## Design decision before implementation

Reuse the reviewed first-party flow/accessibility references in
docs/research/frontend-sources.md and the owner screenshot as concrete evidence.
This is a presentation change; no new financial message, money/currency mapping,
accounting rule or export. Existing standards applicability is unchanged.

Keep the familiar green/white identity. Palette: forest#286047, ink#1f2933,
muted#647067, white#ffffff, canvas#f5f7f6, border#dce3df. Use the existing system
sans family with a restrained title/body hierarchy and tabular monetary figures;
no font/dependency download. Left-aligned content; one compact header with primary
Connect bank and quieter Refresh, followed by readable connected-account rows.
Avoid giant nested panels, repeated status prose and developer identifiers.

Accounts header     [Connect bank] [Refresh]
Bank connection    [Connected]
  Account name     Import status    [Transactions] [Manage]
Relevant interrupted setup only    [Resume] [Cancel]

Critique against the brief: a bank-like account list is useful here because the
user manages accounts and their transaction evidence, not generic dashboard
cards. No decorative totals, invented balances/bank names or placeholder document
feature. Do not merely collapse technical inspectors: remove their customer UI.
Keep legitimate source evidence and receipts in their relevant financial views.

Raw-ID lookup forms and permanent debug/recovery inventories leave the normal
application. Recovery remains contextual and preserves discovery, current locks,
same-request retries, explicit confirm/ack and safe cancel/disconnect. Older-page
discovery and expired/unknown/revoked cases must not become unreachable.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Grounding | Owner screenshot shows provider technical details, known-ID inspector, permanent recovery and competing actions. Source also exposes command IDs, idle recovery/settings and obsolete manual-entry sign-in copy. |
| Scope | Root shell/navigation/styles/docs; disjoint workers own connection presentation, saved-command presentation and proof review. Existing financial/recovery contracts remain governing. |
| Connections | Technical connection/account/attempt IDs and arbitrary-ID inspector removed. One Connect bank action, quiet Refresh/status and compact account rows; known discovered unfinished/expired setup retains exact inspect/retry/binding/cancel/disconnect behavior. No history-only panel when global unfinished_count is zero, even with a cursor; global unfinished work retains earlier-page discovery. |
| Saved work | Fully idle CommandPanel renders nothing; checking/interrupted/prepared/denied/committed states retain safe actions. Raw command/state/JSON inventory gone; exact financial lines/currency/valuation and posted-entry references remain. Last checked wording avoids cross-tab authority claims; denied/error does not display stale loading prose. |
| Shell/evidence | Banking/Accounting navigation, Review accounting label and Refresh books; Manage books appears on Overview/initial setup. Removed obsolete manual-entry sign-in claim and development diagnostics footer; compact Accounting preview remains truthful. Tag results name the exact retained selection; internal IDs/revision pins stay unchanged and currency code/date/precision remain readable. |
| Independent review | No handler/authority/body/key/pinning/money regression found. Review identified unsupported retry timing; copy now distinguishes queued/active/waiting to retry without timing promises. Root confirmed global unfinished count semantics in immutable SQL14 before hiding cursor-only terminal history. |
| Local frontend | All673 units, TypeScript/generated API, production Next build and rebuilt fictional self-contained review PASS. Full98 CT PASS, then final9 affected cases PASS after last CSS/loading changes. Older-page recovery, no-debug idle/denied/prepared states, narrow layout and exact financial/replay/report/CSV/drill assertions retained. |
| Visual proof | Five widths390/768/1024/1280/1440 for Accounts/full shell and prepared/denied recovery. Final ignored captures .local/customer-banking-final-polish-proof; no overflow/clipping and44px+ safe-exit targets. Root/QA inspected mobile/desktop fictional captures. CI upload allowlist adds only exact fictional Accounts/saved-work/idle PNG patterns beside existing review HTML/PNG. |
| Core/limits | make check and core build PASS using installedGo1.26.5/external exact-dependency modfile; no tracked dependency/toolchain change. Full make test attempted: four existing socket suites acquisition/webhookhost/workerhost/workerruntime fail EPERM. Standard real --connections runner stops upfront Synthetic proof port unavailable before fixture/Docker creation. No host, timeout, permission or assertion bypass; canonical exact-head CI remains required. |
| Canonical copy correction | PR480 head7f40f2e, run42559244291/job128971799313 passed real labels, onboarding and API/PostgreSQL recovery. The source journey failed at connections.spec.tsx182 after pause succeeded: the assertion still expected Account acquisition: paused, while the customer UI correctly returned Account sync is paused. Corrected only three paused/active text expectations across Connections and fictional Production. Existing exact request/key replay, source0.10, journal/report/CSV/drill and authority assertions remain intact; source-stage completion and later Production/demo gates require a fresh exact-head run. Frontend job128971799634 passed. |

## Current evidence

Design and invariants recorded before implementation. Source now implements the
customer cleanup and local proof above. Synthetic fixtures only; no live
authenticated app inspection or financial records used. Existing API/SQL/financial
contracts and package lockfiles are unchanged.

## Remaining work

Publish combined source and require every exact-head canonical gate, especially
real Next/Go/restricted SQL source/recovery/report/demo journeys. Existing Sandbox
web-only release follows trusted main image/runtime/scan/receipt acceptance and a
fresh cloud lane; API artifact enforcement remains on its accepted image. No
financial mutation, provider fetch, consent renewal or SQL change is authorized
by this source claim. Owner visual/private-use acceptance remains separate.

## PR handoff

Implementation/local review ready; canonical source/build/runtime acceptance is
pending. Governing artifact policy473 and future document ingestion/matching/
corrections476 remain unchanged. Existing Sandbox00026 is unchanged so far.
