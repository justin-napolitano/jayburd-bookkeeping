<!-- stage-runbook: true -->
# Import money properties

## Status and boundary

Issue #94 implements the decoder property slice of #86. Test-only external
`ingestion_test` package uses exported Decode/types; no production, SQL, provider
or cloud changes. Lease IMPORT-PROPERTIES-2100. No new dependency or migration.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Contract review | Coordinator approved independent integer oracle, 0/2/3-decimal representation and purchase identity properties before implementation. Reused statement-decoder and SI01–SI06. |
| Implementation | Independent ASCII byte scanner and base-ten integer accumulation; exact scale divisibility, canonical rendering by integer division and rational quantity comparison. No production predicate or float reused. |
| Oracle checks | Fifteen manually specified oracle expectations plus actual-decoder boundary examples; CSV fixture preserves candidate bytes without CR normalization. |
| Validation | PASS focused decoder/staging race tests, both 60-second fuzz targets and make check test build on Go 1.26.9. No failure corpus or production defect found. |

## Current evidence

`TestDecoderMoneyOracle` independently checks USD cents, JPY integral units, KWD
thousandths, above-JavaScript-safe-integer magnitude, 29 integer/9 fractional lexical
limits and invalid signs/exponents/control bytes. Numeric equivalence cannot waive
the lexical digit limits, including insignificant zeros.

`FuzzDecoderMoney` compares real decoder acceptance to the independent oracle,
then checks original text, canonical quantity, currency/version, row identity and
ordinal. Invalid input must return a zero Statement. Candidate amounts over 4,096
bytes are excluded; existing file/record-size tests own those bounds.

`FuzzDecoderRepresentationIdentity` starts from positive integer minor units,
adds equivalent leading/trailing zeros, and checks canonical quantity against the
original generated integer. Distinct identical purchases retain both identities
and order; duplicate IDs reject atomically. Equivalent lexemes preserve their
separate original representations. Both targets have 54 committed deterministic
seed cases, also executed by ordinary `go test` without fuzz discovery.

Observed 2026-10-13, base `91891f0`, two workers per target:

| Target | Budget | Executions | New interesting / total | Package duration |
| --- | --- | --- | --- | --- |
| FuzzDecoderMoney | 60 s | 5,041,389 | 40 / 94 | 61.452 s |
| FuzzDecoderRepresentationIdentity | 60 s | 2,685,604 | 80 / 134 | 60.283 s |

Counts include all harness executions, including excluded oversize candidates;
these are bounded coverage observations, not throughput or exhaustive proof.
Interesting inputs live in the local fuzz cache; no failing input needed a committed
regression fixture. No scheduler/CI discovery budget is added in this slice.

Exact commands, from repository root with Go 1.26.9:

```sh
go test -race -count=1 ./internal/ingestion/...
go test ./internal/ingestion -run '^$' -fuzz '^FuzzDecoderMoney$' -fuzztime=60s -parallel=2
go test ./internal/ingestion -run '^$' -fuzz '^FuzzDecoderRepresentationIdentity$' -fuzztime=60s -parallel=2
make check test build
```

Execution used `GOCACHE=/private/tmp/jayburd-quality-fuzz-cache` and existing
`GOMODCACHE=/private/tmp/jayburd-go-modcache`. New target baselines each contained
only their 54 source seeds. Focused race packages passed: decoder 1.521 s, staging
5.111 s. Logs: `/private/tmp/jayburd-import-properties-{race,money,identity,default}.log`.

Source review: reused [statement contract](../contracts/statement-decoder.md) and
[SI source register](../research/statement-import-sources.md). Read Go 1.26.9 local
`src/math/big/int.go` Int.QuoRem and `ratconv.go` Rat.SetString API comments;
[public math/big navigation](https://pkg.go.dev/math/big). Integer quotient/remainder
supports exact minor-unit representability. Rat.SetString accepts broader syntax,
so the independent byte grammar is checked first. These API sections support the
test mechanism, not accounting policy or ISO conformance; the online default
version is newer than the locally pinned source reviewed.

## Remaining work

No durable effects, SQL authorization/isolation, FX valuation or posting arithmetic
is exercised. Generated retry/restart/reorder state sequences, cross-adapter matching,
provider profiles and scheduled fuzz automation remain #86 or dependent issue gates.
Pure decoder rejection proves only no returned rows, not database rollback.

## PR handoff

Only `internal/ingestion/money_property_test.go` and this runbook changed. Default
and focused gates passed; staged runbook guard and diff check required before commit.
Coordinator owns integration, shared indexes and issue handoff.
