package ui

import (
	"cmp"
	"slices"

	"github.com/kanywst/prpr/internal/gh"
)

// sortOrder is how the visible list is ordered. The zero value is the order
// the fetcher returns: most recently updated first.
type sortOrder int

const (
	sortUpdated sortOrder = iota
	// sortOldest puts the longest-open pull requests on top, which is where
	// the forgotten ones are.
	sortOldest
	// sortRepo groups pull requests by repository, keeping the updated order
	// inside each one.
	sortRepo
	sortOrders
)

// next is the order the sort key cycles to.
func (o sortOrder) next() sortOrder { return (o + 1) % sortOrders }

// label names the order for the tab row; the default has none.
func (o sortOrder) label(s Strings) string {
	switch o {
	case sortOldest:
		return s.SortOldest
	case sortRepo:
		return s.SortRepo
	default:
		return ""
	}
}

// sort reorders prs in place. It is stable, so ties keep the updated order
// the list arrived in.
func (o sortOrder) sort(prs []gh.PR) {
	switch o {
	case sortOldest:
		slices.SortStableFunc(prs, func(a, b gh.PR) int { return a.CreatedAt.Compare(b.CreatedAt) })
	case sortRepo:
		slices.SortStableFunc(prs, func(a, b gh.PR) int { return cmp.Compare(a.Repo, b.Repo) })
	}
}
