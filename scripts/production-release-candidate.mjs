import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

const failure = 'Production candidate rejected';
const digest = value => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);
// Closed objects exclude hidden configuration and accidental credential values.
function keys(value, names) {
  if (!value || typeof value !== 'object' || Array.isArray(value) ||
      Object.keys(value).sort().join(',') !== [...names].sort().join(',')) throw new Error(failure);
}
// Resource versions and identifiers remain canonical decimal strings.
function integer(value) {
  return typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) && BigInt(value) <= 9223372036854775807n;
}
// Strict UTC timestamps reject normalized invalid dates and alternate encodings.
function timestamp(value) {
  if (typeof value !== 'string' || !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/.test(value)) throw new Error(failure);
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed) || new Date(parsed).toISOString().replace('.000Z', 'Z') !== value) throw new Error(failure);
  return parsed;
}
// Offline structural validation grants no cloud, provider, migration or posting authority.
export function validateProductionCandidate(bytes, approvedHash, now = Date.now()) {
  try {
    if ((!Buffer.isBuffer(bytes) && typeof bytes !== 'string') || Buffer.byteLength(bytes) > 32768 ||
        !digest(approvedHash) || createHash('sha256').update(bytes).digest('hex') !== approvedHash || !Number.isFinite(now)) throw new Error(failure);
    const m = JSON.parse(bytes);
    // One compact encoding prevents duplicate-key and parser/whitespace ambiguity.
    if (bytes.toString() !== JSON.stringify(m) + '\n') throw new Error(failure);
    keys(m, ['schemaVersion','environment','deployable','operation','project','projectNumber','region','createdAt','expiresAt','scheduling','source','images','database','secrets','keys','evidence']);
    if (m.schemaVersion !== 1 || m.environment !== 'production' || m.deployable !== false || m.operation !== 'prepare' ||
        m.scheduling !== 'paused' || m.region !== 'us-central1' || typeof m.project !== 'string' ||
        !/^[a-z][a-z0-9-]{4,28}[a-z0-9]$/.test(m.project) ||
        !integer(m.projectNumber) || m.projectNumber.length < 6 || ((m.project === 'jayburd-bookkeeping-dev') !== (m.projectNumber === '9088794002'))) throw new Error(failure);
    const created = timestamp(m.createdAt), expires = timestamp(m.expiresAt);
    if (created > now || expires <= now || expires <= created || expires - created > 7 * 86400000) throw new Error(failure);
    keys(m.source, ['commit','run','attempt']);
    if (typeof m.source.commit !== 'string' || !/^[a-f0-9]{40}$/.test(m.source.commit) || !integer(m.source.run) || !integer(m.source.attempt)) throw new Error(failure);
    keys(m.images, ['api','web','worker','migrate']);
    for (const component of Object.keys(m.images)) {
      const prefix = `us-central1-docker.pkg.dev/${m.project}/jayburd-prod/${component}@sha256:`;
      if (typeof m.images[component] !== 'string' || !m.images[component].startsWith(prefix) || !digest(m.images[component].slice(prefix.length))) throw new Error(failure);
    }
    keys(m.database, ['instance','name','migration','checksumsSHA256']);
    if (m.database.instance !== (m.project === 'jayburd-bookkeeping-dev' ? `${m.project}:us-central1:jayburd-dev` : `${m.project}:us-central1:jayburd-prod`) || m.database.name !== (m.project === 'jayburd-bookkeeping-dev' ? 'bookkeeping_prod' : 'bookkeeping') ||
        m.database.migration !== '35' || !digest(m.database.checksumsSHA256)) throw new Error(failure);
    keys(m.secrets, ['clientID','clientSecret']);
    for (const [field, suffix] of [['clientID','id'],['clientSecret','secret']]) {
      const prefix = `projects/${m.projectNumber}/secrets/jayburd-prod-bank-sync-client-${suffix}/versions/`;
      if (typeof m.secrets[field] !== 'string' || !m.secrets[field].startsWith(prefix) || !integer(m.secrets[field].slice(prefix.length))) throw new Error(failure);
    }
    keys(m.keys, ['bank','link']);
    const keyPrefix = `projects/${m.project}/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/`;
    if (m.keys.bank !== `${keyPrefix}bank-credentials` || m.keys.link !== `${keyPrefix}hosted-link`) throw new Error(failure);
    keys(m.evidence, ['mandate','build','schema','identity','secrets','recovery','qualification']);
    if (!Object.values(m.evidence).every(digest)) throw new Error(failure);
    return m;
  } catch { throw new Error(failure); }
}
// CLI prints only a fixed outcome; it never echoes submitted metadata or creates rollout inputs.
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    if (process.argv.length !== 4) throw new Error(failure);
    validateProductionCandidate(readFileSync(process.argv[2]), process.argv[3]);
    console.log('PASS: offline Production candidate; deployment not authorized');
  } catch { console.error(failure); process.exitCode = 1; }
}
