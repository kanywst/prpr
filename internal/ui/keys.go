package ui

import "charm.land/bubbles/v2/key"

// KeyMap is every binding prpr responds to in list mode. It satisfies
// help.KeyMap, so the footer help is generated from the same source of truth
// that Update dispatches on.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Top        key.Binding
	Bottom     key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	NextTab    key.Binding
	PrevTab    key.Binding
	Open       key.Binding
	Detail     key.Binding
	DetailUp   key.Binding
	DetailDown key.Binding
	Copy       key.Binding
	Refresh    key.Binding
	Filter     key.Binding
	Help       key.Binding
	Suspend    key.Binding
	Quit       key.Binding
}

// DefaultKeyMap returns the stock bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "上へ")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "下へ")),
		Top:        key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", "先頭")),
		Bottom:     key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", "末尾")),
		PageUp:     key.NewBinding(key.WithKeys("pgup", "ctrl+b"), key.WithHelp("pgup", "前ページ")),
		PageDown:   key.NewBinding(key.WithKeys("pgdown", "ctrl+f"), key.WithHelp("pgdn", "次ページ")),
		NextTab:    key.NewBinding(key.WithKeys("tab", "right", "l"), key.WithHelp("tab", "次のタブ")),
		PrevTab:    key.NewBinding(key.WithKeys("shift+tab", "left", "h"), key.WithHelp("⇧tab", "前のタブ")),
		Open:       key.NewBinding(key.WithKeys("enter", "o"), key.WithHelp("⏎", "ブラウザで開く")),
		Detail:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "詳細")),
		DetailUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "詳細を上へ")),
		DetailDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "詳細を下へ")),
		Copy:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "URL コピー")),
		Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "更新")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "絞り込み")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "ヘルプ")),
		Suspend:    key.NewBinding(key.WithKeys("ctrl+z"), key.WithHelp("ctrl+z", "一時停止")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("q", "バイバイ")),
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
		{k.DetailUp, k.DetailDown, k.Filter, k.Help},
		{k.Suspend, k.Quit},
	}
}

// FilterKeyMap is the reduced set of bindings active while the filter input
// has focus. Everything else is text, and belongs to the input.
type FilterKeyMap struct {
	Accept key.Binding
	Cancel key.Binding
}

// DefaultFilterKeyMap returns the stock filter-mode bindings.
func DefaultFilterKeyMap() FilterKeyMap {
	return FilterKeyMap{
		Accept: key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "確定")),
		Cancel: key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "やめる")),
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
