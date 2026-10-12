import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { resolve } from 'node:path';

/** Serve compiled assets only; no body parsing, provider requests, proxy or access logging. */
export function createRehearsalServer(root) {
  const server = createServer(async (req, res) => {
    const host = `127.0.0.1:${server.address().port}`;
    res.setHeader('Cache-Control', 'no-store');
    res.setHeader('X-Content-Type-Options', 'nosniff');
    res.setHeader('Referrer-Policy', 'no-referrer');
    res.setHeader('X-Frame-Options', 'DENY');
    res.setHeader('Content-Security-Policy', "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src https://identitytoolkit.googleapis.com https://securetoken.googleapis.com; frame-ancestors 'none'; base-uri 'none'; form-action 'none'");
    if (req.headers.host !== host || (req.headers.origin && req.headers.origin !== `http://${host}`) || req.headers['sec-fetch-site'] === 'cross-site') {
      res.writeHead(403); res.end(); return;
    }
    if (req.method !== 'GET' && req.method !== 'HEAD') { res.writeHead(405); res.end(); return; }
    const path = req.url === '/' ? 'index.html' : req.url?.match(/^\/assets\/([A-Za-z0-9_-]+\.(?:js|css))$/)?.[0].slice(1);
    if (!path) { res.writeHead(404); res.end(); return; }
    try {
      const bytes = await readFile(resolve(root, path));
      res.setHeader('Content-Type', path.endsWith('.js') ? 'text/javascript; charset=utf-8' : path.endsWith('.css') ? 'text/css; charset=utf-8' : 'text/html; charset=utf-8');
      res.writeHead(200); res.end(req.method === 'HEAD' ? undefined : bytes);
    } catch { res.writeHead(404); res.end(); }
  });
  return server;
}

// The only runnable mode has a fixed loopback bind; callers cannot pass a public interface.
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = fileURLToPath(new URL('../.local/identity-rehearsal/', import.meta.url));
  const server = createRehearsalServer(root);
  server.on('error', () => { console.error('BLOCKED: local rehearsal could not start.'); process.exitCode = 1; });
  server.listen(39091, '127.0.0.1', () => console.log('Identity rehearsal: http://127.0.0.1:39091 — enter public configuration and credentials privately in your browser.'));
}
