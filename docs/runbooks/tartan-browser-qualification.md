# Tartan browser registration and initial acquisition
<!-- stage-runbook: true -->

## Status and boundary

Actual owner-authorized Sandbox run2026-10-24, issues272/88/104/156. Lease
VERIFY272-BROWSER-20261024T0308Z; evidence branch fromc63e640. Owner supplied
managed sign-in/MFA privately and completed Tartan Bank consent. No real bank
credentials, Production account, provider payload or financial row was read or
exported by the operator diagnostics. This standard Tartan case is separate from
the pre-authored four-row oracle; that fixture remains unchanged and unexecuted.

## Stage ledger

| Stage | Actual result | Limit |
| --- | --- | --- |
| Deployment | App release40014820903 transport-smoke-passed; exact images/config digest validated | Not financial acceptance |
| Managed setup | Authoritative pending check empty, fresh synthetic setup prepared/committed/acknowledged | No duplicate earlier UNKNOWN setup or journal command |
| Self-service | New book's Connect account prepared access and one Hosted Link attempt without per-book operator SQL | No general runtime owner grant |
| Provider/recovery | Owner completed Tartan; Success observed; original saved attempt recovered after original link expiry within its recovery window | No second enrollment/Item created |
| Binding | All9 supported mappings reviewed/confirmed; Item active, unfinished attempts0 | Five types explicitly unbound |
| Acquisition | Natural publisher HTTP200; nine consumer HTTP204; exact-book READ ONLY metadata confirms all9 current demands completed and published, historical-ready | No complete financial equality/replay/90-day reconciliation claim |
| Review | Initially blocked by browser sockets; owner re-authentication and one source retry recovered9 source lists/46 visible rows | Chart/category follow-ups remain; no draft/post submitted |

## Current evidence

Trusted build40004622277 attempt1/sourcefb4ac0411f8f5673ab0659825bcafbdb4e64253e:
all seven jobs SUCCESS. App release PR288/c63e6408872dce0a0fd1fcb3312dd8a01387b3f9,
API revision jayburd-dev-api-00020-zpd, web jayburd-dev-web-00020-zb4. All five
PR288 metadata CI checks SUCCESS. Actual migration31 apply/status and verifier
login wiring are recorded in [the rollout](verifier-31-rollout.md).
Fresh selected runtime readback confirms publisher revision
jayburd-dev-acquisition-publisher-00007-945 and consumer
jayburd-dev-acquisition-sync-00007-9k9, each100% traffic. Both use worker digest
3db89dd3f8962cf2369a36c2568a52552820e8b345ad834589cc32c791c4a0ef
from the existing [migration29 worker delivery](plaid-258-rollout.md), not the
new API build's undeployed worker image. Publisher args confirm shared-v1 with
empty fixed binding and the existing scheduler/consumer targets. No env read.
Finite new-enrollment consent ends2026-10-24T23:00Z; no existing Item renewed.

The original saved setup had never been executed. A fresh authoritative pending
check returned no unacknowledged command and All books showed only the original
book. The same planned synthetic entity/book/chart was then reviewed, registered,
explicitly created and acknowledged. New book initially displayed revision4 and
zero position; this is UI evidence, not a journal-count assertion. Public evidence
contains no provider IDs, tokens, credentials, transaction rows or payloads.

After owner completed Tartan, the app required managed sign-in again. The exact
cause of that remount/session loss is unknown. Discovery recovered the existing
link-ready attempt after its03:26:28Z link expiry, while recovery remained valid
until09:26:28Z. Check bank setup completion advanced the same attempt to binding
review; confirmation advanced it to active. No Try again/new attempt was used.

Selected synthetic mappings used distinct new ledger accounts and current dated
USD metadata; no prior accounts were overwritten:

| Profile | Active account count |
| --- | --- |
| checking |1 |
| savings |1 |
| money market |1 |
| cash management |1 |
| credit card |2 |
| student loan |1 |
| mortgage |1 |
| auto loan |1 |

CD, IRA,401k,HSA and home-equity accounts were visibly excluded from binding.
They were discovered, not registered as active sources. This case does not test
consumer/line-of-credit profiles, Liabilities API semantics or foreign currency.

Checking initially showed requested1/completed0, NOT_READY, queued/no worker lease.
Read-only Scheduler metadata then showed ENABLED/every minute, exact existing
publisher target and scheduler OIDC identity. No scheduler/runtime change or manual
sync request was made. Selected request metadata showed publisher HTTP200 for
03:38–03:44Z and nine consumer HTTP204 around03:40–03:41Z. No task/request body,
provider diagnostics, environment or private financial data was inspected.

One bounded READ ONLY transaction, through the existing owner IAM proxy, selected
aggregate metadata only for the exact new synthetic entity/book from the managed
UI setup receipt. It returned:

| Check | Count |
| --- | --- |
| Streams / active / with positive requested demand |9 /9 /9 |
| completed_generation=requested_generation with positive demand |9 |
| HISTORICAL_UPDATE_COMPLETE / publication timestamp present |9 /9 |
| Active leases / current acquisition sets |0 /0 |

These are metadata assertions, not transaction observations. Exact generation
bounds across all streams were not separately selected; generation1 was directly
observed for checking before acquisition. Proxy cleanup ran on exit and private
SQL/proxy diagnostics remained unread. No journal/outbox table query or verifier
mandate was substituted for the independent R02/R03 protocol.

### Browser blocker and usability findings

Subsequent status/source/workspace reads failed in the browser. New public-config
navigation reported ERR_SOCKET_NOT_CONNECTED; its normal reload then displayed
valid public configuration. App fetches still failed. Later failed status reads
were absent from the selected backend request metadata; this does not establish
the browser error's root cause. Normal app reload did not restore its config
fetch and displayed the misleading unconfigured screen.

Browser security review rejected edge:// internal network diagnostics because
only HTTP/HTTPS protocols are allowed. No alternate surface or bypass was used;
owner asked to manually recover the ordinary Jayburd page. Auth stays memory-only,
verified identity/MFA unchanged; no credential extraction or synthetic auth adapter.

Open bounded follow-ups: [289](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/289)
configuration loading/failure/retry, [290](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/290)
opaque account labels, [291](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/291)
read errors versus uncertain mutation recovery. Private local screenshots retain
book creation, active connections and the browser blocker; no screenshots uploaded
to public issue evidence. The misleading expired-link restart prompt was avoided
by reviewing and completing the durable original attempt.

### Recovered review and savings repeat,04:05–04:18Z

Owner signed in again. The current workspace displayed revision13 and11 ledger
accounts (two initial accounts plus nine bank bindings). First source discovery
failed generically; one explicit Retry source accounts returned HTTP200 at both
web/API and exposed all nine active sources. No authentication policy, browser
storage or network-security settings changed. This recovered UI testing; it does
not establish or fix the socket error's root cause.

Using the visible Source account and All statuses controls, every source list
loaded. Each ended on one page with Next transactions disabled:

| Source | Visible rows |
| --- | --- |
| checking |19 |
| first credit card |18 |
| savings |6 |
| money market |3 |
| cash management |0 |
| student loan |0 |
| mortgage |0 |
| auto loan |0 |
| second credit card |0 |

Total46 visible rows; the nonempty tables showed needs-review state. Empty
inboxes are not proof of zero bank/loan balance or complete provider coverage.
Checking Needs review also showed19; Posted returned no matches. No fixture
exercised held rows or pagination here. These are bounded UI counts, not R02/R03.

Current-period checking and card details/category forms loaded. Available
counterpart categories were only the initial control asset and opening equity;
the bank-added accounts did not supply expense/revenue categories. Accounts
exposed the11-account posted-balance/trial-balance read, with no observed
existing-book add-category action. No inappropriate category was chosen to force
a passing draft test. [294](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/294)
requires a reviewed existing-book chart setup contract before implementation.

A pre-book-period card row's selection lost its detail after a generic policy
rejection; a current-period row in the same source worked. Exact backend status/
cause was not captured. ReviewInbox.select's category-error catch clears detail;
[293](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/293) tracks
preserving authorized read-only detail while retaining draft/permission gates.
No accounting date/open period was changed and no draft/journal was submitted.

Savings before repeat: requested1/completed1, HISTORICAL_UPDATE_COMPLETE, no
queue/lease. Request sync now displayed the exact savings-only confirmation and
current Item generation2/binding revision1/stream revision1. Explicit Confirm
sync request recorded requested2/completed1 and queued work; no sibling operation
was requested. Natural scheduling then completed generation2, historical-ready,
no queue/lease, publication2026-10-24T04:14:06.840399+00:00. UI All statuses after
repeat still showed6 rows, source revision1 and no next page, matching the same
pre-repeat view. Needs review also showed6. This is savings L06's bounded first/
repeat UI+metadata acceptance, not an independent transaction-field/replay proof.

Final live tab is left on the populated checking inbox for owner review. Private
screenshots record checking rows and savings repeat; none uploaded to public
issue evidence. Lease VERIFY272-REVIEW-20261024T0406Z owned only documentation
and actual UI checks; source fixes remain unimplemented/unclaimed issues.

## Remaining work

Implement reviewed category-account setup and safe detail/error recovery, then
exercise legitimate categorization/draft/post policy with suitable synthetic
counterpart accounts. Held cases, larger pagination and other source repeats are
not covered by this run. Savings first/repeat now has bounded UI/metadata proof.
Author a new isolated custom-user case for the independent four-row fixture; never
adjust expected amounts/identities to match default Tartan observations. Complete
identity-only map, finite one-source mandate, equality/no-journal comparison and
cleanup before clearing R02/R03. OAuth/reconnect/webhook/multicurrency/90-day
coverage and Production gates retain their existing separate evidence requirements.
Issue281's future consent-expiry outage is not fixed by this run.

## PR handoff

Documentation-only patch; no new migration/product code/cloud configuration. Registry owns
lease state;272/88/156 remain open. Exact source/receipt and scope above distinguish
actual managed setup/recovery/initial acquisition from synthetic CI and unexecuted
financial proof. No financial conformance or production-readiness claim.
make check/test/build PASS with documented cached local Go1.26.5 temporary
modfile; tracked1.26.10 unchanged. Canonical source/image/browser/API/SQL gates
already passed in the trusted application build. This documentation adds no
product/browser/API code or database migration requiring new integration gates.
Review follow-up make check/test/build also PASS with the same documented local
toolchain; staged runbook validation required before its evidence commit. The
actual scoped savings sync request is recorded above, not hidden as a read-only test.
