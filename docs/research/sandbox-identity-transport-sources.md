# Private identity transport source review — #299

Reviewed2026-10-24 by coordinator; full selected request/status/date sections,
not complete product manuals. Project API2020-09-14 remains pinned. Public vendor
documentation is mutable and does not attest our deployment or approve a call.

| Source | Fact / interpretation and limits |
| --- | --- |
| [Plaid Transactions sync](https://plaid.com/docs/api/products/transactions/#transactionssync), request client_id/access_token/secret/cursor/count/options.account_id; response transactions_update_status | Credentials authorize an Item; omitted cursor starts history; count maximum500; account filter selects a separate account cursor stream; HISTORICAL_UPDATE_COMPLETE indicates initial and historical pulls complete. Inference: one account-filtered initial read can supply identities for the narrow complete posted-only fixture only when has_more=false and the existing projector accepts every authored tag. Never advance/reuse the application's cursor or claim readiness alone proves equality. |
| [Plaid custom users](https://plaid.com/docs/sandbox/user-custom/), configuring custom user, limitations, keeping dates current | Explicit custom data supports Link; edited saved users do not change existing Items; fixed dates can age out and rolling shifts dates. Inference: inspect fixture date validity before a fresh owner run, disable date rolling and do not rewrite expected dates from observations. No personal or broad history proof. |
| [Go os.NewFile](https://pkg.go.dev/os#NewFile), Unix nonblocking/pollable deadline section; local Go1.26.5 src/os/file.go and file_unix.go NewFile/newFileFromNewFile/newFile selections | Unix inherited blocking descriptors must enter nonblocking mode before wrapping to support pollable cancellation/deadlines. Initial actual pipe test hung in syscall.read despite context close; setting O_NONBLOCK first and requiring SetReadDeadline support makes it pass. Shared opener fixes both offline projector and new producer. Online package page showed1.27.3; attempted1.26.10 package/GitHub pages were unavailable, so no assertion of reviewing that release's source. Canonical1.26.10 CI remains required. |

Alternatives, standards, scope, threats and named evidence belong to the
[transport decision](../contracts/sandbox-identity-transport.md); existing projector
and SQL comparison contracts retain financial semantics and authorization.
