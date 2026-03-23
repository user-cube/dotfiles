return {
  {
    "nvim-neo-tree/neo-tree.nvim",
    branch = "v3.x",
    dependencies = {
      "nvim-lua/plenary.nvim",
      "nvim-tree/nvim-web-devicons",
      "MunifTanjim/nui.nvim",
    },
    config = function()
      require("neo-tree").setup({
        close_if_last_window = true,
        filesystem = {
          follow_current_file = { enabled = true },
          hide_dotfiles = false,
          hide_gitignored = false,
        },
        window = {
          width = 30,
        },
      })

      vim.keymap.set("n", "<leader>e", ":Neotree toggle<CR>", { desc = "Toggle file explorer" })
      vim.keymap.set("n", "<leader>o", ":Neotree focus<CR>", { desc = "Focus file explorer" })
    end,
  },
}
