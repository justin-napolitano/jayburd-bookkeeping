import { beforeEach, describe, expect, it, vi } from 'vitest';

const sdk = vi.hoisted(() => ({
  initializeApp: vi.fn(() => ({ name: 'jayburd-browser' })), getApps: vi.fn(() => []),
  initializeAuth: vi.fn(), inMemoryPersistence: { type: 'NONE' },
  signInWithEmailAndPassword: vi.fn(), signOut: vi.fn(async () => {}), multiFactor: vi.fn(),
  sendEmailVerification: vi.fn(async () => {}), reload: vi.fn(async () => {}), getMultiFactorResolver: vi.fn(), generateSecret: vi.fn(), assertionForEnrollment: vi.fn(), assertionForSignIn: vi.fn(),
}));
vi.mock('firebase/app', () => ({ initializeApp: sdk.initializeApp, getApps: sdk.getApps }));
vi.mock('firebase/auth', () => ({ ...sdk, TotpMultiFactorGenerator: { FACTOR_ID: 'totp', generateSecret: sdk.generateSecret, assertionForEnrollment: sdk.assertionForEnrollment, assertionForSignIn: sdk.assertionForSignIn } }));
import { createBrowserAuth, FirebaseAdapter } from './firebase';
import type { Auth } from 'firebase/auth';

/** Firebase calls are mocked; fixtures contain no provider credentials or real secrets. */
function fixture() {
  const user = { emailVerified: true, getIdTokenResult: vi.fn(async () => ({ signInSecondFactor: 'totp', token: 'synthetic-token' })) };
  const factors = { enrolledFactors: [], getSession: vi.fn(async () => 'synthetic-session'), enroll: vi.fn(async () => {}) };
  sdk.multiFactor.mockReturnValue(factors);
  sdk.signInWithEmailAndPassword.mockResolvedValue({ user });
  const adapter = new FirebaseAdapter({ currentUser: user } as unknown as Auth);
  return { user, factors, adapter };
}

beforeEach(() => vi.clearAllMocks());
describe('FirebaseAdapter boundary', () => {
  it('does not initialize an unconfigured provider; configured persistence is memory only', () => {
    expect(createBrowserAuth({})).toBeNull(); expect(sdk.initializeApp).not.toHaveBeenCalled();
    createBrowserAuth({ apiKey: 'synthetic-public', projectId: 'synthetic-project', authDomain: 'synthetic.example.invalid' });
    expect(sdk.initializeAuth).toHaveBeenCalledWith(expect.anything(), { persistence: sdk.inMemoryPersistence });
  });
  it('requires verified email and never upgrades password signin to MFA', async () => {
    const { user, adapter } = fixture();
    expect(await adapter.signIn('fixture@example.invalid', 'synthetic')).toBe('enrolling');
    user.emailVerified = false;
    expect(await adapter.signIn('fixture@example.invalid', 'synthetic')).toBe('verifying-email');
    expect(sdk.sendEmailVerification).not.toHaveBeenCalled();
    await expect(adapter.beginEnrollment()).rejects.toThrow();
    await expect(adapter.token()).rejects.toThrow();
  });
  it('generates enrollment material only on explicit request and clears it', async () => {
    const { adapter, factors } = fixture();
    sdk.generateSecret.mockResolvedValue({ secretKey: 'SYNTHETIC-KEY' });
    await adapter.signIn('fixture@example.invalid', 'synthetic');
    expect(sdk.generateSecret).not.toHaveBeenCalled();
    expect(await adapter.beginEnrollment()).toBe('SYNTHETIC-KEY');
    adapter.clear(); await expect(adapter.enroll('123456')).rejects.toThrow();
    await adapter.beginEnrollment(); await adapter.enroll('123456');
    expect(factors.enroll).toHaveBeenCalledTimes(1);
    await expect(adapter.enroll('123456')).rejects.toThrow();
  });
  it('requires a managed TOTP challenge and a TOTP token result', async () => {
    const { adapter, user } = fixture();
    sdk.signInWithEmailAndPassword.mockRejectedValue({ code: 'auth/multi-factor-auth-required', message: 'private provider text' });
    const resolveSignIn = vi.fn(async () => ({}));
    sdk.getMultiFactorResolver.mockReturnValue({ hints: [{ factorId: 'totp', uid: 'synthetic-factor' }], resolveSignIn });
    expect(await adapter.signIn('fixture@example.invalid', 'synthetic')).toBe('challenge');
    await adapter.challenge('123456');
    expect(resolveSignIn).toHaveBeenCalledTimes(1);
    expect(await adapter.token()).toBe('synthetic-token');
    user.getIdTokenResult.mockResolvedValue({ signInSecondFactor: 'phone', token: 'synthetic-token' });
    await expect(adapter.token()).rejects.toThrow('TOTP required.');
  });
  it('rejects unsupported factors and never exposes provider errors', async () => {
    const { adapter } = fixture();
    sdk.signInWithEmailAndPassword.mockRejectedValue({ code: 'auth/multi-factor-auth-required' });
    sdk.getMultiFactorResolver.mockReturnValue({ hints: [{ factorId: 'phone', uid: 'synthetic-factor' }] });
    await expect(adapter.signIn('fixture@example.invalid', 'synthetic')).rejects.toThrow('TOTP required.');
    sdk.signInWithEmailAndPassword.mockRejectedValue(new Error('private provider text'));
    await expect(adapter.signIn('fixture@example.invalid', 'synthetic')).rejects.toThrow('Sign in failed.');
  });
});

it('sends only for the current unverified user and reloads provider verification', async () => {
  const { user, adapter } = fixture();
  await expect(adapter.sendVerification()).rejects.toThrow();
  user.emailVerified = false;
  await adapter.sendVerification();
  expect(sdk.sendEmailVerification).toHaveBeenCalledWith(user);
  expect(await adapter.checkVerification()).toBe(false);
  sdk.reload.mockImplementationOnce(async () => { user.emailVerified = true; });
  expect(await adapter.checkVerification()).toBe(true);
  expect(sdk.reload).toHaveBeenCalledWith(user);
  const absent = new FirebaseAdapter({ currentUser: null } as unknown as Auth);
  await expect(absent.sendVerification()).rejects.toThrow();
  await expect(absent.checkVerification()).rejects.toThrow();
});
it('rejects a user replaced during verification refresh', async () => {
  const { user } = fixture();
  const auth = { currentUser: user };
  const adapter = new FirebaseAdapter(auth as unknown as Auth);
  sdk.reload.mockImplementationOnce(async () => { auth.currentUser = { ...user }; });
  await expect(adapter.checkVerification()).rejects.toThrow('Sign in required.');
});
