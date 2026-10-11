# Synthetic accounting corpus review

<!-- stage-runbook: true -->

## Status and boundary

Issue [#17](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/17).
Branch `test/synthetic-accounting-corpus`, base merged #16 (`f88c8bb`). First
fixture/tooling stage implemented; official XSD provenance remains PARTIAL.
No ledger runtime, SQL changes, bank integration or approved accounting policy.
No personal data, live DSN or real provider credentials.

## Stage ledger

| Stage | Result | Next action |
| --- | --- | --- |
| Source review | Four resources adopted with pins, roles and rights | Maintain [source register](../research/testing-sources.md) |
| Original fixtures | Two JSON corpora, five books; exact functional balances and original units | Apply to future SQL/API behavior |
| Reference comparison | Both engines accept five projections and reject wrong expectations | Expand only with reviewed semantics |
| ISO smoke | Positive/negative schema checks pass against pinned mirror | Obtain official artifact; select bank profile |
| Generator review | Reproducible CSVs; exact audit rejects 15 journals | Preserve as invalid import/volume sample |
| Local gates | Check/test/build and bounded fuzz pass | Review PR; resolve remaining issue gate |

## Current evidence

Observed locally 2026-10-11; default commands use Go 1.26 with
`GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache`.

- PASS `make check test build`: context/doc links, migration-file checks,
  stage-runbook structure, vet, race tests, both tooling builds.
- PASS `go test ./cmd/corpus -run '^$' -fuzz FuzzDecimal -fuzztime=3s -parallel=2`:
  333,365 executions; no failing input.
- PASS `bash scripts/test-corpus-references.sh`: five books accepted by
  Beancount 3.2.3 and hledger 1.52.4; wrong fixed balance rejected by both.
- PASS `.local/external/hledger/hledger test`: 246 built-in tests. This is
  upstream tool evidence, not execution of its entire source functional suite.
- PASS `bash scripts/test-corpus-iso.sh`: original statement accepted; wrong
  namespace and negative amount rejected. Intentionally wrong closing balance
  accepted, demonstrating XSD's semantic limit.
- PARTIAL ISO provenance: official download returned HTTP 403. Mirror commit
  and computed SHA-256 pinned in source register; no official comparison.
- PASS generator reproducibility: two seed-42 smoke runs; all 35 CSV files
  byte-identical. Upstream self-validation reported success.
- EXPECTED REJECTION `go run ./cmd/corpus audit-generator
  .local/external/generator-smoke/journal_line.csv`: exit 1, 115 journals,
  293 lines, 15 unbalanced; independent decimal inspection found ±0.01 EUR.
  Our `TestGeneratorAudit` reproduces cent-difference rejection offline.

External preparation and exact invocation steps are in the
[corpus guide](../../tests/corpus/README.md). No external downloads are transitively
invoked by default tests. Artifacts remain ignored. Before commit, run
`bash scripts/check-stage-runbook.sh --staged` with this runbook staged.

## Remaining work

Keep #17 open for comparison with the official XSD artifact. No missing official
provenance is silently promoted to conformance. Bank usage profiles, code-list
validation and reconciliation semantics remain future adapter work. Pinning
external transitive Python dependencies fully remains a reproducibility improvement.

Accounting policy approval and runtime tests remain outstanding: transactional
posting, concurrent writes, replay/idempotency, scoped access, closed periods,
immutability, outbox failures and accountant-reviewed financial statements.

## PR handoff

Review original expected values and the separation between fixture arithmetic,
external engine agreement and standards conformance. Generator defects are
recorded rather than rounded away. This stage introduces no migrations or
production dependency. Next delivery unit: approved policy plus SQL posting tests
using the fixed corpus; official XSD acquisition can proceed independently.
