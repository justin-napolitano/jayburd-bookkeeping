/** Runtime routing is fixed by the operator, never by request headers, URL parameters or ADC. */
type BridgeConfig = { api: string; origin: string; cloud: boolean };
type Transport = typeof fetch;
const limit = 1 << 20;
const tokenPattern = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/;
const uuid = '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}';
const commandAction = new RegExp(`^/v1/commands/${uuid}/(?:execute|cancel|acknowledge)$`, 'i');
const bookPath = `/v1/entities/${uuid}/books/${uuid}`;

/** Read only named configuration and reject ambiguous transport or ambient credential overrides. */
function configuration(env: Record<string, string | undefined>): BridgeConfig {
  if (['GOOGLE_APPLICATION_CREDENTIALS', 'GCE_METADATA_HOST', 'GCE_METADATA_IP', 'FIREBASE_AUTH_EMULATOR_HOST'].some(name => env[name] !== undefined)) throw new Error('configuration');
  const cloud = env.BOOKKEEPING_API_MODE === 'cloud-run';
  if (!cloud && (env.BOOKKEEPING_API_MODE !== 'loopback' || env.K_SERVICE !== undefined)) throw new Error('configuration');
  const api = new URL(env.BOOKKEEPING_API_ORIGIN ?? '');
  const origin = new URL(env.BOOKKEEPING_PUBLIC_ORIGIN ?? '');
  for (const url of [api, origin]) {
    if (url.username || url.password || url.pathname !== '/' || url.search || url.hash) throw new Error('configuration');
    if (cloud ? url.protocol !== 'https:' || !!url.port : url.protocol !== 'http:' || !['127.0.0.1', 'localhost'].includes(url.hostname)) throw new Error('configuration');
  }
  if (cloud && !/^[a-z0-9-]+(?:\.[a-z0-9-]+)*\.run\.app$/.test(api.hostname)) throw new Error('configuration');
  if (api.origin === origin.origin) throw new Error('configuration');
  return {api: api.origin, origin: origin.origin, cloud};
}

/** Permit only currently implemented bookkeeping routes; deferred close/admin paths stay unavailable. */
function allowed(url: URL, method: string): boolean {
  if (url.pathname.length + url.search.length > 4096) return false;
  if (method === 'GET' && new RegExp(`^${bookPath}/trial-balance$`, 'i').test(url.pathname)) {
    return [...url.searchParams.keys()].length === 1 && /^\d{4}-\d{2}-\d{2}$/.test(url.searchParams.get('through_date') ?? '');
  }
  if (url.search) return false;
  return method === 'GET' && (url.pathname === '/v1/workspace' || url.pathname === '/v1/commands/pending' || new RegExp(`^${bookPath}/journals/${uuid}$`, 'i').test(url.pathname)) ||
    method === 'POST' && (url.pathname === '/v1/setup' || url.pathname === '/v1/commands' || commandAction.test(url.pathname) || new RegExp(`^${bookPath}/journals(?:/${uuid}/reversals)?$`, 'i').test(url.pathname));
}

/** Bound streamed bodies and enforce cancellation even if an upstream stalls between chunks. */
async function read(stream: ReadableStream<Uint8Array> | null, maximum: number, signal: AbortSignal): Promise<Uint8Array> {
  if (!stream) return new Uint8Array();
  const reader = stream.getReader();
  const buffer = new Uint8Array(maximum);
  let length = 0;
  const abort = () => { void reader.cancel().catch(() => {}); };
  signal.addEventListener('abort', abort, {once: true});
  try {
    signal.throwIfAborted();
    while (true) {
      const {done, value} = await reader.read();
      signal.throwIfAborted();
      if (done) break;
      length += value.byteLength;
      if (length > maximum) throw new RangeError('body limit');
      buffer.set(value, length - value.byteLength);
    }
    return buffer.slice(0, length);
  } finally {
    signal.removeEventListener('abort', abort);
    void reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}

/** Fixed errors never relay provider messages, tokens, cookies or financial descriptions. */
function failure(status: number): Response {
  return Response.json({type: 'about:blank', title: 'Request unavailable', status}, {
    status, headers: {'Cache-Control': 'no-store', 'Content-Type': 'application/problem+json'},
  });
}

/** Only the fixed metadata endpoint can supply the service token; no redirects, files or fallback credentials. */
async function serviceToken(api: string, transport: Transport, signal: AbortSignal): Promise<string> {
  const endpoint = new URL('http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity');
  endpoint.searchParams.set('audience', api);
  const deadline = AbortSignal.any([signal, AbortSignal.timeout(3000)]);
  const response = await transport(endpoint, {headers: {'Metadata-Flavor': 'Google'}, cache: 'no-store', redirect: 'error', signal: deadline});
  if (!response.ok || response.headers.get('Metadata-Flavor') !== 'Google') {
    void response.body?.cancel(); throw new Error('metadata');
  }
  const token = new TextDecoder().decode(await read(response.body, 16384, deadline));
  if (!tokenPattern.test(token)) throw new Error('metadata');
  return token;
}

/** A process-local admission bound complements Cloud Run limits; tests inject transport without a runtime bypass. */
export function createBridge(transport: Transport = fetch, environment: () => Record<string, string | undefined> = () => process.env) {
  let active = 0;
  return async function forward(request: Request): Promise<Response> {
    if (active >= 32) return failure(503);
    active++;
    try {
      let config: BridgeConfig;
      try { config = configuration(environment()); } catch { return failure(503); }
      const url = new URL(request.url);
      if (!allowed(url, request.method)) return failure(404);
      const origin = request.headers.get('Origin');
      const site = request.headers.get('Sec-Fetch-Site');
      if (origin !== null && origin !== config.origin || site !== null && site !== 'same-origin') return failure(403);
      const bearer = request.headers.get('Authorization') ?? '';
      if (bearer.length > 16384 || !bearer.startsWith('Bearer ') || !tokenPattern.test(bearer.slice(7))) return failure(401);
      const signal = AbortSignal.any([request.signal, AbortSignal.timeout(12000)]);
      const headers = new Headers({Authorization: bearer, Origin: config.origin, Accept: 'application/json'});
      let body: Uint8Array | undefined;
      if (request.method === 'POST') {
        if (!/^application\/json(?:\s*;\s*charset=utf-8)?$/i.test(request.headers.get('Content-Type') ?? '')) return failure(415);
        // Recovery transitions are replay-bound by durable command ID; registration still requires a key.
        const key = request.headers.get('Idempotency-Key');
        if ((key !== null || !commandAction.test(url.pathname)) && !/^[A-Za-z0-9._:-]{1,128}$/.test(key ?? '')) return failure(400);
        headers.set('Content-Type', 'application/json');
        if (key !== null) headers.set('Idempotency-Key', key);
        try { body = await read(request.body, limit, signal); } catch (error) { return failure(error instanceof RangeError ? 413 : 504); }
      } else if (request.body !== null) return failure(400);
      if (config.cloud) headers.set('X-Serverless-Authorization', `Bearer ${await serviceToken(config.api, transport, signal)}`);
      const upstream = await transport(config.api + url.pathname + url.search, {
        method: request.method, headers, body: body as BodyInit | undefined, redirect: 'manual', cache: 'no-store', signal,
      });
      if (!upstream.ok) {
        void upstream.body?.cancel();
        return failure([400,401,403,404,409,413,415,422,429,503,504].includes(upstream.status) ? upstream.status : 502);
      }
      if (!/^application\/json(?:\s*;.*)?$/i.test(upstream.headers.get('Content-Type') ?? '') || ![200,201].includes(upstream.status)) {
        void upstream.body?.cancel(); return failure(502);
      }
      const bytes = await read(upstream.body, limit, signal);
      // Validate framing, retaining original bytes so exact decimal strings and receipts never change.
      JSON.parse(new TextDecoder('utf-8', {fatal:true}).decode(bytes));
      return new Response(bytes as BodyInit, {status: upstream.status, headers: {'Content-Type':'application/json', 'Cache-Control':'no-store', 'X-Content-Type-Options':'nosniff'}});
    } catch { return failure(502); }
    finally { active--; }
  };
}
