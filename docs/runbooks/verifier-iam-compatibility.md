# Verifier IAM compatibility delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue272, lease VERIFY272-IAM-COMPAT-20261023T1820Z. Owner requested the narrow
fix after read-only metadata identified Google's SET-enabled IAM marker edge.
Dedicated branch issue/272-iam-verifier-compat from68612cb; sole migration31
writer. No live SQL modification, rollout, provider call or comparison authorized
by this source evidence. Human review precedes release.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Diagnose | Safe attributes; exact cloudsqliamserviceaccount SET edge caused rejection | Preserve managed authentication |
| Decide | verifier-iam-compatibility contract, official Google/PG sections, alternatives | Implement narrow exception |
| Validate | Core, migration, real-role bootstrap, ledger and full comparator PASS | Exact-head CI and review |
| Hosted | Not applied | Reviewed merge/build/release, then retry bootstrap |

## Current evidence

Migration30 and prior grants remain unchanged. This delivery adds a forward
predicate replacement and bootstrap compatibility; no financial schema or
wire-format changes. Default commands stay offline/provider-free.

Core make check/test/build PASS with documented cached Go1.26.5 and temporary
modfile; tracked1.26.10 remains unchanged, canonical CI still required. Migration
fresh/replay/latest down-up and immutable-history checks PASS. Bootstrap roundtrip
preserves function OID/owner/ACL; positive managed-marker login/replay and marker
ADMIN/NOINHERIT/LOGIN/CREATEDB/parent/table/column/function/unrelated-SET denials
PASS. Marker SET cannot read financial tables, create DDL or invoke comparison.
The first new unsafe-marker test exposed nullable scalar role lookup: changed
to IN membership so a missing/unsafe marker yields false, retaining rejection.
No failed outcome was waived. Docker sandbox denial used approved retry. A later
Go1.26.10 download reset interrupted ledger tests; rerun uses the same documented
cached toolchain without modifying repository pins.
Final cached-toolchain ledger-test and sandbox-verification-test PASS, including
exact comparison, malformed inputs, review identity, foreign history, scope pins
and concurrent revocation. The final real-role test independently calls the SQL
predicate for every unsafe marker case, so bootstrap-only rejection cannot mask
a permissive migrated predicate. Core check/test/build rerun on final SQL PASS.

## Remaining work

Exact-head canonical CI and native PR review remain. Actual managed marker safety may still reject: never
repair its role or widen policy silently. API expiry issue281 and release283
remain separate. Preserve the existing UNKNOWN browser setup registration.

## PR handoff

Refs272/88. Migration31 requires separately reviewed trusted-image rollout;
merged SQL immutable. Record final validation and limits before handoff. No
Production/live-account/automatic-posting claim.
After migration31 applies, a status image embedding only30 cannot validate its
checksum set. Rebuild trusted artifacts and prepare fresh reviewed release
metadata; do not reuse PR283's old migrator image to verify31.
