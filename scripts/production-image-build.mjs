import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
const failure = 'Production image build rejected';
const digest = value => typeof value === 'string' && /^sha256:[a-f0-9]{64}$/.test(value);
const integer = value => typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) && BigInt(value)<=9223372036854775807n;
// A separately reviewed exact-byte public target bounds manual publication, never deployment.
export function buildTarget(bytes, hash, now=Date.now()) {
 try {
  if ((!Buffer.isBuffer(bytes)&&typeof bytes!=='string') || Buffer.byteLength(bytes)>4096 || !/^[a-f0-9]{64}$/.test(hash) || createHash('sha256').update(bytes).digest('hex')!==hash || !Number.isFinite(now)) throw Error();
  const t=JSON.parse(bytes);
  if (!t || Object.keys(t).sort().join(',')!=='expiresAt,mandate,project,projectNumber,schemaVersion' || bytes.toString()!==JSON.stringify(t)+'\n' || t.schemaVersion!==1 || typeof t.project!=='string' || !/^[a-z][a-z0-9-]{4,28}[a-z0-9]$/.test(t.project) || !integer(t.projectNumber) || t.projectNumber.length<6 || ((t.project==='jayburd-bookkeeping-dev') !== (t.projectNumber==='9088794002')) || typeof t.mandate!=='string' || !/^https:\/\/github\.com\/justin-napolitano\/jayburd-bookkeeping\/issues\/337#issuecomment-[1-9][0-9]*$/.test(t.mandate)) throw Error();
  const expiry=Date.parse(t.expiresAt);
  if (typeof t.expiresAt!=='string' || !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/.test(t.expiresAt) || !Number.isFinite(expiry) || new Date(expiry).toISOString().replace('.000Z','Z')!==t.expiresAt || expiry<=now || expiry>now+7*86400000) throw Error();
  return t;
 } catch { throw Error(failure); }
}
// Matching local build identity and published digest are retained alongside the closed candidate receipt.
export function productionImageReceipt(input, build, repositories, scan) {
 try {
  const {project,component,commit,run,attempt,imageID,targetHash}=input;
  if (typeof project!=='string' || !/^[a-z][a-z0-9-]{4,28}[a-z0-9]$/.test(project) || !['api','web','worker','migrate'].includes(component) || !/^[a-f0-9]{40}$/.test(commit) || !integer(run) || !integer(attempt) || !digest(imageID) || !/^[a-f0-9]{64}$/.test(targetHash)) throw Error();
  if (![build['containerimage.config.digest'],build['containerimage.digest']].filter(digest).includes(imageID) || !build['buildx.build.provenance'] || typeof build['buildx.build.provenance']!=='object') throw Error();
  if (scan?.SchemaVersion!==2 || !Array.isArray(scan.Results) || scan.Results.some(r=> !r || (r.Vulnerabilities!==undefined && (!Array.isArray(r.Vulnerabilities) || r.Vulnerabilities.some(v=>!v || typeof v.Severity!=='string' || ['HIGH','CRITICAL'].includes(v.Severity)))))) throw Error();
  const prefix=`us-central1-docker.pkg.dev/${project}/jayburd-prod/${component}@`;
  const matches=repositories.filter(v=>typeof v==='string' && v.startsWith(prefix) && digest(v.slice(prefix.length)));
  if(matches.length!==1) throw Error();
  return {
   candidate:{schemaVersion:1,component,environment:'production',image:matches[0],commit,workflowRun:`https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/${run}/attempts/${attempt}`,status:'candidate-only',deployable:false,scan:{severity:'HIGH,CRITICAL',result:'passed'},runtimeTests:component==='worker'?['worker','webhook']:[component]},
   binding:{testedImageID:imageID,buildDigest:build['containerimage.digest']??null,configDigest:build['containerimage.config.digest']??null,targetSHA256:targetHash,limitations:['Unsigned build metadata; trusted workflow and independent ancestry verification required.']},
  };
 } catch { throw Error(failure); }
}
// CLI reads only public target/build files and named CI identifiers; it never reads credential files.
if(process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href) {
 try {
  const [mode,path,hash]=process.argv.slice(2);
  if(process.argv.length!==5 || !['target','receipt'].includes(mode)) throw Error();
  const t=buildTarget(readFileSync(path),hash);
  if(mode==='target') console.log(JSON.stringify(t));
  else {
   const json=p=>JSON.parse(readFileSync(p,'utf8'));
   const result=productionImageReceipt({project:t.project,component:process.env.COMPONENT,commit:process.env.GITHUB_SHA,run:process.env.GITHUB_RUN_ID,attempt:process.env.GITHUB_RUN_ATTEMPT,imageID:process.env.IMAGE_ID,targetHash:hash},json('receipts/build.json'),json('receipts/repo-digests.json'),json('receipts/vulnerabilities.json'));
   writeFileSync('receipts/candidate.json',JSON.stringify(result.candidate)+'\n');
   writeFileSync('receipts/production-binding.json',JSON.stringify(result.binding)+'\n');
  }
 } catch { console.error(failure);process.exitCode=1; }
}
