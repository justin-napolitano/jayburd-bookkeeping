import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';

// Reject precision-losing JSON numbers while allowing Google's string-encoded generations.
function generation(value) {
  const text=typeof value==='number' && Number.isSafeInteger(value) ? String(value) : value;
  if(typeof text!=='string' || !/^[1-9][0-9]{0,18}$/.test(text) || BigInt(text)>9223372036854775807n) throw new Error('Worker readback rejected');
  return text;
}
// Stable approved-policy digest excludes images (checked independently), credentials and observed claims.
export function workerConfigDigest(m) {
  const keys=['projectNumber','binding_id','client_id_version','client_secret_version'];
  if(!m || keys.some(k=>typeof m[k]!=='string')) throw new Error('Worker readback rejected');
  return createHash('sha256').update(JSON.stringify({projectNumber:m.projectNumber,binding_id:m.binding_id,
    client_id_version:m.client_id_version,client_secret_version:m.client_secret_version,scheduler:'PAUSED'})).digest('hex');
}
// Selected control-plane metadata proves reconciled images/traffic and paused scheduling, never provider readiness or IAM.
export function validateWorkerDeployment(m,observed) {
  if(!m || typeof m.project!=='string' || !/^[a-z][a-z0-9-]{4,28}[a-z0-9]$/.test(m.project) ||
    typeof m.projectNumber!=='string' || generation(m.projectNumber)!==m.projectNumber ||
    typeof m.image!=='string' || !m.image.startsWith(`us-central1-docker.pkg.dev/${m.project}/jayburd-dev/worker@sha256:`) ||
    !/^sha256:[a-f0-9]{64}$/.test(m.image.slice(`us-central1-docker.pkg.dev/${m.project}/jayburd-dev/worker@`.length))) throw new Error('Worker readback rejected');
  const revisions={};
  for(const [component,suffix,account] of [['publisher','publisher','publisher'],['consumer','sync','sync']]) {
    const service=observed?.[component];
    const name=`jayburd-dev-acquisition-${suffix}`;
    const status=service?.status;
    const ready=status?.latestReadyRevisionName;
    const containers=service?.spec?.template?.spec?.containers;
    const conditions=status?.conditions;
    const traffic=status?.traffic;
    if(service?.metadata?.name!==name || generation(service?.metadata?.generation)!==generation(status?.observedGeneration) ||
      typeof ready!=='string' || !new RegExp(`^${name}-[a-z0-9-]{1,45}$`).test(ready) || ready!==status?.latestCreatedRevisionName ||
      !Array.isArray(conditions) || conditions.filter(c=>c?.type==='Ready').length!==1 || conditions.find(c=>c?.type==='Ready')?.status!=='True' ||
      service?.spec?.template?.spec?.serviceAccountName!==`jayburd-dev-${account}@${m.project}.iam.gserviceaccount.com` ||
      !Array.isArray(containers) || containers.length!==1 || containers[0]?.image!==m.image ||
      !Array.isArray(traffic) || traffic.length!==1 || traffic[0]?.revisionName!==ready || traffic[0]?.percent!==100 ||
      (traffic[0].tag!==undefined && traffic[0].tag!=='') || (traffic[0].url!==undefined && traffic[0].url!=='')) throw new Error('Worker readback rejected');
    // A pinned template revision name, when supplied, must describe this reconciled revision.
    const desired=service?.spec?.template?.metadata?.name;
    if(desired!==undefined && desired!=='' && desired!==ready) throw new Error('Worker readback rejected');
    revisions[component]=ready;
  }
  const scheduler=observed?.scheduler;
  const origin=`https://jayburd-dev-acquisition-publisher-${m.projectNumber}.us-central1.run.app`;
  if(scheduler?.name!==`projects/${m.project}/locations/us-central1/jobs/jayburd-dev-acquisition-publisher` || scheduler?.state!=='PAUSED' ||
    scheduler?.httpTarget?.uri!==`${origin}/internal/acquisition/publish` || scheduler?.httpTarget?.httpMethod!=='POST' ||
    scheduler?.httpTarget?.oidcToken?.audience!==origin ||
    scheduler?.httpTarget?.oidcToken?.serviceAccountEmail!==`jayburd-dev-scheduler@${m.project}.iam.gserviceaccount.com` ||
    scheduler?.httpTarget?.oauthToken!==undefined || scheduler?.pubsubTarget!==undefined || scheduler?.appEngineHttpTarget!==undefined) throw new Error('Worker readback rejected');
  return {schemaVersion:1,status:'paused-workers-observed',project:m.project,projectNumber:m.projectNumber,
    build:m.build,image:m.image,operation:m.operation,revisions,scheduler:'PAUSED',approvedWorkerConfigSHA256:workerConfigDigest(m),
    limitations:['Selected control-plane metadata only; IAM policy and workload OIDC remain separately qualified.',
      'Paused scheduling does not cancel existing tasks or prove absence of provider requests.',
      'No provider, SQL mandate, secret propagation or bookkeeping qualification.']};
}
// Read only approved public metadata and selected gcloud projections; emit fixed errors without upstream content.
if(process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href) {
  try {
    const [approved,directory,output]=process.argv.slice(2);
    if(process.argv.length!==5) throw new Error('Worker readback rejected');
    const json=path=>JSON.parse(readFileSync(path,'utf8'));
    const observed={publisher:json(`${directory}/publisher-revision.json`),consumer:json(`${directory}/consumer-revision.json`),scheduler:json(`${directory}/scheduler.json`)};
    const receipt=validateWorkerDeployment(json(approved),observed);
    writeFileSync(output,JSON.stringify(receipt,null,2)+'\n',{mode:0o600});
    console.log('PASS: paused worker control-plane readback; provider qualification remains separate');
  } catch { console.error('BLOCKED: worker control-plane readback rejected'); process.exitCode=1; }
}
