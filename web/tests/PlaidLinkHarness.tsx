import { useEffect, useRef, useState } from "react";
import { createPlaidLink } from "../src/bookkeeping/plaidLink";
// Synthetic browser harness composes the actual adapter with an intercepted SDK only.
export function PlaidLinkHarness() {
  const link = useRef(createPlaidLink());
  const abort = useRef(new AbortController());
  const [status, setStatus] = useState("idle");
  useEffect(() => {
    const current = abort.current;
    return () => current.abort();
  }, []);
  return (
    <section>
      <button
        onClick={() => {
          setStatus("opening");
          void link.current
            .open({ linkToken: "synthetic-link", signal: abort.current.signal })
            .then(
              (result) => setStatus(result ? "handoff ready" : "cancelled"),
              () => setStatus("failed"),
            );
        }}
      >
        Connect synthetic bank
      </button>
      <button onClick={() => abort.current.abort()}>Cancel session</button>
      <p role="status">{status}</p>
    </section>
  );
}
