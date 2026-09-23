package demo

import (
	"context"
	"testing"
	"time"

	"github.com/kanywst/prpr/internal/gh"
)

func TestFixturesAreWellFormed(t *testing.T) {
	prs := fixtures(time.Now())
	if len(prs) < 5 {
		t.Fatalf("only %d fixtures; a recording of a near-empty list shows nothing", len(prs))
	}

	seen := make(map[string]bool, len(prs))
	for _, pr := range prs {
		switch {
		case pr.Repo == "" || pr.Number == 0:
			t.Errorf("fixture %q has no repo or number", pr.Title)
		case pr.Title == "":
			t.Errorf("fixture %s has no title", pr.Key())
		case pr.URL == "":
			t.Errorf("fixture %s has no URL", pr.Key())
		case seen[pr.Key()]:
			t.Errorf("duplicate fixture %s", pr.Key())
		}
		seen[pr.Key()] = true
	}

	// The list arrives sorted, so a recording is stable between runs.
	for i := 1; i < len(prs); i++ {
		if prs[i-1].UpdatedAt.Before(prs[i].UpdatedAt) {
			t.Errorf("fixtures are not newest-first at index %d", i)
		}
	}
}

func TestFixturesCoverEveryTab(t *testing.T) {
	prs := fixtures(time.Now())

	var mine, review, elsewhere, draft int
	for _, pr := range prs {
		if o := pr.Owner(); o != viewer && o != "0-draft" {
			elsewhere++
		}
		if pr.AuthoredBy("kanywst") {
			mine++
		}
		if pr.AwaitsReviewFrom("kanywst") {
			review++
		}
		if pr.IsDraft {
			draft++
		}
	}
	// Every tab has to have something in it, or the recording shows an empty
	// pane the moment it switches tabs.
	if mine == 0 || review == 0 || elsewhere == 0 || draft == 0 {
		t.Errorf("tab coverage: mine=%d review=%d elsewhere=%d draft=%d, want all non-zero", mine, review, elsewhere, draft)
	}
}

func TestMergeHappensOnCamera(t *testing.T) {
	f := New()
	ctx := context.Background()

	before, err := f.Search(ctx, nil)
	if err != nil {
		t.Fatalf("first search: %v", err)
	}

	// Wind the clock forward rather than sleeping through the real delay.
	f.started = time.Now().Add(-mergeAfter - time.Second)
	after, err := f.Search(ctx, nil)
	if err != nil {
		t.Fatalf("second search: %v", err)
	}

	if len(after.PRs) != len(before.PRs)-1 {
		t.Fatalf("after the merge window %d PRs remain, want %d", len(after.PRs), len(before.PRs)-1)
	}
	for _, pr := range after.PRs {
		if pr.Key() == mergedKey {
			t.Fatalf("%s should have been merged away", mergedKey)
		}
	}

	state, err := f.State(ctx, "0-draft/api", 127)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if state != gh.StateMerged {
		t.Errorf("state = %q, want %q", state, gh.StateMerged)
	}
}

func TestContextCancellationIsHonoured(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := New().Search(ctx, nil); err == nil {
		t.Error("a canceled context should abort the simulated latency")
	}
}
