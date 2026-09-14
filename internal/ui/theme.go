package ui

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

// Theme holds every color and style the UI draws with, resolved once per
// background-color change so rendering never has to branch on light/dark.
type Theme struct {
	Dark bool

	Pink     color.Color
	Lavender color.Color
	Mint     color.Color
	Sky      color.Color
	Peach    color.Color
	Rose     color.Color
	Text     color.Color
	Subtle   color.Color
	Dim      color.Color

	Frame      lipgloss.Style
	Logo       lipgloss.Style
	Owners     lipgloss.Style
	Status     lipgloss.Style
	StatusWarm lipgloss.Style
	Rule       lipgloss.Style

	TabActive   lipgloss.Style
	TabIdle     lipgloss.Style
	TabCount    lipgloss.Style
	TabDivider  lipgloss.Style
	TabSelected lipgloss.Style

	Pointer   lipgloss.Style
	Number    lipgloss.Style
	Title     lipgloss.Style
	TitleOn   lipgloss.Style
	Meta      lipgloss.Style
	MetaOn    lipgloss.Style
	RepoTag   lipgloss.Style
	Label     lipgloss.Style
	Additions lipgloss.Style
	Deletions lipgloss.Style

	Party     lipgloss.Style
	PartyFade lipgloss.Style
	Farewell  lipgloss.Style

	Empty      lipgloss.Style
	Error      lipgloss.Style
	Detail     lipgloss.Style
	DetailKey  lipgloss.Style
	DetailBody lipgloss.Style
	Scrollbar  lipgloss.Style
	FilterIcon lipgloss.Style
}

// NewTheme builds the palette for a light or dark terminal background.
//
// Every color is picked as a light/dark pair rather than a single value: the
// pastel-pop palette that reads as "cute" on a dark terminal turns into
// unreadable highlighter on a white one.
func NewTheme(dark bool) Theme {
	pick := lipgloss.LightDark(dark)

	t := Theme{
		Dark:     dark,
		Pink:     pick(lipgloss.Color("#D6337E"), lipgloss.Color("#FF6FB5")),
		Lavender: pick(lipgloss.Color("#7C4DBF"), lipgloss.Color("#B794F6")),
		Mint:     pick(lipgloss.Color("#0E9F6E"), lipgloss.Color("#5BE9B9")),
		Sky:      pick(lipgloss.Color("#0A7EA4"), lipgloss.Color("#6FD3FF")),
		Peach:    pick(lipgloss.Color("#C2670A"), lipgloss.Color("#FFB86C")),
		Rose:     pick(lipgloss.Color("#D01F41"), lipgloss.Color("#FF5C7A")),
		Text:     pick(lipgloss.Color("#1F2033"), lipgloss.Color("#E6E6F0")),
		Subtle:   pick(lipgloss.Color("#5C5C7A"), lipgloss.Color("#A6A6C0")),
		Dim:      pick(lipgloss.Color("#9A9AB5"), lipgloss.Color("#6C6C8A")),
	}

	base := lipgloss.NewStyle()

	t.Frame = base.
		Border(lipgloss.ThickBorder()).
		BorderForeground(t.Pink).
		Padding(0, 1)
	t.Logo = base.Bold(true).Foreground(t.Pink)
	t.Owners = base.Foreground(t.Subtle)
	t.Status = base.Foreground(t.Lavender)
	t.StatusWarm = base.Foreground(t.Peach)
	t.Rule = base.Foreground(t.Dim)

	t.TabActive = base.Bold(true).Foreground(t.Pink)
	t.TabIdle = base.Foreground(t.Subtle)
	t.TabCount = base.Foreground(t.Dim)
	t.TabDivider = base.Foreground(t.Dim)
	t.TabSelected = base.Bold(true).Foreground(t.Mint)

	t.Pointer = base.Bold(true).Foreground(t.Pink)
	t.Number = base.Foreground(t.Lavender)
	t.Title = base.Foreground(t.Text)
	t.TitleOn = base.Bold(true).Foreground(t.Pink)
	t.Meta = base.Foreground(t.Dim)
	t.MetaOn = base.Foreground(t.Subtle)
	t.RepoTag = base.Foreground(t.Sky)
	t.Label = base.Foreground(t.Lavender)
	t.Additions = base.Foreground(t.Mint)
	t.Deletions = base.Foreground(t.Rose)

	t.Party = base.Bold(true).Foreground(t.Mint)
	t.PartyFade = base.Foreground(t.Dim)
	t.Farewell = base.Foreground(t.Subtle)

	t.Empty = base.Foreground(t.Subtle)
	t.Error = base.Foreground(t.Rose)
	t.Detail = base.Foreground(t.Text)
	t.DetailKey = base.Bold(true).Foreground(t.Lavender)
	t.DetailBody = base.Foreground(t.Subtle)
	t.Scrollbar = base.Foreground(t.Dim)
	t.FilterIcon = base.Foreground(t.Pink)

	return t
}

// HelpStyles adapts the help bubble's default styling to prpr's palette. The
// stock styles are a near-invisible gray, which buries the only on-screen hint
// about what the keys do.
func (t Theme) HelpStyles() help.Styles {
	s := help.DefaultStyles(t.Dark)
	s.ShortKey = s.ShortKey.Foreground(t.Lavender)
	s.ShortDesc = s.ShortDesc.Foreground(t.Subtle)
	s.ShortSeparator = s.ShortSeparator.Foreground(t.Dim)
	s.FullKey = s.FullKey.Foreground(t.Lavender)
	s.FullDesc = s.FullDesc.Foreground(t.Subtle)
	s.FullSeparator = s.FullSeparator.Foreground(t.Dim)
	s.Ellipsis = s.Ellipsis.Foreground(t.Dim)
	return s
}
