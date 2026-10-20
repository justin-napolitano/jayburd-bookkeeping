# Multi-account connection plan

## Goal

Allow the owner to repeat the Plaid Sandbox enrollment and bind all three selected
depository accounts in one reviewed action. Each provider account gets a distinct
ledger and source account. No automatic journal or opening balance is created.

## Delivery sequence

1. **Freeze the contract.** Merge the request shape, account classification,
   Plaid type/subtype support profile, atomicity, retry and standards decision in
   `docs/contracts/automatic-account-binding.md`. The initial allowlist is
   depository `checking`, `savings`, `money market` and `cash management`.
2. **Add migration 22.** Extend the restricted binding function so optional
   reviewed ledger-account definitions are validated and inserted before source
   bindings in the same serializable transaction. Add forward, replay, rollback,
   role, conflict and failure-atomicity evidence.
3. **Extend service and HTTP models.** Add the optional account definition and
   bounded provider subtype. Validate the allowlist on the server and regenerate
   OpenAPI types. Preserve token redaction, current permission checks and the
   existing endpoint.
4. **Make review usable.** Default every eligible depository account to a distinct
   new account proposal, show editable code/name/date/currency, and retain choices
   for existing account or do-not-bind. Explain unsupported credit/loan accounts
   with a link to #210's policy boundary.
5. **Prove the boundary.** Add focused Go/SQL/UI tests and a browser→API→PostgreSQL
   journey for three accounts, lost acknowledgement, no duplicates, no partial
   setup and zero automatic journals.
6. **Review and release.** Run repository gates, merge through a linked PR, stage
   migration before runtime, deploy reviewed images, then repeat the owner-driven
   Sandbox enrollment. Hosted success updates #88 but does not approve live use.

## Scope controls

- #209 owns depository auto-creation only.
- #210 owns credit and loan liability mapping after policy review.
- `cd`, `ebt`, `hsa`, `limited purpose checking`, `paypal` and `prepaid` remain
  unbound pending explicit policy; investment, other and payroll are out of scope.
- Existing bindings and one-account enrollment remain compatible.
- No source-provider call or cloud mutation occurs during implementation/testing.
- No credential, provider payload, balance or personal financial record enters
  commits, issues, test fixtures or logs.

## Done

The feature is done when all #209 acceptance tests pass, the migration and runtime
are deployed in order, and a fresh Sandbox attempt binds all selected supported
accounts as distinct active streams. Transaction acquisition/review is separate
evidence; provider success alone is not complete-history or bookkeeping approval.
