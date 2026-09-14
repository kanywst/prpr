package gh

import (
	"testing"
	"time"
)

func TestSortPRsNewestFirstAndStable(t *testing.T) {
	base := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	prs := []PR{
		{Repo: "o/b", Number: 2, UpdatedAt: base},
		{Repo: "o/c", Number: 3, UpdatedAt: base.Add(time.Hour)},
		{Repo: "o/a", Number: 1, UpdatedAt: base},
	}

	SortPRs(prs)

	want := []string{"o/c#3", "o/a#1", "o/b#2"}
	for i, w := range want {
		if got := prs[i].Key(); got != w {
			t.Errorf("prs[%d] = %q, want %q", i, got, w)
		}
	}
}

func TestAuthoredBy(t *testing.T) {
	pr := PR{Author: "KanyWst"}

	if !pr.AuthoredBy("kanywst") {
		t.Error("AuthoredBy should be case-insensitive")
	}
	if pr.AuthoredBy("") {
		t.Error("AuthoredBy(\"\") should be false; an unknown viewer matches nothing")
	}
	if pr.AuthoredBy("someone") {
		t.Error("AuthoredBy matched the wrong login")
	}
}

func TestAwaitsReviewFrom(t *testing.T) {
	pr := PR{Reviewers: []string{"Alice", "platform"}}

	if !pr.AwaitsReviewFrom("alice") {
		t.Error("AwaitsReviewFrom should be case-insensitive")
	}
	if !pr.AwaitsReviewFrom("platform") {
		t.Error("team slugs should count as review requests")
	}
	if pr.AwaitsReviewFrom("") {
		t.Error("AwaitsReviewFrom(\"\") should be false")
	}
	if pr.AwaitsReviewFrom("bob") {
		t.Error("AwaitsReviewFrom matched a non-reviewer")
	}
}

func TestOwnerNameOnMalformedRepo(t *testing.T) {
	pr := PR{Repo: "weird"}

	if got := pr.Owner(); got != "weird" {
		t.Errorf("Owner() = %q, want %q", got, "weird")
	}
	if got := pr.Name(); got != "" {
		t.Errorf("Name() = %q, want empty", got)
	}
}
