package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kanywst/prpr/internal/gh"
)

// ruleGlyph draws the horizontal dividers. It is the heavy box-drawing line
// rather than the light one: the light glyph is missing from enough terminal
// fonts that the divider silently rendered as a blank row.
const ruleGlyph = "━"

// filterPrefix labels the filter input line.
const filterPrefix = "🔍 "

// minWidth and minHeight are the smallest terminal prpr will try to draw in.
const (
	minWidth  = 46
	minHeight = 12
)

// View renders the whole program. Terminal-level settings (alt screen, mouse,
// focus reporting, window title, cursor) are declared here rather than at
// program construction, which is how Bubble Tea v2 wants them expressed.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.ReportFocus = true
	v.WindowTitle = m.windowTitle()
	v.Cursor = m.cursor2D()
	return v
}

// windowTitle keeps the open count visible even when prpr is in a background
// tab.
func (m Model) windowTitle() string {
	if !m.ready {
		return "prpr"
	}
	return fmt.Sprintf(m.s.OpenCount, len(m.prs))
}

// cursor2D places the real terminal cursor inside the filter input, and hides
// it everywhere else.
func (m Model) cursor2D() *tea.Cursor {
	if m.mode != modeFilter {
		return nil
	}
	c := m.filter.Cursor()
	if c == nil {
		return nil
	}
	// Frame border + horizontal padding, then the filter's own label.
	c.X += 2 + ansi.StringWidth(filterPrefix)
	// The filter occupies the last content row, just above the bottom border.
	c.Y += m.height - 2
	return c
}

// render draws the framed UI.
func (m Model) render() string {
	if m.width < minWidth || m.height < minHeight {
		return m.tooSmallView()
	}

	mt := m.metrics()

	sections := []string{
		m.headerView(mt),
		m.ruleView(mt),
		m.tabsView(mt),
		"",
	}
	if band := m.farewellView(mt); band != "" {
		sections = append(sections, band, "")
	}
	sections = append(sections, m.bodyView(mt), "", m.footerView())

	// The frame is deliberately not given an explicit Width/Height: lipgloss
	// counts the border inside Width, so sizing it by hand double-counts the
	// frame. Instead every section is built to exactly innerW x innerH and the
	// border is grown around it.
	return m.theme.Frame.Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
}

// tooSmallView is what a terminal too small for the layout gets.
func (m Model) tooSmallView() string {
	msg := m.theme.Empty.Render(m.s.TooSmall)
	return lipgloss.Place(max(m.width, 1), max(m.height, 1), lipgloss.Center, lipgloss.Center, msg)
}

// headerView is the logo, the watched owners, and the refresh status.
func (m Model) headerView(mt metrics) string {
	logo := m.theme.Logo.Render("🌸 prpr")
	status := m.statusView()

	room := mt.innerW - ansi.StringWidth(logo) - ansi.StringWidth(status) - 2
	owners := ""
	if room > 4 && len(m.owners) > 0 {
		owners = m.theme.Owners.Render(truncate(strings.Join(m.owners, " · "), room))
	}

	left := logo + "  " + owners
	return pad(left, mt.innerW-ansi.StringWidth(status)) + status
}

// statusView is the right side of the header: what prpr is doing right now,
// and how old the list is while it is still the cached one.
func (m Model) statusView() string {
	status := m.liveStatus()
	if m.cachedAt.IsZero() || m.flash != "" {
		return status
	}
	cached := m.theme.StatusWarm.Render(fmt.Sprintf(m.s.Cached, humanAge(m.now.Sub(m.cachedAt), m.s)))
	return cached + "  " + status
}

// liveStatus is what prpr is doing right now.
func (m Model) liveStatus() string {
	switch {
	case m.flash != "":
		return m.theme.Status.Render("✨ " + m.flash)
	case m.loading:
		return m.spinner.View() + m.theme.Status.Render(m.s.Collecting)
	case m.lastErr != nil:
		return m.theme.Error.Render(m.s.Failed)
	case !m.focused:
		return m.theme.StatusWarm.Render(m.s.Paused)
	case m.lastFetch.IsZero():
		return m.theme.Status.Render("⟳ …")
	default:
		left := m.nextFetchIn()
		if left < 0 {
			left = 0
		}
		countdown := m.theme.Status.Render(fmt.Sprintf("⟳ %ds", int(left.Seconds())+1))
		if warn := m.warning(); warn != "" {
			return m.theme.StatusWarm.Render(truncate(warn, maxWarningWidth)) + "  " + countdown
		}
		return countdown
	}
}

// maxWarningWidth keeps a long list of failed scopes from pushing the owner
// list out of the header entirely.
const maxWarningWidth = 36

// warning describes what the last refresh could not see: scopes that failed,
// or failing that, scopes that hit the page cap. Failures come first because
// they hide whole owners, where a cap only hides the oldest pull requests.
func (m Model) warning() string {
	var failed []string
	var capped *gh.Outcome
	for i, o := range m.outcomes {
		switch {
		case o.Err != nil:
			failed = append(failed, o.Scope.String())
		case o.Truncated() && capped == nil:
			capped = &m.outcomes[i]
		}
	}
	switch {
	case len(failed) > 0:
		return fmt.Sprintf(m.s.WarnFailed, strings.Join(failed, ", "))
	case capped != nil:
		return fmt.Sprintf(m.s.WarnCapped, capped.Scope, gh.SearchLimit, capped.Total)
	default:
		return ""
	}
}

// ruleView is the horizontal divider under the header.
func (m Model) ruleView(mt metrics) string {
	return m.theme.Rule.Render(strings.Repeat(ruleGlyph, mt.innerW))
}

// tabsView is the tab bar, with a live count per tab and the active filter.
func (m Model) tabsView(mt metrics) string {
	counts := m.counts()

	tabs := m.tabs()
	parts := make([]string, 0, len(tabs))
	for _, t := range tabs {
		label, count := t.label(m.s, m.issues), fmt.Sprintf(" %d", counts[t])
		if t == m.tab {
			parts = append(parts, m.theme.TabActive.Render("▸ "+label)+m.theme.TabSelected.Render(count))
		} else {
			parts = append(parts, m.theme.TabIdle.Render(label)+m.theme.TabCount.Render(count))
		}
	}
	left := strings.Join(parts, m.theme.TabDivider.Render("  │  "))

	right := ""
	if q := m.filter.Value(); q != "" && m.mode != modeFilter {
		right = m.theme.FilterIcon.Render("🔍 " + truncate(q, 20))
	}
	if right == "" {
		return truncate(left, mt.innerW)
	}
	return pad(truncate(left, mt.innerW-ansi.StringWidth(right)-1), mt.innerW-ansi.StringWidth(right)) + right
}

// farewellView is the goodbye band: pull requests that just left the list.
func (m Model) farewellView(mt metrics) string {
	if len(m.farewells) == 0 {
		return ""
	}

	sparkles := []string{"✨", "🌟", "💫", "⭐"}
	frame := sparkles[int(m.now.UnixNano()/int64(300*time.Millisecond))%len(sparkles)]

	lines := make([]string, 0, len(m.farewells))
	for _, f := range m.farewells {
		icon, word := stateWord(f.state, m.s)
		style := m.theme.Party
		if m.now.Sub(f.born) > farewellFade {
			style = m.theme.PartyFade
		}
		text := fmt.Sprintf("%s %s %s#%d %s %s",
			frame, icon, f.pr.Repo, f.pr.Number, truncate(f.pr.Title, max(mt.innerW/3, 10)), word)
		lines = append(lines, truncate(style.Render(text), mt.innerW))
	}
	return strings.Join(lines, "\n")
}

// bodyView is the list, the detail pane, or both.
func (m Model) bodyView(mt metrics) string {
	switch {
	case mt.fullDetail:
		return m.detailPane(mt, mt.innerW)
	case mt.splitDetail:
		bar := make([]string, mt.listH)
		for i := range bar {
			bar[i] = m.theme.Rule.Render("│")
		}
		sep := strings.Join(bar, "\n")
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.listView(mt, mt.listW),
			" "+sep+" ",
			m.detailPane(mt, mt.detailW),
		)
	default:
		return m.listView(mt, mt.innerW)
	}
}

// detailPane renders the scrollable detail viewport.
func (m Model) detailPane(mt metrics, width int) string {
	if _, ok := m.selected(); !ok {
		return m.centered(width, mt.listH, m.theme.Empty.Render(m.s.NothingSelected))
	}
	return m.detail.View()
}

// listView renders the pull request rows, or whatever stands in for them.
func (m Model) listView(mt metrics, width int) string {
	switch {
	case !m.ready && m.lastErr != nil:
		return m.centered(width, mt.listH, m.theme.Error.Render(fmt.Sprintf(m.s.ErrorHint, m.lastErr)))
	case !m.ready:
		return m.centered(width, mt.listH, m.theme.Empty.Render(m.s.Loading))
	case len(m.visible) == 0:
		return m.centered(width, mt.listH, m.theme.Empty.Render(m.emptyMessage()))
	}

	// The rightmost column belongs to the scrollbar, so rows are built one
	// cell narrower than the pane.
	rowWidth := width - 1

	lines := make([]string, 0, mt.listH)
	end := min(m.offset+mt.rows, len(m.visible))
	for i := m.offset; i < end; i++ {
		lines = append(lines, m.rowView(m.visible[i], i == m.cursor, rowWidth, mt.rowLines == compactRowLines)...)
		if mt.rowLines == roomyRowLines && i < end-1 {
			lines = append(lines, "")
		}
	}

	// Pad to the full list height so the footer never drifts, and so a split
	// layout lines up with the detail pane.
	for len(lines) < mt.listH {
		lines = append(lines, "")
	}
	lines = lines[:mt.listH]

	scroll := m.scrollbar(mt, len(lines))
	for i := range lines {
		lines[i] = pad(truncate(lines[i], rowWidth), rowWidth) + scroll[i]
	}
	return strings.Join(lines, "\n")
}

// emptyMessage is what an empty tab says, which depends on why it is empty.
func (m Model) emptyMessage() string {
	if m.filter.Value() != "" {
		return m.s.EmptyFilter
	}
	switch m.tab {
	case tabMine:
		return m.s.EmptyMine
	case tabReview:
		if m.issues {
			return m.s.EmptyYourTurn
		}
		return m.s.EmptyReview
	case tabIssues:
		return m.s.EmptyIssues
	case tabElsewhere:
		return m.s.EmptyElsewhere
	case tabDraft:
		return m.s.EmptyDraft
	case tabBots:
		return m.s.EmptyBots
	default:
		return m.s.EmptyAll
	}
}

// scrollbar renders a one-column indicator of where the viewport sits.
func (m Model) scrollbar(mt metrics, height int) []string {
	out := make([]string, height)
	for i := range out {
		out[i] = " "
	}
	if len(m.visible) <= mt.rows || height == 0 {
		return out
	}

	thumb := max(height*mt.rows/len(m.visible), 1)
	span := max(len(m.visible)-mt.rows, 1)
	top := m.offset * (height - thumb) / span
	for i := top; i < min(top+thumb, height); i++ {
		out[i] = m.theme.Scrollbar.Render("▐")
	}
	return out
}

// rowView renders one pull request as either one or three lines.
func (m Model) rowView(pr gh.PR, selected bool, width int, compact bool) []string {
	pointer := "  "
	titleStyle, metaStyle := m.theme.Title, m.theme.Meta
	if selected {
		pointer = m.theme.Pointer.Render("▸ ")
		titleStyle, metaStyle = m.theme.TitleOn, m.theme.MetaOn
	}

	icons := kindIcon(pr)
	number := m.theme.Number.Render(fmt.Sprintf("#%d", pr.Number))

	if compact {
		repo := m.theme.RepoTag.Render(truncate(pr.Repo, max(width/4, 8)))
		head := fmt.Sprintf("%s%s %s ", pointer, icons, number)
		room := width - ansi.StringWidth(head) - ansi.StringWidth(repo) - 2
		title := titleStyle.Render(truncate(pr.Title, max(room, 4)))
		return []string{pad(head+title, width-ansi.StringWidth(repo)-1) + repo}
	}

	head := fmt.Sprintf("%s%s %s ", pointer, number, icons)
	title := titleStyle.Render(truncate(pr.Title, max(width-ansi.StringWidth(head)-1, 4)))

	author := pr.Author
	if author == "" {
		author = "?"
	}
	meta := []string{
		"👤 " + author,
		"⏱ " + humanAge(m.now.Sub(pr.UpdatedAt), m.s),
	}
	if !pr.IsIssue {
		meta = append(meta, "📈 "+m.theme.Additions.Render(fmt.Sprintf("+%d", pr.Additions))+
			"/"+m.theme.Deletions.Render(fmt.Sprintf("-%d", pr.Deletions)))
	}
	meta = append(meta, "💬 "+fmt.Sprintf("%d", pr.Comments))
	repo := m.theme.RepoTag.Render(truncate(pr.Repo, max(width/3, 10)))

	metaLine := "     " + metaStyle.Render(strings.Join(meta, "  "))
	metaLine = pad(truncate(metaLine, width-ansi.StringWidth(repo)-1), width-ansi.StringWidth(repo)) + repo

	return []string{head + title, metaLine}
}

// detailContent builds the text shown in the detail viewport.
func (m Model) detailContent(pr gh.PR) string {
	width := max(m.metrics().detailW, 20)
	t := m.theme

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", t.Number.Render(fmt.Sprintf("#%d", pr.Number)),
		t.TitleOn.Render(truncate(pr.Title, width-8)))
	fmt.Fprintf(&b, "%s\n\n", t.RepoTag.Render(pr.Repo))

	row := func(key, value string) {
		if value == "" {
			return
		}
		fmt.Fprintf(&b, "%s %s\n", t.DetailKey.Render(key), t.Detail.Render(value))
	}

	state := issueIcon + " " + m.s.IssueWord
	if !pr.IsIssue {
		state = checkIcon(pr.Check, pr.IsDraft) + " " + checkWord(pr.Check, pr.IsDraft, m.s)
		if r := reviewIcon(pr.Review); r != "" {
			state += "   " + r + " " + reviewWord(pr.Review, m.s)
		}
	}
	ago := func(at time.Time) string {
		return fmt.Sprintf(m.s.AgeAgo, humanAge(m.now.Sub(at), m.s))
	}
	row(m.s.DetailState, state)
	row(m.s.DetailAuthor, pr.Author)
	row(m.s.DetailUpdated, ago(pr.UpdatedAt))
	row(m.s.DetailCreated, ago(pr.CreatedAt))
	if !pr.IsIssue {
		row(m.s.DetailBranch, pr.HeadRef+" → "+pr.BaseRef)
		row(m.s.DetailDiff, fmt.Sprintf("+%d / -%d  %s",
			pr.Additions, pr.Deletions, fmt.Sprintf(m.s.FilesSuffix, pr.ChangedFiles)))
	}
	row(m.s.DetailComments, strconv.Itoa(pr.Comments))
	if len(pr.Labels) > 0 {
		row(m.s.DetailLabels, strings.Join(pr.Labels, ", "))
	}
	if len(pr.Reviewers) > 0 {
		row(m.s.DetailReviewers, strings.Join(pr.Reviewers, ", "))
	}
	if len(pr.Assignees) > 0 {
		row(m.s.DetailAssignees, strings.Join(pr.Assignees, ", "))
	}
	row(m.s.DetailURL, pr.URL)

	if body := strings.TrimSpace(pr.Body); body != "" {
		b.WriteString("\n" + t.Rule.Render(strings.Repeat(ruleGlyph, width)) + "\n\n")
		b.WriteString(t.DetailBody.Render(lipgloss.Wrap(body, width, "")))
	}
	return b.String()
}

// footerView is the filter input while filtering, and the help otherwise.
//
// Both are clamped here rather than trusted to clamp themselves: the help
// bubble's own truncation does not always respect the width it was given, and
// one over-wide line widens the whole frame past the terminal.
func (m Model) footerView() string {
	var raw string
	if m.mode == modeFilter {
		raw = m.theme.FilterIcon.Render(filterPrefix) + m.filter.View()
	} else {
		raw = m.help.View(m.keys)
	}

	width := m.innerWidth()
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		lines[i] = truncate(line, width)
	}
	return strings.Join(lines, "\n")
}

// centered places content in the middle of a width x height box.
func (m Model) centered(width, height int, content string) string {
	return lipgloss.Place(max(width, 1), max(height, 1), lipgloss.Center, lipgloss.Center, content)
}
