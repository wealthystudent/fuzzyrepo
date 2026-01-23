local M = {}

local defaults = {
  width = 0.8,
  height = 0.8,
  border = "rounded",
  cmd = "fuzzyrepo",
}

M.config = {}

function M.setup(opts)
  M.config = vim.tbl_deep_extend("force", defaults, opts or {})
end

local function create_float_win()
  local width = math.floor(vim.o.columns * M.config.width)
  local height = math.floor(vim.o.lines * M.config.height)
  local row = math.floor((vim.o.lines - height) / 2)
  local col = math.floor((vim.o.columns - width) / 2)

  local buf = vim.api.nvim_create_buf(false, true)

  local win = vim.api.nvim_open_win(buf, true, {
    relative = "editor",
    width = width,
    height = height,
    row = row,
    col = col,
    style = "minimal",
    border = M.config.border,
  })

  return buf, win
end

function M.open()
  local buf, win = create_float_win()

  local env = {
    NVIM = vim.v.servername,
  }

  vim.fn.termopen(M.config.cmd, {
    env = env,
    on_exit = function()
      if vim.api.nvim_win_is_valid(win) then
        vim.api.nvim_win_close(win, true)
      end
      if vim.api.nvim_buf_is_valid(buf) then
        vim.api.nvim_buf_delete(buf, { force = true })
      end
    end,
  })

  vim.cmd("startinsert")
end

M.setup()

return M
