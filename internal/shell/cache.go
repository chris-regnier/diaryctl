package shell

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const cacheFileName = ".prompt-cache"

// PromptCache holds cached prompt status data.
type PromptCache struct {
	Today           bool      `json:"today"`
	Streak          int       `json:"streak"`
	TodayDate       string    `json:"today_date"`
	DefaultTemplate string    `json:"default_template"`
	StorageBackend  string    `json:"storage_backend"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CachePath returns the full path to the prompt cache file.
func CachePath(dataDir string) string {
	return filepath.Join(dataDir, cacheFileName)
}

// ReadCache reads the prompt cache from disk. Returns nil if the cache
// does not exist or cannot be parsed.
func ReadCache(dataDir string) *PromptCache {
	data, err := os.ReadFile(CachePath(dataDir))
	if err != nil {
		return nil
	}
	var c PromptCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	return &c
}

// WriteCache writes the prompt cache to disk.
func WriteCache(dataDir string, c *PromptCache) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(CachePath(dataDir), data, 0600)
}

// IsFresh returns true if the cache is still valid given the TTL.
// A cache is stale if the TTL has elapsed or the date has changed (midnight rollover).
func (c *PromptCache) IsFresh(ttl time.Duration) bool {
	if c == nil {
		return false
	}
	now := time.Now()
	today := now.Format("2006-01-02")

	// Date changed — always stale
	if c.TodayDate != today {
		return false
	}

	// TTL expired
	if now.Sub(c.UpdatedAt) > ttl {
		return false
	}

	return true
}

// InvalidateCache removes the prompt cache file.
func InvalidateCache(dataDir string) error {
	path := CachePath(dataDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
