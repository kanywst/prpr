package ui

import "github.com/kanywst/prpr/internal/gh"

// tabID is one of the list's saved views. The zero value is the "everything"
// tab, so a fresh Model starts on it without extra initialization.
type tabID int

const (
	tabAll tabID = iota
	tabMine
	tabReview
	tabDraft
)

// allTabs is the tab bar's order.
var allTabs = []tabID{tabAll, tabMine, tabReview, tabDraft}

// label returns the tab's display name in the active language.
func (t tabID) label(s Strings) string {
	switch t {
	case tabAll:
		return s.TabAll
	case tabMine:
		return s.TabMine
	case tabReview:
		return s.TabReview
	case tabDraft:
		return s.TabDraft
	default:
		return s.TabAll
	}
}

// keep reports whether a pull request belongs in this tab. me is the viewer's
// login; when it is unknown the identity-based tabs simply stay empty rather
// than guessing.
func (t tabID) keep(pr gh.PR, me string) bool {
	switch t {
	case tabAll:
		return true
	case tabMine:
		return pr.AuthoredBy(me)
	case tabReview:
		return pr.AwaitsReviewFrom(me)
	case tabDraft:
		return pr.IsDraft
	default:
		return true
	}
}

// next returns the following tab, wrapping around.
func (t tabID) next() tabID {
	return tabID((int(t) + 1) % len(allTabs))
}

// prev returns the preceding tab, wrapping around.
func (t tabID) prev() tabID {
	return tabID((int(t) - 1 + len(allTabs)) % len(allTabs))
}
