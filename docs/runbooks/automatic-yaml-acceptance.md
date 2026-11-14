<!-- stage-runbook: true -->
# Automatic YAML category acceptance —488

## Status and boundary

Owner requests bulk rule authoring, automatic classification and synthetic proof.
This lane adds acceptance evidence to the existing implementation, not a second
matcher or a provider-category backfill. Root owns500 deployment/hosted UI;
334 owns separate automatic source-backed accounting. All facts below are
independently authored fictional examples. No SQL migration, ad hoc SQL repair,
provider request, real record, cloud mutation or posted entry is introduced.

Read AGENTS/index/current capabilities, current488/65 discussion, the
[automatic contract](../contracts/automatic-categorization.md),
[query/rule contract](../contracts/collection-query-rules.md),
[rule-pack contract](../contracts/rule-packs.md) and their existing source records.
The source capability map and earlier runbooks contain historical pending notes;
current GitHub records distinguish completed source/deployment from hosted proof.
No new financial message or accounting policy is defined by a fictional rule pack;
ISO20022/21378 applicability and the unchanged ISO4217 boundary remain as recorded
in the rule-pack contract. No third-party corpus or matching code is copied.

## Fixed independent examples

The existing real restricted-SQL fixture supplies exactly these original facts:

| Description | Original outflow | Initial effective category | After eligible YAML | After conflicting YAML |
| --- | --- | --- | --- | --- |
| Fictional automatic cafe purchase | USD0.10 | Fictional Automatic Food; automatic rule | Same | Same |
| Fictional automatic protected choice | USD0.20 | Fictional Automatic Travel; manual choice | Same protected manual choice | Same protected manual choice |
| Fictional automatic unknown purchase | USD0.30 | Needs category; missing bank category | Fictional Automatic Food; automatic YAML rule | Needs category; Your rules disagree |

`web/tests/fixtures/automatic-yaml-eligible.yaml` adds one exact-merchant food
rule for Fictional Unknown Shop. The conflict pack adds a second exact-merchant
rule with equal scope/priority/specificity and a different travel output.
These outcomes are fixed here, independently of YAML parsing, the matcher and
automatic decision code. The packs preserve the existing cafe rule and do not
reference provider metadata, accounting accounts or posting actions.

The protected row shares the cafe merchant and contradicts its food rule. Its
manual Travel choice must survive both complete rule editions. Automatic
classification must resolve the unknown row after the first pack without any
selected-row rule-apply command, and must later abstain rather than retain the
earlier automatic food output after the conflicting pack.

## Runnable local composed proof

In a credential-free checkout with the pinned Go toolchain, Docker, locked web
dependencies and installed Chromium:

```sh
npm --prefix web ci
npm --prefix web exec -- playwright install chromium
make automatic-categorization-browser
```

The existing runner creates disposable synthetic PostgreSQL and a restricted API
role, builds the real Next forwarding path and Go test server, then mounts the
real application. Synthetic identity exists only in the test bundle/binary;
it is not a production authentication path. No host database/credential is read.

The journey retains its existing policy lost-response/same-request recovery,
foreign principal/origin denial, whole-result exceptions and responsive checks.
It adds actual YAML file selection → Validate YAML → Add to draft → Preview rule
changes → Save rule review → Confirm rule changes → Acknowledge result twice.
Draft validation has zero interactive command writes. Both saved editions are
processed by the real background runner; no per-transaction rule application is
invoked. A fresh component mount discards all React state and re-reads persisted
labels, provenance, exact conflict queue and the same final automatic run ID.

Independent final database checks require exactly3 original revisions with
unchanged lexemes0.10/0.20/0.30,9 automatic decisions across3 committed runs,
one protected manual decision,5 acknowledged committed interactive commands
(3 fixture setup plus2 rule catalogs), and zero journals/posting outbox/shared
interactive reservation. These are fixture-specific assertions, not hosted
record counts or evidence of complete bank history.

## Normal hosted UI handoff

Root performs the hosted proof only after the qualified500 web deployment, in
the normal verified-owner session whose entire workspace the owner confirmed
is synthetic. The two executable fixture packs above name the isolated local
fixture's categories/merchant and are not blindly usable against the hosted
46-row retained-history catalog. Do not invent matching transactions, provider
attribution or data through SQL to make a test pass.

For the hosted46-row fixture, use existing verified synthetic rows and current
starter category names. Preserve one owner choice before importing a narrow
merchant/date rule, then use two equal-ranked rules for a different exact
description/date. Use exported IDs for subsequent changes and preserve existing
rules/retired history. Do not treat a book filter as policy scope. Root must
record the actual matched rows, current categories, manual protection, conflict
exception and reload persistence; source tests never substitute that evidence.

The product path is Transactions → Manage rules → Import YAML → YAML file or
Rule pack YAML → Validate YAML → inspect added/updated/preserved summary → Add
to draft → Preview rule changes → Save rule review → Confirm rule changes →
Acknowledge result. Automatic categories must remain On. Refresh category status
until0 awaiting categorization, then Refresh transactions. Open Needs attention
and verify it contains the chosen conflict while excluding the resolved and
manual-protected rows. Reload and verify the same outcomes/provenance. Never use
Preview rules for selected as a substitute for automatic processing.

## Stage ledger

| Stage | Result |
| --- | --- |
| Resume | Dedicated test/488-yaml-category-proof at rewritten main88fce16; personal owner Git identity; root records fresh65 claim and disjoint test-file allocation. |
| Baseline attempt | Exact `make automatic-categorization-browser` stops at upfront “Synthetic proof port unavailable” before database/API startup. Independent Go version check is blocked writing the pinned toolchain cache; Docker socket is denied. No substitute identity, network bypass or weakened assertion introduced. |
| Source addition | Two fictional YAML files and existing real journey extended with eligible match, preserved contradictory manual choice, equal-rank conflict and remount. Restricted disposable SQL final oracles extended; product SQL/core unchanged. |
| Static/unit/production | `make web-check web-test web-build` PASS: exact generated API/TypeScript, all738units and optimized Next build. Two YAML files independently parsed and checked for distinct IDs/same conditions/rank/different fixed outputs. Shell syntax and diff checks pass. Playwright automatic config lists the extended journey successfully. |
| Explicit execution limits | Full `make web-browser` builds the bundle, then all102cases cannot execute because binding::1:3100 is denied EPERM. Core `make check test build` each reach the pinnedGo1.26.11 cache-write denial;230Node release tests pass before the Go test block. These are recorded attempts, not successful core/real-SQL/browser evidence. No assertion skipped or test weakened. |

## Current evidence

The source is ready for canonical integration review. Static/unit/production gates
pass as recorded; local execution restrictions prevent claiming that the extended
real browser/API/SQL journey has passed. No hosted workspace was mutated by this
lane. Existing original assertions remain and new fixed effects/zero-effect
assertions are added rather than replaced with weaker conditions.

## Remaining work

Run the extended composed journey on canonical CI and retain exact-head results.
Local type/unit/browser checks do not establish the unavailable restricted-SQL
journey. Root owns actual hosted owner-session proof after500; keep488 open
until its remaining acceptance is explicitly established. Historical source hashes
refer to pre502 rewritten history and remain historical receipts only.

## PR handoff

Refs488/498/500/65. Test/fixture/runbook changes only; no dependency, product API,
SQL migration, provider or runtime changes. Root coordinates normal PR/CI and
hosted acceptance. The existing automatic application CI step executes this
extended journey; passing component tests alone is insufficient. Preserve the
independent fixture expectations and current authorization/recovery assertions.
