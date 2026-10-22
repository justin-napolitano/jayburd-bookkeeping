# Plaid #88 qualification test matrix

## Scope and clearance rule

Owner requested this coverage audit on 2026-10-22. GitHub [#88](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/88)
owns execution state; this document defines cases and evidence. Sources and
review limits: [P88-01–P88-17](../research/plaid-88-test-sources.md).

Current scope: US Transactions, web frontend, one Item per Link attempt,
account-filtered streams, checking/savings/card/loan transaction feeds. Current
hosted acceptance covers seven dynamic types, not savings. Loan binding does
not qualify a Liabilities API integration. No money movement or automatic posting.
The ninety-day request is an intended lookback, not a guarantee of supplied history.

Statuses: **PASS** = the exact named observation was verified; **PARTIAL** =
some evidence exists but the row is not cleared; **OPEN** = no qualifying result;
**BLOCKED** = named prerequisite missing; **N/A** = exclusion with rationale.
Historical synthetic PASS does not clear a required hosted case. Proposed IDs
below are acceptance cases, not assertions that corresponding automated tests exist.

Every executed row must record date, exact source SHA, deployed revisions/image
digests when applicable, named case, fictional fixture and independent expectation,
command/action, result, redacted receipt link, limitations and defect issue.
No raw responses, tokens, cursors, transaction identifiers or financial records
belong in public evidence. A missing result is never PASS. Aggregate UI counts
are weaker than exact fixture/SQL assertions.

## Evidence already available

- **A:** [2026-10-21/22 qualification](../runbooks/plaid-sandbox-20261021.md),
  initial smoke and synthetic suites; preserves the failed browser volume gate.
- **B:** [migration29 rollout and final comparisons](../runbooks/plaid-258-rollout.md),
  runtime source `f334d062e2ee4c572af52b823969c299cd37bf93`, trusted build
  `39715438972` attempt2, API/web revision19 and worker revision7. Final
  read-only comparison recorded at documentation commit `04bebc3` / PR268.
- **C:** [this audit runbook](../runbooks/plaid-88-test-matrix.md), current
  source inventory and fresh synthetic unit/core validation. Integration
  evidence in A/B remains historical; it is not rerun or relabeled by this audit.

## Link and supported clients

| ID / source | Test and exact clearance evidence | Current status / next action | Owner issue |
| --- | --- | --- | --- |
| L01 / 02 | Interactive non-OAuth dynamic Link → verified session → one exchange → reviewed binding → application publication. | PASS for seven dynamic types (B). API-created smoke cannot replace interactive proof. | #88 / #104 |
| L02 / 04 | Successful, exited, failed and incomplete Hosted Link sessions correlate to the verified owner; reject missing/ambiguous/foreign completion. | PARTIAL. Success observed; `TestHostedLinkCompletion`, `TestPlaidHTTPSessionAssociation` cover synthetic rejection. Hosted negative cases OPEN. | #187 / #104 |
| L03 / 04 | Completion is recoverable after lost tab, reload, expired app session or lost exchange response; inspect persisted attempt before safe restart. Test completion delivery mechanism and expiry. | PARTIAL. Inspection recovery observed (A/B); `TestHostedReloadCompletion`, `TestConnectionUncertainEffects` are synthetic. Signed SESSION_FINISHED absent; polling alternative needs reviewed disposition. | #187 |
| L04 / 07 | Exit/cancel and expired/invalid Link token recover without duplicate exchange; replacement begins only after inspecting prior attempt. | PARTIAL. Existing cancellation/claim tests; actual token expiry and user-exit recovery OPEN. Hosted lifetime differs from SDK lifetime. | #187 / #104 |
| L05 / 06 | Same institution/login/account connected again is detected safely; distinguish legitimate distinct logins and partial account selections. No duplicate import or needless token exchange. | OPEN. Same-attempt replay and stable counts do not establish duplicate-Item prevention. | #104 / #88 |
| L06 / 02 | Simple-user depository/savings enrollment, binding, first acquisition and repeat. | OPEN hosted; savings was not part of the seven-type dynamic acceptance. | #104 / #88 |
| O01 / 03 | Desktop OAuth happy path, return to correct attempt, no unsupported-browser loop. | OPEN. Non-OAuth success does not clear OAuth. | #104 / #88 |
| O02 / 03 | iOS mobile web OAuth, institution app installed and absent. Record real device/browser versions. | OPEN; required before mobile web is advertised. | #104 / #88 |
| O03 / 03 | Android mobile web OAuth, institution app installed and absent. | OPEN; same support boundary as O02. | #104 / #88 |
| O04 / 03 | OAuth update-mode repair after Sandbox reset_login. | BLOCKED on I01; run on each supported web client. | #104 / #88 |
| O05 / 03 | Reduced/revoked account consent and OAuth errors preserve scope; no acquisition of unselected accounts. | PARTIAL synthetic authority denials; actual consent cases OPEN. | #87 / #104 |
| O06 / 03 | Native iOS/Android SDK, universal/app links, app webview redirect. | N/A to current web-only release. Reopen before shipping native/webview clients. | #88 |

## Transactions and exact publication

| ID / source | Test and exact clearance evidence | Current status / next action | Owner issue |
| --- | --- | --- | --- |
| T01 / 09,10,16 | First empty pull remains loading; initial/history readiness advances correctly; request policy and available coverage are visible. | PARTIAL. Seven historical-complete UI statuses (B); `TestPlaidEmptyInitialization`, `TestSyncEmptyAccountItemReadiness` synthetic. Fixture-known ninety-day coverage OPEN; do not infer it from readiness. | #226 / #88 |
| T02 / 10 | Each selected account has its own cursor stream; foreign account/Item and switched filter with reused cursor are rejected. | PARTIAL. `TestPlaidLiabilityAccountPin`, `TestSyncSourceProjectionAndBinding`; hosted persisted-cursor assertions OPEN. | #85 / #87 |
| T03 / 09 | Initial publication followed by unchanged repeat has no duplicate effects. | PASS for bounded UI counts (B): checking69 (33 review/36 held), card55, mortgage47, student6, auto/consumer/LOC47 each, all source revision1. Exact DB replay/no-journal proof still R02/R03. | #88 |
| T04 / 11 | Induce dynamic pending removal + posted replacement across pages; retain lineage when matched, tolerate null match/changed amount/name, and disappearing authorization hold. | OPEN hosted. `TestProviderSourceLifecycle` and decoder state tests are synthetic. Provider Refresh required; another app sync alone does not induce the transition. Student schedule is a separate expectation. | #85 / #88 |
| T05 / 11 | Apply added/modified/removed data, including corrections to already-posted provider transactions, without rewriting ledger journals. | PARTIAL. `TestSyncPostedEvidenceRemainsOriginal` synthetic; actual induced source correction/held disposition OPEN. | #85 / #88 |
| T06 / 11 | Exact quantity/currency retained; precision/pending/zero held when ineligible; no rounding or draft bypass. | PARTIAL. Matrix/fuzz and `TestSourcePrecisionHold`, `TestPrecisionEligibilityForgery` pass in B; actual held warning/no-draft observed. Independent full provider-to-review equality OPEN (R02). | #86 / #88 |
| T07 / 10,12 | Multi-page update plus mutation/error restarts at original cursor; publish only complete set and final cursor. | PARTIAL. `TestAcquisitionMutationRestart`, `TestSyncPublicationAtomicity` synthetic. Hosted multi-page/failure evidence OPEN; if not inducible, record limitation and reviewed fault-injection substitute. | #85 |
| T08 / 10 | Durable filtered cursor survives worker/process restart; empty initial tail cannot erase data; terminal receipt replay does not refetch. | PARTIAL synthetic acquisition/sync tests; hosted restart/SQL proof OPEN. | #85 |
| T09 / 09 | Custom known transaction injection reaches only named account; fixed exact expected values match review. | OPEN; never use default first-account selection. Fixture metadata only, no raw payload evidence. | #88 |
| T10 / 09 | Persona fixtures: earned-wage, young-professional and small-business histories; refresh may return no new data. | OPEN as supplemental coverage; one representative small-business fixture is required for bookkeeping acceptance. Others need a recorded disposition. | #88 |

## Webhook delivery and failure recovery

| ID / source | Test and exact clearance evidence | Current status / next action | Owner issue |
| --- | --- | --- | --- |
| W01 / 13 | Set actual webhook URL during enrollment (or explicit update); first sync activates notifications; signed SYNC_UPDATES_AVAILABLE reaches receiver. | BLOCKED. `CreateLink` sets no webhook; webhook package is not mounted on API. Need separately reviewed narrow hosted ingress/configuration. fire_webhook success alone is insufficient. | #103 / #87 / #88 |
| W02 / 13 | Follow signed delivery → durable receipt/demand → task/worker → correct source publication; initial/history flags never themselves publish readiness. | OPEN hosted; `TestWebhookPostgresReceipt`, `TestConnectionOpsWebhookAtomicReplay` are synthetic. | #87 / #88 |
| W03 / 14 | Duplicate/reordered delivery, webhook during running sync and durable-commit failure are safe; only durable receipt acknowledged, response under ten seconds. | PARTIAL. Receipt/fence/concurrency tests synthetic; hosted transport timing and retry OPEN. | #85 / #87 |
| W04 / 14 | Retryable delivery failures recover, permanent denials remain denied, overload/429 bounded; no acceptance before durable commit. | PARTIAL synthetic handler/admission tests; actual safe non-production retry experiment OPEN. Do not assume every4xx retries. | #87 / #89 |
| W05 / 14 | Missed webhook/outage recovers through scheduled catch-up without duplicate effects or cross-source access. | PARTIAL synthetic catch-up/dispatch; Scheduler ENABLED is configuration evidence only. Hosted missed-event recovery OPEN. | #85 / #89 |
| W06 / 15 | ES256/kid/key rotation/expiry, exact raw-body digest, old/future JWT, malformed/oversize and invalid signature fail closed. | PASS synthetic unit behavior only: `TestPlaidWebhook`, `TestWebhookRejectedInputs`, `TestWebhookKeyLifecycle`, `TestEquivalentECDSADelivery`. Real signed delivery remains W01. | #87 |
| W07 / 15 | Key service outage/timeout/cache budget; wrong environment, unknown/revoked Item, account/tenant isolation; no SQL work for unauthenticated input. | PARTIAL synthetic `TestKeyTransportFailures`, `TestWebhookPersistenceFailure`, SQL scope tests. Hosted identity/scope path OPEN. | #87 |
| R01 / 12 + project | API rate limits, timeouts, unavailable institution, malformed page, bound overflow and worker lease expiry preserve last completed publication. | PARTIAL `TestProviderFailures`, acquisition/dispatch fault suites; exact coverage gaps require case inventory, not suite-name inference. | #85 / #89 |
| R02 / project | Independent fixture oracle compares source identity/quantity/unit/date/state to persisted observations and review; explain every discrepancy. | OPEN hosted; UI row-count equality is not this proof. No raw payload retention. | #88 / #84 |
| R03 / project | Before/after DB assertions: acquisition/replay produces zero journal and journal-posting outbox additions. Lost COMMIT, stale writer and permission revocation cannot duplicate effects. | PASS synthetic (`TestAcquisitionCommittedReplyLoss`, sync/providerflow suites); OPEN for current hosted application DB. UI zero balances/Posted filter insufficient. | #85 / #88 |

## Item lifecycle and Link errors

| ID / source | Test and exact clearance evidence | Current status / next action | Owner issue |
| --- | --- | --- | --- |
| I01 / 05 | reset_login → ITEM_LOGIN_REQUIRED → actionable repair → update-mode success → same Item continuity and sync resumes. | BLOCKED. HTTP adapter's reconnect path lacks qualified Hosted Link completion; VerifySession rejects nonempty expectedItem. Synthetic SQL reconnect is not concrete provider proof. | #104 / #88 |
| I02 / 05 | PENDING_DISCONNECT; consent expiry; renewal; LOGIN_REPAIRED clears stale messaging without unauthorized collection. | OPEN. Current ingress accepts only Transactions notifications; Item lifecycle event support requires implementation/review. | #103 / #104 |
| I03 / 05 | NEW_ACCOUNTS_AVAILABLE → explicit account selection; revoked access cannot be reacquired silently. | OPEN; same Item-event gap. | #103 / #104 / #87 |
| I04 / 05 + project | Disconnect/removal acknowledgement and ambiguous result; disable future acquisition, preserve approved immutable history, no stale-worker publication. | PARTIAL synthetic `TestConnectionRotationAndDisconnect`, `TestConnectionRemovalUncertainty`; diagnostic Item cleanup passed(A), app disconnect/reconnect not exercised. | #104 / #87 |
| I05 / 05 | PENDING_EXPIRATION / UK-EU renewal variant. | N/A for current US profile; reopen with country scope. US consent cases remain I02. | #88 |
| E01 / 08 | MFA device OTP, questions, selections and multiple rounds: success, wrong answer, user cancellation. | OPEN actual Link cases; public Sandbox fixtures only. | #104 / #88 |
| E02 / 08 | reCAPTCHA success/failure and recovery. | OPEN, owner-operated challenge; no unattended solving. | #104 / #88 |
| E03 / 08 | Non-OAuth Link errors: ACCESS_NOT_GRANTED, COUNTRY_NOT_SUPPORTED, INSTITUTION_NOT_RESPONDING, INSTITUTION_NO_LONGER_SUPPORTED, INSTITUTION_REGISTRATION_REQUIRED, INSUFFICIENT_CREDENTIALS, INTERNAL_SERVER_ERROR, INVALID_CREDENTIALS, INVALID_MFA, INVALID_SEND_METHOD. | Each OPEN; record individual outcome and safe retry/exit behavior. | #104 / #88 |
| E04 / 08 | ITEM_LOCKED, ITEM_NOT_SUPPORTED, MFA_NOT_SUPPORTED, NO_ACCOUNTS, UNAUTHORIZED_INSTITUTION, USER_INPUT_TIMEOUT, USER_SETUP_REQUIRED. | Each OPEN. Payment and deposit-allocation errors excluded below. | #104 / #88 |
| E05 / 08,17 | OAuth error selector: INSUFFICIENT_CREDENTIALS, ACCESS_NOT_GRANTED; missing institution/account and cancel return safe UI state. | OPEN. NO_AUTH_ACCOUNTS is N/A Transactions-only. | #104 / #88 |

## Production request and exclusions

| ID | Requirement / disposition | Current status / owner |
| --- | --- | --- |
| G01 | Review account-specific Launch Center; record product/country/platform checklist, app/company/use-case profile and OAuth prerequisites without copying private fields. Compare each item to this matrix. | OPEN, #88; public checklist is obsolete (P88-01). |
| G02 | Review Hosted Link vs official web SDK recommendation for owned frontend. Retain current boundary until explicit decision, then qualify chosen completion/recovery flow. | OPEN, #187/#104; P88-04 conflicts with assuming Hosted Link is the final UX. |
| G03 | Production environment/credential/SQL isolation, required products only, consent/privacy, retention and budget; no Sandbox fixture endpoints in live path. | BLOCKED under #90; current runtime intentionally Sandbox-only. |
| G04 | Secure token/identifier handling, redacted support evidence and Item removal/retention review. | PARTIAL synthetic vault/redaction; hosted security and owner policy review remain #164/#62/#90. Raw identifiers seen in Sandbox UI require Production exposure review (P88-01). |
| G05 | Institution-specific history, OAuth and statement reconciliation with one explicitly authorized read-only real account. | BLOCKED #90; impossible to establish from Sandbox alone (P88-02). |
| D01 | Auth, micro-deposits, Identity, Liabilities API, Investments, Assets, Statements, Income, Transfer/Payment Initiation, verification/Monitor and other unrequested products. | N/A to this integration. Also excludes their payment/deposit error fixtures. Reopen if products enabled; loan transaction support does not clear Liabilities tests. |
| D02 | Recurring Transactions, provider Refresh in the customer UX, Multi-Item Link, Link Recovery beta, returning-user Profiles and Link Delivery SMS/email. | N/A as shipped product features, not permanently waived. Refresh remains required as the Sandbox stimulus in T04. SDK onExit/onEvent applies if SDK chosen; Hosted session outcomes remain L02. |

## Execution order and issue boundaries

1. #88: finish G01 applicability and exact remaining-case inventory. Do not close
   #85/#87 from historic suite names; compare their original criteria to cases.
2. #103/#87: W01–W07 narrow ingress and actual signed chain. #104/#187:
   L02–L05, O01–O05, I01–I04 and E01–E05. These are separate reservations in
   #65 before editing shared connection/webhook/API/SQL paths.
3. #85/#88: T04–T09 and R01–R03 with an independent oracle and bounded runtime
   evidence capability. Synthetic-only substitutes require explicit reviewed
   rationale for cases the provider cannot reliably induce.
   Current UI cannot establish R02/R03: design a reviewed redacted comparator/
   count capability before execution; do not obtain general live SQL or raw pages.
4. Close #88 only when required applicable rows have evidence, every supplemental
   row has a reviewed disposition, and Launch Center adds no unmapped cases.
   Failed cases need fixes and reruns on named source, never timeout inflation.
   #89's known 1,000-command failure remains separate and unresolved.
5. #90 owns real-account pilot/statement reconciliation and explicit privacy,
   retention and interval approval. Sandbox completion does not authorize it.

Original #88 obligations remain: actual Link/exchange, initial/incremental and
dynamic lifecycle, signed delivery, provider-to-review/cursor/replay/no-journal
proof, owner-injected credentials, redacted evidence and institution/history/
balance limits. This matrix expands clarity without lowering those requirements.
