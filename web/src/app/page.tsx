"use client";
import { AuthProvider, LoginPanel, useAuth } from "../auth";
import { Bookkeeping } from "../bookkeeping/Bookkeeping";
// Session replacement unmounts every financial view and pending request.
function Application() {
  const auth = useAuth();
  return (
    <main>
      <header>
        <p className="eyebrow">JAYBURD / BOOKKEEPING</p>
        <h1>Your books, with a clear record.</h1>
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
          />
        </>
      ) : (
        <LoginPanel />
      )}
      <footer>
        Development bookkeeping · No bank connection · Live-use accounting
        policy approval remains required.
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
