# Neovim Config

Personal Neovim configuration using [lazy.nvim](https://github.com/folke/lazy.nvim) as plugin manager.

## Plugins

- [monokai-pro.nvim](https://github.com/loctvl842/monokai-pro.nvim) — colorscheme
- [telescope.nvim](https://github.com/nvim-telescope/telescope.nvim) — fuzzy finder
- [nvim-treesitter](https://github.com/nvim-treesitter/nvim-treesitter) — syntax highlighting
- [neo-tree.nvim](https://github.com/nvim-neo-tree/neo-tree.nvim) — file explorer
- [lualine.nvim](https://github.com/nvim-lualine/lualine.nvim) — status line
- [mason.nvim](https://github.com/mason-org/mason.nvim) — LSP/tool installer
- [mason-lspconfig.nvim](https://github.com/mason-org/mason-lspconfig.nvim) — LSP auto-setup
- [telescope-ui-select.nvim](https://github.com/nvim-telescope/telescope-ui-select.nvim) — Telescope dropdown for `vim.ui.select`
- [none-ls.nvim](https://github.com/nvimtools/none-ls.nvim) — linters and formatters via LSP

---

## Keymaps

> Leader key is `Space`

### File Explorer (Neo-tree)

| Keymap | Action |
|--------|--------|
| `<Space>e` | Toggle file explorer |
| `<Space>o` | Focus file explorer |

#### Neo-tree — Navigation

| Keymap | Action |
|--------|--------|
| `<cr>` | Open file / expand directory |
| `<bs>` | Navigate up to parent directory |
| `.` | Set current directory as root |
| `C` | Close node or parent |
| `z` | Close all nodes |
| `<` | Switch to previous source |
| `>` | Switch to next source |

#### Neo-tree — File Operations

| Keymap | Action |
|--------|--------|
| `a` | Create file (append `/` for directory) |
| `A` | Create directory |
| `d` | Delete |
| `r` | Rename |
| `c` | Copy |
| `m` | Move |
| `y` | Copy to clipboard |
| `x` | Cut to clipboard |
| `p` | Paste from clipboard |

#### Neo-tree — Display

| Keymap | Action |
|--------|--------|
| `H` | Toggle hidden files |
| `R` | Refresh |
| `i` | Show file details |
| `P` | Toggle preview |
| `/` | Fuzzy find files |
| `f` | Filter |
| `<C-x>` | Clear filter |

#### Neo-tree — Window

| Keymap | Action |
|--------|--------|
| `S` | Open in horizontal split |
| `s` | Open in vertical split |
| `t` | Open in new tab |

#### Neo-tree — Git

| Keymap | Action |
|--------|--------|
| `[g` | Previous git-modified file |
| `]g` | Next git-modified file |

#### Neo-tree — Sorting

| Keymap | Action |
|--------|--------|
| `oc` | Sort by created date |
| `om` | Sort by modified date |
| `on` | Sort by name |
| `os` | Sort by size |
| `og` | Sort by git status |

> Press `?` inside Neo-tree to show the full help menu.

### Telescope

| Keymap | Action |
|--------|--------|
| `<Space>ff` | Find files |
| `<Space>fg` | Live grep (search in files) |
| `<Space>fb` | List open buffers |
| `<Space>fh` | Search help tags |
| `<Space>fr` | Recent files |
| `<Space>fs` | Grep string under cursor |

#### Telescope — Inside picker

| Keymap | Action |
|--------|--------|
| `<C-n>` / `<C-p>` | Next / previous result |
| `<C-x>` | Open in horizontal split |
| `<C-v>` | Open in vertical split |
| `<C-t>` | Open in new tab |
| `<C-u>` | Scroll preview up |
| `<C-d>` | Scroll preview down |
| `<C-q>` | Send results to quickfix list |
| `<esc>` | Close picker |

### LSP

| Keymap | Action |
|--------|--------|
| `gd` | Go to definition |
| `gr` | Show references |
| `K` | Hover documentation |
| `<Space>rn` | Rename symbol |
| `<Space>ca` | Code actions |
| `<Space>d` | Show diagnostic |
| `<Space>lf` | Format file |

---

## Linters & Formatters

Installed via Homebrew (see [devops-daily-driver](https://github.com/ruicoelho/devops-daily-driver) playbook):

| Tool | Type | Language |
|------|------|----------|
| `stylua` | Formatter | Lua |
| `shellcheck` | Linter | Bash |
| `shfmt` | Formatter | Bash |
| `pylint` | Linter | Python |
| `black` | Formatter | Python |
| `ansible-lint` | Linter | Ansible |
| `hadolint` | Linter | Dockerfile |
| `gofmt` | Formatter | Go |
| `yamllint` | Linter | YAML |
| `tflint` | Linter | Terraform / OpenTofu |
| `terraform_fmt` | Formatter | Terraform / OpenTofu |
| `prettier` | Formatter | YAML, JSON, Markdown |

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
