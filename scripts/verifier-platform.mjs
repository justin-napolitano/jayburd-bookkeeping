import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,mkdirSync} from 'node:fs';
import {pathToFileURL} from 'node:url';

// Owner reviews exact canonical metadata bytes; arbitrary variables, secrets and resource destinations are rejected.
export function validatePlatform(bytes,hash) {
  if(!/^[a-f0-9]{64}$/.test(hash)||createHash('sha256').update(bytes).digest('hex')!==hash) throw Error('Verifier platform rejected');
  const m=JSON.parse(bytes);
  if(JSON.stringify(m,null,2)+'\n'!==bytes.toString()||Object.keys(m).sort().join(',')!=='enabled,project_id,region,run'||m.project_id!=='jayburd-bookkeeping-dev'||m.region!=='us-central1'||m.enabled!==true) throw Error('Verifier platform rejected');
  if(m.run!==null && (!m.run||Array.isArray(m.run)||Object.keys(m.run).sort().join(',')!=='image,run_id'||typeof m.run.image!=='string'||!/^us-central1-docker[.]pkg[.]dev\/jayburd-bookkeeping-dev\/jayburd-dev\/worker@sha256:[a-f0-9]{64}$/.test(m.run.image)||typeof m.run.run_id!=='string'||!/^verify-[a-z0-9]{8,40}$/.test(m.run.run_id))) throw Error('Verifier platform rejected');
  return m;
}

// Generate metadata only, outside Git. This module never authenticates, plans, applies or executes a job.
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href) {
  try {
    const [file,hash,destination]=process.argv.slice(2);
    const m=validatePlatform(readFileSync(file),hash);
    mkdirSync(destination,{recursive:true,mode:0o700});
    writeFileSync(`${destination}/reviewed.tfvars.json`,JSON.stringify(m,null,2)+'\n',{mode:0o600});
    console.log('PASS: exact verifier platform metadata validated');
  } catch {console.error('BLOCKED: verifier platform metadata rejected');process.exitCode=1;}
}
