package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kanywst/prpr/internal/gh"
)

// loadedModel returns a model sized to w x h with the sample pull requests in
// it, ready to render.
func loadedModel(t *testing.T, w, h int, prs []gh.PR) Model {
	t.Helper()
	now := time.Now()
	m := New(Config{Fetcher: &fakeFetcher{me: "kanywst"}, Interval: time.Minute, Timeout: time.Second})
	m.applyTheme(true)
	m.now = now
	m, _ = step(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst", "0-draft"}})
	m, _ = step(t, m, prsMsg{prs: prs, at: now})
	return m
}

// assertFits checks that rendered output stays inside the terminal box, which
// is the failure mode that wide runes and emoji cause most often.
func assertFits(t *testing.T, out string, w, h int) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) > h {
		t.Errorf("rendered %d lines, want at most %d", len(lines), h)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got > w {
			t.Errorf("line %d is %d cells wide, want at most %d:\n%s", i, got, w, line)
		}
	}
}

func TestRenderFitsTerminal(t *testing.T) {
	sizes := []struct{ w, h int }{
		{120, 40}, // roomy
		{80, 24},  // classic
		{60, 16},  // narrow, forces compact rows
		{200, 60}, // very wide
	}
	for _, s := range sizes {
		m := loadedModel(t, s.w, s.h, samplePRs(time.Now()))
		assertFits(t, m.render(), s.w, s.h)
	}
}

func TestRenderShowsTheEssentials(t *testing.T) {
	m := loadedModel(t, 120, 40, samplePRs(time.Now()))
	out := ansi.Strip(m.render())

	for _, want := range []string{"prpr", "#128", "api: add rate limiter", "すべて", "kanywst"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q:\n%s", want, out)
		}
	}
}

func TestRenderEmptyStateIsPerTab(t *testing.T) {
	m := loadedModel(t, 100, 30, nil)
	if out := ansi.Strip(m.render()); !strings.Contains(out, "おつかれさま") {
		t.Errorf("empty list did not show the all-clear message:\n%s", out)
	}

	m.tab = tabReview
	m.recompute()
	if out := ansi.Strip(m.render()); !strings.Contains(out, "レビュー待ちゼロ") {
		t.Errorf("empty review tab did not show its own message:\n%s", out)
	}
}

func TestRenderFarewellBand(t *testing.T) {
	m := loadedModel(t, 120, 40, samplePRs(time.Now()))
	m, _ = step(t, m, goneMsg{pr: m.prs[0], state: gh.StateMerged})

	out := ansi.Strip(m.render())
	if !strings.Contains(out, "🎉") || !strings.Contains(out, "おめでとう") {
		t.Errorf("farewell band not rendered:\n%s", out)
	}
	assertFits(t, m.render(), 120, 40)
}

func TestRenderDetailPane(t *testing.T) {
	m := loadedModel(t, 140, 40, samplePRs(time.Now()))
	m, _ = step(t, m, tea.KeyPressMsg{Code: 'd', Text: "d"})

	if !m.metrics().splitDetail {
		t.Fatal("a 140-column terminal should show the detail pane beside the list")
	}
	out := ansi.Strip(m.render())
	for _, want := range []string{"feat/rate-limiter", "ブランチ", "差分"} {
		if !strings.Contains(out, want) {
			t.Errorf("detail pane missing %q:\n%s", want, out)
		}
	}
	assertFits(t, m.render(), 140, 40)

	// A narrow terminal gives the detail the whole body instead of splitting.
	narrow := loadedModel(t, 70, 30, samplePRs(time.Now()))
	narrow, _ = step(t, narrow, tea.KeyPressMsg{Code: 'd', Text: "d"})
	if !narrow.metrics().fullDetail {
		t.Error("a 70-column terminal should show the detail full width")
	}
	assertFits(t, narrow.render(), 70, 30)
}

func TestRenderTinyTerminal(t *testing.T) {
	m := loadedModel(t, 30, 8, samplePRs(time.Now()))
	out := ansi.Strip(m.render())
	if !strings.Contains(out, "ちいさすぎる") {
		t.Errorf("tiny terminal did not get the size hint:\n%s", out)
	}
	assertFits(t, m.render(), 30, 8)
}

func TestViewDeclaresTerminalFeatures(t *testing.T) {
	m := loadedModel(t, 120, 40, samplePRs(time.Now()))
	v := m.View()

	if !v.AltScreen {
		t.Error("View did not request the alt screen")
	}
	if v.MouseMode != tea.MouseModeCellMotion {
		t.Error("View did not enable mouse reporting")
	}
	if !v.ReportFocus {
		t.Error("View did not enable focus reporting; polling cannot pause without it")
	}
	if !strings.Contains(v.WindowTitle, "3") {
		t.Errorf("WindowTitle = %q, want the open count in it", v.WindowTitle)
	}
}

// TestDumpRender is a human-readable dump of the UI. Run with -v to eyeball it:
//
//	go test ./internal/ui -run TestDumpRender -v
func TestDumpRender(t *testing.T) {
	m := loadedModel(t, 96, 26, samplePRs(time.Now()))
	m, _ = step(t, m, goneMsg{pr: m.prs[1], state: gh.StateMerged})
	t.Logf("\n%s", m.render())
}
