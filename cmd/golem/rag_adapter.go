// RAG Adapter - wires RAG feature into the main agent
// Provides document indexing and retrieval capabilities

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/feature/rag"
)

type RagConfig struct {
	IndexDir string `json:"index_dir"`
	TopK     int    `json:"top_k"`
	APIKey   string `json:"api_key"`
	APIBase  string `json:"api_base"`
	Model    string `json:"model"`
}

// LoadRAGTools loads documents from the specified directory and creates a RAG tool
func LoadRAGTools(ctx context.Context, cfg RagConfig) (*tools.Registry, error) {
	registry := tools.NewRegistry()

	if cfg.IndexDir == "" {
		return registry, nil
	}

	// Check if directory exists
	info, err := os.Stat(cfg.IndexDir)
	if err != nil {
		if os.IsNotExist(err) {
			return registry, nil // No RAG, just return empty registry
		}
		return nil, fmt.Errorf("checking RAG index dir: %w", err)
	}
	if !info.IsDir() {
		return registry, nil // Not a directory, skip
	}

	// Create embedder
	embedder := rag.NewOpenAIEmbedder(cfg.APIKey)
	if cfg.APIBase != "" {
		embedder = rag.NewOpenAIEmbedder(cfg.APIKey, rag.WithAPIBase(cfg.APIBase))
	}
	if cfg.Model != "" {
		embedder = rag.NewOpenAIEmbedder(cfg.APIKey, rag.WithModel(cfg.Model))
	}

	// Create vector store
	store := rag.NewMemoryVectorStore()

	// Create retriever
	topK := cfg.TopK
	if topK <= 0 {
		topK = 3
	}
	retriever := rag.NewRetriever(embedder, store, topK)

	// Load documents from directory
	docs, err := loadDocumentsFromDir(cfg.IndexDir)
	if err != nil {
		return nil, fmt.Errorf("loading documents: %w", err)
	}

	if len(docs) > 0 {
		if err := retriever.AddDocuments(ctx, docs); err != nil {
			return nil, fmt.Errorf("indexing documents: %w", err)
		}
		fmt.Printf("RAG: indexed %d documents from %s\n", len(docs), cfg.IndexDir)
	}

	// Create and register the RAG tool
	ragTool := &ragTool{
		retriever: retriever,
		indexDir:  cfg.IndexDir,
		docCount:  len(docs),
	}
	registry.Register(ragTool)

	return registry, nil
}

// loadDocumentsFromDir loads all text files from a directory
func loadDocumentsFromDir(dir string) ([]rag.RawDocument, error) {
	var docs []rag.RawDocument

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip non-text files
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".txt" && ext != ".md" && ext != ".json" && ext != ".html" && ext != ".xml" && ext != ".yaml" && ext != ".yml" && ext != ".go" && ext != ".py" && ext != ".js" && ext != ".ts" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", path, err)
			continue
		}

		docs = append(docs, rag.RawDocument{
			ID:       entry.Name(),
			Content:  string(content),
			Metadata: map[string]string{"source": path},
		})
	}

	return docs, nil
}

// ragTool implements tools.Tool for RAG retrieval
type ragTool struct {
	retriever *rag.Retriever
	indexDir  string
	docCount  int
}

func (r *ragTool) Name() string {
	return "rag_retrieve"
}

func (r *ragTool) Description() string {
	return fmt.Sprintf("Retrieve relevant information from the indexed documents (%d documents indexed from %s). Use this when you need to answer questions based on the provided document knowledge base.", r.docCount, r.indexDir)
}

func (r *ragTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query to find relevant information from the document index",
			Required:    true,
		},
	}
}

func (r *ragTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &tools.ToolResult{
			ForLLM:  "Error: 'query' parameter is required",
			ForUser: "Please provide a query string",
			IsError: true,
		}, nil
	}

	results, err := r.retriever.Query(ctx, query)
	if err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Error querying the document index: %v", err),
			ForUser: fmt.Sprintf("Error: %v", err),
			IsError: true,
		}, nil
	}

	if len(results) == 0 {
		return &tools.ToolResult{
			ForLLM:  "No relevant information found in the document index for the query.",
			ForUser: "No results found for your query.",
			IsError: false,
		}, nil
	}

	// Format results for the LLM
	var sb strings.Builder
	sb.WriteString("Relevant documents found:\n\n")

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("--- Document %d (score: %.3f) ---\n", i+1, result.Score))
		sb.WriteString(fmt.Sprintf("Source: %s\n", result.Document.Metadata["source"]))
		sb.WriteString(fmt.Sprintf("Content:\n%s\n\n", result.Document.Content))
	}

	return &tools.ToolResult{
		ForLLM:  sb.String(),
		ForUser: fmt.Sprintf("Found %d relevant documents", len(results)),
		IsError: false,
	}, nil
}

// ParseRagConfig parses RAG configuration from JSON string
func ParseRagConfig(jsonStr string) (RagConfig, error) {
	cfg := RagConfig{
		TopK: 3,
	}

	if jsonStr == "" {
		return cfg, nil
	}

	// Try to parse as JSON
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		// If not JSON, treat as directory path
		cfg.IndexDir = jsonStr
	}

	return cfg, nil
}
