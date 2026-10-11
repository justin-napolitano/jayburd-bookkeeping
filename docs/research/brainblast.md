# Brainblast tooling assessment

## Scope and evidence

Read-only inspection on 2026-10-11 of the local Brainblast checkout at base
commit `3440feb`. The worktree is dirty; inspected behavior includes local
uncommitted changes. No external-vault content or implementation was copied.
Only focused synthetic tests were run; this is not a whole-vault audit or a
live GitHub synchronization proof.

| Check | Observed result |
| --- | --- |
| `ruby scripts/brainblast/test_query_layer.rb` | PASS: 4 tests, 12 assertions |
| `ruby .codex/skills/sync-project-tasks/scripts/test_sync_tasks.rb` | PASS: 6 tests, 27 assertions |
| Temporary synthetic task referencing an absent predecessor, run through `work_queue.rb --format json` | REPRODUCED: `dependencies_ready=true`, expected false |
| Installed GitHub CLI 2.96.0 | Native dependency and sub-issue flags available |

## Keep

Bounded context packets; one authoritative note per decision; links between
contracts/tasks/evidence; explicit predecessor graphs; dry-run mutation plans;
stable identities; focused validation; compact next-action handoffs.

## Adapt or defer

The queue scans root task notes, treats unknown dependencies as satisfied and
accepts canceled tasks as predecessors completed. Its priority map also omits
critical, and epic ordering has a project-specific path. Do not use it unchanged
as an execution gate. Focused sync tests cover formatting and identity helpers;
they do not demonstrate full remote conflict/race correctness.

GitHub's native issue dependencies/sub-issues reduce the need for custom task
graph synchronization. Keep project context in this repo; omit a separate vault
status system, custom leases for AI agents, semantic/vector search and complex
catalog governance until demonstrated scale requires them.

## Sources

Local paths inspected: `scripts/brainblast/work_queue.rb`,
`scripts/brainblast/query_layer.rb`, `test_query_layer.rb`, and the
`sync-project-tasks` skill/scripts. Current platform capabilities:
[dependencies](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-issue-dependencies),
[sub-issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues),
[Project automation](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-built-in-automations).
