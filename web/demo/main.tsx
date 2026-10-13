import React, { useCallback, useState } from "react";
import { createRoot } from "react-dom/client";
import { Bookkeeping } from "../src/bookkeeping/Bookkeeping";
import { FlowViewer } from "./FlowViewer";
import { VolumeViewer } from "./VolumeViewer";
import "../src/app/style.css";

// Fictional identity exists only in this separate loopback review bundle.
const getToken = async () => "synthetic.browser.proof";
const rejectSession = () => {
  throw new Error("Demo session unavailable. Restart make demo.");
};

// Reuse production bookkeeping components against real SQL, with an explicit synthetic label.
function Demo() {
  const [view, setView] = useState<"manual" | "guided" | "volume">("manual");
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
          Fictional books use the real ledger. Generator Trading contains
          reviewed EUR journals; Cents Lab demonstrates exact cents. No bank
          accounts are connected. Changes are disposable; restart{" "}
          <code>make demo</code> to reset.
        </p>
      </details>
      <nav className="demo-nav" aria-label="Demo views">
        <button
          disabled={active}
          aria-pressed={view === "manual"}
          onClick={() => setView("manual")}
        >
          Manual books
        </button>
        <button
          disabled={active}
          aria-pressed={view === "guided"}
          onClick={() => setView("guided")}
        >
          Watch a few transactions
        </button>
        <button
          disabled={active}
          aria-pressed={view === "volume"}
          onClick={() => setView("volume")}
        >
          Run thousands
        </button>
      </nav>
      {active && (
        <p role="note">
          This run owns the demo session. Pausing stops new requests after the
          current operation finishes. A refresh can leave saved work to recover
          under Manual books.
        </p>
      )}
      {view === "manual" && (
        <Bookkeeping
          key={version}
          getToken={getToken}
          rejectSession={rejectSession}
        />
      )}
      {view === "guided" && (
        <FlowViewer onChanged={changed} onActiveChange={setActive} />
      )}
      {view === "volume" && (
        <VolumeViewer onChanged={changed} onActiveChange={setActive} />
      )}
    </main>
  );
}
createRoot(document.getElementById("root")!).render(<Demo />);
