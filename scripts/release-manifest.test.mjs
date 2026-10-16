import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdtempSync,writeFileSync,readFileSync,existsSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {execFileSync} from 'node:child_process';
import {validateManifest,validateBuild,validateRollback,requiredGates,connectionConfigDigest} from './release-manifest.mjs';
const now=Date.parse('2026-10-12T00:00:00Z');
// Independent synthetic release/build evidence; no real identities or provider clients.
function fixture() {
 const m={schemaVersion:2,connections:null,project:'jayburd-synthetic-test',projectNumber:'123456789012',operation:'promote',
 build:{run:'123',attempt:'1',commit:'a'.repeat(40)},firebaseApiKey:'synthetic-public',expiresAt:'2026-10-13T00:00:00Z',previousDeployment:null,
 evidence:Object.fromEntries(['bootstrap','identity','secrets','compatibility'].map(k=>[k,'https://github.com/justin-napolitano/jayburd-bookkeeping/issues/27#issuecomment-123'])),
 images:Object.fromEntries(['api','migrate','web'].map(c=>[c,`us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/${c}@sha256:${'a'.repeat(64)}`]))};
 const run={id:123,run_attempt:1,head_sha:m.build.commit,head_branch:'main',event:'workflow_dispatch',path:'.github/workflows/build-images.yml',conclusion:'success',actor:{id:55423093},triggering_actor:{login:'justin-napolitano'},repository:{id:1414515318}};
 const receipts=['api','migrate','web'].map(component=>({schemaVersion:1,component,image:m.images[component],commit:m.build.commit,workflowRun:'https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/123/attempts/1',status:'candidate-only',deployable:false}));
 return {m,run,receipts};
}
// Hash binds the exact file bytes; tests can separately mutate semantic inputs after a fresh owner approval.
function validate(m) {const bytes=Buffer.from(JSON.stringify(m)); return validateManifest(bytes,createHash('sha256').update(bytes).digest('hex'),now);}
test('accepts reviewed immutable release and matching successful owner build',()=>{const {m,run,receipts}=fixture();assert.deepEqual(validate(m),m);validateBuild(m,run,receipts);});
test('rejects hash mismatch, expired or excessively long authorization and unknown fields',()=>{
 const {m}=fixture(); assert.throws(()=>validateManifest(Buffer.from(JSON.stringify(m)),'f'.repeat(64),now));
 for(const change of [{expiresAt:'2026-10-11T00:00:00Z'},{expiresAt:'2027-01-01T00:00:00Z'},{extra:'arbitrary Terraform input'},{project:'evil;command'},{operation:'destroy'},{evidence:{}}]) assert.throws(()=>validate({...m,...change}));
});
test('rejects tags, foreign registries, component swaps and wrong project digests',()=>{
 for(const image of ['api:latest','evil.invalid/api@sha256:'+'a'.repeat(64),fixture().m.images.web,fixture().m.images.api.replace('synthetic','other')]) {const {m}=fixture();m.images.api=image;assert.throws(()=>validate(m));}
});
test('rejects failed, fork, PR, other-owner, other-workflow and mismatched run evidence',()=>{
 const {m,run,receipts}=fixture();
 for(const change of [{conclusion:'failure'},{repository:{id:1}},{actor:{id:1}},{triggering_actor:{login:'other'}},{event:'pull_request'},{head_branch:'feature'},{path:'.github/workflows/other.yml'},{run_attempt:2},{head_sha:'b'.repeat(40)}]) assert.throws(()=>validateBuild(m,{...run,...change},receipts));
});
test('rejects missing, duplicated and mixed build receipts',()=>{
 const {m,run,receipts}=fixture();
 for(const values of [receipts.slice(1),[receipts[0],receipts[0],receipts[2]],receipts.map(r=>({...r,commit:'b'.repeat(40)})),receipts.map(r=>({...r,image:r.image+'extra'}))]) assert.throws(()=>validateBuild(m,run,values));
});
test('rollback needs a successful prior deployment and exact prior images',()=>{
 const {m,run}=fixture();m.operation='rollback';assert.throws(()=>validate(m));m.previousDeployment='456';validate(m);
 const prior={...run,id:456,path:'.github/workflows/deploy-dev.yml'};
 const receipt={status:'transport-smoke-passed',project:m.project,images:m.images,approvedConnectionConfigSHA256:connectionConfigDigest(m.connections)};validateRollback(m,prior,receipt);
 assert.throws(()=>validateRollback(m,{...prior,conclusion:'failure'},receipt));
 assert.throws(()=>validateRollback(m,prior,{...receipt,images:{...m.images,api:'other'}}));
});
test('migration-only release omits runtime configuration but still requires complete owner build',()=>{
 const {m,run,receipts}=fixture();m.operation='migrate';m.schemaVersion=1;delete m.connections;delete m.firebaseApiKey;delete m.previousDeployment;
 delete m.evidence.identity;delete m.evidence.secrets;
 assert.deepEqual(validate(m),m);validateBuild(m,run,receipts);
 assert.throws(()=>validateBuild(m,run,receipts.slice(1)));
 assert.throws(()=>validate({...m,firebaseApiKey:'unexpected-runtime-setting'}));
 assert.throws(()=>validate({...m,previousDeployment:null}));
 assert.throws(()=>validate({...m,connections:null}));
 for(const operation of ['promote','rollback']) assert.throws(()=>validate({...m,operation}));
 assert.throws(()=>validate({...m,evidence:{compatibility:m.evidence.compatibility}}));
});
test('migration prerequisites cannot relax full application closure gates',()=>{
 assert.deepEqual(requiredGates('migrate'),{issues:[5],prs:[44]});
 assert.deepEqual(requiredGates('verify'),{issues:[5],prs:[44]});
 for(const operation of ['promote','rollback']) assert.deepEqual(requiredGates(operation),{issues:[6,11,26,162,163],prs:[23,24,165]});
 assert.throws(()=>requiredGates('force'));
});
test('read-only verification preserves the complete build gate and rejects runtime inputs',()=>{
 const {m,run,receipts}=fixture();m.operation='verify';m.schemaVersion=1;delete m.connections;delete m.firebaseApiKey;delete m.previousDeployment;
 delete m.evidence.identity;delete m.evidence.secrets;
 validate(m);validateBuild(m,run,receipts);
 assert.throws(()=>validateBuild(m,{...run,conclusion:'failure'},receipts));
 assert.throws(()=>validate({...m,firebaseApiKey:'unexpected'}));
});
test('release CLI maps verification to status without producing service configuration',()=>{
 for (const [operation,action] of [['verify','status'],['migrate','up']]) {
  const dir=mkdtempSync(join(tmpdir(),'jayburd-release-inputs-'));
  try {
   const {m}=fixture();m.operation=operation;m.schemaVersion=1;delete m.connections;delete m.firebaseApiKey;delete m.previousDeployment;
   delete m.evidence.identity;delete m.evidence.secrets;
   m.expiresAt=new Date(Date.now()+86400000).toISOString().replace(/\.\d{3}Z$/,'Z');
   const bytes=JSON.stringify(m),path=join(dir,'manifest.json'),out=join(dir,'out');writeFileSync(path,bytes);
   execFileSync(process.execPath,['scripts/release-manifest.mjs',path,createHash('sha256').update(bytes).digest('hex'),out]);
   assert.equal(JSON.parse(readFileSync(join(out,'migration.tfvars.json'))).action,action);
   assert.equal(existsSync(join(out,'runtime.tfvars.json')),false);
  } finally {rmSync(dir,{recursive:true,force:true});}
 }
});

// Public synthetic identifiers mirror the independent Terraform/API contract, never credential values.
function connectionFixture() {
 return {infisical_origin:'https://us.infisical.com',infisical_project:'11111111-1111-4111-8111-111111111111',infisical_identity:'22222222-2222-4222-8222-222222222222',link_events_qualified:true,
 client_id_version:'projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/7',
 client_secret_version:'projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/23',consent_expires_at:'2026-10-20T00:00:00Z'};
}
test('runtime requires explicit connection state; legacy/omitted/partial states fail before output',()=>{
 const {m}=fixture();validate(m);m.connections=connectionFixture();validate(m);
 for (const c of [undefined,{},false,'enabled',[],{...m.connections,extra:'unreviewed'}]) assert.throws(()=>validate({...m,connections:c}));
 const legacy={...m,schemaVersion:1};delete legacy.connections;assert.throws(()=>validate(legacy));
 assert.throws(()=>validate({...m,schemaVersion:1}));
});
test('connection metadata rejects arbitrary origins, identities, versions and ambiguous policy',()=>{
 const {m}=fixture();m.connections=connectionFixture();
 const changes=[{infisical_origin:'https://evil.invalid'},{infisical_project:'00000000-0000-0000-0000-000000000000'},
 {infisical_identity:1},{link_events_qualified:'true'},
 {client_id_version:m.connections.client_secret_version},{client_id_version:m.connections.client_id_version.replace('123456789012','999999999999')},
 ...['latest','0','01','9223372036854775808'].map(v=>({client_secret_version:m.connections.client_secret_version.replace('/23','/'+v)})),
 {consent_expires_at:'2026-10-13T00:00:00Z'},{consent_expires_at:'2026-10-11T00:00:00Z'},
 {consent_expires_at:'2026-12-01T00:00:00Z'},{consent_expires_at:'2026-10-20T00:00:00+00:00'},
 {consent_expires_at:0},{consent_expires_at:'2026-02-30T00:00:00Z'}];
 for (const change of changes) assert.throws(()=>validate({...m,connections:{...m.connections,...change}}));
 assert.throws(()=>validate({...m,projectNumber:'0123456789012'}));
 assert.throws(()=>validate({...m,projectNumber:'9223372036854775808',connections:null}));
 validate({...m,connections:{...m.connections,client_id_version:m.connections.client_id_version.replace('/7','/9223372036854775807')}});
});
test('rollback binds exact approved connection policy and refuses legacy or different policy',()=>{
 const {m,run}=fixture();m.operation='rollback';m.previousDeployment='456';m.connections=connectionFixture();
 const prior={...run,id:456,path:'.github/workflows/deploy-dev.yml'};
 const receipt={status:'transport-smoke-passed',project:m.project,images:m.images,approvedConnectionConfigSHA256:connectionConfigDigest(m.connections)};
 validateRollback(m,prior,receipt);
 assert.throws(()=>validateRollback(m,prior,{...receipt,approvedConnectionConfigSHA256:undefined}));
 assert.throws(()=>validateRollback({...m,connections:null},prior,receipt));
 assert.throws(()=>validateRollback({...m,connections:{...m.connections,consent_expires_at:'2026-10-21T00:00:00Z'}},prior,receipt));
 assert.equal(connectionConfigDigest(m.connections),connectionConfigDigest(Object.fromEntries(Object.entries(m.connections).reverse())));
});
test('release CLI preserves enabled metadata and explicit null, including exact large version strings',()=>{
 for (const enabled of [false,true]) {
  const dir=mkdtempSync(join(tmpdir(),'jayburd-connection-release-'));
  try {
   const {m}=fixture();m.expiresAt=new Date(Date.now()+86400000).toISOString().replace(/\.\d{3}Z$/,'Z');
   if(enabled) {m.connections=connectionFixture();m.connections.consent_expires_at=new Date(Date.now()+8*86400000).toISOString().replace(/\.\d{3}Z$/,'Z');m.connections.client_id_version=m.connections.client_id_version.replace('/7','/9223372036854775807');}
   const bytes=JSON.stringify(m),path=join(dir,'manifest.json'),out=join(dir,'out');writeFileSync(path,bytes);
   execFileSync(process.execPath,['scripts/release-manifest.mjs',path,createHash('sha256').update(bytes).digest('hex'),out]);
   const runtime=JSON.parse(readFileSync(join(out,'runtime.tfvars.json')));
   assert.ok(Object.hasOwn(runtime,'connections'));assert.deepEqual(runtime.connections,m.connections);
   assert.equal(JSON.parse(readFileSync(join(out,'approved.json'))).schemaVersion,2);
  } finally {rmSync(dir,{recursive:true,force:true});}
 }
});

// Hosted Link release metadata does not assert ordinary SDK account enablement.
test('Hosted Link accepts explicit false SDK enablement',()=>{
 const {m}=fixture();m.connections={...connectionFixture(),link_events_qualified:false};validate(m);
});
