package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// newToolsCmd lists all available tools.
type newToolsCmd struct{}

func (c newToolsCmd) Name() string        { return "/tools" }
func (c newToolsCmd) Description() string { return "List available tools" }
func (c newToolsCmd) Execute(m *Model) tea.Cmd {
	var b strings.Builder
	b.WriteString("Available commands:\n")
	b.WriteString("  /tools       — List available tools\n")
	b.WriteString("  /new         — Start a new session\n")
	b.WriteString("  /sessions    — Browse past sessions\n")
	b.WriteString("  /model       — Switch the current model\n")
	b.WriteString("  /compact     — Compact conversation history\n")
	b.WriteString("  /clear       — Clear the current conversation\n")
	b.WriteString("  /fork        — Fork the current session\n")
	b.WriteString("  /help        — Show this help\n")
	b.WriteString("  /quit        — Exit the application\n")

	m.messages = append(m.messages, chatMsg{role: roleProgress, content: b.String()})
	if m.ready {
		m.viewport.SetContent(m.buildTranscript())
		m.viewport.GotoBottom()
		m.atBottom = true
	}
	return nil
}

// newSessionCmd starts a new conversation session.
type newSessionCmd struct{}

func (c newSessionCmd) Name() string        { return "/new" }
func (c newSessionCmd) Description() string { return "Start a new session" }
func (c newSessionCmd) Execute(m *Model) tea.Cmd {
	m.messages = nil
	m.sessionID = ""
	m.input = ""
	m.thinking = false
	m.lastError = ""

	m.messages = append(m.messages, chatMsg{
		role:    roleProgress,
		content: "New session started. Type your message to begin.",
	})
	if m.ready {
		m.viewport.SetContent(m.buildTranscript())
		m.viewport.GotoBottom()
		m.atBottom = true
	}
	return nil
}

// sessionsCmd lists past sessions.
type sessionsCmd struct{}

func (c sessionsCmd) Name() string        { return "/sessions" }
func (c sessionsCmd) Description() string { return "Browse past sessions" }
func (c sessionsCmd) Execute(m *Model) tea.Cmd {
	m.messages = append(m.messages, chatMsg{
		role:    roleProgress,
		content: "Session history will be available when session persistence is enabled.\nUse /new to start a fresh session.",
	})
	if m.ready {
		m.viewport.SetContent(m.buildTranscript())
		m.viewport.GotoBottom()
		m.atBottom = true
	}
	return nil
}

// modelCmd shows or switches the current model.
type modelCmd struct{}

func (c modelCmd) Name() string        { return "/model" }
func (c modelCmd) Description() string { return "Switch the current model" }
func (c modelCmd) Execute(m *Model) tea.Cmd {
	content := fmt.Sprintf("Current session: %s\nModel switching will be available when provider configuration is loaded.\nUse 'golem agent -M <model>' to start with a specific model.", m.sessionID)

	m.messages = append(m.messages, chatMsg{role: roleProgress, content: content})
	if m.ready {
		m.viewport.SetContent(m.buildTranscript())
		m.viewport.GotoBottom()
		m.atBottom = true
	}
	return nil
}
