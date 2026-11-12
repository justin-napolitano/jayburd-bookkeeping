import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,mkdirSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {pathToFileURL} from 'node:url';
import {validateBuild} from './release-manifest.mjs';
import {receipt as imageReceipt} from './image-receipt.mjs';
import {migrationReceipt} from './migration-receipt.mjs';

const project='jayburd-bookkeeping-dev', number='9088794002', region='us-central1';
const source={run:'43158591303',attempt:'1',commit:'eb1828caab584064aa076edefccbb2bb413b84a7'};
const components=['api','migrate','web','worker'];
const phases=['inspect','migrate','stage','activate','web','pause'];
const fail=()=>{throw new Error('Automatic synthetic release blocked');};
const canonical=v=>Array.isArray(v)?v.map(canonical):v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])])):v;
const equal=(a,b)=>JSON.stringify(canonical(a))===JSON.stringify(canonical(b));
const hash=b=>createHash('sha256').update(b).digest('hex');
function keys(v,k){if(!v||typeof v!=='object'||Array.isArray(v)||!equal(Object.keys(v).sort(),[...k].sort()))fail();}
const old={api:{revision:'jayburd-dev-api-00026-5nt',digest:'9747304373c1c9923518b0675cb85a2fcdde1ed1f397244df59208683fc790e0'},web:{revision:'jayburd-dev-web-00027-6jn',digest:'e4521574edecc6a5901c7f590341112484370e6cf93d63a3f13e30c79f3ba16d'}};
// Only selected public runtime fields are requested; environment, credentials and provider logs never enter evidence.
const revisionAnnotations=['run.googleapis.com/cloudsql-instances','run.googleapis.com/cpu-throttling','run.googleapis.com/startup-cpu-boost','run.googleapis.com/execution-environment','run.googleapis.com/vpc-access-connector','run.googleapis.com/vpc-access-egress','autoscaling.knative.dev/minScale','autoscaling.knative.dev/maxScale'];
const serviceAnnotations=['run.googleapis.com/ingress','run.googleapis.com/ingress-status','run.googleapis.com/minScale','run.googleapis.com/maxScale'];
const annotationFields=(prefix,names)=>names.map(k=>`${prefix}."${k}"`);
export const serviceProjection=`json(metadata.name,${annotationFields('metadata.annotations',serviceAnnotations)},spec.template.metadata.name,${annotationFields('spec.template.metadata.annotations',revisionAnnotations)},spec.template.spec.serviceAccountName,spec.template.spec.containerConcurrency,spec.template.spec.timeoutSeconds,spec.template.spec.containers[].image,spec.template.spec.containers[].args,spec.template.spec.containers[].command,spec.template.spec.containers[].ports,spec.template.spec.containers[].resources,spec.template.spec.containers[].startupProbe.initialDelaySeconds,spec.template.spec.containers[].startupProbe.successThreshold,spec.template.spec.containers[].startupProbe.failureThreshold,spec.template.spec.containers[].startupProbe.httpGet.path,spec.template.spec.containers[].startupProbe.httpGet.port,spec.template.spec.containers[].startupProbe.httpGet.scheme,spec.template.spec.containers[].startupProbe.httpGet.httpHeaders[].name,spec.template.spec.containers[].startupProbe.periodSeconds,spec.template.spec.containers[].startupProbe.timeoutSeconds,spec.traffic[].latestRevision,spec.traffic[].revisionName,spec.traffic[].percent,spec.traffic[].tag,status.traffic[].revisionName,status.traffic[].percent,status.traffic[].tag,status.latestReadyRevisionName,status.latestCreatedRevisionName,status.conditions[].type,status.conditions[].status)`;
const jobProjection='json(metadata.name,spec.template.spec.taskCount,spec.template.spec.parallelism,spec.template.spec.template.spec.serviceAccountName,spec.template.spec.template.spec.maxRetries,spec.template.spec.template.spec.timeoutSeconds,spec.template.spec.template.spec.containers[].image,spec.template.spec.template.spec.containers[].command,spec.template.spec.template.spec.containers[].args,spec.template.spec.template.spec.containers[].resources)';
const executionProjection='json(metadata.name,spec.taskCount,spec.template.spec.containers[].image,spec.template.spec.containers[].command,spec.template.spec.containers[].args,spec.template.spec.serviceAccountName,spec.template.spec.maxRetries,spec.template.spec.timeoutSeconds,status.conditions[].type,status.conditions[].status,status.succeededCount,status.failedCount,status.cancelledCount,status.runningCount,status.retriedCount,status.completionTime)';
// Whitelist selected output again locally; omitted provider empty arrays have explicit, stable defaults.
function annotations(raw,names){return Object.fromEntries(names.filter(k=>raw?.[k]!==undefined).map(k=>[k,raw[k]]));}
// Diagnostics contain fixed field-presence/type/count facts, never provider keys, values or messages.
const diagnosticStages=['offline-validation','read-api','read-web','read-migrate','public-api','public-web','public-job','recent-executions'];
const present=(v,k)=>v!==null&&typeof v==='object'&&Object.hasOwn(v,k);
const count=v=>Array.isArray(v)?Math.min(v.length,100):null;
const valueType=v=>v===null?'null':Array.isArray(v)?'array':['undefined','string','number','boolean','object'].includes(typeof v)?typeof v:'other';
export function serviceShape(raw){
 const s=raw?.spec?.template?.spec,c=s?.containers?.[0],p=c?.startupProbe,h=p?.httpGet,port=c?.ports?.[0],resource=c?.resources;
 return {metadataPresent:!!raw?.metadata,templateMetadataPresent:!!raw?.spec?.template?.metadata,templateNamePresent:present(raw?.spec?.template?.metadata,'name'),templateNameType:valueType(raw?.spec?.template?.metadata?.name),templateNameEmpty:raw?.spec?.template?.metadata?.name==='',latestReadyNamePresent:present(raw?.status,'latestReadyRevisionName'),latestReadyNameType:valueType(raw?.status?.latestReadyRevisionName),latestReadyNameEmpty:raw?.status?.latestReadyRevisionName==='',latestCreatedNamePresent:present(raw?.status,'latestCreatedRevisionName'),latestCreatedNameType:valueType(raw?.status?.latestCreatedRevisionName),latestCreatedNameEmpty:raw?.status?.latestCreatedRevisionName==='',templatePresent:!!raw?.spec?.template,statusPresent:!!raw?.status,containers:count(s?.containers),args:count(c?.args),command:count(c?.command),serviceAccountPresent:present(s,'serviceAccountName'),concurrencyPresent:present(s,'containerConcurrency'),timeoutPresent:present(s,'timeoutSeconds'),imagePresent:present(c,'image'),ports:count(c?.ports),portNamePresent:present(port,'name'),portNameKnown:port?.name==='http1',portNameType:valueType(port?.name),portNumberPresent:present(port,'containerPort'),portNumberType:valueType(port?.containerPort),resourcesPresent:!!resource,resourceLimitsPresent:present(resource,'limits'),resourceRequestsPresent:present(resource,'requests'),resourceLimitsUnknownFields:!!resource?.limits&&Object.keys(resource.limits).some(k=>!['cpu','memory'].includes(k)),resourceUnknownFields:!!resource&&Object.keys(resource).some(k=>!['limits','requests'].includes(k)),cpuPresent:present(resource?.limits,'cpu'),cpuType:valueType(resource?.limits?.cpu),memoryPresent:present(resource?.limits,'memory'),memoryType:valueType(resource?.limits?.memory),probePresent:!!p,probeFailureThresholdPresent:present(p,'failureThreshold'),probePeriodPresent:present(p,'periodSeconds'),probeTimeoutPresent:present(p,'timeoutSeconds'),httpPathPresent:present(h,'path'),httpPortPresent:present(h,'port'),httpPortType:valueType(h?.port),probeInitialDelayPresent:present(p,'initialDelaySeconds'),probeInitialDelayZero:p?.initialDelaySeconds===0,probeSuccessThresholdPresent:present(p,'successThreshold'),probeSuccessThresholdOne:p?.successThreshold===1,probeUnknownFields:!!p&&Object.keys(p).some(k=>!['initialDelaySeconds','successThreshold','failureThreshold','httpGet','periodSeconds','timeoutSeconds'].includes(k)),httpGetPresent:!!h,httpSchemePresent:present(h,'scheme'),httpSchemeHTTP:h?.scheme==='HTTP',httpHeadersPresent:present(h,'httpHeaders'),httpHeaders:count(h?.httpHeaders),httpGetUnknownFields:!!h&&Object.keys(h).some(k=>!['path','port','scheme','httpHeaders'].includes(k)),desiredTraffic:count(raw?.spec?.traffic),observedTraffic:count(raw?.status?.traffic),desiredLatestPresent:present(raw?.spec?.traffic?.[0],'latestRevision'),desiredLatestTrue:raw?.spec?.traffic?.[0]?.latestRevision===true,desiredLatestType:valueType(raw?.spec?.traffic?.[0]?.latestRevision),desiredNamedPresent:present(raw?.spec?.traffic?.[0],'revisionName'),observedNamedPresent:present(raw?.status?.traffic?.[0],'revisionName')};
}
function jobShape(raw){const t=raw?.spec?.template?.spec,v=t?.template?.spec;return {metadataPresent:!!raw?.metadata,taskCountPresent:present(t,'taskCount'),parallelismPresent:present(t,'parallelism'),taskTemplatePresent:!!v,serviceAccountPresent:present(v,'serviceAccountName'),maxRetriesPresent:present(v,'maxRetries'),timeoutPresent:present(v,'timeoutSeconds'),containers:count(v?.containers),args:count(v?.containers?.[0]?.args),command:count(v?.containers?.[0]?.command),resourcesPresent:present(v?.containers?.[0],'resources')};}
function diagnostic(onDiagnostic,stage,status,extra={}){if(!diagnosticStages.includes(stage))fail();onDiagnostic({schemaVersion:1,stage,status,...extra});}
function observe(onDiagnostic,stage,fn,shape){
 diagnostic(onDiagnostic,stage,'started');
 try{const value=fn();diagnostic(onDiagnostic,stage,'observed',shape?{shape:shape(value)}:{});return value;}catch{diagnostic(onDiagnostic,stage,'failed');fail();}
}
// Strip only documented harmless HTTP probe defaults; all other fields remain subject to strict equality validation.
function normalizeProbe(raw){
 if(!raw||typeof raw!=='object'||Array.isArray(raw))fail();
 if(Object.keys(raw).some(k=>!['initialDelaySeconds','failureThreshold','httpGet','periodSeconds','timeoutSeconds'].includes(k)))fail();
 if(present(raw,'initialDelaySeconds')&&raw.initialDelaySeconds!==0)fail();
 const h=raw.httpGet;if(!h||typeof h!=='object'||Array.isArray(h)||Object.keys(h).some(k=>!['path','port','scheme','httpHeaders'].includes(k)))fail();
 if(present(h,'scheme')&&h.scheme!=='HTTP'||present(h,'httpHeaders')&&(!Array.isArray(h.httpHeaders)||h.httpHeaders.length!==0))fail();
 return {failureThreshold:raw.failureThreshold,httpGet:{path:h.path,port:h.port},periodSeconds:raw.periodSeconds,timeoutSeconds:raw.timeoutSeconds};
}
// Kubernetes and Cloud Run define exactly 1000m as one CPU; no other resource coercion is allowed.
function normalizeResources(raw){return raw?.limits?.cpu==='1000m'?{...raw,limits:{...raw.limits,cpu:'1'}}:raw;}
function normalizeService(raw,inspection=false){
 const t=raw.spec.template,v=t.spec;
 const traffic=rows=>(rows??[]).map(x=>({revisionName:x.revisionName,percent:x.percent,...(present(x,'tag')?{tag:x.tag}:{})}));
 const observed=traffic(raw.status.traffic),desired=raw.spec.traffic;
 if(!Array.isArray(desired))fail();
 const resolved=desired.map(x=>{
  if(present(x,'latestRevision')&&typeof x.latestRevision!=='boolean')fail();
  if(x.latestRevision!==true)return x;
  const healthy=['Ready','ConfigurationsReady','RoutesReady'].every(type=>raw.status.conditions?.filter(c=>c.type===type&&c.status==='True').length===1);
  if(desired.length!==1||x.percent!==100||present(x,'revisionName')||present(x,'tag')||!healthy||raw.status.latestReadyRevisionName!==raw.status.latestCreatedRevisionName||observed.length!==1||observed[0].percent!==100||present(observed[0],'tag')||observed[0].revisionName!==raw.status.latestReadyRevisionName)fail();
  return {revisionName:raw.status.latestReadyRevisionName,percent:100};
 });
 return {metadata:{name:raw.metadata?.name,annotations:annotations(raw.metadata?.annotations,serviceAnnotations)},spec:{template:{metadata:{name:t.metadata?.name,annotations:annotations(t.metadata?.annotations,revisionAnnotations)},spec:{serviceAccountName:v.serviceAccountName,containerConcurrency:v.containerConcurrency,timeoutSeconds:v.timeoutSeconds,containers:v.containers.map(c=>({image:c.image,args:c.args??[],command:c.command??[],ports:c.ports,resources:normalizeResources(c.resources),startupProbe:inspection?publicProbe(c.startupProbe,raw.metadata?.name==='jayburd-dev-api'?'api':'web'):normalizeProbe(c.startupProbe)}))}},traffic:traffic(resolved)},status:{latestReadyRevisionName:raw.status.latestReadyRevisionName,latestCreatedRevisionName:raw.status.latestCreatedRevisionName,conditions:raw.status.conditions?.map(c=>({type:c.type,status:c.status})),traffic:observed}};
}
// Inspection minimizes public configuration independently of exact promotion compatibility.
function boundedInteger(v,min,max){return Number.isSafeInteger(v)&&v>=min&&v<=max;}
function publicResources(r){
 if(!r||typeof r!=='object'||Array.isArray(r)||Object.keys(r).some(k=>!['limits','requests'].includes(k)))fail();
 keys(r.limits,['cpu','memory']);
 for(const values of [r.limits,...(present(r,'requests')?[r.requests]:[])]){
  if(!values||typeof values!=='object'||Array.isArray(values)||Object.keys(values).some(k=>!['cpu','memory'].includes(k)))fail();
  if(present(values,'cpu')){const v=values.cpu;if(typeof v!=='string'||!(/^(?:[1-8](?:\.[0-9]{1,3})?|0\.[0-9]{1,3}|[1-9][0-9]{0,3}m)$/.test(v))||!(parseFloat(v)/(v.endsWith('m')?1000:1)>0&&parseFloat(v)/(v.endsWith('m')?1000:1)<=8))fail();}
  if(present(values,'memory')){const v=values.memory,m=typeof v==='string'&&/^([1-9][0-9]{0,5})(Mi|Gi)$/.exec(v);if(!m||Number(m[1])*(m[2]==='Gi'?1024:1)>32768)fail();}
 }
 return r;
}
function publicProbe(p,c){
 if(!p||typeof p!=='object'||Array.isArray(p)||Object.keys(p).some(k=>!['initialDelaySeconds','successThreshold','failureThreshold','httpGet','periodSeconds','timeoutSeconds'].includes(k)))fail();
 for(const [k,min,max] of [['initialDelaySeconds',0,3600],['successThreshold',1,10],['failureThreshold',1,100],['periodSeconds',1,3600],['timeoutSeconds',1,3600]])if(present(p,k)&&!boundedInteger(p[k],min,max))fail();
 const h=p.httpGet;if(!h||typeof h!=='object'||Array.isArray(h)||Object.keys(h).some(k=>!['path','port','scheme','httpHeaders'].includes(k))||h.path!==(c==='api'?'/readyz':'/api/auth-config')||!boundedInteger(h.port,1,65535))fail();
 if(present(h,'scheme')&&!['HTTP','HTTPS'].includes(h.scheme)||present(h,'httpHeaders')&&(!Array.isArray(h.httpHeaders)||h.httpHeaders.length!==0))fail();
 const normalized=structuredClone(p);if(normalized.initialDelaySeconds===0)delete normalized.initialDelaySeconds;if(normalized.httpGet.scheme==='HTTP')delete normalized.httpGet.scheme;delete normalized.httpGet.httpHeaders;return normalized;
}
function publicFlags(args,names,booleans=[]){
 if(!Array.isArray(args)||args.length<names.length||args.length>names.length+1)fail();
 const out=new Map();
 for(const raw of args){const m=typeof raw==='string'&&/^--?([a-z][a-z0-9-]*)(?:=([^\r\n\u0000|]*))?$/.exec(raw);if(!m||!names.includes(m[1])&&m[1]!=='automatic-categories'||out.has(m[1]))fail();const value=m[2];if(booleans.includes(m[1])){if(value!==undefined&&!['true','false'].includes(value))fail();}else if(value===undefined||value==='')fail();out.set(m[1],value);}
 if(names.some(k=>!out.has(k)))fail();return out;
}
function publicAPIFlagValue(name,value){
 if(booleanArgumentNames.includes(name))return value===undefined||['true','false'].includes(value);
 const fixed={mode:'managed',listen:'0.0.0.0:8080',project,origin:`https://jayburd-dev-web-${number}.${region}.run.app`,'database-mode':'cloudsql','cloud-sql-instance':`${project}:${region}:jayburd-dev`,'provider-project-number':number,'infisical-environment':'dev','provider-registry-bucket':`${project}-jayburd-dev-credential-registry`,'provider-registry-prefix':'provider-credentials-v1','plaid-webhook-url':`https://jayburd-dev-webhook-${number}.${region}.run.app/webhooks/plaid`};
 if(Object.hasOwn(fixed,name))return value===fixed[name];
 if(name==='infisical-origin')return ['https://us.infisical.com','https://eu.infisical.com','https://app.infisical.com'].includes(value);
 if(['infisical-project','infisical-identity'].includes(name))return typeof value==='string'&&/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/.test(value);
 if(['plaid-client-id-version','plaid-client-secret-version'].includes(name)){const prefix=`projects/${number}/secrets/jayburd-dev-bank-sync-client-${name==='plaid-client-id-version'?'id':'secret'}/versions/`;return typeof value==='string'&&value.startsWith(prefix)&&/^([1-9][0-9]{0,18})$/.test(value.slice(prefix.length))&&BigInt(value.slice(prefix.length))<=9223372036854775807n;}
 if(name==='plaid-consent-expires-at'){const stamp=Date.parse(value);return typeof value==='string'&&/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/.test(value)&&Number.isFinite(stamp)&&new Date(stamp).toISOString().replace('.000Z','Z')===value;}
 return false;
}
function publicAPIArgs(args){
 const flags=publicFlags(args,apiArgumentNames,booleanArgumentNames);
 for(const [name,value] of flags)if(!publicAPIFlagValue(name,value))fail();
}
// Public identifiers and bounded operational quantities are observable; this does not approve a release profile.
function validatePublicService(s,c){
 keys(s,['metadata','spec','status']);keys(s.metadata,['name','annotations']);keys(s.spec,['template','traffic']);keys(s.spec.template,['metadata','spec']);keys(s.spec.template.metadata,['name','annotations']);
 const revision=new RegExp(`^jayburd-dev-${c}-[a-z0-9-]{1,50}$`),t=s.spec.template.spec;
 keys(t,['serviceAccountName','containerConcurrency','timeoutSeconds','containers']);
 if(s.metadata.name!==`jayburd-dev-${c}`||!revision.test(s.spec.template.metadata.name)||!revision.test(s.status?.latestReadyRevisionName)||!revision.test(s.status?.latestCreatedRevisionName)||t.serviceAccountName!==`jayburd-dev-${c==='web'?'frontend':'api'}@${project}.iam.gserviceaccount.com`||!boundedInteger(t.containerConcurrency,1,1000)||!boundedInteger(t.timeoutSeconds,1,3600)||!Array.isArray(t.containers)||t.containers.length!==1)fail();
 const v=t.containers[0];keys(v,['image','args','command','ports','resources','startupProbe']);
 if(typeof v.image!=='string'||!new RegExp(`^us-central1-docker\\.pkg\\.dev/${project}/jayburd-dev/${c}@sha256:[a-f0-9]{64}$`).test(v.image)||!equal(v.command,[])||!Array.isArray(v.ports)||v.ports.length!==1)fail();
 const port=v.ports[0];keys(port,['containerPort',...(present(port,'name')?['name']:[])]);if(!boundedInteger(port.containerPort,1,65535)||present(port,'name')&&!['http1','h2c'].includes(port.name))fail();publicResources(v.resources);publicProbe(v.startupProbe,c);
 if(c==='api')publicAPIArgs(v.args);else if(!equal(v.args,[]))fail();
 for(const [a,names] of [[s.metadata.annotations,serviceAnnotations],[s.spec.template.metadata.annotations,revisionAnnotations]]){
  if(!a||typeof a!=='object'||Array.isArray(a)||Object.keys(a).some(k=>!names.includes(k)))fail();
  for(const [k,value] of Object.entries(a)){const scale=k.endsWith('/minScale')||k.endsWith('/maxScale');const allowed=scale?typeof value==='string'&&/^[0-9]{1,3}$/.test(value):k==='run.googleapis.com/cloudsql-instances'?value===`${project}:${region}:jayburd-dev`:k==='run.googleapis.com/cpu-throttling'||k==='run.googleapis.com/startup-cpu-boost'?['true','false'].includes(value):k==='run.googleapis.com/execution-environment'?['gen1','gen2'].includes(value):k==='run.googleapis.com/vpc-access-egress'?['all-traffic','private-ranges-only'].includes(value):k==='run.googleapis.com/vpc-access-connector'?typeof value==='string'&&new RegExp(`^projects/${project}/locations/${region}/connectors/[a-z][a-z0-9-]{0,62}$`).test(value):['all','internal','internal-and-cloud-load-balancing'].includes(value);if(!allowed)fail();}
 }
 keys(s.status,['latestReadyRevisionName','latestCreatedRevisionName','conditions','traffic']);if(!Array.isArray(s.status.conditions)||s.status.conditions.length>16)fail();
 for(const condition of s.status.conditions){keys(condition,['type','status']);if(typeof condition.type!=='string'||!/^[A-Za-z][A-Za-z0-9]{0,63}$/.test(condition.type)||!['True','False','Unknown'].includes(condition.status))fail();}
 for(const rows of [s.spec.traffic,s.status.traffic]){if(!Array.isArray(rows)||rows.length>16)fail();for(const target of rows){keys(target,['revisionName','percent',...(present(target,'tag')?['tag']:[])]);if(!revision.test(target.revisionName)||!boundedInteger(target.percent,0,100)||present(target,'tag')&&(typeof target.tag!=='string'||!/^[a-z][a-z0-9-]{0,62}$/.test(target.tag)))fail();}}
 return s;
}
function validatePublicJob(j){
 keys(j,['name','taskCount','parallelism','serviceAccountName','maxRetries','timeoutSeconds','containers']);
 const timeout=typeof j.timeoutSeconds==='string'&&/^[1-9][0-9]{0,3}$/.test(j.timeoutSeconds)?Number(j.timeoutSeconds):j.timeoutSeconds;
 if(j.name!=='jayburd-dev-migrate'||j.serviceAccountName!==`jayburd-dev-migrator@${project}.iam.gserviceaccount.com`||!boundedInteger(j.taskCount,1,100)||!boundedInteger(j.parallelism,1,j.taskCount)||!boundedInteger(j.maxRetries,0,10)||!boundedInteger(timeout,1,3600)||!Array.isArray(j.containers)||j.containers.length!==1)fail();
 const c=j.containers[0];keys(c,['image','command','args','resources']);if(!equal(c.command,[])||typeof c.image!=='string'||!new RegExp(`^us-central1-docker\\.pkg\\.dev/${project}/jayburd-dev/migrate@sha256:[a-f0-9]{64}$`).test(c.image))fail();
 const args=publicFlags(c.args,['action','instance']);if(args.size!==2||!['status','up'].includes(args.get('action'))||args.get('instance')!==`${project}:${region}:jayburd-dev`)fail();publicResources(c.resources);return j;
}
// Inspection diagnostics are independent conjunction checks, never a replacement for authoritative validation.
// Labels come exclusively from these fixed code lists; provider names, values and errors are never serialized.
const apiArgumentNames=['mode','listen','project','origin','database-mode','cloud-sql-instance','connections-enabled','plaid-link-events-qualified','provider-project-number','infisical-origin','infisical-project','infisical-environment','infisical-identity','provider-registry-bucket','provider-registry-prefix','plaid-client-id-version','plaid-client-secret-version','plaid-consent-expires-at','plaid-webhook-url','collection-read-enabled'];
const booleanArgumentNames=['connections-enabled','collection-read-enabled','plaid-link-events-qualified','automatic-categories'];
function constraintCollector(){
 const results=[];
 return {check(code,fn){let passed=false;try{passed=fn()===true;}catch{}results.push({code,passed});return passed;},results};
}
function argumentConstraints(args,check,job=false){
 const names=job?['action','instance']:apiArgumentNames,allowed=job?names:[...names,'automatic-categories'],parsed=[];
 check('args-array',()=>Array.isArray(args));check('args-count',()=>Array.isArray(args)&&args.length>=names.length&&args.length<=names.length+(job?0:1));
 if(Array.isArray(args))for(const raw of args){const m=typeof raw==='string'&&/^--?([a-z][a-z0-9-]*)(?:=([^\r\n\u0000|]*))?$/.exec(raw);parsed.push(m?{name:m[1],value:m[2]}:null);}
 check('args-dash-syntax',()=>Array.isArray(args)&&parsed.every(Boolean));
 check('args-known-names',()=>Array.isArray(args)&&parsed.every(v=>v&&allowed.includes(v.name)));
 check('args-unique-names',()=>Array.isArray(args)&&parsed.every(Boolean)&&new Set(parsed.map(v=>v.name)).size===parsed.length);
 for(const name of allowed){
  const items=parsed.filter(v=>v?.name===name),optional=name==='automatic-categories';
  check(optional?'arg-automatic-categories-optional-presence':`arg-${name}-present`,()=>optional||items.length>0);
  check(`arg-${name}-unique`,()=>items.length<2);
  check(`arg-${name}-public-value`,()=>{
   if(optional&&items.length===0)return true;if(items.length!==1)return false;
   const value=items[0].value;
   if(job)return name==='action'?['status','up'].includes(value):value===`${project}:${region}:jayburd-dev`;
   return publicAPIFlagValue(name,value);
  });
 }
}
function resourceConstraints(raw,check){
 check('resources-object-keys',()=>raw&&typeof raw==='object'&&!Array.isArray(raw)&&Object.keys(raw).every(k=>['limits','requests'].includes(k)));
 check('resources-limit-keys',()=>{keys(raw?.limits,['cpu','memory']);return true;});
 for(const section of ['limits','requests']){
  const values=raw?.[section],optional=section==='requests'&&!present(raw,'requests');
  check(`resources-${section}-object-keys`,()=>optional||values&&typeof values==='object'&&!Array.isArray(values)&&Object.keys(values).every(k=>['cpu','memory'].includes(k)));
  for(const field of ['cpu','memory'])check(`resources-${section}-${field}-public`,()=>{
   if(optional||section==='requests'&&!present(values,field))return true;
   const isolated={limits:{cpu:'1',memory:'512Mi'}};if(section==='limits')isolated.limits[field]=values?.[field];else isolated.requests={[field]:values?.[field]};publicResources(isolated);return true;
  });
 }
}
function probeConstraints(p,c,check){
 check('probe-object-known-keys',()=>p&&typeof p==='object'&&!Array.isArray(p)&&Object.keys(p).every(k=>['initialDelaySeconds','successThreshold','failureThreshold','httpGet','periodSeconds','timeoutSeconds'].includes(k)));
 for(const [key,min,max] of [['initialDelaySeconds',0,3600],['successThreshold',1,10],['failureThreshold',1,100],['periodSeconds',1,3600],['timeoutSeconds',1,3600]])check(`probe-${key}-public`,()=>!present(p,key)||boundedInteger(p[key],min,max));
 const h=p?.httpGet;
 check('probe-http-object-known-keys',()=>h&&typeof h==='object'&&!Array.isArray(h)&&Object.keys(h).every(k=>['path','port','scheme','httpHeaders'].includes(k)));
 check('probe-http-path-public',()=>h?.path===(c==='api'?'/readyz':'/api/auth-config'));
 check('probe-http-port-public',()=>boundedInteger(h?.port,1,65535));
 check('probe-http-scheme-public',()=>!present(h,'scheme')||['HTTP','HTTPS'].includes(h.scheme));
 check('probe-http-headers-empty',()=>!present(h,'httpHeaders')||Array.isArray(h.httpHeaders)&&h.httpHeaders.length===0);
}
function annotationConstraints(raw,names,prefix,check){
 const selected=annotations(raw,names);
 check(`${prefix}-selected-keys`,()=>{keys(selected,Object.keys(selected).filter(k=>names.includes(k)));return true;});
 for(const key of names)check(`${prefix}-${key.replaceAll('.','-').replaceAll('/','-')}-public`,()=>{
  if(!present(selected,key))return true;const value=selected[key],scale=key.endsWith('/minScale')||key.endsWith('/maxScale');
  return scale?typeof value==='string'&&/^[0-9]{1,3}$/.test(value):key==='run.googleapis.com/cloudsql-instances'?value===`${project}:${region}:jayburd-dev`:key==='run.googleapis.com/cpu-throttling'||key==='run.googleapis.com/startup-cpu-boost'?['true','false'].includes(value):key==='run.googleapis.com/execution-environment'?['gen1','gen2'].includes(value):key==='run.googleapis.com/vpc-access-egress'?['all-traffic','private-ranges-only'].includes(value):key==='run.googleapis.com/vpc-access-connector'?typeof value==='string'&&new RegExp(`^projects/${project}/locations/${region}/connectors/[a-z][a-z0-9-]{0,62}$`).test(value):['all','internal','internal-and-cloud-load-balancing'].includes(value);
 });
}
function statusConstraints(status,revision,check){
 check('status-conditions-array-count',()=>Array.isArray(status?.conditions)&&status.conditions.length<=16);
 check('status-condition-types-public',()=>Array.isArray(status?.conditions)&&status.conditions.every(v=>typeof v.type==='string'&&/^[A-Za-z][A-Za-z0-9]{0,63}$/.test(v.type)));
 check('status-condition-status-enums',()=>Array.isArray(status?.conditions)&&status.conditions.every(v=>['True','False','Unknown'].includes(v.status)));
 for(const field of ['latestReadyRevisionName','latestCreatedRevisionName'])check(`status-${field}-public`,()=>typeof status?.[field]==='string'&&revision.test(status[field]));
}
function trafficConstraints(rows,revision,prefix,check){
 check(`${prefix}-array-count`,()=>Array.isArray(rows)&&rows.length<=16);
 check(`${prefix}-revision-public`,()=>Array.isArray(rows)&&rows.every(v=>typeof v.revisionName==='string'&&revision.test(v.revisionName)));
 check(`${prefix}-percent-public`,()=>Array.isArray(rows)&&rows.every(v=>boundedInteger(v.percent,0,100)));
 check(`${prefix}-tag-public`,()=>Array.isArray(rows)&&rows.every(v=>!present(v,'tag')||typeof v.tag==='string'&&/^[a-z][a-z0-9-]{0,62}$/.test(v.tag)));
}
// Complete public-field audit continues after independent failures and emits only fixed labels plus booleans.
export function inspectServiceConstraints(raw,c){
 const audit=constraintCollector(),check=audit.check,revision=new RegExp(`^jayburd-dev-${c}-[a-z0-9-]{1,50}$`),t=raw?.spec?.template,v=t?.spec,container=v?.containers?.[0];
 const object=v=>v&&typeof v==='object'&&!Array.isArray(v);
 for(const [code,value] of [['service-object',raw],['metadata-object',raw?.metadata],['spec-object',raw?.spec],['template-object',t],['template-metadata-object',t?.metadata],['template-spec-object',v],['status-object',raw?.status]])check(code,()=>object(value));
 check('service-name-public',()=>raw?.metadata?.name===`jayburd-dev-${c}`);
 check('template-name-public',()=>typeof t?.metadata?.name==='string'&&revision.test(t.metadata.name));
 check('service-account-public',()=>v?.serviceAccountName===`jayburd-dev-${c==='web'?'frontend':'api'}@${project}.iam.gserviceaccount.com`);
 check('concurrency-public',()=>boundedInteger(v?.containerConcurrency,1,1000));check('timeout-public',()=>boundedInteger(v?.timeoutSeconds,1,3600));check('one-container',()=>Array.isArray(v?.containers)&&v.containers.length===1);
 check('container-image-public',()=>typeof container?.image==='string'&&new RegExp(`^us-central1-docker\\.pkg\\.dev/${project}/jayburd-dev/${c}@sha256:[a-f0-9]{64}$`).test(container.image));
 check('container-command-empty',()=>equal(container?.command??[],[]));check('one-port',()=>Array.isArray(container?.ports)&&container.ports.length===1);
 const port=container?.ports?.[0];check('port-known-keys',()=>{keys(port,['containerPort',...(present(port,'name')?['name']:[])]);return true;});check('port-number-public',()=>boundedInteger(port?.containerPort,1,65535));check('port-name-public',()=>!present(port,'name')||['http1','h2c'].includes(port.name));
 resourceConstraints(container?.resources,check);probeConstraints(container?.startupProbe,c,check);
 if(c==='api')argumentConstraints(container?.args,check);else check('web-args-empty',()=>equal(container?.args??[],[]));
 annotationConstraints(raw?.metadata?.annotations,serviceAnnotations,'service-annotations',check);annotationConstraints(t?.metadata?.annotations,revisionAnnotations,'revision-annotations',check);
 statusConstraints(raw?.status,revision,check);
 check('desired-latest-marker-boolean',()=>Array.isArray(raw?.spec?.traffic)&&raw.spec.traffic.every(v=>!present(v,'latestRevision')||typeof v.latestRevision==='boolean'));
 const wantsLatest=Array.isArray(raw?.spec?.traffic)&&raw.spec.traffic.some(v=>v?.latestRevision===true);
 check('latest-resolution-desired-sole100-unnamed-untagged',()=>!wantsLatest||raw.spec.traffic.length===1&&raw.spec.traffic[0].percent===100&&!present(raw.spec.traffic[0],'revisionName')&&!present(raw.spec.traffic[0],'tag'));
 for(const type of ['Ready','ConfigurationsReady','RoutesReady'])check(`latest-resolution-${type}-unique-true`,()=>!wantsLatest||Array.isArray(raw?.status?.conditions)&&raw.status.conditions.filter(v=>v.type===type&&v.status==='True').length===1);
 check('latest-resolution-ready-equals-created',()=>!wantsLatest||raw?.status?.latestReadyRevisionName===raw?.status?.latestCreatedRevisionName);
 check('latest-resolution-observed-sole100-untagged',()=>!wantsLatest||Array.isArray(raw?.status?.traffic)&&raw.status.traffic.length===1&&raw.status.traffic[0].percent===100&&!present(raw.status.traffic[0],'tag'));
 check('latest-resolution-observed-matches-ready',()=>!wantsLatest||Array.isArray(raw?.status?.traffic)&&raw.status.traffic.length===1&&raw.status.traffic[0].revisionName===raw.status.latestReadyRevisionName);
 let normalized;
 check('service-normalization',()=>{normalized=normalizeService(raw,true);return true;});
 const desired=normalized?.spec?.traffic??raw?.spec?.traffic,observed=normalized?.status?.traffic??raw?.status?.traffic;
 trafficConstraints(desired,revision,'desired-traffic',check);trafficConstraints(observed,revision,'observed-traffic',check);
 check('normalized-selected-object-keys',()=>{if(!normalized)return false;keys(normalized,['metadata','spec','status']);keys(normalized.metadata,['name','annotations']);keys(normalized.spec,['template','traffic']);keys(normalized.spec.template,['metadata','spec']);keys(normalized.spec.template.metadata,['name','annotations']);keys(normalized.spec.template.spec,['serviceAccountName','containerConcurrency','timeoutSeconds','containers']);for(const x of normalized.spec.template.spec.containers)keys(x,['image','args','command','ports','resources','startupProbe']);keys(normalized.status,['latestReadyRevisionName','latestCreatedRevisionName','conditions','traffic']);for(const x of normalized.status.conditions)keys(x,['type','status']);for(const rows of [desired,observed])for(const x of rows)keys(x,['revisionName','percent',...(present(x,'tag')?['tag']:[])]);return true;});
 check('complete-public-service-profile',()=>{if(!normalized)return false;validatePublicService(normalized,c);return true;});
 const nameFacts={};for(const [label,value] of [['templateName',t?.metadata?.name],['latestReadyName',raw?.status?.latestReadyRevisionName],['latestCreatedName',raw?.status?.latestCreatedRevisionName]]){nameFacts[`${label}Present`]=value!==undefined;nameFacts[`${label}Type`]=valueType(value);nameFacts[`${label}Empty`]=value==='';nameFacts[`${label}Pattern`]=typeof value==='string'&&revision.test(value);}
 nameFacts.latestResolutionRequired=wantsLatest;
 nameFacts.automaticCategoriesArgumentPresent=c==='api'&&Array.isArray(container?.args)&&container.args.some(v=>typeof v==='string'&&/^--?automatic-categories(?:=|$)/.test(v));
 nameFacts.templateNameEqualsLatestCreated=typeof t?.metadata?.name==='string'&&t.metadata.name===raw?.status?.latestCreatedRevisionName;
 return {constraints:audit.results,facts:nameFacts};
}
export function inspectJobConstraints(raw){
 const audit=constraintCollector(),check=audit.check,t=raw?.spec?.template?.spec,v=t?.template?.spec,c=v?.containers?.[0];
 check('job-name-public',()=>raw?.metadata?.name==='jayburd-dev-migrate');check('job-task-count-public',()=>boundedInteger(t?.taskCount,1,100));check('job-parallelism-public',()=>boundedInteger(t?.parallelism,1,t?.taskCount));check('job-service-account-public',()=>v?.serviceAccountName===`jayburd-dev-migrator@${project}.iam.gserviceaccount.com`);check('job-retries-public',()=>boundedInteger(v?.maxRetries,0,10));
 check('job-timeout-public',()=>boundedInteger(typeof v?.timeoutSeconds==='string'&&/^[1-9][0-9]{0,3}$/.test(v.timeoutSeconds)?Number(v.timeoutSeconds):v?.timeoutSeconds,1,3600));check('job-one-container',()=>Array.isArray(v?.containers)&&v.containers.length===1);check('job-command-empty',()=>equal(c?.command??[],[]));check('job-image-public',()=>typeof c?.image==='string'&&new RegExp(`^us-central1-docker\\.pkg\\.dev/${project}/jayburd-dev/migrate@sha256:[a-f0-9]{64}$`).test(c.image));argumentConstraints(c?.args,check,true);resourceConstraints(c?.resources,check);
 let normalized;check('job-normalization',()=>{normalized=normalizeJob(raw);return true;});check('complete-public-job-profile',()=>{if(!normalized)return false;validatePublicJob(normalized);return true;});return {constraints:audit.results};
}
export function inspectExecutionConstraints(rows){
 const audit=constraintCollector(),check=audit.check;check('executions-array-count',()=>Array.isArray(rows)&&rows.length<=3);
 for(let i=0;i<3;i++){
  const raw=Array.isArray(rows)?rows[i]:undefined;if(raw===undefined)continue;const prefix=`execution-${i+1}`,t=raw?.spec?.template?.spec,c=t?.containers?.[0];
  const sub=(code,fn)=>check(`${prefix}-${code}`,fn);
  sub('name-public',()=>typeof raw?.metadata?.name==='string'&&/^jayburd-dev-migrate-[a-z0-9]{1,20}$/.test(raw.metadata.name));sub('task-count-one',()=>raw?.spec?.taskCount===1);sub('one-container',()=>Array.isArray(t?.containers)&&t.containers.length===1);sub('service-account-public',()=>t?.serviceAccountName===`jayburd-dev-migrator@${project}.iam.gserviceaccount.com`);sub('retries-zero',()=>t?.maxRetries===0);sub('timeout-660',()=>[660,'660'].includes(t?.timeoutSeconds));sub('command-empty',()=>equal(c?.command??[],[]));sub('image-public',()=>typeof c?.image==='string'&&new RegExp(`^us-central1-docker\\.pkg\\.dev/${project}/jayburd-dev/migrate@sha256:[a-f0-9]{64}$`).test(c.image));
  sub('args-exact-public-profile',()=>equal(c?.args??[],['-action=status',`-instance=${project}:${region}:jayburd-dev`])||equal(c?.args??[],['-action=up',`-instance=${project}:${region}:jayburd-dev`]));
  for(const key of ['succeededCount','failedCount','cancelledCount','runningCount','retriedCount'])sub(`${key}-public`,()=>!present(raw?.status,key)||boundedInteger(raw.status[key],0,1));sub('completion-time-public',()=>!present(raw?.status,'completionTime')||typeof raw.status.completionTime==='string'&&Number.isFinite(Date.parse(raw.status.completionTime)));
  sub('conditions-array-count',()=>!present(raw?.status,'conditions')||Array.isArray(raw.status.conditions)&&raw.status.conditions.length<=16);sub('condition-types-public',()=>!present(raw?.status,'conditions')||raw.status.conditions.every(x=>typeof x.type==='string'&&/^[A-Za-z][A-Za-z0-9]{0,63}$/.test(x.type)));sub('condition-status-enums',()=>!present(raw?.status,'conditions')||raw.status.conditions.every(x=>['True','False','Unknown'].includes(x.status)));
  sub('complete-public-execution-profile',()=>{publicExecution(raw);return true;});
 }
 return {constraints:audit.results};
}
// Bind owner-approved bytes, fixed source and finite execution scope before reading build evidence or invoking cloud tools.
export function validateAutomaticManifest(bytes,digest,phase,now=Date.now()){
 if(!phases.includes(phase)||!Number.isFinite(now)||Buffer.byteLength(bytes)>65536||!/^[a-f0-9]{64}$/.test(digest)||hash(bytes)!==digest)fail();
 const m=JSON.parse(bytes); keys(m,['schemaVersion','project','projectNumber','build','images','expiresAt','approval','baseline']);
 keys(m.build,['run','attempt','commit']);keys(m.images,components);keys(m.approval,['issue485','claim65']);if(m.baseline===null){if(phase!=='inspect')fail();}else keys(m.baseline,['api','web','migrate']);
 if(m.schemaVersion!==1||m.project!==project||m.projectNumber!==number||!Object.entries(source).every(([k,v])=>m.build[k]===v))fail();
 const expiry=Date.parse(m.expiresAt);if(typeof m.expiresAt!=='string'||!/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/.test(m.expiresAt)||!Number.isFinite(expiry)||new Date(expiry).toISOString().replace('.000Z','Z')!==m.expiresAt||expiry<=now||expiry>now+86400000)fail();
 for(const [k,n] of [['issue485',485],['claim65',65]])if(typeof m.approval[k]!=='string'||!new RegExp(`^https://github\\.com/justin-napolitano/jayburd-bookkeeping/issues/${n}#issuecomment-[1-9][0-9]*$`).test(m.approval[k]))fail();
 for(const c of components)if(typeof m.images[c]!=='string'||!new RegExp(`^us-central1-docker\\.pkg\\.dev/${project}/jayburd-dev/${c}@sha256:[a-f0-9]{64}$`).test(m.images[c]))fail();
 if(m.baseline!==null){for(const c of ['api','web'])validateService(m.baseline[c],c,true);validateJob(m.baseline.migrate);}
 return m;
}
// Recompute all four immutable receipts independently; retained scans/SBOMs must bind the tested local image.
export function validateAutomaticBuild(m,read,now=Date.now()){
 const run=read('build-run.json'),receipts=[];
 for(const c of components){
  const r=read(`${c}/candidate.json`),b=read(`${c}/build.json`),repos=read(`${c}/repo-digests.json`);
  const prov=b['buildx.build.provenance'];
  if(prov?.invocation?.configSource?.entryPoint!==`Dockerfile.${c}`||prov?.metadata?.['https://mobyproject.org/buildkit@v1#metadata']?.vcs?.revision!==source.commit||prov?.metadata?.['https://mobyproject.org/buildkit@v1#metadata']?.vcs?.source!=='https://github.com/justin-napolitano/jayburd-bookkeeping')fail();
  const rebuilt=imageReceipt({project,component:c,commit:source.commit,run:source.run,attempt:source.attempt,imageID:r.testedImageID},b,repos);
  if(!equal(r,rebuilt)||r.image!==m.images[c])fail(); receipts.push(r);
  const scan=read(`${c}/vulnerabilities.json`),sbom=read(`${c}/sbom.cdx.json`);
  if(scan.Metadata?.ImageID!==r.testedImageID||scan.Metadata?.ImageConfig?.config?.Labels?.['org.opencontainers.image.revision']!==source.commit||scan.Metadata?.ImageConfig?.architecture!=='amd64')fail();
  if(scan.ArtifactName!==r.testedImageID||scan.ArtifactType!=='container_image'||!Array.isArray(scan.Results)||scan.Results.some(x=>(x.Vulnerabilities??[]).some(v=>['HIGH','CRITICAL'].includes(v.Severity))))fail();
  if(!Number.isFinite(Date.parse(scan.CreatedAt))||Date.parse(scan.CreatedAt)>now||now-Date.parse(scan.CreatedAt)>86400000)fail();
  const properties=sbom.metadata?.component?.properties;
  if(!Array.isArray(properties)||!properties.some(p=>p.name==='aquasecurity:trivy:ImageID'&&p.value===r.testedImageID)||!properties.some(p=>p.name==='aquasecurity:trivy:Labels:org.opencontainers.image.revision'&&p.value===source.commit)||!sbom.metadata?.tools?.components?.some(t=>t.name==='trivy'&&t.version==='0.75.0'))fail();
  if(sbom.bomFormat!=='CycloneDX'||sbom.metadata?.component?.name!==r.testedImageID||!Array.isArray(sbom.components)||sbom.components.length===0)fail();
 }
 validateBuild(m,run,receipts.filter(r=>r.component!=='worker'));
 return {build:m.build,images:m.images,components:receipts.map(r=>({component:r.component,testedImageID:r.testedImageID}))};
}
// Validate the finite public service shape; baseline has one fully healthy historical traffic target.
export function validateService(s,c,baseline=false,allowUnhealthy=false){
 keys(s,['metadata','spec','status']); keys(s.metadata,['name','annotations']);keys(s.spec,['template','traffic']);
 keys(s.spec.template,['metadata','spec']);keys(s.spec.template.metadata,['name','annotations']);
 for(const [a,names] of [[s.metadata.annotations,serviceAnnotations],[s.spec.template.metadata.annotations,revisionAnnotations]])if(!a||Object.keys(a).some(k=>!names.includes(k)))fail();
 for(const [a,names] of [[s.metadata.annotations,serviceAnnotations],[s.spec.template.metadata.annotations,revisionAnnotations]])for(const [k,value] of Object.entries(a)){
  const scale=k.endsWith('/minScale')||k.endsWith('/maxScale');
  const allowed=scale?typeof value==='string'&&/^[0-9]{1,3}$/.test(value):k==='run.googleapis.com/cloudsql-instances'?value===`${project}:${region}:jayburd-dev`:k==='run.googleapis.com/cpu-throttling'||k==='run.googleapis.com/startup-cpu-boost'?['true','false'].includes(value):k==='run.googleapis.com/execution-environment'?['gen1','gen2'].includes(value):k==='run.googleapis.com/vpc-access-egress'?['all-traffic','private-ranges-only'].includes(value):k==='run.googleapis.com/vpc-access-connector'?typeof value==='string'&&new RegExp(`^projects/${project}/locations/${region}/connectors/[a-z][a-z0-9-]{0,62}$`).test(value):['all','internal','internal-and-cloud-load-balancing'].includes(value);
  if(!allowed)fail();
 }
 const revisionPattern=new RegExp(`^jayburd-dev-${c}-[a-z0-9-]{1,50}$`);
 if(!revisionPattern.test(s.spec.template.metadata.name)||!revisionPattern.test(s.status?.latestCreatedRevisionName)||!revisionPattern.test(s.status?.latestReadyRevisionName))fail();
 const t=s.spec.template.spec;keys(t,['serviceAccountName','containerConcurrency','timeoutSeconds','containers']);
 if(s.metadata.name!==`jayburd-dev-${c}`||t.serviceAccountName!==`jayburd-dev-${c==='web'?'frontend':'api'}@${project}.iam.gserviceaccount.com`||t.containerConcurrency!==(c==='api'?16:32)||t.timeoutSeconds!==(c==='api'?15:20)||t.containers.length!==1)fail();
 const v=t.containers[0];if(typeof v?.image!=='string'||!new RegExp(`^us-central1-docker\\.pkg\\.dev/${project}/jayburd-dev/${c}@sha256:[a-f0-9]{64}$`).test(v.image))fail();keys(v,['image','args','command','ports','resources','startupProbe']);
 if(!equal(v.command,[])||!equal(v.ports,[{containerPort:8080,name:'http1'}])||!equal(v.resources,{limits:{cpu:'1',memory:'512Mi'}}))fail();
 if(!equal(v.startupProbe,{failureThreshold:12,httpGet:{path:c==='api'?'/readyz':'/api/auth-config',port:8080},periodSeconds:5,timeoutSeconds:3}))fail();
 if(!Array.isArray(v.args)||c==='web'&&v.args.length!==0||c==='api'&&(v.args.length!==20&&v.args.length!==21))fail();
 if(c==='api'){
  const names=['mode','listen','project','origin','database-mode','cloud-sql-instance','connections-enabled','plaid-link-events-qualified','provider-project-number','infisical-origin','infisical-project','infisical-environment','infisical-identity','provider-registry-bucket','provider-registry-prefix','plaid-client-id-version','plaid-client-secret-version','plaid-consent-expires-at','plaid-webhook-url','collection-read-enabled'];
  if(!equal(v.args.filter(x=>!x.startsWith('-automatic-categories=')).map(x=>x.split('=')[0].slice(1)).sort(),names.sort()))fail();
  if(new Set(v.args.map(x=>x.split('=')[0])).size!==v.args.length||v.args.some(x=>typeof x!=='string'||x.includes('|')||x.includes('\n')))fail();
  for(const arg of ['-listen=0.0.0.0:8080',`-origin=https://jayburd-dev-web-${number}.${region}.run.app`,`-provider-project-number=${number}`,'-infisical-environment=dev',`-provider-registry-bucket=${project}-jayburd-dev-credential-registry`,'-provider-registry-prefix=provider-credentials-v1',`-plaid-webhook-url=https://jayburd-dev-webhook-${number}.${region}.run.app/webhooks/plaid`,'-connections-enabled','-mode=managed',`-project=${project}`,'-database-mode=cloudsql',`-cloud-sql-instance=${project}:${region}:jayburd-dev`,'-collection-read-enabled=true','-plaid-link-events-qualified=false','-plaid-consent-expires-at=2026-11-02T04:00:00Z'])if(!v.args.includes(arg))fail();
  for(const [name,suffix] of [['plaid-client-id-version','id'],['plaid-client-secret-version','secret']])if(!v.args.some(x=>new RegExp(`^-${name}=projects/${number}/secrets/jayburd-dev-bank-sync-client-${suffix}/versions/[1-9][0-9]{0,18}$`).test(x)))fail();
  for(const name of ['infisical-project','infisical-identity'])if(!v.args.some(x=>new RegExp(`^-${name}=[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`).test(x)))fail();
  if(!v.args.some(x=>['-infisical-origin=https://us.infisical.com','-infisical-origin=https://eu.infisical.com','-infisical-origin=https://app.infisical.com'].includes(x)))fail();
  if(v.args.some(x=>x.startsWith('-automatic-categories')&&x!=='-automatic-categories=true'))fail();
 }
 keys(s.status,['latestReadyRevisionName','latestCreatedRevisionName','conditions','traffic']);
 if(!Array.isArray(s.status.conditions)||s.status.conditions.length>16||s.status.conditions.some(c=>typeof c.type!=='string'||!/^[A-Za-z][A-Za-z0-9]{0,63}$/.test(c.type)||!['True','False','Unknown'].includes(c.status)))fail();
 for(const rows of [s.spec.traffic,s.status.traffic])if(!Array.isArray(rows)||rows.length>16||rows.some(t=>!revisionPattern.test(t.revisionName)||!Number.isInteger(t.percent)||t.percent<0||t.percent>100||t.tag!==undefined&&(typeof t.tag!=='string'||!/^[a-z][a-z0-9-]{0,62}$/.test(t.tag))))fail();
 for(const condition of s.status.conditions)keys(condition,['type','status']);
 for(const rows of [s.spec.traffic,s.status.traffic])for(const target of rows)keys(target,['revisionName','percent',...(target.tag===undefined?[]:['tag'])]);
 if(allowUnhealthy)return s;
 if(s.status.latestReadyRevisionName!==s.status.latestCreatedRevisionName||!['Ready','ConfigurationsReady','RoutesReady'].every(type=>s.status.conditions?.filter(x=>x.type===type&&x.status==='True').length===1))fail();
 if(!Array.isArray(s.spec.traffic)||s.spec.traffic.length!==1||s.spec.traffic[0].percent!==100||typeof s.spec.traffic[0].revisionName!=='string')fail();
 if(s.spec.traffic.some(x=>x.tag)||s.status.traffic?.some(x=>x.tag)||!equal(s.status.traffic,s.spec.traffic))fail();
 if(baseline&&(s.spec.traffic[0].revisionName!==old[c].revision||v.image!==`us-central1-docker.pkg.dev/${project}/jayburd-dev/${c}@sha256:${old[c].digest}`||v.args.length!==(c==='api'?20:0)))fail();
 return s;
}
// Only this runner touches the cloud CLI. Unknown results stop without retry or provider-output disclosure.
export function cloudRunner(args,onDiagnostic=()=>{},spawn=spawnSync){
 let r;try{r=spawn('gcloud',args,{encoding:'utf8',maxBuffer:1048576,timeout:180000,shell:false});}catch{onDiagnostic({status:'child-failed',failure:'process',exitCode:null,timedOut:false});fail();}
 if(r.error||r.status!==0){onDiagnostic({status:'child-failed',failure:'process',exitCode:Number.isInteger(r.status)&&r.status>=0&&r.status<=255?r.status:null,timedOut:r.error?.code==='ETIMEDOUT'});fail();}
 try{return JSON.parse(r.stdout);}catch{onDiagnostic({status:'child-failed',failure:'invalid-json',exitCode:0,timedOut:false});fail();}
}
const flags=[`--project=${project}`,`--region=${region}`,'--quiet'];
function describe(c,runner){return normalizeService(runner(['run','services','describe',`jayburd-dev-${c}`,...flags,`--format=${serviceProjection}`]));}
// The existing migrator profile is status-only at rest; no arbitrary arguments, tasks or identities are accepted.
export function validateJob(j,allowUp=false){
 keys(j,['name','taskCount','parallelism','serviceAccountName','maxRetries','timeoutSeconds','containers']);
 if(j.name!=='jayburd-dev-migrate'||j.taskCount!==1||j.parallelism!==1||j.serviceAccountName!==`jayburd-dev-migrator@${project}.iam.gserviceaccount.com`||j.maxRetries!==0||![660,'660'].includes(j.timeoutSeconds)||j.containers.length!==1)fail();
 const c=j.containers[0];keys(c,['image','command','args','resources']);
 if(!equal(c.command,[])||(!equal(c.args,['-action=status',`-instance=${project}:${region}:jayburd-dev`])&&!(allowUp&&equal(c.args,['-action=up',`-instance=${project}:${region}:jayburd-dev`])))||!equal(c.resources,{limits:{cpu:'1',memory:'512Mi'}})||!new RegExp(`^us-central1-docker\\.pkg\\.dev/${project}/jayburd-dev/migrate@sha256:[a-f0-9]{64}$`).test(c.image))fail();return j;
}
// Inspection may report failed health, but only typed public migration arguments and identities can reach its receipt.
function publicExecution(raw){
 const name=raw?.metadata?.name,t=raw?.spec?.template?.spec;
 if(typeof name!=='string'||!/^jayburd-dev-migrate-[a-z0-9]{1,20}$/.test(name)||!t||raw.spec.taskCount!==1||!Array.isArray(t.containers)||t.containers.length!==1)fail();
 const container=t.containers[0];
 const profile={name:'jayburd-dev-migrate',taskCount:1,parallelism:1,serviceAccountName:t.serviceAccountName,maxRetries:t.maxRetries,timeoutSeconds:t.timeoutSeconds,containers:[{image:container.image,command:container.command??[],args:container.args??[],resources:{limits:{cpu:'1',memory:'512Mi'}}}]};
 validateJob(profile,true);
 const status={};
 for(const k of ['succeededCount','failedCount','cancelledCount','runningCount','retriedCount'])if(raw.status?.[k]!==undefined){if(!Number.isSafeInteger(raw.status[k])||raw.status[k]<0||raw.status[k]>1)fail();status[k]=raw.status[k];}
 if(raw.status?.completionTime!==undefined){if(typeof raw.status.completionTime!=='string'||!Number.isFinite(Date.parse(raw.status.completionTime)))fail();status.completionTime=new Date(raw.status.completionTime).toISOString();}
 if(raw.status?.conditions!==undefined){if(!Array.isArray(raw.status.conditions)||raw.status.conditions.length>16)fail();status.conditions=raw.status.conditions.map(c=>{if(typeof c.type!=='string'||!/^[A-Za-z][A-Za-z0-9]{0,63}$/.test(c.type)||!['True','False','Unknown'].includes(c.status))fail();return {type:c.type,status:c.status};});}
 return {metadata:{name},spec:{taskCount:1,template:{spec:{serviceAccountName:t.serviceAccountName,maxRetries:t.maxRetries,timeoutSeconds:t.timeoutSeconds,containers:[{image:container.image,command:container.command??[],args:container.args??[]}]}}},status};
}
function normalizeJob(j){
 const t=j.spec.template.spec,v=t.template.spec;
 return {name:j.metadata.name,taskCount:t.taskCount,parallelism:t.parallelism,serviceAccountName:v.serviceAccountName,maxRetries:v.maxRetries,timeoutSeconds:v.timeoutSeconds,containers:v.containers.map(c=>({image:c.image,command:c.command??[],args:c.args??[],resources:normalizeResources(c.resources)}))};
}
function describeJob(runner){return normalizeJob(runner(['run','jobs','describe','jayburd-dev-migrate',...flags,`--format=${jobProjection}`]));}
function migrate(m,before,runner,mutate){
 const original=before.migrate;validateJob(original);if(!equal(original,m.baseline.migrate)||original.containers[0].image!==`us-central1-docker.pkg.dev/${project}/jayburd-dev/migrate@sha256:199ed337de73e622c7398516a90ef196620b100701cb3f07924e6cd86df5f31a`)fail();
 for(const c of ['api','web'])if(!equal(before[c],m.baseline[c]))fail();
 const update=action=>mutate(`migrator-default-${action}`, ['run','jobs','update','jayburd-dev-migrate',...flags,`--image=${m.images.migrate}`,`--args=^|^-action=${action}|-instance=${project}:${region}:jayburd-dev`,'--format=json(metadata.name)']);
 const execute=(operation,image=m.images.migrate)=>{
  const launched=mutate(`migration-execution-${image===original.containers[0].image?'old-status':operation==='migrate'?'candidate-up':'candidate-status'}`,['run','jobs','execute','jayburd-dev-migrate',...flags,'--wait','--format=json(metadata.name)']);
  const name=launched?.metadata?.name;if(typeof name!=='string'||!/^jayburd-dev-migrate-[a-z0-9]{1,20}$/.test(name))fail();
  const raw=runner(['run','jobs','executions','describe',name,...flags,`--format=${executionProjection}`]);
  return migrationReceipt({...m,images:{...m.images,migrate:image},operation,evidence:m.approval},raw,name);
 };
 const oldStatus=execute('verify',original.containers[0].image);
 // The historical preflight image was not produced by the candidate build.
 const preflight={...oldStatus,build:null,targetBuild:m.build,limitations:[...oldStatus.limitations,'Historical schema47 baseline image; not produced by targetBuild and not patched-runtime acceptance.']};
 update('up');
 const up=describeJob(runner),expected=structuredClone(original);expected.containers[0].image=m.images.migrate;expected.containers[0].args[0]='-action=up';if(!equal(up,expected))fail();
 const applied=execute('migrate');
 update('status');const restored=describeJob(runner);expected.containers[0].args[0]='-action=status';if(!equal(restored,expected))fail();validateJob(restored);
 const verified=execute('verify');return {schemaVersion:1,phase:'migrate',status:'forward-migration-and-candidate-status-passed',before,after:restored,preflight,applied,verified};
}
// Direct revision health is separate from the service's aggregate Ready condition.
function revisionHealth(c,name,runner,active){
 for(let i=0;i<6;i++){
  const r=runner(['run','revisions','describe',name,...flags,'--format=json(metadata.name,status.conditions[].type,status.conditions[].status,status.conditions[].reason)']);
  if(r.metadata?.name!==name||!Array.isArray(r.status?.conditions))fail();
  const trueCondition=type=>r.status.conditions.filter(x=>x.type===type&&x.status==='True').length===1;
  const accepted=active?['Ready','ContainerHealthy','ResourcesAvailable','Active'].every(trueCondition):trueCondition('Ready')||r.status.conditions.some(x=>x.type==='Active'&&x.status==='False'&&x.reason==='Retired');
  if(accepted)return {name,status:active?'active-health-passed':trueCondition('ContainerHealthy')?'staged-health-observed':'staged-import-only-no-startup-proof',conditions:r.status.conditions.map(v=>({type:v.type,status:v.status,...(v.reason?{reason:v.reason}:{})}))};
  if(i===5)fail();if(runner===cloudRunner)Atomics.wait(new Int32Array(new SharedArrayBuffer(4)),0,0,5000);
 }
}
// Runtime updates change only image, then separately the approved API scheduling and automatic flag; IAM is never invoked.
export function executeAutomaticPhase(m,phase,runner=cloudRunner,onAttempt=()=>{},onDiagnostic=()=>{}){
 let currentStage='read-api';
 const command=args=>runner===cloudRunner?cloudRunner(args,d=>diagnostic(onDiagnostic,currentStage,d.status,{failure:d.failure,exitCode:d.exitCode,timedOut:d.timedOut})):runner(args);
 const step=(stage,fn,shape)=>{currentStage=stage;return observe(onDiagnostic,stage,fn,shape);};
 const mutate=(operation,args)=>{
  const attempt={schemaVersion:1,phase,status:'mutation-attempted-outcome-not-yet-verified',operation,at:new Date().toISOString()};onAttempt(attempt);
  const reply=runner(args);const observed=reply?.metadata?.name??reply?.status?.latestCreatedRevisionName;
  if(typeof observed==='string'&&/^jayburd-dev-(api|web|migrate)-[a-z0-9-]{1,50}$/.test(observed))onAttempt({...attempt,status:'mutation-replied-requires-readback',observed});
  return reply;
 };
 if(!phases.includes(phase)||m.baseline===null&&phase!=='inspect')fail();
 if(phase==='inspect'){
  let incomplete=false;const before={};
  const attempt=(stage,fn,shape)=>{try{return step(stage,fn,shape);}catch{incomplete=true;return undefined;}};
  for(const c of ['api','web']){
   const raw=attempt(`read-${c}`,()=>command(['run','services','describe',`jayburd-dev-${c}`,...flags,`--format=${serviceProjection}`]),serviceShape);
   before[c]=attempt(`public-${c}`,()=>{diagnostic(onDiagnostic,`public-${c}`,'constraints',inspectServiceConstraints(raw,c));const normalized=normalizeService(raw,true);validatePublicService(normalized,c);return normalized;});
  }
  const rawJob=attempt('read-migrate',()=>command(['run','jobs','describe','jayburd-dev-migrate',...flags,`--format=${jobProjection}`]),jobShape);
  before.migrate=attempt('public-job',()=>{diagnostic(onDiagnostic,'public-job','constraints',inspectJobConstraints(rawJob));const normalized=normalizeJob(rawJob);validatePublicJob(normalized);return normalized;});
  const executions=attempt('recent-executions',()=>{const rows=command(['run','jobs','executions','list','--job=jayburd-dev-migrate','--limit=3',...flags,`--format=${executionProjection}`]);diagnostic(onDiagnostic,'recent-executions','constraints',inspectExecutionConstraints(rows));if(!Array.isArray(rows)||rows.length>3)fail();return rows.map(publicExecution);},rows=>({executions:count(rows)}));
  if(incomplete)fail();
  const validation=Object.fromEntries(['api','web','migrate'].map(c=>{try{c==='migrate'?validateJob(before[c]):validateService(before[c],c);return [c,'healthy-selected-profile'];}catch{return [c,'requires-owner-review'];}}));
  return {schemaVersion:1,phase,status:'inspection-only-not-release-acceptance',before,validation,executions};
 }
 const before={};
 for(const c of ['api','web']){
  const raw=step(`read-${c}`,()=>command(['run','services','describe',`jayburd-dev-${c}`,...flags,`--format=${serviceProjection}`]),serviceShape);
  before[c]=step(`public-${c}`,()=>{const normalized=normalizeService(raw);validateService(normalized,c);return normalized;});
 }
 const rawJob=step('read-migrate',()=>command(['run','jobs','describe','jayburd-dev-migrate',...flags,`--format=${jobProjection}`]),jobShape);
 before.migrate=step('public-job',()=>{const j=normalizeJob(rawJob);validateJob(j);return j;});
 for(const c of ['api','web'])validateService(before[c],c);validateJob(before.migrate);
 if(phase==='migrate')return migrate(m,before,runner,mutate);
 const c=phase==='web'?'web':'api',current=before[c],base=m.baseline[c];
 const expected=structuredClone(base);
 if(['activate','pause'].includes(phase))expected.spec.template.spec.containers[0].image=m.images.api;
 if(phase==='pause'){expected.spec.template.spec.containers[0].args.push('-automatic-categories=true');expected.spec.template.metadata.annotations['run.googleapis.com/cpu-throttling']='false';expected.spec.template.metadata.annotations['autoscaling.knative.dev/maxScale']='1';expected.spec.template.metadata.annotations['autoscaling.knative.dev/minScale']='0';expected.metadata.annotations['run.googleapis.com/maxScale']='1';expected.metadata.annotations['run.googleapis.com/minScale']='0';}
 const config=s=>({metadata:{...s.metadata,annotations:{'run.googleapis.com/minScale':'0',...s.metadata.annotations}},spec:s.spec.template.spec,annotations:{'autoscaling.knative.dev/minScale':'0',...s.spec.template.metadata.annotations}});
 if(!equal(config(current),config(expected)))fail();
 if(!['activate','pause'].includes(phase)&&!equal(current.spec.traffic,base.spec.traffic))fail();
 if(phase==='activate'&&(current.spec.traffic[0].revisionName!==current.status.latestReadyRevisionName||current.status.latestReadyRevisionName!==`jayburd-dev-api-as-${source.run}-stage`))fail();
 const other=c==='api'?'web':'api';
 const otherExpected=structuredClone(m.baseline[other]);
 if(phase==='web'){
  otherExpected.spec.template.spec.containers[0].image=m.images.api;otherExpected.spec.template.spec.containers[0].args.push('-automatic-categories=true');
  otherExpected.spec.template.metadata.annotations['run.googleapis.com/cpu-throttling']='false';otherExpected.spec.template.metadata.annotations['autoscaling.knative.dev/minScale']='0';otherExpected.spec.template.metadata.annotations['autoscaling.knative.dev/maxScale']='1';otherExpected.metadata.annotations['run.googleapis.com/minScale']='0';otherExpected.metadata.annotations['run.googleapis.com/maxScale']='1';
 }
 if(phase==='pause'&&before.web.spec.traffic[0].revisionName===`jayburd-dev-web-as-${source.run}-web`)otherExpected.spec.template.spec.containers[0].image=m.images.web;
 if(!equal(config(before[other]),config(otherExpected)))fail();
 if(phase==='web'&&before.api.spec.traffic[0].revisionName!==`jayburd-dev-api-as-${source.run}-activate`)fail();
 if(phase!=='web'&&phase!=='pause'&&!equal(before[other].spec.traffic,m.baseline[other].spec.traffic))fail();
 if(phase==='pause'&&!equal(before.web.spec.traffic,m.baseline.web.spec.traffic)&&before.web.spec.traffic[0].revisionName!==`jayburd-dev-web-as-${source.run}-web`)fail();
 if(phase==='pause'&&current.spec.traffic[0].revisionName!==`jayburd-dev-api-as-${source.run}-activate`)fail();
 const migrationExpected=structuredClone(m.baseline.migrate);migrationExpected.containers[0].image=m.images.migrate;if(!equal(before.migrate,migrationExpected))fail();
 const suffix=`as-${source.run}-${phase}`,name=`jayburd-dev-${c}-${suffix}`;
 const args=['run','services','update',`jayburd-dev-${c}`,...flags,`--image=${m.images[c]}`,`--revision-suffix=${suffix}`,'--no-traffic'];
 if(phase==='pause')args.push(`--args=^|^${base.spec.template.spec.containers[0].args.join('|')}`,'--cpu-throttling','--min=0','--max=1','--min-instances=0','--max-instances=1');
 if(phase==='activate')args.push(`--args=^|^${[...base.spec.template.spec.containers[0].args,'-automatic-categories=true'].join('|')}`,'--no-cpu-throttling','--min=0','--max=1','--min-instances=0','--max-instances=1');
 args.push('--format=json(status.latestReadyRevisionName,status.latestCreatedRevisionName,status.conditions[].type,status.conditions[].status)');
 mutate(`stage-${c}`,args);
 // A successful mutation is followed by bounded selected-metadata reads; unknown mutation outcomes are never replayed.
 let staged;
 for(let i=0;i<6;i++){
  staged=describe(c,runner);
  if(staged.status.latestReadyRevisionName===name&&staged.status.latestCreatedRevisionName===name&&staged.status.conditions?.some(x=>x.type==='Ready'&&x.status==='True'))break;
  if(i===5)fail();
  if(runner===cloudRunner)Atomics.wait(new Int32Array(new SharedArrayBuffer(4)),0,0,5000);
 }
 validateService(staged,c);
 if(staged.status.latestReadyRevisionName!==name||!equal(staged.spec.traffic,current.spec.traffic)||!equal(staged.status.traffic,current.status.traffic)||staged.spec.template.spec.containers[0].image!==m.images[c])fail();
 const desired=structuredClone(current.spec.template.spec);desired.containers[0].image=m.images[c];
 if(phase==='pause')desired.containers[0].args=[...base.spec.template.spec.containers[0].args];
 if(phase==='activate')desired.containers[0].args.push('-automatic-categories=true');
 if(!equal(staged.spec.template.spec,desired))fail();
 if(!['activate','pause'].includes(phase)&&(!equal(staged.metadata,current.metadata)||!equal(staged.spec.template.metadata.annotations,current.spec.template.metadata.annotations)))fail();
 if(['activate','pause'].includes(phase)){
  const a=staged.spec.template.metadata.annotations;
  if(a['run.googleapis.com/cpu-throttling']!==(phase==='activate'?'false':'true')||a['autoscaling.knative.dev/maxScale']!=='1'||(a['autoscaling.knative.dev/minScale']??'0')!=='0')fail();
  const omit=(a,ks)=>Object.fromEntries(Object.entries(a).filter(([k])=>!ks.includes(k)));
  if(!equal(omit(a,['run.googleapis.com/cpu-throttling','autoscaling.knative.dev/maxScale','autoscaling.knative.dev/minScale']),omit(current.spec.template.metadata.annotations,['run.googleapis.com/cpu-throttling','autoscaling.knative.dev/maxScale','autoscaling.knative.dev/minScale'])))fail();
  if(!equal(omit(staged.metadata.annotations,['run.googleapis.com/minScale','run.googleapis.com/maxScale']),omit(current.metadata.annotations,['run.googleapis.com/minScale','run.googleapis.com/maxScale']))||staged.metadata.annotations['run.googleapis.com/maxScale']!=='1'||(staged.metadata.annotations['run.googleapis.com/minScale']??'0')!=='0')fail();
 }
 const stagedHealth=revisionHealth(c,name,runner,false);
 mutate(`promote-${c}`,['run','services','update-traffic',`jayburd-dev-${c}`,...flags,`--to-revisions=${name}=100`,'--format=json(status.conditions[].type,status.conditions[].status)']);
 const activeHealth=revisionHealth(c,name,runner,true);
 const after=describe(c,runner);validateService(after,c);
 if(!equal(after.spec.template.spec,staged.spec.template.spec)||!equal(after.metadata,staged.metadata)||!equal(after.spec.template.metadata.annotations,staged.spec.template.metadata.annotations)||after.spec.traffic[0].revisionName!==name)fail();
 if(!equal(describe(other,runner),before[other])||!equal(describeJob(runner),before.migrate))fail();
 return {schemaVersion:1,phase,status:'runtime-promoted-not-product-acceptance',before,staged,after,stagedHealth,activeHealth,build:m.build,images:m.images};
}
// CLI accepts only named manifest/evidence files and prints fixed outcomes; no process environment or credential cache is inspected.
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){
 try{
  const [mode,path,digest,phase,evidence,out]=process.argv.slice(2);if(!['validate','execute'].includes(mode)||!out||process.argv.length!==8)fail();
  const diagnostics=[];
  const onDiagnostic=event=>{diagnostics.push(event);if(diagnostics.length>128)diagnostics.shift();mkdirSync(out,{recursive:true,mode:0o700});writeFileSync(`${out}/automatic-${phase}-diagnostics.json`,JSON.stringify({schemaVersion:1,manifestSHA256:digest,phase,diagnostics},null,2)+'\n',{mode:0o600});};
  if(!phases.includes(phase)||!/^[a-f0-9]{64}$/.test(digest))fail();
  const {m,qualification}=observe(onDiagnostic,'offline-validation',()=>{const m=validateAutomaticManifest(readFileSync(path),digest,phase);return {m,qualification:validateAutomaticBuild(m,file=>JSON.parse(readFileSync(`${evidence}/${file}`,'utf8')))};});
  const result=mode==='execute'?executeAutomaticPhase(m,phase,cloudRunner,attempt=>{mkdirSync(out,{recursive:true,mode:0o700});writeFileSync(`${out}/automatic-${phase}-attempt.json`,JSON.stringify({manifestSHA256:digest,...attempt},null,2)+'\n',{mode:0o600});},onDiagnostic):{schemaVersion:1,phase,status:'offline-validation-only',qualification};
  mkdirSync(out,{recursive:true,mode:0o700});writeFileSync(`${out}/automatic-${phase}.json`,JSON.stringify({manifestSHA256:digest,manifest:m,...result},null,2)+'\n',{mode:0o600});
  console.log('PASS: bounded synthetic release phase recorded');
 }catch{console.error('BLOCKED: bounded synthetic release; inspect selected receipt before any new attempt');process.exitCode=1;}
}
