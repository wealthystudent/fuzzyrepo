package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type UsageEntry struct {
	Count      int       `json:"count"`
	LastUsedAt time.Time `json:"last_used_at"`
}

type UsageData map[string]UsageEntry

func getUsagePath() string {
	return filepath.Join(getCacheDir(), "usage.json")
}

func LoadUsage() (UsageData, error) {
	path := getUsagePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(UsageData), nil
		}
		return nil, err
	}

	var usage UsageData
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil, err
	}

	return usage, nil
}

func SaveUsage(usage UsageData) error {
	path := getUsagePath()
	cacheDir := getCacheDir()

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(usage, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func RecordUsage(repo Repository) error {
	usage, err := LoadUsage()
	if err != nil {
		usage = make(UsageData)
	}

	key := strings.ToLower(repo.FullName)
	entry := usage[key]
	entry.Count++
	entry.LastUsedAt = time.Now()
	usage[key] = entry

	return SaveUsage(usage)
}

func GetUsageBoost(usage UsageData, repo Repository) float64 {
	key := strings.ToLower(repo.FullName)
	entry, ok := usage[key]
	if !ok || entry.Count == 0 {
		return 0
	}

	freqScore := math.Log2(1 + float64(entry.Count))

	daysSince := time.Since(entry.LastUsedAt).Hours() / 24
	halfLifeDays := 7.0
	recencyScore := math.Pow(0.5, daysSince/halfLifeDays)

	return freqScore*1.5 + recencyScore*2.0
}

func SortByUsage(repos []Repository, usage UsageData) []Repository {
	result := make([]Repository, len(repos))
	copy(result, repos)

	for i := 1; i < len(result); i++ {
		j := i
		for j > 0 {
			boostJ := GetUsageBoost(usage, result[j])
			boostJMinus1 := GetUsageBoost(usage, result[j-1])
			if boostJ > boostJMinus1 {
				result[j], result[j-1] = result[j-1], result[j]
				j--
			} else {
				break
			}
		}
	}

	return result
}
