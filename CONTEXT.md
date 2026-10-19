# lazygit

A terminal UI for git commands, wrapping the git CLI to give a fast, keyboard-driven interface for everyday repo operations.

## Language

**Pull tags**:
A Tags-panel action that runs `git fetch <remote> --tags` to bring *all* tags from a chosen remote up to date locally — not an action on any single selected tag. Add/update only: never deletes a local tag whose remote counterpart was removed. On a non-fast-forward conflict (a local tag points elsewhere than the remote's), prompts once to retry the whole fetch with `--force`, applying to every conflicting tag in that batch, not per-tag.
_Avoid_: Pull tag (singular — the action always operates on the whole remote tag set, never one tag), Fetch tag, sync tag, update tag

**Fork release**:
A tagged build+publish run of `khanhtd36/lazygit` (the `fork` remote), triggered by pushing a `fork-v*` tag (e.g. `fork-v0.65.2`). Runs goreleaser against `.goreleaser.yml`, publishes a GitHub Release on `khanhtd36/lazygit`, and submits an update to the fork winget package. Distinct namespace from upstream's own `vX.Y.Z` tags (on `origin` = `jesseduffield/lazygit`) precisely so pushing/syncing upstream tags can never accidentally trigger it.
_Avoid_: release (ambiguous — could mean upstream's release), build

**Installer scripts**:
`website/install.sh` (POSIX) and `website/install.ps1` (Windows) — fetch the latest fork release asset for the caller's OS/arch directly from the GitHub Releases API (`api.github.com/repos/khanhtd36/lazygit/releases/latest`), no intermediate manifest file. Served at the install site.
_Avoid_: install script (singular — there are two, one per platform)

**Install site**:
`lazygit.khanhtd36.dev`, a GitHub Pages deployment of `website/` on `khanhtd36/lazygit`, hosting the installer scripts. DNS delegation done by user; GitHub Pages custom-domain config is the remaining piece.

**Self-update**:
Replacing the running lazygit binary with the latest **Fork release**, from the GitHub Releases of `khanhtd36/lazygit` only (upstream is never consulted). Reachable as the `lazygit update` command, as a startup check governed by `update.method`, and as the `u` key in the TUI. Only plain `X.Y.Z` versions are comparable; a build with no version never sees updates. Never restarts lazygit itself.
_Avoid_: upgrade, auto-update (an update is never installed without the user asking or confirming, except `update.method: background`)

**Managed install**:
A lazygit binary owned by a package manager (Homebrew, winget, `go install`, Nix), recognised by its executable path. **Self-update** refuses to replace it and shows that package manager's own upgrade command instead.
_Avoid_: package install, system install

**Fork winget package**:
`khanhtd36.lazygit-khanhtd36` — a winget-pkgs manifest submitted (via `wingetcreate.exe update ... --submit`) on each fork release, using the `WINGET_TOKEN` repo secret (added per-repo manually; `khanhtd36` is a personal account, not an org, so secrets don't share across repos).
