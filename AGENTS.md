# Agent Instructions

This project uses **kata** for issue tracking. Kata is a daemon-backed
tracker (issues live in `~/.kata/kata.db`, NOT in the repo). The workspace
is bound via the committed `.kata.toml`; per-machine overrides go in
`.kata.local.toml` (gitignored). Run `kata whoami` to confirm your actor;
if `.kata.toml` is missing, run `kata init --project ai-tui`.

Issue refs are short_ids derived from each issue's ULID (e.g. `qn3q`).
Cross-project refs are `kata#qn3q`; full 26-char ULIDs also resolve.

## Quick Reference

```bash
kata ready --unowned --agent            # Find unclaimed, unblocked work
kata show <ref> --agent                 # View issue + body + comments
kata claim <ref>                        # Claim ownership (fails if taken)
kata unassign <ref>                     # Release ownership
kata create "<title>" --body "<...>" \  # Create an issue
  --idempotency-key "<key>" --agent
kata search "<query>" --agent           # Search before creating
kata comment <ref> --body "<...>"       # Leave context on an issue
kata close <ref> --done \               # Close with evidence
  --message "<substantive prose>" --commit <sha>
```

Always pass `--agent` for ordinary reads/mutations in agent logs. Use
`--json` only when piping into jq.

---

## Branch Strategy

- `main` — stable, release-ready code. Never commit directly.
- `dev` — integration branch. All feature work merges here first.
- `feature/<name>` — short-lived branches in worktrees, one per task.

If `dev` does not exist, the orchestrator creates it from `main` before dispatching work.

---

## Orchestrator Role (Opus — Main Session)

You are the **orchestrator**. You do NOT write feature code directly. Your job:

1. **Triage work** — Use `kata ready --unowned --agent` to find unclaimed, unblocked tasks. Group related items into atomic work units (one kata issue = one work unit, or a cohesive set).
2. **Dispatch to subagents** — Launch a **sonnet** subagent (via the Task tool) for each work unit. Each subagent works in its own **git worktree**.
3. **Verify and integrate** — After a subagent completes, verify the result and rebase into `dev`.
4. **Push** — Keep `dev` pushed to remote. (Kata state is daemon-backed; there is no `sync` step — do not commit issue state.)

### Dispatching a Subagent

For each work unit, the orchestrator MUST:

```
1. Create the worktree:
   git worktree add ../ai-tui-<feature-name> dev -b feature/<feature-name>

2. Launch a sonnet subagent (Task tool, model=sonnet) with:
   - The worktree path as working directory context
   - The kata issue ref (e.g. dnyr) — the subagent reads full notes via
     `kata show <ref> --agent` from the worktree (the .kata.toml binding
     resolves the project)
   - The full task description (feature + test requirements)
   - Explicit instruction to follow the Subagent Workflow below,
     including `kata claim <ref>` at the start

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

- NEVER create separate kata issues for "write tests" — tests are part of the feature.
- When dispatching, always include: "Implement X **and** write tests for X. Both must pass before you commit."
- One commit (or a small, logical commit chain) per work unit. Not one commit for code and another for tests.

---

## Subagent Workflow (Sonnet — Worktree)

You are a **subagent** working in a dedicated git worktree. Your job is to deliver a complete, tested feature for one kata issue.

### Rules

1. **Work only in your worktree.** Never touch files outside your worktree path.
2. **Claim your issue first.** Run `kata claim <ref>` so other agents do not grab it. If it is already claimed by another actor, stop and report back.
3. **Read the notes.** Run `kata show <ref> --agent` and implement exactly what the body specifies (problem, suggested fix, acceptance, required tests).
4. **Feature + tests are one unit.** Implement the feature and its tests together. Do not commit code without tests or tests without code.
5. **Run quality gates before committing:**
   - Run the project's test suite (the relevant subset at minimum).
   - Run any linters/formatters configured in the project.
   - ALL checks must pass. If they fail, fix and re-run.
6. **Make atomic commits.** Each commit should be a coherent, self-contained change. Feature code and its tests belong in the same commit.
7. **Do NOT rebase or merge into dev.** That is the orchestrator's job. Just leave your branch clean and passing.
8. **Do NOT push.** The orchestrator handles integration.
9. **Close your issue when the work is verified** — not in a batch at the end. Close eagerly with typed evidence:
   ```
   kata close <ref> --done \
     --message "<substantive prose: what was fixed/added, what was tested>" \
     --commit <sha>
   ```
   If the work is NOT actually done, DO NOT close. Instead:
   ```
   kata label add <ref> needs-review
   kata comment <ref> --body "what was attempted, what remains"
   ```
10. **Signal completion clearly.** When done, your final message MUST:
   - Start with: **`READY TO ASSIMILATE INTO DEV!!!!`**
   - The kata issue ref and its close status
   - Summary of what was implemented
   - Test results (paste the output)
   - Confirmation that all quality gates passed
   - List of commits made (shas)

The daemon throttles >3 sibling closes by one actor under one parent in 5
minutes; close issues as each is verified (spread over time) and you will
not hit it.

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
2. **File issues for remaining work** — Create kata issues (`kata create ... --idempotency-key ... --agent`) for anything that needs follow-up. Search before creating to avoid duplicates.
3. **Run full quality gates on `dev`** — Tests, linters, builds on the integrated branch.
4. **Reconcile issue state** — Confirm every dispatched kata issue is either closed (with evidence) or marked `needs-review` with a comment explaining what remains. Reopen/comment any that were closed prematurely.
5. **PUSH TO REMOTE:**
   ```bash
   git checkout dev
   git pull --rebase origin dev
   git push origin dev
   git status  # MUST show "up to date with origin"
   ```
6. **Clean up** — Remove all worktrees, delete merged feature branches, clear stashes.
7. **Verify** — All changes committed AND pushed. `git worktree list` shows only the main worktree.
8. **Hand off** — Provide context for next session (open kata issues, what is blocked, what is ready).

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing — that leaves work stranded locally
- NEVER say "ready to push when you are" — YOU must push
- If push fails, resolve and retry until it succeeds
- NEVER amend commits — always create new commits
- ALWAYS rebase, never merge (when integrating into dev)
- Do NOT run `kata delete` or `kata purge` unless the user explicitly asks for that destructive action on a specific ref.
