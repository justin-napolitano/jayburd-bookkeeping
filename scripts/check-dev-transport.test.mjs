import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { checkDevTransport } from './check-dev-transport.mjs';

const urls = [
  'https://jayburd-dev-web-9088794002.us-central1.run.app/',
  'https://jayburd-dev-web-9088794002.us-central1.run.app/api/auth-config',
  'https://jayburd-dev-web-9088794002.us-central1.run.app/v1/workspace',
  'https://jayburd-dev-web-9088794002.us-central1.run.app/v1/collection',
  'https://jayburd-dev-web-9088794002.us-central1.run.app/v1/collection-recovery',
  'https://jayburd-dev-web-9088794002.us-central1.run.app/v1/collection-onboarding',
  'https://jayburd-dev-web-9088794002.us-central1.run.app/v1/collection-onboarding/commands/pending',
  'https://jayburd-dev-web-9088794002.us-central1.run.app/v3/collections/11111111-1111-4111-8111-111111111111/taxonomy',
  'https://jayburd-dev-api-9088794002.us-central1.run.app/readyz',
];
const statuses = [200, 200, 401, 401, 401, 401, 401, 401, 403];
function transport(overrides = {}) {
  const calls = [];
  let cancellations = 0;
  return {
    calls,
    get cancellations() { return cancellations; },
    async fetch(url, options) {
      const index = calls.length;
      calls.push({ url, options });
      return {
        status: overrides[index] ?? statuses[index],
        body: { async cancel() { cancellations++; } },
        get headers() { throw new Error('PRIVATE_SENTINEL'); },
        async text() { throw new Error('PRIVATE_SENTINEL'); },
        async json() { throw new Error('PRIVATE_SENTINEL'); },
      };
    },
  };
}
const isSafeFailure = error => error.message === 'Development transport check blocked' && error.cause === undefined && !String(error.stack).includes('PRIVATE_SENTINEL');

test('nine fixed endpoints use anonymous GET with closed status-only receipt', async () => {
  const t = transport();
  const receipt = await checkDevTransport(t.fetch);
  assert.deepEqual(t.calls.map(call => call.url), urls);
  assert.equal(t.cancellations, 9);
  assert.deepEqual(Object.keys(receipt).sort(), ['checkedAt', 'checks', 'limitations']);
  assert.equal(new Date(receipt.checkedAt).toISOString(), receipt.checkedAt);
  assert.equal(receipt.checks.length, 9);
  assert.deepEqual(receipt.checks.map(check => check.expected), statuses);
  assert.deepEqual(receipt.checks.map(check => check.observed), statuses);
  for (const check of receipt.checks) {
    assert.deepEqual(Object.keys(check).sort(), ['expected', 'observed', 'target']);
    assert.match(check.target, /^[a-z-]+$/);
  }
  for (const { url, options } of t.calls) {
    assert.equal(options.method, 'GET');
    assert.equal(options.credentials, 'omit');
    assert.equal(options.redirect, 'manual');
    assert.equal(options.cache, 'no-store');
    assert.deepEqual(options.headers, {});
    assert.equal(Object.hasOwn(options, 'body'), false);
    assert.ok(options.signal instanceof AbortSignal);
    assert.equal(options.signal.aborted, true);
    assert.equal(new URL(url).username, '');
    assert.equal(new URL(url).password, '');
  }
  assert.ok(receipt.limitations.every(item => typeof item === 'string'));
  assert.doesNotMatch(JSON.stringify(receipt), /PRIVATE_SENTINEL|https?:|cookie|bearer/i);
});

for (const [index, status] of [[0, 302], [1, 307], [2, 200], [3, 403], [4, 500], [5, 200], [6, 200], [7, 404], [8, 200]]) {
  test(`unexpected status ${status} at target ${index} is disposed and stops without retry`, async () => {
    const t = transport({ [index]: status });
    await assert.rejects(checkDevTransport(t.fetch), isSafeFailure);
    assert.equal(t.calls.length, index + 1);
    assert.equal(t.cancellations, index + 1);
  });
}

test('transport and body-disposal failures reveal no remote diagnostics', async () => {
  for (const fetch of [
    async () => { throw new Error('PRIVATE_SENTINEL'); },
    async () => ({ status: 200, body: { async cancel() { throw new Error('PRIVATE_SENTINEL'); } } }),
  ]) await assert.rejects(checkDevTransport(fetch), isSafeFailure);
});

test('bodyless responses still require exact numeric HTTP status', async () => {
  let index = 0;
  const receipt = await checkDevTransport(async () => ({ status: statuses[index++], body: null }));
  assert.equal(receipt.checks.length, 9);
  await assert.rejects(checkDevTransport(async () => ({ status: '200', body: null })), isSafeFailure);
});

test('a noncooperating fetch is bounded by the same fifteen-second deadline', async t => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  let calls = 0, signal;
  const promise = checkDevTransport(async (_url, options) => { calls++; signal = options.signal; return new Promise(() => {}); });
  const rejected = assert.rejects(promise, isSafeFailure);
  t.mock.timers.tick(15000);
  await rejected;
  assert.equal(calls, 1);
  assert.equal(signal.aborted, true);
});

test('a noncooperating body cancellation is bounded by the request deadline', async t => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  let calls = 0, cancelled = false, signal;
  const promise = checkDevTransport(async (_url, options) => {
    calls++; signal = options.signal;
    return { status: 200, body: { cancel() { cancelled = true; return new Promise(() => {}); } } };
  });
  const rejected = assert.rejects(promise, isSafeFailure);
  for (let i = 0; i < 5 && !cancelled; i++) await Promise.resolve();
  assert.equal(cancelled, true);
  t.mock.timers.tick(15000);
  await rejected;
  assert.equal(calls, 1);
  assert.equal(signal.aborted, true);
});

test('body cancellation cannot receive a fresh deadline after a slow fetch', async t => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  let resolveFetch, cancelled = false;
  const promise = checkDevTransport(() => new Promise(resolve => { resolveFetch = resolve; }));
  const rejected = assert.rejects(promise, isSafeFailure);
  t.mock.timers.tick(14000);
  resolveFetch({ status: 200, body: { cancel() { cancelled = true; return new Promise(() => {}); } } });
  for (let i = 0; i < 5 && !cancelled; i++) await Promise.resolve();
  assert.equal(cancelled, true);
  t.mock.timers.tick(1000);
  await rejected;
});

test('CLI rejects missing and extra arguments before transport', () => {
  for (const args of [[], ['unused.json', 'https://unapproved.invalid']]) {
    const result = spawnSync(process.execPath, ['scripts/check-dev-transport.mjs', ...args], { encoding: 'utf8' });
    assert.equal(result.status, 1);
    assert.equal(result.stdout, '');
    assert.equal(result.stderr, 'BLOCKED: fixed development transport checks\n');
  }
});

test('workflow is manual main-only owner gated and has no authentication or deploy capability', () => {
  const workflow = readFileSync('.github/workflows/check-dev-transport.yml', 'utf8');
  assert.match(workflow, /on:\n  workflow_dispatch:\npermissions:/);
  assert.match(workflow, /permissions:\n  contents: read\njobs:/);
  for (const gate of ["github.event_name == 'workflow_dispatch'", "github.ref == 'refs/heads/main'", "github.actor_id == '55423093'", "github.repository_id == '1414515318'", "github.triggering_actor == 'justin-napolitano'"]) assert.ok(workflow.includes(gate));
  assert.match(workflow, /persist-credentials: false/);
  assert.match(workflow, /node-version: '24\.18\.0'/);
  assert.match(workflow, /retention-days: 90/);
  assert.match(workflow, /path: \$\{\{ runner\.temp \}\}\/dev-transport\.json/);
  assert.doesNotMatch(workflow, /secrets\.|id-token:|GH_TOKEN|GITHUB_TOKEN|Authorization|Cookie|google-github-actions|gcloud|terraform|workflow_call|pull_request|push:|inputs:/i);
  const script = readFileSync('scripts/check-dev-transport.mjs', 'utf8');
  assert.doesNotMatch(script, /process\.env|response\.(?:text|json|headers)|execSync|spawnSync/);
});
