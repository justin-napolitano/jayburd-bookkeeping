import {readFileSync,writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';

// Bind the invoked immutable action and image; Cloud Run encodes its int64 timeout as a decimal string.
export function migrationReceipt(m, execution, name) {
  const task=execution?.spec?.template?.spec;
  const status=execution?.status;
  const expectedArgs=[m.operation==='verify'?'-action=status':'-action=up',`-instance=${m.project}:us-central1:jayburd-dev`];
  if (!['migrate','promote','verify'].includes(m.operation) || typeof name!=='string' || !/^jayburd-dev-migrate-[a-z0-9]{1,20}$/.test(name) ||
      execution?.metadata?.name!==name || execution?.spec?.taskCount!==1 ||
      task?.serviceAccountName!==`jayburd-dev-migrator@${m.project}.iam.gserviceaccount.com` ||
      task?.maxRetries!==0 || ![660,'660'].includes(task?.timeoutSeconds) || !Array.isArray(task?.containers) || task.containers.length!==1 ||
      task.containers[0].image!==m.images.migrate || (task.containers[0].command?.length??0)!==0 || JSON.stringify(task.containers[0].args)!==JSON.stringify(expectedArgs) ||
      status?.succeededCount!==1 || ['failedCount','cancelledCount','runningCount','retriedCount'].some(k=>(status?.[k]??0)!==0) ||
      !Array.isArray(status?.conditions) || status.conditions.filter(c=>c.type==='Completed').length!==1 ||
      !status.conditions.some(c=>c.type==='Completed' && c.status==='True') ||
      typeof status.completionTime!=='string' || !Number.isFinite(Date.parse(status.completionTime))) throw new Error('Migration execution rejected');
  return {schemaVersion:1,status:m.operation==='verify'?'migration-verification-passed':'migration-execution-passed',project:m.project,operation:m.operation,build:m.build,
    image:m.images.migrate,execution:name,completedAt:status.completionTime,evidence:m.evidence,
    limitations:['No API/identity/restore/load acceptance.',m.operation==='verify'?'Read-only version/hash check; replay remains separate.':'SQL version and replay require separate evidence.']};
}
// Read selected execution metadata only; never upload the provider object, environment, logs or state.
if(process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href) {
  try {
    const receipt=migrationReceipt(JSON.parse(readFileSync(process.argv[2])),JSON.parse(readFileSync(process.argv[3])),process.argv[5]);
    writeFileSync(process.argv[4],JSON.stringify(receipt,null,2)+'\n',{mode:0o600});
    console.log('PASS: exact migration execution validated');
  } catch {console.error('BLOCKED: migration execution metadata rejected');process.exitCode=1;}
}
