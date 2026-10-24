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
| Review | Browser transport failed before transaction review could load | Owner manual browser recovery requested |

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

## Remaining work

Recover browser connectivity and inspect review rows/categories/holds, then test
unchanged repeat acquisition without duplicate effects. Savings initial bind/sync
now has bounded metadata evidence, but L06's repeat/review acceptance remains open.
Author a new isolated custom-user case for the independent four-row fixture; never
adjust expected amounts/identities to match default Tartan observations. Complete
identity-only map, finite one-source mandate, equality/no-journal comparison and
cleanup before clearing R02/R03. OAuth/reconnect/webhook/multicurrency/90-day
coverage and Production gates retain their existing separate evidence requirements.
Issue281's future consent-expiry outage is not fixed by this run.

## PR handoff

Documentation only; no new migration/product code/cloud mutation. Registry owns
lease state;272/88/156 remain open. Exact source/receipt and scope above distinguish
actual managed setup/recovery/initial acquisition from synthetic CI and unexecuted
financial proof. No financial conformance or production-readiness claim.
make check/test/build PASS with documented cached local Go1.26.5 temporary
modfile; tracked1.26.10 unchanged. Canonical source/image/browser/API/SQL gates
already passed in the trusted application build. This documentation adds no
product/browser/API code or database migration requiring new integration gates.
