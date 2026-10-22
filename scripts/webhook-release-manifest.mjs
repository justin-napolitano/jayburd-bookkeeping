import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,mkdirSync} from 'node:fs';
import {pathToFileURL} from 'node:url';
import {validateWorkerBuild} from './worker-release-manifest.mjs';

export const webhookProject='jayburd-bookkeeping-dev';
export const webhookProjectNumber='9088794002';
export const webhookURL=`https://jayburd-dev-webhook-${webhookProjectNumber}.us-central1.run.app/webhooks/plaid`;
// Exact keys prevent destinations, plaintext credentials and unrelated Terraform inputs.
function keys(value,names) {
  if(!value || typeof value!=='object' || Array.isArray(value) || Object.keys(value).sort().join(',')!==[...names].sort().join(',')) throw new Error('Webhook release rejected');
}
// Canonical strings preserve integer precision through public release metadata.
function integer(v,max=9223372036854775807n) {
  return typeof v==='string' && /^[1-9][0-9]{0,18}$/.test(v) && BigInt(v)<=max;
}
// Round-trip UTC prevents normalized invalid dates from silently extending approval.
function timestamp(v) {
  const t=typeof v==='string' && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(v)?Date.parse(v):NaN;
  if(!Number.isFinite(t) || new Date(t).toISOString().replace('.000Z','Z')!==v) throw new Error('Webhook release rejected');
  return t;
}
// Exact-byte owner approval covers only this fixed Sandbox service and finite interval.
export function validateWebhookManifest(bytes,hash,now=Date.now()) {
  if(typeof hash!=='string' || !/^[a-f0-9]{64}$/.test(hash) || createHash('sha256').update(bytes).digest('hex')!==hash) throw new Error('Webhook release rejected');
  const m=JSON.parse(bytes);
  keys(m,['schemaVersion','project','projectNumber','operation','build','image','client_id_version','client_secret_version','evidence','expiresAt','qualificationEndsAt']);
  keys(m.build,['run','attempt','commit']);
  keys(m.evidence,['security','bootstrap','identity','secrets','scope','compatibility']);
  if(m.schemaVersion!==1 || m.project!==webhookProject || m.projectNumber!==webhookProjectNumber || m.operation!=='sandbox-promote' ||
    !integer(m.build.run) || !integer(m.build.attempt,9999n) || typeof m.build.commit!=='string' || !/^[a-f0-9]{40}$/.test(m.build.commit) ||
    typeof m.image!=='string' || !new RegExp(`^us-central1-docker[.]pkg[.]dev/${webhookProject}/jayburd-dev/worker@sha256:[a-f0-9]{64}$`).test(m.image) ||
    !integer(m.client_id_version) || !integer(m.client_secret_version)) throw new Error('Webhook release rejected');
  const expiry=timestamp(m.expiresAt),end=timestamp(m.qualificationEndsAt);
  if(!Number.isFinite(now) || expiry<=now || expiry>now+86400000 || end<=expiry || end>now+2*86400000) throw new Error('Webhook release rejected');
  for(const value of Object.values(m.evidence)) if(typeof value!=='string' || !/^https:\/\/github\.com\/justin-napolitano\/jayburd-bookkeeping\/issues\/[1-9][0-9]*#issuecomment-[0-9]+$/.test(value)) throw new Error('Webhook release rejected');
  return m;
}
// Reuse same-source trusted worker provenance; candidate-only is never deployment approval.
export function validateWebhookBuild(m,run,receipt) { validateWorkerBuild(m,run,receipt); }
// Bind approved runtime policy independently from images and observed outcomes.
export function webhookConfigDigest(m) {
  return createHash('sha256').update(JSON.stringify({project:m.project,projectNumber:m.projectNumber,
    client_id_version:m.client_id_version,client_secret_version:m.client_secret_version,
    qualificationEndsAt:m.qualificationEndsAt,url:webhookURL})).digest('hex');
}
// Offline preparation reads only explicit public evidence; no cloud SDK or environment credentials.
if(process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href) {
  try {
    const [manifest,hash,destination,evidence]=process.argv.slice(2);
    if(process.argv.length!==6) throw new Error();
    const m=validateWebhookManifest(readFileSync(manifest),hash);
    const json=file=>JSON.parse(readFileSync(`${evidence}/${file}`,'utf8'));
    validateWebhookBuild(m,json('build-run.json'),json('worker/candidate.json'));
    mkdirSync(destination,{recursive:true,mode:0o700});
    writeFileSync(`${destination}/webhook-runtime.tfvars.json`,JSON.stringify({project_id:m.project,project_number:m.projectNumber,region:'us-central1',runtime:{image:m.image,client_id_version:m.client_id_version,client_secret_version:m.client_secret_version}}),{mode:0o600});
    writeFileSync(`${destination}/approved.json`,JSON.stringify(m),{mode:0o600});
    console.log('PASS: webhook metadata and candidate validated; no deployment performed');
  } catch {console.error('BLOCKED: webhook release or build evidence rejected');process.exitCode=1;}
}
