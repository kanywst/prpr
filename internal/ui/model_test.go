package ui

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kanywst/prpr/internal/cache"
	"github.com/kanywst/prpr/internal/gh"
)

// fakeFetcher serves canned responses, one page per refresh, so tests can walk
// a model through a sequence of refreshes without touching the network.
type fakeFetcher struct {
	me     string
	orgs   []string
	pages  [][]gh.PR
	calls  int
	states map[string]gh.State
	err    error
}

func (f *fakeFetcher) Viewer(context.Context) (login string, orgs []string, err error) {
	if f.err != nil {
		return "", nil, f.err
	}
	return f.me, f.orgs, nil
}

func (f *fakeFetcher) Search(_ context.Context, scopes []gh.Scope) (gh.Result, error) {
	if f.err != nil {
		return gh.Result{}, f.err
	}
	page := f.pages[min(f.calls, len(f.pages)-1)]
	f.calls++
	res := gh.Result{PRs: page}
	for _, s := range scopes {
		res.Outcomes = append(res.Outcomes, gh.Outcome{Scope: s})
	}
	return res, nil
}

func (f *fakeFetcher) State(_ context.Context, repo string, number int) (gh.State, error) {
	pr := gh.PR{Repo: repo, Number: number}
	if s, ok := f.states[pr.Key()]; ok {
		return s, nil
	}
	return gh.StateOpen, nil
}

// step applies one message and asserts the model type came back intact.
func step(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", next)
	}
	return got, cmd
}

// drain runs a command to completion, flattening batches into a message list.
func drain(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case nil:
		return nil
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range msg {
			out = append(out, drain(c)...)
		}
		return out
	default:
		return []tea.Msg{msg}
	}
}

// testModel builds a sized, themed model ready to receive messages.
func testModel(t *testing.T, f Fetcher) Model {
	t.Helper()
	m := New(Config{Fetcher: f, Interval: time.Minute, Timeout: time.Second})
	m.applyTheme(true)
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	return m
}

// samplePRs is a small, deliberately varied fixture.
func samplePRs(now time.Time) []gh.PR {
	return []gh.PR{
		{
			Number: 128, Title: "api: add rate limiter", Repo: "0-draft/api",
			Author: "kanywst", Check: gh.CheckSuccess, Review: gh.ReviewApproved,
			Additions: 142, Deletions: 9, ChangedFiles: 5, Comments: 3,
			HeadRef: "feat/rate-limiter", BaseRef: "main",
			URL: "https://github.com/0-draft/api/pull/128", UpdatedAt: now.Add(-2 * time.Hour),
		},
		{
			Number: 127, Title: "fix: nil deref on empty body", Repo: "0-draft/api",
			Author: "alice", Check: gh.CheckPending, Review: gh.ReviewRequired,
			Reviewers: []string{"kanywst"}, Additions: 8, Deletions: 2,
			URL: "https://github.com/0-draft/api/pull/127", UpdatedAt: now.Add(-5 * time.Hour),
		},
		{
			Number: 12, Title: "docs: update README", Repo: "kanywst/prpr",
			Author: "kanywst", Check: gh.CheckFailure, IsDraft: true,
			Additions: 31, URL: "https://github.com/kanywst/prpr/pull/12",
			UpdatedAt: now.Add(-7 * 24 * time.Hour),
		},
	}
}

func TestInitDiscoversOwners(t *testing.T) {
	f := &fakeFetcher{me: "kanywst", orgs: []string{"0-draft"}, pages: [][]gh.PR{nil}}
	m := testModel(t, f)

	var owners *ownersMsg
	for _, msg := range drain(m.Init()) {
		if o, ok := msg.(ownersMsg); ok {
			owners = &o
		}
	}
	if owners == nil {
		t.Fatal("Init did not produce an ownersMsg")
	}
	// The viewer's own account comes first, then every org they belong to.
	if got, want := owners.owners, []string{"kanywst", "0-draft"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("owners = %v, want %v", got, want)
	}
}

func TestPinnedOwnersSkipDiscovery(t *testing.T) {
	f := &fakeFetcher{me: "kanywst", orgs: []string{"0-draft"}, pages: [][]gh.PR{nil}}
	m := New(Config{Fetcher: f, Owners: []string{"someone-else"}, Interval: time.Minute, Timeout: time.Second})

	var me, fetched bool
	for _, msg := range drain(m.Init()) {
		switch msg := msg.(type) {
		case ownersMsg:
			t.Errorf("pinned owners still ran discovery: %v", msg)
		case meMsg:
			// The login is still resolved, so the identity tabs keep working.
			me = msg.me == "kanywst"
		case prsMsg:
			fetched = true
		}
	}
	if !me {
		t.Error("Init did not resolve the viewer's login")
	}
	// The pinned owners are enough to fetch; the login lookup does not gate it.
	if !fetched {
		t.Error("Init did not fetch the pinned owners")
	}
}

func TestFailedDiscoveryIsRetried(t *testing.T) {
	f := &fakeFetcher{err: errors.New("offline")}
	m := testModel(t, f)

	for _, msg := range drain(m.Init()) {
		m, _ = step(t, m, msg)
	}
	if m.lastErr == nil || len(m.owners) != 0 {
		t.Fatalf("after a failed start: lastErr=%v owners=%v", m.lastErr, m.owners)
	}

	// Once the interval passes, the next refresh retries discovery rather than
	// giving up because there are no owners to fetch.
	f.err = nil
	f.me, f.pages = "kanywst", [][]gh.PR{nil}
	m.now = m.lastFetch.Add(2 * time.Minute)
	if !m.refreshDue() {
		t.Fatal("refreshDue() = false after a failed discovery")
	}
	var found bool
	for _, msg := range drain(m.refreshCmd()) {
		if o, ok := msg.(ownersMsg); ok && o.me == "kanywst" {
			found = true
		}
	}
	if !found {
		t.Error("refresh did not retry owner discovery")
	}
}

func TestTabsPartitionByViewer(t *testing.T) {
	now := time.Now()
	f := &fakeFetcher{me: "kanywst", pages: [][]gh.PR{samplePRs(now)}}
	m := testModel(t, f)
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst", "0-draft"}})
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: samplePRs(now)}, at: now})

	counts := m.counts()
	for _, tt := range []struct {
		tab  tabID
		want int
	}{
		{tabAll, 3},
		{tabMine, 2},      // #128 and #12 are authored by kanywst
		{tabReview, 1},    // #127 requests a review from kanywst
		{tabElsewhere, 0}, // every sample lives under a watched owner
		{tabDraft, 1},     // #12 is a draft
	} {
		if got := counts[tt.tab]; got != tt.want {
			t.Errorf("counts[%v] = %d, want %d", tt.tab, got, tt.want)
		}
	}

	m.tab = tabReview
	m.recompute()
	if len(m.visible) != 1 || m.visible[0].Number != 127 {
		t.Errorf("review tab = %v, want just #127", m.visible)
	}
}

func TestBotsHaveTheirOwnTab(t *testing.T) {
	now := time.Now()
	bump := gh.PR{
		Number: 131, Title: "chore(deps): bump x", Repo: "0-draft/api",
		Author: "dependabot", IsBot: true, IsDraft: true,
		UpdatedAt: now.Add(-time.Hour),
	}
	asked := gh.PR{
		Number: 9, Title: "chore(deps): bump y", Repo: "elsewhere/lib",
		Author: "renovate", IsBot: true, Reviewers: []string{"kanywst"},
		UpdatedAt: now.Add(-2 * time.Hour),
	}
	prs := append(samplePRs(now), bump, asked)

	m := testModel(t, &fakeFetcher{me: "kanywst", pages: [][]gh.PR{prs}})
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst", "0-draft"}})
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: prs}, at: now})

	counts := m.counts()
	for _, tt := range []struct {
		tab  tabID
		want int
	}{
		{tabAll, 3},       // the bots stay out of everything
		{tabReview, 2},    // ...but a review asked of you by name is yours
		{tabElsewhere, 0}, // renovate's PR is outside the owners, and still a bot
		{tabDraft, 1},     // dependabot's draft is not counted
		{tabBots, 2},
	} {
		if got := counts[tt.tab]; got != tt.want {
			t.Errorf("counts[%v] = %d, want %d", tt.tab, got, tt.want)
		}
	}
}

func TestFilterNarrowsAndKeepsSelection(t *testing.T) {
	now := time.Now()
	m := testModel(t, &fakeFetcher{me: "kanywst", pages: [][]gh.PR{samplePRs(now)}})
	m, _ = step(t, m, ownersMsg{me: "kanywst"})
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: samplePRs(now)}, at: now})

	m.cursor = 1 // #127
	m.filter.SetValue("api")
	m.recompute()

	if len(m.visible) != 2 {
		t.Fatalf("filtered to %d PRs, want 2", len(m.visible))
	}
	// The cursor rides along with the pull request it was on.
	if pr, _ := m.selected(); pr.Number != 127 {
		t.Errorf("selection = #%d, want #127", pr.Number)
	}

	m.filter.SetValue("zzz")
	m.recompute()
	if len(m.visible) != 0 {
		t.Errorf("visible = %d, want 0", len(m.visible))
	}
	if _, ok := m.selected(); ok {
		t.Error("selected() returned a PR from an empty list")
	}
}

func TestMergeDetectionProducesFarewell(t *testing.T) {
	now := time.Now()
	all := samplePRs(now)
	f := &fakeFetcher{
		me:     "kanywst",
		states: map[string]gh.State{"0-draft/api#127": gh.StateMerged},
	}
	m := testModel(t, f)
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst"}})

	// First refresh: no farewells, because there is no previous list to diff.
	m, cmd := step(t, m, prsMsg{res: gh.Result{PRs: all}, at: now})
	if msgs := drain(cmd); len(msgs) != 0 {
		t.Fatalf("first refresh emitted %v, want nothing", msgs)
	}

	// Second refresh with #127 gone: prpr asks GitHub how it ended.
	remaining := []gh.PR{all[0], all[2]}
	m, cmd = step(t, m, prsMsg{res: gh.Result{PRs: remaining}, at: now.Add(time.Minute)})

	msgs := drain(cmd)
	if len(msgs) != 1 {
		t.Fatalf("second refresh emitted %d messages, want 1", len(msgs))
	}
	gone, ok := msgs[0].(goneMsg)
	if !ok {
		t.Fatalf("got %T, want goneMsg", msgs[0])
	}
	if gone.pr.Number != 127 || gone.state != gh.StateMerged {
		t.Fatalf("goneMsg = #%d %s, want #127 MERGED", gone.pr.Number, gone.state)
	}

	m, _ = step(t, m, gone)
	if len(m.farewells) != 1 {
		t.Fatalf("farewells = %d, want 1", len(m.farewells))
	}
	if m.farewells[0].state != gh.StateMerged {
		t.Errorf("farewell state = %s, want MERGED", m.farewells[0].state)
	}
}

func TestFarewellsExpireAndAreCapped(t *testing.T) {
	now := time.Now()
	m := testModel(t, &fakeFetcher{})
	m.now = now

	for i := range maxFarewells + 3 {
		m, _ = step(t, m, goneMsg{pr: gh.PR{Repo: "o/r", Number: i}, state: gh.StateMerged})
	}
	if len(m.farewells) != maxFarewells {
		t.Fatalf("farewells = %d, want capped at %d", len(m.farewells), maxFarewells)
	}

	m, _ = step(t, m, tickMsg(now.Add(farewellLife+time.Second)))
	if len(m.farewells) != 0 {
		t.Errorf("farewells = %d after expiry, want 0", len(m.farewells))
	}
}

func TestRefreshPausesWhileUnfocused(t *testing.T) {
	now := time.Now()
	m := testModel(t, &fakeFetcher{me: "kanywst"})
	m.owners = []string{"kanywst"}
	m.loading = false
	m.lastFetch = now.Add(-2 * time.Minute) // well past the interval
	m.now = now

	if !m.refreshDue() {
		t.Fatal("refreshDue() = false, want true when the interval has elapsed")
	}

	m, _ = step(t, m, tea.BlurMsg{})
	if m.refreshDue() {
		t.Error("refreshDue() = true while blurred; polling should pause")
	}

	m, _ = step(t, m, tea.FocusMsg{})
	if !m.refreshDue() {
		t.Error("refreshDue() = false after refocus, want true")
	}
}

func TestErrorDefersRetryByOneInterval(t *testing.T) {
	m := testModel(t, &fakeFetcher{})
	m.owners = []string{"kanywst"}

	m, _ = step(t, m, errMsg{errors.New("boom")})
	if m.loading {
		t.Error("loading stayed true after an error")
	}
	if m.lastErr == nil {
		t.Error("lastErr was not recorded")
	}
	// Without this, a failing endpoint would be retried every tick.
	m.now = time.Now()
	if m.refreshDue() {
		t.Error("refreshDue() = true immediately after an error")
	}
}

func TestCursorStaysInsideTheList(t *testing.T) {
	now := time.Now()
	m := testModel(t, &fakeFetcher{me: "kanywst"})
	m, _ = step(t, m, ownersMsg{me: "kanywst"})
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: samplePRs(now)}, at: now})

	for range 10 {
		m, _ = step(t, m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	}
	if m.cursor != len(m.visible)-1 {
		t.Errorf("cursor = %d after over-scrolling down, want %d", m.cursor, len(m.visible)-1)
	}

	for range 10 {
		m, _ = step(t, m, tea.KeyPressMsg{Code: 'k', Text: "k"})
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d after over-scrolling up, want 0", m.cursor)
	}
}

func TestScrollingKeepsCursorVisible(t *testing.T) {
	now := time.Now()
	many := make([]gh.PR, 50)
	for i := range many {
		many[i] = gh.PR{
			Number: i, Title: "pr", Repo: "o/r", Author: "kanywst",
			UpdatedAt: now.Add(-time.Duration(i) * time.Minute),
		}
	}

	m := testModel(t, &fakeFetcher{me: "kanywst"})
	m, _ = step(t, m, ownersMsg{me: "kanywst"})
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: many}, at: now})

	rows := m.metrics().rows
	m.cursor = len(many) - 1
	m.clampCursor()

	if m.cursor < m.offset || m.cursor >= m.offset+rows {
		t.Errorf("cursor %d outside the visible window [%d, %d)", m.cursor, m.offset, m.offset+rows)
	}
	if want := len(many) - rows; m.offset != want {
		t.Errorf("offset = %d, want %d", m.offset, want)
	}
}

func TestEscapeClearsTheFilterWithoutQuitting(t *testing.T) {
	now := time.Now()
	m := testModel(t, &fakeFetcher{me: "kanywst"})
	m, _ = step(t, m, ownersMsg{me: "kanywst"})
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: samplePRs(now)}, at: now})

	// Filter, accept it with enter, then press esc to clear: esc used to be a
	// quit key, so this sequence killed the program instead.
	m, _ = step(t, m, tea.KeyPressMsg{Code: '/', Text: "/"})
	m.filter.SetValue("readme")
	m.recompute()
	m, _ = step(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeList {
		t.Fatal("enter did not leave filter mode")
	}

	m, cmd := step(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.quit {
		t.Fatal("esc quit the program")
	}
	if cmd != nil {
		t.Errorf("esc produced a command (%T); it should only clear the filter", cmd())
	}
	if m.filter.Value() != "" {
		t.Errorf("filter = %q after esc, want cleared", m.filter.Value())
	}
	if len(m.visible) != 3 {
		t.Errorf("visible = %d after clearing, want 3", len(m.visible))
	}

	// A second esc, with nothing to clear, must still not quit.
	m, _ = step(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.quit {
		t.Error("esc on an unfiltered list quit the program")
	}
}

func TestFilterModeRoundTrip(t *testing.T) {
	now := time.Now()
	m := testModel(t, &fakeFetcher{me: "kanywst"})
	m, _ = step(t, m, ownersMsg{me: "kanywst"})
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: samplePRs(now)}, at: now})

	m, _ = step(t, m, tea.KeyPressMsg{Code: '/', Text: "/"})
	if m.mode != modeFilter {
		t.Fatal("pressing / did not enter filter mode")
	}
	// While filtering, the real terminal cursor is placed in the input.
	if m.cursor2D() == nil {
		t.Error("cursor2D() = nil in filter mode, want a placed cursor")
	}

	m.filter.SetValue("readme")
	m.recompute()
	if len(m.visible) != 1 {
		t.Fatalf("filtered to %d, want 1", len(m.visible))
	}

	m, _ = step(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeList {
		t.Error("escape did not leave filter mode")
	}
	if m.filter.Value() != "" {
		t.Errorf("filter = %q after escape, want cleared", m.filter.Value())
	}
	if len(m.visible) != 3 {
		t.Errorf("visible = %d after clearing the filter, want 3", len(m.visible))
	}
	if m.cursor2D() != nil {
		t.Error("cursor2D() returned a cursor outside filter mode")
	}
}

func TestFailedScopeCarriesItsPRsOver(t *testing.T) {
	now := time.Now()
	all := samplePRs(now)
	m := testModel(t, &fakeFetcher{me: "kanywst"})
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst", "0-draft"}})
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: all}, at: now})

	// 0-draft could not be searched this time; only kanywst/prpr#12 came back.
	res := gh.Result{
		PRs: []gh.PR{all[2]},
		Outcomes: []gh.Outcome{
			{Scope: gh.OwnerScope("kanywst"), Total: 1},
			{Scope: gh.OwnerScope("0-draft"), Err: errors.New("SAML")},
		},
	}
	m, cmd := step(t, m, prsMsg{res: res, at: now.Add(time.Minute)})

	if msgs := drain(cmd); len(msgs) != 0 {
		t.Errorf("a failed scope's PRs were treated as gone: %v", msgs)
	}
	if len(m.prs) != 3 {
		t.Errorf("list has %d PRs, want all 3 kept", len(m.prs))
	}
	if !strings.Contains(m.warning(), "0-draft") {
		t.Errorf("warning = %q, want it to name 0-draft", m.warning())
	}
}

func TestCappedScopeDoesNotWaveAtOpenPRs(t *testing.T) {
	m := testModel(t, &fakeFetcher{})
	pr := gh.PR{Repo: "o/r", Number: 1}

	m, _ = step(t, m, goneMsg{pr: pr, state: gh.StateOpen, capped: true})
	if len(m.farewells) != 0 {
		t.Error("a PR pushed past the page cap got a farewell")
	}
	// A capped PR that really merged still gets its moment.
	m, _ = step(t, m, goneMsg{pr: pr, state: gh.StateMerged, capped: true})
	if len(m.farewells) != 1 {
		t.Error("a merged PR under a capped scope got no farewell")
	}
}

func TestElsewhereTabAndScopes(t *testing.T) {
	now := time.Now()
	m := New(Config{
		Fetcher: &fakeFetcher{}, Interval: time.Minute, Timeout: time.Second,
		Authored: true, ReviewRequests: true,
	})
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst", "0-draft"}})

	scopes := m.scopes()
	kinds := make([]gh.ScopeKind, 0, len(scopes))
	for _, s := range scopes {
		kinds = append(kinds, s.Kind)
	}
	want := []gh.ScopeKind{gh.ScopeOwner, gh.ScopeOwner, gh.ScopeAuthor, gh.ScopeReviewRequested}
	if !slices.Equal(kinds, want) {
		t.Errorf("scopes = %v, want %v", kinds, want)
	}

	outside := gh.PR{Repo: "Someone/lib", Number: 4, Author: "kanywst", UpdatedAt: now}
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: append(samplePRs(now), outside)}, at: now})
	m.tab = tabElsewhere
	m.recompute()
	if len(m.visible) != 1 || m.visible[0].Key() != outside.Key() {
		t.Errorf("elsewhere tab = %v, want just %s", m.visible, outside.Key())
	}
}

func TestDiscoveryHonorsExcludeOwners(t *testing.T) {
	f := &fakeFetcher{me: "kanywst", orgs: []string{"0-draft", "Noisy"}, pages: [][]gh.PR{nil}}
	m := New(Config{Fetcher: f, ExcludeOwners: []string{"noisy"}, Interval: time.Minute, Timeout: time.Second})

	for _, msg := range drain(m.discoverOwnersCmd()) {
		o, ok := msg.(ownersMsg)
		if !ok {
			t.Fatalf("got %T, want ownersMsg", msg)
		}
		if !slices.Equal(o.owners, []string{"kanywst", "0-draft"}) {
			t.Errorf("owners = %v, want the excluded org dropped", o.owners)
		}
	}
}

func TestNoScopesDoesNotSpinForever(t *testing.T) {
	m := testModel(t, &fakeFetcher{})
	// Every owner excluded, and the searches outside them turned off.
	m, cmd := step(t, m, ownersMsg{me: "kanywst"})
	if cmd != nil || m.loading || !m.ready {
		t.Fatalf("with no scopes: cmd=%v loading=%v ready=%v, want a settled empty list", cmd != nil, m.loading, m.ready)
	}
}

func TestSubmittedOutsideReviewLeavesQuietly(t *testing.T) {
	now := time.Now()
	m := testModel(t, &fakeFetcher{})
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst"}})

	outside := gh.PR{Repo: "someone/lib", Number: 7, Author: "alice", Reviewers: []string{"kanywst"}, UpdatedAt: now}
	outcomes := []gh.Outcome{{Scope: gh.OwnerScope("kanywst")}, {Scope: gh.ReviewRequestedScope("kanywst")}}
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: []gh.PR{outside}, Outcomes: outcomes}, at: now})

	m, cmd := step(t, m, prsMsg{res: gh.Result{Outcomes: outcomes}, at: now.Add(time.Minute)})
	if msgs := drain(cmd); len(msgs) != 0 {
		t.Errorf("a reviewed outside PR was looked up for a farewell: %v", msgs)
	}
	if len(m.prs) != 0 {
		t.Errorf("list = %v, want it gone", m.prs)
	}
}

// stateErrFetcher fails every State lookup.
type stateErrFetcher struct{ fakeFetcher }

func (stateErrFetcher) State(context.Context, string, int) (gh.State, error) {
	return "", errors.New("lookup failed")
}

func TestCappedPRWithFailedLookupStillWaves(t *testing.T) {
	m := testModel(t, &stateErrFetcher{})
	msgs := drain(m.stateCmd(gh.PR{Repo: "o/r", Number: 1}, true))
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	m, _ = step(t, m, msgs[0])
	if len(m.farewells) != 1 {
		t.Error("a capped PR whose state lookup failed vanished without a farewell")
	}
}

func TestCappedReviewRequestIsLookedUp(t *testing.T) {
	now := time.Now()
	m := testModel(t, &fakeFetcher{})
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst"}})

	outside := gh.PR{Repo: "someone/lib", Number: 7, Author: "alice", Reviewers: []string{"kanywst"}, UpdatedAt: now}
	outcomes := []gh.Outcome{
		{Scope: gh.OwnerScope("kanywst")},
		{Scope: gh.ReviewRequestedScope("kanywst"), Total: gh.SearchLimit + 1},
	}
	m, _ = step(t, m, prsMsg{res: gh.Result{PRs: []gh.PR{outside}, Outcomes: outcomes}, at: now})

	// Pushed past the cap of a truncated review-request search: that is not
	// evidence the review was done, so prpr asks rather than dropping it.
	_, cmd := step(t, m, prsMsg{res: gh.Result{Outcomes: outcomes}, at: now.Add(time.Minute)})
	msgs := drain(cmd)
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want a state lookup", len(msgs))
	}
	if g, ok := msgs[0].(goneMsg); !ok || !g.capped {
		t.Errorf("got %#v, want a capped goneMsg", msgs[0])
	}
}

func TestCachedListShowsAtOnceAndWavesAtWhatMergedMeanwhile(t *testing.T) {
	now := time.Now()
	cached := samplePRs(now)
	var saved []cache.Snapshot
	f := &fakeFetcher{
		me:     "kanywst",
		orgs:   []string{"0-draft"},
		states: map[string]gh.State{"0-draft/api#127": gh.StateMerged},
	}
	m := New(Config{
		Fetcher: f, Interval: time.Minute, Timeout: time.Second, Authored: true,
		Cached: &cache.Snapshot{
			Me: "kanywst", Owners: []string{"kanywst", "0-draft"},
			PRs: cached, At: now.Add(-3 * time.Hour),
		},
		SaveCache: func(s cache.Snapshot) error { saved = append(saved, s); return nil },
	})
	m.applyTheme(true)
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})

	if !m.ready || len(m.visible) != len(cached) {
		t.Fatalf("ready = %v, visible = %d; want the cached list on screen", m.ready, len(m.visible))
	}
	if !strings.Contains(m.statusView(), "3h") {
		t.Errorf("status %q does not say how old the cache is", m.statusView())
	}

	// The cached login is confirmed before anything is searched as it.
	var owners *ownersMsg
	for _, msg := range drain(m.Init()) {
		if o, ok := msg.(ownersMsg); ok {
			owners = &o
		}
	}
	if owners == nil {
		t.Fatal("a cached start did not re-run discovery")
	}
	m, _ = step(t, m, *owners)

	fresh := slices.DeleteFunc(slices.Clone(cached), func(pr gh.PR) bool { return pr.Number == 127 })
	m, cmd := step(t, m, prsMsg{res: gh.Result{PRs: fresh}, at: now})
	for _, msg := range drain(cmd) {
		m, _ = step(t, m, msg)
	}

	if len(m.farewells) != 1 || m.farewells[0].pr.Number != 127 || m.farewells[0].state != gh.StateMerged {
		t.Errorf("farewells = %+v, want #127 merged while prpr was closed", m.farewells)
	}
	if !m.cachedAt.IsZero() {
		t.Error("the list is still marked cached after a refresh landed")
	}
	if len(saved) != 1 || len(saved[0].PRs) != len(fresh) || saved[0].Me != "kanywst" {
		t.Errorf("saved = %+v, want the fresh list", saved)
	}
}

func TestCachedListFromAnotherAccountIsNotWavedAt(t *testing.T) {
	now := time.Now()
	f := &fakeFetcher{me: "kanywst", pages: [][]gh.PR{nil}}
	m := New(Config{
		Fetcher: f, Interval: time.Minute, Timeout: time.Second, Authored: true,
		Cached: &cache.Snapshot{
			Me: "someone-else", Owners: []string{"someone-else"},
			PRs: []gh.PR{{Number: 1, Repo: "someone-else/repo", Author: "someone-else", UpdatedAt: now}},
			At:  now.Add(-time.Hour),
		},
	})

	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst"}})
	if len(m.prs) != 0 {
		t.Fatalf("prs = %v, want the other account's list dropped", m.prs)
	}
	_, cmd := step(t, m, prsMsg{res: gh.Result{}, at: now})
	if msgs := drain(cmd); len(msgs) != 0 {
		t.Errorf("refresh emitted %v, want no farewell lookups", msgs)
	}
}

func TestCachedListRespectsPinnedOwners(t *testing.T) {
	now := time.Now()
	m := New(Config{
		Fetcher: &fakeFetcher{me: "kanywst", orgs: []string{"0-draft"}},
		Owners:  []string{"kanywst"}, Interval: time.Minute, Timeout: time.Second,
		Cached: &cache.Snapshot{
			Me: "kanywst", Owners: []string{"kanywst", "0-draft"}, PRs: samplePRs(now), At: now,
		},
	})
	// Only kanywst/prpr#12 is under the pinned owner.
	if len(m.prs) != 1 || m.prs[0].Number != 12 {
		t.Errorf("prs = %v, want only what the pinned owner covers", m.prs)
	}
	m, _ = step(t, m, ownersMsg{me: "kanywst", owners: []string{"kanywst", "0-draft"}})
	if len(m.owners) != 1 || m.owners[0] != "kanywst" {
		t.Errorf("owners = %v, discovery replaced the pinned owners", m.owners)
	}
}
