import React, { useCallback, useState } from "react";
import { createRoot } from "react-dom/client";
import { Bookkeeping } from "../src/bookkeeping/Bookkeeping";
import { FlowViewer } from "./FlowViewer";
import { VolumeViewer } from "./VolumeViewer";
import "../src/app/style.css";

// Fictional identity exists only in this separate loopback review bundle.
const getToken = async () => "synthetic-browser-proof";
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
    <main>
      <header>
        <p className="eyebrow">JAYBURD / SYNTHETIC DEMO</p>
        <h1>Explore your demo books.</h1>
        <p>
          Fictional data · Real ledger validation · Nothing connects to your
          accounts.
        </p>
      </header>
      <section aria-label="Demo guide">
        <h2>Try the bookkeeping workflow</h2>
        <p>
          Generator Trading contains four reviewed EUR journals with a fictional
          rate of 1 USD per EUR. Cents Lab starts with 0.10 + 0.20 − 0.01 = 0.29
          USD in the bank. The source generator’s 15 unbalanced journals are
          excluded.
        </p>
        <p>
          Select a fictional entity and book, load its trial balance, or review
          and post a manual journal. Saved requests survive a page refresh.
        </p>
        <p>
          All changes are disposable. To reset, stop the demo with Ctrl+C and
          run <code>make demo</code> again.
        </p>
      </section>
      <nav aria-label="Demo views">
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
