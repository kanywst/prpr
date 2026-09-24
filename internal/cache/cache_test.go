package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kanywst/prpr/internal/gh"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prpr", "cache.json")
	at := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	in := Snapshot{
		Me:     "kanywst",
		Owners: []string{"kanywst", "0-draft"},
		PRs:    []gh.PR{{Number: 1, Repo: "o/r", Author: "dependabot", IsBot: true, UpdatedAt: at}},
		At:     at,
	}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, ok := Load(path)
	if !ok {
		t.Fatal("Load could not read what Save wrote")
	}
	if out.Me != in.Me || len(out.Owners) != 2 || !out.At.Equal(at) {
		t.Errorf("Load = %+v, want %+v", out, in)
	}
	if len(out.PRs) != 1 || out.PRs[0].Key() != "o/r#1" || !out.PRs[0].IsBot {
		t.Errorf("PRs = %+v", out.PRs)
	}

	// Windows has no Unix permission bits to check; the file sits in the
	// user's own profile there.
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("cache mode = %v, want it private to the user", perm)
	}
}

func TestLoadIgnoresWhatItCannotUse(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"garbage":  "not json",
		"outdated": `{"version": 999, "at": "2026-09-24T09:00:00Z"}`,
		"undated":  `{"version": 1}`,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, ok := Load(path); ok {
			t.Errorf("%s: Load accepted it", name)
		}
	}
	if _, ok := Load(filepath.Join(dir, "missing")); ok {
		t.Error("Load accepted a missing file")
	}
}

func TestPathFollowsXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdg)
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(xdg, "prpr", "cache.json"); got != want {
		t.Errorf("Path() = %s, want %s", got, want)
	}
}
