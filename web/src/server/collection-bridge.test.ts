// @vitest-environment node
import {describe, expect, it, vi} from "vitest";
import {createBridge} from "./bridge";

const workspace = "00000000-0000-4000-8000-000000000001";
const command = "00000000-0000-4000-8000-000000000002";
const base = `/v1/collections/${workspace}`;
const env = {BOOKKEEPING_API_MODE:"loopback", BOOKKEEPING_API_ORIGIN:"http://127.0.0.1:18080", BOOKKEEPING_PUBLIC_ORIGIN:"http://localhost:3000"};
const routes = [
  ["GET", "/v1/collection", 0],
  ["GET", `${base}/taxonomy`, 0],
  ["GET", `${base}/label-commands/pending`, 0],
  ["POST", `${base}/review/query`, 8192],
  ["POST", `${base}/labels/preview`, 524288],
  ["POST", `${base}/label-commands`, 4096],
  ...["execute","cancel","acknowledge"].map(action=>["POST",`${base}/label-commands/${command}/${action}`,1024]),
] as [string,string,number][];

/** Fictional user identity and intercepted loopback transport never reach a real service. */
function request(method:string, path:string, body?:string, extra:Record<string,string>={}) {
  return new Request(env.BOOKKEEPING_PUBLIC_ORIGIN+path, {method, headers:{Authorization:"Bearer synthetic.user.signature", Origin:env.BOOKKEEPING_PUBLIC_ORIGIN, ...(method==="POST"?{"Content-Type":"application/json"}:{}), ...extra}, body});
}

describe("private collection bridge",()=>{
  it.each(routes)("preserves exact private bytes on %s %s",async(method,path)=>{
    const body=method==="POST"?'{ "key": "fictional", "amount":"9007199254740993.01" }':undefined;
    const transport=vi.fn<typeof fetch>().mockResolvedValue(Response.json({result:"fictional"}));
    const response=await createBridge(transport,()=>env)(request(method,path,body,{"Cookie":"fictional","X-Actor-ID":"foreign"}));
    expect(response.status).toBe(200);
    expect(response.headers.get("Cache-Control")).toBe("no-store");
    expect(transport).toHaveBeenCalledTimes(1);
    const [url,init]=transport.mock.calls[0];
    expect(url).toBe(env.BOOKKEEPING_API_ORIGIN+path);
    const headers=new Headers(init?.headers);
    expect(headers.get("Authorization")).toBe("Bearer synthetic.user.signature");
    expect(headers.has("Cookie")).toBe(false);
    expect(headers.has("X-Actor-ID")).toBe(false);
    expect(headers.has("Idempotency-Key")).toBe(false);
    if(body!==undefined)expect(new TextDecoder().decode(init?.body as Uint8Array)).toBe(body);
  });
  it.each(routes.filter(r=>r[0]==="POST"))("enforces body cap for %s %s",async(method,path,cap)=>{
    const transport=vi.fn<typeof fetch>();
    const response=await createBridge(transport,()=>env)(request(method,path,"x".repeat(cap+1)));
    expect(response.status).toBe(413);
    expect(transport).not.toHaveBeenCalled();
  });
  it.each(routes)("rejects URL filters and foreign origin on %s %s",async(method,path)=>{
    const transport=vi.fn<typeof fetch>();
    const bridge=createBridge(transport,()=>env);
    expect((await bridge(request(method,path+"?search=fictional",method==="POST"?"{}":undefined))).status).toBe(404);
    expect((await bridge(request(method,path,method==="POST"?"{}":undefined,{Origin:"https://foreign.invalid"}))).status).toBe(403);
    expect(transport).not.toHaveBeenCalled();
  });
  it.each(["Idempotency-Key","Content-Encoding"])("rejects ambiguous %s before upstream",async(header)=>{
    const transport=vi.fn<typeof fetch>();
    const bridge=createBridge(transport,()=>env);
    for(const [method,path] of routes)expect((await bridge(request(method,path,method==="POST"?"{}":undefined,{[header]:"fictional"}))).status).toBe(400);
    expect(transport).not.toHaveBeenCalled();
  });
  it("rejects wrong methods and unreviewed routes",async()=>{
    const transport=vi.fn<typeof fetch>();
    const bridge=createBridge(transport,()=>env);
    for(const path of [`${base}/labels`,`${base}/label-commands/${command}/delete`,`${base}/label-commands/${command}/execute?`,`${base}/taxonomy?`])expect((await bridge(request("POST",path,"{}"))).status).toBe(404);
    expect((await bridge(request("HEAD",`${base}/taxonomy`))).status).toBe(404);
    expect(transport).not.toHaveBeenCalled();
  });
});
