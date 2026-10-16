# Neovim Config

Personal Neovim configuration using [lazy.nvim](https://github.com/folke/lazy.nvim) as plugin manager.

## Plugins

- [monokai-pro.nvim](https://github.com/loctvl842/monokai-pro.nvim) — colorscheme
- [telescope.nvim](https://github.com/nvim-telescope/telescope.nvim) — fuzzy finder
- [nvim-treesitter](https://github.com/nvim-treesitter/nvim-treesitter) (`main` branch) — syntax highlighting and indent; parsers are compiled with `tree-sitter-cli`
- [neo-tree.nvim](https://github.com/nvim-neo-tree/neo-tree.nvim) — file explorer
- [lualine.nvim](https://github.com/nvim-lualine/lualine.nvim) — status line
- [mason.nvim](https://github.com/mason-org/mason.nvim) — LSP/tool installer
- [mason-lspconfig.nvim](https://github.com/mason-org/mason-lspconfig.nvim) — LSP auto-setup
- [telescope-ui-select.nvim](https://github.com/nvim-telescope/telescope-ui-select.nvim) — Telescope dropdown for `vim.ui.select`
- [none-ls.nvim](https://github.com/nvimtools/none-ls.nvim) — linters and formatters via LSP
- [which-key.nvim](https://github.com/folke/which-key.nvim) — press `Space` and wait for a menu of keymaps
- [SchemaStore.nvim](https://github.com/b0o/SchemaStore.nvim) — YAML schemas for yamlls, picked by file name
- [sops.nvim](https://github.com/RMTT/sops.nvim) — SOPS files open decrypted and `:w` re-encrypts them (needs `sops` and the key, e.g. `SOPS_AGE_KEY_FILE` from the repo's `.envrc`); undo history and swap are off for them, so the plaintext stays off disk

---

## Options

`lua/config/options.lua`, loaded before the plugins:

- relative line numbers (absolute on the cursor line), sign column always on
- 2-space indent; Go and Make use tabs (4 wide), Python 4 spaces
- tabs and trailing spaces shown (`»`, `·`)
- `y` / `p` use the macOS clipboard; mouse on
- search ignores case unless it has a capital; `<Esc>` clears the highlight
- persistent undo; files reopen where they were left; yanks flash

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
| `<Space>ff` | Find files (hidden ones included; gitignored ones not) |
| `<Space>fa` | Find all files, gitignored ones too |
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

## Filetypes (Helm, Ansible)

Helm templates and Ansible files are plain `.yaml`, so `init.lua` picks the
filetype from where the file lives; each gets its own LSP instead of yamlls:

| Filetype | Files | LSP |
|----------|-------|-----|
| `helm` | `templates/*` and `*.tpl` under a `Chart.yaml` | `helm_ls` |
| `yaml.helm-values` | `values*.yaml` next to a `Chart.yaml` | `helm_ls`, `yamlls` |
| `yaml.ansible` | `tasks/`, `handlers/`, `playbooks/`, `group_vars/`, `host_vars/`, `roles/*/{defaults,vars,meta}/`, or any yaml with a `hosts:` play | `ansiblels` |
| `yaml` | everything else | `yamlls` |

Check a file with `:set ft?`.

---

## Linters & Formatters

Installed via Homebrew (see [devops-daily-driver](https://github.com/ruicoelho/devops-daily-driver) playbook):

| Tool | Type | Language |
|------|------|----------|
| `stylua` | Formatter | Lua |
| `shellcheck` | Linter (via `bashls`) | Bash |
| `shfmt` | Formatter | Bash |
| `pylint` | Linter | Python |
| `black` | Formatter | Python |
| `ansible-lint` | Linter (via `ansiblels`) | Ansible |
| `hadolint` | Linter | Dockerfile |
| `golangci-lint` | Linter (on save) | Go |
| `yamllint` | Linter | YAML (not Ansible: ansible-lint runs its rules) |
| `terraform validate` | Linter | Terraform / OpenTofu |
| `terraform fmt` | Formatter | Terraform / OpenTofu |
| `prettier` | Formatter | YAML, JSON, Markdown |

Format on save. One formatter per file: the none-ls one when the filetype has
one, else the LSP's. Go is formatted by `gopls` (gofumpt) and gets its imports
organized (missing added, unused removed) on save.

`yamllint` follows the project's `.yamllint` when there is one; otherwise the
defaults without `document-start` and the `truthy` check on keys (GitHub's
`on:`), with line length as a warning at 120 and lists indented or not under
their key.

---

## LSP Servers

Installed automatically via Mason:

| Server | Language |
|--------|----------|
| `lua_ls` | Lua |
| `pyright` | Python |
| `terraformls` | Terraform |
| `tofu_ls` | OpenTofu (`tofu-ls` from Homebrew) |
| `ansiblels` | Ansible |
| `dockerls` | Dockerfile |
| `yamlls` | YAML |
| `helm_ls` | Helm templates and values |
| `bashls` | Bash |
| `gopls` | Go (gofumpt, staticcheck) |

### YAML Schemas (yamlls)

| Schema | Files |
|--------|-------|
| [SchemaStore](https://www.schemastore.org) catalog | by file name: GitHub Actions, Docker Compose, `kustomization.yaml`, `Chart.yaml`, `helmfile.yaml`, ... |
| Kubernetes (yamlls built-in) | `*.k8s.yaml`, `k8s/**`, `manifests/**` |

Another file can pick a schema with a comment on its first line:
`# yaml-language-server: $schema=<url>`.

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
