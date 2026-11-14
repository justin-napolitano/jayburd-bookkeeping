import { writeFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

const web = 'https://jayburd-dev-web-9088794002.us-central1.run.app';
const api = 'https://jayburd-dev-api-9088794002.us-central1.run.app';
const fixture = '11111111-1111-4111-8111-111111111111';
const targets = Object.freeze([
  ['web-root', web + '/', 200],
  ['web-auth-config', web + '/api/auth-config', 200],
  ['web-workspace', web + '/v1/workspace', 401],
  ['web-private-collection', web + '/v1/collection', 401],
  ['web-private-recovery', web + '/v1/collection-recovery', 401],
  ['web-private-onboarding', web + '/v1/collection-onboarding', 401],
  ['web-private-onboarding-pending', web + '/v1/collection-onboarding/commands/pending', 401],
  ['web-private-taxonomy', web + '/v3/collections/' + fixture + '/taxonomy', 401],
  ['direct-api-readiness', api + '/readyz', 403],
].map(Object.freeze));
const blocked = () => new Error('Development transport check blocked');

// The same deadline bounds transport and body disposal, even if either ignores abort.
async function checkTarget(fetchImpl, url, expected) {
  const controller = new AbortController();
  let timer;
  try {
    return await Promise.race([
      (async () => {
        const response = await fetchImpl(url, {
          method: 'GET', credentials: 'omit', redirect: 'manual',
          cache: 'no-store', headers: {}, signal: controller.signal,
        });
        const observed = response.status;
        if (response.body !== null && response.body !== undefined) await response.body.cancel();
        if (!Number.isInteger(observed) || observed !== expected) throw blocked();
        return observed;
      })(),
      new Promise((_, reject) => {
        timer = setTimeout(() => {
          controller.abort();
          reject(blocked());
        }, 15000);
      }),
    ]);
  } catch {
    // No network message, response header/body, URL or credential enters diagnostics.
    throw blocked();
  } finally {
    clearTimeout(timer);
    controller.abort();
  }
}

// Explicit fixed public transport verification; no identity, runtime mutation or product acceptance.
export async function checkDevTransport(fetchImpl = globalThis.fetch) {
  const checks = [];
  for (const [target, url, expected] of targets) {
    const observed = await checkTarget(fetchImpl, url, expected);
    checks.push({ target, expected, observed });
  }
  return {
    checkedAt: new Date().toISOString(),
    checks,
    limitations: [
      'Anonymous fixed development URLs only; response bodies and headers are discarded.',
      'No authenticated workspace, YAML import, categorization, accounting or source-history acceptance.',
      'No image, runtime configuration, deployment, database or provider qualification.',
    ],
  };
}

// Running tests/importing this module never performs requests; only this explicit CLI does.
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    if (process.argv.length !== 3 || !process.argv[2]) throw blocked();
    const receipt = await checkDevTransport();
    writeFileSync(process.argv[2], JSON.stringify(receipt, null, 2) + '\n', { flag: 'wx' });
    process.stdout.write('PASS: fixed development transport statuses\n');
  } catch {
    process.stderr.write('BLOCKED: fixed development transport checks\n');
    process.exitCode = 1;
  }
}
