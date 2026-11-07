# Explicit PFC attribution sidecar —413

## Status and compatibility decision

Owner design approval is recorded in GitHub238/242. This bounded Go slice prepares
safe current-response attribution;239/304 private source/taxonomy SQL and integrated
413 acceptance remain open. It adds no runtime caller, migration, HTTP request
option, historical reader, provider call or publication activation.

The inspected path is `internal/acquisition/plaid_http.go` (account-filtered sync
without a PFC option) → `plaid.DecodePage` → `pageWire` JSON → closed transaction
keys in merged25/29 SQL. The current Transaction type discards PFC from normalized
JSON and retains original bytes separately. Adding even a nullable JSON field to
Transaction would break the closed SQL profile and affect provider-field equality.

Decision: leave DecodePage/Page/Transaction and their default JSON unchanged.
Add an explicit enhanced decoder returning the original financial Page plus a
separate `jayburd-plaid-pfc-attribution/1` sidecar. Bind it to the current page's
original SHA256 and each exact account+transaction identity/change kind, never an
array-position join. A future reviewed additive SQL wire must explicitly consume
this sidecar and define category-only revision behavior before activation.

## Projection and failure rules

Only nested PFC primary/detailed/version/confidence fields are interpreted. No
icon/website/logo/raw payload reaches the sidecar. Primary/detailed code strings
are bounded128bytes uppercase ASCII identifiers; version16bytes; confidence32bytes.
Malformed types/oversize/unknown nested fields reject the enhanced decode atomically
with fixed `invalid_category_metadata`; neither financial Page nor sidecar escapes.
The legacy decoder keeps its existing interpretation of inert PFC metadata.

Missing/null PFC yields distinct fixed `not_provided`/`null_category` reasons;
missing/null codes abstain as `incomplete_category`. Versionless PFC stays
`version_missing`; unsupported versions stay `unsupported_version`. No version
is inferred from request, API header, account enablement or deployment time.
The confidence enum is vendor attribution, never a calibrated Jayburd probability.
Missing/null confidence, UNKNOWN and new values produce separate fixed abstention
reasons. Well-formed unknown codes survive as bounded attributed data; only an
explicit reviewed versioned map can recognize a pair.

The pure map helper requires an explicit immutable map edition and at most1000
supported version/code-pair→topic entries. Duplicate entries or malformed catalog keys fail
before matching. There is no seeded full dictionary or universal topic policy.
Known metadata with a known explicit pair produces an inert suggestion with map
edition and attribution; unknown pairs abstain. A helper result is neither a grant
nor a label/write/post command.304 must supply current source/taxonomy authority
and revision pins; this package cannot select sources, access private labels,
apply overrides, create drafts or post journals.

Removed rows carry no invented category. Pending data may carry attribution but
still has the legacy financial readiness=false. A category-only correction changes
the sidecar while normalized financial Transaction JSON stays identical. Original
raw page/row bytes and SHA256 naturally change; those provenance hashes are never
rewritten to manufacture equality. Future SQL owns separate metadata/financial
revision semantics and unchanged posted anchors, not this decoder.

## Sources, interpretation and alternatives

Official public sections were reviewed2026-11-06 in
[the scoped source record](../research/pfc-normalization-sources.md): Transactions
sync options, PFC nullable object/primary/detailed/confidence/version, and the PFC
migration guide's opt-in/default/backfill sections. Provider schema documents
metadata, not private authority or accounting recognition. Bounded types, exact
pair mapping, abstention and the sidecar are Jayburd engineering choices.

Alternatives rejected: silently extend Transaction JSON (breaks closed SQL), infer
historical version from deployment (unsupported attribution), inspect retained raw
history (new unapproved read/backfill), or refetch categories (provider/consent and
revision effects). No third-party code or dependency is added.

This proprietary current-response JSON is not an ISO20022 financial message.
Original ISO4217 dated references, exact decimals, original dates and direction
remain unchanged in the existing decoder. No new FX/recognition/export practice
or ISO21378 mapping is introduced. Existing standards/source limits apply; no
conformance claim follows from the sidecar profile name.

STRIDE: metadata tampering→closed bounded types and explicit mapping; disclosure→
no raw/icon/secret fields or logging, same account filter and atomic failure;
elevation→no source-selection/write capability; DoS→existing2MiB/500-change page
caps plus finite metadata/map limits; repudiation→original hashes/identities and
future authorized immutable store, not sensitive logs. Human source/security
review and actual SQL/API authorization proofs precede integrated activation.

## Named acceptance

`TestPFCNormalizedCompatibility`, `TestPFCSourceMoneyAndScope`,
`TestPFCVersionMapping`, `TestPFCOmittedUnknown`,
`TestPFCConfidenceNotCalibration`, `TestPFCEnhancedAtomicity`,
`TestPFCRevisionCorrection`, `TestPFCMapClosedImmutable` and `FuzzPFCSidecar`
prove legacy JSON compatibility, exact money/currency/date/filter preservation,
explicit version mapping, omitted/null/version/confidence abstention, malformed
enhanced atomicity, category-only correction and immutable caller-owned map
inputs. They cannot prove SQL grants, taxonomy revision conflicts, durable
label replay or no-journal database behavior. Those remain413 integration gates.
