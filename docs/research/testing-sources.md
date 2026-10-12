# Synthetic accounting test resources

## Decision

Use all four resources below in different roles. None is a universal accounting
certification suite. Original fixtures and fixed expected balances belong to this
repository; downloaded tools, schemas and generated datasets stay in ignored
`.local/`. Reviewed 2026-10-11. See [execution and coverage](../../tests/corpus/README.md)
and [evidence](../runbooks/synthetic-corpus.md).

## T01 — Beancount

[Project and examples](https://github.com/beancount/beancount/tree/v3/examples),
[release metadata](https://pypi.org/pypi/beancount/3.2.3/json),
[license](https://github.com/beancount/beancount/blob/v3/COPYING).
A mature text-based double-entry engine, useful for independent balance assertions
and example vocabulary. Selected 3.2.3; GPL-2.0-only. Reviewed example inventory,
license and CLI behavior; executed `bean-check` on our five functional-currency
projections and an intentionally wrong balance. We do not copy upstream examples
or embed its implementation. Python is an optional external test tool, not a
backend dependency. These checks do not establish equivalent FX recognition,
rounding, tax treatment or valuation policy.

## T02 — hledger

[Testing guide](https://hledger.org/TESTS.html),
[release](https://github.com/hledgerorg/hledger/releases/tag/1.52.4),
[license](https://github.com/hledgerorg/hledger/blob/main/LICENSE).
A separate accounting engine with built-in unit tests and a source-level functional
test collection. Selected 1.52.4; GPL v3 license text reviewed (consult package
notices before redistribution). Reviewed test mechanisms and ran 246 built-in
tests plus our five balance comparisons and a negative assertion. We did not run
its entire source functional suite or vendor its tests. Agreement with Beancount
is useful arithmetic evidence, not proof of our future database transactions.

## T03 — ISO 20022 catalogue and camt.053

[Official catalogue](https://www.iso20022.org/iso-20022-message-definitions),
[official XSD endpoint](https://www.iso20022.org/message/23480/download),
[IPR policy](https://www.iso20022.org/intellectual-property-rights).
Selected `camt.053.001.14` (BankToCustomerStatementV14) for a schema smoke test,
not as an assumed bank-supported version. Reviewed catalogue metadata and IPR;
the official download returned HTTP 403. Contributors retain rights under the
published IPR terms; do not label ISO artifacts as MIT-licensed.

Provisional test artifact: [pinned third-party mirror](https://github.com/sebastienrousseau/camt053/blob/f4ff150917231c71f734abfcc2c25bed486cc184/camt053/xsd/camt.053.001.14.xsd).
Reviewed required statement/entry elements and validated original fictional XML.
Its digest is recorded below; comparison with an official download remains
BLOCKED. Schema acceptance covers structure only: our deliberately wrong closing
balance also validates. Official provenance, message rules, external code lists,
bank usage profiles and reconciliation semantics remain separate gates.

## T04 — Synthetic accounting data generator

[Project at reviewed commit](https://github.com/R3n0va/synthetic-accounting-data-generator/tree/b0f0e34ade2b18995252837e3011f3a89eb1ca52),
[MIT license](https://github.com/R3n0va/synthetic-accounting-data-generator/blob/b0f0e34ade2b18995252837e3011f3a89eb1ca52/LICENSE).
Version 2.3.0 supplies related fictional accounting CSV tables. Reviewed CLI,
smoke/baseline configuration, seeded randomness, money helper and output writer;
this was not a full security audit. The writer can delete its output directory:
use a new ignored directory only. Its two-decimal half-up rounding and German
VAT context do not define our accounting policy.

Two seed-42 smoke runs produced 35 byte-identical CSVs: 8 clients, 115 journals,
293 lines. Although its own validation passed, our exact-decimal audit found
15 journals with ±0.01 EUR residuals. Adopt it for import, volume and rejection
tests; do not bless those journals or auto-plug differences. The bounded sample
is not a scale benchmark. A four-journal balanced review subset is now embedded
in the integration-only demo seeder; its [provenance and MIT notice](../../internal/server/testdata/demo/README.md)
record unchanged EUR amounts, synthetic date/rate mappings and fixed cent oracles.
The remaining dataset is not bundled. The 15 invalid journals remain rejected.

## Local review transport

[Vite shared options, envDir](https://vite.dev/config/shared-options.html#envdir)
was reviewed for the separate loopback demo. `envDir: false` disables environment
file loading; an allowlisted process environment and fixed proxy target keep this
review bundle independent of managed identity configuration. This uses the locked
Vite dependency already supplied by the browser test toolchain, not a Next route.

## Pins and integrity

| Artifact | Pin / SHA-256 | Evidence |
| --- | --- | --- |
| Beancount | 3.2.3 | Executed version verified; dependencies not fully locked |
| Beancount source archive metadata | `f52c4d237bcb092cbf02a29373d3e59d95349c2828a91491818ae437ee220f74` | PyPI advertised digest, not installed-wheel verification |
| hledger mac-arm64 archive | `c90a0fe4537439e958ce530f7175d937749f5f126759c94c1212774abe9eb971` | Download digest matched release metadata |
| Provisional camt XSD | `fdad51c5748d8b05056bb6dc140952e6c87ec749e63368df6834e4f36f9e6a9f` | Download digest computed; official comparison pending |
| Generator | `b0f0e34ade2b18995252837e3011f3a89eb1ca52` | Checkout HEAD verified; Python 3.12 / PyYAML 6.0.3 |

`JB-AC` in fixtures refers to our [accounting scenarios](../contracts/accounting-scenarios.md).
Source IDs identify influence/reference checks, not copied upstream data or
approval of the accounting policy. Changes to any pin require renewed evidence.
