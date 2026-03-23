> Format on save is enabled automatically.

---

## LSP Servers

Installed automatically via Mason:

| Server | Language |
|--------|----------|
| `lua_ls` | Lua |
| `pyright` | Python |
| `terraformls` | Terraform |
| `ansiblels` | Ansible |
| `dockerls` | Dockerfile |
| `yamlls` | YAML (Kubernetes, Helm, Docker Compose, GitHub Actions) |
| `bashls` | Bash |
| `gopls` | Go |

### YAML Schemas (yamlls)

| Schema | Files |
|--------|-------|
| GitHub Actions | `.github/workflows/*.yml` |
| Kubernetes | `*.k8s.yaml`, `templates/*.yaml` |
| Helm Chart | `Chart.yaml` |
| Helmfile | `helmfile.yaml` |
| Docker Compose | `docker-compose*.yml` |

---

## Commands

### Mason

| Command | Action |
|---------|--------|
| `:Mason` | Open Mason UI to install/manage tools |
| `:MasonInstall <name>` | Install a specific tool |
| `:MasonUninstall <name>` | Uninstall a specific tool |

### Lazy

| Command | Action |
|---------|--------|
| `:Lazy` | Open Lazy UI |
| `:Lazy sync` | Install/update/clean plugins |
| `:Lazy update` | Update all plugins |
| `:Lazy reload <plugin>` | Reload a specific plugin |

### Treesitter

| Command | Action |
|---------|--------|
| `:TSInstall <lang>` | Install a language parser |
| `:TSUpdate` | Update all parsers |

### General Vim

| Command | Action |
|---------|--------|
| `:e!` | Discard changes and reload file from disk |
| `:w` | Save file |
| `:q` | Quit |
| `:wq` | Save and quit |
