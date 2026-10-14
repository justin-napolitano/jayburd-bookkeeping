import {test} from 'node:test';
import assert from 'node:assert/strict';
import {smoke,validateRevisions} from './deployment-smoke.mjs';
const manifest={connections:null,project:'jayburd-synthetic-test',projectNumber:'123456789012',images:{api:'synthetic',web:'synthetic'},operation:'promote'};
test('probes no-login transport without obtaining credentials or following redirects',async()=>{
 const calls=[];const statuses=[403,401,200,200];
 const result=await smoke(manifest,async(url,options)=>{calls.push({url,options});return new Response('',{status:statuses.shift()});});
 assert.equal(result.status,'transport-smoke-passed');assert.equal(calls.length,4);
 for(const {url,options} of calls) {assert.ok(url.startsWith('https://jayburd-dev-'));assert.equal(options.redirect,'manual');assert.equal(options.credentials,'omit');assert.equal(options.headers,undefined);}
});
test('public API, broken bridge, unavailable config and redirects fail closed',async()=>{
 for(const statuses of [[200],[403,503],[403,401,503],[403,401,302]]) await assert.rejects(smoke(manifest,async()=>new Response('',{status:statuses.shift()})));
});

test('receipt requires matching observed digest, ready revision and full traffic',()=>{
 const observed=Object.fromEntries(['api','web'].map(c=>[c,{status:{latestReadyRevisionName:`jayburd-dev-${c}-00001-test`,traffic:[{revisionName:`jayburd-dev-${c}-00001-test`,percent:100}]},spec:{template:{spec:{containers:[{image:'synthetic'}]}}}}]));
 assert.equal(validateRevisions(manifest,observed).api,'jayburd-dev-api-00001-test');
 for(const mutate of [v=>{v.api.spec.template.spec.containers[0].image='wrong';},v=>{v.api.status.traffic[0].percent=50;},v=>{v.api.status.latestReadyRevisionName='wrong';}]) {
  const changed=structuredClone(observed);mutate(changed);assert.throws(()=>validateRevisions(manifest,changed));
 }
});

test('transport receipt binds approved configuration without claiming provider readiness',async()=>{
 const c={infisical_origin:'https://us.infisical.com',infisical_project:'11111111-1111-4111-8111-111111111111',infisical_identity:'22222222-2222-4222-8222-222222222222',link_events_qualified:true,client_id_version:'projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1',client_secret_version:'projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2',consent_expires_at:'2026-10-20T00:00:00Z'};
 const probe=async(m)=>{const statuses=[403,401,200,200];return smoke(m,async()=>new Response('',{status:statuses.shift()}));};
 const disabled=await probe(manifest),enabled=await probe({...manifest,connections:c});
 assert.match(enabled.approvedConnectionConfigSHA256,/^[a-f0-9]{64}$/);
 assert.notEqual(disabled.approvedConnectionConfigSHA256,enabled.approvedConnectionConfigSHA256);
 assert.ok(enabled.limitations.some(v=>v.includes('not provider')));
 assert.equal(JSON.stringify(enabled).includes(c.infisical_identity),false);
});
