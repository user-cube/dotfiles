return {
  {
    "mason-org/mason.nvim",
    config = function()
      require("mason").setup({
        ui = {
          icons = {
            package_installed = "✓",
            package_pending = "➜",
            package_uninstalled = "✗",
          },
        },
      })
    end,
  },
  {
    "mason-org/mason-lspconfig.nvim",
    dependencies = {
      "mason-org/mason.nvim",
      "neovim/nvim-lspconfig",
      "b0o/SchemaStore.nvim", -- yaml schemas (GitHub Actions, compose, kustomize, ...)
    },
    config = function()
      -- tofu_ls is enabled too, but not in Mason: it comes from Homebrew (tofu-ls)
      local servers = {
        "lua_ls",
        "pyright",
        "terraformls",
        "ansiblels",
        "dockerls",
        "yamlls",
        "bashls",
        "gopls",
        "helm_ls",
      }

      require("mason-lspconfig").setup({ ensure_installed = servers })

      vim.lsp.config("terraformls", {
        filetypes = { "terraform", "terraform-vars" },
      })

      vim.lsp.config("tofu_ls", {
        cmd = { "tofu-ls", "serve" },
        filetypes = { "opentofu", "opentofu-vars" },
        root_markers = { ".terraform", ".git" },
      })

      vim.lsp.config("lua_ls", {
        settings = {
          Lua = {
            runtime = { version = "LuaJIT" },
            workspace = {
              checkThirdParty = false,
              library = vim.api.nvim_get_runtime_file("", true),
            },
            diagnostics = {
              globals = { "vim" },
              disable = { "undefined-field", "unused-local" },
            },
            telemetry = { enable = false },
          },
        },
      })

      -- yamlls: plain yaml only. Helm templates (ft helm) go to helm_ls, Ansible
      -- (ft yaml.ansible) to ansiblels, see init.lua
      vim.lsp.config("yamlls", {
        settings = {
          yaml = {
            -- SchemaStore.nvim picks the schema from the file name; its catalog
            -- has no Kubernetes entry, so manifests get yamlls' built-in one
            schemaStore = { enable = false, url = "" },
            schemas = vim.tbl_extend("force", require("schemastore").yaml.schemas(), {
              kubernetes = {
                "*.k8s.yaml", "*.k8s.yml",
                "k8s/**/*.yaml", "k8s/**/*.yml",
                "manifests/**/*.yaml", "manifests/**/*.yml",
              },
            }),
            validate = true,
            completion = true,
            hover = true,
          },
        },
      })

      vim.lsp.config("gopls", {
        settings = {
          gopls = {
            gofumpt = true, -- stricter gofmt, built into gopls
            staticcheck = true,
            analyses = { unusedparams = true },
          },
        },
      })

      vim.lsp.enable(vim.list_extend({ "tofu_ls" }, servers))

      -- Keymaps LSP
      vim.api.nvim_create_autocmd("LspAttach", {
        callback = function(event)
          local opts = { buffer = event.buf }
          vim.keymap.set("n", "gd", vim.lsp.buf.definition, vim.tbl_extend("force", opts, { desc = "Go to definition" }))
          vim.keymap.set("n", "gr", vim.lsp.buf.references, vim.tbl_extend("force", opts, { desc = "References" }))
          vim.keymap.set("n", "K", vim.lsp.buf.hover, vim.tbl_extend("force", opts, { desc = "Hover docs" }))
          vim.keymap.set("n", "<leader>rn", vim.lsp.buf.rename, vim.tbl_extend("force", opts, { desc = "Rename" }))
          vim.keymap.set("n", "<leader>ca", vim.lsp.buf.code_action, vim.tbl_extend("force", opts, { desc = "Code action" }))
          vim.keymap.set("n", "<leader>d", vim.diagnostic.open_float, vim.tbl_extend("force", opts, { desc = "Show diagnostic" }))
        end,
      })
    end,
  },
}
