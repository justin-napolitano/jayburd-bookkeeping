import { readFileSync, writeFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { connectionConfigDigest } from './release-manifest.mjs';
// Anonymous probes test deployment transport only: they never obtain or synthesize a user/service credential.
export async function smoke(m, transport=fetch) {
  const web=`https://jayburd-dev-web-${m.projectNumber}.us-central1.run.app`;
  const api=`https://jayburd-dev-api-${m.projectNumber}.us-central1.run.app`;
  const check=async(url,status)=>{
    const response=await transport(url,{redirect:'manual',signal:AbortSignal.timeout(15000),credentials:'omit',cache:'no-store'});
    await response.body?.cancel();
    if(response.status!==status) throw new Error('Transport smoke failed');
  };
  await check(`${api}/v1/workspace`,403);
  await check(`${web}/v1/workspace`,401);
  await check(`${web}/api/auth-config`,200);
  await check(web,200);
  return {schemaVersion:1,status:'transport-smoke-passed',project:m.project,images:m.images,operation:m.operation,
    approvedConnectionConfigSHA256:connectionConfigDigest(m.connections),
    limitations:['No real MFA/entity/revocation proof.','Connection digest records approved policy, not provider or observed runtime qualification.','No atomic multi-service rollout or automatic rollback.']};
}
// Bind the receipt to the observed ready revision, immutable image and sole full-traffic target.
export function validateRevisions(m, observed) {
  const revisions={};
  for(const component of ['api','web']) {
    const view=observed[component];
    const revision=view?.status?.latestReadyRevisionName;
    const containers=view?.spec?.template?.spec?.containers;
    const traffic=view?.status?.traffic;
    if(typeof revision!=='string' || !new RegExp(`^jayburd-dev-${component}-[a-z0-9-]{1,45}$`).test(revision) ||
        !Array.isArray(containers) || containers.length!==1 || containers[0].image!==m.images[component] ||
        !Array.isArray(traffic) || traffic.length!==1 || traffic[0].revisionName!==revision || traffic[0].percent!==100) throw new Error('Revision mismatch');
    revisions[component]=revision;
  }
  return revisions;
}
if(process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href) {
  try {
    const m=JSON.parse(readFileSync(process.argv[2],'utf8'));
    const receipt=await smoke(m);
    const revisions=validateRevisions(m,Object.fromEntries(['api','web'].map(component=>
      [component,JSON.parse(readFileSync(`${process.argv[4]}/${component}-revision.json`,'utf8'))])));
    receipt.revisions=revisions;
    receipt.build=m.build;
    receipt.evidence=m.evidence;
    writeFileSync(process.argv[3],JSON.stringify(receipt,null,2)+'\n');
    console.log('PASS: anonymous transport smoke; identity acceptance remains separate');
  } catch { console.error('BLOCKED: transport smoke failed; review revisions and compatible rollback'); process.exitCode=1; }
}
