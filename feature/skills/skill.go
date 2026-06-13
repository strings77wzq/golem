// Package skills provides a composable skill registry. Each [Skill] bundles a
// system prompt with metadata and optional tool-chain steps. The agent selects
// the appropriate skill and can execute its steps as a workflow.
package skills

// Skill represents a skill with metadata, prompts, and optional workflow steps.
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author,omitempty"`
	Prompts     []Prompt `json:"prompts,omitempty"`
	Tools       []string `json:"tools,omitempty"`
	Steps       []Step   `json:"steps,omitempty"`
	Condition   string   `json:"condition,omitempty"`
	Path        string   `json:"-"`
}

// Prompt represents a prompt within a skill.
type Prompt struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	File    string `json:"file,omitempty"`
}

// Step defines a single step in a skill's tool-chain workflow.
type Step struct {
	Tool       string            `json:"tool"`
	Input      map[string]string `json:"input,omitempty"`
	OutputVar  string            `json:"output_var,omitempty"`
	Condition  string            `json:"condition,omitempty"`
}

// HasSteps returns true if the skill defines a tool-chain workflow.
func (s *Skill) HasSteps() bool {
	return len(s.Steps) > 0
}

// HasCondition returns true if the skill has an activation condition.
func (s *Skill) HasCondition() bool {
	return s.Condition != ""
}
