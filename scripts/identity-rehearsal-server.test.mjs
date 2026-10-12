import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, writeFile, mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { request } from 'node:http';
import { createRehearsalServer } from './identity-rehearsal-server.mjs';

/** Synthetic HTTP probe permits explicit Host/path assertions without external network access. */
function probe(port, path, headers = {}, method = 'GET') {
  return new Promise((resolve, reject) => {
    const req = request({hostname: '127.0.0.1', port, path, method, headers}, res => {
      const chunks = []; res.on('data', data => chunks.push(data));
      res.on('end', () => resolve({status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks).toString()}));
    });
    req.on('error', reject); req.end();
  });
}
test('loopback static boundary rejects finance, source paths, foreign origins and writes', async () => {
  const root = await mkdtemp(join(tmpdir(), 'jayburd-rehearsal-test-'));
  await mkdir(join(root, 'assets'));
  await writeFile(join(root, 'index.html'), '<h1>Synthetic identity rehearsal</h1>');
  await writeFile(join(root, 'assets', 'main-abc.js'), '/* synthetic */');
  const server = createRehearsalServer(root);
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const {port, address} = server.address();
  try {
    assert.equal(address, '127.0.0.1');
    const page = await probe(port, '/'); assert.equal(page.status, 200);
    assert.equal(page.headers['cache-control'], 'no-store');
    assert.equal(page.headers['referrer-policy'], 'no-referrer');
    assert.match(page.headers['content-security-policy'], /connect-src https:\/\/identitytoolkit.googleapis.com https:\/\/securetoken.googleapis.com;/);
    assert.match(page.headers['content-security-policy'], /frame-ancestors 'none'/);
    assert.equal((await probe(port, '/assets/main-abc.js')).status, 200);
    assert.equal((await probe(port, '/', {}, 'HEAD')).body, '');
    for (const path of ['/v1/entities', '/api/auth-config', '/.env', '/assets/../../package.json', '/assets/main.js.map', '/?oobCode=synthetic', '/assets/missing.js']) {
      assert.equal((await probe(port, path)).status, 404);
    }
    assert.equal((await probe(port, '/', {Host: 'evil.example.invalid'})).status, 403);
    assert.equal((await probe(port, '/', {Origin: 'https://evil.example.invalid'})).status, 403);
    assert.equal((await probe(port, '/', {'Sec-Fetch-Site': 'cross-site'})).status, 403);
    assert.equal((await probe(port, '/', {}, 'POST')).status, 405);
  } finally {
    await new Promise(resolve => server.close(resolve)); await rm(root, {recursive: true, force: true});
  }
});
