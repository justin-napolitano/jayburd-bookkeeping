'use client';

import { createContext, useContext, useEffect, useState, useSyncExternalStore, type FormEvent, type ReactNode } from 'react';
import { AuthController } from './controller';
import { createBrowserAuth } from './firebase';

const AuthContext = createContext<AuthController | null>(null);
/** Production composition only; synthetic adapters are injected directly into controller tests. */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [controller, setController] = useState(() => new AuthController(null));
  useEffect(() => {
    let next: AuthController;
    try {
      next = new AuthController(createBrowserAuth({
        apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY,
        authDomain: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN,
        projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID,
      }));
    } catch { next = new AuthController(null); }
    setController(next);
    return () => { void next.signOut(); };
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
