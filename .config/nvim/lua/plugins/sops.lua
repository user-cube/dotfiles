-- SOPS files open decrypted and are re-encrypted on :w
return {
  {
    "RMTT/sops.nvim",
    event = { "BufReadPre", "BufNewFile" },
    opts = {
      pattern = { "*.yaml", "*.yml", "*.json", "*.env", "*.ini" },
    },
    config = function(_, opts)
      require("sops").setup(opts)

      -- The decrypted text must not leak to disk through undo history or swap
      vim.api.nvim_create_autocmd("BufReadPost", {
        group = vim.api.nvim_create_augroup("sops_no_leak", { clear = true }),
        callback = function(event)
          vim.schedule(function()
            if vim.api.nvim_buf_is_valid(event.buf) and require("sops").status(event.buf) ~= "" then
              vim.bo[event.buf].undofile = false
              vim.bo[event.buf].swapfile = false
            end
          end)
        end,
      })
    end,
  },
}
