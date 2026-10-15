import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {execFileSync} from 'node:child_process';
import {validateManifest,requiredGates,validateRollback,connectionConfigDigest} from './release-manifest.mjs';
import {validateWorkerManifest,requiredWorkerGates,validateWorkerRollback} from './worker-release-manifest.mjs';
import {workerConfigDigest} from './worker-deployment-receipt.mjs';
import {smoke} from './deployment-smoke.mjs';
const now=Date.parse('2026-10-15T00:00:00Z');
// Fixed public project identifiers exercise allowlisting; every image and credential identifier is fictional.
function app() {
 return {schemaVersion:2,project:'jayburd-bookkeeping-dev',projectNumber:'9088794002',operation:'sandbox-promote',
 build:{run:'123',attempt:'1',commit:'a'.repeat(40)},images:Object.fromEntries(['api','web','migrate'].map(c=>[c,`us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/${c}@sha256:${'b'.repeat(64)}`])),
 firebaseApiKey:'synthetic-public',connections:null,previousDeployment:null,expiresAt:'2026-10-16T00:00:00Z',
 evidence:Object.fromEntries(['bootstrap','identity','secrets','compatibility'].map(k=>[k,'https://github.com/justin-napolitano/jayburd-bookkeeping/issues/132#issuecomment-1']))};
}
// Worker configuration cannot carry arbitrary environment, schedule or provider overrides.
function worker() {
 const a=app();return {schemaVersion:1,project:a.project,projectNumber:a.projectNumber,operation:a.operation,build:a.build,
 image:a.images.api.replace('/api@','/worker@'),binding_id:'11111111-1111-4111-8111-111111111111',client_id_version:'1',client_secret_version:'2',evidence:a.evidence,expiresAt:a.expiresAt,previousDeployment:null};
}
// Rehash mutations so rejection proves semantic validation rather than hash mismatch alone.
function validate(m,fn=validateManifest) {const b=JSON.stringify(m);return fn(b,createHash('sha256').update(b).digest('hex'),now);}
// Synthetic prior run carries the immutable repository/owner/workflow binding.
function prior(path) {return {id:456,path,head_branch:'main',event:'workflow_dispatch',conclusion:'success',actor:{id:55423093},triggering_actor:{login:'justin-napolitano'},repository:{id:1414515318}};}
test('Sandbox accepts only fixed development project and explicit bounded policy',()=>{
 validate(app());validate(worker(),validateWorkerManifest);
 for(const [key,value] of [['project','other-project'],['projectNumber','123456789'],['expiresAt','2026-10-14T00:00:00Z'],['provider','production'],['evidence',{}]]) {
  assert.throws(()=>validate({...app(),[key]:value}));assert.throws(()=>validate({...worker(),[key]:value},validateWorkerManifest));
 }
 const missing=app();delete missing.connections;assert.throws(()=>validate(missing));
 assert.throws(()=>validate({...app(),connections:{environment:'production'}}));
 assert.throws(()=>validate({...worker(),scheduler_enabled:true},validateWorkerManifest));
});
test('Sandbox gates retain database proof and connection implementation while full gates stay intact',()=>{
 for(const op of ['sandbox-promote','sandbox-rollback']) {
  assert.deepEqual(requiredGates(op),{issues:[5,6,130],prs:[23,24,129]});
  assert.deepEqual(requiredGates(op,{}),{issues:[5,6,130,131],prs:[23,24,129]});
  assert.deepEqual(requiredWorkerGates(op),{issues:[5,6,130,131],prs:[109,110,111,129]});
 }
 for(const op of ['promote','rollback']) {
  assert.deepEqual(requiredGates(op),{issues:[6,11,26],prs:[23,24]});
  assert.deepEqual(requiredWorkerGates(op),{issues:[5,6,11,26],prs:[109,110,111]});
 }
 assert.throws(()=>requiredWorkerGates('force'));
});
test('Sandbox runtime CLI configures status, never up',()=>{
 const dir=mkdtempSync(join(tmpdir(),'jayburd-sandbox-'));
 try {
  const m=app();m.expiresAt=new Date(Date.now()+86400000).toISOString().replace(/\.\d{3}Z$/,'Z');
  const bytes=JSON.stringify(m);writeFileSync(join(dir,'manifest.json'),bytes);
  execFileSync(process.execPath,['scripts/release-manifest.mjs',join(dir,'manifest.json'),createHash('sha256').update(bytes).digest('hex'),join(dir,'out')]);
  assert.equal(JSON.parse(readFileSync(join(dir,'out/migration.tfvars.json'))).action,'status');
  assert.equal(JSON.parse(readFileSync(join(dir,'out/runtime.tfvars.json'))).connections,null);
 } finally {rmSync(dir,{recursive:true,force:true});}
});
test('app rollback cannot cross Sandbox/full receipt boundary or change connection policy',()=>{
 const m={...app(),operation:'sandbox-rollback',previousDeployment:'456'};validate(m);
 const r={status:'transport-smoke-passed',operation:'sandbox-promote',project:m.project,images:m.images,approvedConnectionConfigSHA256:connectionConfigDigest(null)};
 const run=prior('.github/workflows/deploy-dev.yml');validateRollback(m,run,r);
 assert.throws(()=>validateRollback(m,run,{...r,operation:'promote'}));
 assert.throws(()=>validateRollback({...m,operation:'rollback'},run,r));
 assert.throws(()=>validateRollback(m,run,{...r,approvedConnectionConfigSHA256:'wrong'}));
});
test('worker rollback cannot cross Sandbox/full or change binding/version policy',()=>{
 const m={...worker(),operation:'sandbox-rollback',previousDeployment:'456'};validate(m,validateWorkerManifest);
 const r={status:'paused-workers-observed',operation:'sandbox-promote',project:m.project,image:m.image,approvedWorkerConfigSHA256:workerConfigDigest(m)};
 const run=prior('.github/workflows/deploy-workers.yml');validateWorkerRollback(m,run,r);
 assert.throws(()=>validateWorkerRollback(m,run,{...r,operation:'promote'}));
 assert.throws(()=>validateWorkerRollback({...m,operation:'rollback'},run,r));
 assert.throws(()=>validateWorkerRollback({...m,client_id_version:'3'},run,r));
});
test('anonymous transport receipt explicitly limits Sandbox qualification',async()=>{
 const statuses=[403,401,200,200];
 const r=await smoke(app(),async()=>({status:statuses.shift()}));
 assert.equal(r.releaseScope,'sandbox-qualification');assert.equal(r.operation,'sandbox-promote');
});
