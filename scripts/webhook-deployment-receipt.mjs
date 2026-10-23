import {readFileSync,writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';
import {createHash} from 'node:crypto';
import {validateWebhookManifest,webhookConfigDigest,webhookURL} from './webhook-release-manifest.mjs';

// Require one complete deterministic argument list: no hidden flag, URL or identity overrides.
export function webhookArguments(m) {
  return ['-listen=0.0.0.0:8080',`-cloud-sql-instance=${m.project}:us-central1:jayburd-dev`,
    `-project-number=${m.projectNumber}`,`-plaid-client-id-version=projects/${m.projectNumber}/secrets/jayburd-dev-bank-sync-client-id/versions/${m.client_id_version}`,
    `-plaid-client-secret-version=projects/${m.projectNumber}/secrets/jayburd-dev-bank-sync-client-secret/versions/${m.client_secret_version}`];
}
// Accept equivalent argument order but never duplicates or additional arguments.
function sameList(a,b) {return Array.isArray(a) && a.every(v=>typeof v==='string') && JSON.stringify([...a].sort())===JSON.stringify([...b].sort());}
// ProtoJSON may omit implicit scalar defaults; null and wrong types remain invalid.
function falseDefault(v) {return v===undefined || v===false;}
function zeroDefault(v) {return v===undefined || v===0;}
export const webhookUpdaterPermissions=['run.services.get','run.services.getIamPolicy','run.services.update'];
// Service IAM may contain one exact updater only after its actual custom-role definition is verified.
export function validateWebhookPolicy(m,policy,role) {
  if(!Array.isArray(policy?.bindings) || policy.bindings.length<1 || policy.bindings.length>2) throw new Error('Webhook readback rejected');
  const invoker=policy.bindings.filter(b=>b.role==='roles/run.invoker');
  if(invoker.length!==1 || !sameList(invoker[0].members,['allUsers']) || invoker[0].condition!==undefined) throw new Error('Webhook readback rejected');
  const updater=policy.bindings.filter(b=>b.role!=='roles/run.invoker');
  if(updater.length===0) return false;
  const name=`projects/${m.project}/roles/jayburdWebhookRelease`;
  if(updater[0].role!==name || updater[0].condition!==undefined || !sameList(updater[0].members,[`serviceAccount:jayburd-dev-webhook-deployer@${m.project}.iam.gserviceaccount.com`]) ||
    role?.name!==name || role?.stage!=='GA' || !falseDefault(role?.deleted) || !sameList(role?.includedPermissions,webhookUpdaterPermissions)) throw new Error('Webhook readback rejected');
  return true;
}
// Selected public v2 control-plane fields are sufficient for transport configuration only.
export function validateWebhookDeployment(m,service,policy,now=Date.now(),role) {
  const bytes=JSON.stringify(m);
  validateWebhookManifest(bytes,createHash('sha256').update(bytes).digest('hex'),now);
  const name=`projects/${m.project}/locations/us-central1/services/jayburd-dev-webhook`;
  const ready=service?.latestReadyRevision,created=service?.latestCreatedRevision;
  const generation=v=>typeof v==='string' && /^[1-9][0-9]{0,18}$/.test(v) && BigInt(v)<=9223372036854775807n;
  const template=service?.template,containers=template?.containers,c=containers?.[0];
  if(service?.name!==name || !generation(service?.generation) || service.generation!==service?.observedGeneration || !falseDefault(service?.reconciling) ||
    service?.terminalCondition?.state!=='CONDITION_SUCCEEDED' || service?.ingress!=='INGRESS_TRAFFIC_ALL' || !falseDefault(service?.invokerIamDisabled) ||
    !falseDefault(service?.defaultUriDisabled) || !falseDefault(service?.iapEnabled) || !falseDefault(service?.sshEnabled) ||
    !Array.isArray(service?.urls) || !service.urls.includes(webhookURL.replace('/webhooks/plaid','')) ||
    typeof ready!=='string' || !new RegExp(`^${name}/revisions/jayburd-dev-webhook-[a-z0-9-]{1,45}$`).test(ready) || ready!==created ||
    template?.serviceAccount!==`jayburd-dev-webhook@${m.project}.iam.gserviceaccount.com` || template?.timeout!=='10s' || template?.maxInstanceRequestConcurrency!==16 ||
    !zeroDefault(service?.scaling?.minInstanceCount) || service?.scaling?.maxInstanceCount!==1 || !zeroDefault(template?.scaling?.minInstanceCount) || template?.scaling?.maxInstanceCount!==1 ||
    (service?.scaling?.scalingMode!==undefined && service.scaling.scalingMode!=='AUTOMATIC') || !zeroDefault(service?.scaling?.manualInstanceCount) ||
    !Array.isArray(containers) || containers.length!==1 || c?.image!==m.image || !sameList(c?.command,['/webhook']) || !sameList(c?.args,webhookArguments(m)) ||
    (c?.env!==undefined && (!Array.isArray(c.env) || c.env.length!==0)) || (c?.volumeMounts!==undefined && (!Array.isArray(c.volumeMounts) || c.volumeMounts.length!==0)) ||
    (template?.volumes!==undefined && (!Array.isArray(template.volumes) || template.volumes.length!==0)) ||
    !Array.isArray(c?.ports) || c.ports.length!==1 || c.ports[0]?.containerPort!==8080 || (c.ports[0]?.name!==undefined && c.ports[0].name!=='http1') ||
    c?.resources?.limits?.cpu!=='1' || c?.resources?.limits?.memory!=='512Mi' || c?.resources?.cpuIdle!==true || !falseDefault(c?.resources?.startupCpuBoost) ||
    c?.startupProbe?.tcpSocket?.port!==8080 || c?.startupProbe?.periodSeconds!==5 || c?.startupProbe?.timeoutSeconds!==3 || c?.startupProbe?.failureThreshold!==12 ||
    c?.startupProbe?.httpGet!==undefined || c?.startupProbe?.grpc!==undefined ||
    !Array.isArray(service?.trafficStatuses) || service.trafficStatuses.length!==1 || service.trafficStatuses[0]?.revision!==ready || service.trafficStatuses[0]?.percent!==100 ||
    service.trafficStatuses[0]?.type!=='TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST' || (service.trafficStatuses[0]?.tag??'')!=='') throw new Error('Webhook readback rejected');
  const updaterVerified=validateWebhookPolicy(m,policy,role);
  return {schemaVersion:1,status:'webhook-control-plane-observed',project:m.project,build:m.build,image:m.image,
    revision:ready,approvedWebhookConfigSHA256:webhookConfigDigest(m),url:webhookURL,
    updaterRoleVerified:updaterVerified,
    ...(updaterVerified?{updaterRole:{name:role.name,stage:role.stage,deleted:false,includedPermissions:[...role.includedPermissions].sort()}}:{}),
    limitations:['Selected control-plane metadata only; no database mandate, secret propagation or actual signed delivery proof.',
      'Approval expiry does not stop a service; owner must revoke the qualification mandate and complete reviewed cleanup.',
      'No financial equality, absence-of-journals or Production qualification.']};
}
// Only explicitly supplied selected metadata becomes a public receipt; upstream diagnostics stay private.
if(process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href) {
  try {
    const [approved,service,policy,output,role]=process.argv.slice(2);
    if(process.argv.length!==6 && process.argv.length!==7) throw new Error();
    const json=file=>JSON.parse(readFileSync(file,'utf8'));
    writeFileSync(output,JSON.stringify(validateWebhookDeployment(json(approved),json(service),json(policy),Date.now(),role?json(role):undefined),null,2)+'\n',{mode:0o600});
    console.log('PASS: webhook control-plane readback; hosted qualification remains separate');
  } catch {console.error('BLOCKED: webhook control-plane readback rejected');process.exitCode=1;}
}
