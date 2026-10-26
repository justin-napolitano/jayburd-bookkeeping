# Independent Production journey — #315

Engineering test decision2026-10-26, source3a2365c. This issue adds synthetic
acceptance only: no product routes, migrations, cloud resources, provider access,
secrets or real records. Existing Sandbox expectations remain independent.

Exercise real Next bridge, Go service, Production HTTP adapters, encrypted bank
and Link stores, restricted SQL sessions, acquisition and reviewed posting. Inject
fictional identity only in the integration executable/component, fake KMS and a
transport that refuses anything except the exact Production request destination.
The Docker network has no external route and no caller DSN or Production reset.
This does not qualify actual identity/MFA, institution support or hosted IAM/KMS.
The controlled pilot prepares owner/worker permissions before enrollment. Production's explicit service constructor deliberately uses operator-configured
books; this test does not claim unprepared-book self-service (#237).

Keep existing literal 0.10/0.20 USD fixture inputs and independently written
expected 0.1/0.2 source amounts, 0.10 debit/credit report change after one reviewed
posting, and zero acquisition journals. Initial empty and multi-page history need
separate explicit runs; exact completed-claim replay must make no provider call.
One selected checking binding is the bounded Production journey, while current
Sandbox three-account binding tests remain unchanged. Simulated provider origin
must be production:false in SQL; all input data is fictional regardless of origin.

Named acceptance: TestProductionProfileJourney (browser complete/reload/lost-result,
selected binding, initial empty/history/pagination/replay, pause/resume, explicit
post/replay/report); existing TestProductionOwnedSessionSQL (owned signed proof,
same addressed encrypted credential); TestProductionProfileCrossEnvironmentDeny,
TestProductionScopePrivileges, TestProductionCredentialEnvironmentBinding,
TestProductionWebhookCurrentScope and TestProductionPostingAuthorityAndReplay.
The matrix in the stage runbook distinguishes one composed journey from component
negatives and actual outstanding hosted gates. Disconnect/revoke must deny fresh
acquisition after the reviewed final posting, preserving existing journals.

Sources: reviewed PP01–09 in production-pilot-sources.md, current literal browser
fixtures and exact SQL/credential/posting contracts from311–314. No new vendor
mechanism/version or financial mapping. Preserve API2020-09-14, ISO4217 original
units, STD-01 applicability and existing ISO21378 review limits; no ISO20022 or
normative conformance claim. Test operational controls have no forced ISO message.

Rejected adapter-only mocks: they cannot prove cross-layer composition. Rejected
calling Production for tests: real authentication/data belong to separately scoped
pilot. Reuse the test fixture infrastructure with explicit closed test mode rather
than introducing application-route auth or another runtime/service/database.
Acceptance does not waive303/320/276, #337 decisions or #316 release review.
