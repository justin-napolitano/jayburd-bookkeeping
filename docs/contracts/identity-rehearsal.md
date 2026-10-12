# Local managed identity rehearsal

## Decision and boundary

Owner-authorized #39, 2026-10-12. A separate static browser entry reuses the
production AuthProvider, Firebase adapter, controller and LoginPanel. It contains
no Bookkeeping import, bridge, database connection or token display. The existing
Next.js application and deployment workflow do not gain a rehearsal route/mode.

The owner opens `http://127.0.0.1:39091` and supplies only the public Firebase web
API key. Project and auth domain are fixed to `jayburd-bookkeeping-dev` and its
firebaseapp.com domain. Configuration lives in browser memory. No environment
files or credentials are loaded; a missing/invalid key cannot initialize auth.
The shared provider accepts validated public configuration as an optional prop;
production continues fetching its existing runtime endpoint.

A Node static server binds only IPv4 loopback, validates Host and serves only
root HTML and compiled JS/CSS assets. It exposes no config mutation endpoint,
proxy, CORS, directory listing, source map or API routes. CSP limits browser
connections to the two Google identity endpoints; no remote scripts or frames.
Vite builds the separate entry with environment-file loading/public directory
copying disabled. Default tests use synthetic values and block external requests.
Alternative full local application would unnecessarily expose finance routes;
a custom auth implementation would drift from the production login flow.

## Owner proof and limits

The owner privately requests verification mail, opens Google's hosted link,
returns to check verification, signs in again, enrolls TOTP and completes a fresh
password/TOTP challenge. Record only redacted outcomes. Reload/signout clear memory;
provider errors stay generic. API-key HTTP-referrer restrictions must permit the
exact loopback origin if enabled; do not weaken API restrictions globally. This
email/password flow uses no OAuth redirect or custom continue URL; do not add
localhost to hosted redirect configuration without evidence it is needed.

This proves browser/provider behavior only. Revoked/deleted identity denial by
the Go verifier, SQL principal mapping, entity authorization, hosted TLS/Invoker,
link expiry/reuse and recovery require separate reviewed evidence before #6/#27
acceptance. Do not label the browser's MFA state as server authorization.
No ISO financial-message mapping applies; ledger contracts and SQL are unchanged.

## Acceptance

Named server tests cover loopback binding, Host/method/path denial and CSP.
Browser rehearsal tests block external traffic and prove missing/invalid settings,
production verification UI using synthetic provider responses, no finance request
and no persistent credentials. Existing web and Go suites remain required.
Sources BA04 in the browser source register; evidence belongs in the rehearsal
runbook. Actual email/TOTP outcomes are owner-only, not inferred from tests.
