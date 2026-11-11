# Synthetic readiness: boundaries and gaps — 482

## Review scope

Source audit on main `0a7d3731`, 2026-11-11. This records implemented checks and
what they can establish; a named test is not a fresh PASS receipt. Fresh results
belong in `docs/runbooks/synthetic-readiness.md`. No real financial records,
provider calls, credentials, managed database or cloud resources were inspected.

Reviewed repository evidence: artifact-backed-accounting.md, source-review-sql.md,
command-recovery.md, browser-auth.md, tests/corpus/README.md and the relevant test
bodies below. Existing bibliography/source register and standards applicability
are reused. This adds no financial contract, policy or standards-conformance
claim; no external standards text or current vendor behavior was newly reviewed.

## Evidence matrix

| Boundary | Concrete existing evidence | What it proves / limitation |
| --- | --- | --- |
| Application posting | `internal/ledger/artifact_policy_test.go`, `TestArtifactRetriesAndReplay`; actual connection browser journey in `web/tests/integration/connections.spec.tsx` | Public application rejects standalone manual entry; accepted posting follows retained observation→review→immutable draft→saved `source_post`→explicit execute/ack. Isolated ledger primitive tests do not establish an alternative customer workflow. |
| Current source authority | `internal/source/review_integration_test.go`: `TestSourceReviewPosting`, `TestSourceReviewSnapshots`, `TestSourceReviewAuthorization` | Actual restricted PostgreSQL functions enforce exact source/binding/book/review pins, current prepare/post/source-review authority, replay and atomic links. Foreign or absent principals cannot read the review. Synthetic fixture grants are not live enrollment. |
| Scope and revocation races | `internal/ledger/workspace_integration_test.go`: `TestWorkspaceAuthorization`, `TestWorkspaceRevocation`; `recovery_integration_test.go`: `TestCommandAuthorization` | Read-only discovery, hidden unreadable scopes, disabled/absent actors, foreign command ownership and both revoker-first/reader-first SQL lock orders. Revocation removes response bodies and cannot become implicit acknowledgement. Some command-library tests deliberately use historical low-level manual fixtures; these are not public posting acceptance. |
| Provider-flow composition | `internal/providerflow/flow_integration_test.go`: `TestProviderEnrollmentAcquisitionFlow` | Actual HTTP handler, restricted API/worker/webhook logins, encrypted credential repository and source SQL. Synthetic transport returns fixed 0.10 and 0.20 USD; replay exchanges once, publishes once, creates no journal/outbox automatically, and disconnect/revocation prevent later publication. Its identity, key wrapper and provider responses are test doubles. |
| Webhook authenticity | `internal/webhook/webhook_test.go`: `TestWebhookRejectedInputs`, `TestWebhookKeyLifecycle`; provider-flow socket proof | Real ES256 verification and exact-byte SHA256 reject tampering, stale/future timestamps, wrong algorithm, duplicate fields, key URLs, oversized input and unsupported signed scope. Ephemeral fictional signing keys establish implementation behavior, not Plaid key delivery or hosted routing. |
| Webhook durable isolation | `internal/webhook/webhook_integration_test.go`: `TestWebhookPostgresReceipt`, `TestWebhookPostgresRoleIsolation`; `production_integration_test.go`: `TestProductionWebhookCurrentScope`, `TestProductionWebhookRoleIsolation`; signed repair tests | Actual restricted SQL receipt/demand replay and role/current-scope fences. Fictional production-profile tests do not call Production Plaid or establish institution/OAuth qualification. |
| Worker publication fences | `internal/acquisition/acquisition_integration_test.go`: `TestAcquisitionCommittedReplyLoss`, `TestAcquisitionRevocationBeforePublication`, `TestAcquisitionExpiredClaimAndDeadline`, `TestAcquisitionRoleIsolation` | Lost commit replies replay safely; revoke after fetch prevents publication; expired lease/canceled context cannot fetch; API and mixed-capability worker logins fail. This is bounded synthetic fault evidence, not an availability or timing SLA. |
| Identity boundary | `internal/identity/firebase_test.go`: `TestIdentityBoundary`; browser auth-controller/adapter suites | Requires revocation-aware SDK method, correct issuer/audience/subject/UID, nonexpired token, tenant fence and TOTP before principal mapping. SDK outcomes are stubs. It does not prove real Firebase signature/key fetching, verified-email delivery, refresh/revocation propagation or hosted MFA. |
| Money and policy | Fixed corpus, source-review and acquisition precision tests, real report/CSV SQL tests | Independent exact decimal/original-unit and current policy assertions. FX-required and unsupported precision remain holds. Fixture agreement, XML validation and reference-engine agreement do not establish full GAAP/IFRS or ISO conformity. |

## Bounded-load distinction

Existing `TestCollectionPagination3000`, tag/query SQL tests and demo import proof
exercise 3,000 source observations and bounded paging. The demo runner explicitly
asserts 3,000 imported observations and **one** atomic source-linked posting.
They do not prove 1,000 posted entries, concurrent throughput, production latency
or larger-account capacity. The old manual VolumeViewer is retired and must not
be revived or counted as artifact-backed acceptance.

Approved missing proof, now drafted as `TestSourceBackedPostingVolume1000` in
`internal/providerflow/volume_integration_test.go`: 1,000 retained fictional statement rows, each 0.10 USD,
reviewed and posted through the real source-backed HTTP/restricted SQL boundary.
Independent expectations are 1,000 distinct linked entries, 2,000 lines,
100.00 USD debit and 100.00 USD credit, and exactly one effect per replayed
command. Fixed workload duration is measured, not asserted as an SLA. Every execute is replayed before explicit acknowledgement, followed by four old
acknowledged command replays after all later book revisions. Ten actual review
pages enumerate identities before any draft changes the snapshot. Per-line
original/functional USD amounts and source-credit/expense-debit sides are checked
independently as well as report totals. Compile and execution evidence remain
pending until the fixture and standard runner complete successfully.

## Unimplemented and external acceptance

- **NOT IMPLEMENTED — 476:** general receipt/invoice/document ingestion, evidence
  matching and duplicate economic-event prevention across multiple artifacts.
  Retained bank/statement observations are the current supported evidence.
- **NOT IMPLEMENTED — 334:** automatic accounting posting. Private labels/tags/rule
  suggestions and explicit historical apply do not post journals or replace review.
- **NOT IMPLEMENTED — 333:** artifact-linked correction intake and reversal flow.
  Historical immutable entries remain readable; a ledger primitive reversal test
  is not an exposed correction workflow.
- **EXTERNAL VALIDATION:** real Firebase enrollment, verified email, MFA, token
  lifecycle and current identity mapping; bank/institution OAuth redirects,
  Hosted Link completion/reconnect/revocation, Plaid key/webhook delivery,
  provider history completeness, original-currency metadata and real-bank
  reconciliation. No synthetic PASS clears these gates or renews old consent.
- **LIMITED RESTORE:** an isolated logical dump/restore can prove retained
  identities/receipts and exact reports for its fixture. It is not managed backup
  restore, PITR, key recovery, disaster recovery timing or live-account readiness.

The owner selected synthetic-first validation. None of these remaining external
checks require a provider call, consent request or real-data inspection in this
phase. Do not label the whole application financially approved because CI is green.

## Local verification so far

- PASS: integration-tag providerflow package compiles with the existing external
  local module file; `-run '^$'` deliberately executes no SQL tests (0.696s).
- PASS: fresh selected `TestIdentityBoundary`, `TestWebhookRejectedInputs`,
  `TestWebhookKeyLifecycle` and `TestRunnerUnknownCommitAndRevocation` with
  `-count=1` (identity0.660s, webhook0.909s, acquisition1.027s package durations).
  These are offline SDK/provider-double checks, not real provider acceptance.
- PENDING successful acceptance: isolated PostgreSQL restore and 1,000-posting workload under
  the unchanged standard provider-flow runner and its existing240s timeout.
  Compilation or source review must not be reported as that execution result.

## First canonical execution and fixture correction

CI43080241136 first executed the 1,000-posting loop in18.80s, then failed an
incorrect test assumption that the fixture's book began at revision0. SQL1's
`ledger.setup_change` adds one revision for each of the two fixture accounts,
one policy and one period: the independently fixed initial revision is4.
The corrected proof requires initial4, each posting receipt revision5 through1004,
and final1004, while retaining1,000 posted rows and every count/money/replay
oracle. Public posted-state pagination must also return exactly the same1,000
review identities. This is a test correction, not a product change or a PASS
receipt; the first canonical proof failed before later assertions and needs a
fresh successful full run. Existing240s runner timeout remains unchanged.
