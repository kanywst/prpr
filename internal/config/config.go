// Package config loads prpr's optional YAML config file. Every setting also
// has a default, so the file only needs the lines someone wants to change,
// and a command-line flag always wins over the file.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

// MinInterval is the shortest auto-refresh interval allowed. Anything tighter
// spends the search rate limit on a list that barely changes.
const MinInterval = 5 * time.Second

// Config is every setting prpr reads from the file.
type Config struct {
	// Owners pins the owners to watch. Empty means discover them: the
	// logged-in user and every org they belong to.
	Owners []string `yaml:"owners"`
	// ExcludeOwners drops owners from the discovered list, for orgs you
	// belong to but do not want on the dashboard.
	ExcludeOwners []string `yaml:"exclude_owners"`
	// Interval is how often the list auto-refreshes.
	Interval time.Duration `yaml:"interval"`
	// Timeout bounds a single refresh.
	Timeout time.Duration `yaml:"timeout"`
	// Lang is the interface language, en or ja.
	Lang string `yaml:"lang"`
	// Authored adds the PRs you opened anywhere, not just under the owners.
	Authored bool `yaml:"authored"`
	// ReviewRequests adds the PRs asking for your review anywhere, not just
	// under the owners.
	ReviewRequests bool `yaml:"review_requests"`
}

// Default is the configuration prpr runs with when nothing overrides it.
func Default() Config {
	return Config{
		Interval:       time.Minute,
		Timeout:        20 * time.Second,
		Lang:           "en",
		Authored:       true,
		ReviewRequests: true,
	}
}

// Path is where the config file lives: $XDG_CONFIG_HOME/prpr/config.yaml,
// falling back to ~/.config/prpr/config.yaml. The XDG layout is used on every
// platform, macOS included, because that is where people look for a CLI's
// config, not ~/Library/Application Support.
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "prpr", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find the home directory for the config file: %w", err)
	}
	return filepath.Join(home, ".config", "prpr", "config.yaml"), nil
}

// Load reads the config file at path on top of the defaults. A missing file
// is not an error unless required is set, which is how an explicit --config
// that points nowhere gets reported instead of silently ignored.
func Load(path string, required bool) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist) && !required:
		return cfg, nil
	case err != nil:
		return cfg, fmt.Errorf("could not read the config file: %w", err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	// A misspelled key would otherwise be ignored without a word, leaving
	// someone wondering why their setting does nothing.
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Validate rejects settings prpr cannot run with.
func (c Config) Validate() error {
	if c.Interval < MinInterval {
		return fmt.Errorf("interval must be at least %s (got %s)", MinInterval, c.Interval)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive (got %s)", c.Timeout)
	}
	return nil
}
