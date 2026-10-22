import {readFileSync,writeFileSync,mkdirSync} from 'node:fs';
import {execFileSync} from 'node:child_process';
import {pathToFileURL} from 'node:url';
import {validateWebhookManifest,validateWebhookBuild} from './webhook-release-manifest.mjs';

// These immutable merged prerequisites contain the actual host and exact release contract.
export const requiredWebhookAncestors=['a2c9dee793f81ed483055b0ee916876b59265d27','24f263ef2cde697cf0f8cf51c1176d5c1521e679'];
export const webhookSourcePaths=['Dockerfile.worker','go.mod','go.sum','internal','cmd/worker','cmd/webhook','cmd/deliver','cmd/sandbox-probe'];
// Workflow context is an additional guard; cloud WIF independently enforces owner/main/exact workflow SHA.
export function ownerContext(e){
  if(e.GITHUB_ACTIONS!=='true'||e.GITHUB_EVENT_NAME!=='workflow_dispatch'||e.GITHUB_REF!=='refs/heads/main'||e.GITHUB_ACTOR_ID!=='55423093'||e.GITHUB_REPOSITORY_ID!=='1414515318'||e.GITHUB_TRIGGERING_ACTOR!=='justin-napolitano'||
    e.GITHUB_WORKFLOW_REF!=='justin-napolitano/jayburd-bookkeeping/.github/workflows/deploy-webhook.yml@refs/heads/main'||!/^[a-f0-9]{40}$/.test(e.GITHUB_SHA??'')||e.CURRENT_MAIN_SHA!==e.GITHUB_SHA)throw new Error('Webhook rollout rejected');
}
// Git verifies immutable source ancestry and rejects changed binary inputs even on descendant main commits.
export function validateAncestry(m,head,git){
  if(!/^[a-f0-9]{40}$/.test(head??''))throw new Error('Webhook rollout rejected');
  git(['merge-base','--is-ancestor',m.build.commit,head]);
  for(const prerequisite of requiredWebhookAncestors)git(['merge-base','--is-ancestor',prerequisite,m.build.commit]);
  git(['diff','--quiet',m.build.commit,head,'--',...webhookSourcePaths]);
}
// Only validated public metadata is generated; no deployment, credentials or state is read here.
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href){
  try{
    const [manifest,hash,destination,evidence]=process.argv.slice(2);
    if(process.argv.length!==6)throw new Error();
    ownerContext(process.env);
    const bytes=readFileSync(manifest);
    const m=validateWebhookManifest(bytes,hash);
    const json=file=>JSON.parse(readFileSync(`${evidence}/${file}`,'utf8'));
    validateWebhookBuild(m,json('build-run.json'),json('worker/candidate.json'));
    const git=args=>execFileSync('git',args,{stdio:['ignore','pipe','ignore']});
    if(git(['rev-parse','HEAD']).toString().trim()!==process.env.GITHUB_SHA||!git(['show',`${process.env.GITHUB_SHA}:deploy/webhook-release.json`]).equals(bytes))throw new Error();
    validateAncestry(m,process.env.GITHUB_SHA,git);
    mkdirSync(destination,{recursive:true,mode:0o700});
    writeFileSync(`${destination}/approved.json`,JSON.stringify(m),{mode:0o600});
    writeFileSync(`${destination}/webhook-runtime.tfvars.json`,JSON.stringify({project_id:m.project,project_number:m.projectNumber,region:'us-central1',runtime:{image:m.image,client_id_version:m.client_id_version,client_secret_version:m.client_secret_version}}),{mode:0o600});
    console.log('PASS: exact webhook rollout and source ancestry verified');
  }catch{console.error('BLOCKED: webhook rollout rejected');process.exitCode=1;}
}
