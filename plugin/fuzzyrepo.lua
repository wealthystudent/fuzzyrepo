vim.api.nvim_create_user_command("Fuzzyrepo", function()
  require("fuzzyrepo").open()
end, {})
