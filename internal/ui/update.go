package ui

import (
	"slices"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/kanywst/prpr/internal/gh"
)

// maxFarewells caps the goodbye band so a batch merge cannot swallow the list.
const maxFarewells = 4

// Init starts owner discovery, the animation ticker, and the background color
// query that decides the palette.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		m.spinner.Tick,
		tickCmd(),
		m.refreshCmd(),
	)
}

// Update handles a single message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applySize()
		return m, nil

	case tea.BackgroundColorMsg:
		m.applyTheme(msg.IsDark())
		return m, nil

	case tea.FocusMsg:
		// Polling pauses while the terminal is in the background; catch up as
		// soon as it comes back.
		m.focused = true
		return m, nil

	case tea.BlurMsg:
		m.focused = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		return m.handleTick(time.Time(msg))

	case ownersMsg:
		// A pinned-owner start could not confirm the cached login. Keep the
		// cached list and login as they are, search the pinned owners anyway,
		// and try confirming again on the next refresh.
		if msg.me == "" && len(m.pinnedOwners) > 0 {
			return m.startRefresh(m.fetchCmd())
		}
		m.me = msg.me
		// Discovery also runs for pinned owners when the cache needs its
		// login confirmed, and must not replace the owners that were asked
		// for.
		if len(msg.owners) > 0 && len(m.pinnedOwners) == 0 {
			m.owners = msg.owners
		}
		m.unverified = false
		m.dropUncovered()
		m.recompute()
		return m.startRefresh(m.fetchCmd())

	case meMsg:
		if msg.me != "" {
			m.me = msg.me
			m.recompute()
		}
		return m, nil

	case prsMsg:
		return m.handlePRs(msg)

	case goneMsg:
		return m.handleGone(msg)

	case errMsg:
		m.loading = false
		m.lastErr = msg.err
		// Space the retry out by a full interval instead of hammering a
		// failing endpoint every tick.
		m.lastFetch = time.Now()
		return m, nil

	case noticeMsg:
		m.setFlash(msg.text)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		return m.handleWheel(msg)

	case tea.MouseClickMsg:
		return m.handleClick(msg)
	}

	return m, nil
}

// handleTick advances animations and fires the automatic refresh when due.
func (m Model) handleTick(now time.Time) (tea.Model, tea.Cmd) {
	m.now = now
	before := len(m.farewells)
	m.dropExpiredFarewells()
	if m.flash != "" && m.now.After(m.flashTill) {
		m.flash = ""
	}
	if before != len(m.farewells) {
		m.applySize()
	}

	if !m.refreshDue() {
		return m, tickCmd()
	}
	m, cmd := m.startRefresh(m.refreshCmd())
	return m, tea.Batch(tickCmd(), cmd)
}

// startRefresh marks a refresh as in flight, unless there is nothing to run:
// with every owner excluded and the authored and review-request searches off,
// there are no scopes, and waiting on a fetch that never started would leave
// the spinner going forever. That case settles as an empty, finished refresh.
func (m Model) startRefresh(cmd tea.Cmd) (Model, tea.Cmd) {
	if cmd == nil {
		m.loading = false
		m.ready = true
		m.lastFetch = m.now
		m.cachedAt = time.Time{}
		return m, m.saveCmd(nil, m.now)
	}
	m.loading = true
	return m, cmd
}

// refreshDue reports whether the automatic refresh should fire now.
func (m Model) refreshDue() bool {
	switch {
	case m.loading, !m.focused, m.lastFetch.IsZero():
		return false
	default:
		return m.nextFetchIn() <= 0
	}
}

// handlePRs installs a completed refresh and asks about anything that
// disappeared since the previous one.
//
// A pull request missing from the refresh is not necessarily gone: a scope
// that covers it may have failed, in which case it is carried over from the
// previous list, or it may have been pushed past a scope's page cap, in which
// case it only earns a farewell if it really did close. One held only by the
// review-request search just means the review was done, and leaves quietly.
//
// The previous list may be the cached one from the last run, in which case
// whatever was merged while prpr was not running gets its farewell now. Only
// as many are looked up as the farewell band can show: a cache from weeks ago
// would otherwise fire a lookup for every pull request closed since.
func (m Model) handlePRs(msg prsMsg) (tea.Model, tea.Cmd) {
	prs := slices.Clone(msg.res.PRs)
	lookups := len(m.prs)
	if !m.cachedAt.IsZero() {
		lookups = maxFarewells
	}

	var cmds []tea.Cmd
	fresh := make(map[string]bool, len(prs))
	for _, pr := range prs {
		fresh[pr.Key()] = true
	}
	carried := false
	for _, old := range m.prs {
		switch {
		case fresh[old.Key()]:
		case msg.res.Unsure(old):
			prs = append(prs, old)
			carried = true
		case msg.res.ReviewOnly(old) && !msg.res.Capped(old):
		case len(cmds) < lookups:
			cmds = append(cmds, m.stateCmd(old, msg.res.Capped(old)))
		}
	}
	if carried {
		gh.SortPRs(prs)
	}

	m.loading = false
	m.ready = true
	m.lastErr = nil
	m.lastFetch = msg.at
	m.cachedAt = time.Time{}
	m.prs = prs
	m.outcomes = msg.res.Outcomes
	m.recompute()
	m.applySize()

	cmds = append(cmds, m.saveCmd(prs, msg.at))
	return m, tea.Batch(cmds...)
}

// handleGone turns a vanished pull request into a goodbye banner.
func (m Model) handleGone(msg goneMsg) (tea.Model, tea.Cmd) {
	if msg.capped && msg.state == gh.StateOpen {
		return m, nil
	}
	m.farewells = append(m.farewells, farewell{pr: msg.pr, state: msg.state, born: m.now})
	if len(m.farewells) > maxFarewells {
		m.farewells = m.farewells[len(m.farewells)-maxFarewells:]
	}
	m.applySize()
	return m, nil
}

// handleKey dispatches a key press to filter mode or list mode.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeFilter {
		return m.handleFilterKey(msg)
	}
	return m.handleListKey(msg)
}

// handleFilterKey drives the filter input. Everything that is not an explicit
// accept or cancel is text, and belongs to the input.
func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.filterKeys.Cancel):
		m.mode = modeList
		m.filter.Blur()
		m.filter.SetValue("")
		m.recompute()
		m.applySize()
		return m, nil

	case key.Matches(msg, m.filterKeys.Accept):
		m.mode = modeList
		m.filter.Blur()
		m.applySize()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.recompute()
	return m, cmd
}

// handleListKey drives the list.
func (m Model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	mt := m.metrics()

	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quit = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Suspend):
		return m, tea.Suspend

	case key.Matches(msg, m.keys.Up):
		m.cursor--
		m.clampCursor()
		m.syncDetail()

	case key.Matches(msg, m.keys.Down):
		m.cursor++
		m.clampCursor()
		m.syncDetail()

	case key.Matches(msg, m.keys.Top):
		m.cursor = 0
		m.clampCursor()
		m.syncDetail()

	case key.Matches(msg, m.keys.Bottom):
		m.cursor = len(m.visible) - 1
		m.clampCursor()
		m.syncDetail()

	case key.Matches(msg, m.keys.PageUp):
		m.cursor -= mt.rows
		m.clampCursor()
		m.syncDetail()

	case key.Matches(msg, m.keys.PageDown):
		m.cursor += mt.rows
		m.clampCursor()
		m.syncDetail()

	case key.Matches(msg, m.keys.NextTab):
		m.tab = m.tab.next()
		m.offset = 0
		m.recompute()

	case key.Matches(msg, m.keys.PrevTab):
		m.tab = m.tab.prev()
		m.offset = 0
		m.recompute()

	case key.Matches(msg, m.keys.Open):
		if pr, ok := m.selected(); ok {
			return m, m.openCmd(pr.URL)
		}

	case key.Matches(msg, m.keys.Copy):
		if pr, ok := m.selected(); ok {
			return m, m.copyCmd(pr.URL)
		}

	case key.Matches(msg, m.keys.Detail):
		m.detailOpen = !m.detailOpen
		m.applySize()

	case key.Matches(msg, m.keys.DetailUp):
		m.detail.HalfPageUp()

	case key.Matches(msg, m.keys.DetailDown):
		m.detail.HalfPageDown()

	case key.Matches(msg, m.keys.Refresh):
		if !m.loading {
			return m.startRefresh(m.refreshCmd())
		}

	case key.Matches(msg, m.keys.Filter):
		m.mode = modeFilter
		m.applySize()
		return m, m.filter.Focus()

	case key.Matches(msg, m.keys.ClearFilter):
		// With no filter to clear this is a deliberate no-op rather than a
		// quit: esc is the key people press to back out of something, and
		// having it exit the program is a footgun.
		if m.filter.Value() != "" {
			m.filter.SetValue("")
			m.recompute()
			m.applySize()
		}

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.applySize()
	}

	return m, nil
}

// handleWheel scrolls the detail pane when the pointer is over it, and the
// list otherwise.
func (m Model) handleWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	mt := m.metrics()
	mouse := msg.Mouse()
	overDetail := mt.fullDetail || (mt.splitDetail && mouse.X > mt.listW+2)

	switch mouse.Button {
	case tea.MouseWheelUp:
		if overDetail {
			m.detail.ScrollUp(3)
			return m, nil
		}
		m.cursor--
	case tea.MouseWheelDown:
		if overDetail {
			m.detail.ScrollDown(3)
			return m, nil
		}
		m.cursor++
	default:
		return m, nil
	}

	m.clampCursor()
	m.syncDetail()
	return m, nil
}

// handleClick moves the cursor to the clicked row.
func (m Model) handleClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()
	if mouse.Button != tea.MouseLeft {
		return m, nil
	}
	mt := m.metrics()
	if mt.fullDetail {
		return m, nil
	}

	row := mouse.Y - m.listTop()
	if row < 0 || row >= mt.listH {
		return m, nil
	}
	idx := m.offset + row/mt.rowLines
	if idx < 0 || idx >= len(m.visible) {
		return m, nil
	}

	m.cursor = idx
	m.clampCursor()
	m.syncDetail()
	return m, nil
}

// listTop is the terminal row the first list entry is drawn on. It mirrors
// the stacking order in View.
func (m Model) listTop() int {
	// border, header, rule, tabs, blank
	top := 1 + 4
	if n := len(m.farewells); n > 0 {
		top += n + 1
	}
	return top
}

// applyTheme rebuilds every style for the detected background and pushes the
// palette into the bubbles that own their own styling.
func (m *Model) applyTheme(dark bool) {
	m.theme = NewTheme(dark)
	m.help.Styles = m.theme.HelpStyles()
	m.filter.SetStyles(textinput.DefaultStyles(dark))
	m.spinner.Style = m.theme.Logo
}

// applySize propagates the current layout into the child bubbles and re-clamps
// scrolling. It must run after anything that changes the layout, including the
// farewell band growing or shrinking.
func (m *Model) applySize() {
	mt := m.metrics()
	m.help.SetWidth(mt.innerW)
	m.filter.SetWidth(max(mt.innerW-16, 10))
	if m.detailOpen {
		m.detail.SetWidth(max(mt.detailW, 10))
		m.detail.SetHeight(max(mt.listH, 1))
		// Detail content is wrapped to the pane width, so a resize needs a
		// rebuild rather than just a reflow.
		if pr, ok := m.selected(); ok {
			m.detail.SetContent(m.detailContent(pr))
		}
	}
	m.clampCursor()
}
