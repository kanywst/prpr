package ui

import (
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
)

// allTabs is the tab bar's order.
var allTabs = []tabID{tabAll, tabMine, tabReview, tabElsewhere, tabDraft, tabBots}

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

// label returns the tab's display name in the active language.
func (t tabID) label(s Strings) string {
	switch t {
	case tabAll:
		return s.TabAll
	case tabMine:
		return s.TabMine
	case tabReview:
		return s.TabReview
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
		return pr.AwaitsReviewFrom(v.me)
	case tabElsewhere:
		// Only the authored and review-request searches reach outside the
		// watched owners, so this is your contributions to, and review
		// requests from, everyone else.
		return !pr.IsBot && !v.watches(pr.Owner())
	case tabDraft:
		return !pr.IsBot && pr.IsDraft
	case tabBots:
		return pr.IsBot
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
