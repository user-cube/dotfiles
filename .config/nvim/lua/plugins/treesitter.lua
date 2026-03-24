return {
  {
    "nvim-treesitter/nvim-treesitter",
    build = ":TSUpdate",
    init = function()
      vim.opt.runtimepath:append(vim.fn.stdpath("data") .. "/site")
      vim.treesitter.language.register("terraform", "opentofu")
      vim.treesitter.language.register("terraform", "opentofu-vars")
    end,
    opts = {
      ensure_installed = {
        "lua", "vim", "vimdoc",
        "python", "bash", "go",
        "terraform", "hcl",
        "yaml", "json", "dockerfile",
      },
      auto_install = true,
      highlight = { enable = true },
      indent = { enable = true },
    },
  },
}
