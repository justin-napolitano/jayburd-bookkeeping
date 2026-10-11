/** Browser state is presentation only; the API independently verifies every token. */
export type Phase = 'unconfigured' | 'signed-out' | 'loading' | 'enrolling' | 'challenge' | 'signed-in';
export type LoginResult = 'enrolling' | 'challenge' | 'signed-in';
/** The SDK seam keeps provider credentials out of UI state and offline test fixtures. */
export interface AuthAdapter {
  signIn(email: string, password: string): Promise<LoginResult>;
  beginEnrollment(): Promise<string>;
  enroll(code: string): Promise<void>;
  challenge(code: string): Promise<void>;
  token(): Promise<string>;
  signOut(): Promise<void>;
  clear(): void;
}
export interface AuthState {
  phase: Phase;
  busy: boolean;
  error: string;
  enrollmentKey: string;
  sessionKey: string;
}
/** An epoch suppresses late completions after logout, rejection or provider teardown. */
export class AuthController {
  private epoch = 0;
  private pending = false;
  private signingOut = 0;
  private listeners = new Set<() => void>();
  private state: AuthState;
  /** Missing provider configuration starts in an explicit unavailable state. */
  constructor(private adapter: AuthAdapter | null) {
    this.state = { phase: adapter ? 'signed-out' : 'unconfigured', busy: false, error: '', enrollmentKey: '', sessionKey: '0' };
  }
  /** React observes stable snapshots, never SDK objects or bearer tokens. */
  snapshot = () => this.state;
  /** Subscriptions are removable so remounts do not retain application state. */
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  /** Replace state atomically before notifying subscribers. */
  private update(patch: Partial<AuthState>) { this.state = { ...this.state, ...patch }; this.listeners.forEach((f) => f()); }
  /** Keep a cancelled SDK operation from authenticating a later UI session. */
  private async run(work: () => Promise<Partial<AuthState>>, resetOnFailure = false) {
    if (!this.adapter || this.pending || this.signingOut !== 0) return;
    const epoch = this.epoch;
    this.pending = true;
    this.update({ busy: true, error: '' });
    try {
      const result = await work();
      if (epoch === this.epoch) this.update(result);
    } catch {
      if (epoch === this.epoch) {
        if (resetOnFailure) {
          this.adapter.clear();
          await this.adapter.signOut().catch(() => {});
        }
        this.update({ error: 'Unable to authenticate. Check your details or sign in again.', ...(resetOnFailure ? { phase: 'signed-out' as const, enrollmentKey: '' } : {}) });
      }
    } finally {
      if (epoch !== this.epoch) await this.adapter.signOut().catch(() => {});
      this.pending = false;
      this.update({ busy: this.signingOut !== 0 });
    }
  }
  /** Only operator-provisioned accounts can authenticate; no signup method exists. */
  signIn = (email: string, password: string) => this.run(async () => ({ phase: await this.adapter!.signIn(email, password), enrollmentKey: '' }), true);
  /** Generate the enrollment secret only after the user explicitly requests it. */
  beginEnrollment = () => this.run(async () => ({ enrollmentKey: await this.adapter!.beginEnrollment() }));
  /** Enrollment always requires a new password/TOTP sign-in before API access. */
  enroll = (code: string) => this.run(async () => {
    await this.adapter!.enroll(code);
    await this.adapter!.signOut();
    return { phase: 'signed-out', enrollmentKey: '', error: 'Authenticator added. Sign in again to complete verification.' };
  });
  /** Only a successfully completed TOTP challenge opens the application UI. */
  challenge = (code: string) => this.run(async () => { await this.adapter!.challenge(code); return { phase: 'signed-in' }; });
  /** Clear UI and enrollment material immediately, even when provider signout fails. */
  signOut = async () => {
    ++this.epoch;
    ++this.signingOut;
    this.adapter?.clear();
    this.update({ phase: this.adapter ? 'signed-out' : 'unconfigured', enrollmentKey: '', error: '', sessionKey: String(this.epoch), busy: true });
    await this.adapter?.signOut().catch(() => {});
    --this.signingOut;
    this.update({ busy: this.pending || this.signingOut !== 0 });
  };
  /** A backend rejection invalidates all financial UI state without displaying provider errors. */
  rejectSession = () => { void this.signOut(); };
  /** Fetch fresh SDK-managed tokens per request and reject results from obsolete sessions. */
  getToken = async () => {
    if (!this.adapter || this.state.phase !== 'signed-in') throw new Error('Sign in required.');
    const epoch = this.epoch;
    try {
      const token = await this.adapter.token();
      if (epoch !== this.epoch || this.state.phase !== 'signed-in') throw new Error('Sign in required.');
      return token;
    } catch {
      if (epoch === this.epoch) await this.signOut();
      throw new Error('Sign in required.');
    }
  };
}
