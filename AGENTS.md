# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

---

## Branch Strategy

- `main` — stable, release-ready code. Never commit directly.
- `dev` — integration branch. All feature work merges here first.
- `feature/<name>` — short-lived branches in worktrees, one per task.

If `dev` does not exist, the orchestrator creates it from `main` before dispatching work.

---

## Orchestrator Role (Opus — Main Session)

You are the **orchestrator**. You do NOT write feature code directly. Your job:

1. **Triage work** — Use `bd ready` to find tasks. Group related items into atomic work units.
2. **Dispatch to subagents** — Launch a **sonnet** subagent (via the Task tool) for each work unit. Each subagent works in its own **git worktree**.
3. **Verify and integrate** — After a subagent completes, verify the result and rebase into `dev`.
4. **Sync and push** — Keep `dev` pushed to remote.

### Dispatching a Subagent

For each work unit, the orchestrator MUST:

```
1. Create the worktree:
   git worktree add ../ai-tui-<feature-name> dev -b feature/<feature-name>

2. Launch a sonnet subagent (Task tool, model=sonnet) with:
   - The worktree path as working directory context
   - The full task description (feature + test requirements)
   - Explicit instruction to follow the Subagent Workflow below

3. After subagent returns success:
   - Verify the worktree has clean status
   - Rebase onto latest dev:
     git -C ../ai-tui-<feature-name> fetch origin
     git -C ../ai-tui-<feature-name> rebase dev
   - Fast-forward dev:
     git checkout dev
     git merge --ff-only feature/<feature-name>
   - Clean up:
     git worktree remove ../ai-tui-<feature-name>
     git branch -d feature/<feature-name>
```

### CRITICAL: Atomic Work Units

A "work unit" is a **feature + its tests together**. Testing is NOT a separate task.

- NEVER create separate beads/tasks for "write tests" — tests are part of the feature.
- When dispatching, always include: "Implement X **and** write tests for X. Both must pass before you commit."
- One commit (or a small, logical commit chain) per work unit. Not one commit for code and another for tests.

---

## Subagent Workflow (Sonnet — Worktree)

You are a **subagent** working in a dedicated git worktree. Your job is to deliver a complete, tested feature.

### Rules

1. **Work only in your worktree.** Never touch files outside your worktree path.
2. **Feature + tests are one unit.** Implement the feature and its tests together. Do not commit code without tests or tests without code.
3. **Run quality gates before committing:**
   - Run the project's test suite (the relevant subset at minimum).
   - Run any linters/formatters configured in the project.
   - ALL checks must pass. If they fail, fix and re-run.
4. **Make atomic commits.** Each commit should be a coherent, self-contained change. Feature code and its tests belong in the same commit.
5. **Do NOT rebase or merge into dev.** That is the orchestrator's job. Just leave your branch clean and passing.
6. **Do NOT push.** The orchestrator handles integration.
7. **Signal completion clearly.** When done, your final message MUST:
   - Start with: **`READY TO ASSIMILATE INTO DEV!!!!`**
   - Summary of what was implemented
   - Test results (paste the output)
   - Confirmation that all quality gates passed
   - List of commits made

### Commit Message Format

```
<type>: <concise description>

- What was implemented/changed
- What tests were added
- Any notable decisions
```

Types: `feat`, `fix`, `refactor`, `docs`, `chore`

---

## Landing the Plane (Session Completion)

**When ending a work session**, the orchestrator MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **Verify all subagent work is integrated** — All worktrees removed, all features rebased into `dev`.
2. **File issues for remaining work** — Create beads for anything that needs follow-up.
3. **Run full quality gates on `dev`** — Tests, linters, builds on the integrated branch.
4. **Update issue status** — Close finished work, update in-progress items.
5. **PUSH TO REMOTE:**
   ```bash
   git checkout dev
   git pull --rebase origin dev
   bd sync
   git push origin dev
   git status  # MUST show "up to date with origin"
   ```
6. **Clean up** — Remove all worktrees, delete merged feature branches, clear stashes.
7. **Verify** — All changes committed AND pushed. `git worktree list` shows only the main worktree.
8. **Hand off** — Provide context for next session.

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing — that leaves work stranded locally
- NEVER say "ready to push when you are" — YOU must push
- If push fails, resolve and retry until it succeeds
- NEVER amend commits — always create new commits
- ALWAYS rebase, never merge (when integrating into dev)

