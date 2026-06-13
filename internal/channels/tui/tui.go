// Package tui implements the interactive Bubble Tea terminal UI for the AI
// agent. It is the ONLY package in the entire codebase that may import
// github.com/charmbracelet/bubbletea — all other packages must remain
// free of that dependency. Token streaming is driven by a recursive Cmd chain
// (waitNextToken) that consumes a chan string without additional goroutines.
// The TUI auto-activates when stdin is a TTY; use --no-tui to suppress it.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/strings77wzq/golem/core/bus"
)

// MessageHandler is the subset of agent.MessageHandler needed by the TUI.
type MessageHandler interface {
	HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error
	HandleMessageStreamWithProgress(ctx context.Context, sessionID string, message string, tokens chan<- string, progress chan<- bus.OutboundMessage) error
}

type role int

const (
	roleUser role = iota
	roleAssistant
	roleProgress
)

type chatMsg struct {
	role    role
	content string
}

type tokenMsg string
type doneMsg struct{ err error }
type progressMsg struct {
	content      string
	progressType string
}

var (
	userStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	assistantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	promptStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	planStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))  // blue
	stepStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // green
	toolStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
	resultStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))  // magenta
)

// Model is the Bubble Tea model for the chat TUI.
type Model struct {
	agent      MessageHandler
	sessionID  string
	ctx        context.Context
	cancel     context.CancelFunc

	messages    []chatMsg
	tokenCh     <-chan string
	progressCh  <-chan bus.OutboundMessage
	input       string
	thinking    bool
	lastError   string

	viewport viewport.Model
	ready    bool
	width    int
	height   int
	atBottom bool
}

func New(ctx context.Context, sessionID string, handler MessageHandler) Model {
	childCtx, cancel := context.WithCancel(ctx)
	return Model{
		agent:     handler,
		sessionID: sessionID,
		ctx:       childCtx,
		cancel:    cancel,
		atBottom:  true,
	}
}

// SetProgressChannel sets a channel for receiving progress messages.
func (m *Model) SetProgressChannel(ch <-chan bus.OutboundMessage) {
	m.progressCh = ch
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		inputHeight := 2
		viewportHeight := max(1, m.height-inputHeight)

		if !m.ready {
			m.viewport = viewport.New(m.width, viewportHeight)
			m.viewport.SetContent(m.buildTranscript())
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = viewportHeight
			m.viewport.SetContent(m.buildTranscript())
		}

		if m.atBottom {
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tokenMsg:
		wasAtBottom := m.ready && m.atBottom && m.viewport.AtBottom()

		if len(m.messages) > 0 && m.messages[len(m.messages)-1].role == roleAssistant {
			m.messages[len(m.messages)-1].content += string(msg)
		} else {
			m.messages = append(m.messages, chatMsg{role: roleAssistant, content: string(msg)})
		}

		if m.ready {
			m.viewport.SetContent(m.buildTranscript())
			if wasAtBottom {
				m.viewport.GotoBottom()
			}
		}
		return m, waitNextToken(m.tokenCh)

	case progressMsg:
		wasAtBottom := m.ready && m.atBottom && m.viewport.AtBottom()

		// Parse progress type from content prefix
		progressType, content := parseProgress(msg.content)
		if msg.progressType != "" {
			progressType = msg.progressType
		}
		m.messages = append(m.messages, chatMsg{role: roleProgress, content: formatProgress(progressType, content)})

		if m.ready {
			m.viewport.SetContent(m.buildTranscript())
			if wasAtBottom {
				m.viewport.GotoBottom()
			}
		}

		// Continue listening for more progress
		if m.progressCh != nil {
			return m, waitNextProgress(m.progressCh)
		}
		return m, nil

	case doneMsg:
		m.thinking = false
		m.tokenCh = nil
		if msg.err != nil {
			m.lastError = msg.err.Error()
		}
		if m.ready {
			m.viewport.SetContent(m.buildTranscript())
			if m.atBottom {
				m.viewport.GotoBottom()
			}
		}
		return m, nil
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		m.atBottom = m.viewport.AtBottom()
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.ready {
			m.viewport.ScrollUp(1)
			m.atBottom = m.viewport.AtBottom()
			return m, nil
		}
	case tea.KeyDown:
		if m.ready {
			m.viewport.ScrollDown(1)
			m.atBottom = m.viewport.AtBottom()
			return m, nil
		}
	case tea.KeyPgUp:
		if m.ready {
			m.viewport.HalfPageUp()
			m.atBottom = m.viewport.AtBottom()
			return m, nil
		}
	case tea.KeyPgDown:
		if m.ready {
			m.viewport.HalfPageDown()
			m.atBottom = m.viewport.AtBottom()
			return m, nil
		}
	case tea.KeyHome:
		if m.ready {
			m.viewport.GotoTop()
			m.atBottom = false
			return m, nil
		}
	case tea.KeyEnd:
		if m.ready {
			m.viewport.GotoBottom()
			m.atBottom = true
			return m, nil
		}
	}

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancel()
		return m, tea.Quit

	case tea.KeyRunes:
		if string(msg.Runes) == "q" && !m.thinking && m.input == "" {
			m.cancel()
			return m, tea.Quit
		}
		if !m.thinking {
			m.input += string(msg.Runes)
		}

	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	case tea.KeySpace:
		if !m.thinking {
			m.input += " "
		}

	case tea.KeyEnter:
		if m.thinking || strings.TrimSpace(m.input) == "" {
			return m, nil
		}
		text := m.input
		m.input = ""
		m.thinking = true
		m.lastError = ""
		m.messages = append(m.messages, chatMsg{role: roleUser, content: text})

		if m.ready {
			m.viewport.SetContent(m.buildTranscript())
			m.viewport.GotoBottom()
			m.atBottom = true
		}

		tokens := make(chan string, 64)
		progress := make(chan bus.OutboundMessage, 64)
		m.tokenCh = tokens
		m.progressCh = progress
		return m, tea.Batch(m.startStream(text, tokens, progress), waitNextToken(tokens), waitNextProgress(progress))
	}

	return m, nil
}

func (m Model) startStream(text string, tokens chan<- string, progress chan<- bus.OutboundMessage) tea.Cmd {
	return func() tea.Msg {
		err := m.agent.HandleMessageStreamWithProgress(m.ctx, m.sessionID, text, tokens, progress)
		if err != nil {
			return doneMsg{err: err}
		}
		return nil
	}
}

func waitNextToken(tokens <-chan string) tea.Cmd {
	if tokens == nil {
		return nil
	}
	return func() tea.Msg {
		tok, ok := <-tokens
		if !ok {
			return doneMsg{}
		}
		return tokenMsg(tok)
	}
}

func waitNextProgress(ch <-chan bus.OutboundMessage) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return progressMsg{content: msg.Content, progressType: string(msg.ProgressType)}
	}
}

// parseProgress extracts the progress type from content like "[Planning] ..." or "[Step 1/3] ..."
func parseProgress(content string) (string, string) {
	if strings.HasPrefix(content, "[Planning]") {
		return "plan", strings.TrimPrefix(content, "[Planning] ")
	}
	if strings.HasPrefix(content, "[Step") {
		return "step", content
	}
	if strings.HasPrefix(content, "[Tool]") {
		return "tool", strings.TrimPrefix(content, "[Tool] ")
	}
	if strings.HasPrefix(content, "[Result]") {
		return "result", strings.TrimPrefix(content, "[Result] ")
	}
	if strings.HasPrefix(content, "[Error]") {
		return "error", strings.TrimPrefix(content, "[Error] ")
	}
	if strings.HasPrefix(content, "[Plan") {
		return "plan", content
	}
	if strings.HasPrefix(content, "[Exec") {
		return "step", content
	}
	return "info", content
}

// formatProgress renders a progress line with color.
func formatProgress(progressType, content string) string {
	switch progressType {
	case "plan":
		return planStyle.Render("  " + content)
	case "step":
		return stepStyle.Render("  " + content)
	case "tool":
		return toolStyle.Render("  " + content)
	case "result":
		return resultStyle.Render("  " + content)
	case "error":
		return errorStyle.Render("  " + content)
	default:
		return "  " + content
	}
}

func (m Model) buildTranscript() string {
	var b strings.Builder

	for _, msg := range m.messages {
		switch msg.role {
		case roleUser:
			b.WriteString(userStyle.Render("You: "))
			b.WriteString(msg.content)
		case roleAssistant:
			b.WriteString(assistantStyle.Render("AI:  "))
			b.WriteString(msg.content)
		case roleProgress:
			b.WriteString(msg.content)
		}
		b.WriteString("\n")
	}

	if m.lastError != "" {
		b.WriteString(errorStyle.Render(fmt.Sprintf("error: %s", m.lastError)))
		b.WriteString("\n")
	}

	if m.thinking {
		b.WriteString(promptStyle.Render("AI:  ") + "…\n")
	}

	return b.String()
}

func (m Model) View() string {
	if !m.ready {
		return m.buildTranscript() + promptStyle.Render("> ") + m.input
	}

	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	b.WriteString(promptStyle.Render("> "))
	b.WriteString(m.input)

	return b.String()
}

// Run starts the TUI program and blocks until the user quits.
func Run(ctx context.Context, sessionID string, handler MessageHandler) error {
	m := New(ctx, sessionID, handler)
	p := tea.NewProgram(m, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}
