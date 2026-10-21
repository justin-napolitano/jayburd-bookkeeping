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
| Browser review | Pending. CUA blocked the local `file://` URL and explicitly disallowed an indirect browser workaround, so there are no verified desktop/mobile captures or visual keyboard/overflow claims. |
| Repository gates | `make check`, `make test`, and `make build` PASS with local Go 1.26.5 and a temporary modfile outside the repository. `make test` required approved unsandboxed execution for synthetic localhost listeners. Script syntax and fixture interaction smoke PASS. Staged runbook guard recorded at commit. |

## Current evidence

Static script syntax and required control IDs passed with Node's `vm.Script`.
A DOM stub exercised first-run Connect → All Transactions → card detail → Books
→ denied-state clearing; all assertions passed. `git diff --check` passed.
Browser visual inspection is pending because CUA rejected opening the local
`file://` prototype: “The requested URL protocol is not allowed. Allowed
protocols: `http:`, `https:`.” The tool explicitly prohibited using a localhost
or alternate-browser workaround.

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
   document itself should not overflow.

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
production implementation. Visual acceptance remains pending.
