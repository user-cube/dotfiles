local opt = vim.opt

-- Line numbers: absolute on the cursor line, relative elsewhere (5j, 3dk, ...)
opt.number = true
opt.relativenumber = true
opt.signcolumn = "yes" -- diagnostics don't shift the text sideways
opt.cursorline = true
opt.scrolloff = 8

-- Indent: 2 spaces (YAML, HCL, Lua, JSON); Go, Make and Python below
opt.expandtab = true
opt.shiftwidth = 2
opt.tabstop = 2
opt.softtabstop = 2
opt.shiftround = true

-- Tabs and trailing spaces are visible (YAML breaks on a tab)
opt.list = true
opt.listchars = { tab = "» ", trail = "·", nbsp = "␣" }

-- Search: case-insensitive unless it has a capital; :s previews in a split
opt.ignorecase = true
opt.smartcase = true
opt.inccommand = "split"

-- y/p use the macOS clipboard (over ssh nvim falls back to OSC 52)
opt.clipboard = "unnamedplus"
opt.mouse = "a"

opt.splitright = true
opt.splitbelow = true
opt.undofile = true -- undo survives closing the file
opt.updatetime = 250 -- diagnostics float and swap writes sooner
opt.confirm = true -- :q on a changed buffer asks instead of failing

local group = vim.api.nvim_create_augroup("config_options", { clear = true })

vim.api.nvim_create_autocmd("FileType", {
  group = group,
  pattern = { "go", "make" },
  callback = function()
    vim.opt_local.expandtab = false
    vim.opt_local.tabstop = 4
    vim.opt_local.shiftwidth = 4
    vim.opt_local.list = false -- tabs are the norm here
  end,
})

vim.api.nvim_create_autocmd("FileType", {
  group = group,
  pattern = "python",
  callback = function()
    vim.opt_local.shiftwidth = 4
    vim.opt_local.tabstop = 4
    vim.opt_local.softtabstop = 4
  end,
})

-- Flash what was just yanked
vim.api.nvim_create_autocmd("TextYankPost", {
  group = group,
  callback = function()
    vim.hl.on_yank({ timeout = 150 })
  end,
})

-- Reopen a file where it was left
vim.api.nvim_create_autocmd("BufReadPost", {
  group = group,
  callback = function(event)
    local mark = vim.api.nvim_buf_get_mark(event.buf, '"')
    if mark[1] > 0 and mark[1] <= vim.api.nvim_buf_line_count(event.buf) then
      pcall(vim.api.nvim_win_set_cursor, 0, mark)
    end
  end,
})

-- Esc also clears the search highlight
vim.keymap.set("n", "<Esc>", "<cmd>nohlsearch<CR>", { desc = "Clear search highlight" })
