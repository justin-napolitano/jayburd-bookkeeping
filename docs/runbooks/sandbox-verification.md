# Scoped Sandbox verification delivery

<!-- stage-runbook: true -->

## Status and boundary

Issue #272; lease `PLAID-88-WAVE2-20261022T1930Z`, expires2026-10-22T23:59Z.
Branch `issue/272-verifier`, base `24f263ef2cde697cf0f8cf51c1176d5c1521e679`.
Sole SQL writer reserves migration30 after coordinator checkpoint. No cloud,
provider, credential, raw-data or GitHub operations. No actual mandate provisioned.
Source consent/release authorization are separate and are not renewed here.

## Stage ledger

1. Read fresh issue272/85/87/88 exports, index, bibliography, database foundations,
   standards, existing proposal and runtime/permission contracts. Coordinator
   approved exact schema, TTL60min, combined-compare-only grant and actual-role
   semantics before implementation. [Contract](../contracts/sandbox-verification.md)
   records decisions, source applicability, alternatives and acceptance limits.
2. Added forward migration30, safe capability precreation in existing bootstrap,
   finite immutable scope/digest mandate, owner metadata receipt/enroll/revoke and
   private current-authorization checks. Only the combined comparison is externally
   granted. No general snapshot, table SELECT or posting privilege.
3. Wrote independent fixed lifecycle/history fixture and separate identity map;
   actual HTTP adapter/decoder/restricted SQL publishes provider inputs. Two equal
   inflows remain distinct identities. Raw expectations are not derived from
   generated/provider/decoder output and only their SHA256 pins persist.
4. Added real-login proof for exact comparison, every financial/state/date/unit/
   lexeme/review/lineage mismatch, complete-set omissions/extras, unknown/duplicate
   fields/map identities, size bounds, current role/permission/expiry/generation
   failures, revoke replay and explicit controlled posting invalidating zero counts.
5. Coordinator review identified review-collapse risk: enforce one observation per
   review, one evidence entry and exact observation/namespace/provider identity.
   A disposable owner bad-merge fixture retains FKs but disables only the immutable
   assignment trigger to represent corrupted/future merge behavior. Runtime has no
   such authority. Counter output is bounded/saturated1001; fresh enrollment checks
   journal.posted outbox only. SQL catches paused streams and expiry at result time.

## Current evidence

Validation results recorded after final execution below. Local Go1.26.5 uses the
external reviewed modfile/cache; tracked Go1.26.10 unchanged. All data is synthetic.
Initial proof iterations corrected unmerged PL/pgSQL syntax, alias collision and
explicit nullable-history fields; no merged migration was changed. An attempted
fixture-only consent timestamp update correctly hit the existing Item generation
guard and was removed; expiry uses actual finite mandate time and generation pins.

## Remaining work

After coherent SQL proof/commit, release the migration reservation to coordinator
and add bounded Go adapter/CLI. Its four-second deadline must begin before BeginTx;
SQL does not claim that function-local statement_timeout bounds a direct caller's
already-running statement. Root owns connector/component, shared CI/Makefile/index
and final integration. Hosted identity/bootstrap/image/manifest/fixture/interval
execution remain separate explicit authorization. #272/#88 remain open until actual
hosted proof; Production access status does not authorize any real account.

## PR handoff

Links #272/#85/#87/#88. Migration30 is new; empty-only down/up is supported, retained mandates/receipts block downgrade; merged SQL immutable.
No raw expectations/map persisted beyond fixed synthetic repo fixtures and mandate
hash metadata. This slice cannot establish nonzero-book before/after causality or
institution behavior. Required core/migration/ledger and focused proofs precede
commit; staged runbook guard and diff check run before every commit.

SQL slice final local evidence:

- `bash scripts/test-sandbox-verification.sh` PASS: six named top-level groups,
  four pin subcases. Includes full history, exact current review identity, equal
  purchases kept separate, deliberate review collapse, other-namespace extra,
  1001 unique rows, duplicate/unknown input, raw digest probe denial, real mandate
  expiry, finite-consent limit, source/binding/Item/requested-generation changes,
  unsafe role/direct-column access and a comparison blocked across committed revoke.
  Controlled balanced explicit posting yields journal/outbox1 and fails zero proof.
  Log `/private/tmp/jayburd-272-verification.log`; no skips.
- `make check test build` PASS, log `/private/tmp/jayburd-272-sql-core.log`.
- `make migration-test cloud-sql-test ledger-test` PASS, log
  `/private/tmp/jayburd-272-sql-permissions.log`. The initial unconditional down
  failed the newest-migration empty down/up gate; replaced with an empty-only
  reversal that refuses retained mandates/operator receipts. Focused proof also
  asserts retained-evidence downgrade refusal. Existing product migrations unchanged.
- Independent peer review found no confirmed SQL blocker; identified missing named
  evidence for pins, foreign history, array bound and concurrent revoke was added
  and passed. Staged runbook guard and diff check before SQL commit.

## Adapter and command follow-up

6. Added `internal/sandboxverification` and explicit `cmd/sandbox-verify` command.
   Default invocation is offline/inert. Closed metadata manifest and fixed sibling
   files are bounded/hash-checked before transport. Actual IAM identity is checked
   before comparison; only committed validated Boolean/count results are emitted.
   Four-second deadline starts before BeginTx; failed/uncertain commit returns no
   prepared result, rollback is independently bounded, and there is no auto-retry.
   Root connector dependency a1a4116 is present locally as d703b33; do not pick it
   again. Root owns build/CI/index integration and any hosted approval.
7. Coordinator authorized a follow-up of unmerged migration30's four exact
   profile/fixture-ID predicates to typed `ledger.text_value` plus null-safe
   `IS DISTINCT FROM`. Existing `ledger.keys` already rejected null fields; this
   hardens string typing/predicate clarity, not a demonstrated bypass. Added all
   four fixture/map null-profile/null-ID negatives. Nullable history fields remain
   intentional and presence-checked. No other SQL contract changed.

Follow-up validation results are recorded below after execution. Hosted inputs,
identity/bootstrap and execution remain unprovisioned; #272/#88 remain open.

Follow-up evidence at source handoff:

- `go test -race -count=1 ./internal/sandboxverification ./cmd/sandbox-verify`
  PASS; `/private/tmp/jayburd-272-go-focused.log`. Tests cover identity/transport/
  cancellation/uncertain commit, closed output parsing and CLI input/cleanup paths.
- `bash scripts/test-sandbox-verification.sh` PASS with actual Go adapter plus all
  six SQL groups/four pin subcases and four null metadata negatives;
  `/private/tmp/jayburd-272-go-integration.log`.
- Latest core attempt reached existing HTTP tests but sandbox denied loopback
  bind (`operation not permitted`); escalated retry was interrupted before a
  session/result returned. No latest core PASS is claimed. Coordinator completes
  required core/build and packaged-command gates before PR handoff.
- Follow-up disposable migration/cloud-SQL/ledger gate log:
  `/private/tmp/jayburd-272-followup-permissions.log`; terminal status reported to
  coordinator separately. Earlier full SQL gates remain recorded above.
- Source frozen for coordinator handoff; staged runbook guard and diff check run
  immediately before commit. No hosted execution or real financial input used.
