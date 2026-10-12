import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {validateManifest,validateBuild,validateRollback,requiredGates} from './release-manifest.mjs';
const now=Date.parse('2026-10-12T00:00:00Z');
// Independent synthetic release/build evidence; no real identities or provider clients.
function fixture() {
 const m={schemaVersion:1,project:'jayburd-synthetic-test',projectNumber:'123456789012',operation:'promote',
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
 const receipt={status:'transport-smoke-passed',project:m.project,images:m.images};validateRollback(m,prior,receipt);
 assert.throws(()=>validateRollback(m,{...prior,conclusion:'failure'},receipt));
 assert.throws(()=>validateRollback(m,prior,{...receipt,images:{...m.images,api:'other'}}));
});
test('migration-only release omits runtime configuration but still requires complete owner build',()=>{
 const {m,run,receipts}=fixture();m.operation='migrate';delete m.firebaseApiKey;delete m.previousDeployment;
 delete m.evidence.identity;delete m.evidence.secrets;
 assert.deepEqual(validate(m),m);validateBuild(m,run,receipts);
 assert.throws(()=>validateBuild(m,run,receipts.slice(1)));
 assert.throws(()=>validate({...m,firebaseApiKey:'unexpected-runtime-setting'}));
 assert.throws(()=>validate({...m,previousDeployment:null}));
 for(const operation of ['promote','rollback']) assert.throws(()=>validate({...m,operation}));
 assert.throws(()=>validate({...m,evidence:{compatibility:m.evidence.compatibility}}));
});
test('migration prerequisites cannot relax full application closure gates',()=>{
 assert.deepEqual(requiredGates('migrate'),{issues:[5],prs:[44]});
 for(const operation of ['promote','rollback']) assert.deepEqual(requiredGates(operation),{issues:[6,11,26],prs:[23,24]});
 assert.throws(()=>requiredGates('force'));
});
