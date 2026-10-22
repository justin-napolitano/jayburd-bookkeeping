import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdtempSync,writeFileSync,mkdirSync,readFileSync,existsSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawnSync} from 'node:child_process';
import {validateWebhookManifest,validateWebhookBuild,webhookConfigDigest,webhookURL} from './webhook-release-manifest.mjs';
import {validateWebhookDeployment} from './webhook-deployment-receipt.mjs';
import {validateConnections,connectionConfigDigest} from './release-manifest.mjs';

const now=Date.parse('2026-10-22T18:00:00Z');
// Fixed fictional metadata and independently authored control-plane response; never provider data.
function manifest() {
  const evidence='https://github.com/justin-napolitano/jayburd-bookkeeping/issues/271#issuecomment-123';
  return {schemaVersion:1,project:'jayburd-bookkeeping-dev',projectNumber:'9088794002',operation:'sandbox-promote',
    build:{run:'12345',attempt:'1',commit:'a'.repeat(40)},image:`us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:${'b'.repeat(64)}`,
    client_id_version:'1',client_secret_version:'2',evidence:Object.fromEntries(['security','bootstrap','identity','secrets','scope','compatibility'].map(k=>[k,evidence])),
    expiresAt:'2026-10-22T19:00:00Z',qualificationEndsAt:'2026-10-22T20:00:00Z'};
}
function check(m) {const bytes=JSON.stringify(m);return validateWebhookManifest(bytes,createHash('sha256').update(bytes).digest('hex'),now);}
function build(m) {
  return [{id:12345,run_attempt:1,head_sha:m.build.commit,head_branch:'main',event:'workflow_dispatch',path:'.github/workflows/build-images.yml',conclusion:'success',actor:{id:55423093},triggering_actor:{login:'justin-napolitano'},repository:{id:1414515318}},
    {schemaVersion:1,component:'worker',image:m.image,commit:m.build.commit,workflowRun:'https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/12345/attempts/1',status:'candidate-only',deployable:false}];
}
function observed(m) {
  const name='projects/jayburd-bookkeeping-dev/locations/us-central1/services/jayburd-dev-webhook';
  const revision=`${name}/revisions/jayburd-dev-webhook-00001-abc`;
  return {name,generation:'1',observedGeneration:'1',reconciling:false,terminalCondition:{state:'CONDITION_SUCCEEDED'},ingress:'INGRESS_TRAFFIC_ALL',invokerIamDisabled:false,urls:['https://jayburd-dev-webhook-9088794002.us-central1.run.app'],
    latestReadyRevision:revision,latestCreatedRevision:revision,scaling:{minInstanceCount:0,maxInstanceCount:1},
    template:{serviceAccount:'jayburd-dev-webhook@jayburd-bookkeeping-dev.iam.gserviceaccount.com',timeout:'10s',maxInstanceRequestConcurrency:16,scaling:{minInstanceCount:0,maxInstanceCount:1},containers:[{
      image:m.image,command:['/webhook'],args:['-listen=0.0.0.0:8080','-cloud-sql-instance=jayburd-bookkeeping-dev:us-central1:jayburd-dev','-project-number=9088794002',
        '-plaid-client-id-version=projects/9088794002/secrets/jayburd-dev-bank-sync-client-id/versions/1',
        '-plaid-client-secret-version=projects/9088794002/secrets/jayburd-dev-bank-sync-client-secret/versions/2'],
      ports:[{containerPort:8080}],resources:{limits:{cpu:'1',memory:'512Mi'},cpuIdle:true,startupCpuBoost:false},startupProbe:{tcpSocket:{port:8080},periodSeconds:5,timeoutSeconds:3,failureThreshold:12}}]},
    trafficStatuses:[{revision,percent:100,type:'TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST'}]};
}
const policy=()=>({bindings:[{role:'roles/run.invoker',members:['allUsers']}]});

test('exact finite Sandbox approval rejects arbitrary runtime and stale metadata',()=>{
  assert.equal(check(manifest()).projectNumber,'9088794002');
  for(const mutate of [m=>m.operation='promote',m=>m.operation='sandbox-rollback',m=>m.project='other-project',m=>m.projectNumber='123456789',m=>m.image=m.image.replace('worker@','api@'),
    m=>m.image=m.image.replace(/@.*/,':latest'),m=>m.client_id_version='latest',m=>m.client_secret_version='9223372036854775808',m=>m.client_id_version=1,
    m=>m.build.attempt='10000',m=>m.build.run='01',m=>m.build.commit='main',m=>m.expiresAt='2026-10-22T17:00:00Z',m=>m.expiresAt='2026-10-24T19:00:00Z',
    m=>m.qualificationEndsAt=m.expiresAt,m=>m.qualificationEndsAt='2026-10-25T20:00:00Z',m=>m.expiresAt='2026-02-30T19:00:00Z',m=>m.evidence.scope='https://example.com/approve',
    m=>m.token='synthetic-private-sentinel',m=>delete m.evidence.security,m=>m.build.extra=true]) {
    const m=manifest();mutate(m);assert.throws(()=>check(m));
  }
  assert.throws(()=>validateWebhookManifest(JSON.stringify(manifest()),'c'.repeat(64),now));
});
test('trusted same-source worker build is independent of owner release approval',()=>{
  const m=manifest(),[run,receipt]=build(m);validateWebhookBuild(m,run,receipt);
  for(const mutate of [r=>r.conclusion='failure',r=>r.event='pull_request',r=>r.head_sha='d'.repeat(40),r=>r.actor.id=1,r=>r.run_attempt=2,r=>r.repository.id=1]) {
    const r=structuredClone(run);mutate(r);assert.throws(()=>validateWebhookBuild(m,r,receipt));
  }
  for(const mutate of [r=>r.image=r.image.replace('worker@','api@'),r=>r.deployable=true,r=>r.commit='e'.repeat(40)]) {
    const r=structuredClone(receipt);mutate(r);assert.throws(()=>validateWebhookBuild(m,run,r));
  }
});
test('readback rejects identity, command, routing, resource and IAM drift',()=>{
  const m=manifest(),s=observed(m),p=policy();
  const receipt=validateWebhookDeployment(m,s,p,now);
  assert.equal(receipt.url,webhookURL);assert.equal(receipt.approvedWebhookConfigSHA256,webhookConfigDigest(m));
  for(const mutate of [s=>s.observedGeneration='2',s=>s.reconciling=true,s=>s.latestReadyRevision+='old',s=>s.terminalCondition.state='CONDITION_FAILED',
    s=>s.ingress='INGRESS_TRAFFIC_INTERNAL_ONLY',s=>s.invokerIamDisabled=true,s=>s.template.serviceAccount=s.template.serviceAccount.replace('webhook','api'),
    s=>s.defaultUriDisabled=true,s=>s.iapEnabled=true,s=>s.urls=['https://elsewhere.invalid'],s=>s.scaling.scalingMode='MANUAL',s=>s.scaling.manualInstanceCount=100,
    s=>s.reconciling=null,s=>s.scaling.minInstanceCount=null,s=>s.template.containers[0].ports[0].name='h2c',
    s=>s.template.containers[0].image+='x',s=>s.template.containers[0].command=['/worker'],s=>s.template.containers[0].args.push('-listen=127.0.0.1:8080'),
    s=>s.template.containers[0].args[3]+='0',s=>s.template.containers[0].env=[{name:'SECRET',value:'synthetic-private-sentinel'}],
    s=>s.template.containers.push(s.template.containers[0]),s=>s.template.containers[0].ports[0].containerPort=80,
    s=>s.template.scaling.maxInstanceCount=2,s=>s.scaling.maxInstanceCount=2,s=>s.template.maxInstanceRequestConcurrency=80,s=>s.template.timeout='300s',
    s=>s.template.containers[0].resources.cpuIdle=false,s=>s.template.containers[0].resources.limits.memory='1Gi',s=>s.template.containers[0].startupProbe.tcpSocket.port=80,
    s=>s.trafficStatuses[0].percent=99,s=>s.trafficStatuses[0].tag='preview',s=>s.trafficStatuses.push({...s.trafficStatuses[0]})]) {
    const changed=observed(m);mutate(changed);assert.throws(()=>validateWebhookDeployment(m,changed,p,now));
  }
  for(const p of [{bindings:[]},{bindings:[...policy().bindings,{role:'roles/run.admin',members:['allUsers']}]},
    {bindings:[{role:'roles/run.invoker',members:['allUsers'],condition:{expression:'true'}}]},
    {bindings:[{role:'roles/run.invoker',members:['allUsers','user:synthetic@example.invalid']}]}]) assert.throws(()=>validateWebhookDeployment(m,s,p,now));
  assert.throws(()=>validateWebhookDeployment(m,s,policy(),Date.parse(m.expiresAt)));
  delete s.reconciling;delete s.invokerIamDisabled;delete s.scaling.minInstanceCount;delete s.template.scaling.minInstanceCount;delete s.template.containers[0].resources.startupCpuBoost;
  assert.equal(validateWebhookDeployment(m,s,policy(),now).status,'webhook-control-plane-observed');
});
test('enrollment opt-in is fixed and approval digest changes while legacy digest remains stable',()=>{
  const c={infisical_origin:'https://app.infisical.com',infisical_project:'10000000-0000-4000-8000-000000000001',infisical_identity:'10000000-0000-4000-8000-000000000002',link_events_qualified:false,
    client_id_version:'projects/9088794002/secrets/jayburd-dev-bank-sync-client-id/versions/1',client_secret_version:'projects/9088794002/secrets/jayburd-dev-bank-sync-client-secret/versions/2',consent_expires_at:'2026-10-23T20:00:00Z'};
  validateConnections(c,'9088794002',now+3600000,now);
  const digest=connectionConfigDigest(c),enabled={...c,webhook_url:webhookURL};validateConnections(enabled,'9088794002',now+3600000,now);
  assert.notEqual(connectionConfigDigest(enabled),digest);
  for(const value of ['',null,'https://evil.invalid/webhooks/plaid',webhookURL+'?token=x',webhookURL+'/']) assert.throws(()=>validateConnections({...c,webhook_url:value},'9088794002',now+3600000,now));
  assert.equal(connectionConfigDigest({...c}),digest);
});
test('offline CLI requires build evidence and writes only reviewed isolated inputs',()=>{
  const dir=mkdtempSync(join(tmpdir(),'jayburd-webhook-release-')),m=manifest();
  m.expiresAt=new Date(Date.now()+3600000).toISOString().replace(/\.\d{3}Z$/,'Z');
  m.qualificationEndsAt=new Date(Date.now()+7200000).toISOString().replace(/\.\d{3}Z$/,'Z');
  const bytes=JSON.stringify(m),hash=createHash('sha256').update(bytes).digest('hex');
  writeFileSync(join(dir,'manifest.json'),bytes);mkdirSync(join(dir,'worker'));
  const [run,receipt]=build(m);writeFileSync(join(dir,'build-run.json'),JSON.stringify(run));writeFileSync(join(dir,'worker/candidate.json'),JSON.stringify(receipt));
  const invoke=(...args)=>spawnSync(process.execPath,['scripts/webhook-release-manifest.mjs',...args],{encoding:'utf8',env:{PATH:process.env.PATH}});
  assert.notEqual(invoke(join(dir,'manifest.json'),hash,join(dir,'missing')).status,0);
  assert.equal(existsSync(join(dir,'missing')),false);
  const r=invoke(join(dir,'manifest.json'),hash,join(dir,'prepared'),dir);assert.equal(r.status,0,r.stderr);
  const vars=JSON.parse(readFileSync(join(dir,'prepared/webhook-runtime.tfvars.json')));
  assert.deepEqual(vars,{project_id:m.project,project_number:m.projectNumber,region:'us-central1',runtime:{image:m.image,client_id_version:'1',client_secret_version:'2'}});
  assert.equal(existsSync(join(dir,'prepared/runtime.tfvars.json')),false);
  writeFileSync(join(dir,'worker/candidate.json'),'synthetic-private-sentinel');
  const rejected=invoke(join(dir,'manifest.json'),hash,join(dir,'rejected'),dir);assert.notEqual(rejected.status,0);
  assert.equal(existsSync(join(dir,'rejected')),false);assert.doesNotMatch(rejected.stderr,/sentinel|SyntaxError|candidate.json/);
});
