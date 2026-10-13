# AeroSpace cheat sheet

`⌥` = Option. Layout: laptop (left) · M27Q (1) (centre) · M27Q (2) (right).

## Workspaces

| Shortcut | Workspace | Monitor | Apps |
|---|---|---|---|
| `⌥ c` | 2-Code | centre | VS Code, Ghostty |
| `⌥ a` | 3-Claude | right | Claude |
| `⌥ b` | 4-Browser | right | Firefox |
| `⌥ m` | 5-Messages | laptop | Slack, WhatsApp, Discord |
| `⌥ n` | 6-Notes | laptop | Obsidian |
| `⌥ s` | 7-Music | laptop | Spotify |
| `⌥ q` / `⌥ w` | 8-Q / 9-W | laptop / right | free |
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
| `⌥ enter` | new Ghostty window |

## Layout

| Shortcut | Action |
|---|---|
| `⌥ /` | tiles (press again: horizontal ↔ vertical) |
| `⌥ ,` | accordion (stacked) |

## Service mode (`⌥ ⇧ ;`, then)

| Key | Action |
|---|---|
| `esc` | reload config, back to main |
| `r` | reset workspace layout |
| `f` | toggle floating / tiling |
| `backspace` | close all windows but current |
| `⌥ ⇧ h/j/k/l` | join with neighbour |

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
