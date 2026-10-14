import { readFileSync, writeFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

const digest = /^sha256:[a-f0-9]{64}$/;
// Bind registry identity to the exact locally tested image and retain only non-secret build evidence.
export function receipt(input, build, repositories) {
  const { project, component, commit, run, attempt, imageID } = input;
  if (!/^[a-z][a-z0-9-]{4,28}[a-z0-9]$/.test(project) ||
      !['api', 'migrate', 'web', 'worker'].includes(component) || !/^[a-f0-9]{40}$/.test(commit) ||
      !/^[0-9]+$/.test(run) || !/^[0-9]+$/.test(attempt) || !digest.test(imageID)) throw new Error('Invalid build identity');
  // Docker's classic store uses a configuration ID; containerd may expose the manifest/index ID.
  const builtIDs = [build['containerimage.config.digest'], build['containerimage.digest']].filter(value => digest.test(value));
  if (!builtIDs.includes(imageID) ||
      !build['buildx.build.provenance'] || typeof build['buildx.build.provenance'] !== 'object') throw new Error('Missing or mismatched build evidence');
  const repository = `us-central1-docker.pkg.dev/${project}/jayburd-dev/${component}`;
  const matches = repositories.filter(value => typeof value === 'string' && value.startsWith(repository + '@') && digest.test(value.slice(repository.length + 1)));
  if (matches.length !== 1) throw new Error('Ambiguous published digest');
  return {
    schemaVersion: 1, component, commit, image: matches[0], testedImageID: imageID, buildDigest: build['containerimage.digest'] ?? null,
    configDigest: build['containerimage.config.digest'] ?? null,
    workflowRun: `https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/${run}/attempts/${attempt}`,
    evidence: { build: 'build.json', vulnerabilities: 'vulnerabilities.json', sbom: 'sbom.cdx.json' },
    status: 'candidate-only', deployable: false,
    limitations: ['Unsigned build metadata; not a SLSA attestation.', 'Requires separate identity, ingress, migration and rollout acceptance.'],
  };
}
// CLI reads only generated receipt files and explicitly named public CI identifiers, never credential stores.
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const input = { project: process.env.PROJECT_ID, component: process.env.COMPONENT, commit: process.env.GITHUB_SHA,
    run: process.env.GITHUB_RUN_ID, attempt: process.env.GITHUB_RUN_ATTEMPT, imageID: process.env.IMAGE_ID };
  const value = receipt(input, JSON.parse(readFileSync('receipts/build.json', 'utf8')), JSON.parse(readFileSync('receipts/repo-digests.json', 'utf8')));
  writeFileSync('receipts/candidate.json', JSON.stringify(value, null, 2) + '\n');
}
