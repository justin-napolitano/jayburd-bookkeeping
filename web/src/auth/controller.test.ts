import { describe, expect, it, vi } from 'vitest';
import { AuthController, type AuthAdapter } from './controller';

/** All adapter outcomes are synthetic; these tests cannot initialize a real provider. */
function fixture() {
  const adapter: AuthAdapter = { signIn: vi.fn(async () => 'challenge' as const), beginEnrollment: vi.fn(async () => 'SYNTHETIC-KEY'), enroll: vi.fn(async () => {}), challenge: vi.fn(async () => {}), token: vi.fn(async () => 'synthetic-token'), signOut: vi.fn(async () => {}), clear: vi.fn() };
  return { adapter, controller: new AuthController(adapter) };
}
/** Control completion order without timers or guessed network delays. */
function deferred<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>((r) => { resolve = r; }); return { resolve, promise }; }

describe('AuthController', () => {
  it('fails visibly without configuration and cannot issue tokens', async () => {
    const c = new AuthController(null);
    expect(c.snapshot().phase).toBe('unconfigured');
    await expect(c.getToken()).rejects.toThrow('Sign in required.');
  });
  it('requires a TOTP challenge before issuing a token', async () => {
    const { controller: c } = fixture();
    await c.signIn('synthetic@example.invalid', 'synthetic-password');
    expect(c.snapshot().phase).toBe('challenge');
    await expect(c.getToken()).rejects.toThrow();
    await c.challenge('123456');
    expect(await c.getToken()).toBe('synthetic-token');
  });
  it('clears enrollment material and requires fresh signin after enrollment', async () => {
    const { controller: c, adapter } = fixture();
    vi.mocked(adapter.signIn).mockResolvedValue('enrolling');
    await c.signIn('synthetic@example.invalid', 'synthetic-password');
    expect(c.snapshot().enrollmentKey).toBe('');
    await c.beginEnrollment();
    expect(c.snapshot().enrollmentKey).toBe('SYNTHETIC-KEY');
    await c.enroll('123456');
    expect(c.snapshot().enrollmentKey).toBe('');
    expect(c.snapshot().phase).toBe('signed-out');
    expect(adapter.signOut).toHaveBeenCalled();
    await expect(c.getToken()).rejects.toThrow();
  });
  it('suppresses late signin and prevents a new signin during cancellation', async () => {
    const { controller: c, adapter } = fixture();
    const result = deferred<'challenge'>();
    vi.mocked(adapter.signIn).mockReturnValue(result.promise);
    const pending = c.signIn('synthetic@example.invalid', 'synthetic-password');
    await c.signOut();
    await c.signIn('other@example.invalid', 'synthetic-password');
    expect(adapter.signIn).toHaveBeenCalledTimes(1);
    result.resolve('challenge'); await pending;
    expect(c.snapshot().phase).toBe('signed-out');
    expect(c.snapshot().busy).toBe(false);
    expect(c.snapshot().sessionKey).toBe('1');
    expect(adapter.signOut).toHaveBeenCalledTimes(2);
  });
  it('does not race a new signin against provider signout', async () => {
    const { controller: c, adapter } = fixture();
    const result = deferred<void>();
    vi.mocked(adapter.signOut).mockReturnValue(result.promise);
    const pending = c.signOut();
    await c.signIn('synthetic@example.invalid', 'synthetic-password');
    expect(adapter.signIn).not.toHaveBeenCalled();
    result.resolve(); await pending;
    await c.signIn('synthetic@example.invalid', 'synthetic-password');
    expect(adapter.signIn).toHaveBeenCalledTimes(1);
  });
  it('keeps new signin blocked until every concurrent signout completes', async () => {
    const { controller: c, adapter } = fixture();
    const first = deferred<void>(), second = deferred<void>();
    vi.mocked(adapter.signOut).mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise);
    const a = c.signOut(), b = c.signOut();
    first.resolve(); await a;
    await c.signIn('synthetic@example.invalid', 'synthetic-password');
    expect(adapter.signIn).not.toHaveBeenCalled();
    expect(c.snapshot().busy).toBe(true);
    second.resolve(); await b;
    expect(c.snapshot().busy).toBe(false);
    await c.signIn('synthetic@example.invalid', 'synthetic-password');
    expect(adapter.signIn).toHaveBeenCalledTimes(1);
  });
  it('suppresses enrollment key returned after cancellation', async () => {
    const { controller: c, adapter } = fixture();
    const result = deferred<string>();
    vi.mocked(adapter.beginEnrollment).mockReturnValue(result.promise);
    const pending = c.beginEnrollment(); await c.signOut();
    result.resolve('SYNTHETIC-KEY'); await pending;
    expect(c.snapshot().enrollmentKey).toBe('');
  });
  it('never releases a refreshed token after logout or backend rejection', async () => {
    const { controller: c, adapter } = fixture();
    await c.challenge('123456');
    const token = deferred<string>(); vi.mocked(adapter.token).mockReturnValue(token.promise);
    const pending = c.getToken(); c.rejectSession(); token.resolve('synthetic-token');
    await expect(pending).rejects.toThrow('Sign in required.');
    expect(c.snapshot().phase).toBe('signed-out');
  });
  it('clears state for expired/revoked identity and never exposes provider failures', async () => {
    const { controller: c, adapter } = fixture();
    await c.challenge('123456');
    vi.mocked(adapter.token).mockRejectedValue(new Error('sensitive provider message'));
    await expect(c.getToken()).rejects.toThrow('Sign in required.');
    expect(c.snapshot().phase).toBe('signed-out');
    vi.mocked(adapter.signIn).mockRejectedValue(new Error('sensitive provider message'));
    await c.signIn('synthetic@example.invalid', 'synthetic-password');
    expect(JSON.stringify(c.snapshot())).not.toContain('sensitive');
  });
});
