# Frontend source review

## Customer banking cleanup — 2026-11-09,479

Owner screenshot and explicit feedback identify the remaining technical
inspectors, raw-ID lookup, permanent recovery inventory and competing green
actions as customer-flow clutter. Reuse the selected flow/accessibility sources
below as historical design guidance, not as a new current competitor audit.
The owner request supersedes the earlier choice to merely collapse inspectors.

Choose compact account rows with one Connect bank action and quieter refresh,
transaction navigation and management. Remove technical display and arbitrary-ID
input, while retaining scoped discovery and all actionable interrupted/expired/
unknown recovery. Exact internal IDs, revisions and request keys do not change.
Currency code/date/precision and meaningful posted-entry references remain;
routine UUID display does not help users review financial treatment. Tag result
labels come from the exact saved review selection, not guessed merchant data.

No financial wire, schema, currency meaning or posting rule changes; existing
standards applicability remains unchanged. Compare rendered fictional five-width
screens and preserve source/replay/authorization/report/CSV/drill assertions.
The contract and stage evidence are in
[customer banking UI](../runbooks/customer-banking-ui.md).

## Responsive flow references — 2026-11-08,467–471

Owner approved implementing the navigation/setup/transaction-control plan for
review. Preserve the current white/dark-green system and existing authority,
exact money, command confirmation and recovery boundaries. No financial message,
currency definition, accounting rule, database schema or API profile changes;
the existing standards applicability remains unchanged.

| First-party source and reviewed section | Evidence and application |
| --- | --- |
| [Monarch, Quicker and easier transaction review,2024-06-20](https://www.monarch.com/blog/quicker-and-easier-transaction-review-and-more), Transactions Improvements | Read full relevant section. Consolidates filtering, exposes account/category detail in rows and gives prominent detail actions. Apply compact transaction controls and detail-return continuity. This historical vendor example does not establish their current UI or Jayburd usability. |
| [Actual Budget, Transaction Rules](https://actualbudget.org/docs/tour/rules/), short tour and conditional rule example | Read full short page. Conditions lead to understandable payee/category actions. Preserve readable existing rule review; do not introduce automatic execution or reuse another product's financial semantics. |
| [GOV.UK Check answers](https://design-system.service.gov.uk/patterns/check-answers/), relevant guidance and example markup; [Confirmation pages](https://design-system.service.gov.uk/patterns/confirmation-pages/), relevant guidance and example | Read selected guidance/example sections. Review before submission and a clear completion/next step support the guided panel. Jayburd still saves a durable review, explicitly executes and explicitly acknowledges; progress styling never authorizes a mutation. |
| [actions/upload-artifact](https://github.com/actions/upload-artifact), inputs and Uploading Hidden Files | Read selected sections. Hidden directories are excluded by default. The existing pinned action explicitly selects one fictional self-contained HTML under .local and fictional frontend PNGs; include-hidden-files is necessary for that exact file, not permission to upload a directory or environment/configuration. |

Choice: persistent sidebar above960px and explicit menu below, with private
Transactions separated from book-scoped Imported activity/ledger. Alternative:
compact top bar; rejected for this first owner review because six destinations,
scope and utilities competed for width. Retain existing component/state machines
rather than a framework rewrite or another design-system dependency. These are
design inferences requiring rendered acceptance, not conclusions from a captured
current hosted audit. Original owner screenshots are historical grounding only.

Acceptance: focused lock/progress/filter/selection tests, five-width synthetic
component screenshots, keyboard/detail-return checks, current permission and
lost-response regressions, actual Next/Go/restricted SQL journeys, and owner
visual review. Initial local capture/server failures and browser local-file
policy rejection are documented in the delivery runbook; do not infer full
accessibility, live bookkeeping or deployed acceptance from source/build success.

## Decision

Use the repository-selected Next.js/TypeScript stack for artifact-backed bookkeeping. Browser
code renders exact decimal strings and sends reviewed requests; SQL remains the
accounting authority. This is an implementation decision for #9, reviewed 2026-10-11.

| Source and reviewed section | Evidence and interpretation |
| --- | --- |
| [Next.js 16.4 installation](https://nextjs.org/docs/app/getting-started/installation), manual installation and scripts | Reviewed relevant sections. Next separates build from lint/type checks; this package runs explicit TypeScript, unit, browser and build gates. |
| [Next.js rewrites](https://nextjs.org/docs/app/api-reference/config/next-config-js/rewrites), external URLs | Reviewed external rewrite section. A fixed operator-selected backend allows same-origin bearer requests. Browser-controlled upstream destinations are prohibited. |
| [Playwright component tests](https://playwright.dev/docs/test-components), React setup and limitations | Reviewed setup and browser/Node boundary. A separate synthetic test bundle exercises rendered controls without adding an application authentication bypass. This does not prove live Firebase or deployed proxy behavior. |
| [Playwright network mocking](https://playwright.dev/docs/mock), API mocking | Reviewed API interception example. Fixed synthetic responses test pending, denied and replay UX; database correctness uses the independent Go/PostgreSQL suite. |
| [gRPC JavaScript advisory](https://github.com/advisories/GHSA-m9gg-hp2v-232j), affected and patched versions | npm audit identified Firebase's unused Firestore dependency retaining vulnerable gRPC. Pin a patched override and validate the resulting graph; no Firestore APIs are used. |

## Alternatives and limits

### Next.js patch review — 2026-10-22, #271/#273

CI39832934816 reported the pinned16.4.0 version in the affected ranges for
maintainer advisories [GHSA-2464-fhr4-w9xw](https://github.com/vercel/next.js/security/advisories/GHSA-2464-fhr4-w9xw),
[GHSA-673q-mjm6-r5xr](https://github.com/vercel/next.js/security/advisories/GHSA-673q-mjm6-r5xr)
and [GHSA-p3m5-59mp-jv5r](https://github.com/vercel/next.js/security/advisories/GHSA-p3m5-59mp-jv5r).
Reviewed selected affected/patched version, impact and workaround sections in
the maintainer/GitHub-reviewed records, not exploit reproduction. All identify
16.4.1 as patched; the image advisory also requires patched sharp dependency
resolution. [16.4.2 release notes](https://github.com/vercel/next.js/releases/tag/v16.4.2)
were reviewed for the subsequent stable patch changes; the npm audit resolver
selected16.4.2. Choose an exact16.4.2 pin and regenerated locked graph over a
scanner exemption or broad forced update. No application feature, financial
message, accounting or ISO applicability changes. No deployed exploitability or
security-conformance claim follows from version/audit results alone.

Acceptance: locked clean install, unchanged high-severity audit gate, web
type/unit/build/browser gates, both actual API/SQL browser journeys and standalone
frontend image proof. Current evidence belongs in the #271 integration runbook.

### Application boundary

A separate browser API origin would add CORS and more deployment configuration;
server-side cookie sessions would require a distinct CSRF/session contract.
The existing bearer boundary and fixed rewrite minimize this slice. We do not
infer standards conformance, deployed identity proof or accounting correctness
from the UI framework. See [UI contract](../contracts/first-ui.md) for acceptance
and recovery limits; revisit request recovery before live use.

## RE2JS transitive patch — PR356 CI follow-up

Reviewed maintainer [GHSA-gg6h-894w-4m9c](https://github.com/le0pard/re2js/security/advisories/GHSA-gg6h-894w-4m9c),
affected/patched versions and summary: versions2.2.0–2.8.6 have quadratic matching
with opt-in lookbehinds;2.8.7 is patched. Selected sections only, no claim that
Jayburd exercises the vulnerable path. CI40793097455 failed npm audit after the
advisory entered the database. Lock only re2js2.8.6→2.8.7 within its existing
transitive range; avoid an unnecessary minor upgrade. Registry integrity metadata
retrieved for exact2.8.7; no new direct dependency or API change. Acceptance: clean
locked install/audit and frontend type/unit/build/browser gates; retain security
audit gate. No suppression or unsafe override.
