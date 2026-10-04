package gh

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2/pkg/api"
)

// SearchLimit caps how many PRs a single scope contributes. GitHub's search
// API refuses anything above 100 per page, and a dashboard past this size has
// stopped being a dashboard anyway.
const SearchLimit = 60

// Client talks to the GitHub GraphQL API as the logged-in gh user.
type Client struct {
	gql *api.GraphQLClient
}

// New builds a client from the gh CLI's stored credentials.
func New() (*Client, error) {
	c, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("could not read the gh CLI credentials (has gh auth login been run?): %w", err)
	}
	return &Client{gql: c}, nil
}

const viewerQuery = `
query($after: String) {
  viewer {
    login
    organizations(first: 100, after: $after) {
      nodes { login }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type viewerResponse struct {
	Viewer struct {
		Login         string `json:"login"`
		Organizations struct {
			Nodes []struct {
				Login string `json:"login"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"organizations"`
	} `json:"viewer"`
}

// maxOrgPages bounds organization discovery, 100 per page, so a cursor that
// never ends cannot spin forever.
const maxOrgPages = 20

// Viewer returns the authenticated login together with every organization it
// belongs to. prpr uses this to discover which owners to watch, so the tool
// works for anyone who runs it without per-user configuration.
func (c *Client) Viewer(ctx context.Context) (login string, orgs []string, err error) {
	vars := map[string]any{"after": nil}
	for range maxOrgPages {
		var resp viewerResponse
		if err := c.gql.DoWithContext(ctx, viewerQuery, vars, &resp); err != nil {
			return "", nil, fmt.Errorf("could not resolve the logged-in user: %w", err)
		}
		login = resp.Viewer.Login
		for _, n := range resp.Viewer.Organizations.Nodes {
			orgs = append(orgs, n.Login)
		}
		page := resp.Viewer.Organizations.PageInfo
		if !page.HasNextPage || page.EndCursor == "" {
			break
		}
		vars["after"] = page.EndCursor
	}
	sort.Strings(orgs)
	return login, orgs, nil
}

const searchQuery = `
query($q: String!, $limit: Int!) {
  search(query: $q, type: ISSUE, first: $limit) {
    issueCount
    nodes {
      __typename
      ... on Issue {
        number
        title
        bodyText
        url
        createdAt
        updatedAt
        comments { totalCount }
        author { __typename login }
        repository { nameWithOwner }
        labels(first: 10) { nodes { name } }
        assignees(first: 10) { nodes { login } }
      }
      ... on PullRequest {
        number
        title
        bodyText
        url
        isDraft
        createdAt
        updatedAt
        additions
        deletions
        changedFiles
        reviewDecision
        headRefName
        baseRefName
        comments { totalCount }
        author { __typename login }
        repository { nameWithOwner }
        labels(first: 10) { nodes { name } }
        reviewRequests(first: 20) {
          nodes {
            requestedReviewer {
              __typename
              ... on User { login }
              ... on Team { combinedSlug }
            }
          }
        }
        commits(last: 1) {
          nodes { commit { statusCheckRollup { state } } }
        }
      }
    }
  }
}`

type searchResponse struct {
	Search struct {
		IssueCount int          `json:"issueCount"`
		Nodes      []searchNode `json:"nodes"`
	} `json:"search"`
}

type searchNode struct {
	Typename     string `json:"__typename"`
	Number       int    `json:"number"`
	Title        string `json:"title"`
	BodyText     string `json:"bodyText"`
	URL          string `json:"url"`
	IsDraft      bool   `json:"isDraft"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	ChangedFiles int    `json:"changedFiles"`
	HeadRefName  string `json:"headRefName"`
	BaseRefName  string `json:"baseRefName"`
	// reviewDecision is null in repositories without review requirements.
	ReviewDecision *string `json:"reviewDecision"`
	Comments       struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	// author is null when the account has been deleted.
	Author *struct {
		Typename string `json:"__typename"`
		Login    string `json:"login"`
	} `json:"author"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer *struct {
				Typename     string `json:"__typename"`
				Login        string `json:"login"`
				CombinedSlug string `json:"combinedSlug"`
			} `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

// searchConcurrency bounds how many scope searches are in flight at once:
// enough that a handful of orgs does not add up to a slow refresh, few enough
// to stay clear of GitHub's secondary rate limits on search.
const searchConcurrency = 4

// Search runs every scope and returns the union of what they found, most
// recently updated first.
//
// Scopes are searched separately rather than OR-ed into one query: per-scope
// semantics are unambiguous, and one scope failing (an org that enforces SAML
// SSO the token is not authorized for, say) costs only that scope's pull
// requests instead of the whole refresh. The error is non-nil only when every
// scope failed, since then there is nothing to show at all.
func (c *Client) Search(ctx context.Context, scopes []Scope) (Result, error) {
	type answer struct {
		prs     []PR
		outcome Outcome
	}
	answers := make([]answer, len(scopes))

	var wg sync.WaitGroup
	sem := make(chan struct{}, searchConcurrency)
	for i, scope := range scopes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			prs, total, partial, err := c.search(ctx, scope)
			answers[i] = answer{prs: prs, outcome: Outcome{Scope: scope, Err: err, Total: total, Partial: partial}}
		}()
	}
	wg.Wait()

	res := Result{Outcomes: make([]Outcome, 0, len(scopes))}
	seen := make(map[string]bool)
	var errs []error
	for _, a := range answers {
		res.Outcomes = append(res.Outcomes, a.outcome)
		if a.outcome.Err != nil {
			errs = append(errs, a.outcome.Err)
			continue
		}
		for _, pr := range a.prs {
			if !seen[pr.Key()] {
				seen[pr.Key()] = true
				res.PRs = append(res.PRs, pr)
			}
		}
	}
	if len(scopes) > 0 && len(errs) == len(scopes) {
		return Result{}, errors.Join(errs...)
	}

	SortPRs(res.PRs)
	return res, nil
}

// search runs a single scope, returning its pull requests, how many matched
// in total, and whether some of the matches were withheld.
//
// GitHub answers a search that hits a pull request the token may not read (in
// an org enforcing SAML SSO, say) with the readable nodes, a null in place of
// each unreadable one, and an error pointing at it. That is a partial answer,
// not a failed one: keep what was readable.
func (c *Client) search(ctx context.Context, scope Scope) (prs []PR, total int, partial bool, err error) {
	vars := map[string]any{"q": scope.query(), "limit": SearchLimit}
	var resp searchResponse
	if err := c.gql.DoWithContext(ctx, searchQuery, vars, &resp); err != nil {
		if !nodeErrorsOnly(err) {
			return nil, 0, false, fmt.Errorf("pull request search for %s failed: %w", scope, err)
		}
		partial = true
	}
	prs = make([]PR, 0, len(resp.Search.Nodes))
	for _, n := range resp.Search.Nodes {
		if pr, ok := n.toPR(); ok {
			prs = append(prs, pr)
		}
	}
	return prs, resp.Search.IssueCount, partial, nil
}

// nodeErrorsOnly reports whether err is a GraphQL error response in which
// every error is about a single search result node, leaving the rest of the
// data intact.
func nodeErrorsOnly(err error) bool {
	var gerr *api.GraphQLError
	if !errors.As(err, &gerr) || len(gerr.Errors) == 0 {
		return false
	}
	for _, e := range gerr.Errors {
		if len(e.Path) < 3 || e.Path[0] != "search" || e.Path[1] != "nodes" {
			return false
		}
	}
	return true
}

// SortPRs orders pull requests most-recently-updated first, falling back to
// the key so the order is stable for equal timestamps.
func SortPRs(prs []PR) {
	sort.SliceStable(prs, func(i, j int) bool {
		if prs[i].UpdatedAt.Equal(prs[j].UpdatedAt) {
			return prs[i].Key() < prs[j].Key()
		}
		return prs[i].UpdatedAt.After(prs[j].UpdatedAt)
	})
}

// stateQuery aliases the two state fields: Issue.state and PullRequest.state
// are different enum types, and GraphQL rejects the whole query when one
// response key would hold either.
const stateQuery = `
query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    issueOrPullRequest(number: $number) {
      ... on Issue { issueState: state }
      ... on PullRequest { prState: state }
    }
  }
}`

type stateResponse struct {
	Repository struct {
		IssueOrPullRequest struct {
			IssueState string `json:"issueState"`
			PRState    string `json:"prState"`
		} `json:"issueOrPullRequest"`
	} `json:"repository"`
}

// State reports how a pull request or issue left the open list: merged,
// closed, or still open (which happens when it was only edited into a state
// the search query no longer matches). Issues and pull requests share one
// number space per repository, so the number alone says which it is.
func (c *Client) State(ctx context.Context, repo string, number int) (State, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		return "", fmt.Errorf("repository is not in owner/name form: %q", repo)
	}
	vars := map[string]any{"owner": owner, "name": name, "number": number}
	var resp stateResponse
	if err := c.gql.DoWithContext(ctx, stateQuery, vars, &resp); err != nil {
		return "", fmt.Errorf("could not read the state of %s#%d: %w", repo, number, err)
	}
	got := resp.Repository.IssueOrPullRequest
	if got.PRState != "" {
		return State(got.PRState), nil
	}
	return State(got.IssueState), nil
}
