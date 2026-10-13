# Provider sync SQL delivery
<!-- stage-runbook: true -->

## Status and boundary

Implemented bounded #70/#100 synthetic acquisition foundation; lease SYNC-SQL-100.
Migration 12 appends provider evidence and review exceptions with an atomic cursor
receipt. No provider, secret retrieval, live account, cloud or automatic posting.
Existing-source enrollment always holds overlap; full provider workflow remains open.
Contract: [provider-sync-sql](../contracts/provider-sync-sql.md).

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Review | AGENTS, live #70/#100, index/bibliography/provider/import/review/worker contracts and SQL 9–11; official Plaid sync selected sections refreshed. |
| Design | Coordinator approved dedicated session-bound sync role, existing-source synthetic enrollment, immutable revision extension, complete-set publication and explicit 10-page/1000-change/2-MiB bounds before DDL. |
| Implement | Migration 12 pair/checksum; owner enrollment/Item state, five worker functions, atomic observations/lineage/corrections/cursor receipts; original posted anchor and bounded exception references; tagged source namespace reads. |
| Review fixes | Independent review identified numeric insignificant-zero parity, opaque timestamp offsets, nullable readiness, decoder unknown enum, empty optional text and unsafe empty initialization tails. Fixed with named actual-PG regressions. Current/inherited privileged role checks and stale claim replay tightened. |
| Focused proof | Disposable PostgreSQL 17 migrated 1–12; **15 top-level groups + 10 named subtests PASS**. Decoder output is exercised through actual SQL; two concurrent completions and completion/signal contention use separate authenticated sessions. |
| Default gates | `make check`, `make test`, `make build` PASS with Go 1.26.9 and isolated caches. |
| Handoff | Shared runner/Make/CI/schema-version fixtures belong to coordinator; copied runner is not part of this commit. Coordinator runs combined migration/ledger/managed-role/application gates after integration. |

## Current evidence

Executed on 2026-10-13 from `build/sync-state` with:

```sh
PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH \
GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache \
bash scripts/test-sync-state.sh
```

Docker required normal sandbox escalation because the host credential helper cannot
run inside the sandbox. Only private disposable synthetic PostgreSQL databases and
fixed fictional trust-network identities were used. No credentials/provider payloads
or inherited DSNs were read. Evidence log: `/private/tmp/jayburd-sync-sql-proof.log`.

Top-level tests:

- `TestSyncPublicationAtomicity`: private partial pages, explicit final publication,
  distinct identical purchases, exact added replay and immutable completion replay;
  no journal/outbox mutation.
- `TestSyncRollbackAndDuplicateIdentity`: unknown modification and repeated ID reject
  all publication; cursor/evidence remain unchanged.
- `TestSyncFenceAndWakeup`: same live claim, expired restart from C0, stale worker
  rejection, idempotent signal and pending generation preservation.
- `TestProviderSourceLifecycle`: pending/booked explicit lineage across pages,
  removal history, changed unposted holds and immutable revisions.
- `TestSyncAuthorityAndState`: consent grant revocation/regrant, pause/resume,
  current replication/mixed-role denial and authorized receipt replay.
- `TestSyncMoneyAndPageValidation`: exact insignificant-zero lexemes, zero holds,
  changed page conflict; five sign/precision/currency/exponent/scope subtests.
- `TestSyncDecoderPublication`: real `plaid.DecodePage` → SQL with signed exponent,
  empty optional descriptions, newline text and opaque +23:59 timestamp evidence.
- `TestSyncPostedEvidenceRemainsOriginal`: explicit synthetic owner fixture models
  a previously approved provider namespace; real draft/post then two changes and
  removal retain original amount/evidence/journal and one ledger outbox event.
- `TestSyncLimitsAndNullInput`: null status/manifest/fence, invalid times, page cap,
  initialization readiness and exact empty-page replay.
- `TestSyncPrivileges`: exactly five worker function grants, no table/private/helper/
  owner/API authority; enrolled stream identity cannot be rewritten.
- `TestSyncCurrencies`: independent JPY/KWD exact amount/minor-unit assertions (two
  subtests); no FX substitution.
- `TestSyncConcurrentCompletionAndSignal`: two real sessions return one receipt;
  concurrent late signal remains pending without duplicate source effects.
- `TestSyncWholeSetBounds`: change/byte/page limits reject before publication (three
  subtests); staged incomplete work cannot advance the committed cursor.
- `TestSyncRejectsEmptyInitializationTail`: nonempty first page cannot end with an
  empty initialization response and reset the cursor.
- `TestSyncSourceProjectionAndBinding`: typed provider namespace without fabricated
  CSV fields, stale binding/claim rejection, wrong scope and disabled service denial.

Default log files are `/private/tmp/jayburd-sync-check.log`,
`/private/tmp/jayburd-sync-test.log`, `/private/tmp/jayburd-sync-build.log`.
Independent read-only review reports no remaining blockers in this bounded slice.
This evidence is not a 50,000-change capacity test, provider qualification, real
ownership attestation, live retention policy or ISO conformance certification.

## Remaining work

#101 real connection/token lifecycle and provider-first enrollment; durable exact raw
object retention/access/purge; approved overlap release; actual provider client/decoder/
SQL composition; dispatch/catch-up/webhook delivery; changed-value inspection and
reviewed correction workflow. Pending or corrected evidence cannot be approved by
inventing source IDs, resetting cursors or silently editing posted journals.

Operational bounds are 10 pages/1000 changes/2 MiB normalized complete set, 90-second
lease and 24-hour staged expiry. Larger sets require a reviewed scaling design and
measurement. No partial-set adoption, lease renewal, automatic cleanup or scheduler
is implemented. Normative accounting/live-use policy gates remain unchanged.

## PR handoff

Own files: migration 12 pair, two checksum entries, `internal/syncstate` integration
tests, contract and this runbook. Migrations 1–11 are untouched. Decoder prerequisite
is a separately reviewed commit; coordinator owns its shared integration. Stage only
owned files; run `bash scripts/check-stage-runbook.sh --staged` and `git diff --check`
before committing. No push, cloud deployment or external provider calls from this lane.
