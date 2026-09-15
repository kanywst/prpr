# 🌸 prpr

[![CI](https://github.com/kanywst/prpr/actions/workflows/ci.yml/badge.svg)](https://github.com/kanywst/prpr/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/kanywst/prpr.svg)](https://pkg.go.dev/github.com/kanywst/prpr) [![Go Report Card](https://goreportcard.com/badge/github.com/kanywst/prpr)](https://goreportcard.com/report/github.com/kanywst/prpr) [![Release](https://img.shields.io/github/v/release/kanywst/prpr?sort=semver)](https://github.com/kanywst/prpr/releases/latest) [![Go version](https://img.shields.io/github/go-mod/go-version/kanywst/prpr)](go.mod) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**English** | [日本語](README.ja.md)

**prpr** puts every open GitHub pull request you can see on one screen. Press enter to open one in the browser, and watch it wave goodbye with a 🎉 when it gets merged.

There is nothing to configure. prpr asks GitHub who you are and watches **your own account plus every org you belong to**, so it shows the right pull requests for whoever runs it.

![prpr walking through a list of pull requests, opening the detail pane, filtering, switching tabs, and showing a merge banner](docs/demo.gif)

## What it does

- **Finds its own owners.** Asks GitHub for your login and your orgs at startup, then lists every open pull request under them.
- **Notices merges.** When a pull request leaves the list, prpr looks up how it ended and shows a farewell banner for a few seconds before it fades.
- **Four tabs.** Everything, yours, awaiting your review, and drafts, each with a live count.
- **Filtering.** `/` searches titles, repositories, authors, `#number` and labels at once. Space-separated terms are ANDed.
- **Detail pane.** `d` shows the branch, the diff stat, reviewers and the body. Side by side at 100 columns or wider, full width when narrower.
- **Auto refresh.** Every 60 seconds by default, with a countdown in the header.
- **Stops when you are away.** Polling pauses while the terminal does not have focus.
- **Mouse support.** Click to select, wheel to scroll.
- **Light and dark.** The palette is derived from the terminal's reported background color.
- **English or Japanese.** English by default; `--lang ja` switches the whole interface.
- **Reuses `gh`.** No token to configure.

## Install

prpr borrows the `gh` CLI's stored credentials, so `gh auth login` has to have been run.

Homebrew:

```bash
brew install kanywst/tap/prpr
```

Scoop, on Windows:

```bash
scoop bucket add kanywst https://github.com/kanywst/scoop-bucket
scoop install prpr
```

From source:

```bash
go install github.com/kanywst/prpr@latest
```

Prebuilt binaries for macOS, Linux and Windows are on the [releases page](https://github.com/kanywst/prpr/releases/latest).

## Usage

```bash
prpr
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--owner` | discovered | Owner to watch. Repeatable, and accepts a comma-separated list |
| `--interval` | `1m` | Auto-refresh interval (minimum `5s`) |
| `--timeout` | `20s` | Timeout for a single refresh |
| `--lang` | `en` | Interface language: `en` or `ja` |
| `--demo` | off | Run against a fixed fixture list instead of GitHub |
| `--version` | | Print the version and exit |

For example:

```bash
# watch a single org
prpr --owner 0-draft

# refresh every 30 seconds
prpr --interval 30s

# Japanese interface
prpr --lang ja
```

### Keys

| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Move the cursor |
| `g` / `G` | First / last |
| `pgup` / `pgdn` | Page |
| `tab` / `shift+tab` | Switch tabs |
| `enter` / `o` | Open in the browser |
| `y` | Copy the URL |
| `d` | Toggle the detail pane |
| `ctrl+u` / `ctrl+d` | Scroll the detail pane |
| `r` | Refresh now |
| `/` | Filter |
| `esc` | Clear the filter |
| `?` | Expand the help |
| `ctrl+z` | Suspend |
| `q` / `ctrl+c` | Quit |

### Icons

| Icon | Meaning |
| --- | --- |
| 🟢 / 🟡 / 🔴 / ⚪ | Checks passing / running / failing / none |
| 📝 | Draft |
| ✅ / 🔁 / 👀 | Approved / changes requested / review requested |

## Layout

| Package | Role |
| --- | --- |
| `internal/gh` | The domain type (`PR`) and the GitHub GraphQL adapter |
| `internal/browser` | The one OS-dependent side effect |
| `internal/demo` | The fixture list behind `--demo` |
| `internal/ui` | The Bubble Tea MVU loop, with pure helpers kept apart from state |
| `main.go` | Flag parsing and wiring |

`ui.Fetcher` is declared at the point of use, so the whole model is drivable in tests without network access.

Built on [Bubble Tea v2](https://github.com/charmbracelet/bubbletea), [Bubbles v2](https://github.com/charmbracelet/bubbles), [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) and [go-gh](https://github.com/cli/go-gh).

## Development

```bash
make check   # fmt + vet + lint + test
make test    # tests with the race detector
make build   # ./bin/prpr
make demo    # re-record docs/demo.gif with VHS
make help    # list targets
```

To look at the rendered UI without a terminal:

```bash
go test ./internal/ui -run TestDumpRender -v
```

## License

[MIT](LICENSE)
