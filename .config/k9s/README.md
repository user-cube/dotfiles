# k9s

The Kubernetes TUI (⌘K opens it in a tmux popup). `K9S_CONFIG_DIR` points k9s
here instead of `~/Library/Application Support/k9s`; it is set in
`../zsh/kube-guard.zsh`, which the ⌘K popup sources too.

| File | |
|---|---|
| `config.yaml` | the settings (mostly k9s defaults) |
| `aliases.yaml` | `:dp`, `:sec`, `:crb`, … |
| `plugins.yaml` | keys on the selected resource, below |
| `clusters/` | per context: favorite namespaces, last view. Per machine, never committed |

## Plugins

| Key | Where | Runs |
|---|---|---|
| Ctrl-L | pods, deployments, statefulsets, daemonsets | stern: the logs of all its pods together |
| Ctrl-N | any resource | its YAML through `kubectl neat` (no managedFields, status, defaults), in `less` |
| Ctrl-I | containers | dive on the container's image: its layers (pulls through Docker / Colima) |

stern, krew (`kubectl neat`) and dive come from the playbook.

## Production

`k9s` is a function (`kube-guard.zsh`): on a production context (`prod` / `prd`
in its name, `KUBE_PROD_REGEX`) it opens with `--readonly`, so nothing can be
deleted or edited; `k9s --write` overrides it. A context switched to inside k9s
(`:ctx`) keeps the mode k9s started with.
