package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kanywst/prpr/internal/gh"
)

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
	return fmt.Sprintf("prpr · オープン %d 件", len(m.prs))
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
	msg := m.theme.Empty.Render("🌸 ちいさすぎるかも…\nもうすこし広げてね")
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

// statusView is the right side of the header: what prpr is doing right now.
func (m Model) statusView() string {
	switch {
	case m.flash != "":
		return m.theme.Status.Render("✨ " + m.flash)
	case m.loading:
		return m.spinner.View() + m.theme.Status.Render(" あつめてる…")
	case m.lastErr != nil:
		return m.theme.Error.Render("😿 しっぱい")
	case !m.focused:
		return m.theme.StatusWarm.Render("⏸ 休憩中")
	case m.lastFetch.IsZero():
		return m.theme.Status.Render("⟳ …")
	default:
		left := m.nextFetchIn()
		if left < 0 {
			left = 0
		}
		return m.theme.Status.Render(fmt.Sprintf("⟳ %ds", int(left.Seconds())+1))
	}
}

// ruleView is the horizontal divider under the header.
func (m Model) ruleView(mt metrics) string {
	return m.theme.Rule.Render(strings.Repeat("─", mt.innerW))
}

// tabsView is the tab bar, with a live count per tab and the active filter.
func (m Model) tabsView(mt metrics) string {
	counts := m.counts()

	parts := make([]string, 0, len(allTabs))
	for _, t := range allTabs {
		label, count := t.String(), fmt.Sprintf(" %d", counts[t])
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
		icon, word := stateWord(f.state)
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
		return m.centered(width, mt.listH, m.theme.Empty.Render("えらばれてないよ"))
	}
	return m.detail.View()
}

// listView renders the pull request rows, or whatever stands in for them.
func (m Model) listView(mt metrics, width int) string {
	switch {
	case !m.ready && m.lastErr != nil:
		return m.centered(width, mt.listH, m.theme.Error.Render("😿 "+m.lastErr.Error()+"\n\nr でもう一回"))
	case !m.ready:
		return m.centered(width, mt.listH, m.theme.Empty.Render("PR あつめてるよ…"))
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
		return "🔍 みつからなかった\n\nesc で絞り込み解除"
	}
	switch m.tab {
	case tabMine:
		return "✨ 自分の PR はないよ〜 ✨"
	case tabReview:
		return "✨ レビュー待ちゼロ! えらい ✨"
	case tabDraft:
		return "✨ 下書きはないよ ✨"
	default:
		return "✨ PR ないよ〜 おつかれさま ✨"
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

	icons := checkIcon(pr.Check, pr.IsDraft)
	if r := reviewIcon(pr.Review); r != "" {
		icons += r
	}
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
	meta := strings.Join([]string{
		"👤 " + author,
		"⏱ " + humanAge(m.now.Sub(pr.UpdatedAt)),
		"📈 " + m.theme.Additions.Render(fmt.Sprintf("+%d", pr.Additions)) +
			"/" + m.theme.Deletions.Render(fmt.Sprintf("-%d", pr.Deletions)),
		"💬 " + fmt.Sprintf("%d", pr.Comments),
	}, "  ")
	repo := m.theme.RepoTag.Render(truncate(pr.Repo, max(width/3, 10)))

	metaLine := "     " + metaStyle.Render(meta)
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

	state := checkIcon(pr.Check, pr.IsDraft) + " " + checkWord(pr.Check, pr.IsDraft)
	if r := reviewIcon(pr.Review); r != "" {
		state += "   " + r + " " + reviewWord(pr.Review)
	}
	row("状態", state)
	row("作者", pr.Author)
	row("更新", humanAge(m.now.Sub(pr.UpdatedAt))+"前")
	row("作成", humanAge(m.now.Sub(pr.CreatedAt))+"前")
	row("ブランチ", pr.HeadRef+" → "+pr.BaseRef)
	row("差分", fmt.Sprintf("+%d / -%d  (%d ファイル)", pr.Additions, pr.Deletions, pr.ChangedFiles))
	row("コメント", fmt.Sprintf("%d", pr.Comments))
	if len(pr.Labels) > 0 {
		row("ラベル", strings.Join(pr.Labels, ", "))
	}
	if len(pr.Reviewers) > 0 {
		row("レビュー依頼", strings.Join(pr.Reviewers, ", "))
	}
	row("URL", pr.URL)

	if body := strings.TrimSpace(pr.Body); body != "" {
		b.WriteString("\n" + t.Rule.Render(strings.Repeat("─", width)) + "\n\n")
		b.WriteString(t.DetailBody.Render(lipgloss.Wrap(body, width, "")))
	}
	return b.String()
}

// checkWord spells out a rolled-up CI state.
func checkWord(c gh.Check, isDraft bool) string {
	if isDraft {
		return "下書き"
	}
	switch c {
	case gh.CheckSuccess:
		return "CI 通過"
	case gh.CheckPending, gh.CheckExpected:
		return "CI 実行中"
	case gh.CheckFailure, gh.CheckError:
		return "CI 失敗"
	default:
		return "CI なし"
	}
}

// reviewWord spells out a review decision.
func reviewWord(r gh.Review) string {
	switch r {
	case gh.ReviewApproved:
		return "承認済み"
	case gh.ReviewChanges:
		return "変更依頼"
	case gh.ReviewRequired:
		return "レビュー待ち"
	default:
		return ""
	}
}

// footerView is the filter input while filtering, and the help otherwise.
func (m Model) footerView() string {
	if m.mode == modeFilter {
		return m.theme.FilterIcon.Render(filterPrefix) + m.filter.View()
	}
	return m.help.View(m.keys)
}

// centered places content in the middle of a width x height box.
func (m Model) centered(width, height int, content string) string {
	return lipgloss.Place(max(width, 1), max(height, 1), lipgloss.Center, lipgloss.Center, content)
}
