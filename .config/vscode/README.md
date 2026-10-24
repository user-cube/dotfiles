# VS Code

Used through `ws` (`../zsh/vscode-workspaces.zsh`): workspaces live in
`~/workspaces/*.code-workspace` (per machine, never committed).

```sh
ws                                   # pick a workspace with fzf
ws NAME                              # open NAME.code-workspace
ws -n NAME -t terraform DIR:blue ... # create one from folders, then open it
```

- `templates/`: settings and recommended extensions `ws -n ... -t TPL` merges
  into a new workspace: `ansible`, `golang`, `helm`, `opentofu`, `terraform`;
  repeat `-t` to combine
- `extensions/hue/`: Hue, a small extension outside the marketplace, installed
  by `install.sh` when it changes ([its README](extensions/hue/README.md))
  - the window takes the color of the folder you work in (`DIR:COLOR`, or one
    from the folder's name)
  - production: `DIR!` (all of it) or `DIR!hetzner/prod` (that path only)
    gives a red title bar, a PROD marker and a red terminal tab
  - the status bar turns amber on `main` / `master`
