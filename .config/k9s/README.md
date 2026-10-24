# k9s

The Kubernetes TUI (⌘K opens it in a tmux popup). `K9S_CONFIG_DIR` points k9s
here instead of `~/Library/Application Support/k9s`; it is set in
`../zsh/kube-guard.zsh`, which the ⌘K popup sources too.

| File | |
|---|---|
| `config.yaml` | the settings: Monokai skin, 200 log lines with timestamps and wrapping, config reread when it changes (`ui.reactive`) |
| `skins/monokai.yaml` | from the k9s repo (v0.51.0), the Monokai of tmux, SketchyBar and Ghostty |
| `hotkeys.yaml` | Shift-<digit> to a view, below |
| `aliases.yaml` | `:dp`, `:sec`, `:crb`, … |
| `plugins.yaml` | keys on the selected resource, below |
| `clusters/` | per context: favorite namespaces, last view. Per machine, never committed |

## Hotkeys

| Key | View |
|---|---|
| Shift-1 | pods |
| Shift-3 | deployments |
| Shift-4 | Argo CD applications |
| Shift-5 | events |

Not Shift-2: k9s reads it as `@`, which a Portuguese keyboard doesn't send
(it types `"`), and it won't take `"` as a key. The digits alone pick a
favorite namespace.

## Plugins

| Key | Where | Runs |
|---|---|---|
| Ctrl-L | pods, deployments, statefulsets, daemonsets | stern: the logs of all its pods together |
| Ctrl-N | any resource | its YAML through `kubectl neat` (no managedFields, status, defaults), in `less` |
| Ctrl-B | containers | `kubectl debug`: a busybox container next to it, sharing its processes (a shell for distroless images) |
| Ctrl-F, Ctrl-G | Argo CD applications | refresh, hard refresh (the `argocd.argoproj.io/refresh` annotation) |
| Ctrl-Y | Argo CD applications | sync, after a confirmation (a patch of `operation.sync`) |

The Argo CD ones go through `kubectl`, so they need no `argocd login`. stern
and krew (`kubectl neat`) come from the playbook.

Keys that are already something else to a terminal don't work, and k9s then
loads none of the plugins: Ctrl-H (Backspace), Ctrl-I (Tab), Ctrl-J and Ctrl-M
(Enter).

## Production

`k9s` is a function (`kube-guard.zsh`): on a production context (`prod` / `prd`
in its name, `KUBE_PROD_REGEX`) it opens with `--readonly`, so nothing can be
deleted or edited, and the plugins that change the cluster (debug, the Argo CD
ones: `dangerous: true`) are hidden. `k9s --write` overrides it. A context
switched to inside k9s (`:ctx`) keeps the mode k9s started with.
