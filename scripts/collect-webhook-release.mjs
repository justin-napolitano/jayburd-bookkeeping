import {readFileSync,writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';
import {createHash} from 'node:crypto';
import {validateWebhookManifest,validateWebhookBuild,webhookProject} from './webhook-release-manifest.mjs';
import {validateWebhookDeployment,validateWebhookPolicy} from './webhook-deployment-receipt.mjs';

// Server-side partial responses deliberately omit environment values, tokens and provider diagnostics.
export const serviceFields='name,generation,observedGeneration,reconciling,terminalCondition(state),ingress,invokerIamDisabled,defaultUriDisabled,iapEnabled,sshEnabled,urls,latestReadyRevision,latestCreatedRevision,scaling,template(serviceAccount,timeout,maxInstanceRequestConcurrency,scaling,volumes(name),containers(image,command,args,env(name),volumeMounts(name),ports,resources,startupProbe)),trafficStatuses';
const serviceName=`projects/${webhookProject}/locations/us-central1/services/jayburd-dev-webhook`;
export const webhookReadURLs=[
  `https://run.googleapis.com/v2/${serviceName}?fields=${encodeURIComponent(serviceFields)}`,
  `https://run.googleapis.com/v2/${serviceName}:getIamPolicy?options.requestedPolicyVersion=3&fields=bindings(role,members,condition)`,
  `https://iam.googleapis.com/v1/projects/${webhookProject}/roles/jayburdWebhookRelease?fields=name,includedPermissions,deleted,stage`
];
// Fixed destinations and bounded HTTPS reads receive only an explicitly injected token; no ADC discovery.
export function cloudReader(token) {
  if(typeof token!=='string' || !token || token.length>8192 || /\s/.test(token)) throw new Error('Webhook collection rejected');
  return async(url)=>{
    if(!webhookReadURLs.includes(url))throw new Error('Webhook collection rejected');
    const r=await fetch(url,{headers:{Authorization:`Bearer ${token}`},redirect:'error',signal:AbortSignal.timeout(10000)});
    if(!r.ok || !r.body){await r.body?.cancel();throw new Error('Webhook collection rejected');}
    let size=0;const chunks=[];
    for await(const chunk of r.body){size+=chunk.length;if(size>131072)throw new Error('Webhook collection rejected');chunks.push(Buffer.from(chunk));}
    return JSON.parse(Buffer.concat(chunks).toString('utf8'));
  };
}
// collect validates in memory and writes no raw service, role, IAM or error response to disk.
export async function collect(m,read,{preflight=false,now}={}) {
  const bytes=JSON.stringify(m);
  validateWebhookManifest(bytes,createHash('sha256').update(bytes).digest('hex'),now??Date.now());
  const service=await read(webhookReadURLs[0]);
  const policy=await read(webhookReadURLs[1]);
  const role=await read(webhookReadURLs[2]);
  if(!validateWebhookPolicy(m,policy,role)) throw new Error('Webhook collection rejected');
  let target=m;
  if(preflight){
    const args=service?.template?.containers?.[0]?.args;
    const version=(name)=>{
      const prefix=`-plaid-client-${name}-version=projects/${m.projectNumber}/secrets/jayburd-dev-bank-sync-client-${name}/versions/`;
      const match=Array.isArray(args)?args.filter(a=>typeof a==='string'&&a.startsWith(prefix)):[];
      if(match.length!==1) throw new Error('Webhook collection rejected');
      return match[0].slice(prefix.length);
    };
    target={...m,image:service?.template?.containers?.[0]?.image,client_id_version:version('id'),client_secret_version:version('secret')};
  }
  const receipt=validateWebhookDeployment(target,service,policy,now??Date.now(),role);
  return preflight?{status:'existing-webhook-profile-observed'}:receipt;
}
// CLI rechecks approved bytes/provenance each time; credentials exist only in the explicit WIF token environment.
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){
  try{
    const [manifest,hash,evidence,mode,output]=process.argv.slice(2);
    if(process.argv.length!==7||!['preflight','receipt'].includes(mode))throw new Error();
    const m=validateWebhookManifest(readFileSync(manifest),hash);
    const json=file=>JSON.parse(readFileSync(`${evidence}/${file}`,'utf8'));
    validateWebhookBuild(m,json('build-run.json'),json('worker/candidate.json'));
    const result=await collect(m,cloudReader(process.env.CLOUDSDK_AUTH_ACCESS_TOKEN),{preflight:mode==='preflight'});
    if(mode==='receipt')writeFileSync(output,JSON.stringify(result,null,2)+'\n',{mode:0o600});
    console.log(mode==='preflight'?'PASS: existing webhook profile verified':'PASS: webhook control-plane receipt recorded');
  }catch{console.error('BLOCKED: webhook collection rejected');process.exitCode=1;}
}
