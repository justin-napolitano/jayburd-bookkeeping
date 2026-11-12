import {readFileSync,writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';
import {validateAutomaticManifest,validateService} from './automatic-synthetic-release.mjs';
import {validateRevisions} from './deployment-smoke.mjs';

const failure=()=>{throw new Error('Anonymous synthetic smoke blocked');};
const canonical=v=>Array.isArray(v)?v.map(canonical):v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])])):v;
const equal=(a,b)=>JSON.stringify(canonical(a))===JSON.stringify(canonical(b));
const web='https://jayburd-dev-web-9088794002.us-central1.run.app';
const api='https://jayburd-dev-api-9088794002.us-central1.run.app';
const targets=[['web-root',web+'/',200],['web-auth-config',web+'/api/auth-config',200],['web-private-collection',web+'/v1/collection',401],['web-private-automation',web+'/v1/collections/11111111-1111-4111-8111-111111111111/automation',401],['direct-api-readiness',api+'/readyz',403]];
// Transport qualification is bound to accepted selected metadata, never to a desired image or an offline-only receipt.
function binding(m,r){
 if(!equal(m?.build,{run:'43158591303',attempt:'1',commit:'eb1828caab584064aa076edefccbb2bb413b84a7'})||!/^([a-f0-9]{64})$/.test(r?.manifestSHA256)||m?.project!=='jayburd-bookkeeping-dev'||m.projectNumber!=='9088794002'||!equal(r?.manifest,m)||r.schemaVersion!==1)failure();
 let observed;
 if(r.phase==='inspect'&&r.status==='inspection-only-not-release-acceptance')observed=r.before;
 else if(r.phase==='web'&&r.status==='runtime-promoted-not-product-acceptance'&&equal(r.build,m.build)&&equal(r.images,m.images))observed={api:r.before?.api,web:r.after};
 else failure();
 for(const c of ['api','web']){const s=observed?.[c];validateService(s,c);if(s.status.latestReadyRevisionName!==s.status.latestCreatedRevisionName||s.spec.traffic[0].revisionName!==s.status.latestReadyRevisionName||s.status.traffic[0].revisionName!==s.status.latestReadyRevisionName)failure();}
 return validateRevisions(m,observed);
}
// Only five fixed anonymous GETs are allowed. Bodies are cancelled without reading, including on unexpected status.
export async function smoke(m,phaseReceipt,transport=fetch){
 try{
  const revisions=binding(m,phaseReceipt),checks=[];
  for(const [target,url,expected] of targets){
   const controller=new AbortController();let timer;
   const deadline=new Promise((_,reject)=>{timer=setTimeout(()=>{controller.abort();reject(new Error('timeout'));},15000);});
   try{
    const response=await Promise.race([transport(url,{method:'GET',headers:{},credentials:'omit',redirect:'manual',cache:'no-store',signal:controller.signal}),deadline]);
    const observed=response.status;
    await Promise.race([Promise.resolve(response.body?.cancel()),deadline]);
    if(observed!==expected)failure();checks.push({target,expected,observed});
   }finally{clearTimeout(timer);}
  }
  return {schemaVersion:1,status:'anonymous-transport-smoke-passed',project:m.project,build:m.build,images:{api:m.images.api,web:m.images.web},sourcePhase:phaseReceipt.phase,revisions,checks,limitations:['Anonymous transport only; no verified-owner or bookkeeping acceptance.','No authenticated requests, provider calls, workspace-policy changes or accounting writes.']};
 }catch{failure();}
}
// Explicit operator/workflow invocation only. No environment, credential store, response body or provider error is read or emitted.
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){
 try{
  if(process.argv.length!==6)failure();const [manifestPath,digest,receiptPath,out]=process.argv.slice(2);
  const m=validateAutomaticManifest(readFileSync(manifestPath),digest,'inspect'),phaseReceipt=JSON.parse(readFileSync(receiptPath,'utf8'));
  if(phaseReceipt.manifestSHA256!==digest)failure();
  const result=await smoke(m,phaseReceipt);writeFileSync(out,JSON.stringify({...result,manifestSHA256:digest},null,2)+'\n',{mode:0o600});
  console.log('PASS: fixed anonymous transport statuses; owner acceptance remains separate');
 }catch{console.error('BLOCKED: anonymous transport smoke; no private response data retained');process.exitCode=1;}
}
