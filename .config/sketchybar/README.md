# SketchyBar

The status bar, at the bottom (the macOS menu bar stays on top). `sketchybarrc`
builds it; each item runs a script in `plugins/`. Reload with
`sketchybar --reload` (`dotfiles-sync` does it).

| Item | Script | Shows |
|---|---|---|
| workspaces | `aerospace.sh` | AeroSpace workspaces with their apps' icons; empty ones hidden |
| focused app | `front_app.sh` | |
| Spotify | `spotify.sh` | playing / paused; click toggles |
| Slack, Discord | `dock_badge.sh` | the Dock badge: red count, yellow dot = unread |
| kube | `kube.sh` | context / namespace, red when it looks like prod |
| checks | `check/` (Go) | connectivity checks: passed/total, click for each |
| Colima | `colima.sh`, `colima_toggle.sh` | running profiles; the popup starts / stops each |
| GitHub, GitLab | `github.sh`, `gitlab.sh` | PRs / MRs waiting for my review; hidden without a glab login |
| AWS | `aws.sh`, `aws_profiles.sh` | each profile's session, red "expired" when it needs `aws sso login` |
| GCP | `gcp.sh` | active project, red when the login expired |
| CPU, RAM | `cpu.sh`, `ram.sh` | |
| weather | `weather.sh` | wttr.in; click opens the forecast |
| brew | `brew.sh`, `brew_upgrade.sh` | outdated packages (hourly); click upgrades in a new Ghostty window |

`helpers.sh` has what the plugins share (`run`: tools that wait on their own
subprocesses need SIGCHLD reset under SketchyBar); `colors.sh` the Monokai
palette; `display.sh` rebuilds the bar when monitors change.

## Per machine (never committed)

- `~/.config/sketchybar/aws_profiles`: the AWS profiles to show, one per line
  (default: every configured profile)
- `~/.config/sketchybar/checks`: the checks, `name | command`, green when the
  command exits 0 within 10 s (default: every kube context); see `checks.example`

## Not in git

- `plugins/icon_map.sh` (app name → icon): `fetch-icon-map.sh` downloads it from
  the sketchybar-app-font release matching the installed font, so they never drift
- `plugins/check/check`: built from `plugins/check/` by `install.sh`

The Slack/Discord badges need Accessibility for the sketchybar binary, lost on
each `brew upgrade sketchybar`: the playbook's `--tags permissions` walks through it.
