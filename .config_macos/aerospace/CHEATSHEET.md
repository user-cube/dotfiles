# AeroSpace cheat sheet

`⌥` = Option. Layout: laptop (left) · M27Q (1) (centre) · M27Q (2) (right).

## Workspaces

| Shortcut | Workspace | Monitor | Apps |
|---|---|---|---|
| `⌥ c` | 2-Code | centre | VS Code, Ghostty, Xcode, GitHub Desktop |
| `⌥ a` | 3-Claude | right | Claude, ChatGPT |
| `⌥ b` | 4-Browser | right | Firefox, Chrome, Safari |
| `⌥ m` | 5-Messages | laptop | Slack, WhatsApp, Discord |
| `⌥ n` | 6-Notes | laptop | Obsidian |
| `⌥ s` | 7-Music | laptop | Spotify |
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

System Settings, Finder, Archive Utility, Activity Monitor, Passwords, Bitwarden, Calculator.

## Service mode (`⌥ ⇧ ;`, then)

| Key | Action |
|---|---|
| `esc` | reload config, back to main |
| `r` | reset workspace layout |
| `f` | toggle floating / tiling |
| `backspace` | close all windows but current |
| `⌥ ⇧ h/j/k/l` | join with neighbour |

## SketchyBar (bottom bar)

Config in `~/.config/sketchybar/`. Reload with `sketchybar --reload`.

| Item | Click |
|---|---|
| Workspaces (with app icons) | go to that workspace |
| Spotify (only while playing) | play / pause |
| ☸ kube context/namespace (red if `prod`/`prd`, see `plugins/kube.sh`) | — |
| Colima running profiles | popup: start/stop each profile |
| GitHub PRs / GitLab MRs to review | open the review list |
| AWS `role@account` (default profile; red `expired` → `aws sso login`) | — |
| GCP project (red `expired` → `gcloud auth login`) | — |
| CPU, RAM (yellow > 70%, red > 85%), volume, battery | — |
| 📦 outdated brew packages | `brew upgrade` in a new Ghostty |

The macOS menu bar stays on top (status icons can't be mirrored on macOS 27).

## Keyboard (PT layout)

Avoid binding `⌥ 2/3/8/9` (`@ € [ ]`), `⌥ ⇧ 8/9` (`{ }`) and `⌥ x` (`« »`).

## macOS prerequisites

Applied by `devops-daily-driver` (`--tags macos`):

- Mission Control "Move left/right a space" (`ctrl ←/→`) disabled
- "Automatically rearrange Spaces based on most recent use" disabled

Manual:

- Keep a single macOS Desktop (Mission Control): AeroSpace only manages the current Space
- Firefox `about:config`: `full-screen-api.macos-native-full-screen = false`
- Hidden windows peek in the bottom-right corner: expected AeroSpace behaviour
