import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdtempSync,writeFileSync,readFileSync,existsSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawnSync} from 'node:child_process';
import {validateWorkerDeployment,workerConfigDigest} from './worker-deployment-receipt.mjs';
// Fixed synthetic control-plane fixtures contain no credentials or provider observations.
function fixture() {
  const m={schemaVersion:1,project:'jayburd-synthetic-test',projectNumber:'123456789012',operation:'promote',
    image:'us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/worker@sha256:'+'a'.repeat(64),
    binding_id:'11111111-1111-4111-8111-111111111111',client_id_version:'2',client_secret_version:'3',build:{run:'123',attempt:'1',commit:'a'.repeat(40)}};
  const observed={};
  for(const [component,suffix] of [['publisher','publisher'],['consumer','sync']]) {
    const name=`jayburd-dev-acquisition-${suffix}`;
    observed[component]={metadata:{name,generation:'7'},spec:{template:{spec:{serviceAccountName:`jayburd-dev-${suffix}@${m.project}.iam.gserviceaccount.com`,containers:[{image:m.image}]}}},
      status:{observedGeneration:7,conditions:[{type:'Ready',status:'True'}],latestCreatedRevisionName:name+'-00007-abc',latestReadyRevisionName:name+'-00007-abc',traffic:[{revisionName:name+'-00007-abc',percent:100,latestRevision:true}]}};
  }
  const origin='https://jayburd-dev-acquisition-publisher-123456789012.us-central1.run.app';
  observed.scheduler={name:'projects/jayburd-synthetic-test/locations/us-central1/jobs/jayburd-dev-acquisition-publisher',state:'PAUSED',httpTarget:{uri:origin+'/internal/acquisition/publish',httpMethod:'POST',oidcToken:{audience:origin,serviceAccountEmail:'jayburd-dev-scheduler@jayburd-synthetic-test.iam.gserviceaccount.com'}}};
  return {m,observed};
}
test('accepts reconciled sole-ready image and paused exact scheduler with bounded receipt',()=>{
  const {m,observed}=fixture();
  const r=validateWorkerDeployment(m,observed);
  assert.equal(r.status,'paused-workers-observed');assert.equal(r.scheduler,'PAUSED');assert.equal(r.image,m.image);
  assert.equal(r.revisions.consumer,'jayburd-dev-acquisition-sync-00007-abc');
  assert.equal(r.approvedWorkerConfigSHA256,createHash('sha256').update('{"projectNumber":"123456789012","binding_id":"11111111-1111-4111-8111-111111111111","client_id_version":"2","client_secret_version":"3","scheduler":"PAUSED"}').digest('hex'));
  assert.equal(Object.hasOwn(r,'httpTarget'),false);assert.equal(Object.hasOwn(r,'observed'),false);
});
test('rejects stale generation unready desired revisions and precision-losing generations',()=>{
  for(const mutate of [o=>o.publisher.status.observedGeneration=6,o=>o.consumer.status.latestCreatedRevisionName+='new',o=>o.publisher.status.conditions[0].status='False',o=>o.publisher.status.conditions.push({type:'Ready',status:'True'}),o=>o.consumer.metadata.generation=9007199254740992,o=>o.consumer.metadata.generation='07',o=>o.consumer.metadata.generation='0',o=>o.consumer.metadata.generation='9223372036854775808',o=>o.publisher.spec.template.metadata={name:'other'}]) {
    const {m,observed}=fixture();mutate(observed);assert.throws(()=>validateWorkerDeployment(m,observed));
  }
});
test('rejects unexpected service identity image sidecars and revision traffic',()=>{
  for(const mutate of [o=>o.publisher.metadata.name='other',o=>o.consumer.spec.template.spec.serviceAccountName='other',o=>o.publisher.spec.template.spec.containers[0].image='wrong',o=>o.consumer.spec.template.spec.containers.push({image:'other'}),o=>o.publisher.status.traffic[0].percent=99,o=>o.publisher.status.traffic[0].revisionName+='old',o=>o.consumer.status.traffic.push({revisionName:'old',percent:0}),o=>o.consumer.status.traffic[0].tag='alternate',o=>o.consumer.status.traffic[0].url='https://other.invalid']) {
    const {m,observed}=fixture();mutate(observed);assert.throws(()=>validateWorkerDeployment(m,observed));
  }
});
test('rejects enabled unknown foreign or misdirected scheduling without echoing diagnostics',()=>{
  for(const mutate of [s=>s.state='ENABLED',s=>s.state='DISABLED',s=>delete s.state,s=>s.name='other',s=>s.httpTarget.uri+='?target=other',s=>s.httpTarget.httpMethod='GET',s=>s.httpTarget.oidcToken.audience='other',s=>s.httpTarget.oidcToken.serviceAccountEmail='other',s=>s.httpTarget.oauthToken={},s=>s.pubsubTarget={}]) {
    const {m,observed}=fixture();mutate(observed.scheduler);assert.throws(()=>validateWorkerDeployment(m,observed),{message:'Worker readback rejected'});
  }
});
test('approved config digest is order-independent and binds each runtime pin but not image',()=>{
  const {m}=fixture();const original=workerConfigDigest(m);
  assert.equal(workerConfigDigest(Object.fromEntries(Object.entries(m).reverse())),original);
  for(const key of ['projectNumber','binding_id','client_id_version','client_secret_version']) assert.notEqual(workerConfigDigest({...m,[key]:m[key]+'1'}),original);
  assert.equal(workerConfigDigest({...m,image:'other'}),original);
});
test('CLI uses selected file names and fixed failure output without writing false receipts',()=>{
  const dir=mkdtempSync(join(tmpdir(),'jayburd-worker-receipt-'));
  try {
    const {m,observed}=fixture();const approved=join(dir,'approved.json'),output=join(dir,'receipt.json');
    writeFileSync(approved,JSON.stringify(m));
    for(const [key,file] of [['publisher','publisher-revision.json'],['consumer','consumer-revision.json'],['scheduler','scheduler.json']]) writeFileSync(join(dir,file),JSON.stringify(observed[key]));
    const invoke=()=>spawnSync(process.execPath,['scripts/worker-deployment-receipt.mjs',approved,dir,output],{encoding:'utf8'});
    assert.equal(invoke().status,0);assert.equal(JSON.parse(readFileSync(output)).status,'paused-workers-observed');rmSync(output);
    writeFileSync(join(dir,'scheduler.json'),JSON.stringify({privateDiagnostic:'synthetic-do-not-echo'}));
    const bad=invoke();assert.equal(bad.status,1);assert.equal(existsSync(output),false);assert.equal(bad.stderr,'BLOCKED: worker control-plane readback rejected\n');
  } finally {rmSync(dir,{recursive:true,force:true});}
});
