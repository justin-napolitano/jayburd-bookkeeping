# Posting policy and contract review

<!-- stage-runbook: true -->

## Status and boundary

Owner requested the next policy/contract PR on 2026-10-11. Branch
`docs/posting-policy-contract` is based on #18, commit `2295367`; #18 remained open
at final base check. Scoped work under #1/#2; neither broader issue is closed.
No product SQL, server, live policy approval, tax election or deployment.

## Stage ledger

| Stage | Result | Next action |
| --- | --- | --- |
| Verify dependency | #18 open; clean parent branch | Stack PR; retarget after merge |
| Review sources | P01–P04 distinguish textbook, PostgreSQL and US GAAP access limits | Preserve gaps before real use |
| Select development policy | ADR 0005: accrual/USD, UTC, exact half-even, reviewed synthetic spot rates | Owner reviews PR |
| Finalize posting target | SQL authority, interfaces, replay equality, archive/revoke locks, failure/test matrix | Implement bounded synthetic proof after review |
| Align API | 0.2.0-review, identity/spot examples, 500 errors, explicit deferred close | Validate and review |

## Current evidence

PASS `make check test build`: doc/context links, stage-runbook structure,
migration-file checks, Go vet/race tests and both tooling builds. No new Go or SQL
implementation. Commands used Go 1.26 with
`GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache`.

PASS one-off API inspection: valid JSON, all 80 internal references resolved,
five operation stage/error annotations checked. PASS full specification validation:

```sh
uv run --python 3.12 --with openapi-spec-validator==0.7.2 --no-project python -c 'import json; from openapi_spec_validator import validate_spec; validate_spec(json.load(open("api/openapi.json")))'
```

PASS opening and spot request examples against `PostJournal` using the same
validator environment's Draft 2020-12 validator and resolved local references.

The validator is an ephemeral review tool, not a project dependency or default CI
requirement. This is specification evidence, not HTTP or PostgreSQL behavior.

Before commit: `bash scripts/check-stage-runbook.sh --staged` with this runbook
staged alongside all contract changes.

## Remaining work

Next implementation PR must execute the named PostgreSQL acceptance matrix,
including failure/COMMIT injection, actual runtime-role grants, concurrency and
fixed fixture balances. A synthetic actor adapter cannot approve live identity.
Real-LLC policy, ASC review, opening balances, accountant export/cutover, close
evidence and full FX position lifecycle remain separate gates.

## PR handoff

Review [ADR 0005](../decisions/0005-development-accounting-policy.md),
[posting boundary](../contracts/posting-boundary.md) and the aligned OpenAPI.
No migration impact. Merge #18 first, then retarget this stacked PR to main and
confirm only this contract change remains. First implementation has no close
endpoint, worker, live rate provider or public deployment.
