package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type helpCmd struct {
	registry *CommandRegistry
}

func (c helpCmd) Name() string        { return "/help" }
func (c helpCmd) Description() string { return "Show available commands" }
func (c helpCmd) Execute(m *Model) tea.Cmd {
	var b strings.Builder
	b.WriteString("Available commands:\n")
	for _, cmd := range c.registry.List() {
		b.WriteString(fmt.Sprintf("  %-12s — %s\n", cmd.Name(), cmd.Description()))
	}
	m.messages = append(m.messages, chatMsg{role: roleProgress, content: b.String()})
	if m.ready {
		m.viewport.SetContent(m.buildTranscript())
		m.viewport.GotoBottom()
		m.atBottom = true
	}
	return nil
}
