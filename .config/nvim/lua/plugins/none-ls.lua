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

          -- Bash
          null_ls.builtins.diagnostics.shellcheck,
          null_ls.builtins.formatting.shfmt,

          -- Python
          null_ls.builtins.diagnostics.pylint,
          null_ls.builtins.formatting.black,

          -- Terraform / OpenTofu
          null_ls.builtins.diagnostics.terraform_validate,
          null_ls.builtins.formatting.terraform_fmt,

          -- Ansible
          null_ls.builtins.diagnostics.ansiblelint,

          -- Docker
          null_ls.builtins.diagnostics.hadolint,

          -- Go
          null_ls.builtins.formatting.gofmt,

          -- YAML
          null_ls.builtins.diagnostics.yamllint,

          -- General
          null_ls.builtins.formatting.prettier.with({
            filetypes = { "yaml", "json", "markdown" },
          }),
        },
      })

      -- Format on save
      vim.api.nvim_create_autocmd("BufWritePre", {
        callback = function()
          vim.lsp.buf.format({ async = false })
        end,
      })

      -- Format manually
      vim.keymap.set("n", "<leader>lf", function()
        vim.lsp.buf.format({ async = true })
      end, { desc = "Format file" })
    end,
  },
}
