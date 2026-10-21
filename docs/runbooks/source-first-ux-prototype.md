<!-- stage-runbook: true -->
# Source-first workspace prototype

## Status and boundary

Issue #246, lease `UX-246-20261021T2315Z`. This branch adds one standalone,
fixture-only HTML design preview. It makes no production route, API, SQL, provider,
or accounting change. The prototype is not evidence that the connect-first epic
has been implemented or that any account is ready for live bookkeeping.

## Reviewed basis and design choice

- Read `docs/research/bibliography.md` first, then the selected internal contracts:
  `docs/contracts/source-review-sql.md` (source review and explicit posting),
  `docs/contracts/plaid-history-currency.md` (requested history versus observed
  availability), `docs/contracts/plaid-liability-bindings.md` (credit/loan review
  limits), and `docs/contracts/plaid-decoder.md` (provider categories are inert
  source metadata). Existing product expectations are in issues #104, #52 and #9.
- Current review reads are book and source scoped. This preview explores a global
  owner workspace before those contracts are changed; it does not imply an
  authorized global API already exists.
- Chosen layout: connection inventory followed by All Transactions, then a
  detail view with distinct owner topics, suggestion provenance, and book-specific
  accounting state. A book-first layout was considered, but it cannot show
  unassigned source history as the owner requested. A single merged financial
  total was rejected because separate entities and currencies cannot be pooled.
- First-release suggestions in this fixture come from owner rules or inert
  provider metadata. Model suggestions remain future work until a separate
  evaluation and authority contract is reviewed; no model result is simulated.
- The initial history request can be up to 90 days; the interface reports only
  observed coverage. A simulated Item includes checking, credit card and student
  loan accounts. Liability entries stay held for later accounting policy. Source
  amounts remain decimal strings in USD or EUR.
- The prototype adds no new financial message or data contract. Existing
  proprietary Plaid adapter and ISO 4217 currency applicability assessments
  remain in the cited contracts. No ISO or accounting conformance is claimed.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source and issue review | Read issue #246 and current docs/index, bibliography, source-review, history and liability contracts before implementation. |
| Prototype | `docs/prototypes/source-first-workspace.html` shows first-run simulated Connect, three account statuses, two entity books, six fictional transactions, global search/filter, detail, topic toggle and distinct classification authority. No network or browser persistence. |
| Display states | Loading, empty, unavailable and denied replace prior transaction content. Native controls, skip link, focus transfer, semantic tables, labels and mobile layout are built in. |
| Rule-first review correction | TX-101 now shows an owner rule suggestion. No active model result appears; later model proposals require evaluation and a reviewed contract. Focused static assertion and `make check` pass. |
| Browser review | Owner explicitly requested an HTTP-hosted walkthrough. Localhost Edge rendered the fixture at `http://127.0.0.1:8765/source-first-workspace.html`. Desktop and 390×844 mobile views, keyboard activation, search/filter, detail, topic edit, ownership notice, and replacement loading/empty/error/denied states were inspected. |
| Browser corrections | The sidebar now reports three fictional accounts after simulated Connect; the topic notice clears when returning to the list; mobile book cards stack instead of squeezing into two columns. The mobile document width stayed at the viewport width while its transaction table scrolled inside its wrapper. |
| Repository gates | `make check`, `make test`, and `make build` PASS with local Go 1.26.5 and a temporary modfile outside the repository. `make test` required approved unsandboxed execution for synthetic localhost listeners. Locked `npm ci` and `make web-check web-test web-build web-browser` PASS (286 unit and 17 browser tests). Script syntax and fixture interaction smoke PASS. Staged runbook guard recorded at commit. |

## Current evidence

Static script syntax and required control IDs passed with Node's `vm.Script`.
A DOM stub exercised first-run Connect → All Transactions → card detail → Books
→ denied-state clearing; all assertions passed. `git diff --check` passed.
The original `file://` preview was rejected by CUA: “The requested URL protocol
is not allowed. Allowed protocols: `http:`, `https:`.” On the owner's subsequent
explicit request to host and use the fixture, a local HTTP listener and Edge
browser opened it successfully. The desktop screenshot showed the source-first
overview and cross-account table. A mobile viewport showed the transaction
detail, scrollable table, and corrected single-column entity cards. Browser
DOM measurements showed no document-level horizontal overflow at the mobile
viewport; the transaction table itself had an internal horizontal scroll area.
The browser images were inspected in-session and a separate Playwright capture
saved `/private/tmp/jayburd-246-{desktop,mobile}-{transactions,books}.png` for
local owner review; no image artifact is committed. That script checked six
rows, document width, connection-status text and mobile book grid columns.

Keyboard Enter activated All Transactions and a filtered Review button. Search
for `TX-099` selected one cross-entity posted example; loan + Held selected
one EUR fixture; loan + Posted produced the explicit no-match state. The card
topic toggle changed only its topic while its held accounting state stayed
unchanged. A posted cross-entity example likewise kept its journal evidence
after a topic toggle. The ownership action reported that no assignment was made.
Loading, empty, unavailable, and denied previews replaced prior transaction
content rather than leaving stale rows visible.

The pinned Go 1.26.10 download failed due unavailable network access. A copy
of `go.mod`/`go.sum` under `/private/tmp` changed only the Go directive to the
installed 1.26.5 for these gates; tracked module files are unchanged. Initial
sandboxed `make test` failed solely where `httptest` and worker tests could not
bind localhost; the approved unsandboxed rerun passed.

### Explicit review scenarios

1. Open the file: first-run Connect explanation says the button is simulated.
   Activate it with Enter; three fictional accounts appear with distinct history
   coverage and legal-owner review status.
2. Open All Transactions; search and filter across accounts, including unassigned,
   held and posted examples. Original USD and EUR labels stay on each row, with no
   pooled total. An empty search result is not presented as complete bank history.
3. Open a card transaction: Plaid category is a suggestion with provenance,
   owner topic is an analytic label, and the bookkeeping section has no draft or
   journal. Toggle a topic; no book or posting status changes.
4. Open posted checking activity: posted journal evidence is separate from the
   topic. Inspect the cross-entity example: source ownership and book treatment
   are separately explained, with no automated allocation asserted.
5. Preview loading, error and denied; prior transaction rows disappear. On a
   narrow screen, navigation and table remain reachable by keyboard and the
   document itself should not overflow. These behaviors were observed in Edge
   on the local HTTP fixture; no production accessibility certification follows.

## Remaining work

`web/src/bookkeeping/BookWorkspace.tsx` and `ReviewInbox.tsx` currently select a
book/source before review. `docs/contracts/source-review-sql.md` has no global
transaction query. The synthetic topic toggle and suggestion presentation have
no authority, audit or persistence implementation. Book assignment, transaction
allocation, liabilities, model evaluation, search pagination and 3,000-item
performance require their own reviewed contracts and tests. A visual prototype
does not prove accessibility, permissions, sync completeness or posted balances.

## PR handoff

This bounded branch changes only the prototype and this stage runbook. The
coordinator owns shared index updates, issue discussion, PR creation and any
production implementation. The browser walkthrough is complete; product and
accounting acceptance remain separate.
