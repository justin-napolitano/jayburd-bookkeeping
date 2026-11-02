import type { DurableCommand, Workspace } from "../src/bookkeeping/types";
const id = (n: number) =>
  `00000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
// Fixed synthetic saved work includes no bank names, masks, balances or real account identifiers.
export const savedRegistration: DurableCommand = {
  command_id: id(11),
  operation: "register_accounts",
  entity_id: id(2),
  book_id: id(3),
  state: "prepared",
  acknowledged: false,
  created_at: "2026-11-02T00:00:00Z",
  result: null,
  body: {
    selection: ["synthetic-checking"],
    attempt_id: id(4),
    sync_item_id: id(5),
    item_generation: "1",
    worker_revision: "1",
    scope_revision: "0",
    limit_revision: "0",
    manifest_digest: "a".repeat(64),
    book_revision: "1",
    policy_id: id(7),
    binding_date: "2026-11-02",
    expires_at: "2030-01-01T00:00:00Z",
    initial: true,
    digest: "b".repeat(64),
    bindings: [
      {
        account_id: "synthetic-checking",
        ledger_account_id: id(9),
        currency_version_id: id(8),
        source_account_id: null,
        expected_binding_revision: null,
        binding_date: "2026-11-02",
        binding_evidence_ref: id(13),
        new_ledger_account: {
          account_id: id(9),
          code: "1001",
          name: "Checking 1",
          category: "asset",
          normal_side: "debit",
          monetary: true,
          active_from: "2026-11-02",
        },
      },
    ],
  },
};
// Authorized workspace metadata is required to review the exact target and currency before execution.
export const registrationWorkspace: Workspace = {
  development_policy: "dev-accrual-usd-v1",
  entities: [
    {
      entity_id: id(2),
      name: "Fictional owner",
      kind: "legal",
      permissions: ["read", "import"],
      books: [
        {
          book_id: id(3),
          name: "Personal",
          book_revision: "1",
          policies: [],
          periods: [],
          accounts: [],
        },
      ],
    },
  ],
  currencies: [
    {
      currency_version_id: id(8),
      alphabetic_code: "USD",
      minor_unit: 2,
      valid_from: "2000-01-01",
      valid_to: null,
    },
  ],
};
