<!-- stage-runbook: true -->
# Source activity implementation —429

## Status and boundary

MET429, dedicated issue/429-source-metrics based on merged5b68ca5/PR430 in the
parent development branch.430/47813ea passed all five exact-head checks and merged
after owner go-ahead; bounded contract issue428 completed, parent stack integration
pending. Wire/source stages committed before DDL.41 adds restricted read functions
and an observation lookup index, only applied to disposable synthetic databases.
40 and all older SQL unchanged. No private/provider/cloud,
bank consent, deployment, financial mutation or account registration authority.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Read index,429/428/430/65 and contract/oracle; reviewed database foundations/decision evidence, source read/observation/review/source-permission schemas and restricted service/bridge composition |
| Wire | Closed private POST, complete inventory grain, current-grant checks, exact unit/merchant drill, source/binding/review snapshot and unavailable fields specified |
| Prerequisite |430 all five checks PASS; merged5b68ca5 into parent427;428 contract scope completed, no parent/main/live completion inferred |
| Runtime |41 owner-only minimal current-fact helper, API-only private query, exact PostgreSQL aggregates/merchant rows, source/binding/review pins, Go route, generated wire and Next bridge |
| Final review | Headline date/search digest binds first drills as well as cursor continuations; absent summary units reject rather than invent zero. Readiness requires query EXECUTE and rejects direct API access to the private helper |
| Exact labels | Grouping and equality explicitly use C collation; ASCII vertical tab uses an octal escape so a valid one-letter v is never treated as blank. Other Unicode labels remain exact |
| Design validation | make packet ID=JB-001 and core check/test/build PASS with documented temporary local Go1.26.5 adapter; tracked toolchain/dependencies unchanged; no runtime metrics tested |
| SQL source review | SM07 records selected PostgreSQL17 aggregate/math entries and full RepeatableRead subsection, facts versus proposed application and pending runtime evidence |
| Final validation | PASS core check/test/build, migration-test/ledger-test/readiness, source-registry-test/sync-state-test, generated API/type checks,410 frontend units,52 browser tests and actual Next→Go→SQL connection/metrics/posting-recovery journey. Local Go1.26.5 modfile adapter/locked Node24.18.0; tracked dependencies unchanged; temporary integration overlays removed |
| Linux CI correction |431/65e4bb3 hosted3000-row initial summary exceeded the unchanged statement deadline. Synthetic nested plans exposed three prefix-only scans averaging1500 rows per observation and4498500 join-filter removals apiece.41 remains unmerged; bounded lateral primary-key probes now preserve full lookup keys under cold statistics. No timeout, auth, fixture target or result assertion relaxed; corrected-source validation pending |
| Index plan refinement |39's scoped review index remains intact.41 adds an equality-only observation UUID hash probe: avoids another competing entity/book-prefix index or covering scan that slows legacy reads. Entity/book predicates remain mandatory and UUID global uniqueness is not assumed. Diagnostic plans are synthetic only |
| Assignment scope | Assignment lookup probes observation equality first and applies a null-safe entity/book row match inside the bounded subquery before LIMIT. Authorized entity/book IDs are nonnull, so scope equality is unchanged while generic estimates cannot turn the scope prefix into the lookup key |
| Scope pins | Headline digest also binds entity/book. Equal source UUIDs and counters in two independently authorized scopes cannot reuse a headline pin. This is a consistency fence, never an access grant |
| Corrected-source gates | PASS core, full source-registry suite including real Go budget/two-book UUID/two-entity headline collisions/shared-review ambiguity, migration/ledger/readiness and actual Next→Go→SQL connection/metrics/posting-recovery journey. New3000-row proof3.64s including fixture setup, legacy30 pages3.97s on this synthetic run. Hosted correction CI remains pending; no deadline/assertion raised |
| Cardinality plan | Shared-review ambiguity is checked once with a scoped grouped guard before rendering current facts. Repeating its aggregate per observation let cold estimates choose the opposite prefix index again. The same ambiguity rejection is retained and directly tested |
| Plan evidence | Synthetic nested-plan summary about40ms versus original3525ms; fact scans about14ms, full observation/review lookup keys and one-row probes. Acceptance now runs the cold initial3000-row summary through the actual Go service's unchanged four-second budget. Diagnostic overlays/logging are private test tooling only |

## Current evidence

[Wire](../contracts/source-activity-api.md) refines
[meaning](../contracts/spending-metrics.md) with independent
[golden cases](../../tests/corpus/spending/README.md). Source runtime now includes
source-activity/query; it is not deployed or consumed by Overview yet.
The wire explicitly pins mutable review revisions as well as imported source and
binding revisions so changing overlap attention cannot silently change a drill.

Golden checks prove2017.5 incoming/940 outgoing and separate2-unit USD-version
activity, original JPY1250/KWD1.005,6 exact merchant groups and unavailable metadata.
Pending/zero fixture observations deliberately have no review rows; inventory
still counts them. Exact returned rows and every merchant page reconcile to the
independently fixed complete totals, not merely returned headline amounts.
The3000-row recipe gives whole60 versus first50 rows0.99 and30 unique100-row pages.
The first50 inputs use an earlier booked day to fix that page boundary despite
generated UUIDs. No expected result is derived from runtime aggregation/generation.

Initial proof failures caught nullable-key validation, missing fixture embedding,
overlapping baseline reference metadata, setup-revision assumptions and an invalid
test-only review transition. The guards were retained and fixtures corrected.
An assignment-cardinality plan timed out under the existing deadline;41 adds an
exact lookup index and keeps that deadline. The complete source suite now passes;
no hosted latency claim. Actual fenced synthetic publication explicitly posts100,
imports valid15/16 corrections and removal, reads current activity, and preserves
the original100 journal/evidence, one journal and one posting outbox.

Ordinary browser→Next→Go→SQL recovery proof PASS. Full fictional connection journey
PASS with the new authorized0.30 summary and exact0.10/0.20 rows, then existing
posting/recovery and SQL oracles. Port isolation initially missed the acquisition
fixture's own origin binding; private overlays were fixed without product auth
changes. A mistaken12.5 test expectation was replaced with its actual independent
0.10/0.20 inputs. No managed identity/external bank/Production qualification claim.

## Remaining work

PR and human sensitive-source review before runtime merge/apply. All final local
gates passed; hosted CI remains separate. Parent417's Phase B taxonomy/classification/allocation and418's visible
Overview remain open. No bank balance/functional valuation/freshness invented.

## PR handoff

Refs429/417/412.41 read functions/index only; no financial data rewrite or old SQL
change. Ordinary index creation has normal migration locking. Human security/source
review before runtime merge/apply; no deployment, consent renewal or live approval.
