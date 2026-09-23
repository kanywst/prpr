package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadOverlaysTheDefaults(t *testing.T) {
	path := writeConfig(t, `
owners: [acme, kanywst]
exclude_owners: [noisy-org]
interval: 30s
lang: ja
review_requests: false
`)
	cfg, err := Load(path, true)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(cfg.Owners, []string{"acme", "kanywst"}) {
		t.Errorf("owners = %v", cfg.Owners)
	}
	if !slices.Equal(cfg.ExcludeOwners, []string{"noisy-org"}) {
		t.Errorf("exclude_owners = %v", cfg.ExcludeOwners)
	}
	if cfg.Interval != 30*time.Second || cfg.Lang != "ja" || cfg.ReviewRequests {
		t.Errorf("got interval=%s lang=%s review_requests=%v", cfg.Interval, cfg.Lang, cfg.ReviewRequests)
	}
	// Keys the file leaves out keep their defaults.
	if cfg.Timeout != Default().Timeout || !cfg.Authored {
		t.Errorf("unset keys lost their defaults: timeout=%s authored=%v", cfg.Timeout, cfg.Authored)
	}
}

func TestLoadMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.yaml")

	cfg, err := Load(missing, false)
	if err != nil {
		t.Fatalf("the default path not existing is an error: %v", err)
	}
	if cfg.Interval != Default().Interval {
		t.Error("a missing file did not yield the defaults")
	}

	if _, err := Load(missing, true); err == nil {
		t.Error("an explicit --config pointing nowhere was accepted")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	if _, err := Load(writeConfig(t, ""), true); err != nil {
		t.Errorf("an empty config file was rejected: %v", err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	_, err := Load(writeConfig(t, "intervall: 30s\n"), true)
	if err == nil || !strings.Contains(err.Error(), "intervall") {
		t.Errorf("a misspelled key was not reported: %v", err)
	}
}

func TestLoadRejectsBadDurations(t *testing.T) {
	if _, err := Load(writeConfig(t, "interval: soon\n"), true); err == nil {
		t.Error("an unparseable interval was accepted")
	}
}

func TestValidate(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("the defaults do not validate: %v", err)
	}
	cfg.Interval = time.Second
	if err := cfg.Validate(); err == nil {
		t.Error("an interval under the minimum validated")
	}
	cfg = Default()
	cfg.Timeout = 0
	if err := cfg.Validate(); err == nil {
		t.Error("a zero timeout validated")
	}
}

func TestPathFollowsXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(xdg, "prpr", "config.yaml"); got != want {
		t.Errorf("Path() = %s, want %s", got, want)
	}
}
