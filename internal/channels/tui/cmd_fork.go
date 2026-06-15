package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type forkCmd struct{}

func (c forkCmd) Name() string        { return "/fork" }
func (c forkCmd) Description() string { return "Fork session from a message index" }
func (c forkCmd) Execute(m *Model) tea.Cmd {
	return func() tea.Msg {
		return forkHelpMsg{}
	}
}

// forkHelpMsg is displayed when /fork is invoked.
type forkHelpMsg struct{}

func parseForkArgs(input string) (int, string, bool) {
	// Format: /fork <index> <new message>
	rest := strings.TrimSpace(strings.TrimPrefix(input, "/fork"))
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return 0, "", false
	}
	idx, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", false
	}
	return idx, parts[1], true
}

// forkResultMsg is sent after a successful fork.
type forkResultMsg struct {
	newSessionID string
	message      string
}
