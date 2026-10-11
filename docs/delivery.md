# Delivery workflow

## Ownership

GitHub Issue = executable task and discussion. PR = implementation and review.
Markdown = decisions/contracts/evidence. GitHub Actions run = build evidence.
Issue registry = stable links. No local task-status mirror.

## Start a slice

1. Use `make backlog` before publication, or GitHub Issues afterwards.
2. Inspect native blockers and their completion evidence. Missing or canceled
   blockers remain unresolved; do not infer readiness from a closed checkbox.
3. Run `make packet ID=JB-001` and read the linked current docs. The output is
   marked as a seed packet, never proof of live readiness.
4. Assign the issue and create `issue/<number>-short-description` in an isolated
   worktree. Add a brief claim comment; GitHub assignments are not distributed locks.
5. Define named tests and keep the issue bounded. Update its scope before coding
   if understanding changes.

## Review and completion

Each PR links its issue, relevant decision, stage runbook, exact verification and
migration/rollback implications. Run `make check test build` plus issue-specific
tests. A Go tooling build does not prove a ledger, migration or deployment.
Before committing, stage the branch's updated runbook and run
`bash scripts/check-stage-runbook.sh --staged`.
Review the `Seed / validate` workflow result initially (job context `validate`).
GitHub returned HTTP 403 for private branch protection on 2026-10-11: this
account needs Pro to enforce it. Until upgraded, the owner must review passing
checks before merge; this is a manual convention, not an enforced gate. Replace/extend it with product
unit, PostgreSQL integration, UI, migration and infrastructure checks as those
components appear. Validate branch protection capability before relying on it.

Complete only when acceptance evidence and review/merge are recorded. Closure
as not planned never satisfies a dependency. End sessions with outcome, branch,
commit/PR, checks, blockers and one next action in the issue and runbook.

## Publish the initial issues

`make issue-drafts` validates the graph and writes ignored `.local/issues/*.md`.
Review those bodies. Publication needs an explicit destination and authorization.
Search all open and closed issues for each `jayburd-seed:JB-...` marker before
creating; adopt the matching identity, never overwrite its body or reopen it.
Use `gh issue create --body-file` and native `--blocked-by` relationships.
Record only stable ID, number and URL in `planning/issues.json`.

The seed is an initial proposal, not an issue-sync engine. New work is authored
directly in GitHub using the issue form. Project boards are optional views.
