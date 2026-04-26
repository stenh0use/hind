# Reboot Handoff — Team Lead

## Resume target
Continue backlog execution for team `hind` from current state with worktree-base alignment rules in effect.

## Canonical context (do not duplicate)
- Backlog + priorities: @.claude/team/backlog.md
- Team runtime state: @.claude/team/hind/work-items.md
- Detailed handoffs/findings: @.claude/team/hind/handoff.md
- Skill workflow rules (updated): @.claude/skills/dev-team/SKILL.md
- Project workflow constraints: @AGENTS.md

## Current branch/commit anchors
- Coordinator branch: `refactor-cleanup`
- Latest coordinator commits:
  - `df28e90` chore: enforce worktree base alignment in dev-team skill
  - `aba91c9` fix: remove unsupported start version contract

## Worktree branch status
- BL-001 worktree branch: `worktree-agent-adb08eca2723fce95`
  - Latest commit: `db0524a` (panic guard + tests)
  - Rebased onto latest `refactor-cleanup`
- BL-002 worktree branch: `worktree-agent-a0d98ce5a4a60f2f4`
  - No commit beyond baseline; has unstaged edits in:
    - `pkg/cluster/cluster.go`
    - `pkg/cluster/manager.go`
    - `pkg/file/file.go`
  - Rebased onto latest `refactor-cleanup`
  - Currently not merge-ready (build/test failure)

## Immediate next actions after reboot
1. Re-open team state from @.claude/team/hind/work-items.md and @.claude/team/hind/handoff.md.
2. Keep using @.claude/skills/dev-team/SKILL.md worktree rules:
   - commit coordinator changes before spawning new worktrees,
   - base worktrees on current branch,
   - rebase worktrees before review/integration.
3. Resume BL-002 in its existing worktree first (do not start parallel follow-ons until BL-002 is build-green).
4. After BL-002 compiles/tests, run staff/QA gates for BL-001/BL-002/BL-005 batch per @.claude/skills/dev-team/SKILL.md.

## BL-002 blocker snapshot
Last observed failure while testing BL-002 worktree:
- `pkg/cluster/manager.go:39:12: undefined: ValidateClusterName`
- `pkg/cluster/cluster.go`: unused imports (`path/filepath`, `strings`)
