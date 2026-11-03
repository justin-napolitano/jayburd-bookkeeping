import { useMemo, useState } from "react";
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
export function AccountLabelsHarness({
  multiple = false,
}: {
  multiple?: boolean;
}) {
  const [source, setSource] = useState("");
  const [mutations, setMutations] = useState(0);
  // Count attempted writes, including unexpected ones; opening/cancelling management must remain inert.
  const trackedAPI = useMemo(
    () =>
      (async (path: string, body?: string, key?: string) => {
        if (body !== undefined) setMutations((value) => value + 1);
        const value = await api(path, body, key);
        if (multiple && path.endsWith("/connections")) {
          const listing = value as { items: { accounts: object[] }[] };
          return {
            items: listing.items.map((item) => ({
              ...item,
              accounts: [
                ...item.accounts,
                {
                  stream_id: id(8),
                  source_account_id: id(9),
                  account_id: "private-card-b",
                  state: "disconnected",
                  binding_revision: "2",
                },
              ],
            })),
          };
        }
        return value;
      }) as API,
    [multiple],
  );
  return (
    <>
      <Connections
        api={trackedAPI}
        entity={registrationWorkspace.entities[0]}
        book={registrationWorkspace.entities[0].books[0]}
        currencies={registrationWorkspace.currencies}
        locked={false}
        denied={() => {
          throw new Error("synthetic denial");
        }}
        onReview={setSource}
      />
      <output className="sr-only" aria-label="Reviewed synthetic source">
        {source}
      </output>
      <output className="sr-only" aria-label="Synthetic mutation count">
        {mutations}
      </output>
    </>
  );
}
