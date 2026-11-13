# Rule-pack source review —498

Reviewed2026-11-12; selected public documentation only. No private Infisical
workspace, secret, provider payload or rule corpus accessed.

| Source | Sections reviewed | Evidence and limit |
| --- | --- | --- |
| [eemeli YAML2 documentation](https://eemeli.org/yaml/) | API overview; parseDocument; parser uniqueKeys/strict/version options; schema/merge/custom tags; AST collections/isAlias; toJS maxAliasCount; stringify | Supports parsing and inspecting YAML1.2 syntax without executing it, duplicate detection and disabling alias materialization. Our byte/node/depth limits and closed rule schema are application policy, not a library security guarantee. Documentation states ISC license, modern browser support and no external dependencies. npm registry resolved2.9.1/ISC for this build. |
| [Infisical secrets management](https://infisical.com/docs/documentation/platform/secrets-mgmt/overview) | Overview, organization/versioning/access-control navigation | Establishes environment/path-oriented management and versioned changes. Compact table + side editor is our interpretation of the owner's requested aesthetic, not a claim that unseen screenshots or Infisical code were reviewed/copied. |
| [Existing rules review](categorization-rules-sources.md) | CR01–04/08 recorded review and limitations reused | Existing exact/contains/regex transport and conflict semantics remain; no Actual/hledger corpus/engine compatibility or license reuse is asserted. |

Alternatives and acceptance are in [the contract](../contracts/rule-packs.md).
`yaml` is the only new dependency. Its locked registry integrity and ISC license
are retained in package-lock; no source was vendored. A parser license does not
grant rights to unrelated merchant dictionaries or exported third-party packs.
