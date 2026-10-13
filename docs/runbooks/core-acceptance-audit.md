# Accounting-core acceptance audit

<!-- stage-runbook: true -->

## Status and boundary

Decision: preserve the implemented synthetic posting core; fill specific evidence
gaps rather than rebuild it. Issues #1–4 and #17 remain open. This audit does not
approve real-LLC policy, broaden supported currencies, or accept hosted deployment.

Issue [#60](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/60),
parent [#54](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/54).
Reviewed main `0376b8793c882311f2d9cf0fbb2e99ecfc0bf540` on 2026-10-13.
Branch `issue/60-core-acceptance-audit`; sole changed path is this runbook.
[Lease #65](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/65)
and the claim on #60 own session status, not this document.

**Main gaps:** exact foreign-currency reversal assertions; explicit deletion
semantics; stale contract/tooling status; non-USD functional-book policy and the
full FX lifecycle; close/reconciliation/export contracts; official XSD provenance.
These are coverage or delivery gaps, not reproduced accounting defects.

## Stage ledger

| Stage | Evidence / result | Next action |
| --- | --- | --- |
| Claim | #60 open, no native blockers or competing claims; no open PRs at claim | Isolated worktree from main |
| Review | All 24 original criteria across #1–4/#17 mapped below; inspected actual assertions and SQL | Review bounded follow-ups |
| Validation | Default checks and all three isolated database suites passed | Review documentation PR |

## Current evidence

**Evidence vocabulary:** PROVEN means this audit ran the relevant synthetic test;
RECORDED means a prior runbook reports an observation, not rerun here; PARTIAL
means only a bounded subset is supported; OPEN means required evidence is absent
from the reviewed scope. Documented policy is not an automated test result.

Merged delivery anchors verified through GitHub: #14 migration tooling, #16
contracts, #18 corpus, #20 posting (`76f64c3`), #21 identity/setup, #30 restricted
migrator, #48 exact permission denials and #49 read-only migration status. Earlier
runbooks describe their historical slice; their “not implemented yet” wording
must not override later source or evidence.

Inspected executable evidence: [ledger integration tests](../../internal/ledger/integration_test.go),
[restricted migration tests](../../internal/migration/integration_test.go),
[corpus tests](../../cmd/corpus/main_test.go),
[generator rejection](../../cmd/corpus/generator_test.go),
[posting SQL](../../db/migrations/000002_posting.up.sql),
[Seed workflow](../../.github/workflows/seed.yml) and
[image workflow](../../.github/workflows/build-images.yml).

Observed on 2026-10-13 with Go 1.26.9 and the scripts' pinned synthetic images:

```sh
PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH \
GOCACHE=/private/tmp/jayburd-go-cache \
GOMODCACHE=/private/tmp/jayburd-go-modcache make check test build

PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH \
GOCACHE=/private/tmp/jayburd-go-cache \
GOMODCACHE=/private/tmp/jayburd-go-modcache make migration-test ledger-test cloud-sql-test
```

PASS: default vet/race/build/context/integrity checks; fresh/replay and expected
dirty-migration rejection; ledger integration, forward-upgrade sentinel and
restored receipt replay; runtime readiness denials; restricted migration,
read-only status, contention and dirty-failure tests. Initial Docker credential
helper failure inside the sandbox was resolved by an approved outside-sandbox
rerun; no credential values were inspected. Expected division-by-zero/dirty
errors are fault-test outcomes, not a recommendation to force a migration.

### Criterion map

Numbers within each issue follow its original acceptance list, not later planning
paragraphs. Links below identify inspected sources; test names refer to their
actual assertions, not a promise of complete coverage.

| Issue / criterion | Evidence and disposition |
| --- | --- |
| #1.1 Workflow, currency, basis, timezone, opening source | PARTIAL: [ADR 0005](../decisions/0005-development-accounting-policy.md) selects fictional accrual/USD/UTC and capital opening. Real entity, complete opening balances and owner/accountant approval remain OPEN. |
| #1.2 Fictional transaction scenarios | [AC-01–10](../contracts/accounting-scenarios.md) cover every listed scenario. `TestCorpusFixtures` checks fixed expectations; `TestGoldenMonth` posts the first book through SQL. This does not implement statement deduplication or reconciliation. |
| #1.3 Chart, precision, permissions, exports, cutover | PARTIAL: ADR 0005 and [posting boundary](../contracts/posting-boundary.md) specify the synthetic chart, exact amounts and permissions. Report/export contracts #51 and accountant/cutover review #10/#13 remain OPEN. |
| #1.4 Unsupported cases and decision owners | Documented: scenario policy register identifies owner/accountant/schema decisions and excluded automation. Full FX and tax treatment are not selected by an agent. |
| #2.1 Ownership, principals, journals, source references, opening, reversals, periods | PARTIAL: [SQL dictionary](../contracts/ledger-sql.md) and migrations define the first slice; `TestControlledSetup`, `TestSecondBookIsolation`, `TestJournalCorrectionHistory` exercise parts. Source-observation/import links and close evidence remain future contracts; rate provenance is not statement provenance. |
| #2.2 Restricted writes, composite keys, finite exact money | `TestSchemaPrivileges`, `TestSecondBookIsolation`, `TestMoneyInputBounds`, `TestManualSpotPosting`, `TestCurrencyScalesAndRounding`; restricted bootstrap also has exact SQLSTATE checks in `TestRestrictedMigration`. These are local roles, not actual API IAM login proof. |
| #2.3 Close locking and replay conflicts | Posting contract plus `TestPostingCloseRace`, `TestPostingReplayConcurrency`, `TestReceiptActorAndArchive`. The closer is privileged fixture code; no production close endpoint/evidence workflow is established. |
| #2.4 Every invariant mapped; migration/restore approach | PARTIAL: posting matrix and tests below; `TestLedgerRestoreReplay` restores synthetic data with roles already present. Coverage qualifications below and actual restore/PITR #26 remain OPEN. |
| #3.1 Pinned stack and query-generation decision | PARTIAL: CI pins Go 1.26.9; scripts pin PostgreSQL 17.11 and migrate 4.20.1 image digests; go.mod pins pgx/v5 5.11.0. ADR 0003 still says application dependencies are absent and recommends sqlc; current service uses explicit pgx calls. Record the actual query-access decision before closure. |
| #3.2 Isolated local/CI PostgreSQL and cleanup | `make migration-test ledger-test cloud-sql-test`; inspected scripts use dedicated synthetic containers/networks and invocation-owned cleanup, not an inherited live DSN. No database port is published. |
| #3.3 Lock/checksum/fresh/replay/failure/compatibility | `TestArtifactIntegrity`, `TestRestrictedMigration`, `TestMigrationContention`, `TestMigrationDirtyFailure`, `TestMigrationStatus`, shell fresh/replay/failure checks, `TestSchemaCompatibility`, forward-upgrade sentinel. Current-schema checks do not establish arbitrary old/new application compatibility. |
| #3.4 Non-root health-only image | RECORDED: [GCP evidence](gcp-delivery.md) and `scripts/test-api-container.sh`; not rerun in this audit. Current server also has managed accounting mode. Review the historical health-only criterion against inert/default mode rather than removing working routes. |
| #3.5 Integration CI, dependency review, image scans, required checks | PARTIAL: `seed.yml` runs database suites, govulncheck and npm audit; `build-images.yml` scans candidate images. Required-check enforcement was unavailable in the recorded private-repo plan; [delivery workflow](../delivery.md) requires owner review. No fresh scan or branch-protection capability probe here. |
| #4.1 Atomic journal/audit/receipt/outbox | `TestPostingInvariants` injects failures after book/header/line/audit/receipt/outbox writes and at COMMIT; `TestGoldenMonth` checks committed counts and fixed balances. This proves outbox intent, not worker delivery. |
| #4.2 Invalid/scope/revoked/deleted/closed/money rejection | PARTIAL: `TestPostingInvariants`, `TestEntityReferencesAndPrivileges`, `TestSecondBookIsolation`, both revocation-order tests, `TestPostingCloseRace`, money/scale tests. Disabled/missing principals are tested; a distinct physical-deletion lifecycle is not established. Resolve what “deleted” means before marking the entire criterion complete. |
| #4.3 Replay/conflicts/concurrency | `TestPostingReplayConcurrency`, `TestReceiptActorAndArchive`, `TestCommitAcknowledgementLoss`, `TestHTTPPostCommitBoundary` prove one effect, actor/payload conflict, historical replay and uncertain-COMMIT recovery. |
| #4.4 Immutable history and exact correction provenance | PARTIAL: `TestJournalCorrectionHistory` rejects mutation and repeat reversal and checks net trial balance; `TestReceiptActorAndArchive` rejects reversal into an archived account. No dedicated assertion compares every foreign original/reversal line and historical rate/policy identity; add that proof. Source-document provenance belongs to future import contracts. |
| #4.5 Write-stage failures and close races | `TestPostingInvariants` and `TestPostingCloseRace` execute against PostgreSQL with observed lock barriers. This is transaction ordering, not full month-close acceptance. |
| #17.1 Source versions/hashes/rights/review depth | RECORDED in [testing source register](../research/testing-sources.md); official schema provenance is explicitly incomplete, not silently promoted. |
| #17.2 Independent month/FX exact fixtures | `TestCorpusFixtures`, `TestCorpusRejectsMutations`, `TestFXExposureMutation`, `TestDecodeBoundary` check independent fixture expectations. FX lifecycle fixture arithmetic is not SQL lifecycle implementation. |
| #17.3 Beancount/hledger comparison | RECORDED: five functional projections and deliberately wrong balances tested in [corpus runbook](synthetic-corpus.md); `TestFunctionalExports` only checks emitted fixture assertions locally. External engines not rerun here. |
| #17.4 Official camt.053 schema and negatives | OPEN: prior official download failed; pinned mirror checks are RECORDED only. Obtain authoritative artifact/version/profile evidence, then rerun schema and semantic cases. |
| #17.5 Bounded generator evaluation | RECORDED: 15/115 journals rejected for cent residuals. `TestGeneratorAudit` independently rejects a one-cent difference, cross-journal netting and malformed amounts. Do not “fix” the oracle to match generated output. |
| #17.6 Offline defaults and optional checks | Inspected [corpus guide](../../tests/corpus/README.md), Makefile and scripts: external accounting engines/generator/XSD acquisition are explicit optional steps. Stale corpus-only “no database in default CI” wording does not describe today's full Seed workflow. |

### Assertion-level qualifications

1. **Multicurrency scope:** `ledger.mutate` currently accepts the selected
   accrual/USD/UTC/scale-2 profile and rejects other functional-book policies.
   JPY/EUR/KWD transaction-unit conversions are tested. Arbitrary functional
   currencies, position settlement/remeasurement and consolidation are not thereby
   implemented. See [multicurrency contract](../contracts/multicurrency.md) and #59.
2. **Reversal evidence:** migration 000002 constructs reversed lines from stored
   amounts, currency-version IDs and rate IDs. That is inspected implementation,
   not proof all those fields survive correctly: existing correction tests use
   identity-valued examples and aggregate assertions. Add exact field comparisons,
   later-period reversal with a different eligible rate, and reversal-of-reversal
   denial. Do not edit immutable migration history to add tests.
3. **Named matrix drift:** `TestMoneyInputBounds` does not alone exercise every
   case assigned to it in the posting matrix; spot-rate tests and scale tests
   supply other cases. `TestGoldenMonth` checks the first fixture book;
   `TestSecondBookIsolation` separately checks isolation. Avoid attributing a
   full product workflow to either name.
4. **Denial quality:** older `TestSchemaPrivileges` and immutability probes accept
   any SQL error. #48's `TestRestrictedMigration` checks 42501, but only for its
   enumerated subset. Harden remaining probes to expected errors and independent
   unchanged-state assertions; a failed query alone is weaker evidence.

### Source-review boundary

This is repository evidence review, not a new accounting policy or standards
interpretation. Read the [bibliography](../research/bibliography.md),
[decision evidence rules](../research/decision-evidence.md), ADR 0005,
[standards contract](../contracts/standards.md), and their recorded review limits.
No external normative text or external accounting engine was newly reviewed/run.
ASC 830 review, real policy approval and full ISO mappings remain unresolved
where their source registers say so. Do not label the system GAAP/ISO compliant.

## Remaining work

| Priority / owner | Bounded next delivery and acceptance |
| --- | --- |
| Next / ACCT, #4 | Add proposed `TestSpotReversalHistoricalProvenance` and `TestReversalOfReversalDenied`; compare original/reversal values, sides, reference IDs, links and original unchanged state. Extend denied-mutation checks with exact SQLSTATE and unchanged-state evidence. Run `make ledger-test`; no schema change expected. |
| Next / ACCT, #3/#2 | Review ADR 0003 query-access choice against explicit pgx implementation; reconcile historical “no schema/runtime” statements in database/corpus docs. Map current evidence accurately without rewriting historical stage results or inventing sqlc adoption. Shared-doc reservation required. |
| Next / ACCESS, #62 | Resolve disabled/absent/deleted-principal semantics against retention and provider revocation boundaries, then specify the missing named tests. No deletion or IAM expansion during review. |
| Policy / #1, #2, #59 | Owner/accountant resolves real functional currencies, basis, timezone, rates and openings; contract and independently worked non-USD/FX cases precede broader implementation. Keep this required scope visible. |
| Product / #51, #61, #10 | Report, source-import and close/reconciliation contracts and independent expected results. Trial-balance success does not replace these workflows. |
| Reference / #17 | Authoritative XSD acquisition/provenance and profile/semantic evidence; preserve generator rejection cases. |
| Cloud / #26, #27, #12 | Actual workload migration verification/replay, API role denials, restore/load and hosted synthetic proof remain their existing gates. Local reruns do not satisfy them. |

Prefer these scoped follow-ups over rebuilding the ledger or closing all old
issues because a suite passes. Only #60's audit deliverable is proposed for review;
the parent criteria and policy decisions retain their owners.

## PR handoff

Documentation-only change in the leased path. No SQL/API/runtime changes, no cloud
execution, no financial data or credentials. Review the criterion map and proposed
test gaps against the linked source before assigning implementation. Required
checks and synthetic database suites passed as recorded above. Hosted PR checks
remain separate evidence; this audit does not close #1–4/#17.
