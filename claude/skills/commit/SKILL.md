---
name: commit
description: Commit the working tree changes with a message that follows this setup's rules (conventional header, English, no attribution), on a feature branch. Use only when the user asks to commit.
argument-hint: "[what the commit is about]"
disable-model-invocation: true
allowed-tools: Bash(git status:*), Bash(git diff:*), Bash(git log:*), Bash(git branch:*), Bash(git switch:*), Bash(git add:*), Bash(git commit:*)
---

Commit the changes in the working tree. Optional hint from the user: $ARGUMENTS

## Steps

1. Look first: `git status`, `git diff` (staged and unstaged) and `git log -5 --format=%s`, so the message
   matches what the repository already does.
2. On `main`, `master` or `develop`: switch to a new branch first (`git switch -c feat/<short-slug>`).
   The global commit-msg hook blocks commits there. Keep an issue key from the work in the branch
   name (`feat/ABC-123-what`): the hook copies it into the header.
3. Stage the files that belong to this change, by name. No `git add -A` or `git add .` unless every
   changed file belongs. Leave out anything that looks like a secret (env files, keys, tokens): say so
   and stop if one is among the changes.
4. Commit with `-m` for the header and another `-m` for the body.

## The message

- Header: `type(scope): what changed`, in English, imperative or descriptive, under 100 characters,
  **no full stop at the end**. Types: feat, fix, chore, docs, style, refactor, perf, test, build, ci.
  Write the type yourself; the hook only adds a missing one.
- Body: the why, when the header does not say it. One or two short paragraphs, not a list of files.
- No `Co-Authored-By`, no "Generated with Claude", no other attribution: the `git-guard` hook denies it.
- No `--no-verify` / `-n` and no `-c core.hooksPath=...`: the hooks are the checks, and `git-guard`
  denies the bypass.
- Do not write a literal secret-looking path in the text (a private key or credentials path): the
  `deny-sensitive-reads` hook reads the command text and blocks it. Say "the ssh keys" instead.

## When it fails

A hook that rejects the commit is telling you something: read its message, fix the cause and make a
**new** commit. Do not amend, do not skip the hook.

## After

Say the branch, the short hash and the header. Do not push: that is `/pr` or the user's call.
