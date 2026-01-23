# FuzzyRepo Refactor Plan

## Project Overview

**FuzzyRepo** is a fast, minimal TUI tool for managing GitHub repositories and viewing them as projects in Neovim. It provides fuzzy search across both remote and local repositories with seamless workspace switching in Neovim.

### Core Principles
- One TUI everywhere (inside/outside Neovim) with identical keybinds
- "Workspace = tab-local cwd" in Neovim (no heavy session magic)
- Fast and simple (lazygit-inspired UX)
- Editor-agnostic design for open-source distribution
- Portable: no shell dependencies, works on macOS/Linux/Windows

## Current State Assessment

**Existing Code Structure:**
- `main.go` - Entry point with hardcoded early exit blocking execution
- `ui.go` - Bubble Tea TUI with basic list + fuzzy search
- `config.go` - YAML config with single `LocalRepoPath`
- `github.go` - GitHub API integration via `gh auth token`
- `helper.go` - Cache management, utility functions
- `repos.go` - Local repo scanning (incomplete implementation)
- `mock.go` - Test data

**Current Blockers:**
1. `main.go` has `if true { ... os.Exit(1) }` preventing execution
2. Local repo indexing incomplete (`getClonedRepos` finds `.git` but doesn't build proper DTOs)
3. Data model confusion: `Path` field overloaded (SSH URL or local path)
4. Cache directory not ensured to exist
5. No atomic cache writes
6. Noisy debug printing in helpers

## Target UX

### CLI Commands
```bash
fuzzyrepo              # Open TUI
fuzzyrepo init         # Interactive config setup
fuzzyrepo refresh      # Update cache (remote + local)
fuzzyrepo list [--json] # Dump cached repos
fuzzyrepo open <query>  # Non-TUI open
fuzzyrepo clone <query> # Non-TUI clone
```

### TUI Keybinds (Lazygit-style)
- **Type**: Filter repositories
- **Enter**: Open (clone first if needed)
- **c**: Clone only (no open)
- **y**: Copy SSH URL
- **p**: Copy local path (if cloned)
- **o**: Copy `owner/name`
- **r**: Refresh index
- **q/Esc**: Quit

### Open Behavior
**Outside Neovim:**
- Launches `$EDITOR` (default: `nvim`) at repository root

**Inside Neovim:**
- Detects via `$NVIM` environment variable
- Queries existing tab cwds via remote API
- If repo already open: jumps to existing tab
- Else: creates new workspace (`:tabnew | :tcd <repo_root>`)
- Result: each repo = isolated workspace with preserved buffers

## Implementation Plan

### Phase 0: Unblock & Cleanup
- [ ] Remove hardcoded early exit in `main.go`
- [ ] Remove debug printing from helpers (`getHomeDir()`, etc.)
- [ ] Ensure cache directory exists before writing
- [ ] Implement atomic cache writes (tmp + rename)

### Phase 1: Config & Install (Open-Source Ready)
- [ ] Switch to XDG config paths:
  - macOS/Linux: `~/.config/fuzzyrepo/config.yaml`
  - Windows: `%APPDATA%\fuzzyrepo\config.yaml`
- [ ] Implement `fuzzyrepo init` interactive setup
- [ ] Support multi-path repo roots
- [ ] Add backward compatibility for existing config

**New Config Schema:**
```yaml
repo_roots:
  - "/path/to/repos1"
  - "/path/to/repos2"
clone_root: "/path/to/repos1"  # Optional, defaults to repo_roots[0]
github:
  affiliation: "owner,collaborator,organization_member"
  orgs: []  # Optional allowlist
max_results: 200
refresh_ttl_minutes: 60
```

### Phase 2: Data Model Refactor
Replace current `RepoDTO` with explicit, unambiguous structure:

```go
type Repository struct {
    // Identity
    Host     string `json:"host"`     // "github.com"
    Owner    string `json:"owner"`    // "username"
    Name     string `json:"name"`     // "repo-name"
    FullName string `json:"fullname"` // "owner/repo-name"
    
    // Remote URLs
    SSHURL   string `json:"ssh_url"`   // "git@github.com:owner/repo.git"
    HTTPSURL string `json:"https_url"` // "https://github.com/owner/repo.git"
    
    // Local state
    LocalPath string `json:"local_path"` // "/path/to/local/repo" or ""
    Source    string `json:"source"`     // "local", "remote", "both"
    
    // Performance
    SearchText string `json:"search_text"` // Precomputed for fuzzy search
}
```

### Phase 3: Indexers & Merge Logic

**Local Indexer (Portable):**
- [ ] Use `filepath.WalkDir` on each `repo_root` (no shell `find`)
- [ ] Detect repositories by `.git` directory
- [ ] Parse `.git/config` to extract `remote "origin"` URL
- [ ] Derive `Host/Owner/Name` from origin when possible
- [ ] Always store `LocalPath` for local repos

**Remote Indexer (GitHub Only):**
- [ ] Auth via `GITHUB_TOKEN` or `gh auth token`
- [ ] Use go-github library with config-driven filters
- [ ] Apply `affiliation` and optional `orgs` allowlist
- [ ] Fetch SSH/HTTPS URLs

**Merge Strategy:**
- [ ] Key primarily by `Host + FullName`
- [ ] Fallback to normalized URL for repos without clear `FullName`
- [ ] When both local + remote exist: set `LocalPath`, `Source=both`

### Phase 4: Clone & Collision Handling

**Clone Destination Logic:**
- Default: `<clone_root>/<host>/<owner>/<repo>`
- [ ] Check if destination exists
- [ ] If exists with same remote: treat as already cloned
- [ ] If exists with different remote: suffix `repo__2`, `repo__3`, etc.

### Phase 5: Open Backend Implementation

**Outside Neovim:**
```go
func openExternal(repo Repository) error {
    editor := os.Getenv("EDITOR")
    if editor == "" {
        editor = "nvim"
    }
    return exec.Command(editor, repo.LocalPath).Run()
}
```

**Inside Neovim (Workspace Switching):**
- [ ] Detect Neovim via `$NVIM` environment variable
- [ ] Query existing tab cwds:
  ```bash
  nvim --server $NVIM --remote-expr 'join(map(range(1, tabpagenr("$")), {_,t -> getcwd(-1,t)}), "\n")'
  ```
- [ ] If repo root matches existing tab cwd: jump to that tab
- [ ] Else create new workspace:
  ```bash
  nvim --server $NVIM --remote-send ':tabnew | :tcd <repo_root><CR>'
  ```
- [ ] Optional: best-effort tree root change (editor-agnostic)

### Phase 6: TUI Performance & Polish

**Performance Optimizations:**
- [ ] Precompute `SearchText` once per repository
- [ ] Avoid rebuilding haystack on every keystroke
- [ ] Cap results to `max_results` from config
- [ ] Maintain responsive UI during background refresh

**UI Improvements:**
- [ ] Better status messages for clone/open operations
- [ ] Progress indicators for long operations
- [ ] Error handling with user-friendly messages

### Phase 7: Documentation & Packaging

**Documentation (`README.md`):**
- [ ] Project description & non-goals
- [ ] Installation instructions (`go install` + binaries)
- [ ] Configuration examples
- [ ] Commands reference
- [ ] TUI keybindings table
- [ ] Neovim integration guide

**Packaging:**
- [ ] GitHub Actions for cross-platform binaries
- [ ] Version management
- [ ] Release notes template

## Git Strategy

**Branch Management:**
1. Create feature branch: `git checkout -b feat/workspaces-indexer`
2. Handle current dirty worktree (stash or commit current changes)
3. Implement phases incrementally with commits per logical unit
4. Test each phase before moving to next

**Commit Strategy:**
- Phase 0: "fix: remove debug blocks and add atomic cache writes"
- Phase 1: "feat: add XDG config and fuzzyrepo init command"
- Phase 2: "refactor: replace RepoDTO with explicit Repository model"
- Phase 3: "feat: implement portable local indexer and merge logic"
- Phase 4: "feat: add clone with collision handling"
- Phase 5: "feat: implement Neovim workspace switching"
- Phase 6: "perf: optimize TUI and add better UX"
- Phase 7: "docs: add comprehensive README and examples"

## Technical Decisions

**Chosen Approaches:**
- **Config**: XDG-compliant paths with backward compatibility
- **Indexing**: Pure Go (no shell dependencies) for portability
- **Neovim Integration**: Built-in remote API only (no required plugins)
- **Collision Handling**: Predictable suffix numbering
- **Performance**: Precomputed search strings, capped results

**Explicitly Avoided:**
- Heavy session management (use simple tab + tcd)
- Editor-specific hard dependencies
- Complex preview panes (keep it fast and simple)
- Multiple VCS providers in v1 (GitHub only)

## File Structure Changes

**New Files:**
- `cmd/` - CLI command implementations
- `internal/config/` - Configuration management
- `internal/index/` - Local and remote indexers
- `internal/nvim/` - Neovim integration
- `internal/clone/` - Clone operations
- `docs/` - User documentation

**Modified Files:**
- `main.go` - CLI router and entry point
- `ui.go` - Updated for new Repository model
- `config.go` - XDG paths and multi-root support
- `github.go` - Enhanced with merge logic
- `helper.go` - Atomic operations and utilities

This plan provides a complete roadmap for transforming fuzzyrepo into a robust, open-source repository management tool with seamless Neovim integration.