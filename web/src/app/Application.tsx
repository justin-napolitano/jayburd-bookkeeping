"use client";

import { AuthProvider, LoginPanel, useAuth } from "../auth";
import { Bookkeeping } from "../bookkeeping/Bookkeeping";
import { createPlaidLink } from "../bookkeeping/plaidLink";

// Inert until the user explicitly begins managed bank setup; demos inject their own adapter.
const managedConnectionLink = createPlaidLink();

// Session replacement unmounts every financial view and pending request.
function AuthenticatedApplication() {
  const auth = useAuth();
  return (
    <main className={auth.phase === "signed-in" ? "app-shell" : undefined}>
      {auth.phase !== "signed-in" && (
        <header>
          <p className="login-brand">Jayburd</p>
          <h1>Your books, with a clear record.</h1>
          <p>
            Connect your accounts, review activity and keep every entry linked
            to its evidence.
          </p>
        </header>
      )}
      {auth.phase === "signed-in" ? (
        <>
          <Bookkeeping
            key={auth.sessionKey}
            getToken={auth.getToken}
            rejectSession={auth.rejectSession}
            connectionLink={managedConnectionLink}
            onSignOut={() => void auth.signOut()}
          />
        </>
      ) : (
        <LoginPanel />
      )}
      <footer className="application-footer">
        <span>Jayburd</span>
        <span>Accounting preview</span>
      </footer>
    </main>
  );
}

// The provider boundary keeps verified identity state scoped to one application tree.
export function Application() {
  return (
    <AuthProvider>
      <AuthenticatedApplication />
    </AuthProvider>
  );
}
