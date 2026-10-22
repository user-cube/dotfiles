return {
  {
    "loctvl842/monokai-pro.nvim",
    lazy = false,
    priority = 1000,
    config = function()
      require("monokai-pro").setup({
        transparent_background = false,
        terminal_colors = true,
        devicons = true,
        styles = {
          comment = { italic = true },
          keyword = { italic = true },
          type = { italic = true },
          storageclass = { italic = true },
          structure = { italic = true },
          parameter = { italic = true },
          annotation = { italic = true },
          tag_attribute = { italic = true },
        },
        filter = "pro", -- classic | octagon | pro | machine | ristretto | spectrum
        inc_search = "background", -- underline | background
        background_clear = { "telescope" },
        override = function(_scheme)
          return {
            -- strings
            ["@string"]              = { fg = "#e6db74" },
            -- functions
            ["@function"]            = { fg = "#a6e22e" },
            ["@function.call"]       = { fg = "#a6e22e" },
            ["@function.builtin"]    = { fg = "#a6e22e" },
            -- variables
            ["@variable"]            = { fg = "#f8f8f2" },
            -- properties / fields
            ["@variable.member"]     = { fg = "#78dce8" },
            ["@property"]            = { fg = "#78dce8" },
            -- keywords
            ["@keyword"]             = { fg = "#f92672", italic = true },
            -- numbers
            ["@number"]              = { fg = "#ae81ff" },
            -- comments
            ["@comment"]             = { fg = "#75715e", italic = true },
            -- operators
            ["@operator"]            = { fg = "#f8f8f2" },
            -- types
            ["@type"]                = { fg = "#66d9ef", italic = true },
          }
        end,
      })
      vim.cmd("colorscheme monokai-pro")
    end,
  },
}
