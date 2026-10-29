import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';
import {buildTarget,productionImageReceipt} from './production-image-build.mjs';
const now=Date.parse('2026-10-28T14:00:00Z');
const encode=v=>JSON.stringify(v)+'\n';
const hash=b=>createHash('sha256').update(b).digest('hex');
// All identifiers and hashes below are fictional and grant no cloud authority.
const target=()=>({schemaVersion:1,project:'fictional-prod-project',projectNumber:'123456789012',expiresAt:'2026-10-29T14:00:00Z',mandate:'https://github.com/justin-napolitano/jayburd-bookkeeping/issues/337#issuecomment-123'});
test('shared build target requires matched project pair and retains finite exact-byte authority',()=>{
 const t={...target(),project:'jayburd-bookkeeping-dev',projectNumber:'9088794002'};
 const bytes=encode(t);assert.deepEqual(buildTarget(bytes,hash(bytes),now),t);
 const sharedImage='us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod/worker@sha256:'+'d'.repeat(64);
 assert.equal(productionImageReceipt({...input,project:t.project},build,[sharedImage],scan).candidate.image,sharedImage);
 assert.throws(()=>productionImageReceipt({...input,project:t.project},build,[sharedImage.replace('/jayburd-prod/','/jayburd-dev/')],scan));
});
test('target binds exact reviewed bytes and finite nondevelopment metadata',()=>{
 const t=target(),bytes=encode(t);assert.deepEqual(buildTarget(bytes,hash(bytes),now),t);
 assert.throws(()=>buildTarget(bytes, '0'.repeat(64),now));
 for(const mutate of [t=>t.project='jayburd-bookkeeping-dev',t=>t.projectNumber='9088794002',t=>t.projectNumber='01',t=>t.expiresAt='2026-10-28T14:00:00Z',t=>t.expiresAt='2026-11-05T14:00:00Z',t=>t.expiresAt='2026-10-29T14:00:00+00:00',t=>t.mandate='https://example.test',t=>t.secret='synthetic-canary',t=>delete t.project]) {
  const bad=target();mutate(bad);assert.throws(()=>buildTarget(encode(bad),hash(encode(bad)),now),/^Error: Production image build rejected$/);
 }
 for(const bytes of ['null','[]','synthetic-canary',JSON.stringify(t,null,2),'x'.repeat(4097)]) assert.throws(()=>buildTarget(bytes,hash(bytes),now));
});
const input={project:'fictional-prod-project',component:'worker',commit:'a'.repeat(40),run:'123',attempt:'1',imageID:'sha256:'+'b'.repeat(64),targetHash:'c'.repeat(64)};
const build={'containerimage.config.digest':input.imageID,'buildx.build.provenance':{builder:{id:'fictional'}}};
const image='us-central1-docker.pkg.dev/fictional-prod-project/jayburd-prod/worker@sha256:'+'d'.repeat(64);
const scan={SchemaVersion:2,Results:[{Vulnerabilities:[]}]};
test('receipt binds one published tested image and records inert worker/webhook proof',()=>{
 const r=productionImageReceipt(input,build,[image],scan);
 assert.deepEqual(r.candidate,{schemaVersion:1,component:'worker',environment:'production',image,commit:'a'.repeat(40),workflowRun:'https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/123/attempts/1',status:'candidate-only',deployable:false,scan:{severity:'HIGH,CRITICAL',result:'passed'},runtimeTests:['worker','webhook']});
 assert.equal(r.binding.testedImageID,input.imageID);assert.equal(r.binding.targetSHA256,input.targetHash);
});
test('receipt rejects mismatched images absent provenance and failed scan evidence',()=>{
 for(const args of [
 [{...input,project:'jayburd-bookkeeping-dev'},build,[image],scan],
 [input,{...build,'containerimage.config.digest':'sha256:'+'e'.repeat(64)},[image],scan],
 [input,{'containerimage.config.digest':input.imageID},[image],scan],
 [input,build,[image,image],scan],[input,build,[image.replace('/worker@','/api@')],scan],
 [input,build,[image.replace('/jayburd-prod/','/jayburd-dev/')],scan],
 [input,build,[image],{SchemaVersion:2}],
 [input,build,[image],{SchemaVersion:2,Results:[{Vulnerabilities:[{Severity:'HIGH'}]}]}],
 [input,build,[image],{SchemaVersion:2,Results:[{Vulnerabilities:[{Severity:'CRITICAL'}]}]}],
 ]) assert.throws(()=>productionImageReceipt(...args),/^Error: Production image build rejected$/);
});
test('manual Production workflow retains pre-auth tests scan and independent federation',()=>{
 const w=readFileSync('.github/workflows/build-production-images.yml','utf8');
 assert.match(w,/workflow_dispatch:/);assert.doesNotMatch(w,/pull_request:|\n  push:|jayburd-dev|gcloud run|terraform apply/);
 assert.match(w,/github\.actor_id == '55423093'/);assert.match(w,/github\.repository_id == '1414515318'/);
 assert.match(w,/create_credentials_file: false/);assert.match(w,/jayburd-prod-github\/providers\/owner-build/);
 assert.match(w,/jayburd-prod-builder@/);assert.match(w,/component: \[api, migrate, web, worker\]/);
 assert.ok(w.indexOf('Prove exact candidate runtime')<w.indexOf('id: auth'));
 assert.ok(w.indexOf('Block high and critical')<w.indexOf('id: auth'));
 assert.ok(w.indexOf('Recheck target before cloud authentication')<w.indexOf('id: auth'));
 assert.match(w,/node scripts\/production-image-build.mjs receipt/);
 assert.equal((w.match(/production-image-build.mjs target/g)||[]).length,3);
});
