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
