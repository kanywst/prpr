// Package demo serves a canned pull request list so prpr can be recorded and
// screenshotted without reaching GitHub. Recording against a real account
// would put whatever repositories that account can see into a published GIF,
// and would make the result different every time it was re-recorded.
package demo

import (
	"context"
	"slices"
	"time"

	"github.com/kanywst/prpr/internal/gh"
)

// Timings chosen so a short recording catches the whole story: the list
// appears, a pull request is merged on camera, and the farewell banner has
// time to be read before it fades.
const (
	mergeAfter = 11 * time.Second
	latency    = 400 * time.Millisecond
)

const (
	// mergedKey is the pull request that disappears part-way through a recording.
	mergedKey = "0-draft/api#127"
	// viewer is the fictional account the fixtures are shown as.
	viewer = "kanywst"
	// mainRef is the base branch every fixture targets.
	mainRef = "main"
	// apiRepo and prprRepo are the busiest fixture repositories.
	apiRepo  = "0-draft/api"
	prprRepo = "kanywst/prpr"
	// alice is the busiest fixture teammate.
	alice = "alice"
	// enhancement is the most common fixture label.
	enhancement = "enhancement"
)

// Fetcher implements the interface the UI consumes, from fixtures.
type Fetcher struct {
	started time.Time
}

// New returns a Fetcher whose clock starts now.
func New() *Fetcher {
	return &Fetcher{started: time.Now()}
}

// Viewer returns the fictional account the fixtures belong to.
func (f *Fetcher) Viewer(ctx context.Context) (login string, orgs []string, err error) {
	if err := sleep(ctx, latency); err != nil {
		return "", nil, err
	}
	return viewer, []string{"0-draft"}, nil
}

// Search returns the fixture list, dropping one pull request once the
// recording has been running long enough for the merge to land on camera.
func (f *Fetcher) Search(ctx context.Context, scopes []gh.Scope) (gh.Result, error) {
	if err := sleep(ctx, latency); err != nil {
		return gh.Result{}, err
	}

	all := fixtures(time.Now())
	res := gh.Result{Outcomes: make([]gh.Outcome, 0, len(scopes))}
	issues := false
	for _, s := range scopes {
		res.Outcomes = append(res.Outcomes, gh.Outcome{Scope: s})
		issues = issues || s.Issues
	}
	// The issue fixtures only appear under --issues, as they would live.
	if !issues {
		all = slices.DeleteFunc(all, func(pr gh.PR) bool { return pr.IsIssue })
	}
	res.PRs = all
	if time.Since(f.started) < mergeAfter {
		return res, nil
	}

	res.PRs = make([]gh.PR, 0, len(all))
	for _, pr := range all {
		if pr.Key() != mergedKey {
			res.PRs = append(res.PRs, pr)
		}
	}
	return res, nil
}

// State reports every vanished pull request as merged, which is the outcome
// worth showing.
func (f *Fetcher) State(ctx context.Context, _ string, _ int) (gh.State, error) {
	if err := sleep(ctx, latency); err != nil {
		return "", err
	}
	return gh.StateMerged, nil
}

// sleep waits, or gives up early if the caller's context does.
func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// fixtures is the pull request list, with ages relative to now so a recording
// never shows a stale "updated 8 months ago".
func fixtures(now time.Time) []gh.PR {
	prs := []gh.PR{
		{
			Number: 128, Title: "api: add a per-token rate limiter",
			Repo: apiRepo, Author: viewer,
			Check: gh.CheckSuccess, Review: gh.ReviewApproved,
			Additions: 142, Deletions: 9, ChangedFiles: 5, Comments: 3,
			HeadRef: "feat/rate-limiter", BaseRef: mainRef,
			Labels:    []string{enhancement, "api"},
			UpdatedAt: now.Add(-2 * time.Hour), CreatedAt: now.Add(-3 * 24 * time.Hour),
			URL: "https://github.com/0-draft/api/pull/128",
			Body: "Adds a token-bucket limiter in front of the public endpoints. " +
				"The bucket is per API token rather than per IP, so a shared NAT does " +
				"not take down a whole office.",
		},
		{
			Number: 127, Title: "fix: nil deref when the request body is empty",
			Repo: apiRepo, Author: alice,
			Check: gh.CheckPending, Review: gh.ReviewRequired,
			Reviewers: []string{viewer},
			Additions: 8, Deletions: 2, ChangedFiles: 1, Comments: 1,
			HeadRef: "fix/empty-body", BaseRef: mainRef,
			Labels:    []string{"bug"},
			UpdatedAt: now.Add(-5 * time.Hour), CreatedAt: now.Add(-6 * time.Hour),
			URL:  "https://github.com/0-draft/api/pull/127",
			Body: "decode() assumed a non-empty body. It does not.",
		},
		{
			Number: 64, Title: "worker: retry the webhook delivery with jitter",
			Repo: "0-draft/worker", Author: "bob",
			Check: gh.CheckFailure, Review: gh.ReviewChanges,
			Reviewers: []string{viewer, alice},
			Additions: 96, Deletions: 41, ChangedFiles: 7, Comments: 12,
			HeadRef: "feat/retry-jitter", BaseRef: mainRef,
			Labels:    []string{"reliability"},
			UpdatedAt: now.Add(-26 * time.Hour), CreatedAt: now.Add(-5 * 24 * time.Hour),
			URL:  "https://github.com/0-draft/worker/pull/64",
			Body: "Exponential backoff with full jitter, so a downstream outage does not turn into a thundering herd on recovery.",
		},
		{
			Number: 63, Title: "chore: drop the vendored copy of the OUI database",
			Repo: "0-draft/worker", Author: viewer,
			Check:     gh.CheckSuccess,
			Additions: 4, Deletions: 21418, ChangedFiles: 3,
			HeadRef: "chore/unvendor-oui", BaseRef: mainRef,
			UpdatedAt: now.Add(-2 * 24 * time.Hour), CreatedAt: now.Add(-2 * 24 * time.Hour),
			URL: "https://github.com/0-draft/worker/pull/63",
		},
		{
			Number: 12, Title: "docs: write the getting-started page",
			Repo: prprRepo, Author: viewer,
			Check: gh.CheckNone, IsDraft: true,
			Additions: 31, ChangedFiles: 1,
			HeadRef: "docs/getting-started", BaseRef: mainRef,
			UpdatedAt: now.Add(-7 * 24 * time.Hour), CreatedAt: now.Add(-7 * 24 * time.Hour),
			URL:  "https://github.com/kanywst/prpr/pull/12",
			Body: "Still a sketch.",
		},
		{
			Number: 9, Title: "feat: remember the selected tab between runs",
			Repo: prprRepo, Author: "carol",
			Check: gh.CheckSuccess, Review: gh.ReviewRequired,
			Reviewers: []string{viewer},
			Additions: 58, Deletions: 6, ChangedFiles: 4, Comments: 2,
			HeadRef: "feat/sticky-tab", BaseRef: mainRef,
			Labels:    []string{enhancement, "good first issue"},
			UpdatedAt: now.Add(-3 * 24 * time.Hour), CreatedAt: now.Add(-4 * 24 * time.Hour),
			URL: "https://github.com/kanywst/prpr/pull/9",
		},
		{
			Number: 3, Title: "ci: cross-compile for windows/arm64 too",
			Repo: "kanywst/scoop-bucket", Author: "dave",
			Check:     gh.CheckPending,
			Additions: 11, Deletions: 1, ChangedFiles: 1,
			HeadRef: "ci/windows-arm64", BaseRef: mainRef,
			UpdatedAt: now.Add(-40 * 24 * time.Hour), CreatedAt: now.Add(-40 * 24 * time.Hour),
			URL: "https://github.com/kanywst/scoop-bucket/pull/3",
		},
		{
			// A dependency bump, for the bots tab.
			Number: 131, Title: "chore(deps): bump golang.org/x/net from 0.33.0 to 0.34.0",
			Repo: apiRepo, Author: "dependabot", IsBot: true,
			Check:     gh.CheckSuccess,
			Additions: 3, Deletions: 3, ChangedFiles: 2,
			HeadRef: "dependabot/go_modules/golang.org/x/net-0.34.0", BaseRef: mainRef,
			Labels:    []string{"dependencies"},
			UpdatedAt: now.Add(-9 * time.Hour), CreatedAt: now.Add(-9 * time.Hour),
			URL: "https://github.com/0-draft/api/pull/131",
		},
		{
			// An issue assigned to the viewer, for --issues.
			Number: 140, Title: "rate limiter ignores the Retry-After header",
			Repo: apiRepo, Author: alice, IsIssue: true,
			Assignees: []string{viewer}, Comments: 4,
			Labels:    []string{"bug"},
			UpdatedAt: now.Add(-4 * time.Hour), CreatedAt: now.Add(-2 * 24 * time.Hour),
			URL:  "https://github.com/0-draft/api/issues/140",
			Body: "Clients that honor Retry-After still get throttled, because the limiter refills on its own clock.",
		},
		{
			Number: 20, Title: "idea: a compact one-line layout",
			Repo: prprRepo, Author: "carol", IsIssue: true, Comments: 1,
			Labels:    []string{enhancement},
			UpdatedAt: now.Add(-6 * 24 * time.Hour), CreatedAt: now.Add(-6 * 24 * time.Hour),
			URL: "https://github.com/kanywst/prpr/issues/20",
		},
		{
			// A contribution outside the watched owners, for the elsewhere tab.
			Number: 214, Title: "fix: redraw after SIGWINCH while a prompt is open",
			Repo: "tiny-lantern/lantern", Author: viewer,
			Check: gh.CheckSuccess, Review: gh.ReviewApproved,
			Additions: 23, Deletions: 4, ChangedFiles: 2, Comments: 5,
			HeadRef: "fix/sigwinch-redraw", BaseRef: mainRef,
			UpdatedAt: now.Add(-26 * time.Hour), CreatedAt: now.Add(-5 * 24 * time.Hour),
			URL: "https://github.com/tiny-lantern/lantern/pull/214",
		},
	}

	gh.SortPRs(prs)
	return prs
}
