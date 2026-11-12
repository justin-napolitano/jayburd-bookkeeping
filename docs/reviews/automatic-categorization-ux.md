# Automatic categorization: UX investigation — 488

## Boundary and evidence

2026-11-11, frozen consolidation base `bea304ec`. This is a design and fictional
product review, not implemented automatic behavior or owner acceptance. Only this
document and ignored synthetic screenshots are authored by this lane so far.
No runtime/spec, dependency, lockfile, API, SQL or cloud changes were made.

Read AGENTS, the automatic-categorization stage, current categorization/query/rule
contracts, current collection components and frontend source review. Existing
white/green visual identity and native accessible controls remain appropriate.
Reuse the previously reviewed frontend sources as design guidance; no new claim
about a competitor's current product or external standard is introduced.

The owner's problem is routine per-transaction work. The default proposed scope
is private **analytic categories**, not accounting treatment, business purpose,
deductibility, ownership, journal creation or automatic posting. Existing explicit
artifact-backed accounting review remains unchanged.

## What the application currently does

- `CollectionReview.tsx` shows description, account, merchant, source/assignment
  and journal state. Effective owner topic/tags and rule provenance are inside
  each row's **Labels and provenance** disclosure. Categories are not visible in
  the closed list, even when a current saved decision exists.
- `RuleViews.tsx` deliberately distinguishes current inert suggestions from
  `rule_application` accepted provenance. Suggestions never prove persistence or
  the historical cause of a saved category.
- Existing rules are owner-defined catalog editions. Applying them to historical
  rows requires selection, preview, saving the review, explicit confirmation and
  acknowledgement. This is safe current behavior but does not satisfy routine
  automatic categorization.
- Current v3 read projections and saved v1/v2/v3 command recovery must survive.
  Existing manual primary topic and tags are protected independently; a stale
  manual field makes the rule engine abstain for the whole row. Equal-rank
  conflicting rules abstain; account rules precede workspace rules.
- A new private catalog has no owner topics/rules. Merely running the existing
  matcher in the background would not classify routine rows without a useful,
  evidenced taxonomy/map. Historical missing PFC cannot be invented by the UI.

## Proposed low-effort flow — pending server contract freeze

### Normal transaction list

The routine path is: import completes → server durably categorizes eligible
observations → the current list displays the saved category automatically.
No selection, per-row review, saved-command panel or acknowledgement is required
for an ordinary automatic category result. Refresh/read itself performs no write.

Show an always-visible short category beside the description, e.g.
`Travel · Automatic`, `Supplies · Your choice`, or `Needs category`.
These are product-copy examples, not hardcoded mapping outputs. Resolve names
from the current authorized saved projection. Never derive a saved category from
`rule_suggestion`, provider confidence, merchant text or current-page contents.
Unknown, unavailable, stale and explicit Uncategorized remain distinct.

Keep the amount/date/account visible and exact. Put journal status in its existing
separate accounting dimension. Do not turn a category badge into an assertion that
an entry was posted, that all bank history arrived, or that an expense is deductible.

Offer **All transactions** and a contextual **Needs attention** filter/count. A
positive count comes from a complete server query, not `page.items` or the number
of visible chips. Keep filters and paging server-owned. Do not use a floating
cross-currency amount total as an automation success metric.

### Exceptions instead of an inbox for every row

| Server-evidenced state | Customer presentation | Action |
| --- | --- | --- |
| Current durable automatic result | Category name; quiet Automatic origin | Nothing required; optional Change category / Why this category |
| Current manual choice | Category name; Your choice | Optional change; automation cannot overwrite it |
| Processing not yet completed | Categorizing… | No false success, no requirement to select the row |
| No approved match/missing old provider category | Needs category | Correct this item; optionally create a reusable rule |
| Equal-rank conflicting rules | Rules disagree | Explain the competing named conditions and let the user resolve them |
| Stale manual choice/source or retired output | Check your earlier choice | Retain historical context; no silent automatic fallback |
| Read/classify denied or source projection unavailable | Category unavailable with current access | Remove private payloads; follow current recovery/access behavior |
| Retryable background failure | Categorization delayed | Current server status; no invented retry timing or success |

Processing is not the same as an exception or failure. If the server has not
provided a distinct durable status, do not invent this display from an empty
category or a slow HTTP response. Exception reasons need closed typed authority.

### Correct once; stop repeating routine work

**Change category** opens the existing focused row/detail context. Default scope
is this exact composite source observation/revision. A successful manual choice
is durable and protected; the UI does not relabel it automatic on refresh.

Offer **Use this for similar transactions** as a separate optional rule step.
Seed an understandable exact-merchant rule when the authorized original merchant
exists; otherwise require the user to choose an actual supported condition.
Show account/workspace scope, conditions and proposed category before saving.
Keep regex, numeric priority and compound conditions under an Advanced section.
Never create a broad rule silently from a one-row correction, infer business
purpose, or choose a conflict winner by rule UUID/order.

Reuse the existing immutable rule editor, selected preview and command recovery
for owner-initiated changes until a reviewed replacement contract exists. The
current confirmed-acknowledgement seed fencing remains: unknown acknowledgement
must not open or submit another rule. A manual exception can require explicit
recovery; routine automatic results should not create thousands of user-ack gates.

### Why this category

Reuse the current detail view with a concise explanation above technical evidence:
`Your choice`, `Matched your account rule`, or `Mapped from the bank category`
only when immutable stored origin supports it. Show the actual matched condition
and saved category name where authorized and historically accurate.

Current rule names can explain a current edition. They cannot be used as names
of an older edition merely because the UUID matches. If historical names are not
provided, say **An earlier saved rule** rather than inventing a name. UUIDs,
revision pins, raw confidence enums and repeated implementation prose should not
be routine customer content. Preserve exact internal pins for requests/recovery;
removing their display does not remove authority checks or saved versions.

### Automation management and existing history

One compact status in Manage rules can say **Automatic categories are on** only
when current server state confirms the enabled scope. Read-only access cannot
self-enable classification. An unavailable capability must not be rendered Off
or offered as an implicit permission grant.

The server design must specify default taxonomy/mapping, active scope and whether
existing imported history is eligible automatically. A bounded background sweep
must cover the authorized dataset, not only the open page. If status/progress is
shown, it must use server-owned counts/status. A long-running job must not require
the user to keep the tab open. Do not auto-execute or auto-acknowledge existing
interactive label/book commands to make a batch appear finished.

Pause/retry/backfill controls are conditional on a reviewed actual port. No
endpoint, job schedule, response field, public operation or fake client store is
assumed here. Existing source/taxonomy/tag/ruleset/map/manual pins and current
revocation/tenant fences remain server authority.

## Concrete engineering findings from fictional product review

| Finding | Evidence | Priority / recommended scope |
| --- | --- | --- |
| Saved category is hidden in normal rows | Inspected 390/1024/1440 Transactions screenshots and row markup | High: always-visible persisted category is necessary for488 to feel automatic |
| Routine apply is still a multi-stage manual workflow | Existing rules browser case selects2 rows, previews, saves, retries unknown prepare, confirms and acknowledges | High: replace routine action with durable automatic effects; retain explicit owner edit recovery |
| Rule explanations expose raw IDs/revision jargon | 390 rule-result screenshot and WinningRules source | Medium: customer explanation should use evidenced names/reasons; preserve exact machine identity internally |
| Selection pushes the first row down at mobile width | After selecting, action toolbar is at y698; first selection moves to y897; initial row fold assertion still passes before toolbar insertion | Usability observation, not a failed test. Keep exception correction compact and avoid using selection as routine automation |
| Empty suggestion detail repeats multiple technical/negative sections | 390 focused detail screenshot: no saved rule, no current rule, inert-suggestion explanation | Medium: contextual provenance/exception explanation instead of repeated empty sections |
| Source detail uses raw account UUID where chart metadata is not supplied | Fictional journal-detail screenshot | Separate polish limit: do not fabricate account names; use an honest neutral fallback if changed later |

The fictional setup screenshot includes disabled starter-book guidance when its
fixture omits bounded USD metadata. This is an honest fixture/access state, not
proof that real setup is broken. The standalone review Accounts view is explicitly
fictional presentation and does not qualify actual account management.

## Executed fictional checks and artifacts

Locked `npm ci` completed without package/lock edits. Existing component-browser
specs ran unchanged:

`tests/frontend-flow.spec.tsx`, `tests/collection-rules.spec.tsx`,
`tests/collection-usability.spec.tsx`, `tests/collection-filters-detail.spec.tsx`,
`tests/collection-onboarding.spec.tsx` — **16 PASS in2.7s**.

Evidence includes five nav widths390/768/1024/1280/1440, accessible44px navigation,
keyboard Space selection, post-commit review focus, detail→Back focus return,
no horizontal overflow, exact original amounts, filter intent, rule conflict and
partial-manual preservation, unknown prepare/same-key retry, lost acknowledgement
seed gating, no implicit executions, no browser storage, and composed private
setup/adoption recovery. These are mocked-port component journeys; they do not
prove current server automation, real identity, restricted SQL or owner usability
acceptance. This review is engineering interaction/screenshot evidence, not an
unaided user study or hosted authenticated audit.

Preserved18 fictional PNGs outside test-results in ignored
`.local/automatic-categorization-review/`. Visually inspected:

- `frontend-flow-390px-integr-1f07b-ion-controls-stay-reachable-chromium/frontend-flow-390.png`
- `frontend-flow-1024px-integ-c1f4e-ion-controls-stay-reachable-chromium/frontend-flow-1024.png`
- `frontend-flow-1440px-integ-1d143-ion-controls-stay-reachable-chromium/frontend-flow-1440.png`
- `collection-rules-rule-conf-cff12-ut-implicit-writes-at-390px-chromium/fictional-rule-conflict-result.png`
- `collection-usability-synth-7671d-on-usability-review-at390px-chromium/fictional-first-selection.png`
- `collection-filters-detail--71433-nd-keyboard-return-at-390px-chromium/fictional-source-journal-detail.png`
- `onboarding-mobile.png`

The green/white layout is readable without clipping in these captures, with
usable mobile Menu and normal-width sidebar. Category visibility and explanation
burden remain the concrete product gaps despite the passing checks.

## Design decisions frozen before implementation

1. Exact persisted automatic decision projection and current/manual/provider/rule
   provenance; distinguish current suggestions and historical accepted effects.
2. Default analytic taxonomy and finite mapping with honest missing/historical
   PFC behavior. Never invent financial purpose or silently perform a provider call.
3. Dataset-wide eligibility, exception reasons/filter/count semantics, background
   progress/retry bounds and manual-field protection during concurrent changes.
4. Current read/classify/automation permissions, configuration/recovery semantics
   and behavior under unknown commit, revoked scope and restored sessions.
5. Existing-history policy and component/service adapter ownership. Source-backed
   accounting posting remains explicitly confirmed under its current contract.

The automatic contract now freezes these decisions. The UI implementation and
its local evidence follow below; prerequisite consolidation487 still cannot be
bypassed.

## Implemented flow and local evidence

Saved category and readable origin are visible on each closed row. Current manual
choices remain marked Your choice; retained rule names come from their actual
edition, and malformed/unsupported metadata cannot become an automatic success.
The compact Automatic categories bar exposes a whole-result Needs attention
selector and keeps secondary configuration in native Settings disclosure. The
additive v3 `topic.kind=needs_attention` selector excludes work still awaiting
automatic categories and pending/removed source facts before pagination; its
badge describes retained exceptions rather than every unclassified row. Old
catalogs without the capability marker make no automatic-category requests.

An unknown settings response retains the same body/key and private-work lock;
retry never fabricates a new intent or acknowledges an interactive command.
Active pending work triggers at most six read-only status polls at five-second
intervals per mount. A newly observed committed run refreshes the workspace
catalog once per run, so new taxonomy/labels arrive without per-row work. Polls
stop on scope change, closed view or competing private work. Late replies cannot
refresh another/locked scope, while a settings receipt already in flight is not
discarded merely because the view becomes inactive. No financial request/token
is stored in browser persistence.

The local frontend gate `make web-check web-test web-build web-browser` passed:
API generated-type check, TypeScript,707 unit tests/40 files, optimized Next build
and100 Chromium component tests (98 retained plus2 new automatic journeys).
The22 focused automatic tests include bounded reads, once-per-run refresh,
late-lock fencing, retained unknown intent and inactive-view receipt completion.
They also retain a pending automatic row and pending source fact alongside a
retained exception, then verify that the explicit server selector returns only
the exception without any client-side command effects.
A saved-review focus timing issue found under the full suite was corrected by
focusing its committed heading before paint; the original assertion is unchanged.
This combined run includes the separate lane's16 bridge cases. Real application
proof remains separately recorded by that integration lane.

Visually inspected390px/1440px fictional automatic initial captures. New browser
checks retain keyboard Space selection,44px targets, first selection before844px,
no horizontal clipping, complete-result attention selection and exact unknown
settings retry with competing navigation/actions locked. These are component
journeys with synthetic ports; real Next/API/restricted SQL proof is separately
owned by the integration lane. No hosted, provider, real identity or owner-study
acceptance follows from these UI results.
