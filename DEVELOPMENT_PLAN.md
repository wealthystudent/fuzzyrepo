# FuzzyRepo Development Plan

This document outlines the comprehensive plan for optimizing and enhancing fuzzyrepo's caching,
syncing, bootstrap experience, and configuration system.

## Overview

**Goals:**

1. Optimize startup speed - minimal checks before showing search UI
2. Implement robust caching with automatic background sync
3. Create a smooth first-run bootstrap experience
4. Add informational message system (text box at bottom)
5. Improve config UI with field descriptions
6. Add regex-based clone path rules (future phase)

**Key Principles:**

- Speed is paramount - UI must appear instantly
- Background operations must not block the user
- Sync processes must complete even if the tool exits
- All changes should be testable incrementally

---

## Architecture Decisions

### 1. Caching Strategy

**Full Cache** (`~/.local/share/fuzzyrepo/repos.json`):

- Contains ALL repos from GitHub (owner + orgs + collaborator)
- Updated by background sync process
- Written atomically (tmp file + rename)

**Filtered View** (in-memory `[]Repository`):

- Loaded from full cache, filtered by config (affiliation, orgs)
- Config changes reload from cache instantly (no network)
- This is what searches operate on

**Metadata** (`~/.local/share/fuzzyrepo/metadata.json`):

```json
{
  "last_remote_sync": "2024-01-15T10:30:00Z",
  "last_local_scan": "2024-01-16T08:00:00Z",
  "remote_sync_in_progress": false,
  "local_scan_in_progress": false
}
```

### 2. Sync Strategy

| Type   | Frequency | Trigger                                    | Method           |
| ------ | --------- | ------------------------------------------ | ---------------- |
| Remote | Weekly    | Startup if due                             | Detached process |
| Local  | Daily     | Startup if due                             | Inline (fast)    |
| Remote | Manual    | User triggers refresh                      | Detached process |
| Local  | Manual    | User triggers refresh OR repo_roots change | Inline           |

**Detached Process:**

- Spawned using `os/exec` with `SysProcAttr` for process detachment
- Runs as `fuzzyrepo --sync-remote` (same binary, different mode)
- Uses lock file to prevent concurrent syncs
- Updates cache file, main process detects via mtime

### 3. Startup Flow

```
main()
+-- Fast bootstrap check: does config file exist? (<1ms)
|   +-- NO (First Run):
|   |   +-- Check dependencies (git, gh, gh auth status)
|   |   +-- Show config wizard with field descriptions
|   |   +-- Spawn detached remote sync (owner -> orgs -> collaborator)
|   |   +-- Show search UI with "Syncing repositories..." message
|   |
|   +-- YES (Normal Run):
|       +-- Load config
|       +-- Load cache into filtered view
|       +-- Show search UI immediately
|       +-- Check if local scan due -> run inline if yes
|       +-- Check if remote sync due -> spawn detached if yes
|
+-- Background: Watch cache file mtime, reload on change
```

### 4. Message System

The informational text box at the bottom of the screen:

- Shows only the **latest** message (replaces previous)
- Hidden when no message to display (keeps UI clean)
- Message types: Info, Warning, Error (different colors)
- Used for: sync status, config descriptions, errors

---

## Implementation Phases

### Phase 1: Foundation & Code Cleanup

_Goal: Clean up codebase, establish new file structure_

### Phase 2: Core Caching & Metadata

_Goal: Implement metadata tracking and two-tier caching_

### Phase 3: Background Sync (Detached Process)

_Goal: Implement detached sync that survives tool exit_

### Phase 4: Local Scan Integration

_Goal: Integrate local scanning with metadata tracking_

### Phase 5: Bootstrap & First-Run Experience

_Goal: Smooth first-run with dependency checks and config wizard_

### Phase 6: Message System

_Goal: Implement informational message box_

### Phase 7: Config UI Enhancements

_Goal: Add field descriptions, improve config overlay_

### Phase 8: Filter System

_Goal: Config-based filtering loaded into memory_

### Phase 9: Regex Clone Rules (Deferred)

_Goal: Regex-based clone path determination_

---

## TODO List

Each task is designed to be independently testable. Complete each section before moving to the next.

### Phase 1: Foundation & Code Cleanup

- [ ] **1.1** Create new branch `feat/caching-and-sync-v2` from main
- [ ] **1.2** Remove duplicate `stripAnsi()` function (keep in helper.go, remove from ui.go)
- [ ] **1.3** Remove unused functions: `mapToSlice()`, `updateRepoCache()`, `getOs()`
- [ ] **1.4** Add `Affiliation` field to Repository struct for filtering

```go
type Repository struct {
    // ... existing fields
    Affiliation string `json:"affiliation"` // "owner", "collaborator", "org"
}
```

**Test:** Run `go build` - should compile. Run `fuzzyrepo` - should work as before.

---

### Phase 2: Core Caching & Metadata

- [ ] **2.1** Create `metadata.go` with metadata types and functions:

  - `CacheMetadata` struct with `LastRemoteSync`, `LastLocalScan`, `RemoteSyncPID` fields
  - `getMetadataPath()`, `LoadMetadata()`, `SaveMetadata()` functions
  - `IsRemoteSyncDue()` - returns true if >7 days since last sync
  - `IsLocalScanDue()` - returns true if >1 day since last scan

- [ ] **2.2** Update `loadRepoCache()` to also return metadata (or load separately)

- [ ] **2.3** Add cache file watcher in UI:
  - Store cache file mtime on startup
  - Periodically check mtime (every 2 seconds)
  - Reload cache if mtime changed

**Test:**

1. Run `fuzzyrepo`, manually edit `repos.json` in another terminal
2. Changes should appear in fuzzyrepo within 2 seconds without restart

---

### Phase 3: Background Sync (Detached Process)

- [ ] **3.1** Add `--sync-remote` flag to main.go for sync mode:

```go
if len(os.Args) > 1 && os.Args[1] == "--sync-remote" {
    runRemoteSync()
    return
}
```

- [ ] **3.2** Implement `runRemoteSync()` function:

  - Check/create lock file (`~/.local/share/fuzzyrepo/sync.lock`)
  - Write own PID to lock file
  - Fetch repos in phases (owner -> orgs -> collaborator)
  - Update `repos.json` atomically
  - Update metadata with sync timestamp
  - Remove lock file

- [ ] **3.3** Implement `spawnDetachedSync()`:

  - Check if sync already running (lock file + PID check)
  - Spawn `fuzzyrepo --sync-remote` as detached process
  - Return immediately (don't wait)

- [ ] **3.4** Integrate into startup:
  - Load metadata
  - If `IsRemoteSyncDue()` and no sync running -> `spawnDetachedSync()`
  - Show message "Syncing repositories in background..."

**Test:**

1. Delete `metadata.json`, run `fuzzyrepo`
2. Should see "Syncing repositories in background..."
3. Exit immediately, check if sync continues (watch `repos.json` mtime)
4. Run again - should not start another sync (lock file exists)

---

### Phase 4: Local Scan Integration

- [ ] **4.1** Modify `indexLocalRepos()` to update metadata after completion

- [ ] **4.2** On startup, if `IsLocalScanDue()`:

  - Run local scan inline (it's fast)
  - Merge with existing cache
  - Save cache
  - Update metadata

- [ ] **4.3** Trigger local scan when `repo_roots` changes in config

**Test:**

1. Change `repo_roots` in config, save
2. Should immediately scan new directories
3. New local repos should appear in search

---

### Phase 5: Bootstrap & First-Run Experience

- [ ] **5.1** Add `IsFirstRun()` function:

```go
func IsFirstRun() bool {
    _, err := os.Stat(xdgConfigPath())
    return os.IsNotExist(err)
}
```

- [ ] **5.2** Add `checkDependencies()` function:

  - Check `git` is installed: `exec.LookPath("git")`
  - Check `gh` is installed: `exec.LookPath("gh")`
  - Check `gh auth status`: `exec.Command("gh", "auth", "status")`
  - Return error with clear message if any fail

- [ ] **5.3** Update `main()` for first-run flow:

```go
if IsFirstRun() {
    if err := checkDependencies(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    // Show config wizard (handled in UI)
    spawnDetachedSync() // Start fetching repos immediately
}
```

- [ ] **5.4** Add `firstRun` flag to UI, auto-open config on first run

**Test:**

1. Delete config file, run `fuzzyrepo`
2. Should show dependency errors if git/gh missing
3. Should auto-open config wizard if deps OK
4. Should start background sync immediately

---

### Phase 5.5: Bug Fixes (Before Phase 6)

- [ ] **5.5.1** Fix terminal size coverage - UI should always fill the entire terminal, even when repo list is small or empty

- [ ] **5.5.2** Fix first-run sync - repositories should update/appear after first-run config save (currently sync may be too slow or not working properly)

---

### Phase 6: Message System

- [ ] **6.1** Create `message.go`:

```go
type MessageLevel int
const (
    InfoLevel MessageLevel = iota
    WarningLevel
    ErrorLevel
)

type StatusMessage struct {
    Text    string
    Level   MessageLevel
}
```

- [ ] **6.2** Add `statusMessage` field to UI Model

- [ ] **6.3** Render message box at bottom of screen:

  - Only show if `statusMessage.Text != ""`
  - Style based on level (cyan/yellow/red)
  - Single line, above the search input

- [ ] **6.4** Add message update mechanism:
  - `setMessage(text, level)` method
  - `clearMessage()` method
  - Messages from sync, errors, config save, etc.

**Test:**

1. Trigger a sync - should show "Syncing..." message
2. Sync complete - message should clear or show "Sync complete"
3. Config error - should show red error message

---

### Phase 7: Config UI Enhancements

- [ ] **7.1** Add field descriptions map:

```go
var configDescriptions = map[int]string{
    cfgRepoRoots:   "Directories to scan for local git repositories (comma-separated absolute paths)",
    cfgCloneRoot:   "Where to clone new repositories (absolute path, defaults to first repo_root)",
    cfgAffiliation: "Types of repos to show: owner, collaborator, organization_member",
    cfgOrgs:        "Specific organizations to include (comma-separated)",
}
```

- [ ] **7.2** Show description for focused field in message box

- [ ] **7.3** Add hint about regex clone rules in config overlay:

```
[Config]
...

Note: Clone path rules can be configured in the config file.
See README for regex rule syntax.
```

**Test:**

1. Open config (Space -> c)
2. Tab through fields - description should change
3. Description should appear in bottom message area

---

### Phase 8: Filter System

- [ ] **8.1** Add filter fields to Config:

```go
type Config struct {
    // ... existing
    ShowOwner        bool     `yaml:"show_owner"`        // default true
    ShowCollaborator bool     `yaml:"show_collaborator"` // default true
    ShowOrgMember    bool     `yaml:"show_org_member"`   // default true
    FilterOrgs       []string `yaml:"filter_orgs"`       // empty = show all
}
```

- [ ] **8.2** Add `filterRepos()` function:

  - Takes full cache, returns filtered slice based on config
  - Filters by affiliation and org membership

- [ ] **8.3** Use filtered view in UI:

  - `m.all` = filtered repos (not all cached repos)
  - Config change -> reload filtered view from cache

- [ ] **8.4** On config save:
  - If filter settings changed -> reload filtered view
  - If `repo_roots` changed -> trigger local scan
  - Update message to indicate refresh

**Test:**

1. Set `show_owner: false` in config
2. Owner repos should disappear from search
3. Set back to `true` - repos should reappear instantly

---

### Phase 9: Regex Clone Rules

_Regex-based clone path determination_

- [x] **9.1** Add `CloneRules` to Config:

```go
type CloneRule struct {
    Pattern string `yaml:"pattern"` // Regex pattern
    Path    string `yaml:"path"`    // Target directory
}

type Config struct {
    // ... existing
    CloneRules []CloneRule `yaml:"clone_rules"` // only via file edit
}
```

- [x] **9.2** Add `GetClonePath(fullName, repoName)` function:

  - Iterate rules in order
  - Match pattern against full_name (owner/repo)
  - Return first matching path + repo name
  - Fall back to default clone_root + repo name

- [x] **9.3** Add validation for clone_rules:

  - Pattern must be valid regex
  - Path must be absolute

- [x] **9.4** Update config field description to mention clone_rules

**Usage:**

```yaml
# Example: Work repos go to ~/work, OSS repos go to ~/oss
clone_rules:
  - pattern: "^my-company/.*"
    path: /Users/me/work
  - pattern: "^opensource-org/.*"
    path: /Users/me/oss
# Everything else falls back to clone_root
```

**Test:**

1. Add clone rule for `my-org/*` repos
2. Clone a `my-org/foo` repo - should go to specified path
3. Clone a normal repo - should go to default path

---

## File Changes Summary

### New Files

- `metadata.go` - Cache metadata tracking
- `message.go` - Status message types and rendering
- `sync.go` - Detached sync process logic

### Modified Files

- `main.go` - First-run detection, sync mode flag, startup flow
- `config.go` - Filter fields, clone rules, field descriptions
- `helper.go` - Cache file watching, lock file management
- `ui.go` - Message box rendering, config descriptions, first-run handling
- `repos.go` - Affiliation field, filter function
- `github.go` - Phased fetching (owner -> orgs -> collaborator)
- `README.md` - Document clone rules syntax

### Data Files

- `~/.local/share/fuzzyrepo/repos.json` - Full repo cache (unchanged format)
- `~/.local/share/fuzzyrepo/metadata.json` - NEW: Sync timestamps
- `~/.local/share/fuzzyrepo/sync.lock` - NEW: Sync lock file

---

## Testing Checklist

After each phase, verify:

- [ ] `go build` succeeds
- [ ] `go vet ./...` passes
- [ ] `fuzzyrepo` starts without errors
- [ ] Search still works correctly
- [ ] Existing features (clone, open, browse, copy) work
- [ ] No visual regressions in UI

---

## Notes

1. **Performance**: Always measure startup time. Target: <100ms to first UI render.

2. **Error handling**: All background operations should fail gracefully without crashing the UI.

3. **Backwards compatibility**: Existing config files should work without modification.

4. **Atomic writes**: All file writes use tmp + rename pattern for safety.

5. **Process cleanup**: Lock files include PID for stale lock detection.
