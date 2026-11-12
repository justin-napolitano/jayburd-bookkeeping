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
export const serviceProjection=`json(metadata.name,${annotationFields('metadata.annotations',serviceAnnotations)},spec.template.metadata.name,${annotationFields('spec.template.metadata.annotations',revisionAnnotations)},spec.template.spec.serviceAccountName,spec.template.spec.containerConcurrency,spec.template.spec.timeoutSeconds,spec.template.spec.containers[].image,spec.template.spec.containers[].args,spec.template.spec.containers[].command,spec.template.spec.containers[].ports,spec.template.spec.containers[].resources,spec.template.spec.containers[].startupProbe.failureThreshold,spec.template.spec.containers[].startupProbe.httpGet.path,spec.template.spec.containers[].startupProbe.httpGet.port,spec.template.spec.containers[].startupProbe.periodSeconds,spec.template.spec.containers[].startupProbe.timeoutSeconds,spec.traffic[].revisionName,spec.traffic[].percent,spec.traffic[].tag,status.traffic[].revisionName,status.traffic[].percent,status.traffic[].tag,status.latestReadyRevisionName,status.latestCreatedRevisionName,status.conditions[].type,status.conditions[].status)`;
const jobProjection='json(metadata.name,spec.template.spec.taskCount,spec.template.spec.parallelism,spec.template.spec.template.spec.serviceAccountName,spec.template.spec.template.spec.maxRetries,spec.template.spec.template.spec.timeoutSeconds,spec.template.spec.template.spec.containers[].image,spec.template.spec.template.spec.containers[].command,spec.template.spec.template.spec.containers[].args,spec.template.spec.template.spec.containers[].resources)';
const executionProjection='json(metadata.name,spec.taskCount,spec.template.spec.containers[].image,spec.template.spec.containers[].command,spec.template.spec.containers[].args,spec.template.spec.serviceAccountName,spec.template.spec.maxRetries,spec.template.spec.timeoutSeconds,status.conditions[].type,status.conditions[].status,status.succeededCount,status.failedCount,status.cancelledCount,status.runningCount,status.retriedCount,status.completionTime)';
// Whitelist selected output again locally; omitted provider empty arrays have explicit, stable defaults.
function annotations(raw,names){return Object.fromEntries(names.filter(k=>raw?.[k]!==undefined).map(k=>[k,raw[k]]));}
function normalizeService(raw){
 const t=raw.spec.template,v=t.spec;
 const traffic=rows=>(rows??[]).map(x=>({revisionName:x.revisionName,percent:x.percent,...(x.tag?{tag:x.tag}:{})}));
 return {metadata:{name:raw.metadata?.name,annotations:annotations(raw.metadata?.annotations,serviceAnnotations)},spec:{template:{metadata:{name:t.metadata?.name,annotations:annotations(t.metadata?.annotations,revisionAnnotations)},spec:{serviceAccountName:v.serviceAccountName,containerConcurrency:v.containerConcurrency,timeoutSeconds:v.timeoutSeconds,containers:v.containers.map(c=>({image:c.image,args:c.args??[],command:c.command??[],ports:c.ports,resources:c.resources,startupProbe:c.startupProbe}))}},traffic:traffic(raw.spec.traffic)},status:{latestReadyRevisionName:raw.status.latestReadyRevisionName,latestCreatedRevisionName:raw.status.latestCreatedRevisionName,conditions:raw.status.conditions?.map(c=>({type:c.type,status:c.status})),traffic:traffic(raw.status.traffic)}};
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
  for(const arg of ['-listen=0.0.0.0:8080',`-origin=https://jayburd-dev-web-${number}.${region}.run.app`,`-provider-project-number=${number}`,'-infisical-environment=dev',`-provider-registry-bucket=${project}-jayburd-dev-credential-registry`,'-provider-registry-prefix=provider-credentials-v1',`-plaid-webhook-url=https://jayburd-dev-webhook-${number}.${region}.run.app`,'-connections-enabled','-mode=managed',`-project=${project}`,'-database-mode=cloudsql',`-cloud-sql-instance=${project}:${region}:jayburd-dev`,'-collection-read-enabled=true','-plaid-link-events-qualified=false','-plaid-consent-expires-at=2026-11-02T04:00:00Z'])if(!v.args.includes(arg))fail();
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
export function cloudRunner(args){const r=spawnSync('gcloud',args,{encoding:'utf8',maxBuffer:1048576,timeout:180000,shell:false});if(r.error||r.status!==0)fail();try{return JSON.parse(r.stdout);}catch{fail();}}
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
function describeJob(runner){
 const j=runner(['run','jobs','describe','jayburd-dev-migrate',...flags,`--format=${jobProjection}`]);
 const t=j.spec.template.spec,v=t.template.spec;
 return {name:j.metadata.name,taskCount:t.taskCount,parallelism:t.parallelism,serviceAccountName:v.serviceAccountName,maxRetries:v.maxRetries,timeoutSeconds:v.timeoutSeconds,containers:v.containers.map(c=>({image:c.image,command:c.command??[],args:c.args??[],resources:c.resources}))};
}
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
 const preflight=execute('verify',original.containers[0].image);
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
export function executeAutomaticPhase(m,phase,runner=cloudRunner,onAttempt=()=>{}){
 const mutate=(operation,args)=>{
  const attempt={schemaVersion:1,phase,status:'mutation-attempted-outcome-not-yet-verified',operation,at:new Date().toISOString()};onAttempt(attempt);
  const reply=runner(args);const observed=reply?.metadata?.name??reply?.status?.latestCreatedRevisionName;
  if(typeof observed==='string'&&/^jayburd-dev-(api|web|migrate)-[a-z0-9-]{1,50}$/.test(observed))onAttempt({...attempt,status:'mutation-replied-requires-readback',observed});
  return reply;
 };
 if(!phases.includes(phase)||m.baseline===null&&phase!=='inspect')fail();
 const before={api:describe('api',runner),web:describe('web',runner),migrate:describeJob(runner)};
 if(phase==='inspect'){
  // Health failures remain observable; configuration/privacy failures stop before any receipt is written.
  for(const c of ['api','web'])validateService(before[c],c,false,true);validateJob(before.migrate,true);
  const rawExecutions=runner(['run','jobs','executions','list','--job=jayburd-dev-migrate','--limit=3',...flags,`--format=${executionProjection}`]);
  if(!Array.isArray(rawExecutions)||rawExecutions.length>3)fail();
  const executions=rawExecutions.map(publicExecution);
  const validation=Object.fromEntries(['api','web','migrate'].map(c=>{try{c==='migrate'?validateJob(before[c]):validateService(before[c],c);return [c,'healthy-selected-profile'];}catch{return [c,'requires-owner-review'];}}));
  return {schemaVersion:1,phase,status:'inspection-only-not-release-acceptance',before,validation,executions};
 }
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
  const m=validateAutomaticManifest(readFileSync(path),digest,phase);
  const qualification=validateAutomaticBuild(m,file=>JSON.parse(readFileSync(`${evidence}/${file}`,'utf8')));
  const result=mode==='execute'?executeAutomaticPhase(m,phase,cloudRunner,attempt=>{mkdirSync(out,{recursive:true,mode:0o700});writeFileSync(`${out}/automatic-${phase}-attempt.json`,JSON.stringify({manifestSHA256:digest,...attempt},null,2)+'\n',{mode:0o600});}):{schemaVersion:1,phase,status:'offline-validation-only',qualification};
  mkdirSync(out,{recursive:true,mode:0o700});writeFileSync(`${out}/automatic-${phase}.json`,JSON.stringify({manifestSHA256:digest,manifest:m,...result},null,2)+'\n',{mode:0o600});
  console.log('PASS: bounded synthetic release phase recorded');
 }catch{console.error('BLOCKED: bounded synthetic release; inspect selected receipt before any new attempt');process.exitCode=1;}
}
