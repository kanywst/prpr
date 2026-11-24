package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kanywst/prpr/internal/gh"
)

// agedPRs is samplePRs with creation times that disagree with the update
// order: #127 was opened first, #12 last.
func agedPRs(now time.Time) []gh.PR {
	prs := samplePRs(now)
	prs[0].CreatedAt = now.Add(-10 * 24 * time.Hour) // #128
	prs[1].CreatedAt = now.Add(-30 * 24 * time.Hour) // #127
	prs[2].CreatedAt = now.Add(-8 * 24 * time.Hour)  // #12
	return prs
}

func numbers(prs []gh.PR) []int {
	out := make([]int, len(prs))
	for i, pr := range prs {
		out[i] = pr.Number
	}
	return out
}

func TestSortKeyCyclesOrders(t *testing.T) {
	now := time.Now()
	m := loadedModel(t, 120, 40, agedPRs(now))
	press := func(m Model) Model {
		m, _ = step(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
		return m
	}

	if got, want := numbers(m.visible), []int{128, 127, 12}; !slices.Equal(got, want) {
		t.Fatalf("default order = %v, want %v", got, want)
	}
	if out := ansi.Strip(m.render()); strings.Contains(out, "↕") {
		t.Error("the default order got a label; it should have none")
	}

	m.cursor = 2 // #12
	m = press(m)
	if got, want := numbers(m.visible), []int{127, 128, 12}; !slices.Equal(got, want) {
		t.Errorf("oldest first = %v, want %v", got, want)
	}
	if pr, _ := m.selected(); pr.Number != 12 {
		t.Errorf("selection = #%d after sorting, want #12", pr.Number)
	}
	if out := ansi.Strip(m.render()); !strings.Contains(out, "↕ oldest first") {
		t.Errorf("tab row does not name the order:\n%s", out)
	}

	m = press(m)
	// Repository order keeps the updated order inside 0-draft/api.
	if got, want := numbers(m.visible), []int{128, 127, 12}; !slices.Equal(got, want) {
		t.Errorf("by repository = %v, want %v", got, want)
	}
	if out := ansi.Strip(m.render()); !strings.Contains(out, "↕ by repository") {
		t.Errorf("tab row does not name the order:\n%s", out)
	}

	m = press(m)
	if m.order != sortUpdated {
		t.Errorf("order = %v after three presses, want back to the default", m.order)
	}
}

func TestSortSurvivesRefreshAndFilter(t *testing.T) {
	now := time.Now()
	m := loadedModel(t, 120, 40, agedPRs(now))
	m, _ = step(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})

	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: agedPRs(now)}, at: now})
	if got, want := numbers(m.visible), []int{127, 128, 12}; !slices.Equal(got, want) {
		t.Errorf("after a refresh = %v, want %v", got, want)
	}
	// The underlying list stays in updated order for merge and arrival diffs.
	if got, want := numbers(m.prs), []int{128, 127, 12}; !slices.Equal(got, want) {
		t.Errorf("m.prs = %v, want the fetcher's order %v", got, want)
	}

	m.filter.SetValue("api")
	m.recompute()
	if got, want := numbers(m.visible), []int{127, 128}; !slices.Equal(got, want) {
		t.Errorf("filtered = %v, want %v", got, want)
	}
}

func TestSortLabelFitsANarrowTabRow(t *testing.T) {
	for _, w := range []int{minWidth, 60} {
		m := loadedModel(t, w, 20, agedPRs(time.Now()))
		m, _ = step(t, m, tea.KeyPressMsg{Code: 's', Text: "s"})
		m, _ = step(t, m, tea.KeyPressMsg{Code: 's', Text: "s"}) // by repository, the longest label
		m.filter.SetValue("a-long-filter-query-that-hits-the-cap")
		m.recompute()
		assertFits(t, m.render(), w, 20)
		// The marks must not squeeze the tabs out of their own row.
		if out := ansi.Strip(m.render()); !strings.Contains(out, "▸ all") {
			t.Errorf("width %d: the active tab was pushed out by the marks:\n%s", w, out)
		}
	}
}

func TestOldestFirstPutsMissingCreationTimesLast(t *testing.T) {
	now := time.Now()
	prs := agedPRs(now)
	prs[1].CreatedAt = time.Time{} // #127, the oldest, lost its timestamp
	sortOldest.sort(prs)
	if got, want := numbers(prs), []int{128, 12, 127}; !slices.Equal(got, want) {
		t.Errorf("oldest first = %v, want %v", got, want)
	}
}

func TestByRepositoryIgnoresCase(t *testing.T) {
	prs := []gh.PR{
		{Number: 1, Repo: "Zeta/app"},
		{Number: 2, Repo: "alpha/app"},
		{Number: 3, Repo: "Beta/app"},
	}
	sortRepo.sort(prs)
	if got, want := numbers(prs), []int{2, 3, 1}; !slices.Equal(got, want) {
		t.Errorf("by repository = %v, want %v", got, want)
	}
}
