package ui

import "strings"

// Lang selects the interface language.
type Lang string

// Supported languages. English is the default: prpr is published where most
// readers do not read Japanese, and the Japanese interface is opt-in.
const (
	LangEN Lang = "en"
	LangJA Lang = "ja"
)

// ParseLang resolves a --lang value. The bool reports whether the value was
// recognized; the Lang is English either way, so a caller that chooses to
// carry on has something usable.
//
// prpr itself does not carry on: main treats an unrecognized value as a fatal
// flag error, because silently running in a language the user did not ask for
// is worse than refusing to start.
func ParseLang(s string) (Lang, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "en", "en-us", "en_us", "english":
		return LangEN, true
	case "ja", "jp", "ja-jp", "ja_jp", "japanese", "日本語":
		return LangJA, true
	default:
		return LangEN, false
	}
}

// Strings is every piece of text the interface can show. Keeping it as one
// flat struct rather than a map means a missing translation is a compile
// error instead of an empty label at runtime.
type Strings struct {
	TabAll    string
	TabMine   string
	TabReview string
	TabDraft  string

	Collecting string
	Failed     string
	Paused     string
	OpenCount  string // one %d

	AgeNow   string
	AgeMin   string // one %d
	AgeHour  string // one %d
	AgeDay   string // one %d
	AgeWeek  string // one %d
	AgeMonth string // one %d
	AgeAgo   string // one %s

	FarewellMerged  string
	FarewellClosed  string
	FarewellDropped string

	EmptyAll        string
	EmptyMine       string
	EmptyReview     string
	EmptyDraft      string
	EmptyFilter     string
	ErrorHint       string // one %s: the error
	TooSmall        string
	NothingSelected string
	Loading         string

	DetailState     string
	DetailAuthor    string
	DetailUpdated   string
	DetailCreated   string
	DetailBranch    string
	DetailDiff      string
	DetailComments  string
	DetailLabels    string
	DetailReviewers string
	DetailURL       string
	FilesSuffix     string // one %d

	CheckDraft string
	CheckPass  string
	CheckRun   string
	CheckFail  string
	CheckNone  string

	ReviewApproved string
	ReviewChanges  string
	ReviewRequired string

	OpenedInBrowser string
	CopiedURL       string
	ClipboardFailed string

	WarnFailed string // one %s: the scopes that could not be searched
	WarnCapped string // %s: the scope, %d: how many it returned, %d: how many matched

	FilterPlaceholder string

	HelpUp          string
	HelpDown        string
	HelpTop         string
	HelpBottom      string
	HelpPageUp      string
	HelpPageDown    string
	HelpNextTab     string
	HelpPrevTab     string
	HelpOpen        string
	HelpDetail      string
	HelpDetailUp    string
	HelpDetailDown  string
	HelpCopy        string
	HelpRefresh     string
	HelpFilter      string
	HelpClearFilter string
	HelpHelp        string
	HelpSuspend     string
	HelpQuit        string
	HelpAccept      string
	HelpCancel      string
}

// Catalog returns the strings for a language.
func Catalog(l Lang) Strings {
	if l == LangJA {
		return japanese
	}
	return english
}

var english = Strings{
	TabAll:    "all",
	TabMine:   "mine",
	TabReview: "to review",
	TabDraft:  "drafts",

	Collecting: " collecting…",
	Failed:     "😿 failed",
	Paused:     "⏸ paused",
	OpenCount:  "prpr · %d open",

	AgeNow:   "just now",
	AgeMin:   "%dm",
	AgeHour:  "%dh",
	AgeDay:   "%dd",
	AgeWeek:  "%dw",
	AgeMonth: "%dmo",
	AgeAgo:   "%s ago",

	FarewellMerged:  "merged! nice one 🎊",
	FarewellClosed:  "closed",
	FarewellDropped: "left the list",

	EmptyAll:        "✨ no open pull requests — nice work ✨",
	EmptyMine:       "✨ nothing of yours is open ✨",
	EmptyReview:     "✨ your review queue is empty ✨",
	EmptyDraft:      "✨ no drafts ✨",
	EmptyFilter:     "🔍 nothing matched\n\nesc clears the filter",
	ErrorHint:       "😿 %s\n\nr to try again",
	TooSmall:        "🌸 a little too small…\ngive me some more room",
	NothingSelected: "nothing selected",
	Loading:         "fetching pull requests…",

	DetailState:     "state",
	DetailAuthor:    "author",
	DetailUpdated:   "updated",
	DetailCreated:   "created",
	DetailBranch:    "branch",
	DetailDiff:      "diff",
	DetailComments:  "comments",
	DetailLabels:    "labels",
	DetailReviewers: "reviewers",
	DetailURL:       "url",
	FilesSuffix:     "(%d files)",

	CheckDraft: "draft",
	CheckPass:  "checks passing",
	CheckRun:   "checks running",
	CheckFail:  "checks failing",
	CheckNone:  "no checks",

	ReviewApproved: "approved",
	ReviewChanges:  "changes requested",
	ReviewRequired: "review requested",

	OpenedInBrowser: "opened in the browser",
	CopiedURL:       "URL copied",
	ClipboardFailed: "the clipboard was not available",

	WarnFailed: "⚠ skipped %s",
	WarnCapped: "⚠ %s: %d of %d",

	FilterPlaceholder: "title / repo / author / #number",

	HelpUp:          "up",
	HelpDown:        "down",
	HelpTop:         "top",
	HelpBottom:      "bottom",
	HelpPageUp:      "page up",
	HelpPageDown:    "page down",
	HelpNextTab:     "next tab",
	HelpPrevTab:     "previous tab",
	HelpOpen:        "open in browser",
	HelpDetail:      "detail",
	HelpDetailUp:    "detail up",
	HelpDetailDown:  "detail down",
	HelpCopy:        "copy URL",
	HelpRefresh:     "refresh",
	HelpFilter:      "filter",
	HelpClearFilter: "clear filter",
	HelpHelp:        "help",
	HelpSuspend:     "suspend",
	HelpQuit:        "quit",
	HelpAccept:      "apply",
	HelpCancel:      "cancel",
}

var japanese = Strings{
	TabAll:    "すべて",
	TabMine:   "自分の",
	TabReview: "レビュー待ち",
	TabDraft:  "下書き",

	Collecting: " あつめてる…",
	Failed:     "😿 しっぱい",
	Paused:     "⏸ 休憩中",
	OpenCount:  "prpr · オープン %d 件",

	AgeNow:   "いま",
	AgeMin:   "%d分",
	AgeHour:  "%d時間",
	AgeDay:   "%d日",
	AgeWeek:  "%d週間",
	AgeMonth: "%dヶ月",
	AgeAgo:   "%s前",

	FarewellMerged:  "マージされたよ〜 おめでとう!",
	FarewellClosed:  "クローズされたよ",
	FarewellDropped: "一覧から外れたよ",

	EmptyAll:        "✨ PR ないよ〜 おつかれさま ✨",
	EmptyMine:       "✨ 自分の PR はないよ〜 ✨",
	EmptyReview:     "✨ レビュー待ちゼロ! えらい ✨",
	EmptyDraft:      "✨ 下書きはないよ ✨",
	EmptyFilter:     "🔍 みつからなかった\n\nesc で絞り込み解除",
	ErrorHint:       "😿 %s\n\nr でもう一回",
	TooSmall:        "🌸 ちいさすぎるかも…\nもうすこし広げてね",
	NothingSelected: "えらばれてないよ",
	Loading:         "PR あつめてるよ…",

	DetailState:     "状態",
	DetailAuthor:    "作者",
	DetailUpdated:   "更新",
	DetailCreated:   "作成",
	DetailBranch:    "ブランチ",
	DetailDiff:      "差分",
	DetailComments:  "コメント",
	DetailLabels:    "ラベル",
	DetailReviewers: "レビュー依頼",
	DetailURL:       "URL",
	FilesSuffix:     "(%d ファイル)",

	CheckDraft: "下書き",
	CheckPass:  "CI 通過",
	CheckRun:   "CI 実行中",
	CheckFail:  "CI 失敗",
	CheckNone:  "CI なし",

	ReviewApproved: "承認済み",
	ReviewChanges:  "変更依頼",
	ReviewRequired: "レビュー待ち",

	OpenedInBrowser: "ブラウザで開いたよ",
	CopiedURL:       "URL コピーしたよ",
	ClipboardFailed: "クリップボードが使えなかった",

	WarnFailed: "⚠ %s は取得失敗",
	WarnCapped: "⚠ %s は %d/%d 件のみ",

	FilterPlaceholder: "タイトル / リポ / 作者 / #番号",

	HelpUp:          "上へ",
	HelpDown:        "下へ",
	HelpTop:         "先頭",
	HelpBottom:      "末尾",
	HelpPageUp:      "前ページ",
	HelpPageDown:    "次ページ",
	HelpNextTab:     "次のタブ",
	HelpPrevTab:     "前のタブ",
	HelpOpen:        "ブラウザで開く",
	HelpDetail:      "詳細",
	HelpDetailUp:    "詳細を上へ",
	HelpDetailDown:  "詳細を下へ",
	HelpCopy:        "URL コピー",
	HelpRefresh:     "更新",
	HelpFilter:      "絞り込み",
	HelpClearFilter: "絞り込み解除",
	HelpHelp:        "ヘルプ",
	HelpSuspend:     "一時停止",
	HelpQuit:        "バイバイ",
	HelpAccept:      "確定",
	HelpCancel:      "やめる",
}
