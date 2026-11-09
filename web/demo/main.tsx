import React, { useCallback, useState } from "react";
import { createRoot } from "react-dom/client";
import { Bookkeeping } from "../src/bookkeeping/Bookkeeping";
import { LoadDemo } from "./LoadDemo";
import { createAPI } from "../src/bookkeeping/api";
import "../src/app/style.css";

// Fictional identity exists only in this separate loopback review bundle.
const getToken = async () => "synthetic.browser.proof";
const rejectSession = () => {
  throw new Error("Demo session unavailable. Restart make demo.");
};

const demoAPI = createAPI(getToken, rejectSession);

// Reuse production bookkeeping components against real SQL, with an explicit synthetic label.
function Demo() {
  const [active, setActive] = useState(false);
  const [version, setVersion] = useState(0);
  const changed = useCallback(() => setVersion((value) => value + 1), []);
  return (
    <main className="app-shell">
      <header className="app-masthead">
        <p className="eyebrow">JAYBURD / SYNTHETIC DEMO</p>
        <h1>Books &amp; reports</h1>
        <p>
          Fictional data · Real ledger validation · Nothing connects to your
          accounts.
        </p>
      </header>
      <details className="demo-guide">
        <summary>About this synthetic workspace</summary>
        <p>
          Load demo books to generate 3,000 source transactions across three
          fictional LLCs. Review transactions and explicitly post entries to
          update balances. Historical synthetic balances demonstrate exact
          EUR/USD units and cents; new postings require an imported transaction.
          No bank accounts are connected. Changes are disposable; restart{" "}
          <code>make demo</code> to reset.
        </p>
      </details>
      {active && (
        <p role="note">
          This run owns the demo session. Pausing stops new requests after the
          current operation finishes. A refresh can leave saved work to recover
          in the bookkeeping workspace.
        </p>
      )}
      <LoadDemo
        api={demoAPI}
        locked={false}
        onComplete={changed}
        onActiveChange={setActive}
      />
      {!active && (
        <Bookkeeping
          key={version}
          getToken={getToken}
          rejectSession={rejectSession}
        />
      )}
    </main>
  );
}
createRoot(document.getElementById("root")!).render(<Demo />);
