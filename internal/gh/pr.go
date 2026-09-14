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

// PR is one open pull request, as prpr cares about it.
type PR struct {
	Number       int
	Title        string
	Body         string
	URL          string
	Repo         string // "owner/name"
	Author       string
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
	if login == "" {
		return false
	}
	for _, r := range p.Reviewers {
		if strings.EqualFold(r, login) {
			return true
		}
	}
	return false
}
