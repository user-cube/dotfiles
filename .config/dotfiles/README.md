# dotfiles (install steps)

What runs after the files are copied into place, the same for `dotfiles-sync`
and the playbook: a new step goes in `install.sh`, in this repo only.

| File | |
|---|---|
| `install.sh` | builds the Go tools whose sources changed, downloads the SketchyBar icon map, installs the Hue VS Code extension, links lazygit's config, removes obsolete files; prints a line per change, nothing when all was up to date, fails when a build fails |
| `obsolete` | files this repo used to install, relative to `$HOME`: rsync never deletes, so they are removed here. Add a line when a file leaves the repo (never a directory, a glob or a path outside `$HOME`) |
| `remove-obsolete.sh` | removes them (only files and symlinks, then the folder when it is left empty) |
| `local-files/` | `local-backup` / `local-restore`: a Go tool, see below |

The Go tools `install.sh` builds (source folder → binary in it, rebuilt when a
`.go` file or `go.mod` is newer than the binary):

- `~/git_hooks/go/commit-msg`: the global commit-msg hook
- `~/.config/dotfiles/local-files`
- `~/.config/sketchybar/plugins/check`: the SketchyBar connectivity checks
- `~/.claude/hooks/deny-sensitive-reads`: Claude Code's PreToolUse hook (Bash
  and Read are blocked until it is built)

## local-files

The per-machine files that never go to git (`~/.zshrc.local`, `~/.gitconfig-work`,
`~/.kube/config`, the tmux and SketchyBar per-machine files, Terraform tokens,
the `mcpServers` of `~/.claude.json`, …), as one `.tar.gz` attached to a
Bitwarden secure note `Local files (<machine>)`. It uses the `bw` wrapper's
session.

```sh
local-backup                 # this machine -> Bitwarden (replaces the previous copy)
local-restore                # Bitwarden -> this machine; existing files are kept
local-restore other-mac      # another machine's backup (e.g. a new Mac)
local-restore --force        # overwrite existing files
```

The list of files is in `main.go`; `go test ./...` covers the archive and the
MCP merge.
