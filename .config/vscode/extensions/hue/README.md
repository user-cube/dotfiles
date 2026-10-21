# Hue

Every folder of a multi-root workspace gets its own color, and the VS Code window
follows the folder you are working in: title bar, activity bar, sidebar and status
bar are tinted with the color of the folder of the active file. Production places
and protected branches get a loud warning.

It only works in a `.code-workspace`: VS Code cannot color a window per folder, so
Hue writes the colors into the workspace's `workbench.colorCustomizations`, only
the keys it owns, and removes them when they stop applying. A repo's own
`.vscode/settings.json` is never touched.

## What it does

- **Folder colors.** `hue.folders` maps a folder name to a color. A folder with no
  color of its own gets a stable one from its name.
- **Strength and parts.** Choose how saturated the tint is and which parts of the
  window are tinted (for example only the title and status bars).
- **Dangerous places.** Folders, or paths inside them (`infra/hetzner/prod`, globs
  allowed), turn the title bar red and show `⚠ PROD` in the status bar while the
  active file is in them. The same for a Kubernetes context (`prod`, `*-prod`).
- **Protected branches.** The status bar turns amber on `main` / `master`
  (configurable).
- **Terminals.** `⌘⌥T` opens a terminal in the folder of the active file, its tab in
  the folder's color, red with a ⚠ in a dangerous place.
- **Go to Folder.** A quick pick of the workspace folders that opens the last file
  you had in the one you choose.

## Commands

| Command | What it does |
|---|---|
| Hue: Set Color | Pick the color of the active folder (or "Automatic") |
| Hue: Toggle Danger (PROD) for Folder | Mark or unmark the folder, the active file's directory, or an existing entry |
| Hue: Toggle Danger (PROD) for Current Kubernetes Context | Add or remove the current context in `hue.dangerContexts` |
| Hue: Open Terminal in Folder (`⌘⌥T`) | Terminal in the active folder, colored tab |
| Hue: Go to Folder | Jump to a workspace folder's last file |
| Hue: Refresh | Apply the colors again |
| Hue: Clear | Remove all of Hue's colors from the workspace and turn it off there |

## Settings

| Setting | Default | |
|---|---|---|
| `hue.folders` | `{}` | Folder name → color name or `#rrggbb` |
| `hue.colors` | the 8 below | Named colors: name → hue (0-360) or `#rrggbb`. Added to or replacing `red`, `orange`, `yellow`, `green`, `teal`, `blue`, `purple`, `pink` |
| `hue.auto` | `true` | A color from the folder's name when it has none |
| `hue.strength` | `normal` | `subtle`, `normal`, `strong` |
| `hue.parts` | all | `titleBar`, `activityBar`, `sideBar`, `statusBar` |
| `hue.danger` | `[]` | Folder names, or `folder/path` (`*` and `**` allowed) |
| `hue.dangerContexts` | `[]` | Kubernetes contexts: names or patterns (`*-prod`) |
| `hue.protectedBranches` | `["main", "master"]` | `[]` to turn the amber status bar off |
| `hue.enabled` | `true` | Off removes the colors from the workspace |

The Kubernetes context is read from the `current-context` line of `~/.kube/config`
(the file `kubectl` and `kontext` change). A `KUBECONFIG` exported in a terminal is
not visible to VS Code and is not used.

## Example

```json
{
  "folders": [{ "path": "../work/github/infra" }, { "path": "../work/github/docs" }],
  "settings": {
    "hue.folders": { "infra": "purple", "docs": "teal" },
    "hue.danger": ["infra/hetzner/prod"],
    "hue.dangerContexts": ["*-prod"]
  }
}
```

With the `ws` helper from the same dotfiles:

```sh
ws -n homelab infra:purple'!hetzner/prod' docs:teal
```

## Install

Not on the marketplace. `.config/vscode/extensions/hue/install.sh` packages it as a
`.vsix` and installs it with `code --install-extension`; `dotfiles-sync` runs it
when something changed. Reload the VS Code windows afterwards.
