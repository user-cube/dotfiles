-- Rex: the same ⌘ shortcuts as Ghostty + tmux (tmux/README.md), bound to Rex's
-- own tabs, panes and sessions. Popups are floating layers that close when
-- their program exits. `rex config reload` after a change, `rex config check`
-- to validate, ⌘⇧H (the command palette) lists every action and its key.

local SHELL = "com.superlogical.terminal.shell"
local TERMINAL = "com.superlogical.terminal"
local HOME = os.getenv("HOME")

-- The focused pane's folder: the OSC 7 the shell reported (a file:// URL),
-- else where its foreground process is
local function pane_dir(ctx)
  local r = rex.block.call(TERMINAL, "pwd", { block_id = ctx.block_id })
  if r and r.pwd then
    local path = r.pwd:gsub("^file://[^/]*", "")
    return (path:gsub("%%(%x%x)", function(h) return string.char(tonumber(h, 16)) end))
  end
  local p = rex.block.call(TERMINAL, "process", { block_id = ctx.block_id })
  return p and p.foreground and p.foreground.cwd or HOME
end

local function check(_, err)
  if err ~= nil then error(err, 0) end
end

local function bounds(size)
  local m = (1 - size) / 2
  return { m, m, size, size }
end

-- A floating layer running argv (a shell when nil) in dir
local function popup(argv, size, dir, label)
  check(rex.session.new_layer{
    bounds = bounds(size),
    focus = true,
    layout = rex.layout.block{
      flavor = SHELL,
      label = label,
      options = { command = argv, cwd = dir },
    },
  })
end

-- A popup that stays: the first press opens it, the same key inside hides it
-- (the block is detached and keeps running), the next press brings it back.
-- Found by its block label, so one per label and session. Quitting the
-- program closes it for good.
local function toggle_popup(ctx, label, argv, size, dir)
  local list = rex.session.list_blocks{}
  local b
  for _, x in ipairs(list and list.blocks or {}) do
    if x.label == label then b = x end
  end
  if not b then
    return popup(argv, size, dir, label)
  end
  if b.block_id == ctx.block_id then
    return check(rex.session.detach_block{ block_id = b.block_id })
  end
  if b.placement ~= "placed" then
    -- a detached block can't float straight away: place it beside this pane first
    check(rex.session.move_block{
      block_id = b.block_id, anchor_block_id = ctx.block_id,
      direction = "horizontal", side = "after",
    })
    check(rex.session.float_block{ block_id = b.block_id, bounds = bounds(size) })
  end
  check(rex.session.focus_block{ block_id = b.block_id })
end

local function git_root(dir)
  local r = rex.exec({ "git", "rev-parse", "--show-toplevel" }, { cwd = dir })
  return r and r.ok and r.stdout:gsub("%s+$", "") or dir
end

local function popup_action(name, title, run, category)
  rex.action{ name = name, title = title, category = category or "Popups", run = run }
end

-- one lazygit per repo
popup_action("lazygit", "lazygit", function(ctx)
  local root = git_root(pane_dir(ctx))
  toggle_popup(ctx, "lazygit:" .. root, { "lazygit" }, 0.9, root)
end)
-- one yazi per folder
popup_action("files", "Files (yazi)", function(ctx)
  local dir = pane_dir(ctx)
  toggle_popup(ctx, "yazi:" .. dir, { "yazi" }, 0.9, dir)
end)
popup_action("lazydocker", "lazydocker", function(ctx)
  toggle_popup(ctx, "lazydocker", { "lazydocker" }, 0.95, HOME)
end)
-- read-only on production, and ~/.config/k9s (kube-guard.zsh)
popup_action("k9s", "k9s", function(ctx)
  toggle_popup(ctx, "k9s", { "zsh", "-c", "source ~/.config/zsh/kube-guard.zsh && k9s" }, 0.95, HOME)
end)
-- ⌘J: scratch terminal, one per session
popup_action("scratch", "Scratch Terminal", function(ctx)
  toggle_popup(ctx, "scratch", nil, 0.8, pane_dir(ctx))
end)
-- always a fresh one: the diff of the working tree as it is now
popup_action("git_diff", "Diff (lumen)", function(ctx)
  popup({ "lumen", "diff" }, 0.95, pane_dir(ctx))
end)
popup_action("open_repo", "Open Repo", function(ctx)
  popup({ HOME .. "/.config/rex/sessionizer.sh" }, 0.7, HOME)
end, "Sessions")
popup_action("open_ssh", "Open SSH Host", function(ctx)
  popup({ HOME .. "/.config/tmux/ssh-picker.sh", "--rex" }, 0.6, HOME)
end, "Sessions")

-- ⌘D / ⌘⇧D: split in the pane's folder; a pane running `ssh host` opens the
-- same ssh in the new one (tmux/split.sh)
rex.action{
  name = "split_here", title = "Split Pane", category = "Panes",
  description = "In the pane's folder; a pane running ssh opens the same ssh",
  args = { direction = "string" },
  run = function(ctx, args)
    local options = { cwd = pane_dir(ctx) }
    local p = rex.block.call(TERMINAL, "process", { block_id = ctx.block_id })
    local fg = p and p.foreground
    if fg and fg.name == "ssh" then
      local ps = rex.exec({ "ps", "-o", "args=", "-p", tostring(fg.pid) })
      if ps and ps.ok and ps.stdout ~= "" then
        options.command = { "sh", "-c", ps.stdout:gsub("%s+$", "") }
      end
    end
    check(rex.session.new_split{
      anchor_id = ctx.block_id,
      direction = args.direction == "down" and "vertical" or "horizontal",
      side = "after",
      focus = true,
      layout = rex.layout.block{ flavor = SHELL, options = options },
    })
  end,
}

-- Tabs
rex.bind("cmd+t", "client.tab.new")
rex.bind("cmd+w", "pane.close")                -- the tab too, if it was the last pane
rex.bind("cmd+shift+left", "client.tab.previous")
rex.bind("cmd+shift+right", "client.tab.next")
rex.bind("cmd+shift+r", "client.tab.rename")
for i = 1, 9 do
  rex.bind("cmd+[Digit" .. i .. "]", "client.tab.goto", { index = i })
end

-- Panes
rex.bind("cmd+d", "split_here", { direction = "right" })
rex.bind("cmd+shift+d", "split_here", { direction = "down" })
rex.bind("cmd+alt+left", "pane.focus.left")
rex.bind("cmd+alt+right", "pane.focus.right")
rex.bind("cmd+alt+up", "pane.focus.up")
rex.bind("cmd+alt+down", "pane.focus.down")
rex.bind("cmd+shift+f", "pane.zoom")
rex.bind("cmd+shift+e", "pane.balance")

-- Previous / next command (OSC 133 marks from p10k)
rex.bind("cmd+shift+up", "client.prompt.previous")
rex.bind("cmd+shift+down", "client.prompt.next")

-- Sessions
rex.bind("cmd+shift+s", "session.switch")
rex.bind("cmd+o", "open_repo")
rex.bind("cmd+shift+o", "open_ssh")

-- Popups
rex.bind("cmd+g", "lazygit")
rex.bind("cmd+shift+g", "git_diff")
rex.bind("cmd+e", "files")
rex.bind("cmd+shift+l", "lazydocker")
rex.bind("cmd+k", "k9s")
rex.bind("cmd+j", "scratch")

-- ⌘⇧H: the command palette (also ⌘⇧P) lists every action with its key and runs it
rex.bind("cmd+shift+h", "client.palette.toggle")
