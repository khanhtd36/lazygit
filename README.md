# lazygit-khanhtd36

> **Unofficial fork.** This is `khanhtd36`'s fork of [lazygit](https://github.com/jesseduffield/lazygit), not affiliated with or endorsed by the original project. For the official release, see [jesseduffield/lazygit](https://github.com/jesseduffield/lazygit).

<p align="center">
  <img width="536" src="https://user-images.githubusercontent.com/8456633/174470852-339b5011-5800-4bb9-a628-ff230aa8cd4e.png">
</p>

<p align="center">

[![License](https://img.shields.io/github/license/khanhtd36/lazygit)](LICENSE) [![Release](https://img.shields.io/github/v/release/khanhtd36/lazygit)](https://github.com/khanhtd36/lazygit/releases)

</p>

## what's different in this fork

- Add a keybinding to collapse the parent directory of the selected file ([#1](https://github.com/khanhtd36/lazygit/pull/1))
- Push/pull the selected branch from the Branches panel ([#2](https://github.com/khanhtd36/lazygit/pull/2))
- Always confirm before pushing the selected branch ([#3](https://github.com/khanhtd36/lazygit/pull/3))
- Always prompt for the upstream when pushing the selected branch ([#4](https://github.com/khanhtd36/lazygit/pull/4))
- fix(gocui): bail on UI thread and wait once MainLoop exits ([#5](https://github.com/khanhtd36/lazygit/pull/5))

## install (this fork)

```sh
curl -fsSL https://lazygit.khanhtd36.dev/install.sh | sh
```

```powershell
irm https://lazygit.khanhtd36.dev/install.ps1 | iex
```

```sh
winget install khanhtd36.lazygit-khanhtd36
```

Binaries and checksums: [releases](https://github.com/khanhtd36/lazygit/releases).

## docs

Everything else — usage, keybindings, configuration, contributing — lives upstream: [jesseduffield/lazygit](https://github.com/jesseduffield/lazygit).

## development

- `just build`
- `just unit-test`
- `just e2e`
- `just lint`
- `just format` (run before every commit)

## license

MIT
