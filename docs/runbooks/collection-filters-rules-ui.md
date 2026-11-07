<!-- stage-runbook: true -->
# Collection filters, detail and private rules UI — #242/#414

## Status and boundary

Dedicated UI lane `feat/242414-ui` from `ca1b37e`, worktree `jayburd-filter-rule-ui`. Lease `PRIVATE-WORKFLOWS-20261107` is active through 2026-11-07T14:00Z in [#65](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/65#issuecomment-6703317926). Coordinator owns shared app/HTTP/OpenAPI/server composition; SQL/rule lane owns authoritative store/evaluation. This lane owns collection components, styles, service adapters, focused tests and UI contract. No cloud/provider/private financial data, browser financial persistence or new dependency.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | AGENTS, index, live #242/#414/#65, standards/decision evidence, bibliography, current v2 collection/recovery and bounded evaluator reviewed |
| Proposal | v3 richer selectors and detail pins, authorized-only book references, separate rule catalog and server-evaluated selected results proposed to coordinator and SQL/rule lane; approved exact45 v3 read wire;46 rule wire frozen separately |
| Tooling | Locked `npm ci --ignore-scripts` completed, 233 packages, zero reported vulnerabilities |
| Design | Frontend-design guidance applied to existing native controls/system type; preserve ink #202124, muted #555b63, line #dce0e3, paper #fff and accent #286044. Keep results and selected actions above long lists; put broader filters in a disclosure, use focused source detail with Back restoring row focus |

## Current evidence

Current v2 supports account/date/search, assignment, topic/tag and date ordering. It does not supply institution/owner/specific-book/amount/status filters, detailed journal links or an integrated rule store/evaluator. The merged pure evaluator permits at most 100 rules, eight conjunctive conditions and 20 output tags. Its scope equality grants no authority, and it accepts no amount condition. Existing manual tag editing permits 25 tags. These distinctions must remain explicit in the reviewed v3 contract.

Reviewed sources are the project contracts' named current authority/immutable preview/recovery semantics, the evaluator's actual closed condition/result types and the existing compact-detail focus-return precedent. Internal rule editing/filtering introduces no ISO 20022 exchange or ISO 21378 export; dated ISO 4217 original units and decimal strings remain unchanged, without conversion or accounting recognition. No conformance claim follows. Browser evaluation or direct mutation was rejected because it would duplicate matcher logic or bypass reviewed durable work.

## Remaining work

Migration45 read/UI boundary is implemented; migration46 rule wire is now frozen. Implement explicit preview/save/confirm/cancel/acknowledge, historical v1/v2/v3 recovery and current denial/staleness handling. Add independently expected synthetic unit and desktop/mobile/keyboard/reload browser cases; run required gates proportional to the final composition. Coordinator owns real SQL/HTTP/app proof and final review.

## PR handoff

Refs #242/#414/#65. Manual overrides precede account then workspace rules; equal-rank conflict abstains and stale manual fields require confirmation. Historical apply is an explicit bounded selection. No automatic label, draft or journal from reading or suggestion. Do not merge before coordinator review or infer managed owner activation from source implementation.

Read checkpoint: v3 catalog/query/detail, exact decimal filters, honest missing metadata, server-only sorting and safe journal reader implemented.86 collection units, TypeScript and11 component browsers pass, including two new390/1440 rich-selector/keyboard/focus/Back/remount cases and all original fold/larger-font/tag cases. Independent negative fixtures cover wrong pins/IDs, unavailable anchors, duplicate/foreign journal lines, denial and old-scope late responses. New synthetic screenshots inspected; long references wrap without horizontal overflow. Actual v3 shared HTTP transport and SQL/app proof remain coordinator gates. No46 rule runtime is claimed in this checkpoint.

Read-checkpoint production build passes. Its route list intentionally still reflects baseline v1/v2 because the coordinator's reviewed v3 bridge/Next route changes are not imported yet; actual-client browser/API/SQL acceptance remains blocked on that composition, not inferred from injected tests. Final assignment/journal availability consistency and positive historical anchor guards align with the SQL lane; five detail units and both new desktop/mobile browsers passed after that refinement.

Shared import checkpoint: coordinator-reviewed activation package `02b9ce8` and v3 HTTP/Next/bridge/API transport `13b1882` imported unchanged, independently of the ongoing rule UI edits. The prerequisite package prevents dangling Go imports; no activation action or cloud request runs. Root owns their source/117 shared transport-test evidence. Rule UI is still in progress and is not claimed validated by this import commit.

Rule transport import: coordinator-reviewed `9ce4612` extends the shared v3 preview/command admission and body caps with 134 shared tests; imported unchanged. Rule editor/result/provenance and same-command acknowledgement seed changes remain in progress, separately uncommitted. Current 95 collection units and TypeScript pass; actual-client preview tests await this exact import and final SQL46 implementation before full app claims.

Rule UI checkpoint: complete stable rule editor, server-only matching, explicit selected rule preview/application, separate stored application/current suggestion views, v1/v2/v3 immutable recovery and acknowledgement-bound correction seeding are implemented. The editor lazily mounts one rule's controls, avoiding 100×500 hidden checkbox multiplication. Same-command lost acknowledgement, immediate API replacement before passive effects, old-response fencing and source-omission seed destruction have independent regressions. Definite no-write preview rejection retains the draft; uncertain saved work keeps its exact keys.

Validation: all 606 frontend units, all 82 component browsers, Next type generation/TypeScript and production build pass. Original 390px/larger-font fold, keyboard, 3000-row server paging, cross-engine recovery and all book/connection/report browser assertions remain. Rule mobile/desktop screenshots were inspected; a scattered desktop result layout was made linear/readable, and the affected six browsers passed again. Selected action remains y578.8 on 390px and y577.5 on desktop; native targets and no horizontal overflow/storage assertions remain. The build now includes the reviewed v3 Next route. Root's independent real browser→Next→Go→PostgreSQL rule proof is in progress; these component tests do not establish SQL authority or no-journal invariants themselves.
