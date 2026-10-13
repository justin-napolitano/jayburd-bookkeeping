import { useMemo, useState } from "react";
import {
  ReviewInbox,
  type SourcePostCommand,
} from "../src/bookkeeping/ReviewInbox";
import { createAPI } from "../src/bookkeeping/api";
import type { Book, Currency, Entity } from "../src/bookkeeping/types";
// Synthetic-only browser harness observes recovery handoff without executing a command.
export function ReviewInboxHarness() {
  const [command, setCommand] = useState<SourcePostCommand | null>(null);
  const [denied, setDenied] = useState(false);
  const api = useMemo(
    () =>
      createAPI(
        async () => "synthetic-token",
        () => setDenied(true),
      ),
    [],
  );
  const book = {
    book_id: "b",
    book_revision: "4",
    accounts: [
      { account_id: "cash", name: "Checking" },
      { account_id: "expense", name: "Supplies" },
    ],
  } as Book;
  return (
    <main>
      {denied ? (
        <p>Access unavailable</p>
      ) : (
        <ReviewInbox
          api={api}
          entity={{ entity_id: "e" } as Entity}
          book={book}
          currencies={
            [{ currency_version_id: "usd", minor_unit: 2 }] as Currency[]
          }
          locked={command !== null}
          prepare={setCommand}
          denied={() => setDenied(true)}
        />
      )}
      {command && (
        <section aria-label="Recovery handoff">
          <h2>Posting command awaiting confirmation</h2>
          <p>{command.body.posting_body.description}</p>
          <p>No execute request was sent.</p>
        </section>
      )}
    </main>
  );
}
