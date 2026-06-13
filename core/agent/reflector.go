package agent

import (
	"strings"

	"github.com/strings77wzq/golem/core/planner"
)

// Reflector evaluates whether a step achieved its goal.
type Reflector struct{}

// NewReflector creates a new reflector.
func NewReflector() *Reflector {
	return &Reflector{}
}

// ReflectionResult represents the outcome of a reflection.
type ReflectionResult struct {
	Success      bool   `json:"success"`
	Reason       string `json:"reason"`
	ShouldRevise bool   `json:"should_revise"`
}

// Evaluate checks if a step achieved its expected outcome.
func (r *Reflector) Evaluate(step *planner.Step, result string, err error) ReflectionResult {
	// Error → failure
	if err != nil {
		return ReflectionResult{
			Success:      false,
			Reason:       err.Error(),
			ShouldRevise: true,
		}
	}

	// Empty result → failure
	if strings.TrimSpace(result) == "" {
		return ReflectionResult{
			Success:      false,
			Reason:       "step produced no output",
			ShouldRevise: true,
		}
	}

	// Keyword matching against expected outcome
	if step.ExpectedOut != "" {
		keywords := extractKeywords(step.ExpectedOut)
		resultLower := strings.ToLower(result)
		matches := 0
		for _, kw := range keywords {
			if strings.Contains(resultLower, kw) {
				matches++
			}
		}

		if len(keywords) > 0 {
			ratio := float64(matches) / float64(len(keywords))
			if ratio >= 0.5 {
				return ReflectionResult{
					Success:      true,
					Reason:       "result matches expected outcome",
					ShouldRevise: false,
				}
			}
			if ratio > 0 && ratio < 0.2 && len(result) < 100 {
				return ReflectionResult{
					Success:      false,
					Reason:       "partial match, result too short",
					ShouldRevise: true,
				}
			}
		}
	}

	// Default: assume success (optimistic)
	return ReflectionResult{
		Success:      true,
		Reason:       "result appears valid",
		ShouldRevise: false,
	}
}

// extractKeywords extracts meaningful keywords from text.
func extractKeywords(s string) []string {
	words := strings.Fields(strings.ToLower(s))
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "and": true, "or": true, "but": true,
	}
	var keywords []string
	for _, word := range words {
		word = strings.Trim(word, ".,;:!?\"'()-")
		if len(word) < 2 || stopWords[word] {
			continue
		}
		keywords = append(keywords, word)
	}
	return keywords
}
