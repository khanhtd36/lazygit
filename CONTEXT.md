# lazygit

A terminal UI for git commands, wrapping the git CLI to give a fast, keyboard-driven interface for everyday repo operations.

## Language

**Pull tag**:
A Tags-panel action that runs `git fetch <remote> --tags` to bring *all* tags from a chosen remote up to date locally — not an action on any single selected tag. Add/update only: never deletes a local tag whose remote counterpart was removed. On a non-fast-forward conflict (a local tag points elsewhere than the remote's), prompts once to retry the whole fetch with `--force`, applying to every conflicting tag in that batch, not per-tag.
_Avoid_: Fetch tag, sync tag, update tag
