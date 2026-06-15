package tui

import tea "github.com/charmbracelet/bubbletea"

type compactCmd struct{}

func (c compactCmd) Name() string        { return "/compact" }
func (c compactCmd) Description() string { return "Compress conversation history" }
func (c compactCmd) Execute(m *Model) tea.Cmd {
	return m.doCompact()
}
