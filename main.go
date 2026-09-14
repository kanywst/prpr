// Command prpr is a terminal dashboard for the open pull requests across
// every GitHub owner you can see: your own account and each org you belong to.
// Press enter to open one, and watch it wave goodbye when it gets merged.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kanywst/prpr/internal/gh"
	"github.com/kanywst/prpr/internal/ui"
)

// version is overridden at release time via -ldflags; otherwise it is read
// from the build info embedded by the Go toolchain.
var version = ""

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "prpr:", err)
		os.Exit(1)
	}
}

// ownerList collects repeatable --owner flags.
type ownerList []string

func (o *ownerList) String() string { return strings.Join(*o, ",") }

func (o *ownerList) Set(v string) error {
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			*o = append(*o, part)
		}
	}
	return nil
}

func run(args []string) error {
	fs := flag.NewFlagSet("prpr", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "prpr — GitHub の open PR を眺める TUI\n\n使い方: prpr [オプション]\n\n")
		fs.PrintDefaults()
	}

	var owners ownerList
	fs.Var(&owners, "owner", "監視する owner (繰り返し・カンマ区切り可)。既定はログインユーザーと所属 org")
	interval := fs.Duration("interval", time.Minute, "自動更新の間隔")
	timeout := fs.Duration("timeout", 20*time.Second, "1 回の更新のタイムアウト")
	showVersion := fs.Bool("version", false, "バージョンを表示して終了")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Println("prpr", buildVersion())
		return nil
	}
	if *interval < 5*time.Second {
		return fmt.Errorf("--interval は 5s 以上にしてね (指定: %s)", *interval)
	}
	if *timeout <= 0 {
		return errors.New("--timeout は正の値にしてね")
	}

	client, err := gh.New()
	if err != nil {
		return err
	}

	model := ui.New(ui.Config{
		Fetcher:  client,
		Owners:   owners,
		Interval: *interval,
		Timeout:  *timeout,
	})

	if _, err := tea.NewProgram(model).Run(); err != nil {
		return fmt.Errorf("TUI が落ちた: %w", err)
	}
	return nil
}

// buildVersion reports the release version, falling back to whatever the Go
// toolchain stamped into the binary.
func buildVersion() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}
