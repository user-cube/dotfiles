# zsh functions

Shared shell functions: `.zshrc` sources every `*.zsh` here, before
`~/.zshrc.local`. One file per tool, no secrets and no machine-specific names
(those go in `~/.zshrc.local`).

| File | Commands | |
|---|---|---|
| `bw.zsh` | `bw` | Bitwarden CLI wrapper: the session key in a root-only file, unlocked with Touch ID/sudo; needs `BW_USER` in `~/.zshrc.local` |
| `secrets.zsh` | `secret T [item]` | read a token without echoing it (or from Bitwarden) for the next command; lines that look like they hold a token never reach the history |
| `sops.zsh` | `sops-on [60m]`, `sops-off`, `sops-status`, `sops-save` | the SOPS age keys from Bitwarden, in a file for a limited time |
| `dotfiles.zsh` | `dotfiles-sync`, `dotfiles-diff [-p]` | update this machine from `~/dotfiles` without Ansible, or see what that would change |
| `local-files.zsh` | `local-backup`, `local-restore [machine]` | the per-machine files (never in git) to and from Bitwarden |
| `vscode-workspaces.zsh` | `ws` | open or create a VS Code workspace in `~/workspaces` (templates, Hue colors) |
| `kube-guard.zsh` | `kubectl`, `kubecolor` (`k`), `helm`, `k9s` | on a production context (`prod` / `prd`, `KUBE_PROD_REGEX`) asks before a command that changes the cluster (apply, delete, edit, scale, rollout restart, drain, helm upgrade, …); reads pass. `KUBE_GUARD=0` skips it once. k9s opens read-only there |
| `plugins-outdated.zsh` | `plugins-outdated` | the zsh/tmux plugins the playbook pins that master has moved past; every 14 days a background check and a macOS notification |

The details are in each file's header comment and in the [main README](../../README.md).

## What `.zshrc` sets up

- Oh My Zsh with Powerlevel10k; plugins from `custom/plugins`, cloned at pinned
  commits by the playbook: zsh-autosuggestions (also suggests from completions),
  fzf-tab (Tab completes in an fzf list, hidden files included), and
  zsh-syntax-highlighting, which must stay last
- Oh My Zsh only notes its updates at startup (`omz update` when you want)
- small Oh My Zsh plugins: **Esc Esc** puts `sudo` in front of the line (or of the
  last command); **Ctrl+O** copies the line being typed, without running it;
  `copypath [file]` copies a path; `extract FILE` unpacks any archive; `ofd` opens
  this folder in Finder, `cdf` goes to Finder's, `quick-look FILE` previews
- history: 100000 lines shared between panes, with start time and duration
  (`history -i`), no duplicates, `!!` expanded before it runs
- fzf: Ctrl+T files, Ctrl+G folders, Ctrl+R history, listed with `fd`
  (hidden files in, `.gitignore` respected) and previewed with `bat` / `ls`
- `cd` is zoxide: `cd dotfiles` from anywhere, `cdi` to pick; `z` still works
- `~/.krew/bin` on PATH: `kubectl neat`, `tree`, `view-secret`, `who-can`; their Tab
  completion (and `kubectl krew`'s) comes from `../kubectl/complete/`
- `k` = kubecolor; `vim` = nvim; `lz` / `lzd` = lazygit / lazydocker; `ktx` = kontext
- startup around 0.1 s (`time zsh -i -c exit`): nvm loads on first use, and
  the pyenv/zoxide/fzf inits and tool completions are cached in `~/.cache/zsh`,
  regenerated when the tool or its command line changes (delete the folder to
  force it)
