import {test} from 'node:test';
import assert from 'node:assert/strict';
import {smoke,validateRevisions} from './deployment-smoke.mjs';
const manifest={project:'jayburd-synthetic-test',projectNumber:'123456789012',images:{api:'synthetic',web:'synthetic'},operation:'promote'};
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
