// @vitest-environment node
import {afterEach, describe, expect, it, vi} from 'vitest';
import {createBridge} from './bridge';
const env = {BOOKKEEPING_API_MODE:'cloud-run', BOOKKEEPING_API_ORIGIN:'https://api-synthetic.run.app', BOOKKEEPING_PUBLIC_ORIGIN:'https://books.example.invalid'};
const bearer = 'Bearer synthetic.user.signature';
/** Synthetic requests exercise the transport boundary without executing real provider calls. */
function request(path='/v1/workspace', headers: Record<string,string>={}, body?:string) {
  return new Request('https://books.example.invalid'+path, {method:body === undefined ? 'GET':'POST',
    headers:{Authorization:bearer, Origin:env.BOOKKEEPING_PUBLIC_ORIGIN, ...headers}, body});
}
/** The fake metadata response is intentionally not a valid signed identity credential. */
function metadata() { return new Response('synthetic.service.signature', {headers:{'Metadata-Flavor':'Google'}}); }
/** Model one metadata fetch followed by an owned JSON API response. */
function transport(response=Response.json({amount:'12.30'})) { return vi.fn<typeof fetch>().mockResolvedValueOnce(metadata()).mockResolvedValueOnce(response); }
afterEach(() => vi.useRealTimers());
describe('private API bridge', () => {
 it('keeps user/service identities separate and forwards only exact replay bytes and allowed headers', async () => {
  const fetcher=transport(); const handler=createBridge(fetcher,()=>env);
  const body='{"amount":"12.30","request":"unchanged"}';
  const result=await handler(request('/v1/setup', {'Content-Type':'application/json','Idempotency-Key':'synthetic-key','Cookie':'private-cookie','X-Serverless-Authorization':'Bearer injected','X-Actor-ID':'injected','Forwarded':'host=evil.invalid'},body));
  expect(result.status).toBe(200); expect(await result.text()).toBe('{"amount":"12.30"}');
  const metadataURL=new URL(String(fetcher.mock.calls[0][0]));
  expect(metadataURL.hostname).toBe('metadata.google.internal');
  expect(metadataURL.searchParams.get('audience')).toBe(env.BOOKKEEPING_API_ORIGIN);
  const [url,init]=fetcher.mock.calls[1]; const headers=new Headers(init?.headers);
  expect(url).toBe(env.BOOKKEEPING_API_ORIGIN+'/v1/setup');
  expect(headers.get('Authorization')).toBe(bearer);
  expect(headers.get('X-Serverless-Authorization')).toBe('Bearer synthetic.service.signature');
  expect(headers.get('Idempotency-Key')).toBe('synthetic-key');
  for (const name of ['cookie','x-actor-id','forwarded']) expect(headers.has(name)).toBe(false);
  expect(new TextDecoder().decode(init?.body as Uint8Array)).toBe(body);
  expect(init).toMatchObject({cache:'no-store',redirect:'manual'});
  expect(result.headers.get('Cache-Control')).toBe('no-store');
 });
 it.each([
  ['/v1/workspace',{Authorization:''},401],
  ['/v1/workspace',{Authorization:bearer+', Bearer another'},401],
  ['/v1/workspace',{Origin:'https://evil.invalid'},403],
  ['/v1/workspace',{'Sec-Fetch-Site':'cross-site'},403],
  ['/v1/workspace?access_token=synthetic',{},404],
  ['/v1/admin',{},404],
  ['/v1/entities/00000000-0000-4000-8000-000000000001/books/00000000-0000-4000-8000-000000000002/periods/00000000-0000-4000-8000-000000000003/close',{},404],
 ] as const)('rejects %s before any metadata or backend access',async(path,headers,status)=>{
  const fetcher=vi.fn<typeof fetch>();
  expect((await createBridge(fetcher,()=>env)(request(path,headers))).status).toBe(status);
  expect(fetcher).not.toHaveBeenCalled();
 });
 it.each([
  {}, {...env,BOOKKEEPING_API_ORIGIN:'https://evil.invalid'}, {...env,BOOKKEEPING_API_ORIGIN:env.BOOKKEEPING_PUBLIC_ORIGIN},
  {...env,BOOKKEEPING_API_ORIGIN:'https://api-synthetic.run.app/path'}, {...env,GOOGLE_APPLICATION_CREDENTIALS:''},
  {...env,GCE_METADATA_HOST:'override'}, {...env,BOOKKEEPING_API_MODE:'loopback',K_SERVICE:'cloud-service'},
 ])('fails closed for missing or ambiguous configuration',async(config)=>{
  const fetcher=vi.fn<typeof fetch>();
  expect((await createBridge(fetcher,()=>config)(request())).status).toBe(503);
  expect(fetcher).not.toHaveBeenCalled();
 });
 it('permits explicit local transport without fetching metadata',async()=>{
  const fetcher=vi.fn<typeof fetch>().mockResolvedValue(Response.json({entities:[]}));
  const config={BOOKKEEPING_API_MODE:'loopback',BOOKKEEPING_API_ORIGIN:'http://127.0.0.1:8080',BOOKKEEPING_PUBLIC_ORIGIN:'http://localhost:3000'};
  expect((await createBridge(fetcher,()=>config)(request('/v1/workspace',{Origin:config.BOOKKEEPING_PUBLIC_ORIGIN}))).status).toBe(200);
  expect(fetcher).toHaveBeenCalledTimes(1);
  expect(new Headers(fetcher.mock.calls[0][1]?.headers).has('X-Serverless-Authorization')).toBe(false);
 });
 it.each([401,403,404,409,429])('preserves %s denial semantics without provider body or cookies',async(status)=>{
  const fetcher=transport(new Response('private provider detail',{status,headers:{'Set-Cookie':'private','Authorization':'private'}}));
  const result=await createBridge(fetcher,()=>env)(request());
  expect(result.status).toBe(status); expect(await result.text()).not.toContain('private');
  expect(result.headers.has('Set-Cookie')).toBe(false); expect(result.headers.has('Authorization')).toBe(false);
 });
 it.each([new Response('private',{status:302,headers:{Location:'https://evil.invalid'}}),new Response('not JSON',{headers:{'Content-Type':'text/html'}}),new Response('{broken',{headers:{'Content-Type':'application/json'}})])('rejects redirects or invalid upstream framing',async(response)=>{
  const fetcher=transport(response);
  expect((await createBridge(fetcher,()=>env)(request())).status).toBe(502);
  expect(fetcher).toHaveBeenCalledTimes(2);
 });
 it('bounds bodies in both directions and never forwards an oversized write',async()=>{
  const fetcher=vi.fn<typeof fetch>();
  const oversized=request('/v1/setup',{'Content-Type':'application/json','Idempotency-Key':'synthetic'},'x'.repeat((1<<20)+1));
  expect((await createBridge(fetcher,()=>env)(oversized)).status).toBe(413); expect(fetcher).not.toHaveBeenCalled();
  const response=new Response('x'.repeat((1<<20)+1),{headers:{'Content-Type':'application/json'}});
  expect((await createBridge(transport(response),()=>env)(request())).status).toBe(502);
 });
 it.each([new Response('private',{status:500}),new Response('synthetic.service.signature'),new Response('malformed',{headers:{'Metadata-Flavor':'Google'}})])('does not contact the API after a metadata failure',async(response)=>{
  const fetcher=vi.fn<typeof fetch>().mockResolvedValue(response);
  const result=await createBridge(fetcher,()=>env)(request());
  expect(result.status).toBe(502); expect(await result.text()).not.toContain('private'); expect(fetcher).toHaveBeenCalledTimes(1);
 });
 it('does not retry or fabricate confirmation after a lost posting response',async()=>{
  const fetcher=vi.fn<typeof fetch>().mockResolvedValueOnce(metadata()).mockRejectedValueOnce(new Error('private token detail'));
  const result=await createBridge(fetcher,()=>env)(request('/v1/setup',{'Content-Type':'application/json','Idempotency-Key':'same-key'},'{}'));
  expect(result.status).toBe(502); expect(await result.text()).not.toContain('private'); expect(fetcher).toHaveBeenCalledTimes(2);
 });
 it('bounds concurrent work and releases admission after failures',async()=>{
  const pending:Array<(response:Response)=>void>=[];
  const fetcher=vi.fn<typeof fetch>().mockImplementation(()=>new Promise(resolve=>pending.push(resolve)));
  const handler=createBridge(fetcher,()=>env);
  const calls=Array.from({length:32},()=>handler(request()));
  expect((await handler(request())).status).toBe(503); expect(fetcher).toHaveBeenCalledTimes(32);
  for(const resolve of pending) resolve(new Response('',{status:503}));
  await Promise.all(calls);
  fetcher.mockResolvedValue(new Response('',{status:503}));
  expect((await handler(request())).status).toBe(502);
 });
 it('cancels a stalled request body when the caller disconnects',async()=>{
  let cancelled=false;
  const abort=new AbortController();
  const body=new ReadableStream<Uint8Array>({cancel(){cancelled=true;}});
  const req=new Request('https://books.example.invalid/v1/setup',{method:'POST',headers:{Authorization:bearer,'Content-Type':'application/json','Idempotency-Key':'synthetic'},body,signal:abort.signal,duplex:'half'} as RequestInit);
  const fetcher=vi.fn<typeof fetch>();
  const call=createBridge(fetcher,()=>env)(req);
  abort.abort();
  expect((await call).status).toBe(504); expect(cancelled).toBe(true); expect(fetcher).not.toHaveBeenCalled();
 });
});

describe('durable command transport', () => {
 const id='00000000-0000-4000-8000-000000000001';
 it.each(['execute','cancel','acknowledge'])('forwards %s by durable identity without inventing a new key',async(action)=>{
  const fetcher=transport();const body='{"expected_revision":1}';
  const result=await createBridge(fetcher,()=>env)(request(`/v1/commands/${id}/${action}`,{'Content-Type':'application/json'},body));
  expect(result.status).toBe(200);
  expect(new Headers(fetcher.mock.calls[1][1]?.headers).has('Idempotency-Key')).toBe(false);
  expect(new TextDecoder().decode(fetcher.mock.calls[1][1]?.body as Uint8Array)).toBe(body);
 });
 it('requires a key for registration and allows pending discovery',async()=>{
  const missing=vi.fn<typeof fetch>();
  expect((await createBridge(missing,()=>env)(request('/v1/commands',{'Content-Type':'application/json'},'{}'))).status).toBe(400);
  expect(missing).not.toHaveBeenCalled();
  expect((await createBridge(transport(),()=>env)(request('/v1/commands',{'Content-Type':'application/json','Idempotency-Key':'same-command'},'{}'))).status).toBe(200);
  expect((await createBridge(transport(),()=>env)(request('/v1/commands/pending'))).status).toBe(200);
 });
 it('forwards explicit inaccessible-command clearing without a key',async()=>{
  const fetcher=transport(Response.json({cleared:true}));
  const result=await createBridge(fetcher,()=>env)(request('/v1/commands/pending/clear',{'Content-Type':'application/json'},'{}'));
  expect(result.status).toBe(200);
  expect(new Headers(fetcher.mock.calls[1][1]?.headers).has('Idempotency-Key')).toBe(false);
  expect(new TextDecoder().decode(fetcher.mock.calls[1][1]?.body as Uint8Array)).toBe('{}');
 });
 it.each(['/v1/commands/pending?actor=other','/v1/commands/pending/clear?actor=other',`/v1/commands/${id}/force`,`/v1/commands/invalid/execute`])('rejects non-contract route %s before provider access',async(path)=>{
  const fetcher=vi.fn<typeof fetch>();
  expect((await createBridge(fetcher,()=>env)(request(path,{'Content-Type':'application/json'},'{}'))).status).toBe(404);
  expect(fetcher).not.toHaveBeenCalled();
 });
});

const reportsBase = '/v1/entities/00000000-0000-4000-8000-000000000001/books/00000000-0000-4000-8000-000000000002/reports/';
describe('financial reporting bridge', () => {
 it.each(['trial-balance','balance-sheet','profit-and-loss','general-ledger'])('forwards bounded %s reads without financial conversion', async kind => {
  const fetcher=transport(Response.json({amount:'9007199254740993.01'}));
  const result=await createBridge(fetcher,()=>env)(request(reportsBase+kind+'?through_date=2026-01-31&limit=100&snapshot_revision=0&cursor=abc_123'));
  expect(result.status).toBe(200);
  expect(await result.json()).toEqual({amount:'9007199254740993.01'});
  expect(result.headers.get('Cache-Control')).toBe('no-store');
  expect(new Headers(fetcher.mock.calls[1][1]?.headers).has('Cookie')).toBe(false);
 });
 it.each(['?limit=501','?limit=01','?actor=other','?cursor=abc=','?snapshot_revision=-1','?through_date=2026-01-31&through_date=2026-01-31','?account_id=other','?through_date=%QQ'])('rejects unreviewed query %s before upstream', async query => {
  const fetcher=vi.fn<typeof fetch>();
  const result=await createBridge(fetcher,()=>env)(request(reportsBase+'trial-balance'+query));
  expect(result.status).toBe(404); expect(fetcher).not.toHaveBeenCalled();
 });
 it('allows at most 2MiB for reports while preserving 1MiB for other reads', async () => {
  const data={synthetic:'x'.repeat((1<<20)+20)};
  expect((await createBridge(transport(Response.json(data)),()=>env)(request(reportsBase+'trial-balance?through_date=2026-01-31'))).status).toBe(200);
  expect((await createBridge(transport(Response.json(data)),()=>env)(request())).status).toBe(502);
  expect((await createBridge(transport(Response.json({synthetic:'x'.repeat(2<<20)})),()=>env)(request(reportsBase+'trial-balance?through_date=2026-01-31'))).status).toBe(502);
 });
 it('preserves stale snapshot status while hiding upstream problem details', async () => {
  const response=await createBridge(transport(Response.json({code:'report_snapshot_changed',detail:'untrusted sensitive details'},{status:409})),()=>env)(request(reportsBase+'trial-balance?through_date=2026-01-31'));
  expect(response.status).toBe(409); expect(await response.text()).not.toContain('untrusted');
 });
});

describe('report follow-up bridge',()=>{
 const account='00000000-0000-4000-8000-000000000003';
 const journal='00000000-0000-4000-8000-000000000004';
 const csv=reportsBase+'trial-balance/export.csv?through_date=2026-01-31&snapshot_revision=1';
 it('returns exact CSV with fixed no-store download framing',async()=>{
   const content='"value"\r\n"0.01"\r\n';
   const upstream=new Response(content,{headers:{'Content-Type':'text/csv; charset=utf-8; header=present','Content-Disposition':'inline; filename="unsafe.html"','Set-Cookie':'private'}});
   const fetcher=transport(upstream);const result=await createBridge(fetcher,()=>env)(request(csv));
   expect(result.status).toBe(200);expect(await result.text()).toBe(content);
   expect(result.headers.get('Content-Disposition')).toBe('attachment; filename="financial-reporting-review.csv"');
   expect(result.headers.get('Cache-Control')).toBe('no-store');expect(result.headers.has('Set-Cookie')).toBe(false);
   expect(new Headers(fetcher.mock.calls[1][1]?.headers).get('Accept')).toBe('text/csv');
 });
 it.each(['?through_date=2026-01-31','?snapshot_revision=1&cursor=abc','?snapshot_revision=1&limit=100'])('rejects unpinned or paged CSV %s',async query=>{
   const fetcher=vi.fn<typeof fetch>();expect((await createBridge(fetcher,()=>env)(request(reportsBase+'trial-balance/export.csv'+query))).status).toBe(404);expect(fetcher).not.toHaveBeenCalled();
 });
 it('bounds CSV bytes and rejects wrong content type',async()=>{
   for(const response of [new Response(new Uint8Array(10*1024*1024+1),{headers:{'Content-Type':'text/csv; charset=utf-8; header=present'}}),Response.json({})])
     expect((await createBridge(transport(response),()=>env)(request(csv))).status).toBe(502);
 });
 it('requires complete bounded journal selection without extra filters',async()=>{
   const path=reportsBase+`general-ledger/journals/${journal}?account_id=${account}&starts_on=2026-01-01&ends_before=2026-02-01&line_number=1&snapshot_revision=1`;
   expect((await createBridge(transport(),()=>env)(request(path))).status).toBe(200);
   for(const bad of [path.replace('&snapshot_revision=1',''),path.replace('line_number=1','line_number=1001'),path+'&cursor=abc']){
     const fetcher=vi.fn<typeof fetch>();expect((await createBridge(fetcher,()=>env)(request(bad))).status).toBe(404);expect(fetcher).not.toHaveBeenCalled();
   }
 });
});

describe("source review bridge", () => {
  const id = "00000000-0000-4000-8000-000000000001";
  const base = `/v1/entities/${id}/books/${id}`;
  it.each([
    `${base}/sources`,
    `${base}/review-items?source_account_id=${id}&limit=100`,
    `${base}/review-items?source_account_id=${id}&state=correction_hold`,
    `${base}/review-items/${id}`,
    `${base}/review-items/${id}/categories`,
  ])("forwards reviewed read %s", async (path) => {
    const fetcher = transport(Response.json({ amount: "9007199254740993.01" }));
    const response = await createBridge(fetcher, () => env)(request(path));
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ amount: "9007199254740993.01" });
  });
  it.each([
    `${base}/review-items?source_account_id=${id}&source_account_id=${id}`,
    `${base}/review-items?source_account_id=${id}&limit=101`,
    `${base}/review-items?source_account_id=${id}&actor=other`,
    `${base}/review-items`,
    "/v1/demo/acquisition",
  ])("rejects non-contract surface %s before network", async (path) => {
    const fetcher = vi.fn<typeof fetch>();
    expect((await createBridge(fetcher, () => env)(request(path))).status).toBe(
      404,
    );
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("retains the immutable draft request and its key without retrying", async () => {
    const fetcher = transport(Response.json({ draft_id: id }));
    const body = '{"description":"reviewed","source_revision":"4"}';
    const response = await createBridge(
      fetcher,
      () => env,
    )(
      request(
        `${base}/review-items/${id}/drafts`,
        { "Content-Type": "application/json", "Idempotency-Key": "same-draft" },
        body,
      ),
    );
    expect(response.status).toBe(200);
    expect(
      new TextDecoder().decode(fetcher.mock.calls[1][1]?.body as Uint8Array),
    ).toBe(body);
    expect(
      new Headers(fetcher.mock.calls[1][1]?.headers).get("Idempotency-Key"),
    ).toBe("same-draft");
  });
});

/** These seven routes carry connection intent only; SQL remains the authorization authority. */
describe('connection bridge routes', () => {
 const id='00000000-0000-4000-8000-0000000000ab';
 const base=`/v1/entities/${id}/books/${id}`;
 const routes=[
  ['POST','/connection-attempts',true],
  ['GET',`/connection-attempts/${id}`,false],
  ['POST',`/connection-attempts/${id}/execute`,false],
  ['POST',`/connection-attempts/${id}/reconcile`,false],
  ['POST',`/connection-attempts/${id}/bindings`,true],
  ['GET','/connections',false],
  ['GET','/connection-attempts?limit=100&before='+id,false],
  ['GET',`/connections/streams/${id}/status`,false],
  ['POST',`/connections/streams/${id}/operations`,true],
  ['POST',`/connections/streams/${id}/disconnect`,true],
 ] as const;
 it.each(routes)('allows only reviewed %s %s with required operation identity', async(method,suffix,key)=>{
  const fetcher=transport();
  const headers:Record<string,string>={'Content-Type':'application/json'};
  if(key) headers['Idempotency-Key']='synthetic-connection';
  const response=await createBridge(fetcher,()=>env)(request(base+suffix,headers,method==='POST'?'{}':undefined));
  expect(response.status).toBe(200);
  expect(fetcher.mock.calls[1][0]).toBe(env.BOOKKEEPING_API_ORIGIN+base+suffix);
  expect(fetcher.mock.calls[1][1]?.method).toBe(method);
  if(key){const denied=vi.fn<typeof fetch>();expect((await createBridge(denied,()=>env)(request(base+suffix,{'Content-Type':'application/json'},'{}'))).status).toBe(400);expect(denied).not.toHaveBeenCalled();}
 });
 it.each(routes)('rejects wrong method or query before metadata for %s %s',async(method,suffix)=>{
  for(const [path,verb] of [[base+suffix,method==='GET'?'POST':suffix==='/connection-attempts'?'PUT':'GET'],[base+suffix+'?cursor=secret',method],[base+suffix+'?',method],[base+suffix,'HEAD'],[base+suffix,'DELETE']]){
   const fetcher=vi.fn<typeof fetch>();
   const input=new Request(env.BOOKKEEPING_PUBLIC_ORIGIN+path,{method:verb,headers:{Authorization:bearer,Origin:env.BOOKKEEPING_PUBLIC_ORIGIN,'Content-Type':'application/json','Idempotency-Key':'synthetic'}});
   expect((await createBridge(fetcher,()=>env)(input)).status).toBe(404);expect(fetcher).not.toHaveBeenCalled();
  }
 });
 it.each(['/connection-attempts?limit=101','/connection-attempts?limit=1&limit=2','/connection-attempts?actor=other','/connection-attempts?before=','/connection-attempts?before=%xx','/connection-attempts/'+id+'/exchange','/connections/plaid','/connections/streams/'+id+'/sync','/connection-attempts/%30'+id.slice(1),'/connection-attempts/'+id.toUpperCase(),'/connection-attempts/'+id+'/execute%2f'])('rejects unreviewed or encoded suffix %s',async suffix=>{
  const fetcher=vi.fn<typeof fetch>();expect((await createBridge(fetcher,()=>env)(request(base+suffix,{'Content-Type':'application/json','Idempotency-Key':'synthetic'},'{}'))).status).toBe(404);expect(fetcher).not.toHaveBeenCalled();
 });
 it('forwards transient handoff bytes once without caching, retries or diagnostic leakage',async()=>{
  const body='{"nonce":"'+id+'","link_token":"synthetic-link","public_token":"synthetic-public"}';
  const fetcher=transport(new Response('synthetic-public',{status:503}));
  const result=await createBridge(fetcher,()=>env)(request(base+`/connection-attempts/${id}/execute`,{'Content-Type':'application/json','Cookie':'synthetic-cookie'},body));
  expect(fetcher).toHaveBeenCalledTimes(2);
  expect(new TextDecoder().decode(fetcher.mock.calls[1][1]?.body as Uint8Array)).toBe(body);
  expect(fetcher.mock.calls[1][1]?.cache).toBe('no-store');
  expect(new Headers(fetcher.mock.calls[1][1]?.headers).has('Cookie')).toBe(false);
  expect(result.status).toBe(503);expect(await result.text()).not.toContain('synthetic-public');
  expect(result.headers.get('Cache-Control')).toBe('no-store');
 });
 it('bounds handoff bytes and denies missing identity before backend or metadata access',async()=>{
  for(const [body,authorization,status] of [['x'.repeat((32<<10)+1),bearer,413],['{}','',401]] as const){
   const fetcher=vi.fn<typeof fetch>();
   expect((await createBridge(fetcher,()=>env)(request(base+`/connection-attempts/${id}/execute`,{'Content-Type':'application/json',Authorization:authorization},body))).status).toBe(status);expect(fetcher).not.toHaveBeenCalled();
  }
 });
});
