import { initializeApp, getApps } from 'firebase/app';
import { initializeAuth, inMemoryPersistence, signInWithEmailAndPassword, signOut,
  multiFactor, TotpMultiFactorGenerator, getMultiFactorResolver, sendEmailVerification, reload,
  type Auth, type MultiFactorError, type MultiFactorResolver, type TotpSecret } from 'firebase/auth';
import type { AuthAdapter, LoginResult } from './controller';

import type { PublicAuthConfig } from './config';
/** Initialize only the memory persistence adapter, avoiding default browser storage. */
export function createBrowserAuth(config: Partial<PublicAuthConfig>): AuthAdapter | null {
  if (!config.apiKey || !config.authDomain || !config.projectId) return null;
  const name = 'jayburd-browser';
  const existing = getApps().find((candidate) => candidate.name === name);
  if (existing && (existing.options.apiKey !== config.apiKey || existing.options.projectId !== config.projectId || existing.options.authDomain !== config.authDomain)) throw new Error('Configuration changed.');
  const app = existing ?? initializeApp(config, name);
  return new FirebaseAdapter(initializeAuth(app, { persistence: inMemoryPersistence }));
}
/** Firebase owns authentication; this adapter only orchestrates managed TOTP flows. */
export class FirebaseAdapter implements AuthAdapter {
  private resolver: MultiFactorResolver | null = null;
  private secret: TotpSecret | null = null;
  /** Accept an explicitly initialized memory-only SDK instance. */
  constructor(private auth: Auth) {}
  /** Remove intermediate enrollment/challenge material without retaining user credentials. */
  clear() { this.resolver = null; this.secret = null; }
  /** Distinguish first-factor enrollment from a provider-issued TOTP challenge. */
  async signIn(email: string, password: string): Promise<LoginResult> {
    this.clear();
    try {
      const { user } = await signInWithEmailAndPassword(this.auth, email, password);
      if (!user.emailVerified) return 'verifying-email';
      if (multiFactor(user).enrolledFactors.length !== 0) throw new Error('Second factor required.');
      return 'enrolling';
    } catch (error) {
      if ((error as { code?: string })?.code !== 'auth/multi-factor-auth-required') throw new Error('Sign in failed.');
      const resolver = getMultiFactorResolver(this.auth, error as MultiFactorError);
      if (!resolver.hints.some((hint) => hint.factorId === TotpMultiFactorGenerator.FACTOR_ID)) throw new Error('TOTP required.');
      this.resolver = resolver;
      return 'challenge';
    }
  }
  /** Send only to the current unverified user, using Google's default hosted handler. */
  async sendVerification() {
    const user = this.auth.currentUser;
    if (!user || user.emailVerified) throw new Error('Verification unavailable.');
    await sendEmailVerification(user);
  }
  /** Reload provider truth; a stale or replaced SDK session cannot complete this step. */
  async checkVerification() {
    const user = this.auth.currentUser;
    if (!user) throw new Error('Sign in required.');
    await reload(user);
    if (this.auth.currentUser !== user) throw new Error('Sign in required.');
    return user.emailVerified;
  }
  /** Only fresh, verified first-factor sessions can request a new TOTP secret. */
  async beginEnrollment() {
    const user = this.auth.currentUser;
    if (!user?.emailVerified || multiFactor(user).enrolledFactors.length !== 0) throw new Error('Sign in required.');
    this.secret = await TotpMultiFactorGenerator.generateSecret(await multiFactor(user).getSession());
    return this.secret.secretKey;
  }
  /** Enrollment is not treated as an MFA-completed sign-in. */
  async enroll(code: string) {
    if (!this.secret || !this.auth.currentUser || !/^\d{6}$/.test(code)) throw new Error('Code required.');
    await multiFactor(this.auth.currentUser).enroll(TotpMultiFactorGenerator.assertionForEnrollment(this.secret, code), 'Jayburd authenticator');
    this.clear();
  }
  /** Try managed TOTP factors only; never accept an SMS challenge for bookkeeping. */
  async challenge(code: string) {
    if (!this.resolver || !/^\d{6}$/.test(code)) throw new Error('Code required.');
    const factors = this.resolver.hints.filter((hint) => hint.factorId === TotpMultiFactorGenerator.FACTOR_ID);
    // Enrollment currently creates one authenticator; multiple existing factors require operator recovery.
    if (factors.length !== 1) throw new Error('Authenticator recovery required.');
    await this.resolver.resolveSignIn(TotpMultiFactorGenerator.assertionForSignIn(factors[0].uid, code));
    await this.token();
    this.clear();
  }
  /** The claim check is a UX gate only; Go independently verifies signature and revocation. */
  async token() {
    const user = this.auth.currentUser;
    if (!user?.emailVerified) throw new Error('Sign in required.');
    const result = await user.getIdTokenResult();
    if (result.signInSecondFactor !== 'totp') throw new Error('TOTP required.');
    return result.token;
  }
  /** Signout clears SDK in-memory state and all intermediate MFA material. */
  async signOut() { this.clear(); await signOut(this.auth); }
}
