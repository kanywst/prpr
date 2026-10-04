package browser

import (
	"strings"
	"testing"
	"time"
)

// stub replaces the launcher with a shell snippet for the length of a test.
func stub(t *testing.T, script string) {
	t.Helper()
	orig := command
	command = func(string) (string, []string) { return "sh", []string{"-c", script} }
	t.Cleanup(func() { command = orig })
}

func TestOpenSucceeds(t *testing.T) {
	stub(t, "exit 0")
	if err := Open("https://example.com"); err != nil {
		t.Fatalf("Open() = %v, want nil", err)
	}
}

func TestOpenReportsLauncherFailure(t *testing.T) {
	// The launcher starting is not the same as the URL opening: open exits
	// non-zero when Launch Services refuses the request.
	stub(t, "echo 'LSOpenURLsWithRole() failed' >&2; exit 1")
	err := Open("https://example.com")
	if err == nil {
		t.Fatal("Open() = nil for a launcher that exited 1")
	}
	if !strings.Contains(err.Error(), "LSOpenURLsWithRole") {
		t.Errorf("Open() error %q does not carry the launcher's stderr", err)
	}
}

func TestOpenReportsMissingLauncher(t *testing.T) {
	orig := command
	command = func(string) (string, []string) { return "prpr-no-such-launcher", nil }
	t.Cleanup(func() { command = orig })
	if err := Open("https://example.com"); err == nil {
		t.Fatal("Open() = nil for a launcher that does not exist")
	}
}

func TestOpenDoesNotWaitOnALingeringLauncher(t *testing.T) {
	// xdg-open can stay attached to the browser it started; that is not a
	// failure, and must not block the caller indefinitely.
	stub(t, "sleep 10")
	start := time.Now()
	if err := Open("https://example.com"); err != nil {
		t.Fatalf("Open() = %v, want nil", err)
	}
	if took := time.Since(start); took > settle+2*time.Second {
		t.Errorf("Open() took %v, want about %v", took, settle)
	}
}
