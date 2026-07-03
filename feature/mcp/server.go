package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/strings77wzq/golem/core/tools"
)

// Server is an MCP server that exposes tools via stdio JSON-RPC.
type Server struct {
	stdin       io.Reader
	stdout      io.Writer
	scanner     *bufio.Scanner
	registry    *tools.Registry
	mu          sync.Mutex
	initialized bool
}

// NewServer creates a new MCP server with stdio transport.
func NewServer(stdin io.Reader, stdout io.Writer, registry *tools.Registry) *Server {
	return &Server{
		stdin:    stdin,
		stdout:   stdout,
		scanner:  bufio.NewScanner(stdin),
		registry: registry,
	}
}

// Start runs the server main loop until stdin is closed or context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !s.scanner.Scan() {
			if err := s.scanner.Err(); err != nil {
				return fmt.Errorf("scanner error: %w", err)
			}
			return nil // EOF
		}

		line := s.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req map[string]interface{}
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(-1, -32700, "Parse error")
			continue
		}

		id, _ := req["id"].(float64)
		method, _ := req["method"].(string)

		s.handleRequest(context.Background(), int(id), method, req)
	}
}

// handleMessage processes a single JSON-RPC message (used in tests).
func (s *Server) handleMessage(ctx context.Context) error {
	if !s.scanner.Scan() {
		return io.EOF
	}

	line := s.scanner.Bytes()
	if len(line) == 0 {
		return nil
	}

	var req map[string]interface{}
	if err := json.Unmarshal(line, &req); err != nil {
		s.sendError(-1, -32700, "Parse error")
		return nil
	}

	id, _ := req["id"].(float64)
	method, _ := req["method"].(string)

	s.handleRequest(ctx, int(id), method, req)
	return nil
}

func (s *Server) handleRequest(ctx context.Context, id int, method string, req map[string]interface{}) {
	switch method {
	case "initialize":
		s.handleInitialize(id)
	case "notifications/initialized":
		// No response needed for notifications
	case "tools/list":
		s.handleToolsList(id)
	case "tools/call":
		params, _ := req["params"].(map[string]interface{})
		s.handleToolsCall(ctx, id, params)
	default:
		s.sendError(id, -32601, fmt.Sprintf("Method not found: %s", method))
	}
}

func (s *Server) handleInitialize(id int) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "golem-mcp",
			"version": "0.7.0",
		},
		"capabilities": map[string]interface{}{
			"tools": map[string]bool{"listChanged": false},
		},
		"instructions": "Golem MCP Server provides database query, schema introspection, and analysis tools.",
	}
	s.sendResult(id, result)
	s.initialized = true
}

func (s *Server) handleToolsList(id int) {
	var mcpTools []map[string]interface{}

	for _, t := range s.registry.ListTools() {
		schema := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		}

		// Build schema from tool parameters
		props := map[string]interface{}{}
		var required []string
		for _, p := range t.Parameters() {
			prop := map[string]interface{}{
				"type":        p.Type,
				"description": p.Description,
			}
			props[p.Name] = prop
			if p.Required {
				required = append(required, p.Name)
			}
		}
		schema["properties"] = props
		schema["required"] = required

		mcpTools = append(mcpTools, map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"inputSchema": schema,
		})
	}

	s.sendResult(id, map[string]interface{}{
		"tools": mcpTools,
	})
}

func (s *Server) handleToolsCall(ctx context.Context, id int, params map[string]interface{}) {
	name, _ := params["name"].(string)
	args, _ := params["arguments"].(map[string]interface{})

	if name == "" {
		s.sendError(id, -32602, "Missing tool name")
		return
	}

	tool, found := s.registry.Get(name)
	if !found {
		s.sendError(id, -32602, fmt.Sprintf("Tool not found: %s", name))
		return
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		s.sendError(id, -32000, fmt.Sprintf("Tool execution failed: %v", err))
		return
	}

	s.sendResult(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": result.ForLLM},
		},
	})
}

func (s *Server) sendResult(id int, result interface{}) {
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	s.send(resp)
}

func (s *Server) sendError(id int, code int, message string) {
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	s.send(resp)
}

func (s *Server) send(resp interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "MCP server marshal error: %v\n", err)
		return
	}
	data = append(data, '\n')
	s.stdout.Write(data) //nolint:errcheck
}
