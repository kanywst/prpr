// Package ui implements prpr's terminal interface: a Bubble Tea program that
// lists the open pull requests across every owner the viewer can see, and
// waves goodbye to each one as it gets merged.
package ui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/kanywst/prpr/internal/gh"
)

// Fetcher is the slice of GitHub that the UI needs. It is declared here, at
// the point of use, so the UI can be driven by a fake in tests without the gh
// package knowing anything about it.
type Fetcher interface {
	// Viewer returns the authenticated login and the orgs it belongs to.
	Viewer(ctx context.Context) (login string, orgs []string, err error)
	// SearchOpenPRs returns every open PR under the given owners.
	SearchOpenPRs(ctx context.Context, owners []string) ([]gh.PR, error)
	// State reports how a PR left the open list.
	State(ctx context.Context, repo string, number int) (gh.State, error)
}

// Config is the wiring a Model needs to run.
type Config struct {
	// Fetcher is the GitHub data source. Required.
	Fetcher Fetcher
	// Owners pins the list of owners to watch. When empty, prpr discovers
	// them from the authenticated user: their own account plus every org they
	// belong to.
	Owners []string
	// Interval is how often the list auto-refreshes.
	Interval time.Duration
	// Timeout bounds a single refresh.
	Timeout time.Duration
	// Lang selects the interface language. The zero value is English.
	Lang Lang
}

// Tunables that are deliberately not exposed as flags: they are timings the
// UI's feel depends on, not preferences.
const (
	tickInterval    = 250 * time.Millisecond
	farewellLife    = 6 * time.Second
	farewellFade    = 4 * time.Second
	sideBySideWidth = 100
	compactHeight   = 14
	roomyRowLines   = 3
	compactRowLines = 1
)

// mode is which input the keyboard is currently driving.
type mode int

const (
	modeList mode = iota
	modeFilter
)

// farewell is a pull request that has just left the open list, kept on screen
// for a moment so a merge is something you see happen rather than something
// you notice is missing.
type farewell struct {
	pr    gh.PR
	state gh.State
	born  time.Time
}

// Model is the Bubble Tea model for prpr.
type Model struct {
	fetcher  Fetcher
	interval time.Duration
	timeout  time.Duration

	// pinnedOwners is non-empty when the user passed --owner, in which case
	// owner discovery is skipped entirely.
	pinnedOwners []string
	owners       []string
	me           string

	prs       []gh.PR
	visible   []gh.PR
	farewells []farewell

	cursor int
	offset int
	tab    tabID

	mode      mode
	loading   bool
	ready     bool
	focused   bool
	lastErr   error
	lastFetch time.Time
	flash     string
	flashTill time.Time

	width, height int
	theme         Theme
	s             Strings

	spinner    spinner.Model
	filter     textinput.Model
	help       help.Model
	detail     viewport.Model
	detailOpen bool
	detailKey  string

	keys       KeyMap
	filterKeys FilterKeyMap

	now  time.Time
	quit bool
}

// New builds a Model from cfg. It does no I/O; the first fetch is kicked off
// by Init.
func New(cfg Config) Model {
	s := Catalog(cfg.Lang)

	sp := spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: []string{"🌸", "🌺", "🌷", "🌼", "🌻", "🌼", "🌷", "🌺"},
		FPS:    time.Second / 6,
	}))

	fi := textinput.New()
	fi.Prompt = ""
	fi.Placeholder = s.FilterPlaceholder
	fi.CharLimit = 80
	// prpr draws the real terminal cursor at the input's position, so the
	// input must not also draw a fake one.
	fi.SetVirtualCursor(false)

	h := help.New()
	vp := viewport.New()
	vp.SoftWrap = true
	vp.MouseWheelEnabled = true

	return Model{
		fetcher:      cfg.Fetcher,
		interval:     cfg.Interval,
		timeout:      cfg.Timeout,
		pinnedOwners: cfg.Owners,
		owners:       cfg.Owners,
		focused:      true,
		loading:      true,
		theme:        NewTheme(true),
		s:            s,
		spinner:      sp,
		filter:       fi,
		help:         h,
		detail:       vp,
		keys:         DefaultKeyMap(s),
		filterKeys:   DefaultFilterKeyMap(s),
		now:          time.Now(),
	}
}

// selected returns the pull request under the cursor.
func (m Model) selected() (gh.PR, bool) {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return gh.PR{}, false
	}
	return m.visible[m.cursor], true
}

// counts returns how many pull requests each tab would show under the current
// filter, so the tab bar can carry a badge.
func (m Model) counts() map[tabID]int {
	out := make(map[tabID]int, len(allTabs))
	for _, pr := range m.prs {
		if !matchesFilter(pr, m.filter.Value()) {
			continue
		}
		for _, t := range allTabs {
			if t.keep(pr, m.me) {
				out[t]++
			}
		}
	}
	return out
}

// recompute rebuilds the visible slice from the current tab and filter,
// keeping the cursor on the same pull request when it survives.
func (m *Model) recompute() {
	var anchor string
	if pr, ok := m.selected(); ok {
		anchor = pr.Key()
	}

	m.visible = m.visible[:0]
	for _, pr := range m.prs {
		if m.tab.keep(pr, m.me) && matchesFilter(pr, m.filter.Value()) {
			m.visible = append(m.visible, pr)
		}
	}

	m.cursor = 0
	if anchor != "" {
		for i, pr := range m.visible {
			if pr.Key() == anchor {
				m.cursor = i
				break
			}
		}
	}
	m.clampCursor()
	m.syncDetail()
}

// clampCursor keeps the cursor and the scroll offset inside the list.
func (m *Model) clampCursor() {
	if len(m.visible) == 0 {
		m.cursor, m.offset = 0, 0
		return
	}
	m.cursor = min(max(m.cursor, 0), len(m.visible)-1)

	rows := m.metrics().rows
	if rows <= 0 {
		m.offset = 0
		return
	}
	m.offset = min(m.offset, m.cursor)
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	m.offset = min(max(m.offset, 0), max(len(m.visible)-rows, 0))
}

// syncDetail refreshes the detail viewport when the selection changes. The
// content is only rebuilt on an actual change so scroll position survives a
// background refresh.
func (m *Model) syncDetail() {
	pr, ok := m.selected()
	if !ok {
		m.detailKey = ""
		m.detail.SetContent("")
		return
	}
	if pr.Key() == m.detailKey {
		return
	}
	m.detailKey = pr.Key()
	m.detail.SetContent(m.detailContent(pr))
	m.detail.GotoTop()
}

// setFlash shows a transient one-line message in the status area.
func (m *Model) setFlash(s string) {
	m.flash = s
	m.flashTill = m.now.Add(2 * time.Second)
}

// dropExpiredFarewells removes goodbye banners that have had their moment.
func (m *Model) dropExpiredFarewells() {
	if len(m.farewells) == 0 {
		return
	}
	kept := m.farewells[:0]
	for _, f := range m.farewells {
		if m.now.Sub(f.born) < farewellLife {
			kept = append(kept, f)
		}
	}
	m.farewells = kept
}

// nextFetchIn is how long until the next automatic refresh. It is negative
// once a refresh is due, and meaningless while the terminal is unfocused
// (polling is paused then).
func (m Model) nextFetchIn() time.Duration {
	if m.lastFetch.IsZero() {
		return 0
	}
	return m.interval - m.now.Sub(m.lastFetch)
}

// metrics is the resolved layout for the current terminal size and state.
type metrics struct {
	innerW, innerH int
	listW, listH   int
	detailW        int
	rowLines       int
	rows           int
	splitDetail    bool
	fullDetail     bool
}

// metrics computes the layout. It is a pure function of the model so that
// Update (which clamps scrolling) and View (which draws) can never disagree
// about how many rows fit.
func (m Model) metrics() metrics {
	mt := metrics{
		innerW: m.innerWidth(),
		innerH: max(m.height-2, 6),
	}

	// header + rule + tabs + blank, then the footer block and its blank line.
	chrome := 4 + 1 + m.footerHeight()
	if n := len(m.farewells); n > 0 {
		chrome += n + 1
	}
	mt.listH = max(mt.innerH-chrome, 1)

	mt.rowLines = roomyRowLines
	if mt.listH < compactHeight {
		mt.rowLines = compactRowLines
	}
	mt.rows = max(mt.listH/mt.rowLines, 1)

	mt.listW = mt.innerW
	switch {
	case m.detailOpen && mt.innerW >= sideBySideWidth:
		mt.splitDetail = true
		mt.listW = mt.innerW/2 - 1
		mt.detailW = mt.innerW - mt.listW - 3
	case m.detailOpen:
		mt.fullDetail = true
		mt.detailW = mt.innerW
		mt.listW = 0
	}
	return mt
}

// innerWidth is the drawable width inside the frame: the terminal less the
// border and the frame's horizontal padding.
//
// It is computed here rather than read off metrics because footerView needs it
// and metrics needs footerView; deriving it from m.width alone keeps that from
// becoming a cycle.
func (m Model) innerWidth() int {
	return max(m.width-4, 20)
}

// footerHeight measures the rendered footer so metrics can reserve exactly the
// lines it will occupy.
func (m Model) footerHeight() int {
	return lipgloss.Height(m.footerView())
}
