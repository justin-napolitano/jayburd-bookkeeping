import { useCallback, useEffect, useRef, useState } from "react";
import type { Journal } from "../bookkeeping/types";
import { CollectionFailure } from "./types";
import type { JournalIdentity, JournalReader } from "./journal";
import styles from "./collection.module.css";

// Mounted only after an explicit authorized anchor click; this view has no posting or correction controls.
export function CollectionJournal({
  identity,
  read,
  onDenied,
}: {
  identity: JournalIdentity;
  read: JournalReader;
  onDenied: () => void;
}) {
  const [journal, setJournal] = useState<Journal | null>(null),
    [busy, setBusy] = useState(true),
    [error, setError] = useState(false);
  const epoch = useRef(0),
    heading = useRef<HTMLHeadingElement>(null),
    denied = useRef(onDenied);
  denied.current = onDenied;
  const load = useCallback(async () => {
    const n = ++epoch.current;
    setJournal(null);
    setBusy(true);
    setError(false);
    try {
      const value = await read(identity);
      if (n === epoch.current) setJournal(value);
    } catch (reason) {
      if (n !== epoch.current) return;
      if (reason instanceof CollectionFailure && reason.kind === "denied")
        denied.current();
      else setError(true);
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  }, [identity.entity_id, identity.book_id, identity.journal_id, read]);
  useEffect(() => {
    void load();
    return () => {
      epoch.current++;
    };
  }, [load]);
  useEffect(() => {
    if (journal && !busy) heading.current?.focus();
  }, [journal, busy]);
  return (
    <section
      className={styles.review}
      aria-label="Linked posted journal"
      aria-busy={busy}
    >
      <h3 ref={heading} tabIndex={-1}>
        Posted journal
      </h3>
      {busy && <p role="status">Loading the authorized journal…</p>}
      {error && (
        <>
          <p role="alert">
            This journal is unavailable. Retry the current authorized read.
          </p>
          <button onClick={() => void load()}>Retry linked journal</button>
        </>
      )}
      {journal && (
        <>
          <p>
            {journal.accounting_date} · {journal.description}
          </p>
          <p>
            Book revision {journal.book_revision}. This journal is immutable;
            analytic labels do not change it.
          </p>
          <ul
            className={styles.reviewRows}
            aria-label="Immutable posted journal lines"
          >
            {journal.lines.map((line) => (
              <li key={line.line_number}>
                <span>
                  Account {line.account_id} · {line.side}
                </span>
                <span>
                  Original: {line.transaction_amount}{" "}
                  {line.transaction_currency}
                </span>
                <span>
                  Functional: {line.functional_amount}{" "}
                  {journal.functional_currency}
                </span>
                <details>
                  <summary>Currency references</summary>
                  <p>
                    Original reference {line.transaction_currency_version_id}.
                    Functional reference{" "}
                    {journal.functional_currency_version_id}.
                  </p>
                </details>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
}
