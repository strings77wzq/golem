package agent

import (
	"sort"
	"strings"

	"github.com/strings77wzq/golem/core/tools"
)

// ToolSelector selects the most relevant tools for a given step.
type ToolSelector struct {
	registry *tools.Registry
	stopWords map[string]bool
}

// NewToolSelector creates a new tool selector.
func NewToolSelector(registry *tools.Registry) *ToolSelector {
	return &ToolSelector{
		registry: registry,
		stopWords: map[string]bool{
			"the": true, "a": true, "an": true, "is": true, "are": true,
			"was": true, "were": true, "be": true, "been": true, "being": true,
			"have": true, "has": true, "had": true, "do": true, "does": true,
			"did": true, "will": true, "would": true, "could": true, "should": true,
			"may": true, "might": true, "shall": true, "can": true,
			"to": true, "of": true, "in": true, "for": true, "on": true,
			"with": true, "at": true, "by": true, "from": true, "as": true,
			"into": true, "through": true, "during": true, "before": true,
			"after": true, "above": true, "below": true, "between": true,
			"out": true, "off": true, "over": true, "under": true,
			"again": true, "further": true, "then": true, "once": true,
			"here": true, "there": true, "when": true, "where": true,
			"why": true, "how": true, "all": true, "both": true,
			"each": true, "few": true, "more": true, "most": true,
			"other": true, "some": true, "such": true, "no": true,
			"nor": true, "not": true, "only": true, "own": true,
			"same": true, "so": true, "than": true, "too": true,
			"very": true, "just": true, "because": true, "but": true,
			"and": true, "if": true, "or": true,
		},
	}
}

// Select returns the most relevant tools for a step.
func (ts *ToolSelector) Select(stepDescription string, toolHints []string, maxTools int) []tools.ToolDefinition {
	allTools := ts.registry.ListDefinitions()
	if len(allTools) == 0 {
		return nil
	}

	// Score each tool
	type scored struct {
		tool  tools.ToolDefinition
		score int
	}

	var scoredTools []scored
	for _, td := range allTools {
		score := ts.scoreTool(td, stepDescription, toolHints)
		scoredTools = append(scoredTools, scored{tool: td, score: score})
	}

	// Sort by score descending
	sort.Slice(scoredTools, func(i, j int) bool {
		return scoredTools[i].score > scoredTools[j].score
	})

	// Take top N
	if maxTools <= 0 {
		maxTools = 8
	}
	if maxTools > len(scoredTools) {
		maxTools = len(scoredTools)
	}

	result := make([]tools.ToolDefinition, maxTools)
	for i := 0; i < maxTools; i++ {
		result[i] = scoredTools[i].tool
	}

	return result
}

// scoreTool calculates relevance score for a tool.
func (ts *ToolSelector) scoreTool(td tools.ToolDefinition, stepDescription string, toolHints []string) int {
	score := 0
	stepKeywords := ts.extractKeywords(stepDescription)

	// ToolHints match (highest priority)
	for _, hint := range toolHints {
		if strings.EqualFold(td.Name, hint) {
			score += 10
		}
	}

	// Name match
	nameLower := strings.ToLower(td.Name)
	for _, keyword := range stepKeywords {
		if strings.Contains(nameLower, keyword) {
			score += 5
		}
	}

	// Description match
	descLower := strings.ToLower(td.Description)
	for _, keyword := range stepKeywords {
		if strings.Contains(descLower, keyword) {
			score += 1
		}
	}

	return score
}

// extractKeywords tokenizes a string into keywords.
func (ts *ToolSelector) extractKeywords(s string) []string {
	words := strings.Fields(strings.ToLower(s))
	var keywords []string
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,;:!?\"'()-")
		if len(word) < 2 {
			continue
		}
		if ts.stopWords[word] {
			continue
		}
		keywords = append(keywords, word)
	}
	return keywords
}
