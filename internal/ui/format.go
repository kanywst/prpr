package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/kanywst/prpr/internal/gh"
)

// humanAge renders a duration the way a person would say it out loud.
func humanAge(d time.Duration) string {
	switch {
	case d < 0:
		return "いま"
	case d < time.Minute:
		return "いま"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "分"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "時間"
	case d < 7*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/24)) + "日"
	case d < 35*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/24/7)) + "週間"
	default:
		return strconv.Itoa(int(d.Hours()/24/30)) + "ヶ月"
	}
}

// checkIcon maps a rolled-up CI state to a single glyph. Draft wins over the
// check state: a draft's checks are not what you are scanning the list for.
func checkIcon(c gh.Check, isDraft bool) string {
	if isDraft {
		return "📝"
	}
	switch c {
	case gh.CheckSuccess:
		return "🟢"
	case gh.CheckPending, gh.CheckExpected:
		return "🟡"
	case gh.CheckFailure, gh.CheckError:
		return "🔴"
	case gh.CheckNone:
		return "⚪"
	default:
		return "⚪"
	}
}

// reviewIcon maps a review decision to a glyph, or "" when the repository has
// no review requirement at all.
func reviewIcon(r gh.Review) string {
	switch r {
	case gh.ReviewApproved:
		return "✅"
	case gh.ReviewChanges:
		return "🔁"
	case gh.ReviewRequired:
		return "👀"
	case gh.ReviewNone:
		return ""
	default:
		return ""
	}
}

// stateWord describes how a pull request left the list, for the farewell band.
func stateWord(s gh.State) (icon, word string) {
	switch s {
	case gh.StateMerged:
		return "🎉", "マージされたよ〜 おめでとう!"
	case gh.StateClosed:
		return "🌙", "クローズされたよ"
	case gh.StateOpen:
		return "👋", "一覧から外れたよ"
	default:
		return "👋", "一覧から外れたよ"
	}
}

// matchesFilter reports whether a pull request matches a filter query. The
// query is matched case-insensitively against every field a person is likely
// to type: title, repo, author, number (with or without "#"), and labels.
func matchesFilter(pr gh.PR, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	fields := []string{
		strings.ToLower(pr.Title),
		strings.ToLower(pr.Repo),
		strings.ToLower(pr.Author),
		"#" + strconv.Itoa(pr.Number),
		strconv.Itoa(pr.Number),
	}
	for _, l := range pr.Labels {
		fields = append(fields, strings.ToLower(l))
	}
	// Every whitespace-separated term must match somewhere, so "kt api" narrows
	// rather than widens.
	for _, term := range strings.Fields(q) {
		hit := false
		for _, f := range fields {
			if strings.Contains(f, term) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// truncate shortens s to width display cells, appending an ellipsis when it
// had to cut. It is ANSI- and wide-rune-aware.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return ansi.Truncate(s, width, "…")
}

// pad right-pads s with spaces to width display cells, leaving longer strings
// untouched.
func pad(s string, width int) string {
	gap := width - ansi.StringWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}
