import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {validatePlatform} from './verifier-platform.mjs';
const base={project_id:'jayburd-bookkeeping-dev',region:'us-central1',enabled:true,run:null};
const live={...base,run:{image:'us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/worker@sha256:'+'a'.repeat(64),run_id:'verify-synthetic123'}};
const bytes=m=>Buffer.from(JSON.stringify(m,null,2)+'\n');
const hash=b=>createHash('sha256').update(b).digest('hex');
test('bootstrap and bounded runtime metadata validate without cloud work',()=>{
 for(const m of [base,live]){const b=bytes(m);assert.deepEqual(validatePlatform(b,hash(b)),m);}
});
test('canonical exact-byte review rejects duplicate keys and hash substitution',()=>{
 const b=bytes(live);assert.throws(()=>validatePlatform(b,'f'.repeat(64)));
 const duplicate=Buffer.from(b.toString().replace('"enabled": true','"enabled": false,\n  "enabled": true'));assert.throws(()=>validatePlatform(duplicate,hash(duplicate)));
});
test('foreign scope, destruction, secrets, mutable image and traversal fail before output',()=>{
 for(const m of [{...base,project_id:'other-project'},{...base,region:'europe-west1'},{...base,enabled:false},{...base,secret:'synthetic'},{...live,run:{...live.run,image:'https://untrusted.example/worker:latest'}},{...live,run:{...live.run,run_id:'verify-../escape'}},{...live,run:{...live.run,env:{secret:'synthetic'}}},{...base,run:[]}]){const b=bytes(m);assert.throws(()=>validatePlatform(b,hash(b)));}
});
