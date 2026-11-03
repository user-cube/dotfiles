# sofka

> The guide: [Kubernetes](https://github.com/user-cube/dotfiles/wiki/Kubernetes) in the wiki. This README is the reference.

A Kubernetes TUI in Rust, next to k9s (⌘K opens it in a Rex / tmux popup;
`:q` or ctrl+c closes). The popup opens on the kube context of the pane ⌘K
was pressed in, a `ktx -s` shell's own too (`kube-guard.zsh`). In Rex it is
one per session: hidden with ⌘K, it comes back on the context it had. Its config started from
`sofka import k9s` (`../k9s`) and keeps only what sofka doesn't do itself.

| File | |
|---|---|
| `conf.d/00-k9s.yaml` | Monokai skin, 200 log lines, the Shift-<digit> views, the plugins and the production guardrails below |
| `config.toml` | not committed: sofka runs on its defaults plus `conf.d/` |

Not `sofka import k9s --force` again: it rewrites `conf.d/00-k9s.yaml` with
the plugins removed below back in. `sofka info --offline` lists the config
files read and any validation warnings.

## Bookmarks

| Key | View |
|---|---|
| Shift-1 | pods |
| Shift-3 | deployments |
| Shift-4 | Argo CD applications |
| Shift-5 | events |

The same keys as in k9s (`!`, `#`, `$`, `%`, so they work on a Portuguese
keyboard). The digits alone pick a favorite namespace.

## Plugins

| Key | Where | Runs |
|---|---|---|
| Ctrl-N | any resource | its YAML through `kubectl neat` (no managedFields, status, defaults), in `less` |
| Ctrl-U | pods, deployments, statefulsets, daemonsets | popeye's report on its namespace |

popeye and krew (`kubectl neat`) come from the playbook. `:sanitize`
(delete finished pods) ships with sofka.

## Built in, so not plugins

| k9s plugin | sofka |
|---|---|
| stern (Ctrl-L) | `l` on a workload: the logs of all its pods, following the selector |
| krr (Ctrl-T) | `:rightsize` on a workload: P50/P95/P99 CPU and memory, suggested requests and a patch to copy (`c`); finds the cluster's Prometheus itself, through the API server |
| Argo CD refresh / hard refresh / sync (Ctrl-F, Ctrl-G, Ctrl-Y) | `t` on an Application: refresh, hard refresh, sync, sync with prune, suspend / resume; `:argo` for sync and health |

dive and debug (Ctrl-O, Ctrl-B) are k9s only: sofka plugins can't target
containers. Ctrl-F is sofka's page down, so a plugin can't use it.

The k9s aliases (`:dp`, `:sec`, `:jo`, `:cr`, `:crb`, `:ro`, `:rb`, `:np`)
are sofka's own, so there are none here.

## Production

`sofka` is a function (`kube-guard.zsh`), like `k9s`: on a production context
(`prod` / `prd` in its name, `KUBE_PROD_REGEX`) it opens with `--readonly`;
`sofka --write` overrides it. Its subcommands (`info`, `import`, `plugin`, …)
pass as they are.

A context switched to inside sofka (`:ctx`) keeps the mode it started with, so
the `guardrails` hold on any `*prod*` / `*prd*` context however it was reached:
delete, force-delete and drain are blocked, and a shell asks for the context
name typed out.
