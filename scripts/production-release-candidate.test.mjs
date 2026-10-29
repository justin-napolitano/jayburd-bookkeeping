import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { validateProductionCandidate } from './production-release-candidate.mjs';

const now = Date.parse('2026-10-27T12:00:00Z');
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
// Fictional public metadata is independent of running resources and build receipts.
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
const encode = m => JSON.stringify(m) + '\n';
test('shared candidate retains separate database and Production resource profile',()=>{
 const m=JSON.parse(JSON.stringify(fixture()).replaceAll('fictional-pilot-prod','jayburd-bookkeeping-dev').replaceAll('123456789012','9088794002'));
 m.database.instance='jayburd-bookkeeping-dev:us-central1:jayburd-dev';m.database.name='bookkeeping_prod';
 const bytes=encode(m);assert.deepEqual(validateProductionCandidate(bytes,hash(bytes),now),m);
 for(const change of [m=>m.database.name='bookkeeping',m=>m.database.instance='jayburd-bookkeeping-dev:us-central1:jayburd-prod',m=>m.projectNumber='123456789012',m=>m.images.api=m.images.api.replace('/jayburd-prod/','/jayburd-dev/')]) {
  const bad=structuredClone(m);change(bad);const b=encode(bad);assert.throws(()=>validateProductionCandidate(b,hash(b),now));
 }
});
// Every negative is rehashed so it exercises semantics rather than only the digest gate.
function rejected(change) {
  const m = fixture(); change(m); const bytes = encode(m);
  assert.throws(()=>validateProductionCandidate(bytes,hash(bytes),now), /^Error: Production candidate rejected$/);
}
test('candidate remains offline and binds exact bytes',()=>{
  const m=fixture(), bytes=encode(m);
  assert.deepEqual(validateProductionCandidate(Buffer.from(bytes),hash(bytes),now),m);
  assert.equal(validateProductionCandidate(bytes,hash(bytes),now).deployable,false);
  assert.throws(()=>validateProductionCandidate(bytes+' ',hash(bytes),now));
  assert.throws(()=>validateProductionCandidate(bytes,'0'.repeat(64),now));
});
test('mixed environment targets and unpinned resources fail',()=>{
  for (const change of [
    m=>m.environment='sandbox', m=>m.project='jayburd-bookkeeping-dev',m=>m.projectNumber='9088794002',
    m=>m.project='other-project',m=>m.projectNumber='123456789013',m=>m.region='europe-west1',
    m=>m.database.instance='fictional-pilot-prod:us-central1:jayburd-dev',m=>m.database.migration='34',
    m=>m.keys.bank=m.keys.link,m=>m.keys.link=m.keys.link.replace('jayburd-prod','jayburd-dev'),
    m=>m.secrets.clientSecret=m.secrets.clientSecret.replace('/2','/latest'),
    m=>m.secrets.clientID=m.secrets.clientID.replace('/1','/01'),m=>m.secrets.clientID='secret-value-canary',
    m=>m.images.api=m.images.web,m=>m.images.worker=m.images.worker.replace('@sha256:',':latest-'),
    m=>m.images.migrate=m.images.migrate.replace('/jayburd-prod/','/jayburd-dev/'),
  ]) rejected(change);
});
test('expired future and excessive approval windows fail',()=>{
  for (const change of [m=>m.expiresAt='2026-10-27T12:00:00Z',m=>m.createdAt='2026-10-28T11:00:00Z',
    m=>m.expiresAt='2026-11-04T11:00:00Z',m=>m.createdAt='2026-02-30T11:00:00Z',m=>m.expiresAt='2026-10-28T11:00:00+00:00']) rejected(change);
  const m=fixture();m.expiresAt='2026-11-03T11:00:00Z';
  assert.doesNotThrow(()=>validateProductionCandidate(encode(m),hash(encode(m)),now));
  assert.throws(()=>validateProductionCandidate(encode(m),hash(encode(m)),NaN));
});
test('no activation rollback omitted proof or extra fields',()=>{
  for (const change of [m=>m.deployable=true,m=>m.scheduling='enabled',m=>m.operation='rollback',
    m=>m.evidence.qualification=null,m=>delete m.evidence.mandate,m=>m.evidence.secrets='https://example.test/proof',
    m=>m.source.extra=true,m=>m.database.extra=true,m=>m.secrets.extra=true,m=>m.keys.extra=true,
    m=>m.images.extra=true,m=>m.evidence.extra='d'.repeat(64),m=>m.extra=true,m=>m.source.run=123,
    m=>m.source.attempt='0',m=>m.source.commit='main',m=>m.schemaVersion=2]) rejected(change);
  for (const key of Object.keys(fixture())) rejected(m=>delete m[key]);
});
test('malformed and ambiguous JSON fails without echoing content',()=>{
  for (const bytes of ['secret-value-canary','null','[]','{}',JSON.stringify(fixture()),
    encode(fixture()).replace('"schemaVersion":1','"schemaVersion":2,"schemaVersion":1'),
    JSON.stringify(fixture(),null,2)+'\n','x'.repeat(32769)]) {
    assert.throws(()=>validateProductionCandidate(bytes,hash(bytes),now),/^Error: Production candidate rejected$/);
  }
  const result=spawnSync(process.execPath,['scripts/production-release-candidate.mjs','missing-secret-value-canary','0'.repeat(64)],{encoding:'utf8'});
  assert.equal(result.status,1); assert.equal(result.stdout,'');
  assert.equal(result.stderr,'Production candidate rejected\n');
});
