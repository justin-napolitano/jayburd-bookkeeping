import type { components } from "../src/lib/api.generated";
import type { Book, Currency, Entity } from "../src/bookkeeping/types";
export type ActivityQuery = components["schemas"]["ActivityQuery"];
// Fixed fictional server quantities deliberately exceed binary-safe integer precision.
export const activityBook: Book = {
  book_id: "b",
  name: "Personal",
  book_revision: "1",
  policies: [],
  periods: [],
  accounts: [],
};
export const activityEntity: Entity = {
  entity_id: "e",
  name: "Fictional household",
  kind: "personal",
  permissions: ["read"],
  books: [activityBook],
};
export const activityCurrencies: Currency[] = [
  {
    currency_version_id: "usd",
    alphabetic_code: "USD",
    minor_unit: 2,
    valid_from: "2000-01-01",
    valid_to: null,
  },
];
export const activitySources = {
  profile: "source-list-v1",
  items: [
    {
      source_account_id: "s",
      display_name: "Fictional checking",
      currency: "USD",
      currency_version_id: "usd",
      ledger_account_id: "cash",
      state: "active",
      binding_revision: "1",
      source_revision: "4",
    },
  ],
};
// This wire fixture asserts explicit strings rather than calculating expected totals in the browser.
export function activityPage(
  q: ActivityQuery,
): components["schemas"]["ActivityPage"] {
  const {
    snapshot: _snapshot,
    cursor: _cursor,
    limit: _limit,
    ...selector
  } = q;
  return {
    profile: "source-activity-v1",
    entity_id: "e",
    book_id: "b",
    source_account_id: q.source_account_id,
    source_state: "active",
    snapshot: {
      source_revision: "4",
      binding_revision: "1",
      review_revision_sum: "2",
      filter_digest: "fictional-pin",
    },
    selector,
    generated_at: "2026-11-01T12:00:00Z",
    bank_balance: null,
    functional_total: null,
    categorized_spending: null,
    last_successful_sync: null,
    unavailable_reasons: {
      bank_balance: "not_projected",
      functional_total: "no_valuation",
      categorized_spending: "not_implemented",
      last_successful_sync: "not_projected",
    },
    history_completeness: "unknown",
    available_from: "2026-08-01",
    available_through: "2026-11-01",
    inventory: {
      source_count: "6",
      matching_count: "6",
      eligible_count: "2",
      pending_count: "1",
      removed_count: "1",
      zero_count: "1",
      quality_hold_count: "1",
      correction_attention_count: "1",
      partial: true,
    },
    units: [
      {
        currency_version_id: "usd",
        currency: "USD",
        numeric_code: "840",
        minor_unit: 2,
        count: "2",
        inflow: "9007199254740993.01",
        outflow: "24.01",
        net_inflow: "9007199254740969.00",
      },
    ],
    totals:
      q.kind === "summary"
        ? null
        : {
            count: "2",
            inflow: "9007199254740993.01",
            outflow: "24.01",
            net_inflow: "9007199254740969.00",
          },
    items:
      q.kind !== "observations"
        ? []
        : [
            {
              currency_version_id: "usd",
              currency: "USD",
              numeric_code: "840",
              minor_unit: 2,
              observation_id: q.cursor ? "second" : "first",
              observation_revision: "1",
              booked_date: q.cursor ? "2026-11-01" : "2026-08-01",
              description: q.cursor
                ? "Fictional payment"
                : "Fictional merchant & cafe",
              source_state: "booked",
              direction: "outflow",
              amount: "24.01",
              review_id: null,
              review_state: null,
              merchant: {
                status: "attributed",
                label: "Fictional merchant & cafe",
                profile: "provider-merchant-exact-v1",
              },
            },
          ],
    merchant_groups:
      q.kind !== "merchants"
        ? []
        : [
            {
              status: "attributed",
              label: "Fictional merchant & cafe",
              profile: "provider-merchant-exact-v1",
              count: "2",
              inflow: "9007199254740993.01",
              outflow: "24.01",
              net_outflow: "-9007199254740969.00",
            },
          ],
    group_count: q.kind === "merchants" ? "1" : null,
    remaining_group_count: q.kind === "merchants" ? "0" : null,
    next_cursor:
      q.kind === "observations" && !q.cursor ? "fictional-continuation" : null,
  };
}
