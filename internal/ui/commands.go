package ui

import (
	"context"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

	"github.com/kanywst/prpr/internal/browser"
	"github.com/kanywst/prpr/internal/cache"
	"github.com/kanywst/prpr/internal/gh"
)

// ownersMsg carries the result of owner discovery.
type ownersMsg struct {
	me     string
	owners []string
}

// meMsg carries the viewer's login for pinned owners. An empty login means the
// lookup failed and is retried on the next refresh.
type meMsg struct{ me string }

// prsMsg carries a completed refresh.
type prsMsg struct {
	res gh.Result
	at  time.Time
}

// errMsg carries a failure that should be shown to the user and retried on
// the next refresh rather than crashing the program.
type errMsg struct{ err error }

// goneMsg reports how a pull request that vanished from the open list ended
// up, which is what turns a disappearance into a celebration.
type goneMsg struct {
	pr    gh.PR
	state gh.State
	// capped is set when the pull request may only have been pushed past a
	// search's page cap. If it turns out to be still open, nothing happened
	// to it and there is nothing to wave at.
	capped bool
}

// noticeMsg is a transient status line, used for things like "copied".
type noticeMsg struct{ text string }

// tickMsg drives the countdown, the farewell animation, and the check for
// whether an automatic refresh is due.
type tickMsg time.Time

// tickCmd schedules the next animation frame.
func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// discoverOwnersCmd resolves which owners to watch from the authenticated
// user, so prpr needs no per-user configuration to be useful.
//
// It also confirms a cached login when the owners are pinned. Then a failed
// lookup must not hold the pinned owners' search hostage, so it settles as an
// unknown login, which the next refresh looks up again alongside the fetch.
func (m Model) discoverOwnersCmd() tea.Cmd {
	fetcher, timeout, exclude := m.fetcher, m.timeout, m.excludeOwners
	pinned := len(m.pinnedOwners) > 0
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		me, orgs, err := fetcher.Viewer(ctx)
		switch {
		case err != nil && pinned:
			return ownersMsg{}
		case err != nil:
			return errMsg{err}
		}
		return ownersMsg{me: me, owners: withoutExcluded(append([]string{me}, orgs...), exclude)}
	}
}

// withoutExcluded drops the excluded owners, ignoring case as GitHub does.
func withoutExcluded(owners, exclude []string) []string {
	out := make([]string, 0, len(owners))
	for _, o := range owners {
		if !slices.ContainsFunc(exclude, func(x string) bool { return strings.EqualFold(x, o) }) {
			out = append(out, o)
		}
	}
	return out
}

// viewerCmd resolves just the viewer's login. It is used when --owner pinned
// the owner list but the identity-based tabs still need to know who "me" is.
// It runs alongside the first fetch rather than gating it: the pinned owners
// are all a fetch needs, and a failed lookup here is retried on the next
// refresh instead of holding the list hostage.
func (m Model) viewerCmd() tea.Cmd {
	fetcher, timeout := m.fetcher, m.timeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		me, _, err := fetcher.Viewer(ctx)
		if err != nil {
			return meMsg{}
		}
		return meMsg{me: me}
	}
}

// refreshCmd does whatever the next refresh needs. Owner discovery is retried
// here too, so a start-up that could not reach GitHub recovers on its own
// instead of sitting on an error until the program is restarted.
func (m Model) refreshCmd() tea.Cmd {
	// Discovery can legitimately leave no owners, when exclude_owners drops
	// them all; knowing the login is what says it has run. A login read from
	// the cache does not count until discovery has confirmed it: gh may have
	// switched accounts since.
	if m.unverified || (len(m.owners) == 0 && m.me == "") {
		return m.discoverOwnersCmd()
	}
	if m.me == "" {
		return tea.Batch(m.viewerCmd(), m.fetchCmd())
	}
	return m.fetchCmd()
}

// fetchCmd refreshes the open pull request list.
func (m Model) fetchCmd() tea.Cmd {
	fetcher, timeout, scopes := m.fetcher, m.timeout, m.scopes()
	if len(scopes) == 0 {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		res, err := fetcher.Search(ctx, scopes)
		if err != nil {
			return errMsg{err}
		}
		return prsMsg{res: res, at: time.Now()}
	}
}

// saveCmd stores a completed refresh for the next run. A failed save is not
// worth interrupting anyone over: the cache is only ever a head start.
func (m Model) saveCmd(prs []gh.PR, at time.Time) tea.Cmd {
	if m.saveCache == nil {
		return nil
	}
	save := m.saveCache
	snap := cache.Snapshot{Me: m.me, Owners: slices.Clone(m.owners), PRs: prs, At: at}
	return func() tea.Msg {
		_ = save(snap)
		return nil
	}
}

// stateCmd looks up how a vanished pull request ended.
func (m Model) stateCmd(pr gh.PR, capped bool) tea.Cmd {
	fetcher, timeout := m.fetcher, m.timeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		state, err := fetcher.State(ctx, pr.Repo, pr.Number)
		if err != nil {
			// A disappearance we cannot explain is still worth waving at, and
			// is not worth interrupting the user with an error. It is not
			// marked capped: that suppression is for a PR confirmed still
			// open, and this one's state is unknown.
			return goneMsg{pr: pr, state: gh.StateOpen}
		}
		return goneMsg{pr: pr, state: state, capped: capped}
	}
}

// openCmd opens a pull request in the browser.
func (m Model) openCmd(url string) tea.Cmd {
	opened := m.s.OpenedInBrowser
	return func() tea.Msg {
		if err := browser.Open(url); err != nil {
			return errMsg{err}
		}
		return noticeMsg{opened}
	}
}

// copyCmd puts a pull request URL on the system clipboard.
func (m Model) copyCmd(url string) tea.Cmd {
	copied, failed := m.s.CopiedURL, m.s.ClipboardFailed
	return func() tea.Msg {
		if err := clipboard.WriteAll(url); err != nil {
			return noticeMsg{failed}
		}
		return noticeMsg{copied}
	}
}
