// Package builtins registers the default skills shipped with Golem.
package builtins

import "github.com/strings77wzq/golem/feature/skills"

const summarizeSystemPrompt = `You are a summarization assistant. Given a piece of text, produce a concise summary that captures the key points. Follow these rules:
- Keep the summary under 3 paragraphs
- Preserve important facts and numbers
- Use clear, direct language
- Do not add information not present in the original text`

const codeReviewSystemPrompt = `You are a code reviewer. Analyze the given code for bugs, security issues, performance problems, and style violations. Provide actionable feedback.`

func SummarizeSkill() *skills.Skill {
	return &skills.Skill{
		Name:        "summarize",
		Description: "Summarizes text into concise key points",
		Version:     "1.0.0",
		Author:      "Golem",
		Condition:   "",
		Prompts: []skills.Prompt{
			{Name: "system", Content: summarizeSystemPrompt},
		},
		Tools: []string{"file_read"},
		Steps: []skills.Step{
			{
				Tool:      "file_read",
				Input:     map[string]string{"path": "{{file_path}}"},
				OutputVar: "file_content",
				Condition: "var:file_path",
			},
		},
	}
}

func CodeReviewSkill() *skills.Skill {
	return &skills.Skill{
		Name:        "code-review",
		Description: "Reviews code for bugs, style issues, and improvements",
		Version:     "1.0.0",
		Author:      "Golem",
		Condition:   "",
		Prompts: []skills.Prompt{
			{Name: "system", Content: codeReviewSystemPrompt},
		},
		Tools: []string{"file_read"},
		Steps: []skills.Step{
			{
				Tool:      "file_read",
				Input:     map[string]string{"path": "{{file_path}}"},
				OutputVar: "code",
				Condition: "var:file_path",
			},
		},
	}
}

func RegisterAll(registry *skills.Registry) error {
	builtins := []*skills.Skill{
		SummarizeSkill(),
		CodeReviewSkill(),
	}
	for _, s := range builtins {
		if err := registry.Register(s); err != nil {
			return err
		}
	}
	return nil
}
