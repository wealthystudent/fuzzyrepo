# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-02-01

### Added

- **Background Sync**: Repository data is now synced in a detached background process that continues even after fuzzyrepo exits
- **Cache File Watching**: UI automatically reloads when cache is updated by background sync (checks every 2 seconds)
- **Repository Filter System**: New config options to filter repos by type
  - `show_owner`: Show repositories you own (default: true)
  - `show_collaborator`: Show repositories where you're a collaborator (default: true)
  - `show_org_member`: Show organization repositories (default: true)
  - `show_local`: Show local-only repositories (default: true)
- **Regex Clone Rules**: Configure where to clone repos based on pattern matching
  - Define rules in config file under `clone_rules`
  - Patterns match against full repo name (owner/repo)
  - First matching rule determines clone destination
- **Config Editor Shortcut**: Press 'e' in config overlay to open config file in `$EDITOR`
- **First-Run Bootstrap**: Dependency checks (git, gh, gh auth) on first run with helpful error messages
- **Status Message System**: Informational messages shown at bottom of screen for sync status, errors, etc.
- **Config Field Descriptions**: Each config field shows a helpful description when focused
- **Metadata Tracking**: Track last sync times for intelligent sync scheduling (weekly remote, daily local)
- **Lock File**: Prevents concurrent sync processes with PID-based stale lock detection

### Changed

- **Faster Startup**: UI appears instantly, cache is loaded immediately, sync happens in background
- **Affiliation Tracking**: Repositories now track their affiliation type (owner, collaborator, organization_member)
- **Atomic File Writes**: All file operations use tmp + rename pattern for safety
- **go-github Upgraded**: Updated from v17 to v68 for latest API compatibility

### Fixed

- Config changes now take effect immediately without restarting
- Filter changes update the repository list in real-time
- Fixed race condition in sync lock file creation using `O_CREATE|O_EXCL`
- Neovim integration now properly escapes paths in Lua commands

### Security

- **EDITOR Validation**: `$EDITOR` environment variable is now validated to prevent command injection
- **Path Sanitization**: Paths are properly escaped when passed to Neovim Lua commands
- **File Permissions**: Config, cache, and metadata files now use `0o600` instead of `0o644`
- **Lock File Atomic Creation**: Uses `O_CREATE|O_EXCL` flags to prevent race conditions

### Removed

- Unused functions: `GetOrgs()`, `ConfigPath()`, `RenderSimple()`, `mapToSlice()`, `updateRepoCache()`, `getOs()`
- "GitHub Affiliation" field from config UI (now fetches all types automatically)

## [1.0.9] - Previous Release

Initial stable release with core functionality:
- Fuzzy search for GitHub and local repositories
- Clone, open in editor, browse on GitHub
- SSH URL copying to clipboard
- Basic configuration system
