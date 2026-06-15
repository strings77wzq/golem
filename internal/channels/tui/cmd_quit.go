package tui

import tea "github.com/charmbracelet/bubbletea"

type quitCmd struct{}

func (c quitCmd) Name() string        { return "/quit" }
func (c quitCmd) Description() string { return "Exit the application" }
func (c quitCmd) Execute(m *Model) tea.Cmd {
	m.cancel()
	return tea.Quit
}
