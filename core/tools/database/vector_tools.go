// Package database provides SQL tools for database operations.
package database

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/tools"
)

// VectorSearchTool performs semantic search on a vector database.
type VectorSearchTool struct {
	registry *database.Registry
}

func NewVectorSearchTool(registry *database.Registry) *VectorSearchTool {
	return &VectorSearchTool{registry: registry}
}

func (t *VectorSearchTool) Name() string        { return "vector_search" }
func (t *VectorSearchTool) Description() string { return "Semantic search in vector database" }
func (t *VectorSearchTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "database", Type: "string", Description: "Vector DB instance name", Required: false},
		{Name: "collection", Type: "string", Description: "Collection name", Required: true},
		{Name: "query", Type: "string", Description: "Search query", Required: true},
		{Name: "top_k", Type: "number", Description: "Number of results (default 5)", Required: false},
	}
}

func (t *VectorSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	dbName := t.getDefaultDB(args)
	collection, _ := args["collection"].(string)
	query, _ := args["query"].(string)
	if collection == "" || query == "" {
		return &tools.ToolResult{ForLLM: "Error: collection and query are required", IsError: true}, nil
	}

	topK := 5
	if tk, ok := args["top_k"].(float64); ok {
		topK = int(tk)
	}

	driver, err := t.registry.GetVector(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	results, err := driver.Search(ctx, collection, query, topK)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Search error: %v", err), IsError: true}, nil
	}

	if len(results) == 0 {
		return &tools.ToolResult{ForLLM: "No results found.", ForUser: "No matches found."}, nil
	}

	result := fmt.Sprintf("Found %d results:\n", len(results))
	for i, r := range results {
		result += fmt.Sprintf("%d. [Score: %.4f] ID: %s\n", i+1, r.Score, r.ID)
	}

	return &tools.ToolResult{ForLLM: result, ForUser: fmt.Sprintf("Found %d results", len(results))}, nil
}

func (t *VectorSearchTool) getDefaultDB(args map[string]interface{}) string {
	if db, ok := args["database"].(string); ok && db != "" {
		return db
	}
	return t.registry.Default()
}

// VectorCollectionsTool lists all collections in a vector database.
type VectorCollectionsTool struct {
	registry *database.Registry
}

func NewVectorCollectionsTool(registry *database.Registry) *VectorCollectionsTool {
	return &VectorCollectionsTool{registry: registry}
}

func (t *VectorCollectionsTool) Name() string { return "vector_collections" }
func (t *VectorCollectionsTool) Description() string {
	return "List all collections in vector database"
}
func (t *VectorCollectionsTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "database", Type: "string", Description: "Vector DB instance name", Required: false},
	}
}

func (t *VectorCollectionsTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	dbName := t.getDefaultDB(args)

	driver, err := t.registry.GetVector(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	schema, err := driver.GetSchema(ctx)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: schema, ForUser: "Schema retrieved"}, nil
}

func (t *VectorCollectionsTool) getDefaultDB(args map[string]interface{}) string {
	if db, ok := args["database"].(string); ok && db != "" {
		return db
	}
	return t.registry.Default()
}
