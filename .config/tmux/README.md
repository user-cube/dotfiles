# tmux

Tabs, splits and sessions for Ghostty: every Ghostty window runs inside tmux
(`../ghostty/tmux-start.sh`), and Ghostty's ⌘ shortcuts send the tmux prefix
(`C-b`) plus a key. ⌘⇧H lists them all (`cheatsheet.sh`); keep that list in sync
when a shortcut is added.

## Shortcuts

| ⌘ (Ghostty) | tmux | What |
|---|---|---|
| ⌘T, ⌘W | `C-b c`, `C-b X` | new tab (current folder), close pane |
| ⌘⇧← / ⌘⇧→, ⌘1–9 | `C-b p` / `n`, `C-b 1`–`9` | previous / next tab, tab N |
| ⌘⇧R | `C-b ,` | rename the tab (`rename-window.sh`) |
| ⌘D / ⌘⇧D | `C-b %` / `"` | split right / down; a pane in `ssh host` opens the same ssh (`split.sh`) |
| ⌘⌥←↑↓→, ⌘⇧F, ⌘⇧E | `C-b h j k l`, `z`, `E` | move between panes, zoom, even sizes |
| ⌘O | `C-b f` | open a repo under `~/work` as a session (`sessionizer.sh`) |
| ⌘⇧O | `C-b O` | open an ssh host in a new tab (`ssh-picker.sh`) |
| ⌘⇧S, — | `C-b s`, `C-b S` | switch session, manage sessions (`sessions.sh`) |
| ⌘G, ⌘⇧G, ⌘E, ⌘⇧L, ⌘K | `C-b g G e D K` | lazygit, diff (lumen), yazi, lazydocker, k9s (read-only on production), in popups |
| ⌘J | `C-b J` | scratch terminal: its own tmux server (`scratch.conf`), keeps running |
| ⌘Y, ⌘U | `C-b Y`, `C-b u` | copy a token on screen (tmux-fingers), open a URL from the pane |
| ⌘A | `C-b a` | select the whole pane history (`y` copies) |
| ⌘⇧C | `C-b y` | copy the last command's output |
| ⌘⇧↑ / ⌘⇧↓ | `C-b P` / `N` | previous / next command (copy mode) |
| ⌘⇧H | `C-b ?` | the cheat sheet, runs the one you pick |
| F12 | | nested tmux (ssh to a server with tmux): every key goes to the inner one |

`C-b r` reloads `tmux.conf`.

## Copying

- **Mouse**: drag copies to the clipboard on release, and copy mode stays where
  it was scrolled to; double click copies a word, triple click a line. ⇧ + drag
  uses Ghostty's own selection (also in apps that take the mouse: nvim, Claude Code).
- **Keyboard** (copy mode: scroll up, or `C-b [`): `v` or `Space` starts a
  selection, `V` a line, `C-v` a rectangle; the wheel, `j`/`k`, `C-u`/`C-d`
  extend it; `y` or `Enter` copies; `q` / `Esc` leave.
- **A command's output**: p10k marks every prompt and output (OSC 133, see
  `.p10k.zsh`), and tmux keeps the marks.
  - ⌘⇧C copies the last command's output, without its prompt or trailing blank lines
  - ⌘⇧↑ goes to the command before the last, each ⌘⇧↑ / ⌘⇧↓ one more; then ⌘⇧C copies that one
  - in copy mode `[` / `]` jump between prompts the same way
  - only commands run since the shell started marking them: a pane opened before
    that says so instead of copying; `exec zsh` starts it

A trackpad scroll while dragging can end a mouse selection: tmux takes the
sideways scroll Ghostty sends for a button press. For long selections use the
keyboard.

## Files

| File | |
|---|---|
| `tmux.conf` | the config |
| `scratch.conf` | the ⌘J scratch server: only ⌘J bound, no plugins |
| `cheatsheet.sh` | ⌘⇧H: every ⌘ shortcut in fzf |
| `sessionizer.sh`, `sessions.sh`, `ssh-picker.sh`, `rename-window.sh`, `split.sh` | the popups and helpers above |
| `start-server.sh` | starts tmux with the `main` session when it isn't running: restores the last tmux-resurrect save, then preloads `sessions.local` (run by Ghostty's `tmux-start.sh` and VS Code's `tmux.sh`) |
| `preload-sessions.sh` | creates the sessions in `sessions.local` (run by `start-server.sh`, and `tmux-sessions` after editing it) |
| `sessions.local.example` | the format of `~/.config/tmux/sessions.local` (per machine, never committed) |

## Plugins

No TPM. The playbook clones them at pinned commits into `plugins/` (not in git;
`tmux_plugins` in devops-daily-driver `roles/terminal/defaults/main.yml`):
tmux-resurrect and tmux-continuum (sessions survive reboots: saved every 15 min,
restored when tmux starts from scratch; `C-b C-s` / `C-b C-r` by hand) and
tmux-fzf-url (⌘U). tmux-fingers (⌘Y) comes from Homebrew. `plugins-outdated`
(zsh) shows which pins master has moved past.
