import { createHash } from 'node:crypto';
import { validateProductionCandidate } from './production-release-candidate.mjs';

const repo = 'justin-napolitano/jayburd-bookkeeping';
const rejected = 'Production build proof rejected';
// Artifact records are closed to prevent ignored fields from hiding differing intent.
function exactKeys(value, names) {
  if (!value || typeof value !== 'object' || Array.isArray(value) ||
      Object.keys(value).sort().join(',') !== [...names].sort().join(',')) throw new Error(rejected);
}
// Compare public artifacts against separately retrieved trusted run metadata; never grant rollout authority.
export function validateProductionBuildProof(candidateBytes, approvedHash, bundleBytes, trustedRun, now = Date.now()) {
  try {
    const m = validateProductionCandidate(candidateBytes, approvedHash, now);
    if ((!Buffer.isBuffer(bundleBytes) && typeof bundleBytes !== 'string') || Buffer.byteLength(bundleBytes) > 32768 ||
        createHash('sha256').update(bundleBytes).digest('hex') !== m.evidence.build) throw new Error(rejected);
    const bundle = JSON.parse(bundleBytes);
    if (bundleBytes.toString() !== JSON.stringify(bundle) + '\n') throw new Error(rejected);
    exactKeys(bundle, ['schemaVersion','receipts']);
    const r = trustedRun;
    if (!r || r.path !== '.github/workflows/build-production-images.yml' || r.conclusion !== 'success' ||
        r.status !== 'completed' || r.event !== 'workflow_dispatch' || r.head_branch !== 'main' ||
        String(r.actor?.id) !== '55423093' || r.actor?.login !== 'justin-napolitano' ||
        String(r.triggering_actor?.id) !== '55423093' || r.triggering_actor?.login !== 'justin-napolitano' ||
        String(r.repository?.id) !== '1414515318' || r.repository?.full_name !== repo ||
        String(r.head_repository?.id) !== '1414515318' || r.head_repository?.full_name !== repo ||
        String(r.id) !== m.source.run || String(r.run_attempt) !== m.source.attempt || r.head_sha !== m.source.commit) throw new Error(rejected);
    if (bundle.schemaVersion !== 1 || !Array.isArray(bundle.receipts) || bundle.receipts.length !== 4 ||
        new Set(bundle.receipts.map(v=>v?.component)).size !== 4) throw new Error(rejected);
    for (const component of ['api','web','worker','migrate']) {
      const receipt = bundle.receipts.find(v=>v?.component === component);
      exactKeys(receipt,['schemaVersion','component','environment','image','commit','workflowRun','status','deployable','scan','runtimeTests']);
      exactKeys(receipt.scan,['severity','result']);
      const tests = component === 'worker' ? ['worker','webhook'] : [component];
      if (receipt.schemaVersion !== 1 || receipt.environment !== 'production' || receipt.image !== m.images[component] ||
          receipt.commit !== m.source.commit || receipt.workflowRun !== `https://github.com/${repo}/actions/runs/${m.source.run}/attempts/${m.source.attempt}` ||
          receipt.status !== 'candidate-only' || receipt.deployable !== false || receipt.scan.severity !== 'HIGH,CRITICAL' ||
          receipt.scan.result !== 'passed' || !Array.isArray(receipt.runtimeTests) ||
          JSON.stringify(receipt.runtimeTests) !== JSON.stringify(tests)) throw new Error(rejected);
    }
    return {status:'build-metadata-matched', deployable:false, commit:m.source.commit};
  } catch { throw new Error(rejected); }
}
