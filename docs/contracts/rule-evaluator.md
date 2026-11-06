# Bounded rule evaluator —414

Status: reviewable unintegrated implementation of the deterministic algorithm in
[private categorization](private-categorization.md). This package does not satisfy
238/239/242/304 or implement the whole414 product. No API, SQL, provider, service
startup or browser imports this package. Do not activate it until current private
authority, taxonomy and reviewed preview/apply integration are proved.

## Inputs and meaning

`Compile` accepts a current active taxonomy catalog and at most100 approved rule
versions. The caller must obtain these through the future restricted store; a
caller-created catalog is not trusted authority. Its principal/workspace keys
are finite opaque internal identities resolved from verified issuer+subject, not
emails or request-supplied grants. Cross-scope rules and facts reject before any
result. The catalog/rules are defensively copied and immutable during evaluation.

`Evaluate` accepts at most100 distinct explicit source observations. Source
revision, account, original currency-version, direction and canonical source date
are mandatory pins. Merchant is nullable and description/merchant are limited to
500 Unicode codepoints/2000 UTF8 bytes. Invalid input rejects the whole batch,
never a partial list or false no-match. The algorithm accepts no monetary amount,
raw payload, URL, token, book assignment or journal operation.

Outputs contain one active topic or explicit Uncategorized, plus at most20
distinct active tags. Topic and tag labels are resolved elsewhere. Rules cannot
create a topic, account, source, draft or journal. The catalog permits500 active
topic and500 tag IDs; hierarchy/cycle/version/edit/retirement enforcement belongs
to the future taxonomy store. No taxonomy or source permission is inferred here.

## Exact behavior

1. Current source/taxonomy-pinned manual override wins. A stale source or taxonomy
   revision returns `stale_override` with no fallback. Historical override audit
   remains the store's responsibility, including retired topic labels.
2. Matching account-specific rules precede workspace rules, regardless of the
   workspace rule's priority. Within scope, larger explicit priority wins; its
   closed integer range is−1000000 through1000000.
3. Equal priority ranks merchant exact4, merchant literal contains3, description
   literal contains/exact2, regex1, other restrictive conditions0. Extra
   conjunctive conditions do not increase rank. Description exact shares the
   description literal tier rather than inventing new hidden precedence.
4. Every equal-rank winner is returned in stable rule-ID order. Topic/Uncategorized
   or tag-set disagreement yields `conflict` with no output. Identical tag sets
   in different orders agree. IDs order explanations and never resolve conflict.
5. No matching rule returns `unclassified`. The future reviewed PFC mapper may
   supply fallback only for this status, never `conflict`/`stale_override`. This
   implementation cannot assert provider taxonomy compatibility or attribution.

All conditions are conjunctive. Merchant/description support exact, literal
contains and Go regexp; account/currency-version/direction support equality;
source dates support inclusive on-or-after/on-or-before. Matching is case-sensitive;
regex flags must be explicit. An absent merchant never matches. Unknown fields,
operators, invalid UTF8, duplicate identities and inactive outputs reject.

At most8 conditions/rule,128 UTF8 bytes/pattern. Compile once per ruleset version.
Parse using Go regexp/syntax, reject a conservative expanded-work estimate over1024
**before** Simplify, then reject an actual compiled program over1024 instructions.
This preflight may conservatively reject a pattern below the final instruction cap;
it prevents nested counted-repeat allocation rather than compiling first and
checking memory use afterward. No backreferences/lookaround/backtracking engine,
script, network call or third-party matcher is added. Runtime bounds arise from
the finite rule/condition/input/program caps and Go's documented linear matching;
wall-clock benchmark results are not a service latency guarantee.

## Authority, recovery and standards

The service/store must authorize before constructing inputs and again under SQL
locks before applying any result. It must pin exact source/taxonomy/ruleset/map
revisions, preserve immutable label history and use durable command/receipt and
explicit acknowledgement recovery. A computed Result is never a capability or
write command. Scope equality in this package cannot prove a grant, revocation
race, source selection, atomic bulk application or replay. Those tests stay open.

This internal inert text classifier introduces no ISO20022 financial exchange.
It preserves ISO4217 dated currency identity only as an equality condition and
performs no monetary arithmetic, recognition or FX. No audit/export wire or
ISO21378 conformance is claimed. Existing [standards](standards.md) and private
categorization mapping remain authoritative; provider integration requires413's
additive normalized-profile/SQL compatibility before any activation.

STRIDE scope: cross-owner disclosure/elevation→reject mixed keys and require the
future authoritative source selection; tampering→active taxonomy validation,
defensive copies and exact provenance; DoS→pre-expansion regex and batch caps;
repudiation→future immutable store receipts, not logging source text. The package
does no I/O or logging and adds no dependency. Sensitive integration still requires
human source/security review; a pure-engine unit test is not an authorization test.

## Sources and alternatives

Go standard library [regexp](https://pkg.go.dev/regexp), overview/Compile/MatchString,
and [regexp/syntax](https://pkg.go.dev/regexp/syntax), Parse/Simplify/Compile/Prog,
revisited2026-11-06; public API sections reviewed, not the full runtime source.
The installed1.26.5 implementation was used for local tests; tracked1.26.10 CI is
still required. No external code copied. [CR08](../research/categorization-rules-sources.md)
and433's [independent examples](../reviews/classification-433.md) are the existing
source and product decision records.

Alternatives: compile a regex per transaction (repeated unnecessary work); check
program size after full expansion only (allocation exposure); last-rule-wins
(silent ambiguity); browser/SQL duplicate matcher (divergent decisions). The
immutable shared Go engine is prepared for future authorized API/jobs reuse.

## Named evidence

`TestRulePrecedence`, `TestAccountBeforeWorkspaceRule`, `TestRuleSpecificity`,
`TestRuleConflictAbstains`, `TestRuleAgreement`, `TestOwnerRuleScope`,
`TestOverridePreserved`, `TestRuleConditions`, `TestRegexWorkBound`,
`TestRuleInputClosed`, `TestCompiledRuleImmutability`, `TestConcurrentEvaluation`
and `FuzzBoundedRegex` execute fictional cases. They prove algorithmic behavior,
closed inputs, immutable compilation and bounded matching, not database authority,
preview/apply equality, durable replay, browser recovery, provider mapping or
no-journal database invariants. Those remain integration acceptance in414.
