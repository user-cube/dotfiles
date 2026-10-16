-- nvim-treesitter `main` branch: it only installs parsers (compiled with the
-- tree-sitter CLI, `brew install tree-sitter-cli`); highlighting and indent
-- are switched on per buffer below
local parsers = {
  "lua", "vim", "vimdoc", "query",
  "python", "bash", "go", "gomod", "gosum", "gowork",
  "terraform", "hcl", "helm", "yaml", "json", "toml", "dockerfile",
  "markdown", "markdown_inline", "make", "gitignore", "diff", "regex",
}

return {
  {
    "nvim-treesitter/nvim-treesitter",
    branch = "main",
    lazy = false,
    build = ":TSUpdate",
    config = function()
      vim.treesitter.language.register("terraform", { "opentofu", "opentofu-vars" })
      vim.treesitter.language.register("yaml", { "yaml.ansible", "yaml.helm-values" })

      require("nvim-treesitter").install(parsers)

      vim.api.nvim_create_autocmd("FileType", {
        callback = function(event)
          if not pcall(vim.treesitter.start, event.buf) then
            return -- no parser for this filetype
          end
          local lang = vim.treesitter.language.get_lang(vim.bo[event.buf].filetype)
          if lang and vim.treesitter.query.get(lang, "indents") then
            vim.bo[event.buf].indentexpr = "v:lua.require'nvim-treesitter'.indentexpr()"
          end
        end,
      })
    end,
  },
}
