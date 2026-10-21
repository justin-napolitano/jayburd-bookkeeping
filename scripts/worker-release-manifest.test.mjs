import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdtempSync,readFileSync,writeFileSync,rmSync,readdirSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawnSync} from 'node:child_process';
import {validateWorkerManifest,validateWorkerBuild,validateWorkerRollback,workerGates} from './worker-release-manifest.mjs';
import {workerConfigDigest} from './worker-deployment-receipt.mjs';

const now=Date.parse('2026-10-14T00:00:00Z');
// Fixed identifiers and immutable bytes are synthetic; no provider or GitHub request is made.
export function manifest() {
  return {schemaVersion:1,project:'jayburd-synthetic-test',projectNumber:'123456789',operation:'promote',
    build:{run:'123',attempt:'1',commit:'a'.repeat(40)},image:'us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:'+'b'.repeat(64),
    binding_id:'11111111-1111-4111-8111-111111111111',client_id_version:'1',client_secret_version:'2',
    evidence:Object.fromEntries(['bootstrap','identity','secrets','compatibility'].map(k=>[k,'https://github.com/justin-napolitano/jayburd-bookkeeping/issues/7#issuecomment-1'])),
    expiresAt:'2026-10-15T00:00:00Z',previousDeployment:null};
}
// Hash the exact candidate bytes, mirroring owner-dispatch approval.
function validate(m) {const bytes=JSON.stringify(m);return validateWorkerManifest(bytes,createHash('sha256').update(bytes).digest('hex'),now);}
// Canonical successful build metadata is independent of the supplied artifact.
function run() {return {id:123,run_attempt:1,head_sha:'a'.repeat(40),path:'.github/workflows/build-images.yml',head_branch:'main',event:'workflow_dispatch',conclusion:'success',actor:{id:55423093},triggering_actor:{login:'justin-napolitano'},repository:{id:1414515318}};}
// A candidate is only packaging evidence, never deployment approval.
function candidate(m) {return {schemaVersion:1,component:'worker',image:m.image,commit:m.build.commit,workflowRun:'https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/123/attempts/1',status:'candidate-only',deployable:false};}
test('exact worker approval and fixed closure gates',()=>{
  const m=manifest();assert.deepEqual(validate(m),m);assert.deepEqual(workerGates,{issues:[5,6,11,26,162,163],prs:[109,110,111,165]});
  assert.throws(()=>validateWorkerManifest(JSON.stringify(m),'0'.repeat(64),now));
});
test('reject active scheduling, arbitrary input, invalid scope and numeric aliases',()=>{
  for(const [key,value] of [['scheduler_enabled',true],['schemaVersion',2],['operation','migrate'],['project','other/path'],['projectNumber','9223372036854775808'],['projectNumber',123456789],['binding_id','00000000-0000-0000-0000-000000000000'],['client_id_version','latest'],['client_secret_version','01'],['client_id_version','9223372036854775808'],['image',manifest().image.replace('/worker@','/api@')],['previousDeployment','12']]) assert.throws(()=>validate({...manifest(),[key]:value}),key);
  for(const bad of ['2026-10-14T00:00:00Z','2026-10-22T00:00:00Z','2026-10-15T00:00:00+00:00','2026-02-30T00:00:00Z']) assert.throws(()=>validate({...manifest(),expiresAt:bad}));
  const m=manifest();delete m.evidence.secrets;assert.throws(()=>validate(m));
});
test('candidate requires successful exact owner build and worker receipt',()=>{
  const m=manifest();validateWorkerBuild(m,run(),candidate(m));
  for(const [key,value] of [['path','.github/workflows/other.yml'],['head_branch','feature'],['event','push'],['conclusion','failure'],['run_attempt',2],['head_sha','c'.repeat(40)],['actor',{id:1}],['triggering_actor',{login:'other'}],['repository',{id:1}]]) assert.throws(()=>validateWorkerBuild(m,{...run(),[key]:value},candidate(m)),key);
  for(const [key,value] of [['component','api'],['image','other'],['deployable',true],['status','published'],['workflowRun','foreign']]) assert.throws(()=>validateWorkerBuild(m,run(),{...candidate(m),[key]:value}),key);
});
test('rollback requires prior successful worker receipt with identical configuration',()=>{
  const m={...manifest(),operation:'rollback',previousDeployment:'321'};
  validate(m);
  const r={...run(),id:321,path:'.github/workflows/deploy-workers.yml'};
  const receipt={status:'paused-workers-observed',project:m.project,image:m.image,approvedWorkerConfigSHA256:workerConfigDigest(m)};
  validateWorkerRollback(m,r,receipt);
  for(const [key,value] of [['client_id_version','3'],['client_secret_version','4'],['binding_id','22222222-2222-4222-8222-222222222222'],['projectNumber','987654321'],['image',m.image.replace('b'.repeat(64),'c'.repeat(64))]]) assert.throws(()=>validateWorkerRollback({...m,[key]:value},r,receipt),key);
  assert.throws(()=>validateWorkerRollback(m,{...r,path:'.github/workflows/deploy-dev.yml'},receipt));
  assert.throws(()=>validateWorkerRollback(m,r,{...receipt,approvedWorkerConfigSHA256:undefined}));
});
test('CLI emits only worker inputs with exact version strings after evidence validation',()=>{
  const dir=mkdtempSync(join(tmpdir(),'jayburd-worker-release-'));
  try {
    const m=manifest();m.expiresAt=new Date(Date.now()+86400000).toISOString().replace(/\.\d{3}Z$/,'Z');m.client_id_version='9223372036854775807';
    const bytes=JSON.stringify(m),hash=createHash('sha256').update(bytes).digest('hex');
    writeFileSync(join(dir,'manifest.json'),bytes);
    const result=spawnSync(process.execPath,['scripts/worker-release-manifest.mjs',join(dir,'manifest.json'),hash,join(dir,'out')],{encoding:'utf8'});
    assert.equal(result.status,0,result.stderr);
    assert.deepEqual(readdirSync(join(dir,'out')).sort(),['approved.json','worker-runtime.tfvars.json']);
    const tf=JSON.parse(readFileSync(join(dir,'out','worker-runtime.tfvars.json')));
    assert.deepEqual(tf,{project_id:m.project,project_number:m.projectNumber,region:'us-central1',runtime:{image:m.image,binding_id:m.binding_id,publisher_policy:'binding-v1',client_id_version:m.client_id_version,client_secret_version:m.client_secret_version}});
    // Missing retained build evidence must fail without writing a new approved directory.
    const denied=spawnSync(process.execPath,['scripts/worker-release-manifest.mjs',join(dir,'manifest.json'),hash,join(dir,'denied'),join(dir,'missing')],{encoding:'utf8'});
    assert.equal(denied.status,1);assert.equal(denied.stderr.trim(),'BLOCKED: worker release metadata or evidence rejected');
    assert.ok(!readdirSync(dir).includes('denied'));
  } finally {rmSync(dir,{recursive:true,force:true});}
});
test('shared publisher requires no pinned binding and changes the approval digest',()=>{
  const m={...manifest(),publisher_policy:'shared-v1',binding_id:''};
  assert.deepEqual(validate(m),m);
  assert.throws(()=>validate({...m,binding_id:manifest().binding_id}));
  assert.throws(()=>validate({...m,publisher_policy:'unknown'}));
  assert.notEqual(workerConfigDigest(m),workerConfigDigest(manifest()));
});
