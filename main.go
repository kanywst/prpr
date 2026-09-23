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

	tea "charm.land/bubbletea/v2"

	"github.com/kanywst/prpr/internal/config"
	"github.com/kanywst/prpr/internal/demo"
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
		fmt.Fprint(fs.Output(),
			"prpr — a TUI for the open GitHub pull requests you can see\n\nusage: prpr [options]\n\n")
		fs.PrintDefaults()
		fmt.Fprint(fs.Output(), "\nevery option but --config, --demo and --version can also be set in the config file;\nflags win over the file.\n")
	}

	def := config.Default()
	var owners ownerList
	fs.Var(&owners, "owner", "owner to watch; repeatable and comma-separated.\ndefaults to the logged-in user and every org they belong to")
	interval := fs.Duration("interval", def.Interval, "auto-refresh interval")
	timeout := fs.Duration("timeout", def.Timeout, "timeout for a single refresh")
	lang := fs.String("lang", def.Lang, "interface language: en or ja")
	configPath := fs.String("config", "", "config file (default $XDG_CONFIG_HOME/prpr/config.yaml, or ~/.config/prpr/config.yaml)")
	demoMode := fs.Bool("demo", false, "run against a canned pull request list, for screenshots and recordings")
	showVersion := fs.Bool("version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Println("prpr", buildVersion())
		return nil
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	// Only flags given on the command line override the file; their defaults
	// must not silently undo a setting someone wrote down.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "owner":
			cfg.Owners = owners
		case "interval":
			cfg.Interval = *interval
		case "timeout":
			cfg.Timeout = *timeout
		case "lang":
			cfg.Lang = *lang
		}
	})
	if err := cfg.Validate(); err != nil {
		return err
	}
	parsed, ok := ui.ParseLang(cfg.Lang)
	if !ok {
		return fmt.Errorf("lang must be en or ja (got %q)", cfg.Lang)
	}

	fetcher, err := newFetcher(*demoMode)
	if err != nil {
		return err
	}

	model := ui.New(ui.Config{
		Fetcher:        fetcher,
		Owners:         cfg.Owners,
		ExcludeOwners:  cfg.ExcludeOwners,
		Interval:       cfg.Interval,
		Timeout:        cfg.Timeout,
		Lang:           parsed,
		Authored:       cfg.Authored,
		ReviewRequests: cfg.ReviewRequests,
	})

	if _, err := tea.NewProgram(model).Run(); err != nil {
		return fmt.Errorf("the TUI stopped: %w", err)
	}
	return nil
}

// loadConfig reads the config file named by --config, or the default one when
// it exists.
func loadConfig(path string) (config.Config, error) {
	if path != "" {
		return config.Load(path, true)
	}
	path, err := config.Path()
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(path, false)
}

// newFetcher picks the live GitHub client, or the fixture one behind --demo.
func newFetcher(demoMode bool) (ui.Fetcher, error) {
	if demoMode {
		return demo.New(), nil
	}
	return gh.New()
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
