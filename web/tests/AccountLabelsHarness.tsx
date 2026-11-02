import { useState } from "react";
import { Connections } from "../src/bookkeeping/Connections";
import type { API } from "../src/bookkeeping/api";
import { registrationWorkspace } from "./registration.fixture";

const id = (n: number) =>
  `00000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
// Missing display names/masks and duplicate product types are intentional synthetic privacy fixtures.
const fixture = {
  attempt_id: id(4),
  entity_id: id(2),
  book_id: id(3),
  mode: "new",
  state: "binding_review",
  expires_at: "2030-01-01T00:00:00Z",
  sync_item_id: id(5),
  manifest: {
    item_id: "private-fixture-item",
    consent_expires_at: "2030-01-01T00:00:00Z",
    accounts: [
      {
        account_id: "private-card-b",
        type: "credit",
        subtype: "credit card",
        iso_currency_code: "USD",
      },
      {
        account_id: "private-investment",
        type: "investment",
        subtype: "brokerage",
        iso_currency_code: "USD",
      },
      {
        account_id: "private-card-a",
        type: "credit",
        subtype: "credit card",
        iso_currency_code: "USD",
      },
    ],
  },
};
// An explicit test adapter supplies only synthetic projections; no production route or identity is replaced.
const api = (async (path: string) => {
  if (path.endsWith("/connections"))
    return {
      items: [
        {
          sync_item_id: id(5),
          item_id: "private-fixture-item",
          state: "active",
          item_generation: "1",
          accounts: [
            {
              stream_id: id(6),
              source_account_id: id(7),
              account_id: "private-card-a",
              state: "active",
              binding_revision: "1",
            },
          ],
        },
      ],
    };
  if (path.includes("/connection-attempts?"))
    return { items: [], next_cursor: null, unfinished_count: "0" };
  if (path.endsWith(`/connection-attempts/${id(4)}`)) return fixture;
  throw new Error("Unexpected synthetic label request");
}) as API;
// The review handoff remains an immutable source ID even though its button has a readable label.
export function AccountLabelsHarness() {
  const [source, setSource] = useState("");
  return (
    <>
      <Connections
        api={api}
        entity={registrationWorkspace.entities[0]}
        book={registrationWorkspace.entities[0].books[0]}
        currencies={registrationWorkspace.currencies}
        locked={false}
        denied={() => {
          throw new Error("synthetic denial");
        }}
        onReview={setSource}
      />
      <output aria-label="Reviewed synthetic source">{source}</output>
    </>
  );
}
