'use client';

import { createContext, useContext, useEffect, useState, useSyncExternalStore, type FormEvent, type ReactNode } from 'react';
import { AuthController } from './controller';
import { createBrowserAuth } from './firebase';
import { parsePublicAuthConfig, type PublicAuthConfig } from './config';

/** Configuration availability is independent of verified authentication phases. */
type BootstrapPhase = 'loading' | 'unavailable' | 'invalid' | 'ready';
/** Bootstrap availability never grants identity or changes the controller's verified session rules. */
interface BootstrapState {
  source: PublicAuthConfig | undefined;
  attempt: number;
  phase: BootstrapPhase;
  controller: AuthController;
}
const AuthContext = createContext<{ controller: AuthController; bootstrapPhase: BootstrapPhase; retryConfiguration: () => void } | null>(null);
/** Production composition only; synthetic adapters are injected directly into controller tests. */
export function AuthProvider({ children, config: suppliedConfig }: { children: ReactNode; config?: PublicAuthConfig }) {
  const [empty] = useState(() => new AuthController(null));
  const [attempt, setAttempt] = useState(0);
  const [bootstrap, setBootstrap] = useState<BootstrapState>(() => ({source: suppliedConfig, attempt: 0, phase: 'loading', controller: empty}));
  useEffect(() => {
    const abort = new AbortController();
    let next: AuthController | undefined;
    // Publish only to the exact active source/attempt; cleanup fences even transports ignoring abort.
    const publish = (phase: BootstrapPhase, controller = empty) => {
      if (!abort.signal.aborted) setBootstrap({source: suppliedConfig, attempt, phase, controller});
    };
    publish('loading');
    void (async () => {
      try {
        let value: unknown = suppliedConfig;
        let requestSignal: AbortSignal | undefined;
        if (suppliedConfig === undefined) {
          requestSignal = AbortSignal.any([abort.signal, AbortSignal.timeout(5000)]);
          const response = await fetch('/api/auth-config', {cache: 'no-store', credentials: 'omit', redirect: 'error',
            signal: requestSignal});
          if (abort.signal.aborted) return;
          if (requestSignal.aborted) { publish('unavailable'); return; }
          if (!response.ok) { publish('unavailable'); return; }
          value = await response.json();
        }
        if (abort.signal.aborted) return;
        if (requestSignal?.aborted) { publish('unavailable'); return; }
        const config = parsePublicAuthConfig(value);
        if (!config) { publish('invalid'); return; }
        try {
          const adapter = createBrowserAuth(config);
          if (!adapter) { publish('invalid'); return; }
          next = new AuthController(adapter);
        } catch { publish('invalid'); return; }
        publish('ready', next);
      } catch { publish('unavailable'); }
    })();
    return () => { abort.abort(); if (next) void next.signOut(); };
  }, [suppliedConfig, attempt, empty]);
  // Mask an old signed-in controller immediately on source/retry change, before effect cleanup runs.
  const active = bootstrap.source === suppliedConfig && bootstrap.attempt === attempt;
  const bootstrapPhase = active ? bootstrap.phase : 'loading';
  const controller = active ? bootstrap.controller : empty;
  const retryConfiguration = () => { if (bootstrapPhase === 'unavailable') setAttempt((value) => value + 1); };
  return <AuthContext.Provider value={{controller, bootstrapPhase, retryConfiguration}}>{children}</AuthContext.Provider>;
}
/** Expose session boundaries without exposing Firebase users or token snapshots. */
export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('AuthProvider required.');
  const {controller, bootstrapPhase, retryConfiguration} = context;
  const state = useSyncExternalStore(controller.subscribe, controller.snapshot, controller.snapshot);
  return { ...state, bootstrapPhase, retryConfiguration, getToken: controller.getToken, signOut: controller.signOut, rejectSession: controller.rejectSession,
    signIn: controller.signIn, sendVerification: controller.sendVerification, checkVerification: controller.checkVerification,
    beginEnrollment: controller.beginEnrollment, enroll: controller.enroll, challenge: controller.challenge };
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
  if (auth.bootstrapPhase === 'loading') return <section className="panel" aria-busy="true"><h2>Preparing sign-in</h2><p role="status">Loading sign-in settings…</p></section>;
  if (auth.bootstrapPhase === 'unavailable') return <section className="panel"><h2>Sign-in is temporarily unavailable</h2><p role="alert">We could not load sign-in. Please try again.</p><button type="button" onClick={auth.retryConfiguration}>Retry sign-in</button></section>;
  if (auth.bootstrapPhase === 'invalid' || auth.phase === 'unconfigured') return <section className="panel"><h2>Sign-in is not configured</h2><p>This installation needs its public identity settings before you can sign in.</p></section>;
  if (auth.phase === 'signed-in') return null;
  return <section className="panel" aria-labelledby="signin-title">
    <h2 id="signin-title">{auth.phase === 'verifying-email' ? 'Verify your email' : auth.phase === 'enrolling' ? 'Set up your authenticator' : auth.phase === 'challenge' ? 'Verify your sign-in' : 'Sign in to Jayburd'}</h2>
    {auth.error && <p role="alert">{auth.error}</p>}
    {auth.notice && <p role="status">{auth.notice}</p>}
    {auth.phase === 'signed-out' && <p>Use your invited account. We will help you verify your email and set up an authenticator before accessing your books.</p>}
    {auth.phase === 'verifying-email' && <>
      <p>Verify your email before setting up your authenticator. Creating your account does not send this email; use the button below.</p>
      <button disabled={auth.busy} onClick={() => { void auth.sendVerification(); }}>Send verification email</button>
      <p>Open the link in your email, then return to this tab. If the link has expired, request another email.</p>
      <button disabled={auth.busy} onClick={() => { void auth.checkVerification(); }}>I have verified my email</button>
    </>}
    {auth.phase === 'enrolling' && !auth.enrollmentKey && <><p>Add a time-based authenticator before accessing your books.</p><button disabled={auth.busy} onClick={() => { void auth.beginEnrollment(); }}>Set up authenticator</button></>}
    {auth.phase === 'enrolling' && auth.enrollmentKey && <><p>In your authenticator, add a time-based account called Jayburd with this setup key. Keep it private.</p><p aria-label="Authenticator setup key"><code>{auth.enrollmentKey}</code></p></>}
    {(auth.phase === 'signed-out' || auth.phase === 'challenge' || (auth.phase === 'enrolling' && auth.enrollmentKey)) && <form onSubmit={submit}>
      {auth.phase === 'signed-out' ? <>
        <label>Email<input type="email" autoComplete="username" value={email} onChange={(e) => setEmail(e.target.value)} required maxLength={254} disabled={auth.busy} /></label>
        <label>Password<input type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required maxLength={4096} disabled={auth.busy} /></label>
      </> : <label>Authenticator code<input type="text" inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" maxLength={6} value={code} onChange={(e) => setCode(e.target.value)} required disabled={auth.busy} /></label>}
      <button type="submit" disabled={auth.busy}>{auth.busy ? 'Please wait…' : auth.phase === 'signed-out' ? 'Sign in' : 'Verify code'}</button>
    </form>}
    {(auth.phase === 'verifying-email' || auth.phase === 'enrolling' || auth.phase === 'challenge' || auth.busy) && <button type="button" onClick={() => { void auth.signOut(); }}>Cancel and sign out</button>}
    {(auth.phase === 'enrolling' || auth.phase === 'challenge') && <p>Lost access? Contact your operator for identity recovery.</p>}
  </section>;
}
