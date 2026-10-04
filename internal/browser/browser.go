// Package browser opens URLs in the user's default browser.
package browser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// settle is how long Open waits for the launcher to report back. macOS open
// and rundll32 hand the URL off and exit within milliseconds, but xdg-open can
// stay attached to a browser it had to start, so a launcher still running at
// this point is taken to have succeeded.
const settle = 3 * time.Second

// pipeGrace is how long Wait keeps reading stderr after the launcher exits.
// A browser the launcher started inherits the pipe and holds it open for as
// long as it runs; without a bound, Wait would not return until then.
const pipeGrace = 250 * time.Millisecond

// maxStderr caps how much launcher output is kept for an error message, so a
// browser logging to the inherited pipe cannot grow it without limit.
const maxStderr = 4 << 10

// command returns the platform's "open this URL" command. It is a variable so
// tests can stand in a launcher that fails or hangs.
var command = func(url string) (name string, args []string) {
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

// Open launches url in the default browser. It reports a launcher that could
// not start or that exited with an error, such as macOS open failing to reach
// Launch Services, rather than assuming the hand-off worked.
func Open(url string) error {
	name, args := command(url)

	var stderr cappedBuffer
	// Not bound to a cancelable context: killing a launcher that outlives
	// settle would only race the browser it is handing off to.
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Stderr = &stderr
	cmd.WaitDelay = pipeGrace
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not launch %s: %w", name, err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		// ErrWaitDelay means the launcher succeeded and something it started
		// kept the pipe open.
		if err == nil || errors.Is(err, exec.ErrWaitDelay) {
			return nil
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%s failed: %w: %s", name, err, msg)
		}
		return fmt.Errorf("%s failed: %w", name, err)
	case <-time.After(settle):
		// The launcher is still reaped by the goroutine above once it exits,
		// so it does not linger as a zombie.
		return nil
	}
}

// cappedBuffer keeps the first maxStderr bytes written to it and drops the
// rest, reporting every write as complete so the writer is never blocked.
type cappedBuffer struct{ buf bytes.Buffer }

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := maxStderr - b.buf.Len(); room > 0 {
		b.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func (b *cappedBuffer) String() string { return b.buf.String() }
