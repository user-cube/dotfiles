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
- **Terminals count too.** Switching to a terminal of another folder takes the
  window there (its current directory with shell integration, else the one it
  was opened in, as inside tmux); the next editor change takes it back.
- **Strength and parts.** Choose how saturated the tint is and which parts of the
  window are tinted (for example only the title and status bars).
- **Dangerous places.** Folders, or paths inside them (`infra/hetzner/prod`, globs
  allowed), turn the title bar red and show `⚠ PROD` in the status bar while the
  active file is in them. ⌘S there asks first, once per place until VS Code
  restarts (only ⌘S: VS Code can't stop a save from the menu or auto save).
  With `hue.dangerReadonly` they open read-only instead, which stops those
  too: Hue adds them to the workspace's `files.readonlyInclude` (and takes out
  only what it added); **File: Set Active Editor Writeable in Session** opens
  one file up.
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
| Hue: Open Terminal in Folder (`⌘⌥T`) | Terminal in the active folder, colored tab |
| Hue: Go to Folder | Jump to a workspace folder's last file |
| Hue: Refresh | Apply the colors again |
| Hue: Save (asks first in a PROD place) | What ⌘S runs in a danger place |
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
| `hue.dangerReadonly` | `false` | Open the `hue.danger` places read-only |
| `hue.protectedBranches` | `["main", "master"]` | `[]` to turn the amber status bar off |
| `hue.enabled` | `true` | Off removes the colors from the workspace |

## Example

```json
{
  "folders": [{ "path": "../work/github/infra" }, { "path": "../work/github/docs" }],
  "settings": {
    "hue.folders": { "infra": "purple", "docs": "teal" },
    "hue.danger": ["infra/hetzner/prod"]
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

## Code

- `extension.js`: the VS Code side (settings, events, commands, status bar)
- `colors.js`: the rest, without VS Code: a folder's color, the palette of
  each part, the danger places, the `colorCustomizations` they make
- `colors.test.js`: `node --test .config/vscode/extensions/hue/` (CI runs it),
  including `extension.js` loaded with a fake `vscode`. A new file the
  extension needs goes into `install.sh`'s list too; a test checks it
- bump `version` in `package.json` with every change, so VS Code shows which
  one is installed: `install.sh` warns when the files changed and it didn't
