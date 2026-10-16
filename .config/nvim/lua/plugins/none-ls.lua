return {
  {
    "nvimtools/none-ls.nvim",
    dependencies = { "nvim-lua/plenary.nvim" },
    config = function()
      local null_ls = require("null-ls")

      null_ls.setup({
        sources = {
          -- Lua
          null_ls.builtins.formatting.stylua,

          -- Bash (diagnostics: shellcheck runs through bashls, see mason.lua;
          -- none-ls no longer ships a shellcheck builtin)
          null_ls.builtins.formatting.shfmt,

          -- Python
          null_ls.builtins.diagnostics.pylint,
          null_ls.builtins.formatting.black,

          -- Terraform / OpenTofu
          null_ls.builtins.diagnostics.terraform_validate,
          null_ls.builtins.formatting.terraform_fmt,

          -- Docker
          null_ls.builtins.diagnostics.hadolint,

          -- Go (formatting: gopls with gofumpt, see mason.lua; ansible-lint
          -- runs inside ansiblels)
          null_ls.builtins.diagnostics.golangci_lint,

          -- YAML. Ansible files skip it: ansible-lint runs the yamllint rules.
          -- Without a project .yamllint, the defaults minus the noisy rules
          -- (`---` on top, `on:` in GitHub workflows, line length, and lists
          -- indented or not under their key: both styles are common in k8s)
          null_ls.builtins.diagnostics.yamllint.with({
            runtime_condition = function(params)
              return params.ft ~= "yaml.ansible"
            end,
            extra_args = function(params)
              local own = vim.fs.find({ ".yamllint", ".yamllint.yaml", ".yamllint.yml" },
                { upward = true, path = vim.fs.dirname(params.bufname) })[1]
              if own then
                return {}
              end
              return { "-d", "{extends: default, rules: {document-start: disable, "
                .. "truthy: {check-keys: false}, line-length: {max: 120, level: warning}, "
                .. "indentation: {indent-sequences: whatever}}}" }
            end,
          }),

          -- General
          null_ls.builtins.formatting.prettier.with({
            filetypes = { "yaml", "json", "markdown" },
          }),
        },
      })

      -- One formatter per file: the none-ls one when there is one for the
      -- filetype (prettier, black, shfmt, ...), else the LSP's (gopls, yamlls)
      local function format(bufnr, async)
        local ft = vim.bo[bufnr].filetype
        local has_null = #require("null-ls.sources").get_available(ft, null_ls.methods.FORMATTING) > 0
        vim.lsp.buf.format({
          bufnr = bufnr,
          async = async,
          -- the 1s default silently skips slow starters (black's first run)
          timeout_ms = 5000,
          filter = function(client)
            return not has_null or client.name == "null-ls"
          end,
        })
      end

      -- gopls' organize imports: adds missing imports, drops unused ones
      local function organize_imports(bufnr)
        local client = vim.lsp.get_clients({ bufnr = bufnr, name = "gopls" })[1]
        if not client then
          return
        end
        local params = vim.lsp.util.make_range_params(0, client.offset_encoding)
        params.context = { only = { "source.organizeImports" }, diagnostics = {} }
        local res = client:request_sync("textDocument/codeAction", params, 1000, bufnr)
        for _, action in ipairs(res and res.result or {}) do
          if action.edit then
            vim.lsp.util.apply_workspace_edit(action.edit, client.offset_encoding)
          end
        end
      end

      -- Format on save
      vim.api.nvim_create_autocmd("BufWritePre", {
        callback = function(event)
          if vim.bo[event.buf].filetype == "go" then
            organize_imports(event.buf)
          end
          format(event.buf, false)
        end,
      })

      -- Format manually
      vim.keymap.set("n", "<leader>lf", function()
        format(0, true)
      end, { desc = "Format file" })
    end,
  },
}
