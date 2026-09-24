package gh

import (
	"encoding/json"
	"testing"
	"time"
)

// decodeNode parses a GraphQL search hit the way the client does, so the tests
// exercise the real JSON shape rather than a hand-built struct.
func decodeNode(t *testing.T, raw string) searchNode {
	t.Helper()
	var n searchNode
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return n
}

func TestToPRFullNode(t *testing.T) {
	n := decodeNode(t, `{
		"number": 128,
		"title": "api: add rate limiter",
		"bodyText": "hello",
		"url": "https://github.com/0-draft/api/pull/128",
		"isDraft": false,
		"createdAt": "2026-09-10T09:00:00Z",
		"updatedAt": "2026-09-12T09:00:00Z",
		"additions": 142,
		"deletions": 9,
		"changedFiles": 5,
		"reviewDecision": "APPROVED",
		"headRefName": "feat/rate-limiter",
		"baseRefName": "main",
		"comments": {"totalCount": 3},
		"author": {"login": "kanywst"},
		"repository": {"nameWithOwner": "0-draft/api"},
		"labels": {"nodes": [{"name": "enhancement"}, {"name": ""}]},
		"reviewRequests": {"nodes": [
			{"requestedReviewer": {"__typename": "User", "login": "alice"}},
			{"requestedReviewer": {"__typename": "Team", "slug": "platform"}},
			{"requestedReviewer": null}
		]},
		"commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "SUCCESS"}}}]}
	}`)

	pr, ok := n.toPR()
	if !ok {
		t.Fatal("toPR returned ok=false for a complete node")
	}

	if got, want := pr.Key(), "0-draft/api#128"; got != want {
		t.Errorf("Key() = %q, want %q", got, want)
	}
	if got, want := pr.Owner(), "0-draft"; got != want {
		t.Errorf("Owner() = %q, want %q", got, want)
	}
	if got, want := pr.Name(), "api"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
	if pr.Check != CheckSuccess {
		t.Errorf("Check = %q, want %q", pr.Check, CheckSuccess)
	}
	if pr.Review != ReviewApproved {
		t.Errorf("Review = %q, want %q", pr.Review, ReviewApproved)
	}
	if got, want := pr.UpdatedAt, time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("UpdatedAt = %v, want %v", got, want)
	}
	// Empty label names are dropped; users and teams both become reviewers.
	if got, want := len(pr.Labels), 1; got != want {
		t.Errorf("len(Labels) = %d, want %d (%v)", got, want, pr.Labels)
	}
	if got, want := len(pr.Reviewers), 2; got != want {
		t.Errorf("len(Reviewers) = %d, want %d (%v)", got, want, pr.Reviewers)
	}
}

func TestToPRNullableFields(t *testing.T) {
	// A PR whose author was deleted, in a repo with no review requirement and
	// no CI configured: every optional field comes back null.
	n := decodeNode(t, `{
		"number": 7,
		"title": "chore: tidy",
		"url": "https://github.com/kanywst/x/pull/7",
		"updatedAt": "2026-09-12T09:00:00Z",
		"reviewDecision": null,
		"author": null,
		"repository": {"nameWithOwner": "kanywst/x"},
		"commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}
	}`)

	pr, ok := n.toPR()
	if !ok {
		t.Fatal("toPR returned ok=false")
	}
	if pr.Author != "" {
		t.Errorf("Author = %q, want empty", pr.Author)
	}
	if pr.Review != ReviewNone {
		t.Errorf("Review = %q, want empty", pr.Review)
	}
	if pr.Check != CheckNone {
		t.Errorf("Check = %q, want empty", pr.Check)
	}
	if !pr.CreatedAt.IsZero() {
		t.Errorf("CreatedAt = %v, want zero", pr.CreatedAt)
	}
}

func TestToPRSkipsNonPullRequests(t *testing.T) {
	// search(type: ISSUE) also returns issues, which match none of the
	// PullRequest inline fragment and so arrive zero-valued.
	if _, ok := decodeNode(t, `{}`).toPR(); ok {
		t.Error("toPR accepted an empty node")
	}
	if _, ok := decodeNode(t, `{"number": 1}`).toPR(); ok {
		t.Error("toPR accepted a node with no repository")
	}
}

func TestParseTimeRejectsGarbage(t *testing.T) {
	if got := parseTime("not a time"); !got.IsZero() {
		t.Errorf("parseTime(garbage) = %v, want zero", got)
	}
	if got := parseTime(""); !got.IsZero() {
		t.Errorf("parseTime(\"\") = %v, want zero", got)
	}
}

func TestToPRMarksBots(t *testing.T) {
	for _, tt := range []struct {
		author string
		want   bool
	}{
		{`{"__typename": "Bot", "login": "dependabot"}`, true},
		{`{"__typename": "User", "login": "kanywst"}`, false},
		{`null`, false},
	} {
		n := decodeNode(t, `{"number": 1, "repository": {"nameWithOwner": "o/r"}, "author": `+tt.author+`}`)
		pr, ok := n.toPR()
		if !ok {
			t.Fatalf("toPR(%s) returned ok=false", tt.author)
		}
		if pr.IsBot != tt.want {
			t.Errorf("toPR(%s).IsBot = %v, want %v", tt.author, pr.IsBot, tt.want)
		}
	}
}
