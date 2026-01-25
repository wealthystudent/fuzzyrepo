local M = {}

local defaults = {
	width = 0.5,
	height = 0.4,
	border = "rounded",
	cmd = "fuzzyrepo",
}

M.config = {}

function M.setup(opts)
	M.config = vim.tbl_deep_extend("force", defaults, opts or {})
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
	if not M.config.cmd then
		M.setup({})
	end

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

local function normalize_path(path)
	if not path or path == "" then
		return ""
	end
	local p = vim.fn.fnamemodify(path, ":p")
	local real = vim.loop.fs_realpath(p)
	return real or p
end

function M.open_repo(path)
	local target = normalize_path(path)
	if target == "" then
		return
	end
	for _, tab in ipairs(vim.api.nvim_list_tabpages()) do
		local tabnr = vim.api.nvim_tabpage_get_number(tab)
		local tabcwd = normalize_path(vim.fn.getcwd(-1, tabnr))
		if tabcwd == target then
			vim.cmd("tabnext " .. tabnr)
			return
		end
	end
	vim.cmd("tabnew")
	vim.cmd("tcd " .. vim.fn.fnameescape(target))
end

return M
