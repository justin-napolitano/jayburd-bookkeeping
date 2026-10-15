import {readFileSync,writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';

// Accept only the exact reviewed API image and one successful bounded API-identity execution.
export function databaseProbeReceipt(image, execution, name) {
  const project='jayburd-bookkeeping-dev', task=execution?.spec?.template?.spec, status=execution?.status;
  if(typeof image!=='string' || !/^us-central1-docker\.pkg\.dev\/jayburd-bookkeeping-dev\/jayburd-dev\/api@sha256:[a-f0-9]{64}$/.test(image) ||
    typeof name!=='string' || !/^jayburd-dev-database-probe-[a-z0-9]{1,20}$/.test(name) || execution?.metadata?.name!==name || execution?.spec?.taskCount!==1 ||
    task?.serviceAccountName!==`jayburd-dev-api@${project}.iam.gserviceaccount.com` || task?.maxRetries!==0 || ![120,'120'].includes(task?.timeoutSeconds) ||
    !Array.isArray(task?.containers) || task.containers.length!==1 || task.containers[0].image!==image ||
    JSON.stringify(task.containers[0].command)!==JSON.stringify(['/database-probe']) ||
    JSON.stringify(task.containers[0].args)!==JSON.stringify(['-action=probe',`-instance=${project}:us-central1:jayburd-dev`]) ||
    status?.succeededCount!==1 || ['failedCount','cancelledCount','runningCount','retriedCount'].some(k=>(status?.[k]??0)!==0) ||
    !Array.isArray(status?.conditions) || status.conditions.filter(c=>c.type==='Completed').length!==1 || !status.conditions.some(c=>c.type==='Completed' && c.status==='True') ||
    typeof status.completionTime!=='string' || !Number.isFinite(Date.parse(status.completionTime))) throw new Error('Database probe execution rejected');
  return {schemaVersion:1,status:'api-database-permission-proof-passed',project,image,execution:name,completedAt:status.completionTime,
    limitations:['No end-user identity, business-data, restore, load or real-bank acceptance.','Migration replay and clean-version/hash verification require separate receipts.']};
}

// The CLI consumes only selected public execution metadata, never logs or credentials.
if(process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href) {
  try {
    const receipt=databaseProbeReceipt(process.argv[2],JSON.parse(readFileSync(process.argv[3])),process.argv[5]);
    writeFileSync(process.argv[4],JSON.stringify(receipt,null,2)+'\n',{mode:0o600});
    console.log('PASS: exact API database probe execution validated');
  } catch {console.error('BLOCKED: database probe execution metadata rejected');process.exitCode=1;}
}
