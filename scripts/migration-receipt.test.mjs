import {test} from 'node:test';
import assert from 'node:assert/strict';
import {migrationReceipt} from './migration-receipt.mjs';

// Independent provider-shaped fixture; no credentials, provider access or database content.
function fixture() {
 const m={operation:'migrate',project:'jayburd-synthetic-test',images:{migrate:'registry.invalid/migrate@sha256:'+'a'.repeat(64)},build:{run:'123'},evidence:{bootstrap:'synthetic'}};
 const name='jayburd-dev-migrate-abc12';
 const e={metadata:{name},spec:{taskCount:1,template:{spec:{serviceAccountName:'jayburd-dev-migrator@jayburd-synthetic-test.iam.gserviceaccount.com',maxRetries:0,timeoutSeconds:660,containers:[{image:m.images.migrate,args:['-action=up','-instance=jayburd-synthetic-test:us-central1:jayburd-dev']} ]}}},status:{succeededCount:1,conditions:[{type:'Completed',status:'True'}],completionTime:'2026-10-12T00:00:00Z'}};
 return {m,name,e};
}
test('exact successful migration emits only bounded evidence',()=>{
 const {m,name,e}=fixture();e.spec.template.spec.environment='must-not-be-copied';
 const r=migrationReceipt(m,e,name);assert.equal(r.status,'migration-execution-passed');
 assert.equal(r.execution,name);assert.equal(JSON.stringify(r).includes('must-not-be-copied'),false);
});
test('rejects foreign execution, wrong image/identity, unsafe arguments and unbounded task settings',()=>{
 const changes=[e=>e.metadata.name+='other',e=>e.spec.taskCount=2,e=>e.spec.template.spec.serviceAccountName='administrator',e=>e.spec.template.spec.maxRetries=1,e=>e.spec.template.spec.timeoutSeconds=900,
 e=>e.spec.template.spec.containers[0].image='wrong',e=>e.spec.template.spec.containers[0].command=['sh'],e=>e.spec.template.spec.containers[0].args[0]='-action=down',e=>e.spec.template.spec.containers.push({}),e=>delete e.spec.template.spec.maxRetries];
 for(const change of changes){const {m,name,e}=fixture();change(e);assert.throws(()=>migrationReceipt(m,e,name));}
});
test('rejects incomplete, failed, retried, cancelled or ambiguous completion',()=>{
 const changes=[e=>e.status.succeededCount=0,e=>e.status.failedCount=1,e=>e.status.cancelledCount=1,e=>e.status.runningCount=1,e=>e.status.retriedCount=1,e=>e.status.conditions[0].status='False',e=>e.status.conditions.push({type:'Completed',status:'False'}),e=>delete e.status.completionTime];
 for(const change of changes){const {m,name,e}=fixture();change(e);assert.throws(()=>migrationReceipt(m,e,name));}
 const {m,name,e}=fixture();assert.throws(()=>migrationReceipt({...m,operation:'rollback'},e,name));
});
