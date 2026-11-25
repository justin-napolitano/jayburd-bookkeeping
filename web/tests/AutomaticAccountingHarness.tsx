import { useMemo, useState } from "react";
import { AutomaticAccounting } from "../src/bookkeeping/AutomaticAccounting";
import type { API } from "../src/bookkeeping/api";
import type { Book, Entity } from "../src/bookkeeping/types";
const id = (n: number) =>
  `77000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
const rule = {
  rule_id: id(6),
  name: "Synthetic paper",
  source_account_id: id(3),
  direction: "outflow",
  description: "Synthetic office paper",
  offset_account_id: id(4),
  treatment: "expense",
};
const book = {
  book_id: id(2),
  book_revision: "1",
  accounts: [
    {
      account_id: id(4),
      name: "Office supplies",
      category: "expense",
      normal_side: "debit",
    },
  ],
  policies: [
    {
      policy_id: id(5),
      functional_currency: "USD",
      effective_from: "2026-01-01",
      effective_to: null,
    },
  ],
} as Book;
// Fixed synthetic outputs prove presentation and transport intent; financial execution is proved by SQL.
export function AutomaticAccountingHarness() {
  const [writes, setWrites] = useState(0);
  const api = useMemo(
    () =>
      (async (path: string, body?: string) => {
        if (body) setWrites((v) => v + 1);
        if (path.endsWith("/sources"))
          return {
            profile: "source-list-v1",
            items: [
              {
                source_account_id: id(3),
                display_name: "Synthetic checking",
                currency: "USD",
                state: "active",
              },
            ],
          };
        if (path.endsWith("/status"))
          return {
            profile: "automatic-accounting-status-v1",
            entity_id: id(1),
            book_id: id(2),
            policy_revision: "1",
            enabled: true,
            runtime_enabled: false,
            pending_count: "0",
            attention_count: "1",
            blocked_reason: null,
            last_operation: null,
            holds: [
              {
                review_id: id(7),
                source_account_id: id(3),
                reason: "unmapped_rule",
                operation_id: id(8),
              },
            ],
            holds_truncated: false,
          };
        return {
          profile: "automatic-accounting-policy-v1",
          entity_id: id(1),
          book_id: id(2),
          revision: "1",
          enabled: true,
          supported_profile: "source-identity-usd-v1",
          policy: {
            book_policy_id: id(5),
            effective_from: "2026-01-01",
            effective_to: null,
            rules: [rule],
          },
        };
      }) as API,
    [],
  );
  return (
    <>
      <AutomaticAccounting
        api={api}
        entity={{ entity_id: id(1) } as Entity}
        book={book}
        locked={false}
        denied={() => {}}
      />
      <output aria-label="Configuration writes">{writes}</output>
    </>
  );
}
