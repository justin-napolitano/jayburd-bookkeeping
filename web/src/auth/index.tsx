'use client';

import { createContext, useContext, useEffect, useState, useSyncExternalStore, type FormEvent, type ReactNode } from 'react';
import { AuthController } from './controller';
import { createBrowserAuth } from './firebase';
import { parsePublicAuthConfig } from './config';

const AuthContext = createContext<AuthController | null>(null);
/** Production composition only; synthetic adapters are injected directly into controller tests. */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [controller, setController] = useState(() => new AuthController(null));
  useEffect(() => {
    const abort = new AbortController();
    let next: AuthController | undefined;
    // Configuration is fetched once per mount; late responses cannot recreate a signed-out session.
    void (async () => {
      try {
        const response = await fetch('/api/auth-config', {cache: 'no-store', credentials: 'omit', redirect: 'error',
          signal: AbortSignal.any([abort.signal, AbortSignal.timeout(5000)])});
        if (!response.ok) return;
        const config = parsePublicAuthConfig(await response.json());
        if (abort.signal.aborted || !config) return;
        next = new AuthController(createBrowserAuth(config));
        setController(next);
      } catch { /* Configuration failures keep sign-in unavailable without exposing raw errors. */ }
    })();
    return () => { abort.abort(); if (next) void next.signOut(); };
  }, []);
  return <AuthContext.Provider value={controller}>{children}</AuthContext.Provider>;
}
/** Expose session boundaries without exposing Firebase users or token snapshots. */
export function useAuth() {
  const controller = useContext(AuthContext);
  if (!controller) throw new Error('AuthProvider required.');
  const state = useSyncExternalStore(controller.subscribe, controller.snapshot, controller.snapshot);
  return { ...state, getToken: controller.getToken, signOut: controller.signOut, rejectSession: controller.rejectSession,
    signIn: controller.signIn, beginEnrollment: controller.beginEnrollment, enroll: controller.enroll, challenge: controller.challenge };
}
/** Accessible invitation-only credentials/MFA UI; sensitive inputs clear immediately on submit. */
export function LoginPanel() {
  const auth = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  useEffect(() => { setPassword(''); setCode(''); setEmail(''); }, [auth.phase, auth.sessionKey]);
  // React state never receives provider error text, signed tokens or resolver objects.
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (auth.busy) return;
    if (auth.phase === 'signed-out') { const value = password; setPassword(''); void auth.signIn(email, value); }
    else { const value = code; setCode(''); void (auth.phase === 'enrolling' ? auth.enroll(value) : auth.challenge(value)); }
  };
  if (auth.phase === 'unconfigured') return <section className="panel"><h2>Sign-in is not configured</h2><p>This installation needs its public identity settings before you can sign in.</p></section>;
  if (auth.phase === 'signed-in') return null;
  return <section className="panel" aria-labelledby="signin-title">
    <h2 id="signin-title">{auth.phase === 'enrolling' ? 'Set up your authenticator' : auth.phase === 'challenge' ? 'Verify your sign-in' : 'Sign in to Jayburd'}</h2>
    {auth.error && <p role="alert">{auth.error}</p>}
    {auth.phase === 'signed-out' && <p>Use your invited, verified account. New accounts are provisioned by your operator.</p>}
    {auth.phase === 'enrolling' && !auth.enrollmentKey && <><p>Add a time-based authenticator before accessing your books.</p><button disabled={auth.busy} onClick={() => { void auth.beginEnrollment(); }}>Set up authenticator</button></>}
    {auth.phase === 'enrolling' && auth.enrollmentKey && <><p>In your authenticator, add a time-based account called Jayburd with this setup key. Keep it private.</p><p aria-label="Authenticator setup key"><code>{auth.enrollmentKey}</code></p></>}
    {(auth.phase === 'signed-out' || auth.phase === 'challenge' || (auth.phase === 'enrolling' && auth.enrollmentKey)) && <form onSubmit={submit}>
      {auth.phase === 'signed-out' ? <>
        <label>Email<input type="email" autoComplete="username" value={email} onChange={(e) => setEmail(e.target.value)} required maxLength={254} disabled={auth.busy} /></label>
        <label>Password<input type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required maxLength={4096} disabled={auth.busy} /></label>
      </> : <label>Authenticator code<input type="text" inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" maxLength={6} value={code} onChange={(e) => setCode(e.target.value)} required disabled={auth.busy} /></label>}
      <button type="submit" disabled={auth.busy}>{auth.busy ? 'Please wait…' : auth.phase === 'signed-out' ? 'Sign in' : 'Verify code'}</button>
    </form>}
    {(auth.phase === 'enrolling' || auth.phase === 'challenge' || auth.busy) && <button type="button" onClick={() => { void auth.signOut(); }}>Cancel and sign out</button>}
    {(auth.phase === 'enrolling' || auth.phase === 'challenge') && <p>Lost access? Contact your operator for identity recovery.</p>}
  </section>;
}
