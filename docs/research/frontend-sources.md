# Frontend source review

## Decision

Use the repository-selected Next.js/TypeScript stack for manual bookkeeping. Browser
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
