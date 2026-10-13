import { useMemo, useState } from "react";
import { Reports } from "../src/bookkeeping/Reports";
import { createAPI } from "../src/bookkeeping/api";
import type { Book, Entity } from "../src/bookkeeping/types";
// This lifecycle harness and synthetic bearer are confined to the browser test bundle.
export function ReportsHarness() {
  const [scope, setScope] = useState("one");
  const [signedIn, setSignedIn] = useState(true);
  const api = useMemo(
    () =>
      createAPI(
        async () => "synthetic-token",
        () => setSignedIn(false),
      ),
    [],
  );
  const book = {
    book_id: scope,
    name: "Operating",
    book_revision: "1",
    accounts: [{ account_id: "cash", code: "1000", name: "Cash" }],
    policies: [],
    periods: [],
  } as unknown as Book;
  const entity = {
    entity_id: "entity",
    name: `Synthetic ${scope}`,
    permissions: ["read"],
  } as Entity;
  return (
    <>
      <button onClick={() => setSignedIn(false)}>Sign out test session</button>
      <label>
        Test scope
        <select value={scope} onChange={(e) => setScope(e.target.value)}>
          <option>one</option>
          <option>two</option>
        </select>
      </label>
      {signedIn && (
        <Reports
          key={scope}
          entity={entity}
          book={book}
          api={api}
          locked={false}
        />
      )}
    </>
  );
}
