package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

	"github.com/kanywst/prpr/internal/browser"
	"github.com/kanywst/prpr/internal/gh"
)

// ownersMsg carries the result of owner discovery.
type ownersMsg struct {
	me     string
	owners []string
}

// prsMsg carries a completed refresh.
type prsMsg struct {
	prs []gh.PR
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
func (m Model) discoverOwnersCmd() tea.Cmd {
	fetcher, timeout := m.fetcher, m.timeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		me, orgs, err := fetcher.Viewer(ctx)
		if err != nil {
			return errMsg{err}
		}
		return ownersMsg{me: me, owners: append([]string{me}, orgs...)}
	}
}

// viewerCmd resolves just the viewer's login. It is used when --owner pinned
// the owner list but the identity-based tabs still need to know who "me" is.
func (m Model) viewerCmd() tea.Cmd {
	fetcher, timeout, owners := m.fetcher, m.timeout, m.pinnedOwners
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		me, _, err := fetcher.Viewer(ctx)
		if err != nil {
			return errMsg{err}
		}
		return ownersMsg{me: me, owners: owners}
	}
}

// fetchCmd refreshes the open pull request list.
func (m Model) fetchCmd() tea.Cmd {
	fetcher, timeout, owners := m.fetcher, m.timeout, m.owners
	if len(owners) == 0 {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		prs, err := fetcher.SearchOpenPRs(ctx, owners)
		if err != nil {
			return errMsg{err}
		}
		return prsMsg{prs: prs, at: time.Now()}
	}
}

// stateCmd looks up how a vanished pull request ended.
func (m Model) stateCmd(pr gh.PR) tea.Cmd {
	fetcher, timeout := m.fetcher, m.timeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		state, err := fetcher.State(ctx, pr.Repo, pr.Number)
		if err != nil {
			// A disappearance we cannot explain is still worth waving at, and
			// is not worth interrupting the user with an error.
			return goneMsg{pr: pr, state: gh.StateOpen}
		}
		return goneMsg{pr: pr, state: state}
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
