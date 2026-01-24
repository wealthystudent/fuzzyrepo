# fuzzyrepo

A fast TUI for fuzzy searching GitHub (remote) + local repositories, with clone/open/copy actions.

## Features

- Fuzzy search across remote GitHub repos and local repos
- Marks whether a repo already exists locally
- `Enter` opens the repo (clones first if needed)
- `y` copies the local path (clones first if needed) via OSC52
- `r` refreshes the repo cache (remote + local)
- `,` opens your config in `$EDITOR`
- Neovim integration via a lightweight floating-terminal plugin

## Installation

```bash
go install github.com/wealthystudent/fuzzyrepo@latest
```

Prereqs:

- `git`
- GitHub CLI (`gh`) and an authenticated session:

```bash
gh auth login
```

`fuzzyrepo` uses `gh auth token` under the hood.

## Configuration

Config file:

- macOS/Linux: `~/.config/fuzzyrepo/config.yaml` (or `$XDG_CONFIG_HOME/fuzzyrepo/config.yaml`)
- Windows: `%APPDATA%\fuzzyrepo\config.yaml`
- Legacy fallback (if present): `~/.fuzzyrepo.yaml`

All config fields (from `config.go`):

```yaml
repo_roots:
  - /abs/path/to/repos
  - /abs/path/to/other/repos

clone_root: /abs/path/to/clone/root  # defaults to first repo_roots entry; else ~/repos

github:
  affiliation: owner,collaborator,organization_member
  orgs: my-org,another-org  # optional filter (comma-separated)

max_results: 200  # 0 = unlimited
```

Notes:

- `repo_roots` is a YAML list of absolute paths.
- `clone_root` must be an absolute path.
- Clone destination is `<clone_root>/<owner>/<repo>`.

## Usage

Run:

```bash
fuzzyrepo
```

Keybinds:

| Key | Action |
| --- | --- |
| ↑ / ↓ | Navigate |
| Enter | Open (clone if needed) |
| y | Copy local path (clone if needed) |
| r | Refresh |
| , | Edit config in `$EDITOR` |
| q | Quit |

## Neovim plugin

The plugin runs `fuzzyrepo` in a floating terminal and sets `NVIM=$VIM_SERVERNAME` so selecting a repo opens it in the same Neovim instance (new tab + `:tcd` to the repo).

### lazy.nvim

```lua
{
  "wealthystudent/fuzzyrepo",
  config = function()
    require("fuzzyrepo").setup({
      width = 0.8,
      height = 0.8,
      border = "rounded",
      cmd = "fuzzyrepo",
    })
  end,
}
```

### Command + keymap

The plugin defines `:Fuzzyrepo`.

```lua
vim.keymap.set("n", "<leader>fr", "<cmd>Fuzzyrepo<cr>", { desc = "fuzzyrepo" })
```

### Setup options

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `width` | number | `0.5` | Float width as a fraction of `vim.o.columns` |
| `height` | number | `0.4` | Float height as a fraction of `vim.o.lines` |
| `border` | string | `"rounded"` | Floating window border style |
| `cmd` | string | `"fuzzyrepo"` | Command to run |

### Tab names (optional)

When opening a repo from Neovim, fuzzyrepo sets `t:tabname` to the repo name. To display this in your tabline instead of just the tab number, add a custom tabs module.

#### NvChad

In `lua/chadrc.lua`:

```lua
M.ui = {
  tabufline = {
    modules = {
      tabs = function()
        local fn = vim.fn
        local result = ""
        local tabs = fn.tabpagenr("$")

        if tabs > 1 then
          for nr = 1, tabs do
            local name = fn.gettabvar(nr, "tabname", tostring(nr))
            local hl = "TabO" .. (nr == fn.tabpagenr() and "n" or "ff")
            result = result .. "%" .. nr .. "@TbGotoTab@%#Tb" .. hl .. "# " .. name .. " %X"
          end

          local new_tab = "%@TbNewTab@%#TbTabNewBtn# 󰐕 %X"
          local toggle = "%@TbToggleTabs@%#TbTabTitle# TABS %X"
          local small = "%@TbToggleTabs@%#TbTabTitle# 󰅁 %X"

          return vim.g.TbTabsToggled == 1 and small or new_tab .. toggle .. result
        end

        return ""
      end,
    },
  },
}
```

#### Vanilla Neovim / other configs

```lua
vim.o.showtabline = 2
vim.o.tabline = "%!v:lua.TabLine()"

function _G.TabLine()
  local s = ""
  for i = 1, vim.fn.tabpagenr("$") do
    local hl = (i == vim.fn.tabpagenr()) and "%#TabLineSel#" or "%#TabLine#"
    local name = vim.fn.gettabvar(i, "tabname", tostring(i))
    s = s .. hl .. " " .. name .. " "
  end
  return s .. "%#TabLineFill#"
end
```

## Clipboard (OSC52)

`y` copies using OSC52 escape sequences. Your terminal (and tmux, if used) must allow OSC52 clipboard passthrough.
