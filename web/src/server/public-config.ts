import { parsePublicAuthConfig } from '../auth/config';
/** Return an explicit allowlist; no other runtime environment values enter the response. */
export function publicConfig(env: Record<string, string | undefined>): Response {
  const config = parsePublicAuthConfig({apiKey: env.JAYBURD_FIREBASE_API_KEY,
    authDomain: env.JAYBURD_FIREBASE_AUTH_DOMAIN, projectId: env.JAYBURD_FIREBASE_PROJECT_ID});
  return Response.json(config ?? {error: 'Sign-in is not configured'}, {
    status: config ? 200 : 503,
    headers: {'Cache-Control': 'no-store', 'X-Content-Type-Options': 'nosniff'},
  });
}
