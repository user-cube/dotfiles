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
| `⌥ u` | 7-Music | laptop | Spotify (`⌥ s` is grabbed by another app on the company Mac) |
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
Repos listed in `~/.config/tmux/sessions.local` (one path per line, per machine) are
preloaded as sessions when tmux starts.

| Shortcut | Action |
|---|---|
| `⌘ t` / `⌘ w` | new tab / close pane (tab if last) |
| `⌘ ⇧ ←/→`, `⌘ 1…9` | previous/next tab, go to tab N |
| `⌘ d` / `⌘ ⇧ d` | split right / down |
| `⌘ ⌥ arrows` | move between splits |
| `⌘ ⇧ f` / `⌘ ⇧ e` | zoom split / equalize |
| `⌘ o` | open a repo under `~/work` as a session (fuzzy search; reuses it if open) |
| `⌘ ⇧ s` | session picker (enter switches, `x` kills) |
| `ctrl b d` | detach (session keeps running; reopen Ghostty to come back) |
| mouse drag | copies to clipboard; `⇧`+drag uses Ghostty's own selection |

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
