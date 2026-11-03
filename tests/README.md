# Tests

> The guide: [Development](https://github.com/user-cube/dotfiles/wiki/Development#tests) in the wiki. This README is the reference.

pytest, run by CI (`.github/workflows/lint.yml`). Fake commands (kubectl,
helm, git, osascript...) stand in for the real ones on PATH, so no test
reaches a cluster, the network or the notification center.

```sh
python3 -m venv .venv && .venv/bin/pip install -r tests/requirements.txt
.venv/bin/pytest tests/            # everything, ~15 s
.venv/bin/pytest tests/ -k guard   # only the tests whose name has "guard"
```

They need `tmux`, `zsh`, `bat`, `fzf` (without it the Ctrl+R test is skipped)
`tofu` (or `terraform`; without either the hook's tests are skipped),
`shellcheck` and `hadolint` (each hook's tests skipped without its tool).

| File | Checks |
|---|---|
| `test_shortcuts.py` | each Ghostty ⌘ shortcut is bound in tmux and listed in the ⌘⇧H cheat sheet, with the same ⌘ keys; each cheat sheet line runs a key tmux binds |
| `test_dotfiles_sync.py` | after the pull, `dotfiles-sync` goes on with the pulled `dotfiles.zsh`, not the older one the shell loaded |
| `test_k9s.py` | plugin keys k9s can use (not Ctrl-H/I/J/M/P), none twice in a view, the ones that change the cluster `dangerous`; hotkeys that work on a Portuguese keyboard |
| `test_sofka.py` | plugin keys that aren't sofka's own table keys (ctrl-f is page down), none twice in a view, each says whether it changes the cluster; `sofka info --offline` loads the config without warnings (skipped without sofka) |
| `test_config_files.py` | every TOML, YAML and JSON file parses; `tmux.conf` loads without errors; bat's theme exists |
| `test_kube_guard.py` | (builds `.config/kube-guard` with go) on production, kubectl / kubecolor / helm ask before changing the cluster (answered y or n in a terminal, through pexpect), reads pass, `--context` anywhere, a piped manifest still reaches kubectl; k9s and sofka read-only; the ⌘K popup gets the pane's KUBECONFIG, and the file is put back or removed when the shell exits; not built yet: every command asks |
| `test_terraform_fmt_hook.py` | the `tofu fmt` pre-commit hook on staged files, in a throwaway repo |
| `test_lint_hooks.py` | the shellcheck, hadolint and sops-guard pre-commit hooks on staged files (shebang detection, the staged `.sops.yaml`'s `path_regex`, encrypted formats), in a throwaway repo; sops-guard is built with go first |
| `test_kubectl_complete.py` | the Tab completion scripts of krew and its plugins |
| `test_sops_on.py` | at login (the LaunchAgent): the keys `sops-on` left are removed, also from a `SOPS_KEYS_FILE` only `~/.zshrc.local` set; a keys file `sops-on` didn't create stays; `sops-login on` / `off` (a fake launchctl) |
| `test_plugins_outdated.py` | the pins read from the playbook, which master moved past, and the 14-day notification |
| `test_tmux_sessions.py` | the resurrect hook that keeps throwaway sessions (a second Ghostty window) out of the save; on a tmux server of its own |
| `test_zsh_history.py` | ↑ (both cursor modes) and Ctrl+R see what another shell ran since the last prompt, and ↑ keeps its prefix search going; two zsh in terminals (pexpect) on a throwaway history |

Hue, the VS Code extension, has its own: `node --test .config/vscode/extensions/hue/colors.test.js`
(`colors.test.js`: its colors, the danger places, the colorCustomizations it
writes, and `extension.js` loaded with a fake `vscode`).

`test_install.py` is the install path end to end: `dotfiles-sync` into an empty
fake HOME (one scenario, its tests in order), with fakes for the apps a sync
touches (`defaults` for Rex's Settings, tmux, AeroSpace, SketchyBar, brew, VS
Code), so it runs on a Mac without touching them. It downloads the SketchyBar
icon map (network) and needs go, zsh and rsync.
