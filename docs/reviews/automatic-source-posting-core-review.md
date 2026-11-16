# Independent source review —334 (in progress)

Reviewer yaml_rules. Read-only sibling jayburd-auto-post; no core edits or financial execution. Root owns integration/runtime; compact_rules owns sole SQL49. Review is not final while implementation/tests are incomplete.

Reviewed baseline AGENTS, database foundations, standards, artifact-backed/source-review/posting contracts, proposed automatic-source-posting contract, draft49 and internal/autopost service.

## Checkpoints sent to implementation owner

1. Current shared-core draft admission mismatch: unchanged source_post_authorize can accept a new automatic-origin draft into a durable interactive source_post command, but changed commit_source_post(command,NULL,...) rejects that draft's nonnull automatic_operation_id. Ensure manual admission rejects before durable registration or performs an explicit new manual-origin review. Otherwise an unexecutable saved command can block the automatic principal gate. Existing render_review_draft omits origin, so ordinary UI cannot distinguish it yet. Inference from exact current helper call chain, not an executed exploit; actual restricted/API regression requested.
2. Prepare stores operation before create_source_draft because draft FK is immediate; builder increments source/review revisions while operation context/hash is immutable. Execution must account for its own preparation increments rather than treating every operation as stale. Design checkpoint pending implementation.
3. Contract proposed errors P2900–P2904 vs SQL/Go P2A00–P2A04 need one frozen mapping across HTTP/OpenAPI.
4. Fixed JPY123/KWD1.234 original-unit holds must accompany EUR10.00 hold tests; no USD substitution/rounding or fabricated rate.

## Required final evidence

- Actual restricted role admission/helper/table denial and cross-actor/book isolation.
- Current authority rechecked before configuration, discovery/payload, prepare, execute and committed replay; revocation after prepare produces zero effect.
- Exact source/book/binding/policy/account/date/period pins and equal-match ambiguity holds.
- Source-to-journal immutable unique link, exact operation recovery after lost prepare/execute commit; changed rule ID/policy/reordered worker rounds cannot duplicate.
- Manual draft_ready protected; protected analytic labels neither confer financial treatment nor get overwritten.
- Shared interactive saved work defers auto; auto never fabricates/executes/acks interactive commands.
- Atomic journal + source/review projection + link + immutable receipt/outbox; injected failure at every boundary rolls back every financial effect.
- Source50→55 after posting retains original50 and correction hold, no new55 journal/unbound reversal.
- Supported USD outflow/inflow precise debit/credit; transfer/card/loan explicit holds; EUR/JPY/KWD original units retained and held.
- Empty migration rollback/reapply plus refusal for retained financial effects; schema1–48 untouched.

## Current execution limitation

This reviewer has not run financial SQL. Local automatic-category journey is blocked before startup by listener permissions, Docker socket denied and pinnedGo cache writes denied. Canonical actual restrictedSQL/Go/Next evidence is mandatory; static review or green unit mocks do not replace it.

## Substantial prepare/execute checkpoint

Draft49 now includes full prepare/execute/status/work functions. Execution compares fresh source/review/book pins to its own created draft, so the earlier own-prepare increment checkpoint appears addressed without mutating operation context. New authorize_source_post_core(origin) plus manual source_post_authorize(NULL) rejects automatic-origin drafts before manual command registration, addressing the earlier admission mismatch in source; actual tests still required.

New current-source isolation finding (sent to compact_rules and root): status and prepare obtain last_operation from the whole book, authorize current policy source scopes only, then render historical operation IDs/review/journal/reason. After rotating policyS1→S2(or empty rules) and revoking oldS1 read/review, an actor retaining book/S2 grants can receive oldS1 operation metadata. Old-policy prepared-operation staling path also needs own-source disclosure authorization. Execution itself correctly binds operation.actor_id and calls automatic_accounting_source on the operation's actual source before replay.

Requested fix: current-authorized operation-source filter or explicit safe omission before every returned operation/last_operation, respecting lock order; actual policy-source rotation/revocation/different-actor regression. No exploit or financial SQL executed by reviewer.

## Source corrections observed (tests still pending)

- Automatic-origin draft admission now uses private authorize_source_post_core(origin), with the manual wrapper passing NULL. New automatic-source-draft-v1 marks auto provenance without changing old manual draft keys. Root informed that strict existing source-review/API/UI validators need the new distinct arm.
- Historical operation visibility now checks actual actor/entity/book/source grants. Current policy source authorization remains mandatory. Prepared counts are policy/visibility scoped, and removed-source operations are hidden rather than serialized.
- Visibility no longer acquires historical source locks after current sources: a stable EXISTS check relies on the caller-held book lock and supported grant/state writers' book FOR UPDATE fence. This avoids the S2→oldS1 vs source-listS1→S2 ordering inversion identified during the first fix.
- Runner Round records the first per-scope error but continues later independent books, addressing the bad-first <=10-book fairness checkpoint. Execute response validation occurs inside the committing transaction; unknown commits do not repeat via the known-abort retry loop.
- Execute uses exact immutable draft post-preparation pins plus current observation revision/policy and current actor/source authorization. Policy disable/edit or source/book/binding changes terminate stale rather than posting a revised interpretation.

These are static source observations, not final acceptance. Required actual privilege/concurrency/failure/precision tests and rollback migration remain to be reviewed before a final verdict.

## Receipt recovery checkpoint

Added receipt_observed_at is a monotonic machine-consumption marker only. Prepare now selects the earliest currently visible unobserved terminal receipt before creating another source operation. Worker validates the exact recovered receipt before a distinct observe transaction; execute validates its exact operation receipt before its financial Commit, then observes separately. Observation itself checks current actor/entity/book/source grants and does not post or human-acknowledge anything. Unknown-observe commit may safely allow the next source after the marker actually committed. Explicit disabled/current-source policy authorization still precedes receipt recovery, so disable or current-source revoke delays worker recovery by design.

Actual restricted-pgx lost-reply test source now wraps the real Commit, loses exactly one successful prepare/execute/observe response, restarts Service, checks original draft/operation/source-link and fixed eventual journal counts. This is the required form of test, but reviewer has not seen executed SQL results. No remaining source blocker identified in this new path.

Current tests still need expanded actual restricted helper/table ACL checks (including authorize_source_post_core and observe), correction50-to55 retention, full source-state/concurrency/saved-work cases, and empty rollback/reapply+retained-history refusal. Disposable shell runner currently only migrates up and runs Go tests; no migration down cycle is present. Parent/implementation owner notified. Review remains in progress.

## Expanded proof source checkpoint (2026-11-16)

Reviewed new real local-adapter provider correction50-to55 and pending/zero/precision/overlap cases, independent-session concurrent prepare/execute, exact saved interactive command/reservation snapshots, retained-history down refusal, and shell pinned migration49-to48-to49 before fixture templates. Found ordering bug: two-row review fixture plus outflow-only mapping could select unmapped inflow first in concurrency/down tests. Writer corrected both fixtures with income+expense mappings. Found provider-holds proof could pass via silent exclusion; writer added exact observation/review/held operation counts, current attention1/pending0 and fixed reasons. Correction now additionally checks operation_receipt count1.

Verified migration29 initializes pending/zero/precision rows as correction_hold. Under existing49 reason priority they deliberately yield source_correction_required, while overlap yields overlap_review_required. This is honest source behavior, not evidence that dedicated pending/zero branches execute for provider data. Requested independent original source_state/direction/amount lexeme assertions distinguish the fixture cases. Current whole-observation snapshots protect retained evidence across rounds.

Broad API helper/table DML denials and all seven entrypoint denials for observer/publisher are now present in source. No additional financial product defect found in this targeted review. All these expanded SQL tests remain unexecuted in available environment; private CI billing/local pinned runtime limitations block product acceptance. Static/source review may proceed to source checkpoint, but does not authorize runtime activation or claim financial proof.
