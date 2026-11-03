# zsh functions

Shared shell functions: `.zshrc` sources every `*.zsh` here, before
`~/.zshrc.local`. One file per tool, no secrets and no machine-specific names
(those go in `~/.zshrc.local`).

| File | Commands | |
|---|---|---|
| `bw.zsh` | `bw` | Bitwarden CLI wrapper: the session key in a root-only file, unlocked with Touch ID/sudo; needs `BW_USER` in `~/.zshrc.local` |
| `secrets.zsh` | `secret T [item]` | read a token without echoing it (or from Bitwarden) for the next command; lines that look like they hold a token never reach the history |
| `sops.zsh` | `sops-on [60m]`, `sops-off`, `sops-status`, `sops-save`, `sops-login [on\|off]` | the SOPS age keys from Bitwarden, in a file for a limited time; a log out or restart removes them (a LaunchAgent at login, `../dotfiles/launchd`; `sops-login off` turns that off on one machine) |
| `dotfiles.zsh` | `dotfiles-sync`, `dotfiles-diff [-p]` | update this machine from `~/dotfiles` without Ansible, or see what that would change |
| `local-files.zsh` | `local-backup`, `local-restore [machine]` | the per-machine files (never in git) to and from Bitwarden |
| `vscode-workspaces.zsh` | `ws` | open or create a VS Code workspace in `~/workspaces` (templates, Hue colors) |
| `kube-guard.zsh` | `kubectl`, `kubecolor` (`k`), `helm`, `k9s`, `sofka` | on a production context (`prod` / `prd`; per machine `~/.config/kube-guard/prod-regex`, per shell `KUBE_PROD_REGEX`) asks before a command that changes the cluster (apply, delete, edit, scale, rollout restart, drain, helm upgrade, …); reads pass. `KUBE_GUARD=0` skips it once. k9s and sofka open read-only there; the ⌘K popup gets the pane's KUBECONFIG (a `ktx -s` shell's own), kept per pane in `~/.cache/kube-pane`. The decision is the Go binary `../kube-guard` (built by `install.sh`); until it is built every kubectl / helm command asks and the TUIs open read-only |
| `loki.zsh` | `logcli` | Loki's logs (deleted pods and finished jobs too): without `LOKI_ADDR`, port-forwards to `LOKI_SERVICE` (default `monitoring/loki`) for the command |
| `plugins-outdated.zsh` | `plugins-outdated` | the zsh/tmux plugins the playbook pins that master has moved past; every 14 days a background check and a macOS notification |

The details are in each file's header comment. What `.zshrc` sets up (plugins,
history, fzf, aliases, startup time) and how to use it day to day: the
[Shell](https://github.com/user-cube/dotfiles/wiki/Shell) page of the wiki.
