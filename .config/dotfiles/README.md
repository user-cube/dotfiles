# dotfiles (install steps)

What runs after the files are copied into place, the same for `dotfiles-sync`
and the playbook: a new step goes in `install.sh`, in this repo only. The guide:
[Installing and updating](https://github.com/user-cube/dotfiles/wiki/Installing-and-updating).

| File | |
|---|---|
| `install.sh` | builds the Go tools whose sources changed, downloads the SketchyBar icon map, installs the VS Code extensions and links its settings, links lazygit's config, copies the LaunchAgents, applies Rex's Settings, removes obsolete files; prints a line per change, nothing when all was up to date, fails when a build fails |
| `home-files` | the files of the repo's root copied to `~` (`.zshrc`, the git configs…), overwriting them: the one list `dotfiles-sync`, `dotfiles-diff`, the install test and the playbook read. `.gitconfig-work` is apart: only seeded when missing |
| `launchd/` | LaunchAgents, copied to `~/Library/LaunchAgents` by `install.sh` and run at login (not loaded on install): `com.user-cube.sops-on-login` removes the age keys `sops-on` left when the session ended. Off on one machine: `launchctl disable gui/$(id -u)/<label>` (macOS keeps it across reboots and syncs; `enable` turns it back on); for this one, `sops-login off` / `on` |
| `obsolete` | files this repo used to install, relative to `$HOME`: rsync never deletes, so they are removed here. Add a line when a file leaves the repo (never a directory, a glob or a path outside `$HOME`) |
| `remove-obsolete.sh` | removes them (only files and symlinks, then the folder when it is left empty) |
| `local-files/` | `local-backup` / `local-restore`: a Go tool, see below |

The Go tools `install.sh` builds (source folder → binary in it, rebuilt when a
`.go` file or `go.mod` is newer than the binary):

- `~/git_hooks/go/commit-msg`: the global commit-msg hook
- `~/git_hooks/go/sops-guard`: the global pre-commit hook against cleartext
  secrets (commits stop until it is built)
- `~/.config/kube-guard`: what `kubectl` / `helm` / k9s / sofka do on production,
  and the SketchyBar kube item's context
  (until it is built every command asks)
- `~/.config/dotfiles/local-files`
- `~/.config/sketchybar/plugins/check`: the SketchyBar connectivity checks
- `~/.claude/hooks/deny-sensitive-reads`: Claude Code's PreToolUse hook (Bash
  and Read are blocked until it is built)

## local-files

`local-backup` / `local-restore`: the per-machine files that never go to git, as
one `.tar.gz` attached to the Bitwarden secure note `Local files (<machine>)`.
The list of files is in `main.go`; `go test ./...` covers the archive and the
MCP merge. Usage: [Installing and updating](https://github.com/user-cube/dotfiles/wiki/Installing-and-updating#backup-in-bitwarden)
in the wiki.
