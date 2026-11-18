---
name: pr
description: Push the current branch and open a pull request ready for review, with a body that says what changed, why and what a reviewer should know. Use only when the user asks for a PR.
argument-hint: "[draft] [base branch]"
disable-model-invocation: true
allowed-tools: Bash(git status:*), Bash(git diff:*), Bash(git log:*), Bash(git fetch:*), Bash(git ls-remote:*), Bash(git push:*), Bash(git merge:*), Bash(git add:*), Bash(git commit:*), Bash(gh pr:*)
---

Open a pull request for the current branch. Arguments from the user: $ARGUMENTS
(`draft` makes it a draft; a branch name is the base. Without arguments: ready for review, against `main`.)

## Steps

1. `git status`. Uncommitted changes: commit them first, with the rules of `/commit`
   (and a feature branch if you are on `main`, `master` or `develop`).
2. `gh pr list --head <branch> --json url,state`. An open PR already exists: push the new commits,
   report its URL and stop. Do not create a duplicate and do not change its draft state.
3. The base: the one asked for, else `main`. Check it exists with `git ls-remote --heads origin <base>`;
   if it does not, leave `--base` off and let gh infer it.
4. `git push -u origin <branch>`.
5. Read what the PR holds: `git log <base>..HEAD --format=%s` and `git diff <base>...HEAD --stat`, and
   the diff itself for what the body must explain.
6. Write the body to a file with the Write tool (the session's scratchpad directory if it has one,
   else `${TMPDIR:-/tmp}`) and pass it with `--body-file`. Not `--body` with a heredoc: backticks and
   `$` get escaped and show up as stray backslashes in the PR.
7. `gh pr create --base <base> --title "<title>" --body-file <file>`, plus `--draft` only when asked.
8. A minute later: `gh pr view --json mergeable,mergeStateStatus`. `CONFLICTING`: merge `origin/<base>`
   into the branch, resolve it (keep both sides when both added something), run the tests, push.
9. Report the URL, on a line of its own.

## The title

Same rule as a commit header: `type(scope): what changed`, English, under 100 characters, no full stop.

## The body

```
## What
What the change is, in terms of what now exists or behaves differently. The files that matter, not all.

## Why
The problem it solves, or what was missing.

## For the reviewer
What is not obvious: a limit, a decision, a thing left out on purpose, how to try it, tests that
fail and whether they failed before this change.
```

Say what you did not verify. If tests fail, say which and why, never "all green" over a red one.
No attribution lines: the `git-guard` hook denies `Co-Authored-By` and "Generated with Claude" in a PR.

## Never

Merge the PR, turn on auto-merge, or force-push. Those are the user's calls.
