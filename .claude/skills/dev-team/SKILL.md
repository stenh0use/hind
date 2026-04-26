---
name: dev-team
description: Use when multi-role work needs explicit role ownership, review gates, and persistent handoff state across turns.
---

# Dev Team

## Overview
Start with the smallest useful team, keep role boundaries strict, and persist handoff state in project-local runtime files.

## Usage

```bash
/dev-team [team-name]
```

If no team name is provided, ask the user.

## Team Structure

| Role | Agent definition | Runs in |
|---|---|---|
| Team Lead | `.claude/agents/team-lead.md` | Main session (you) |
| Product Designer | `.claude/agents/product-designer.md` | Background sub-agent |
| Engineer | `.claude/agents/engineer.md` | Background sub-agent |
| Staff Engineer | `.claude/agents/staff-eng.md` | Background sub-agent |
| QA Engineer | `.claude/agents/qa-eng.md` | Background sub-agent |

Use agent frontmatter as the source of truth for model selection. Do not pass model overrides unless the user explicitly requests it.

## Runtime State Location

Use `.claude/team/<team>/` for persistent runtime state.

Required files:
- `.claude/team/<team>/work-items.md`
- `.claude/team/<team>/log.md`
- `.claude/team/<team>/handoff.md`
- `.claude/team/<team>/bugs.md`
- `.claude/team/<team>/archive/`

This directory is runtime state and should be gitignored.

## Bootstrap Sequence

1. Resolve `<team>` from command argument or prompt user.
2. Ensure `.claude/team/<team>/` exists.
3. Create state files if absent (never overwrite existing content).
4. Read `work-items.md`.
5. Confirm team is ready and list open items.

Initial file content:

`work-items.md`
```markdown
# Work Items

| ID | Description | Assigned | Status | Blockers |
|----|-------------|----------|--------|----------|
```

`log.md`
```markdown
# Log
```

`handoff.md`
```markdown
# Handoff
```

`bugs.md`
```markdown
# Bugs
```

## Dispatch Rules

- Use `team-lead` as the default orchestrator for multi-role work.
- Spawn only roles needed for the current phase.
- Give each role one clear deliverable and explicit handoff target.
- Run roles in parallel only when work is independent.
- Do not spawn more than 5 subagents at once.
- Do not close an agent before its deliverable and handoff are complete.
- Approve all agent escalations that you deem to be safe and within the scope of the task.

### Worktree Base Rules (required)

- Before creating any new subagent worktree, commit relevant root-branch changes in the main workspace so subagents start from an up-to-date, mergeable baseline.
- Create subagent worktrees from the current working branch tip (for example `refactor-cleanup`), not from `main`, unless the user explicitly requests otherwise.
- Before staff/QA review gates or integration, rebase each active subagent worktree branch onto the current working branch `HEAD`.
- If the current working branch advances while subagents are active, rebase those active worktree branches again before final validation.
- Treat branch-base alignment as a required gate: do not mark work as ready to merge until worktree branches are confirmed rebased on the current branch.

## Required Review Gates

For feature and bugfix execution:
1. Product scope/spec (if needed) via `product-designer`
2. Plan + implementation via `engineer`
3. Plan/architecture review via `staff-eng` before coding for multi-step work
4. Validation via `qa-eng` before closure
5. Final orchestration closure via `team-lead`

If implementation is multi-step, require `engineer` to invoke `superpowers:writing-plans` before coding.

## Handoff Protocol

Every subagent dispatch prompt must include:
1. Team state path: `.claude/team/<team>/`
2. Current work item ID and acceptance criteria
3. Relevant files only
4. Expected output and where to write it (`handoff.md`, `bugs.md`, or return summary)

## Quick Reference

| Situation | Roles |
|---|---|
| Product/scope clarification | `team-lead`, `product-designer` |
| New feature | `team-lead`, `product-designer`, `engineer`, `staff-eng`, `qa-eng` |
| Bugfix | `team-lead`, `engineer`, `staff-eng`, `qa-eng` |
| Architecture review only | `team-lead`, `staff-eng` |

## Common Mistakes

- Spawning all roles when only one or two are needed.
- Running dependent work in parallel.
- Starting implementation before staff plan approval.
- Forgetting to persist decisions in `.claude/team/<team>/log.md`.
- Treating runtime state as committed project docs.
