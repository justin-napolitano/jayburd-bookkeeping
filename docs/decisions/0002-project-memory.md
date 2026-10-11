# ADR 0002: repository context and GitHub execution

## Decision

Adopt Brainblast's linked decisions, bounded work packets, explicit dependencies,
dry-run mutations and evidence-based handoffs. Keep this project's context beside
its code. GitHub owns issue state, discussion and native dependency relationships.
Repository Markdown owns product contracts, decisions, research and runbooks.

The initial backlog is a one-time publication seed. After publication, its
acceptance text is historical input; update active scope in the Issue and link
revised contracts. Never mirror ongoing status into this file. The issue registry
stores stable identity links only. Generated drafts/packets are not authority.

## Why

[Local inspection](../research/brainblast.md) found useful tested query/sync
primitives, but also a fail-open queue rule and project-specific coupling.
GitHub now supports native dependencies and sub-issues. A single-project repo
does not need bidirectional Markdown/Issue synchronization or an agent scheduler.

This intentionally adapts the task-orchestrator workflow: the external vault is
a read-only reference, not the canonical task store for Jayburd. No private vault
notes, organization records, or copied implementation are incorporated.

## Rules

- Unknown, missing, canceled or inaccessible predecessors block scheduling.
- A closed issue is complete only when its closure reason is completed and its
  acceptance/merge evidence is present. Labels and checkboxes are not proof.
- Native dependency relationships own execution order after publication.
- One issue per independently reviewable outcome; one active slice per person.
- An optional GitHub Project is a view, not another database of task truth.
- No automated deployment or agent execution from issue text or labels.

## Sources

[GitHub dependencies](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-issue-dependencies)
and [Project automation](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-built-in-automations),
checked 2026-10-11. Board auto-close is not an accounting acceptance gate.
