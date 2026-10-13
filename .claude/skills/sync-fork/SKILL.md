---
name: sync-fork
description: Sync this fork's master with upstream jesseduffield/lazygit, verify, and push — then report. Invoke by hand as /sync-fork.
disable-model-invocation: true
allowed-tools: Bash(git:*) Bash(just:*) Bash(gh:*)
---

Fixed pipeline, no arguments, run start to finish every time it's invoked. Every
step below still owes the final report — an early stop skips later steps, not
the report.

## 1. Preflight

Confirm the current branch is `master`, `git status --porcelain` is empty, and
`git remote -v` shows `fork` → `khanhtd36/lazygit` and `origin` →
`jesseduffield/lazygit` (the layout `CONTEXT.md` documents: `origin` is
upstream here). Any check fails → stop, tell the user which one and why, skip
everything else including push, then go straight to the Report with the other
sections marked not reached.

## 2. Sync with fork

`GH_CONFIG_DIR="$HOME/.config/gh-khanhtd36" git pull --ff-only fork master`.
Done when `git merge-base --is-ancestor fork/master master` succeeds: local
`master` holds everything on `fork`. Work also happens from other machines,
so a stale `master` means merging against old code and a rejected push.
Local commits ahead of `fork` are fine. Not a fast-forward (both sides have
new commits) → stop, list each side's commits, skip everything else, and go
straight to the Report.

## 3. Fetch upstream

`git fetch origin master`. Run `git log --oneline master..origin/master` and
keep its output verbatim as the **changelog** — one line per upstream commit
about to be merged in.

Empty output → nothing to merge. Skip straight to the Report: changelog is
"up to date, nothing to merge", conflict resolution and verify/push status are
"not reached".

## 4. Merge

`git merge origin/master --no-edit`.

On conflict: stop immediately. Do not resolve it. Leave the merge in
progress exactly as git left it. Run `git diff --name-only --diff-filter=U`
and keep that file list for the report. Skip build/test/push.

## 5. Verify

`just build`, then `just unit-test`. If either fails, stop before push. Keep the failing
command and the tail of its output for the report.

## 6. Push

`GH_CONFIG_DIR="$HOME/.config/gh-khanhtd36" git push fork master`. If that
fails or hangs, fall back to:

```
GH_CONFIG_DIR="$HOME/.config/gh-khanhtd36" git push https://github.com/khanhtd36/lazygit.git master:master
```

(the personal `gh` login provides the HTTPS credential, bypassing the SSH
agent). Record which of the two actually succeeded.

## 7. Report

Always end with exactly these three sections, in this order, regardless of
where the pipeline stopped:

```
## Changelog
<the commits merged in, or "up to date, nothing to merge">

## Conflict resolution
<"none", or the conflicting files and that the merge was left in progress
for the user to resolve by hand>

## Verify and push status
<build result, unit-test result, and which push path
succeeded — or "not reached" for whichever steps a stop above skipped>
```
