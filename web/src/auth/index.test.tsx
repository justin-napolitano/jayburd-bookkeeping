import { beforeEach, afterEach, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createElement } from 'react';
import type { AuthAdapter } from './controller';
const boundary = vi.hoisted(() => ({ createBrowserAuth: vi.fn() }));
vi.mock('./firebase', () => boundary);
import { AuthProvider, LoginPanel, useAuth } from './index';

/** This fixture replaces the provider boundary only in test-owned module loading. */
function fixture(): AuthAdapter {
  return { signIn: vi.fn(async () => 'enrolling' as const), sendVerification: vi.fn(async () => {}), checkVerification: vi.fn(async () => false), beginEnrollment: vi.fn(async () => 'SYNTHETIC-KEY'), enroll: vi.fn(async () => {}), challenge: vi.fn(async () => {}), token: vi.fn(async () => 'synthetic-token'), signOut: vi.fn(async () => {}), clear: vi.fn() };
}
/** Expose only public session state as a test probe, never credentials. */
function Probe() { const auth = useAuth(); return createElement('output', { 'data-testid': 'phase', role: 'note' }, auth.phase); }
/** Construct the same provider/component composition as the real application. */
function mount() { render(createElement(AuthProvider, null, createElement(LoginPanel), createElement(Probe))); }
beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal('fetch', vi.fn(async () => Response.json({apiKey:'synthetic-public',projectId:'synthetic-project',authDomain:'synthetic-project.firebaseapp.com'})));
});
afterEach(() => {cleanup(); vi.unstubAllGlobals();});
it('renders an unavailable sign-in screen without configuration', () => {
  boundary.createBrowserAuth.mockReturnValue(null); mount();
  expect(screen.getByText('Sign-in is not configured')).toBeTruthy();
  expect(screen.queryByLabelText('Password')).toBeNull();
});
it('shows the key only after explicit enrollment and removes it on cancel', async () => {
  const adapter = fixture(); boundary.createBrowserAuth.mockReturnValue(adapter);
  // Flush runtime configuration and the session-reset effect before entering credentials.
  await act(async () => { mount(); });
  await screen.findByLabelText('Email');
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'fixture@example.invalid' } });
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'synthetic-password' } });
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
  expect(adapter.signIn).toHaveBeenCalledWith('fixture@example.invalid', 'synthetic-password');
  await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('enrolling'));
  expect(screen.queryByText('SYNTHETIC-KEY')).toBeNull();
  expect(adapter.beginEnrollment).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Set up authenticator' }));
  await waitFor(() => expect(screen.getByText('SYNTHETIC-KEY')).toBeTruthy());
  fireEvent.click(screen.getByRole('button', { name: 'Cancel and sign out' }));
  await waitFor(() => expect(screen.getByTestId('phase').textContent).toBe('signed-out'));
  expect(screen.queryByText('SYNTHETIC-KEY')).toBeNull();
  expect((screen.getByLabelText('Password') as HTMLInputElement).value).toBe('');
});
it('does not render provider errors or leave a submitted password in the input', async () => {
  const adapter = fixture(); vi.mocked(adapter.signIn).mockRejectedValue(new Error('private provider text'));
  boundary.createBrowserAuth.mockReturnValue(adapter);
  // Await provider configuration and field-reset effects before interacting.
  await act(async () => { mount(); });
  await screen.findByLabelText('Email');
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'fixture@example.invalid' } });
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'synthetic-password' } });
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
  expect(adapter.signIn).toHaveBeenCalledWith('fixture@example.invalid', 'synthetic-password');
  await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy());
  expect(screen.getByRole('alert').textContent).not.toContain('private provider text');
  expect((screen.getByLabelText('Password') as HTMLInputElement).value).toBe('');
});

it('does not initialize an SDK after unmount or a rejected runtime configuration', async () => {
  let resolve!: (response: Response) => void;
  vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>(done => {resolve=done;})));
  const view=render(createElement(AuthProvider,null,createElement(LoginPanel)));
  view.unmount();
  resolve(Response.json({apiKey:'synthetic-public',projectId:'synthetic-project',authDomain:'synthetic-project.firebaseapp.com'}));
  await new Promise(done => setTimeout(done,0));
  expect(boundary.createBrowserAuth).not.toHaveBeenCalled();
  vi.stubGlobal('fetch', vi.fn(async () => new Response('',{status:503})));
  mount();
  await new Promise(done => setTimeout(done,0));
  expect(boundary.createBrowserAuth).not.toHaveBeenCalled();
});

it('offers explicit verification and returns verified users to a fresh login', async () => {
  const adapter = fixture(); vi.mocked(adapter.signIn).mockResolvedValue('verifying-email');
  boundary.createBrowserAuth.mockReturnValue(adapter);
  // Await provider configuration and field-reset effects before interacting.
  await act(async () => { mount(); });
  await screen.findByLabelText('Email');
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'fixture@example.invalid' } });
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'synthetic-password' } });
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
  expect(adapter.signIn).toHaveBeenCalledWith('fixture@example.invalid', 'synthetic-password');
  await screen.findByRole('heading', { name: 'Verify your email' });
  expect(adapter.sendVerification).not.toHaveBeenCalled();
  expect(screen.queryByLabelText('Password')).toBeNull();
  expect(screen.queryByRole('button', { name: 'Set up authenticator' })).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Send verification email' }));
  await waitFor(() => expect(screen.getByRole('status').textContent).toContain('Verification email requested'));
  fireEvent.click(screen.getByRole('button', { name: 'I have verified my email' }));
  await waitFor(() => expect(screen.getByRole('status').textContent).toContain('not verified yet'));
  vi.mocked(adapter.checkVerification).mockResolvedValue(true);
  fireEvent.click(screen.getByRole('button', { name: 'I have verified my email' }));
  await screen.findByLabelText('Password');
  expect(screen.getByRole('status').textContent).toContain('Email verified');
  expect(screen.getByTestId('phase').textContent).toBe('signed-out');
});

it('uses validated explicit public configuration without fetching a runtime endpoint', async () => {
  boundary.createBrowserAuth.mockReturnValue(fixture());
  render(createElement(AuthProvider, {children: null,config:{apiKey:'synthetic-public',projectId:'synthetic-project',authDomain:'synthetic-project.firebaseapp.com'}}, createElement(LoginPanel)));
  await screen.findByLabelText('Email');
  expect(fetch).not.toHaveBeenCalled();
  expect(boundary.createBrowserAuth).toHaveBeenCalledOnce();
});
it('fails closed for invalid supplied configuration without trying another source', async () => {
  render(createElement(AuthProvider, {children: null,config:{apiKey:'synthetic-public',projectId:'synthetic-project',authDomain:'foreign.example.invalid'}}, createElement(LoginPanel)));
  await new Promise(done => setTimeout(done,0));
  expect(screen.getByText('Sign-in is not configured')).toBeTruthy();
  expect(fetch).not.toHaveBeenCalled(); expect(boundary.createBrowserAuth).not.toHaveBeenCalled();
});
