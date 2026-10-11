import { beforeEach, afterEach, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createElement } from 'react';
import type { AuthAdapter } from './controller';
const boundary = vi.hoisted(() => ({ createBrowserAuth: vi.fn() }));
vi.mock('./firebase', () => boundary);
import { AuthProvider, LoginPanel, useAuth } from './index';

/** This fixture replaces the provider boundary only in test-owned module loading. */
function fixture(): AuthAdapter {
  return { signIn: vi.fn(async () => 'enrolling' as const), beginEnrollment: vi.fn(async () => 'SYNTHETIC-KEY'), enroll: vi.fn(async () => {}), challenge: vi.fn(async () => {}), token: vi.fn(async () => 'synthetic-token'), signOut: vi.fn(async () => {}), clear: vi.fn() };
}
/** Expose only public session state as a test probe, never credentials. */
function Probe() { const auth = useAuth(); return createElement('output', { 'data-testid': 'phase' }, auth.phase); }
/** Construct the same provider/component composition as the real application. */
function mount() { render(createElement(AuthProvider, null, createElement(LoginPanel), createElement(Probe))); }
beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);
it('renders an unavailable sign-in screen without configuration', () => {
  boundary.createBrowserAuth.mockReturnValue(null); mount();
  expect(screen.getByText('Sign-in is not configured')).toBeTruthy();
  expect(screen.queryByLabelText('Password')).toBeNull();
});
it('shows the key only after explicit enrollment and removes it on cancel', async () => {
  const adapter = fixture(); boundary.createBrowserAuth.mockReturnValue(adapter); mount();
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'fixture@example.invalid' } });
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'synthetic-password' } });
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
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
  boundary.createBrowserAuth.mockReturnValue(adapter); mount();
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'fixture@example.invalid' } });
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'synthetic-password' } });
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
  await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy());
  expect(screen.getByRole('alert').textContent).not.toContain('private provider text');
  expect((screen.getByLabelText('Password') as HTMLInputElement).value).toBe('');
});
