import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { validateProductionBuildProof } from './production-build-proof.mjs';

const now = Date.parse('2026-10-27T12:00:00Z');
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
// Independent fictional candidate; no real resources or build evidence.
function fixture() {
  const project = 'fictional-pilot-prod', projectNumber = '123456789012';
  return {
    schemaVersion:1, environment:'production', deployable:false, operation:'prepare',
    project, projectNumber, region:'us-central1', createdAt:'2026-10-27T11:00:00Z',
    expiresAt:'2026-10-28T11:00:00Z', scheduling:'paused',
    source:{commit:'a'.repeat(40),run:'123',attempt:'1'},
    images:Object.fromEntries(['api','web','worker','migrate'].map(c=>[c,`us-central1-docker.pkg.dev/${project}/jayburd-prod/${c}@sha256:${'b'.repeat(64)}`])),
    database:{instance:`${project}:us-central1:jayburd-prod`,name:'bookkeeping',migration:'35',checksumsSHA256:'c'.repeat(64)},
    secrets:{clientID:`projects/${projectNumber}/secrets/jayburd-prod-bank-sync-client-id/versions/1`,clientSecret:`projects/${projectNumber}/secrets/jayburd-prod-bank-sync-client-secret/versions/2`},
    keys:{bank:`projects/${project}/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/bank-credentials`,link:`projects/${project}/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/hosted-link`},
    evidence:Object.fromEntries(['mandate','build','schema','identity','secrets','recovery','qualification'].map(k=>[k,'d'.repeat(64)])),
  };
}
const encode = value => JSON.stringify(value)+'\n';
// Run metadata stands in for an independent authenticated GitHub read, never artifact content.
function inputs() {
  const m=fixture();
  const repository={id:1414515318,full_name:'justin-napolitano/jayburd-bookkeeping'};
  const owner={id:55423093,login:'justin-napolitano'};
  const run={path:'.github/workflows/build-production-images.yml',conclusion:'success',status:'completed',event:'workflow_dispatch',head_branch:'main',actor:{...owner},triggering_actor:{...owner},repository:{...repository},head_repository:{...repository},id:123,run_attempt:1,head_sha:m.source.commit};
  const bundle={schemaVersion:1,receipts:['api','web','worker','migrate'].map(component=>({schemaVersion:1,component,environment:'production',image:m.images[component],commit:m.source.commit,workflowRun:'https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/123/attempts/1',status:'candidate-only',deployable:false,scan:{severity:'HIGH,CRITICAL',result:'passed'},runtimeTests:component==='worker'?['worker','webhook']:[component]}))};
  return {m,run,bundle};
}
// Rebind changed receipts to exercise semantic rejection independently of digest mismatch.
function check({m,run,bundle}) {
  const bytes=encode(bundle);m.evidence.build=hash(bytes);
  return validateProductionBuildProof(encode(m),hash(encode(m)),bytes,run,now);
}
test('four same-build Production receipts match but cannot authorize deployment',()=>{
  assert.deepEqual(check(inputs()),{status:'build-metadata-matched',deployable:false,commit:'a'.repeat(40)});
});
test('trusted build metadata denies dev workflow forks reruns and unauthorized actors',()=>{
  for (const mutate of [
    r=>r.path='.github/workflows/build-images.yml',r=>r.conclusion='failure',r=>r.status='in_progress',
    r=>r.event='pull_request',r=>r.head_branch='feature',r=>r.actor.id=1,r=>r.actor.login='other',
    r=>r.triggering_actor.id=1,r=>r.triggering_actor.login='other',r=>r.repository.id=1,
    r=>r.repository.full_name='fork/repo',r=>r.head_repository.id=1,r=>delete r.head_repository,
    r=>r.id=124,r=>r.run_attempt=2,r=>r.head_sha='b'.repeat(40),
  ]) {const x=inputs();mutate(x.run);assert.throws(()=>check(x),/^Error: Production build proof rejected$/);}
});
test('complete tested scanned exact images required even with approved bundle hash',()=>{
  for (const mutate of [
    b=>b.receipts.pop(),b=>b.receipts[3]=b.receipts[0],b=>b.extra='secret-canary',b=>b.schemaVersion=2,
    b=>b.receipts[0].image=b.receipts[1].image,b=>b.receipts[0].image='repo:latest',
    b=>b.receipts[0].commit='b'.repeat(40),b=>b.receipts[0].workflowRun+='0',
    b=>b.receipts[0].environment='sandbox',b=>b.receipts[0].deployable=true,
    b=>b.receipts[0].status='deployed',b=>b.receipts[0].scan.result='failed',
    b=>b.receipts[0].scan.severity='CRITICAL',b=>b.receipts[0].scan.extra=true,
    b=>b.receipts[2].runtimeTests=['worker'],b=>b.receipts[0].runtimeTests=['web'],
    b=>b.receipts[0].secret='secret-canary',b=>delete b.receipts[0].scan,
  ]) {const x=inputs();mutate(x.bundle);assert.throws(()=>check(x),/^Error: Production build proof rejected$/);}
});
test('exact digest canonical bytes expired candidate and absent trusted metadata deny',()=>{
  const {m,run,bundle}=inputs(), bytes=encode(bundle);m.evidence.build=hash(bytes);
  const c=encode(m), h=hash(c);
  for (const bad of [bytes+' ', 'secret-canary',null,'x'.repeat(32769)])
    assert.throws(()=>validateProductionBuildProof(c,h,bad,run,now),/^Error: Production build proof rejected$/);
  assert.throws(()=>validateProductionBuildProof(c,h,bytes,null,now));
  assert.throws(()=>validateProductionBuildProof(c,h,bytes,run,Date.parse(m.expiresAt)));
  const noncanonical=JSON.stringify(bundle,null,2);m.evidence.build=hash(noncanonical);
  assert.throws(()=>validateProductionBuildProof(encode(m),hash(encode(m)),noncanonical,run,now));
});
