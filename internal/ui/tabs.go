package ui

import (
	"slices"
	"strings"

	"github.com/kanywst/prpr/internal/gh"
)

// tabID is one of the list's saved views. The zero value is the "everything"
// tab, so a fresh Model starts on it without extra initialization.
type tabID int

const (
	tabAll tabID = iota
	tabMine
	tabReview
	tabElsewhere
	tabDraft
	tabBots
	tabIssues
)

// allTabs is the tab bar's order when issues are shown.
var allTabs = []tabID{tabAll, tabMine, tabReview, tabElsewhere, tabIssues, tabDraft, tabBots}

// tabsFor is the tab bar: every tab, less the issues tab when issues are not
// being searched and it could only ever be empty.
func tabsFor(issues bool) []tabID {
	if issues {
		return allTabs
	}
	out := make([]tabID, 0, len(allTabs)-1)
	for _, t := range allTabs {
		if t != tabIssues {
			out = append(out, t)
		}
	}
	return out
}

// viewer is who is looking at the list: what the identity-based tabs need to
// sort a pull request into place.
type viewer struct {
	me     string
	owners []string
}

// watches reports whether owner is one of the watched owners.
func (v viewer) watches(owner string) bool {
	for _, o := range v.owners {
		if strings.EqualFold(o, owner) {
			return true
		}
	}
	return false
}

// label returns the tab's display name in the active language. With issues
// shown, the review tab also holds the issues assigned to you, and is named for
// that.
func (t tabID) label(s Strings, issues bool) string {
	switch t {
	case tabAll:
		return s.TabAll
	case tabMine:
		return s.TabMine
	case tabReview:
		if issues {
			return s.TabYourTurn
		}
		return s.TabReview
	case tabIssues:
		return s.TabIssues
	case tabElsewhere:
		return s.TabElsewhere
	case tabDraft:
		return s.TabDraft
	case tabBots:
		return s.TabBots
	default:
		return s.TabAll
	}
}

// keep reports whether a pull request belongs in this tab. When the viewer's
// login is unknown the identity-based tabs simply stay empty rather than
// guessing.
//
// Bot pull requests (dependency bumps, mostly) get a tab of their own and are
// kept out of the general views, where a week of dependabot would otherwise
// bury the pull requests people wrote. A review request is the exception: one
// addressed to you by name is yours to act on, whoever opened it.
func (t tabID) keep(pr gh.PR, v viewer) bool {
	switch t {
	case tabAll:
		return !pr.IsBot
	case tabMine:
		return pr.AuthoredBy(v.me)
	case tabReview:
		return pr.WaitsOn(v.me)
	case tabElsewhere:
		// Only the authored and review-request searches reach outside the
		// watched owners, so this is your contributions to, and review
		// requests from, everyone else.
		return !pr.IsBot && !v.watches(pr.Owner())
	case tabDraft:
		return !pr.IsBot && pr.IsDraft
	case tabIssues:
		return !pr.IsBot && pr.IsIssue
	case tabBots:
		return pr.IsBot
	default:
		return true
	}
}

// step returns the tab delta places along tabs, wrapping around.
func (t tabID) step(tabs []tabID, delta int) tabID {
	i := slices.Index(tabs, t)
	if i < 0 {
		return tabs[0]
	}
	return tabs[(i+delta+len(tabs))%len(tabs)]
}
