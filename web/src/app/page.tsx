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
            Keep this page open while a posting outcome is unknown. Signing out
            clears pending requests; verify the earlier request key with an
            authorized operator before entering it again.
          </p>
          <button
            onClick={() => {
              if (
                window.confirm(
                  "Sign out and clear this session? If a posting outcome is unknown, save its request key and verify whether it committed before entering it again.",
                )
              )
                void auth.signOut();
            }}
          >
            Sign out
          </button>
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
        If your session ended during an unconfirmed request, verify that request
        with an authorized operator before re-entering it. Development
        bookkeeping · No bank connection · Live-use accounting policy approval
        remains required.
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
