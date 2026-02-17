# fuzzyrepo

A fast TUI for fuzzy searching GitHub (remote) + local repositories, with clone/open/copy actions.

## Features

- Fuzzy search across remote GitHub repos and local repos (all keys go to search)
- Command palette (`Space`) for actions - keeps search uninterrupted
- Marks whether a repo already exists locally
- `Enter` opens the repo (clones first if needed)
- **Background sync**: Repository data syncs in background, survives tool exit
- **Instant startup**: Cached repos appear immediately, sync happens in background
- Frecency-based ranking: frequently/recently used repos appear first
- **Repository filters**: Show/hide repos by type (owner, collaborator, org member, local-only)
- **Regex clone rules**: Route repos to different directories based on pattern matching
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

On first run, fuzzyrepo will check for these dependencies and show helpful error messages if anything is missing.

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

# Repository filters (all default to true)
show_owner: true        # Show repos you own
show_collaborator: true # Show repos where you're a collaborator
show_org_member: true   # Show organization repos
show_local: true        # Show local-only repos (not on GitHub)

# Regex clone rules (optional) - see Clone Rules section
clone_rules:
  - pattern: "^my-company/.*"
    path: /Users/me/work
```

Notes:

- First-time usage will trigger a background sync - repos will appear as they're fetched.
- `repo_roots` is a YAML list of absolute paths.
- `clone_root` must be an absolute path.
- Clone destination is `<clone_root>/<owner>/<repo>` (unless overridden by clone rules).
- Alias proposal: `frp`

### Clone Rules

Clone rules let you route repositories to different directories based on regex patterns. Rules are evaluated in order; the first match wins.

```yaml
clone_rules:
  - pattern: "^work-org/.*"      # Matches work-org/any-repo
    path: /Users/me/work
  - pattern: "^opensource/.*"    # Matches opensource/any-repo
    path: /Users/me/oss
  - pattern: ".*-config$"        # Matches any repo ending in -config
    path: /Users/me/dotfiles
```

- `pattern`: Regex matched against the full repo name (`owner/repo`)
- `path`: Absolute path to clone into (repo name is appended)

If no rule matches, the default `clone_root` is used.

**Tip**: Press `e` in the config overlay to open the config file directly in `$EDITOR` for editing clone rules.

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
| o | Enter path |
| y | Copy local path |
| b | Open in browser |
| p | Open pull requests |
| r | Refresh |
| c | Config |
| q | Quit |

### Config Overlay

Press `Space` then `c` to open the config overlay. Each field shows a helpful description when focused.

Press `e` in the config overlay to open the config file in your `$EDITOR` - useful for editing clone rules or other advanced settings.

## Background Sync

fuzzyrepo syncs repository data intelligently:

- **Remote sync** (GitHub API): Runs weekly in a detached background process
- **Local scan** (filesystem): Runs daily, inline (fast)
- Cache file is watched - UI updates automatically when sync completes

The sync process continues even if you exit fuzzyrepo. A lock file prevents concurrent syncs.

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

## License

MIT License - see [LICENSE](LICENSE) for details.
