local M = {}

local defaults = {
  width = 0.5,
  height = 0.4,
  border = "rounded",
  cmd = "fuzzyrepo",
}

M.config = {}
M._term_buf = nil
M._term_win = nil

function M.setup(opts)
  M.config = vim.tbl_deep_extend("force", defaults, opts or {})

  vim.keymap.set("n", "<leader>tt", M.toggle_terminal, { desc = "Toggle terminal" })
end

local function create_float_win(width_pct, height_pct)
  local width = math.floor(vim.o.columns * width_pct)
  local height = math.floor(vim.o.lines * height_pct)
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
  local buf, win = create_float_win(M.config.width, M.config.height)

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

function M.toggle_terminal()
  if M._term_win and vim.api.nvim_win_is_valid(M._term_win) then
    vim.api.nvim_win_hide(M._term_win)
    M._term_win = nil
    return
  end

  if M._term_buf and vim.api.nvim_buf_is_valid(M._term_buf) then
    M._term_win = vim.api.nvim_open_win(M._term_buf, true, {
      relative = "editor",
      width = math.floor(vim.o.columns * 0.9),
      height = math.floor(vim.o.lines * 0.8),
      row = math.floor(vim.o.lines * 0.1),
      col = math.floor(vim.o.columns * 0.05),
      style = "minimal",
      border = "rounded",
    })
    vim.cmd("startinsert")
    return
  end

  M._term_buf = vim.api.nvim_create_buf(false, true)
  M._term_win = vim.api.nvim_open_win(M._term_buf, true, {
    relative = "editor",
    width = math.floor(vim.o.columns * 0.9),
    height = math.floor(vim.o.lines * 0.8),
    row = math.floor(vim.o.lines * 0.1),
    col = math.floor(vim.o.columns * 0.05),
    style = "minimal",
    border = "rounded",
  })

  vim.fn.termopen(vim.o.shell, {
    on_exit = function()
      if M._term_buf and vim.api.nvim_buf_is_valid(M._term_buf) then
        vim.api.nvim_buf_delete(M._term_buf, { force = true })
      end
      M._term_buf = nil
      M._term_win = nil
    end,
  })

  vim.cmd("startinsert")
end

M.setup()

return M
