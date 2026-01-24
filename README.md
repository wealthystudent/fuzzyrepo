# fuzzyrepo

A fast TUI for fuzzy searching GitHub (remote) + local repositories, with clone/open/copy actions.

## Features

- Fuzzy search across remote GitHub repos and local repos (all keys go to search)
- Command palette (`Space`) for actions - keeps search uninterrupted
- Marks whether a repo already exists locally
- `Enter` opens the repo (clones first if needed)
- Progressive loading: cached repos appear instantly, GitHub repos stream in batches
- Frecency-based ranking: frequently/recently used repos appear first
- Neovim integration via a lightweight floating-terminal plugin

## Installation

```bash
brew install wealthystudent/tap/fuzzyrepo
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
```

Notes:

- `repo_roots` is a YAML list of absolute paths.
- `clone_root` must be an absolute path.
- Clone destination is `<clone_root>/<owner>/<repo>`.
- Alias proposal: `frp`

## Usage

Run:

```bash
fuzzyrepo
```

Keybinds:

| Key | Action |
| --- | --- |
| ↑ / ↓ | Navigate repos |
| Enter | Open selected repo (clone if needed) |
| Esc | Clear search / Quit |
| Space | Open command palette |

### Command Palette

Press `Space` to open the command palette, then use arrows to navigate or press the shortcut key:

| Key | Command |
| --- | --- |
| o | Open in editor |
| y | Copy local path |
| b | Open in browser |
| p | Open pull requests |
| r | Refresh |
| c | Config |
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
