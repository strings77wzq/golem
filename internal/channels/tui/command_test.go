package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandRegistry_RegisterAndGet(t *testing.T) {
	r := NewCommandRegistry()
	cmd := &mockSlashCmd{name: "/test", desc: "test command"}
	r.Register(cmd)

	got := r.Get("/test")
	if got == nil {
		t.Fatal("expected command to be found")
	}
	if got.Name() != "/test" {
		t.Errorf("expected '/test', got %q", got.Name())
	}
}

func TestCommandRegistry_GetNotFound(t *testing.T) {
	r := NewCommandRegistry()

	got := r.Get("/nonexistent")
	if got != nil {
		t.Error("expected nil for nonexistent command")
	}
}

func TestCommandRegistry_List(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(&mockSlashCmd{name: "/a", desc: "a"})
	r.Register(&mockSlashCmd{name: "/b", desc: "b"})
	r.Register(&mockSlashCmd{name: "/c", desc: "c"})

	list := r.List()
	if len(list) != 3 {
		t.Errorf("expected 3 commands, got %d", len(list))
	}
}

func TestSlashCommandInterface(t *testing.T) {
	cmd := &mockSlashCmd{name: "/help", desc: "Show help"}
	if cmd.Name() != "/help" {
		t.Errorf("expected '/help', got %q", cmd.Name())
	}
	if cmd.Description() != "Show help" {
		t.Errorf("expected 'Show help', got %q", cmd.Description())
	}
}

func TestSlashCommandExecute(t *testing.T) {
	cmd := &mockSlashCmd{name: "/test", desc: "test"}
	m := &Model{}
	result := cmd.Execute(m)
	if result != nil {
		t.Fatal("expected nil cmd from mock")
	}
}

// mockSlashCmd is a test implementation of SlashCommand.
type mockSlashCmd struct {
	name string
	desc string
}

func (m *mockSlashCmd) Name() string        { return m.name }
func (m *mockSlashCmd) Description() string { return m.desc }
func (m *mockSlashCmd) Execute(_ *Model) tea.Cmd {
	return nil
}

func TestNewToolsCommand(t *testing.T) {
	cmd := &newToolsCmd{}
	if cmd.Name() != "/tools" {
		t.Errorf("expected '/tools', got %q", cmd.Name())
	}
	if cmd.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestNewToolsCommandExecute(t *testing.T) {
	m := &Model{}
	cmd := &newToolsCmd{}
	result := cmd.Execute(m)
	if result != nil {
		t.Fatal("expected nil cmd")
	}
	if len(m.messages) == 0 {
		t.Error("expected tools message to be added")
	}
	if m.messages[0].role != roleProgress {
		t.Error("expected progress role")
	}
}

func TestNewSessionCommand(t *testing.T) {
	cmd := &newSessionCmd{}
	if cmd.Name() != "/new" {
		t.Errorf("expected '/new', got %q", cmd.Name())
	}
}

func TestSessionsCommand(t *testing.T) {
	cmd := &sessionsCmd{}
	if cmd.Name() != "/sessions" {
		t.Errorf("expected '/sessions', got %q", cmd.Name())
	}
}

func TestModelCommand(t *testing.T) {
	cmd := &modelCmd{}
	if cmd.Name() != "/model" {
		t.Errorf("expected '/model', got %q", cmd.Name())
	}
}
