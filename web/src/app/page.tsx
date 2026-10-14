"use client";
import { AuthProvider, LoginPanel, useAuth } from "../auth";
import { Bookkeeping } from "../bookkeeping/Bookkeeping";
import { createPlaidLink } from "../bookkeeping/plaidLink";
// Inert until the user explicitly begins managed bank setup; demos inject their own adapter.
const managedConnectionLink = createPlaidLink();
// Session replacement unmounts every financial view and pending request.
function Application() {
  const auth = useAuth();
  return (
    <main className={auth.phase === "signed-in" ? "app-shell" : undefined}>
      <header
        className={auth.phase === "signed-in" ? "app-masthead" : undefined}
      >
        <p className="eyebrow">JAYBURD / BOOKKEEPING</p>
        <h1>
          {auth.phase === "signed-in"
            ? "Books & reports"
            : "Your books, with a clear record."}
        </h1>
        <p>Manual entries. Explicit review. Immutable posted journals.</p>
      </header>
      {auth.phase === "signed-in" ? (
        <>
          <p role="note">
            Saved commands are recovered after sign-in. Review and acknowledge
            the previous result before preparing new work.
          </p>
          <button onClick={() => void auth.signOut()}>Sign out</button>
          <Bookkeeping
            key={auth.sessionKey}
            getToken={auth.getToken}
            rejectSession={auth.rejectSession}
            connectionLink={managedConnectionLink}
          />
        </>
      ) : (
        <LoginPanel />
      )}
      <footer>
        Development bookkeeping · Bank setup requires configured Sandbox access
        · Live-use accounting policy approval remains required.
      </footer>
    </main>
  );
}
export default function Page() {
  return (
    <AuthProvider>
      <Application />
    </AuthProvider>
  );
}
