# Evidence standard for design decisions

## Decision

Owner direction: material accounting, database and architecture decisions must
identify their sources before implementation. Use this record inside the relevant
ADR or contract; keep one authoritative decision rather than a separate status copy.

## Required record

| Field | Required content |
| --- | --- |
| Question and scope | Concrete operation/problem, workload and exclusions |
| Status and owner | Proposed, accepted or superseded; reviewer and date |
| Sources | Author/publisher, title, URL, version/date, exact section/pages, access date |
| Review depth | Full relevant section, abstract only, or preview only |
| Evidence | What the source actually establishes, in a short paraphrase |
| Interpretation | Our inference and why it applies to this project |
| Alternatives | At least one plausible competing approach and its tradeoffs |
| Uncertainty | Assumptions, contradictions, missing policy and disconfirming evidence |
| Acceptance | Named test, worked example, benchmark or explicit reviewed evidence |
| Follow-up | Issue, decision owner and condition for revisiting |

## Source discipline

Prefer original publications, official versioned manuals, legitimate textbooks
and first-party operational accounts. Distinguish implementation documentation
from normative accounting rules, academic results and vendor case studies.
Read the cited section; a search snippet or linked title is discovery, not review.
An arXiv preprint is not automatically peer-reviewed. Do not attribute claims to
unseen book chapters or mirror unauthorized copies. Retain links and original
notes; preserve source/version scope rather than calling every finding current.

A citation does not approve a design. Example: PostgreSQL's deferred-trigger
documentation establishes a mechanism; our complete-journal invariant and
concurrency proof are separate work. Revisit mutable service/tool documentation
when implementing or upgrading. Material disagreement belongs in the decision,
not hidden in a bibliography.

Use the [first research review](database-foundations.md) and
[source register](database-sources.md) as the initial worked example.
