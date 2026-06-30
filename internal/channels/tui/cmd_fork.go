package tui

import tea "github.com/charmbracelet/bubbletea"

type forkCmd struct{}

func (c forkCmd) Name() string        { return "/fork" }
func (c forkCmd) Description() string { return "Fork session from a message index" }
func (c forkCmd) Execute(_ *Model) tea.Cmd {
	return func() tea.Msg {
		return forkHelpMsg{}
	}
}

type forkHelpMsg struct{}

type forkResultMsg struct {
	newSessionID string
	message      string
}
