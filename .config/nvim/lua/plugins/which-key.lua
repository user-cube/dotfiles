return {
  {
    -- Press <Space> and wait: a popup lists every keymap, from their `desc`
    "folke/which-key.nvim",
    event = "VeryLazy",
    opts = {
      preset = "modern",
      delay = 300,
      spec = {
        { "<leader>f", group = "Find" },
        { "<leader>l", group = "LSP" },
        { "<leader>r", group = "Refactor" },
        { "<leader>c", group = "Code" },
        { "g", group = "Go to" },
      },
    },
    keys = {
      {
        "<leader>?",
        function()
          require("which-key").show({ global = false })
        end,
        desc = "Keymaps for this buffer",
      },
    },
  },
}
