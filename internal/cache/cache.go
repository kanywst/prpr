// Package cache keeps the last list prpr saw on disk, so a fresh start can
// show it straight away instead of an empty screen while the first search is
// in flight.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kanywst/prpr/internal/gh"
)

// version is bumped whenever Snapshot changes shape. A file written by
// another version is ignored rather than half-read.
const version = 1

// Snapshot is one completed refresh, as it is kept between runs.
type Snapshot struct {
	Version int `json:"version"`
	// Me and Owners are who the list was fetched for. They are shown until
	// discovery confirms them, and are what the cached list is judged
	// against when it does not.
	Me     string    `json:"me"`
	Owners []string  `json:"owners"`
	PRs    []gh.PR   `json:"prs"`
	At     time.Time `json:"at"`
}

// Path is where the cache lives: $XDG_CACHE_HOME/prpr/cache.json, falling
// back to ~/.cache/prpr/cache.json, on every platform for the same reason the
// config file follows XDG.
func Path() (string, error) {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "prpr", "cache.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find the home directory for the cache: %w", err)
	}
	return filepath.Join(home, ".cache", "prpr", "cache.json"), nil
}

// Load reads the snapshot at path. A missing, unreadable or outdated file
// yields ok == false: the cache is only ever a head start, so there is nothing
// to report when it is not there.
func Load(path string) (Snapshot, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, false
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil || s.Version != version || s.At.IsZero() {
		return Snapshot{}, false
	}
	return s, true
}

// Save writes s to path, replacing the file in one step so a crash mid-write
// cannot leave a torn cache behind. The file is private to the user: it holds
// the titles and bodies of pull requests in private repositories.
func Save(path string, s Snapshot) error {
	s.Version = version
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("could not create the cache directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".cache-*.json")
	if err != nil {
		return fmt.Errorf("could not write the cache: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("could not write the cache: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not write the cache: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("could not write the cache: %w", err)
	}
	return nil
}
