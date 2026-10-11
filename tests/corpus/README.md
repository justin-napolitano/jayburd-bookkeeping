# Original synthetic accounting corpus

## Decision and coverage

Keep fixed expected results independent of generators and future posting code.
Use exact decimals, retain original currency quantities, and compare a supported
subset with external engines. Sources and pin/rights information live in the
[resource register](../../docs/research/testing-sources.md).

| Corpus/check | Acceptance evidence | Limit |
| --- | --- | --- |
| `golden-month.json` | `TestCorpusFixtures`: AC01–10, two fictional entity books, fixed balances | No runtime authorization or posting |
| `fx-lifecycle.json` | Same test: full/partial EUR settlement, remeasurement, conversion and fee | Illustrative policy, not approved GAAP treatment |
| Invalid inputs and scope | `TestCorpusRejectsMutations`, `TestDecodeBoundary` | Fixture format, not production API parser |
| Original units | `TestFXExposureMutation` | No currency-code registry or rate selection |
| Decimal syntax | `FuzzDecimal` | No product rounding engine |
| Generator rejection | `TestGeneratorAudit`, `corpus audit-generator` | Exact EUR balance only; not full import validation |
| Independent balances | `scripts/test-corpus-references.sh` | Functional projections; does not test original-unit FX semantics |
| ISO XML | `scripts/test-corpus-iso.sh` | Provisional mirror; no bank profile or semantic conformance |

For book A the fixed ending checking balance is 1,045 USD, savings 100,
net income 70 and capital credits 1,100. Book B has a separate 70 USD opening.
The FX examples assert remaining receivable exposure separately from USD value.
Reversal fixtures preserve original stored amounts and negate posting sides.

## Default offline checks

```sh
make check test build
go run ./cmd/corpus check tests/corpus/golden-month.json
go run ./cmd/corpus check tests/corpus/fx-lifecycle.json
go test ./cmd/corpus -run '^$' -fuzz FuzzDecimal -fuzztime=3s -parallel=2
```

Default CI needs no external accounting engines, generated datasets, database,
credentials or provider network. Go toolchain installation is normal setup.
No SQL migration changes are included.

## Optional reference tools

Explicit acquisition requires network. Downloads remain ignored. For the reviewed
macOS ARM64 setup, download the 1.52.4 `hledger-mac-arm64.tar.gz` release asset to
`.local/external/`, verify the register's digest with `shasum -a 256`, and extract
only `hledger` into `.local/external/hledger/`. Other platforms need their own
verified release digest; do not reuse the macOS digest.

```sh
uv tool run --python 3.12 --from beancount==3.2.3 bean-check --version
bash scripts/test-corpus-references.sh
.local/external/hledger/hledger test
```

The reference script runs offline using that cached Beancount installation and
checks versions. It exports all five books, tests explicit balance assertions,
and requires both engines to reject a deliberately wrong expected balance.

Download only the XSD file from T03's pinned mirror into
`.local/external/camt.053.001.14.xsd`, then run:

```sh
bash scripts/test-corpus-iso.sh
```

The script verifies its digest, accepts the original statement, rejects a wrong
namespace and negative amount, and demonstrates that a wrong closing balance
still passes XSD. It uses `xmllint --nonet`; no DTD/entity expansion is requested.
Do not substitute real statements or claim official artifact verification.

## Generator smoke reproduction

Clone T04 into `.local/external/accounting-generator` and check out its exact
commit. From that checkout, with a **new output directory** (the generator can
delete an existing output directory):

```sh
test ! -e ../generator-smoke && \
PYTHONPATH=src uv run --python 3.12 --with PyYAML==6.0.3 --no-project python \
  -m synthetic_accounting_generator.cli generate \
  --config config/config.yaml --profile config/profiles/smoke.yaml \
  --scenario config/scenarios/baseline.yaml --output ../generator-smoke
```

Only proceed past the presence check if it succeeds. Repeat into a different
new directory to compare CSV bytes. From the Jayburd root:

```sh
go run ./cmd/corpus audit-generator .local/external/generator-smoke/journal_line.csv
```

For the pinned baseline the expected result is **rejection**: 115 journals,
293 lines, 15 unbalanced journals. This is a recorded upstream-data limitation,
not a passing accounting dataset. Default tests reproduce rejection with tiny
original synthetic examples; upstream data is not bundled.

## Next delivery gates

Approve accounting policy, then drive SQL/API implementation from these fixed
expectations. Add transaction rollback, concurrent posting, duplicate/replayed
requests, immutable posting, closed periods, entity authorization, outbox failure,
reconciliation and bank-profile tests against actual implementations. Include
reviewed FX rates/rounding/valuation cases and accountant-reviewed statements.
Neither engine agreement nor ISO XSD validation replaces those gates.
