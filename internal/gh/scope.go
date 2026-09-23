package gh

import (
	"fmt"
	"strings"
)

// ScopeKind is what a Scope searches by.
type ScopeKind int

// Scope kinds.
const (
	// ScopeOwner is every open pull request in repositories an owner holds.
	ScopeOwner ScopeKind = iota
	// ScopeAuthor is every open pull request a user opened, wherever it is:
	// contributions to other people's projects included.
	ScopeAuthor
	// ScopeReviewRequested is every open pull request that asks a user by
	// name for a review, wherever it is. Requests made to a team are left
	// out, matching what PR.AwaitsReviewFrom counts.
	ScopeReviewRequested
)

// Scope is one search prpr runs on every refresh. A refresh is the union of
// its scopes, and each one succeeds, fails, or hits the page cap on its own.
type Scope struct {
	Kind  ScopeKind
	Login string
}

// OwnerScope searches the repositories that owner holds.
func OwnerScope(owner string) Scope { return Scope{Kind: ScopeOwner, Login: owner} }

// AuthorScope searches the pull requests login opened.
func AuthorScope(login string) Scope { return Scope{Kind: ScopeAuthor, Login: login} }

// ReviewRequestedScope searches the pull requests awaiting login's review.
func ReviewRequestedScope(login string) Scope {
	return Scope{Kind: ScopeReviewRequested, Login: login}
}

// String names the scope for the status line, in search-qualifier form for
// everything but plain owners.
func (s Scope) String() string {
	switch s.Kind {
	case ScopeAuthor:
		return "author:" + s.Login
	case ScopeReviewRequested:
		return "review-requested:" + s.Login
	default:
		return s.Login
	}
}

// query is the search string for the scope.
func (s Scope) query() string {
	qualifier := "user"
	switch s.Kind {
	case ScopeAuthor:
		qualifier = "author"
	case ScopeReviewRequested:
		qualifier = "user-review-requested"
	}
	return fmt.Sprintf("is:pr is:open archived:false %s:%s", qualifier, s.Login)
}

// Covers reports whether pr is something this scope would have returned. The
// UI uses it to decide what a failed or capped scope leaves unexplained: a
// pull request missing from a refresh has only really gone if every scope that
// covers it came back complete.
func (s Scope) Covers(pr PR) bool {
	switch s.Kind {
	case ScopeAuthor:
		return pr.AuthoredBy(s.Login)
	case ScopeReviewRequested:
		return pr.AwaitsReviewFrom(s.Login)
	default:
		return strings.EqualFold(pr.Owner(), s.Login)
	}
}

// Outcome is how a single scope's search went.
type Outcome struct {
	Scope Scope
	// Err is non-nil when the search failed. Its pull requests are missing
	// from the result, not gone.
	Err error
	// Total is how many pull requests matched, which exceeds the number
	// returned when the search hit the page cap.
	Total int
}

// Truncated reports whether the search matched more than it returned.
func (o Outcome) Truncated() bool { return o.Err == nil && o.Total > SearchLimit }

// Result is one refresh: the union of every scope that answered, and how each
// scope went.
type Result struct {
	PRs      []PR
	Outcomes []Outcome
}

// Unsure reports whether pr's absence from this result says nothing about
// whether it is still open, because a scope that covers it failed. Such a
// pull request should be kept from the previous refresh rather than waved off.
func (r Result) Unsure(pr PR) bool {
	for _, o := range r.Outcomes {
		if o.Err != nil && o.Scope.Covers(pr) {
			return true
		}
	}
	return false
}

// Capped reports whether pr may simply have been pushed past the page cap of
// a scope that covers it, rather than having left the open list.
func (r Result) Capped(pr PR) bool {
	for _, o := range r.Outcomes {
		if o.Truncated() && o.Scope.Covers(pr) {
			return true
		}
	}
	return false
}
