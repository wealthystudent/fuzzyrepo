package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CacheMetadata tracks sync timestamps and state
type CacheMetadata struct {
	LastRemoteSync time.Time `json:"last_remote_sync"`
	LastLocalScan  time.Time `json:"last_local_scan"`
	RemoteSyncPID  int       `json:"remote_sync_pid,omitempty"` // PID of running sync process (0 if none)
}

// Sync frequency constants (not configurable by user)
const (
	RemoteSyncInterval = 7 * 24 * time.Hour // Weekly
	LocalScanInterval  = 24 * time.Hour     // Daily
)

func getMetadataPath() string {
	return filepath.Join(getCacheDir(), "metadata.json")
}

// LoadMetadata loads the cache metadata from disk
func LoadMetadata() (CacheMetadata, error) {
	var meta CacheMetadata

	data, err := os.ReadFile(getMetadataPath())
	if err != nil {
		if os.IsNotExist(err) {
			return meta, nil // Return empty metadata if file doesn't exist
		}
		return meta, err
	}

	if err := json.Unmarshal(data, &meta); err != nil {
		return CacheMetadata{}, err
	}

	return meta, nil
}

// SaveMetadata saves the cache metadata to disk atomically
func SaveMetadata(meta CacheMetadata) error {
	path := getMetadataPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// IsRemoteSyncDue returns true if a remote sync should be triggered
// (more than 7 days since last sync, or never synced)
func IsRemoteSyncDue(meta CacheMetadata) bool {
	if meta.LastRemoteSync.IsZero() {
		return true
	}
	return time.Since(meta.LastRemoteSync) > RemoteSyncInterval
}

// IsLocalScanDue returns true if a local scan should be triggered
// (more than 1 day since last scan, or never scanned)
func IsLocalScanDue(meta CacheMetadata) bool {
	if meta.LastLocalScan.IsZero() {
		return true
	}
	return time.Since(meta.LastLocalScan) > LocalScanInterval
}

// UpdateRemoteSyncTime updates the last remote sync timestamp
func (m *CacheMetadata) UpdateRemoteSyncTime() {
	m.LastRemoteSync = time.Now()
}

// UpdateLocalScanTime updates the last local scan timestamp
func (m *CacheMetadata) UpdateLocalScanTime() {
	m.LastLocalScan = time.Now()
}

// GetCacheMtime returns the modification time of the cache file
// Returns zero time if file doesn't exist or error occurs
func GetCacheMtime() time.Time {
	info, err := os.Stat(getCachePath())
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
