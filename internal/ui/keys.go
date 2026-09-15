package ui

import "charm.land/bubbles/v2/key"

// KeyMap is every binding prpr responds to in list mode. It satisfies
// help.KeyMap, so the footer help is generated from the same source of truth
// that Update dispatches on.
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Top         key.Binding
	Bottom      key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	NextTab     key.Binding
	PrevTab     key.Binding
	Open        key.Binding
	Detail      key.Binding
	DetailUp    key.Binding
	DetailDown  key.Binding
	Copy        key.Binding
	Refresh     key.Binding
	Filter      key.Binding
	ClearFilter key.Binding
	Help        key.Binding
	Suspend     key.Binding
	Quit        key.Binding
}

// DefaultKeyMap returns the stock bindings, with help text in the active
// language.
func DefaultKeyMap(s Strings) KeyMap {
	return KeyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", s.HelpUp)),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", s.HelpDown)),
		Top:         key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", s.HelpTop)),
		Bottom:      key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", s.HelpBottom)),
		PageUp:      key.NewBinding(key.WithKeys("pgup", "ctrl+b"), key.WithHelp("pgup", s.HelpPageUp)),
		PageDown:    key.NewBinding(key.WithKeys("pgdown", "ctrl+f"), key.WithHelp("pgdn", s.HelpPageDown)),
		NextTab:     key.NewBinding(key.WithKeys("tab", "right", "l"), key.WithHelp("tab", s.HelpNextTab)),
		PrevTab:     key.NewBinding(key.WithKeys("shift+tab", "left", "h"), key.WithHelp("⇧tab", s.HelpPrevTab)),
		Open:        key.NewBinding(key.WithKeys("enter", "o"), key.WithHelp("⏎", s.HelpOpen)),
		Detail:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", s.HelpDetail)),
		DetailUp:    key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", s.HelpDetailUp)),
		DetailDown:  key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", s.HelpDetailDown)),
		Copy:        key.NewBinding(key.WithKeys("y"), key.WithHelp("y", s.HelpCopy)),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", s.HelpRefresh)),
		Filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", s.HelpFilter)),
		ClearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", s.HelpClearFilter)),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", s.HelpHelp)),
		Suspend:     key.NewBinding(key.WithKeys("ctrl+z"), key.WithHelp("ctrl+z", s.HelpSuspend)),
		// esc is deliberately not a quit key. It is what clears the filter,
		// and the two collided: accepting a filter with enter and then
		// pressing esc to clear it quit the program instead.
		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", s.HelpQuit)),
	}
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Open, k.Refresh, k.NextTab, k.Filter, k.Detail, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom},
		{k.PageUp, k.PageDown, k.PrevTab, k.NextTab},
		{k.Open, k.Copy, k.Detail, k.Refresh},
		{k.DetailUp, k.DetailDown, k.Filter, k.ClearFilter},
		{k.Help, k.Suspend, k.Quit},
	}
}

// FilterKeyMap is the reduced set of bindings active while the filter input
// has focus. Everything else is text, and belongs to the input.
type FilterKeyMap struct {
	Accept key.Binding
	Cancel key.Binding
}

// DefaultFilterKeyMap returns the stock filter-mode bindings.
func DefaultFilterKeyMap(s Strings) FilterKeyMap {
	return FilterKeyMap{
		Accept: key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", s.HelpAccept)),
		Cancel: key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", s.HelpCancel)),
	}
}

// ShortHelp implements help.KeyMap.
func (k FilterKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Accept, k.Cancel}
}

// FullHelp implements help.KeyMap.
func (k FilterKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
