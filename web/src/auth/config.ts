/** Only these public identifiers may cross from runtime configuration into the browser. */
export interface PublicAuthConfig { apiKey: string; authDomain: string; projectId: string }
/** Reject partial, extra or malformed configuration before initializing a provider. */
export function parsePublicAuthConfig(value: unknown): PublicAuthConfig | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null;
  const config = value as Record<string, unknown>;
  if (Object.keys(config).sort().join(',') !== 'apiKey,authDomain,projectId') return null;
  const { apiKey, authDomain, projectId } = config;
  if (typeof apiKey !== 'string' || !/^[A-Za-z0-9_-]{1,128}$/.test(apiKey) ||
      typeof projectId !== 'string' || !/^[a-z][a-z0-9-]{4,28}[a-z0-9]$/.test(projectId) ||
      typeof authDomain !== 'string' || authDomain !== `${projectId}.firebaseapp.com`) return null;
  return { apiKey, authDomain, projectId };
}
