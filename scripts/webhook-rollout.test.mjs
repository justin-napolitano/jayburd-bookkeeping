import {test} from 'node:test';
import assert from 'node:assert/strict';
import {mkdtempSync,mkdirSync,writeFileSync,readFileSync,copyFileSync,existsSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawnSync} from 'node:child_process';
import {validateWebhookPolicy,validateWebhookDeployment} from './webhook-deployment-receipt.mjs';
import {collect,cloudReader,serviceFields,webhookReadURLs} from './collect-webhook-release.mjs';
import {ownerContext,validateAncestry,requiredWebhookAncestors,webhookSourcePaths} from './prepare-webhook-rollout.mjs';

// Original synthetic control-plane metadata only; no bank/provider or credential fixture.
const project='jayburd-bookkeeping-dev',number='9088794002',head='a'.repeat(40);
function manifest(){const evidence='https://github.com/justin-napolitano/jayburd-bookkeeping/issues/271#issuecomment-123';return {schemaVersion:1,project,projectNumber:number,operation:'sandbox-promote',build:{run:'123',attempt:'1',commit:head},image:`us-central1-docker.pkg.dev/${project}/jayburd-dev/worker@sha256:${'b'.repeat(64)}`,client_id_version:'1',client_secret_version:'2',evidence:Object.fromEntries(['security','bootstrap','identity','secrets','scope','compatibility'].map(k=>[k,evidence])),expiresAt:new Date(Date.now()+3600000).toISOString().replace(/\.\d{3}Z$/,'Z'),qualificationEndsAt:new Date(Date.now()+7200000).toISOString().replace(/\.\d{3}Z$/,'Z')};}
function service(m){const name=`projects/${project}/locations/us-central1/services/jayburd-dev-webhook`,revision=`${name}/revisions/jayburd-dev-webhook-00001-abc`;return {name,generation:'1',observedGeneration:'1',terminalCondition:{state:'CONDITION_SUCCEEDED'},ingress:'INGRESS_TRAFFIC_ALL',urls:[`https://jayburd-dev-webhook-${number}.us-central1.run.app`],latestReadyRevision:revision,latestCreatedRevision:revision,scaling:{maxInstanceCount:1},template:{serviceAccount:`jayburd-dev-webhook@${project}.iam.gserviceaccount.com`,timeout:'10s',maxInstanceRequestConcurrency:16,scaling:{maxInstanceCount:1},containers:[{image:m.image,command:['/webhook'],args:['-listen=0.0.0.0:8080',`-cloud-sql-instance=${project}:us-central1:jayburd-dev`,`-project-number=${number}`,`-plaid-client-id-version=projects/${number}/secrets/jayburd-dev-bank-sync-client-id/versions/1`,`-plaid-client-secret-version=projects/${number}/secrets/jayburd-dev-bank-sync-client-secret/versions/2`],ports:[{containerPort:8080}],resources:{limits:{cpu:'1',memory:'512Mi'},cpuIdle:true},startupProbe:{tcpSocket:{port:8080},periodSeconds:5,timeoutSeconds:3,failureThreshold:12}}]},trafficStatuses:[{revision,percent:100,type:'TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST'}]};}
function role(){return {name:`projects/${project}/roles/jayburdWebhookRelease`,includedPermissions:['run.services.update','run.services.get','run.services.getIamPolicy'],stage:'GA'};}
function policy(){return {bindings:[{role:'roles/run.invoker',members:['allUsers']},{role:`projects/${project}/roles/jayburdWebhookRelease`,members:[`serviceAccount:jayburd-dev-webhook-deployer@${project}.iam.gserviceaccount.com`]}]};}
function context(){return {GITHUB_ACTIONS:'true',GITHUB_EVENT_NAME:'workflow_dispatch',GITHUB_REF:'refs/heads/main',GITHUB_ACTOR_ID:'55423093',GITHUB_REPOSITORY_ID:'1414515318',GITHUB_TRIGGERING_ACTOR:'justin-napolitano',GITHUB_WORKFLOW_REF:'justin-napolitano/jayburd-bookkeeping/.github/workflows/deploy-webhook.yml@refs/heads/main',GITHUB_SHA:head,CURRENT_MAIN_SHA:head};}

test('optional updater needs actual exact role authority, never merely its name',()=>{
  const m=manifest();assert.equal(validateWebhookPolicy(m,policy(),role()),true);
  assert.throws(()=>validateWebhookPolicy(m,policy()));
  for(const mutate of [r=>r.deleted=true,r=>r.deleted=null,r=>r.stage='DISABLED',r=>r.name+='wrong',r=>r.includedPermissions.push('run.services.setIamPolicy'),r=>r.includedPermissions.pop()]){const r=role();mutate(r);assert.throws(()=>validateWebhookPolicy(m,policy(),r));}
  for(const mutate of [p=>p.bindings[1].members.push('allUsers'),p=>p.bindings[1].condition={expression:'true'},p=>p.bindings.push(p.bindings[1]),p=>p.bindings[1].role='roles/run.admin']){const p=policy();mutate(p);assert.throws(()=>validateWebhookPolicy(m,p,role()));}
  assert.equal(validateWebhookDeployment(m,service(m),policy(),Date.now(),role()).updaterRoleVerified,true);
});
test('collector fixes destinations, omits raw environment values and checks real current configuration',async()=>{
  const m=manifest(),s=service(m),urls=[];
  const read=async url=>{urls.push(url);if(url.includes(':getIamPolicy'))return policy();if(url.includes('iam.googleapis.com'))return role();return s;};
  const receipt=await collect(m,read);assert.equal(receipt.status,'webhook-control-plane-observed');assert.equal(urls.length,3);
  assert.ok(decodeURIComponent(urls[0]).includes('env(name)'));assert.ok(!serviceFields.includes('env(value'));
  s.template.containers[0].image=m.image.replace('b'.repeat(64),'c'.repeat(64));
  s.template.containers[0].args[3]=s.template.containers[0].args[3].replace('/versions/1','/versions/3');
  assert.equal((await collect(m,read,{preflight:true})).status,'existing-webhook-profile-observed');
  await assert.rejects(collect(m,read));
  s.template.containers[0].env=[{name:'SYNTHETIC_PRIVATE'}];await assert.rejects(collect(m,read,{preflight:true}));
  let calls=0;await assert.rejects(collect({...m,project:'unapproved-project'},async()=>{calls++;}));assert.equal(calls,0);
});
test('collector transport is bounded and never accepts redirects, errors or oversized results',async()=>{
  const original=globalThis.fetch;
  try{
    let options;
    globalThis.fetch=async(_url,o)=>{options=o;return new Response('{"synthetic":true}');};
    assert.deepEqual(await cloudReader('synthetic-in-memory-token')(webhookReadURLs[0]),{synthetic:true});
    assert.equal(options.redirect,'error');assert.ok(options.signal);await assert.rejects(cloudReader('synthetic')('https://unapproved.invalid/'));
    globalThis.fetch=async()=>new Response('synthetic-private-diagnostic',{status:403});await assert.rejects(cloudReader('synthetic')(webhookReadURLs[0]));
    globalThis.fetch=async()=>new Response('x'.repeat(131073));await assert.rejects(cloudReader('synthetic')(webhookReadURLs[0]));
  }finally{globalThis.fetch=original;}
});
test('owner context, immutable prerequisites and binary-source equality fail closed',()=>{
  ownerContext(context());for(const name of Object.keys(context())){const e=context();e[name]='wrong';assert.throws(()=>ownerContext(e));}
  const calls=[];validateAncestry(manifest(),head,args=>calls.push(args));
  assert.deepEqual(calls[0],['merge-base','--is-ancestor',head,head]);
  assert.deepEqual(calls.slice(1,3).map(a=>a[2]),requiredWebhookAncestors);
  assert.deepEqual(calls[3],['diff','--quiet',head,head,'--',...webhookSourcePaths]);
  for(let failAt=0;failAt<4;failAt++){let n=0;assert.throws(()=>validateAncestry(manifest(),head,()=>{if(n++===failAt)throw new Error('synthetic history mismatch');}));}
});

// Fake executables exercise actual shell sequencing offline, with no credentials, Terraform backend or network.
function shellFixture(){const dir=mkdtempSync(join(tmpdir(),'jayburd-webhook-rollout-'));for(const p of ['bin','scripts','release','evidence','infra/gcp/webhook-runtime'])mkdirSync(join(dir,p),{recursive:true});copyFileSync('scripts/deploy-webhook.sh',join(dir,'scripts/deploy-webhook.sh'));writeFileSync(join(dir,'infra/gcp/webhook-runtime/backend.tf.example'),'synthetic backend');
  const executable=(name,body)=>writeFileSync(join(dir,'bin',name),'#!/bin/bash\nset -eu\nprintf "%s\\n" "'+name+' $*" >> "$TRACE"\n'+body,{mode:0o700});
  executable('gh','[[ "$FAULT" != gh ]] || { echo synthetic-private-diagnostic >&2; exit 1; }; printf "%s\\n" "'+head+'"\n');
  executable('node','if [[ "$FAULT" == preflight && "$*" == *collect-webhook-release* ]]; then echo "BLOCKED: webhook collection rejected" >&2; exit 1; fi\nif [[ "$*" == *prepare-webhook-rollout* ]]; then n=0; [[ ! -f "$COUNTER" ]] || n=$(cat "$COUNTER"); n=$((n+1)); echo "$n" > "$COUNTER"; [[ "$FAULT" != expiry || "$n" != 2 ]] || { echo "BLOCKED: webhook rollout rejected" >&2; exit 1; }; fi\n');
  executable('terraform','case "$*" in *" plan "*) [[ "$FAULT" != plan ]] || { echo synthetic-private-diagnostic >&2; exit 1; } ;; *" apply "*) [[ "$FAULT" != apply ]] || { echo synthetic-private-diagnostic >&2; exit 1; } ;; esac\n');return dir;}
function invoke(dir,fault='',authorized=true,tokens=true){return spawnSync('/bin/bash',['scripts/deploy-webhook.sh',join(dir,'release'),join(dir,'evidence'),'f'.repeat(64)],{cwd:dir,encoding:'utf8',env:{PATH:join(dir,'bin')+':'+process.env.PATH,...(authorized?context():{}),...(tokens?{GOOGLE_OAUTH_ACCESS_TOKEN:'synthetic-memory-only',CLOUDSDK_AUTH_ACCESS_TOKEN:'synthetic-memory-only'}:{}),FAULT:fault,TRACE:join(dir,'trace'),COUNTER:join(dir,'counter')}});}
test('actual rollout shell rejects defaults/preflight/expiry/provider failures and redacts private diagnostics',()=>{
  const denied=shellFixture();assert.notEqual(invoke(denied,'',false).status,0);assert.equal(existsSync(join(denied,'trace')),false);const noToken=shellFixture();assert.notEqual(invoke(noToken,'',true,false).status,0);assert.equal(existsSync(join(noToken,'trace')),false);
  for(const fault of ['gh','preflight','plan','expiry','apply']){const dir=shellFixture(),r=invoke(dir,fault);assert.notEqual(r.status,0,fault);assert.ok(!(r.stdout+r.stderr).includes('synthetic-private-diagnostic'));const trace=readFileSync(join(dir,'trace'),'utf8');if(fault!=='apply')assert.ok(!trace.includes(' apply '));}
  const dir=shellFixture(),r=invoke(dir);assert.equal(r.status,0,r.stderr);const trace=readFileSync(join(dir,'trace'),'utf8');assert.equal((trace.match(/gh api/g)||[]).length,2);assert.ok(!trace.includes('state list'));assert.ok(trace.indexOf(' apply ')<trace.indexOf(' receipt '));
});
test('workflow pins owner/main controls and runs provenance/tests before explicit federation',()=>{
  const source=readFileSync('.github/workflows/deploy-webhook.yml','utf8');
  for(const text of ["github.actor_id == '55423093'","github.repository_id == '1414515318'","github.triggering_actor == 'justin-napolitano'","github.ref == 'refs/heads/main'",'create_credentials_file: false','export_environment_variables: false'])assert.ok(source.includes(text));
  assert.ok(source.indexOf('prepare-webhook-rollout.mjs')<source.indexOf('- id: auth'));
  assert.ok(source.indexOf('Run credential-free rollout')<source.indexOf('- id: auth'));
  assert.ok(!source.includes('deploy-workers.sh'));assert.ok(!source.includes('--allow-unauthenticated'));
});
