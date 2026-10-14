import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
const repo = 'justin-napolitano/jayburd-bookkeeping';
const components = ['api', 'migrate', 'web'];
// Exact object fields prevent arbitrary Terraform variables, environment values or destinations entering promotion.
function keys(value, names) {
  if (!value || typeof value !== 'object' || Array.isArray(value) || Object.keys(value).sort().join(',') !== [...names].sort().join(',')) throw new Error('Release rejected');
}
// Validate only public identifiers and fixed Sandbox policy before generating any Terraform input.
// Hosted Link does not require ordinary-SDK enablement; retain its explicit boolean in the digest.
export function validateConnections(c, projectNumber, releaseExpiry, now) {
  if (c === null) return;
  keys(c, ['infisical_origin','infisical_project','infisical_identity','link_events_qualified','client_id_version','client_secret_version','consent_expires_at']);
  if (!['https://us.infisical.com','https://eu.infisical.com','https://app.infisical.com'].includes(c.infisical_origin) || typeof c.link_events_qualified !== 'boolean') throw new Error('Release rejected');
  for (const id of [c.infisical_project,c.infisical_identity]) if (typeof id !== 'string' || !/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/.test(id) || id === '00000000-0000-0000-0000-000000000000') throw new Error('Release rejected');
  for (const [key, name] of [['client_id_version','client-id'],['client_secret_version','client-secret']]) {
    const prefix = `projects/${projectNumber}/secrets/jayburd-dev-bank-sync-${name}/versions/`;
    if (typeof c[key] !== 'string' || !c[key].startsWith(prefix)) throw new Error('Release rejected');
    const version = c[key].slice(prefix.length);
    if (!/^[1-9][0-9]{0,18}$/.test(version) || BigInt(version) > 9223372036854775807n) throw new Error('Release rejected');
  }
  const expiry = Date.parse(c.consent_expires_at);
  if (typeof c.consent_expires_at !== 'string' || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(c.consent_expires_at) || !Number.isFinite(expiry) ||
      new Date(expiry).toISOString().replace('.000Z','Z') !== c.consent_expires_at || expiry <= now || expiry <= releaseExpiry || expiry > now + 30*86400000) throw new Error('Release rejected');
}
// Stable digest binds approved metadata independent of object key order; it is not observed-provider evidence.
export function connectionConfigDigest(c) {
  if (c !== null) keys(c, ['infisical_origin','infisical_project','infisical_identity','link_events_qualified','client_id_version','client_secret_version','consent_expires_at']);
  const normalized = c === null ? null : Object.fromEntries(Object.keys(c).sort().map(k=>[k,c[k]]));
  return createHash('sha256').update(JSON.stringify(normalized)).digest('hex');
}
// Owner dispatch approves these exact public bytes, not an unsigned build artifact by itself.
export function validateManifest(bytes, expectedHash, now = Date.now()) {
  if (!/^[a-f0-9]{64}$/.test(expectedHash) || createHash('sha256').update(bytes).digest('hex') !== expectedHash) throw new Error('Release rejected');
  const m = JSON.parse(bytes);
  const migrationOnly = ['migrate','verify'].includes(m?.operation);
  keys(m, migrationOnly ? ['schemaVersion','project','projectNumber','operation','build','images','evidence','expiresAt'] : ['schemaVersion','project','projectNumber','operation','build','images','firebaseApiKey','evidence','expiresAt','previousDeployment','connections']);
  keys(m.build, ['run','attempt','commit']); keys(m.images, components);
  keys(m.evidence, migrationOnly ? ['bootstrap','compatibility'] : ['bootstrap','identity','secrets','compatibility']);
  if (m.schemaVersion !== (migrationOnly ? 1 : 2) || !/^[a-z][a-z0-9-]{4,28}[a-z0-9]$/.test(m.project) || !/^[1-9][0-9]{5,19}$/.test(m.projectNumber) ||
      !['promote','rollback','migrate','verify'].includes(m.operation) || (!migrationOnly && !/^[A-Za-z0-9_-]{1,128}$/.test(m.firebaseApiKey)) ||
      !/^[1-9][0-9]{0,19}$/.test(m.build.run) || !/^[1-9][0-9]{0,3}$/.test(m.build.attempt) || !/^[a-f0-9]{40}$/.test(m.build.commit)) throw new Error('Release rejected');
  if ([m.project,m.projectNumber,...(migrationOnly ? [] : [m.firebaseApiKey]),m.build.run,m.build.attempt,m.build.commit].some(v=>typeof v!=='string') ||
      typeof m.expiresAt!=='string' || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(m.expiresAt)) throw new Error('Release rejected');
  if (BigInt(m.projectNumber) > 9223372036854775807n) throw new Error('Release rejected');
  const expiry = Date.parse(m.expiresAt);
  if (!Number.isFinite(expiry) || expiry <= now || expiry > now + 7*86400000) throw new Error('Release rejected');
  for (const component of components) {
    const prefix = `us-central1-docker.pkg.dev/${m.project}/jayburd-dev/${component}@`;
    if (typeof m.images[component] !== 'string' || !m.images[component].startsWith(prefix) || !/^sha256:[a-f0-9]{64}$/.test(m.images[component].slice(prefix.length))) throw new Error('Release rejected');
  }
  for (const value of Object.values(m.evidence)) if (typeof value !== 'string' || !/^https:\/\/github\.com\/justin-napolitano\/jayburd-bookkeeping\/issues\/[1-9][0-9]*#issuecomment-[0-9]+$/.test(value)) throw new Error('Release rejected');
  if (!migrationOnly && (m.operation === 'rollback' ? typeof m.previousDeployment!=='string' || !/^[1-9][0-9]{0,19}$/.test(m.previousDeployment) : m.previousDeployment !== null)) throw new Error('Release rejected');
  if (!migrationOnly) validateConnections(m.connections, m.projectNumber, expiry, now);
  return m;
}
// Migration and read-only status gather database evidence; full service promotion retains every closure gate.
export function requiredGates(operation) {
  if (['migrate','verify'].includes(operation)) return {issues:[5],prs:[44]};
  if (['promote','rollback'].includes(operation)) return {issues:[6,11,26],prs:[23,24]};
  throw new Error('Release rejected');
}
// GitHub run metadata and all three retained receipts must agree with the independently approved manifest.
export function validateBuild(m, run, receipts) {
  if (String(run.id) !== m.build.run || String(run.run_attempt) !== m.build.attempt || run.head_sha !== m.build.commit || run.head_branch !== 'main' ||
      run.event !== 'workflow_dispatch' || run.path !== '.github/workflows/build-images.yml' || run.conclusion !== 'success' ||
      String(run.actor?.id) !== '55423093' || run.triggering_actor?.login !== 'justin-napolitano' || String(run.repository?.id) !== '1414515318') throw new Error('Build rejected');
  if (!Array.isArray(receipts) || receipts.length !== 3 || new Set(receipts.map(r=>r.component)).size !== 3) throw new Error('Build rejected');
  for (const c of components) {
    const r = receipts.find(value=>value.component === c);
    if (!r || r.schemaVersion !== 1 || r.image !== m.images[c] || r.commit !== m.build.commit ||
        r.workflowRun !== `https://github.com/${repo}/actions/runs/${m.build.run}/attempts/${m.build.attempt}` || r.status !== 'candidate-only' || r.deployable !== false) throw new Error('Build rejected');
  }
}
// Rollback targets must be evidenced by a successful earlier deployment; SQL compatibility still needs owner review.
export function validateRollback(m, run, receipt) {
  if (m.operation !== 'rollback' || String(run.id) !== m.previousDeployment || run.path !== '.github/workflows/deploy-dev.yml' || run.conclusion !== 'success' ||
      run.event !== 'workflow_dispatch' || run.head_branch !== 'main' || String(run.actor?.id) !== '55423093' || run.triggering_actor?.login !== 'justin-napolitano' || String(run.repository?.id) !== '1414515318' ||
      receipt.status !== 'transport-smoke-passed' || receipt.project !== m.project || receipt.images?.api !== m.images.api || receipt.images?.web !== m.images.web || receipt.approvedConnectionConfigSHA256 !== connectionConfigDigest(m.connections)) throw new Error('Rollback rejected');
}
// Only named public configuration reaches generated inputs. Never read credentials, plans or state.
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const [manifest, hash, destination, evidence] = process.argv.slice(2);
    const m = validateManifest(readFileSync(manifest), hash);
    if (evidence) {
      const json = file => JSON.parse(readFileSync(`${evidence}/${file}`, 'utf8'));
      validateBuild(m, json('build-run.json'), components.map(c=>json(`${c}/candidate.json`)));
      if (m.operation === 'rollback') validateRollback(m, json('previous-run.json'), json('previous/deployment.json'));
    }
    mkdirSync(destination, {recursive:true, mode:0o700});
    const common = {project_id:m.project, region:'us-central1'};
    if (!['migrate','verify'].includes(m.operation)) writeFileSync(`${destination}/runtime.tfvars.json`, JSON.stringify({...common, project_number:m.projectNumber, images:{api:m.images.api,web:m.images.web}, firebase_api_key:m.firebaseApiKey, connections:m.connections}), {mode:0o600});
    writeFileSync(`${destination}/migration.tfvars.json`, JSON.stringify({...common, image:m.images.migrate, action:m.operation==='verify'?'status':'up'}), {mode:0o600});
    writeFileSync(`${destination}/approved.json`, JSON.stringify(m), {mode:0o600});
    console.log('PASS: release metadata validated');
  } catch { console.error('BLOCKED: release metadata or build evidence rejected'); process.exitCode=1; }
}
