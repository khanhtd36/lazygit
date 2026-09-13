---
name: sync-fork
description: Sync this fork's master with upstream/master, verify, deploy, and push — then report. Invoke by hand as /sync-fork.
disable-model-invocation: true
allowed-tools: Bash(git:*) Bash(just:*) Bash(gh:*)
---

Fixed pipeline, no arguments, run start to finish every time it's invoked. Every
step below still owes the final report — an early stop skips later steps, not
the report.

## 1. Preflight

Confirm the current branch is `master` and `git status --porcelain` is empty.
Either check fails → stop, tell the user which one and why, skip everything
else including deploy/push, then go straight to the Report with the other
sections marked not reached.

## 2. Fetch

`git fetch upstream`. Run `git log --oneline master..upstream/master` and keep
its output verbatim as the **changelog** — one line per upstream commit about
to be merged in.

Empty output → nothing to merge. Skip straight to the Report: changelog is
"up to date, nothing to merge", conflict resolution and deploy status are
"not reached".

## 3. Merge

`git merge upstream/master --no-edit`.

On conflict: stop immediately. Do not resolve it. Leave the merge in
progress exactly as git left it. Run `git diff --name-only --diff-filter=U`
and keep that file list for the report. Skip build/test/deploy/push.

## 4. Verify

`just build`, then `just unit-test`. If either fails, stop before deploy and
push. Keep the failing command and the tail of its output for the report.

## 5. Deploy

`just install` (the release build via `go install`, to `~/go/bin/lazygit`).

## 6. Push

`git push` on `origin` first. If that fails or hangs, fall back to:

```
git push https://github.com/khanhtd36/lazygit.git master:master
```

(this repo's `gh` login provides the HTTPS credential, bypassing the SSH
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

## Deploy status
<build result, unit-test result, install result, and which push path
succeeded — or "not reached" for whichever steps a stop above skipped>
```
