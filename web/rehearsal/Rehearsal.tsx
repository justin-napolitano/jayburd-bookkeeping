import { useState, type FormEvent } from 'react';
import { AuthProvider, LoginPanel, useAuth } from '../src/auth';
import { parsePublicAuthConfig, type PublicAuthConfig } from '../src/auth/config';

/** Show only browser MFA completion; never fetch/display a token or mount financial views. */
function IdentityResult() {
  const auth = useAuth();
  if (auth.phase !== 'signed-in') return <LoginPanel />;
  return <section className="panel"><h2>Authenticator sign-in completed</h2>
    <p>Your managed sign-in completed. Bookkeeping access has not been tested.</p>
    <button onClick={() => { void auth.signOut(); }}>Sign out</button>
  </section>;
}
/** Public identifiers stay in component memory; refresh clears configuration and sign-in. */
export function Rehearsal() {
  const [apiKey, setApiKey] = useState('');
  const [config, setConfig] = useState<PublicAuthConfig>();
  const [error, setError] = useState('');
  /** Accept only the fixed development project and a syntactically valid public browser key. */
  function configure(event: FormEvent) {
    event.preventDefault();
    const value = parsePublicAuthConfig({apiKey: apiKey.trim(), projectId: 'jayburd-bookkeeping-dev', authDomain: 'jayburd-bookkeeping-dev.firebaseapp.com'});
    if (!value || !/^AIza[A-Za-z0-9_-]{35}$/.test(value.apiKey)) { setError('Enter the public Firebase web API key for the development project.'); return; }
    setApiKey(''); setError(''); setConfig(value);
  }
  return <main><header><p className="eyebrow">JAYBURD / LOCAL SIGN-IN REHEARSAL</p>
    <h1>Verify your sign-in.</h1><p>Verify your email, add an authenticator, then sign in again.</p></header>
    {!config ? <section className="panel"><h2>Connect development sign-in</h2>
      <p>Project: jayburd-bookkeeping-dev</p>
      <p>Open Identity Platform → Providers → Application setup details and copy the apiKey field. This is the browser identifier, not a password or service-account key.</p>
      <form onSubmit={configure}><label>Public Firebase web API key<input value={apiKey} onChange={event => setApiKey(event.target.value)} autoComplete="off" maxLength={128} required /></label>
        <button type="submit">Continue to sign in</button></form>
      {error && <p role="alert">{error}</p>}
    </section> : <AuthProvider config={config}><IdentityResult /></AuthProvider>}
    <footer>Local identity check · No bookkeeping or bank connection · Refresh to clear this session</footer>
  </main>;
}
