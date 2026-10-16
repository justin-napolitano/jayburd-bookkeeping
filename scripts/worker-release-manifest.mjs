import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { isSandbox, isRollback } from './release-manifest.mjs';
import { workerConfigDigest } from './worker-deployment-receipt.mjs';

// A separate strict manifest cannot select application rollout, SQL, destinations or scheduler activation.
function keys(value, names) {
  if (!value || typeof value !== 'object' || Array.isArray(value) || Object.keys(value).sort().join(',') !== [...names].sort().join(',')) throw new Error('Worker release rejected');
}
// Bound canonical integers before using them in GitHub paths, Terraform or resource names.
function integer(value, max=9223372036854775807n) {
  return typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) && BigInt(value) <= max;
}
// Approval covers exact checked-in public bytes and expires; secret values are never accepted.
export function validateWorkerManifest(bytes, hash, now=Date.now()) {
  if (!/^[a-f0-9]{64}$/.test(hash) || createHash('sha256').update(bytes).digest('hex') !== hash) throw new Error('Worker release rejected');
  const m=JSON.parse(bytes);
  keys(m,['schemaVersion','project','projectNumber','operation','build','image','binding_id','client_id_version','client_secret_version','evidence','expiresAt','previousDeployment']);
  keys(m.build,['run','attempt','commit']);
  keys(m.evidence,['bootstrap','compatibility','identity','secrets']);
  if (m.schemaVersion!==1 || typeof m.project!=='string' || !/^[a-z][a-z0-9-]{4,28}[a-z0-9]$/.test(m.project) ||
      !integer(m.projectNumber) || m.projectNumber.length<6 || !['promote','rollback','sandbox-promote','sandbox-rollback'].includes(m.operation) ||
      !integer(m.build.run) || !integer(m.build.attempt,9999n) || typeof m.build.commit!=='string' || !/^[a-f0-9]{40}$/.test(m.build.commit)) throw new Error('Worker release rejected');
  const prefix=`us-central1-docker.pkg.dev/${m.project}/jayburd-dev/worker@sha256:`;
  if (typeof m.image!=='string' || !m.image.startsWith(prefix) || !/^[a-f0-9]{64}$/.test(m.image.slice(prefix.length)) ||
      typeof m.binding_id!=='string' || !/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/.test(m.binding_id) ||
      m.binding_id==='00000000-0000-0000-0000-000000000000' || !integer(m.client_id_version) || !integer(m.client_secret_version)) throw new Error('Worker release rejected');
  const expiry=Date.parse(m.expiresAt);
  if (typeof m.expiresAt!=='string' || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(m.expiresAt) || !Number.isFinite(expiry) ||
      new Date(expiry).toISOString().replace('.000Z','Z')!==m.expiresAt || expiry<=now || expiry>now+7*86400000) throw new Error('Worker release rejected');
  for (const value of Object.values(m.evidence)) if (typeof value!=='string' || !/^https:\/\/github\.com\/justin-napolitano\/jayburd-bookkeeping\/issues\/[1-9][0-9]*#issuecomment-[0-9]+$/.test(value)) throw new Error('Worker release rejected');
  if (isRollback(m.operation) ? !integer(m.previousDeployment) : m.previousDeployment!==null) throw new Error('Worker release rejected');
  if (isSandbox(m.operation) && (m.project!=='jayburd-bookkeeping-dev' || m.projectNumber!=='9088794002')) throw new Error('Worker release rejected');
  return m;
}
// Only successful owner-dispatched main builds from the canonical repository qualify.
function trustedRun(run,path) {
  return run?.path===path && run.conclusion==='success' && run.event==='workflow_dispatch' && run.head_branch==='main' &&
    String(run.actor?.id)==='55423093' && run.triggering_actor?.login==='justin-napolitano' && String(run.repository?.id)==='1414515318';
}
// Worker artifacts come from the same complete build workflow, but never authorize application or migration deployment.
export function validateWorkerBuild(m,run,receipt) {
  if (!trustedRun(run,'.github/workflows/build-images.yml') || String(run.id)!==m.build.run || String(run.run_attempt)!==m.build.attempt || run.head_sha!==m.build.commit ||
      receipt?.schemaVersion!==1 || receipt.component!=='worker' || receipt.image!==m.image || receipt.commit!==m.build.commit ||
      receipt.workflowRun!==`https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/${m.build.run}/attempts/${m.build.attempt}` ||
      receipt.status!=='candidate-only' || receipt.deployable!==false) throw new Error('Worker build rejected');
}
// Rollback keeps binding and secret-version policy identical; changed policy requires a forward promotion.
export function validateWorkerRollback(m,run,receipt) {
  if (!isRollback(m.operation) || !trustedRun(run,'.github/workflows/deploy-workers.yml') || String(run.id)!==m.previousDeployment ||
      isSandbox(receipt?.operation)!==isSandbox(m.operation) || receipt?.status!=='paused-workers-observed' || receipt.project!==m.project || receipt.image!==m.image ||
      receipt.approvedWorkerConfigSHA256!==workerConfigDigest(m)) throw new Error('Worker rollback rejected');
}
// Existing infrastructure/identity/secret closure gates remain independent of an operator evidence URL.
export const workerGates={issues:[5,6,11,26,162,163],prs:[109,110,111,165]};
// Qualification deploys only paused workers; hosted activation remains a separate acceptance.
export function requiredWorkerGates(operation) {
  if (isSandbox(operation)) return {issues:[5,6,130,131,162,163],prs:[109,110,111,129,165]};
  if (['promote','rollback'].includes(operation)) return workerGates;
  throw new Error('Worker release rejected');
}

if (process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href) {
  try {
    const [manifest,hash,destination,evidence]=process.argv.slice(2);
    const m=validateWorkerManifest(readFileSync(manifest),hash);
    if (evidence) {
      const json=file=>JSON.parse(readFileSync(`${evidence}/${file}`,'utf8'));
      validateWorkerBuild(m,json('build-run.json'),json('worker/candidate.json'));
      if(isRollback(m.operation)) validateWorkerRollback(m,json('previous-run.json'),json('previous/worker-deployment.json'));
    }
    mkdirSync(destination,{recursive:true,mode:0o700});
    writeFileSync(`${destination}/worker-runtime.tfvars.json`,JSON.stringify({project_id:m.project,project_number:m.projectNumber,region:'us-central1',runtime:{image:m.image,binding_id:m.binding_id,client_id_version:m.client_id_version,client_secret_version:m.client_secret_version}}),{mode:0o600});
    writeFileSync(`${destination}/approved.json`,JSON.stringify(m),{mode:0o600});
    console.log('PASS: paused worker release metadata validated');
  } catch {console.error('BLOCKED: worker release metadata or evidence rejected');process.exitCode=1;}
}
