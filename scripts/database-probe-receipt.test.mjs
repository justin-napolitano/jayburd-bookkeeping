import test from 'node:test';
import assert from 'node:assert/strict';
import {databaseProbeReceipt} from './database-probe-receipt.mjs';
const image='us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:'+'a'.repeat(64);
const name='jayburd-dev-database-probe-abc12';
const execution={metadata:{name},spec:{taskCount:1,template:{spec:{serviceAccountName:'jayburd-dev-api@jayburd-bookkeeping-dev.iam.gserviceaccount.com',maxRetries:0,timeoutSeconds:'120',containers:[{image,command:['/database-probe'],args:['-action=probe','-instance=jayburd-bookkeeping-dev:us-central1:jayburd-dev']}]}}},status:{succeededCount:1,conditions:[{type:'Completed',status:'True'}],completionTime:'2026-10-15T15:00:00Z'}};
test('exact receipt passes',()=>assert.equal(databaseProbeReceipt(image,execution,name).status,'api-database-permission-proof-passed'));
test('substitution and incomplete execution fail closed',()=>{
  for(const mutate of [e=>e.spec.taskCount=2,e=>e.spec.template.spec.serviceAccountName='migrator',e=>e.spec.template.spec.maxRetries=1,e=>e.spec.template.spec.timeoutSeconds=121,e=>e.spec.template.spec.containers[0].image='different',e=>e.spec.template.spec.containers[0].command=[],e=>e.spec.template.spec.containers[0].args=['-action=verify'],e=>e.status.retriedCount=1,e=>e.status.failedCount=1,e=>e.status.completionTime='bad',e=>e.status.conditions=[]]) {
    const e=structuredClone(execution);mutate(e);assert.throws(()=>databaseProbeReceipt(image,e,name));
  }
});
