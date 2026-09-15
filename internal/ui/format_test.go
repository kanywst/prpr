package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/kanywst/prpr/internal/gh"
)

func TestHumanAge(t *testing.T) {
	en := Catalog(LangEN)
	tests := []struct {
		in   time.Duration
		want string
	}{
		{-time.Hour, "just now"}, // a clock skewed into the future reads as now
		{30 * time.Second, "just now"},
		{90 * time.Second, "1m"},
		{59 * time.Minute, "59m"},
		{3 * time.Hour, "3h"},
		{47 * time.Hour, "1d"},
		{6 * 24 * time.Hour, "6d"},
		{10 * 24 * time.Hour, "1w"},
		{60 * 24 * time.Hour, "2mo"},
	}
	for _, tt := range tests {
		if got := humanAge(tt.in, en); got != tt.want {
			t.Errorf("humanAge(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}

	if got := humanAge(3*time.Hour, Catalog(LangJA)); got != "3時間" {
		t.Errorf("humanAge in Japanese = %q, want %q", got, "3時間")
	}
}

func TestCheckIconDraftWins(t *testing.T) {
	if got := checkIcon(gh.CheckSuccess, true); got != "📝" {
		t.Errorf("draft PR icon = %q, want 📝", got)
	}
	if got := checkIcon(gh.CheckFailure, false); got != "🔴" {
		t.Errorf("failing PR icon = %q, want 🔴", got)
	}
	if got := checkIcon(gh.CheckNone, false); got != "⚪" {
		t.Errorf("no-checks PR icon = %q, want ⚪", got)
	}
	if got := checkIcon(gh.Check("SOMETHING_NEW"), false); got != "⚪" {
		t.Errorf("unknown check state icon = %q, want ⚪", got)
	}
}

func TestReviewIconEmptyWhenNoDecision(t *testing.T) {
	if got := reviewIcon(gh.ReviewNone); got != "" {
		t.Errorf("reviewIcon(none) = %q, want empty", got)
	}
	if got := reviewIcon(gh.ReviewApproved); got != "✅" {
		t.Errorf("reviewIcon(approved) = %q, want ✅", got)
	}
}

func TestMatchesFilter(t *testing.T) {
	pr := gh.PR{
		Number: 128,
		Title:  "api: add Rate Limiter",
		Repo:   "0-draft/api",
		Author: "kanywst",
		Labels: []string{"enhancement"},
	}

	tests := []struct {
		query string
		want  bool
	}{
		{"", true},
		{"   ", true},
		{"rate", true},
		{"RATE", true},
		{"0-draft", true},
		{"kanywst", true},
		{"#128", true},
		{"128", true},
		{"enhancement", true},
		{"kanywst rate", true},  // every term must match
		{"kanywst nope", false}, // ...so one miss rejects
		{"nothing here", false},
	}
	for _, tt := range tests {
		if got := matchesFilter(pr, tt.query); got != tt.want {
			t.Errorf("matchesFilter(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}

func TestTruncateRespectsDisplayWidth(t *testing.T) {
	if got := truncate("hello", 0); got != "" {
		t.Errorf("truncate(width 0) = %q, want empty", got)
	}
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate of a short string = %q, want unchanged", got)
	}
	// Wide runes count as two cells, so the result must be measured, not counted.
	got := truncate("こんにちは世界", 6)
	if w := ansi.StringWidth(got); w > 6 {
		t.Errorf("truncate produced width %d, want <= 6 (%q)", w, got)
	}
}

func TestPadReachesExactWidth(t *testing.T) {
	if w := ansi.StringWidth(pad("abc", 10)); w != 10 {
		t.Errorf("pad width = %d, want 10", w)
	}
	// Padding never truncates: an over-long string comes back untouched.
	if got := pad("abcdefghijk", 3); got != "abcdefghijk" {
		t.Errorf("pad shortened a long string: %q", got)
	}
}

func TestStateWord(t *testing.T) {
	en := Catalog(LangEN)
	if icon, _ := stateWord(gh.StateMerged, en); icon != "🎉" {
		t.Errorf("merged icon = %q, want 🎉", icon)
	}
	if icon, _ := stateWord(gh.StateClosed, en); icon != "🌙" {
		t.Errorf("closed icon = %q, want 🌙", icon)
	}
}
