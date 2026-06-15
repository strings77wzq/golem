package tui

import tea "github.com/charmbracelet/bubbletea"

// SlashCommand is a TUI slash command.
type SlashCommand interface {
	Name() string
	Description() string
	Execute(m *Model) tea.Cmd
}

// CommandRegistry manages available slash commands.
type CommandRegistry struct {
	commands map[string]SlashCommand
}

// NewCommandRegistry creates a new empty command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]SlashCommand),
	}
}

// Register adds a command. Panics on duplicate names.
func (r *CommandRegistry) Register(cmd SlashCommand) {
	name := cmd.Name()
	if _, exists := r.commands[name]; exists {
		panic("duplicate slash command: " + name)
	}
	r.commands[name] = cmd
}

// Get returns a command by name, or nil if not found.
func (r *CommandRegistry) Get(name string) SlashCommand {
	return r.commands[name]
}

// List returns all registered commands.
func (r *CommandRegistry) List() []SlashCommand {
	result := make([]SlashCommand, 0, len(r.commands))
	for _, cmd := range r.commands {
		result = append(result, cmd)
	}
	return result
}
