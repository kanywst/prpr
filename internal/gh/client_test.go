package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

// fakeGraphQL answers search queries by the owner in the query string, so a
// test can make one scope fail or overflow while the others succeed.
type fakeGraphQL map[string]string

func (f fakeGraphQL) RoundTrip(req *http.Request) (*http.Response, error) {
	var body struct {
		Variables struct {
			Q string `json:"q"`
		} `json:"variables"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return nil, err
	}
	payload := `{"data":{"search":{"issueCount":0,"nodes":[]}}}`
	for owner, p := range f {
		if strings.HasSuffix(body.Variables.Q, ":"+owner) {
			payload = p
		}
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(payload)),
		Request:    req,
	}, nil
}

func searchPayload(total int, repos ...string) string {
	nodes := make([]string, 0, len(repos))
	for i, r := range repos {
		nodes = append(nodes, fmt.Sprintf(`{"number":%d,"repository":{"nameWithOwner":%q},"updatedAt":"2026-01-0%dT00:00:00Z"}`, i+1, r, i+1))
	}
	return fmt.Sprintf(`{"data":{"search":{"issueCount":%d,"nodes":[%s]}}}`, total, strings.Join(nodes, ","))
}

func testClient(t *testing.T, f fakeGraphQL) *Client {
	t.Helper()
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test",
		Transport: f,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{gql: gql}
}

func TestSearchKeepsWhatAnsweredWhenAScopeFails(t *testing.T) {
	c := testClient(t, fakeGraphQL{
		"alice": searchPayload(1, "alice/app"),
		"sso":   `{"data":{"search":null},"errors":[{"type":"FORBIDDEN","message":"Resource protected by organization SAML enforcement."}]}`,
		"bob":   searchPayload(1, "bob/lib"),
	})

	res, err := c.Search(context.Background(), []Scope{OwnerScope("alice"), OwnerScope("sso"), OwnerScope("bob")})
	if err != nil {
		t.Fatalf("one failing scope failed the whole refresh: %v", err)
	}
	if len(res.PRs) != 2 {
		t.Fatalf("got %d PRs, want the 2 from the scopes that answered", len(res.PRs))
	}
	if res.Outcomes[1].Err == nil {
		t.Error("the SAML-protected scope did not report its error")
	}
	// Its pull requests are unaccounted for, not gone.
	if !res.Unsure(PR{Repo: "sso/private", Number: 1}) {
		t.Error("a PR under the failed scope was not marked unsure")
	}
	if res.Unsure(PR{Repo: "alice/app", Number: 9}) {
		t.Error("a PR under a scope that answered was marked unsure")
	}
}

func TestSearchFailsWhenEveryScopeFails(t *testing.T) {
	failing := `{"errors":[{"message":"nope"}]}`
	c := testClient(t, fakeGraphQL{"a": failing, "b": failing})

	if _, err := c.Search(context.Background(), []Scope{OwnerScope("a"), OwnerScope("b")}); err == nil {
		t.Fatal("Search succeeded with no scope answering")
	}
}

func TestSearchReportsThePageCap(t *testing.T) {
	c := testClient(t, fakeGraphQL{"big": searchPayload(SearchLimit+40, "big/one")})

	res, err := c.Search(context.Background(), []Scope{OwnerScope("big")})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Outcomes[0].Truncated() {
		t.Error("a search matching more than the cap was not reported as truncated")
	}
	if !res.Capped(PR{Repo: "big/old", Number: 3}) {
		t.Error("a PR under a capped scope was not marked capped")
	}
}

func TestReviewOnly(t *testing.T) {
	res := Result{Outcomes: []Outcome{
		{Scope: OwnerScope("kanywst")},
		{Scope: AuthorScope("kanywst")},
		{Scope: ReviewRequestedScope("kanywst")},
	}}
	outside := PR{Repo: "someone/lib", Number: 1, Author: "alice", Reviewers: []string{"kanywst"}}
	if !res.ReviewOnly(outside) {
		t.Error("an outside PR held only by the review request was not review-only")
	}
	owned := PR{Repo: "kanywst/prpr", Number: 2, Author: "alice", Reviewers: []string{"kanywst"}}
	if res.ReviewOnly(owned) {
		t.Error("a PR under a watched owner was treated as review-only")
	}
	if res.ReviewOnly(PR{Repo: "someone/lib", Number: 3, Author: "alice"}) {
		t.Error("a PR no scope covers was treated as review-only")
	}
}

func TestIssueScopes(t *testing.T) {
	issue := PR{Repo: "0-draft/api", Number: 1, Author: "alice", IsIssue: true, Assignees: []string{"kanywst"}}
	pr := PR{Repo: "0-draft/api", Number: 2, Author: "alice", Reviewers: []string{"kanywst"}}

	for _, tt := range []struct {
		scope Scope
		query string
		name  string
		issue bool
		pr    bool
	}{
		{OwnerScope("0-draft"), "is:pr is:open archived:false user:0-draft", "0-draft", false, true},
		{OwnerScope("0-draft").ForIssues(), "is:issue is:open archived:false user:0-draft", "issues:0-draft", true, false},
		{AuthorScope("alice").ForIssues(), "is:issue is:open archived:false author:alice", "issues:author:alice", true, false},
		{ReviewRequestedScope("kanywst"), "is:pr is:open archived:false user-review-requested:kanywst", "review-requested:kanywst", false, true},
		{ReviewRequestedScope("kanywst").ForIssues(), "is:issue is:open archived:false assignee:kanywst", "assignee:kanywst", true, false},
	} {
		if got := tt.scope.query(); got != tt.query {
			t.Errorf("%v query = %q, want %q", tt.scope, got, tt.query)
		}
		if got := tt.scope.String(); got != tt.name {
			t.Errorf("String() = %q, want %q", got, tt.name)
		}
		if got := tt.scope.Covers(issue); got != tt.issue {
			t.Errorf("%v covers the issue = %v, want %v", tt.scope, got, tt.issue)
		}
		if got := tt.scope.Covers(pr); got != tt.pr {
			t.Errorf("%v covers the PR = %v, want %v", tt.scope, got, tt.pr)
		}
	}
}
