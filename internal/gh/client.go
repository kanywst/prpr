package gh

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// searchLimit caps how many PRs a single owner contributes. GitHub's search
// API refuses anything above 100 per page, and a dashboard past this size has
// stopped being a dashboard anyway.
const searchLimit = 60

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
query {
  viewer {
    login
    organizations(first: 100) { nodes { login } }
  }
}`

type viewerResponse struct {
	Viewer struct {
		Login         string `json:"login"`
		Organizations struct {
			Nodes []struct {
				Login string `json:"login"`
			} `json:"nodes"`
		} `json:"organizations"`
	} `json:"viewer"`
}

// Viewer returns the authenticated login together with every organization it
// belongs to. prpr uses this to discover which owners to watch, so the tool
// works for anyone who runs it without per-user configuration.
func (c *Client) Viewer(ctx context.Context) (login string, orgs []string, err error) {
	var resp viewerResponse
	if err := c.gql.DoWithContext(ctx, viewerQuery, nil, &resp); err != nil {
		return "", nil, fmt.Errorf("could not resolve the logged-in user: %w", err)
	}
	orgs = make([]string, 0, len(resp.Viewer.Organizations.Nodes))
	for _, n := range resp.Viewer.Organizations.Nodes {
		orgs = append(orgs, n.Login)
	}
	sort.Strings(orgs)
	return resp.Viewer.Login, orgs, nil
}

const searchQuery = `
query($q: String!, $limit: Int!) {
  search(query: $q, type: ISSUE, first: $limit) {
    nodes {
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
        author { login }
        repository { nameWithOwner }
        labels(first: 10) { nodes { name } }
        reviewRequests(first: 20) {
          nodes {
            requestedReviewer {
              __typename
              ... on User { login }
              ... on Team { slug }
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
		Nodes []searchNode `json:"nodes"`
	} `json:"search"`
}

type searchNode struct {
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
		Login string `json:"login"`
	} `json:"author"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer *struct {
				Typename string `json:"__typename"`
				Login    string `json:"login"`
				Slug     string `json:"slug"`
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

// SearchOpenPRs returns every open pull request under the given owners, most
// recently updated first.
//
// Owners are queried one at a time rather than OR-ed into a single search:
// per-owner semantics are unambiguous, a failure names the owner that caused
// it, and the owner list is short by construction.
func (c *Client) SearchOpenPRs(ctx context.Context, owners []string) ([]PR, error) {
	seen := make(map[string]bool)
	out := make([]PR, 0, len(owners)*8)

	for _, owner := range owners {
		vars := map[string]any{
			"q":     fmt.Sprintf("is:pr is:open archived:false user:%s", owner),
			"limit": searchLimit,
		}
		var resp searchResponse
		if err := c.gql.DoWithContext(ctx, searchQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("pull request search for %s failed: %w", owner, err)
		}

		for _, n := range resp.Search.Nodes {
			pr, ok := n.toPR()
			if !ok || seen[pr.Key()] {
				continue
			}
			seen[pr.Key()] = true
			out = append(out, pr)
		}
	}

	SortPRs(out)
	return out, nil
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

const stateQuery = `
query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) { state }
  }
}`

type stateResponse struct {
	Repository struct {
		PullRequest struct {
			State string `json:"state"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

// State reports how a pull request left the open list: merged, closed, or
// still open (which happens when it was only edited into a state the search
// query no longer matches).
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
	return State(resp.Repository.PullRequest.State), nil
}
