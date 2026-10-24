import { AuthProvider, LoginPanel } from "../src/auth";

/** Exercise production public bootstrap and memory-only SDK initialization without authenticating a user. */
export function AuthBootstrapHarness() {
  return (
    <AuthProvider>
      <LoginPanel />
    </AuthProvider>
  );
}
