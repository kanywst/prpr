// Package gh wraps the handful of GitHub GraphQL calls prpr needs, reusing
// whatever credentials the gh CLI already stores.
package gh

import (
	"fmt"
	"strings"
	"time"
)

// Check is the rolled-up CI state of a pull request's head commit.
type Check string

// Rolled-up check states, as reported by statusCheckRollup.
const (
	CheckNone     Check = ""
	CheckSuccess  Check = "SUCCESS"
	CheckPending  Check = "PENDING"
	CheckFailure  Check = "FAILURE"
	CheckError    Check = "ERROR"
	CheckExpected Check = "EXPECTED"
)

// Review is the aggregate review decision of a pull request.
type Review string

// Review decisions, as reported by reviewDecision. A PR in a repo without
// review requirements reports no decision at all.
const (
	ReviewNone     Review = ""
	ReviewApproved Review = "APPROVED"
	ReviewChanges  Review = "CHANGES_REQUESTED"
	ReviewRequired Review = "REVIEW_REQUIRED"
)

// State is how a pull request left the open list.
type State string

// Pull request states.
const (
	StateOpen   State = "OPEN"
	StateMerged State = "MERGED"
	StateClosed State = "CLOSED"
)

// PR is one open pull request, as prpr cares about it, or an open issue when
// IsIssue is set. Issues ride in the same type because the list, the tabs and
// the farewells treat both alike; the fields only a pull request has (draft,
// diff, branches, checks, review) are simply left empty on an issue.
type PR struct {
	Number int
	Title  string
	Body   string
	URL    string
	Repo   string // "owner/name"
	Author string
	// IsBot is set when the author is a GitHub App (dependabot, renovate,
	// github-actions and the like) rather than a person.
	IsBot        bool
	IsIssue      bool
	IsDraft      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Additions    int
	Deletions    int
	ChangedFiles int
	Comments     int
	HeadRef      string
	BaseRef      string
	Check        Check
	Review       Review
	Reviewers    []string
	Assignees    []string
	Labels       []string
}

// Key uniquely identifies a pull request across repositories.
func (p PR) Key() string { return fmt.Sprintf("%s#%d", p.Repo, p.Number) }

// Owner is the "owner" half of Repo.
func (p PR) Owner() string {
	owner, _, _ := strings.Cut(p.Repo, "/")
	return owner
}

// Name is the "name" half of Repo.
func (p PR) Name() string {
	_, name, _ := strings.Cut(p.Repo, "/")
	return name
}

// AuthoredBy reports whether login opened the pull request.
func (p PR) AuthoredBy(login string) bool {
	return login != "" && strings.EqualFold(p.Author, login)
}

// AwaitsReviewFrom reports whether login has been asked to review.
func (p PR) AwaitsReviewFrom(login string) bool {
	return !p.IsIssue && containsLogin(p.Reviewers, login)
}

// AssignedTo reports whether an issue is assigned to login.
func (p PR) AssignedTo(login string) bool {
	return p.IsIssue && containsLogin(p.Assignees, login)
}

// WaitsOn reports whether the next move is login's: a review asked of them on
// a pull request, or an issue assigned to them.
func (p PR) WaitsOn(login string) bool {
	return p.AwaitsReviewFrom(login) || p.AssignedTo(login)
}

func containsLogin(logins []string, login string) bool {
	if login == "" {
		return false
	}
	for _, l := range logins {
		if strings.EqualFold(l, login) {
			return true
		}
	}
	return false
}
