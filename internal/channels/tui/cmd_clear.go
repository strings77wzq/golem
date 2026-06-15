package tui

import tea "github.com/charmbracelet/bubbletea"

type clearCmd struct{}

func (c clearCmd) Name() string        { return "/clear" }
func (c clearCmd) Description() string { return "Clear conversation history" }
func (c clearCmd) Execute(m *Model) tea.Cmd {
	m.messages = nil
	m.lastError = ""
	if m.ready {
		m.viewport.SetContent(m.buildTranscript())
	}
	return nil
}
