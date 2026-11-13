# Portable rule packs —498

## Decision and boundary

Owner2026-11-12 requests bulk YAML rules and compact Infisical-inspired management,
superseding the manual hosted rule setup. YAML is a local authoring format for
the existing complete rule edition. It is not a new evaluator, public execution
route, provider fact, or accounting treatment. Current Go/SQL rule precedence,
manual protection, automatic category policy and saved command recovery remain.

The browser reads a selected file locally or accepts pasted text, validates it,
shows additions/updates/preserved rules, then explicitly adds it to the draft.
The existing server preview → saved review → confirmation → acknowledgement is
the only durable write path. Closing an importer or downloading an export never
saves rules. No upload to another service, URL fetching, browser persistence,
provider call, migration or new API is introduced.

## Format

```yaml
version: 1
rules:
  - name: Fictional cafe
    account: {name: Fictional checking}
    priority: 0
    conditions:
      - field: merchant
        operator: exact
        value: Fictional Cafe
    topic: {name: Food and drink}
    tags: []
```

`account`, `priority`, `status`, `tags` and `id` are optional; defaults are the
whole private workspace,0,active,empty tags and a newly generated UUID. Exported
packs include stable IDs. `topic` is `{name: Exact name}`, `{id: UUID}`, or the
literal `uncategorized`. Accounts/tags use the same explicit reference objects;
names must resolve exactly once in the current authorized catalog. This avoids
silently picking one of several similarly named accounts. Renames and repeated
imports should use exported IDs. No-ID entries matching an existing name are
rejected with guidance to export the ID, rather than silently replacing a rule.

Conditions use the existing field/operator/string-value profile. Match text is
case-sensitive. Regex is transported as bounded text and reviewed by the single
server evaluator; the browser does not compile or run it. Account and dated
currency condition values retain their existing UUID form. Numeric-looking or
date values must be quoted when YAML would otherwise resolve a non-string value.

Merge updates by rule ID and retain all omitted rules. Retirement is explicit.
Retired history cannot be changed or reactivated; exporting its original ID
references still works when a referenced choice is no longer available. Current
unsaved visual edits participate in the merge, rather than being discarded.
Validation summaries are bound to the draft they validated; edits/locks prevent
applying a stale summary. Successful import does not bypass server stale-snapshot
checks or recovery reservations.

## Bounds and trust

One YAML1.2 document, at most256KiB UTF-8, depth12 and25,000 AST nodes. Reject
duplicate keys, non-string/prototype keys, unknown fields, explicit tags,
anchors/aliases/merge keys, multiple documents and non-profile object types.
Existing complete-edition bounds also apply:256KiB JSON,1,000 retained identities,
100active rules,1–8conditions,128UTF-8 bytes per condition,20output tags and
current topic/tag/source authority. The file size is checked before reading;
late file reads are ignored after unmount, draft replacement or a recovery lock.

An explicit maintained open-source parser is used, rather than writing a YAML
subset parser or executing arbitrary third-party rule formats. `yaml@2.9.1` is
pinned under ISC. Open-source rule corpora are not included: compatibility,
license and financial-purpose assumptions need review before importing a corpus.
The example is application-authored fictional data, not a production rule pack.

## Design evidence and alternatives

Selected source sections and access limits are in
[rule-pack source review](../research/rule-pack-sources.md). The owner specifically
chooses Infisical's compact management direction. Our implementation uses a
searchable semantic table and one selected editor, quiet separators, short
controls and a narrow-screen stacked panel. No Infisical code/assets are copied.
Compare maintaining stacked forms (rejected owner experience) and direct YAML
execution (would bypass current authority and duplicate the matcher). A local
format adapter reuses the implemented reviewed-write boundary.

Standards: this local authoring representation is not an ISO20022 financial
exchange or ISO21378 audit export; it changes no money, currency or source facts.
Existing dated ISO4217 projections are unaffected. No ISO or accounting
conformance is claimed for YAML or the UI.

Acceptance: codec fixed adversarial/roundtrip/history tests, importer draft/lock
tests, compact selection/search tests, real-browser YAML-to-preview plus narrow
viewport tests, existing full web gates. These do not establish hosted owner
acceptance or third-party corpus classification accuracy. #488 still owns actual
automatic-category acceptance; #334 owns source-backed automatic accounting.
