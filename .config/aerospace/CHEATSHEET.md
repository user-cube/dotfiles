# AeroSpace cheat sheet

`⌥` = Option. Layout: laptop (left) · M27Q (1) (centre) · M27Q (2) (right).

## Workspaces

| Shortcut | Workspace | Monitor | Apps |
|---|---|---|---|
| `⌥ z` | 1-Meet | right | Zoom (main/meeting window; its popups float where you are) |
| `⌥ c` | 2-Code | centre | VS Code, Ghostty, Xcode, GitHub Desktop |
| `⌥ a` | 3-Claude | right | Claude, ChatGPT |
| `⌥ b` | 4-Browser | right | Firefox, Chrome, Safari |
| `⌥ m` | 5-Messages | laptop | Slack, WhatsApp, Discord |
| `⌥ n` | 6-Notes | laptop | Obsidian |
| `⌥ u` | 7-Music | laptop | Spotify |
| `⌥ p` | 8-K8s | right | Freelens, Headlamp |
| `⌥ o` | 9-Other | laptop | TradingView; free for anything else |
| `ctrl ←/→` | previous/next non-empty workspace on the focused monitor | | |
| `⌥ tab` | back to the previous workspace | | |

With fewer monitors, workspaces fall back to the remaining one (see `workspace-to-monitor-force-assignment`).

## Windows

| Shortcut | Action |
|---|---|
| `⌥ h/j/k/l` | focus ← ↓ ↑ → (crosses monitors) |
| `⌥ ⇧ h/j/k/l` | move window ← ↓ ↑ → (within the monitor) |
| `⌥ ⇧ <letter>` | send window to that workspace; focus follows |
| `⌥ ⇧ f` | toggle fullscreen (AeroSpace, not macOS native) |
| `⌥ -` / `⌥ =` | shrink / grow |
| `⌥ r` | resize mode (below) |
| `⌥ enter` | new Ghostty window |
| `⌥ i` | this cheat sheet, filling the monitor (`q` closes, `/` searches) |
| `⌥ f` | find any window (Raycast, AeroSpace extension): `⇧ enter` brings it here, `enter` jumps to it |

## Layout

| Shortcut | Action |
|---|---|
| `⌥ /` | tiles (press again: horizontal ↔ vertical) |
| `⌥ ,` | accordion (stacked) |

## Resize mode (`⌥ r`, then)

| Key | Action |
|---|---|
| `h` / `l` | narrower / wider |
| `k` / `j` | shorter / taller |
| `b` | balance sizes, back to main |
| `esc` / `enter` | back to main |

## Floating apps

System Settings, Finder, Archive Utility, Activity Monitor, Passwords, Bitwarden, Calculator, Zoom popups (share toolbar, chat, controls).

## Service mode (`⌥ ⇧ ;`, then)

| Key | Action |
|---|---|
| `esc` | reload config, back to main |
| `r` | reset workspace layout |
| `f` | toggle floating / tiling |
| `backspace` | close all windows but current |
| `⌥ ⇧ h/j/k/l` | join with neighbour |

## Ghostty + tmux (tabs, splits, sessions)

Every Ghostty window runs inside tmux. First window = persistent `main` session.
Sessions listed in `~/.config/tmux/sessions.local` (`name=path` per line, repeat a name for more windows; per machine) are
preloaded as sessions when tmux starts.
Sessions (tabs, splits, folders) are auto-saved every 15 min and restored after a
reboot; `ctrl b ctrl s` saves now, `ctrl b ctrl r` restores.

| Shortcut | Action |
|---|---|
| `⌘ t` / `⌘ w` | new tab / close pane (tab if last) |
| `⌘ ⇧ ←/→`, `⌘ 1…9` | previous/next tab, go to tab N |
| `⌘ d` / `⌘ ⇧ d` | split right / down (in an `ssh host` pane, the new pane opens the same ssh) |
| `⌘ ⌥ arrows` | move between splits |
| `⌘ ⇧ f` / `⌘ ⇧ e` | zoom split / equalize |
| `⌘ o` | open a repo under `~/work` as a session (fuzzy search; reuses it if open) |
| `⌘ ⇧ o` | open an ssh host (from the ssh config) in a new tab named after it |
| `⌘ ⇧ s` | session picker (enter switches, `x` kills) |
| `⌘ g` | lazygit for the current folder (popup; `q` closes) |
| `⌘ ⇧ g` | side-by-side diff of uncommitted changes (lumen popup; `q` closes) |
| `⌘ e` | yazi file manager in the current folder (popup; `q` closes) |
| `⌘ ⇧ l` | lazydocker: containers, logs, restarts (popup; `q` closes) |
| `⌘ k` | k9s on the current kube context (popup; `:q` or `ctrl c` closes) |
| `⌘ j` | scratch terminal (popup; keeps its history; `⌘ j` again closes) |
| `⌘ a` | select the whole pane history (`y`/`enter` copies, `esc` cancels) |
| `ctrl b d` | detach (session keeps running; reopen Ghostty to come back) |
| `F12` | tmux inside tmux (ssh to a server with tmux): keys go to the inner one, bar shows OFF; `F12` again returns |
| mouse drag | copies to clipboard; `⇧`+drag uses Ghostty's own selection |

## Neovim (`nvim`)

Leader is `Space`: press it and wait for a menu of every command (which-key);
`Space ?` lists the ones for the current file. Modes: `esc` back to normal,
`i` insert before the cursor, `a` after, `v` select, `V` select lines.

| Shortcut | Action |
|---|---|
| `Space f f` / `Space f g` | find file / search text in the project (Telescope) |
| `Space f b` / `Space f r` | open buffers / recent files |
| `Space e` / `Space o` | toggle / focus the file tree (Neo-tree) |
| `g d` / `g r` / `K` | go to definition / references / hover docs |
| `Space r n` / `Space c a` | rename symbol / code action |
| `Space d` / `Space l f` | show diagnostic / format file |
| `w` / `b` / `0` / `$` | next word / previous word / line start / line end |
| `g g` / `G` / `ctrl d` / `ctrl u` | top / bottom / half page down / up |
| `/text` then `n` / `N` | search, next / previous match |
| `d d` / `y y` / `p` | cut line / copy line / paste |
| `c i w` / `d i w` | change / delete the word under the cursor |
| `u` / `ctrl r` / `.` | undo / redo / repeat the last change |
| `:w` / `:q` / `:wq` / `:q!` | save / quit / save and quit / quit without saving |

## SketchyBar (bottom bar)

Config in `~/.config/sketchybar/`. Reload with `sketchybar --reload`.

| Item | Click |
|---|---|
| Workspaces (with app icons) | go to that workspace |
| Spotify (dim with ▶ while paused, hidden when closed) | play / pause |
| Slack / Discord badges (red count = mentions/DMs, yellow • = unread; hidden when closed) | open the app |
| ☸ kube context/namespace (red if `prod`/`prd`, see `plugins/kube.sh`) | — |
| Colima running profiles | popup: start/stop each profile |
| GitHub PRs / GitLab MRs to review | open the review list |
| AWS: one entry per profile (`~/.config/sketchybar/aws_profiles`, one per line, per machine; else all profiles). Red `expired` → `aws sso login --profile <name>` | — |
| GCP project (red `expired` → `gcloud auth login`) | — |
| CPU, RAM (yellow > 70%, red > 85%) | — |
| Weather (wttr.in, location from IP) | open forecast |
| 📦 outdated brew packages | `brew upgrade` in a new Ghostty |

On narrow displays (< 1800 pt, e.g. the 14" laptop) only the essentials show: workspaces,
focused app, Slack/Discord, kube, AWS/GCP. Spotify, weather, CPU, RAM, Colima, GitHub,
GitLab and brew only appear on the wide monitors (`LOW_PRIORITY` in `sketchybarrc`).

The macOS menu bar stays on top (status icons can't be mirrored on macOS 27). Battery, sound and clock live there.

## Keyboard (PT layout)

Avoid binding `⌥ 2/3/8/9` (`@ € [ ]`), `⌥ ⇧ 8/9` (`{ }`) and `⌥ x` (`« »`).

## Permissions (System Settings → Privacy & Security)

| Where | App | Why |
|---|---|---|
| Accessibility | AeroSpace | move and tile windows |
| Accessibility | sketchybar (`/opt/homebrew/Cellar/sketchybar/<version>/bin/sketchybar`) | Slack/Discord Dock badges |
| Automation → System Events | sketchybar (prompted) | Dock badges |
| Automation → Spotify | sketchybar (prompted) | play/pause |

Guided setup: `ansible-playbook daily-driver.yml --tags permissions`.
After `brew upgrade sketchybar` the badges turn grey: the grant is tied to the old
Cellar path — remove it and run the permissions tag again.

## macOS prerequisites

Applied by `devops-daily-driver` (`--tags macos`):

- Mission Control "Move left/right a space" (`ctrl ←/→`) disabled
- "Automatically rearrange Spaces based on most recent use" disabled

Manual:

- Keep a single macOS Desktop (Mission Control): AeroSpace only manages the current Space
- Firefox `about:config`: `full-screen-api.macos-native-full-screen = false`
- Hidden windows peek in the bottom-right corner: expected AeroSpace behaviour
