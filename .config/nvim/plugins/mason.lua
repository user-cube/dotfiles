            diagnostics = {
              globals = { "vim" },
              disable = { "undefined-field", "unused-local" },
            },
            telemetry = { enable = false },
          },
        },
      })

      vim.lsp.config("yamlls", {
        settings = {
          yaml = {
            schemaStore = {
              enable = false,
              url = "",
            },
            schemas = {
              ["https://json.schemastore.org/github-workflow.json"] = "/.github/workflows/*.yml",
              ["https://raw.githubusercontent.com/instrumenta/kubernetes-json-schema/master/v1.18.0-standalone-strict/all.json"] = {
                "/*.k8s.yaml",
                "/*.k8s.yml",
                "/templates/*.yaml",
                "/templates/*.yml",
              },
              ["https://json.schemastore.org/helmfile.json"] = "helmfile.yaml",
              ["https://json.schemastore.org/chart.json"] = "Chart.yaml",
              ["https://json.schemastore.org/docker-compose.json"] = {
                "docker-compose*.yml",
                "docker-compose*.yaml",
              },
            },
            validate = true,
            completion = true,
            hover = true,
          },
        },
      })

      vim.lsp.enable({
        "lua_ls",
        "pyright",
        "terraformls",
        "tofu_ls",
        "ansiblels",
        "dockerls",
        "yamlls",
        "bashls",
        "gopls",
      })

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
