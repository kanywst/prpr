// Package browser opens URLs in the user's default browser.
package browser

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// command returns the platform's "open this URL" command.
func command(url string) (name string, args []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}
	case "windows":
		// rundll32 sidesteps cmd.exe's quoting rules, which mangle URLs
		// containing "&".
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}

// Open launches url in the default browser and returns without waiting for it.
func Open(url string) error {
	return OpenContext(context.Background(), url)
}

// OpenContext launches url in the default browser. The context bounds only the
// launcher process, which exits immediately; the browser it hands off to is
// unaffected by cancellation.
func OpenContext(ctx context.Context, url string) error {
	name, args := command(url)

	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not launch %s: %w", name, err)
	}
	// The browser outlives prpr; reap the launcher so it does not linger as a
	// zombie for the rest of the session.
	go func() { _ = cmd.Wait() }()
	return nil
}
