# Server composition sources

## Decision record

Question: expose the reviewed SQL/identity adapters through one bounded server and
return enough authorized configuration for the first browser workflow. Status:
implementation review, owner project maintainer, reviewed 2026-10-11. This is an
application/security decision, not a change to accounting policy. The precise
contract and acceptance names are in [server contract](../contracts/server.md).

| Source and reviewed section | Established fact | Project interpretation and limit |
| --- | --- | --- |
| [Go net/http: Server, Shutdown, MaxBytesReader](https://pkg.go.dev/net/http#Server), Go 1.26.9 source/API, relevant field and lifecycle sections | Header, read/write and idle deadlines are independent; Shutdown drains active HTTP connections. | Apply all bounds, concurrency cap and bounded drain; handler cancellation alone is not a connection timeout. Health probes do not prove financial readiness. |
| [Firebase Admin Go: NewApp](https://pkg.go.dev/firebase.google.com/go/v4#NewApp), v4.22.0 source `firebase.go` NewApp/getProjectID and Auth composition | Explicit config avoids Firebase config-file defaults; explicit token source bypasses default credential discovery. | Supply both project and token source. Reject emulator and credential-file environment presence; never execute live auth during default tests. |
| [Google OAuth2: ComputeTokenSource](https://pkg.go.dev/golang.org/x/oauth2/google#ComputeTokenSource), v0.36.0 source `google.go` computeSource | Obtains service-account tokens from the metadata service rather than a JSON key. | Choose metadata-only GCP workload identity. Local ADC or external file-based federation is deliberately unsupported; actual target metadata/IAM remains a deployment gate. |
| [pgx pool Config](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool#Config), v5.11.0, and local `pgconn/config.go` ParseConfig security notes | Pool limits are explicit; connection parsing can consult service, password and TLS files and environment defaults. | Narrow DSN URL input, reject inherited PG settings and overwrite file paths/defaults; bound pool and statements. Never expose parse errors. |
| [PostgreSQL 17 §13.3.2 and §13.3.4](https://www.postgresql.org/docs/17/explicit-locking.html) | Row share locks conflict with updates; consistent acquisition order reduces deadlock risk. | Lock principal/entity/membership/book before returning discovery; retain full-transaction retry and independent race proof. This is not proof of provider revocation atomicity. |

Review depth: full relevant linked API/locking sections plus the named locally
pinned SDK/driver source functions; no unseen text or normative ISO claim. Mutable
API pages were checked on the review date; local module versions above govern code.

## Alternatives and follow-up

ADC simplifies local development but can silently discover private files; rejected
for this service boundary. Stateless discovery without locks can race membership
revocation; rejected. An unbounded whole-account export scales poorly; bounded
configuration is the initial scope, with fail-closed limits rather than truncation.
Paginated discovery, distributed admission control and detailed provider readiness
need separate contracts when production size and deployment topology are known.
Issues #3/#6 retain those deployment dependencies; no source citation closes them.
