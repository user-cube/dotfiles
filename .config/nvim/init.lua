-- Helm and Ansible files are plain .yaml: tell them apart by where they live,
-- so helm_ls / ansiblels attach instead of yamlls flagging template syntax
local function in_chart(dir)
  return vim.fs.find("Chart.yaml", { upward = true, path = dir, stop = vim.env.HOME })[1] ~= nil
end

local ansible_dirs = {
  "/tasks/", "/handlers/", "/playbooks/", "/group_vars/", "/host_vars/",
  "/roles/[^/]+/defaults/", "/roles/[^/]+/vars/", "/roles/[^/]+/meta/",
}

local function yaml_filetype(path, bufnr)
  local dir = vim.fs.dirname(path)
  if path:match("/templates/") and in_chart(dir) then
    return "helm"
  end
  if vim.fs.basename(path):match("^values.*%.ya?ml$") and in_chart(dir) then
    return "yaml.helm-values"
  end
  for _, p in ipairs(ansible_dirs) do
    if path:match(p) then
      return "yaml.ansible"
    end
  end
  -- A playbook anywhere else: a play with `hosts:` near the top
  for _, line in ipairs(vim.api.nvim_buf_get_lines(bufnr, 0, 30, false)) do
    if line:match("^%s*%-?%s*hosts:") then
      return "yaml.ansible"
    end
  end
  return "yaml"
end

vim.filetype.add({
  extension = {
    tofu = "opentofu",
    yaml = yaml_filetype,
    yml = yaml_filetype,
    tpl = function(path)
      return in_chart(vim.fs.dirname(path)) and "helm" or nil
    end,
  },
})

require("config.options")
require("config.lazy")
