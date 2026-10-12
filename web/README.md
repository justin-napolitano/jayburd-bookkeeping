# Bookkeeping web application

## Local checks

Use Node 24.18+ within Node 24 and the committed lockfile:

```sh
npm ci
npm run api:check
npm run check
npm test
npm run build
npx playwright install chromium
npm run test:browser
```

Browser tests use a separate synthetic component bundle. They never connect to
Firebase, the accounting database or a cloud deployment. A successful browser
journey proves UI behavior against its mock contract, not real posting/identity.

## Run

`npm run dev` binds loopback. Without public identity settings the page explicitly
shows sign-in unavailable. See [browser authentication](../docs/contracts/browser-auth.md)
for operator-owned setup. The [private API bridge](../docs/contracts/private-api-bridge.md)
requires explicit `BOOKKEEPING_API_MODE`, `BOOKKEEPING_API_ORIGIN` and
`BOOKKEEPING_PUBLIC_ORIGIN`. Local mode uses `loopback` with distinct HTTP loopback
origins; Cloud Run uses `cloud-run` and separate service/user bearer headers.
Public Firebase identifiers are read at runtime through `/api/auth-config`;
they are never Admin credentials or database secrets. Missing settings fail closed.

The production build does not require credentials. No application auth bypass,
provider emulator, server-side cookie authority or fictional financial data is
included. `npm run api:generate` regenerates types from the repository OpenAPI;
`npm run api:check` fails on drift. SQL owns accounting validations.

## Review limits

See [first UI contract](../docs/contracts/first-ui.md). Unknown mutation outcomes
retain request key/body while the session stays open. Signing out, expiry or tab
closure clears memory; verify the request before re-entry to avoid duplication.
The gRPC override patches Firebase's unused Firestore transitive dependency;
revisit when Firebase publishes a compatible patched dependency graph.
