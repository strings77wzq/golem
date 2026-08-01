package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strings77wzq/golem/core/tools"
)

// Limits for the external MCP server boundary (see security review H1):
// a hostile or broken third-party server must not be able to stall startup
// indefinitely, exhaust memory via unbounded tool lists, or stream
// unbounded content into the agent.
const (
	serverConnectTimeout = 10 * time.Second
	maxDiscoveredTools   = 100
	maxProxyOutputBytes  = 10 << 10 // 10 KiB, mirrors core/tools/exec
	maxToolNameLen       = 64
)

// ServerConfig describes an external MCP server to connect to.
type ServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type serverConnection struct {
	config  ServerConfig
	session *sdk.ClientSession
	tools   []MCPTool
}

// Manager connects to external MCP servers and proxies their tools into the
// golem registry under the mcp_<server>_<tool> naming scheme.
type Manager struct {
	mu      sync.RWMutex
	servers map[string]*serverConnection
}

func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*serverConnection),
	}
}

func (m *Manager) AddServer(cfg ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if cfg.Command == "" {
		return fmt.Errorf("server command is required")
	}

	if _, exists := m.servers[cfg.Name]; exists {
		return fmt.Errorf("server %s already exists", cfg.Name)
	}

	m.servers[cfg.Name] = &serverConnection{
		config: cfg,
	}

	return nil
}

// Start connects to all registered servers, runs tools/list on each, and
// caches the discovered tools. Connections are established lazily per server
// in parallel goroutines; a failure on one server is reported but does not
// block the others.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	serverList := make([]*serverConnection, 0, len(m.servers))
	for _, conn := range m.servers {
		serverList = append(serverList, conn)
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(serverList))

	for _, conn := range serverList {
		wg.Add(1)
		go func(conn *serverConnection) {
			defer wg.Done()

			// Per-server timeout so one stalled server cannot block startup
			// or the agent loop forever.
			connCtx, cancel := context.WithTimeout(ctx, serverConnectTimeout)
			defer cancel()

			env := make([]string, 0, len(conn.config.Env))
			for k, v := range conn.config.Env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}

			cmd := exec.Command(conn.config.Command, conn.config.Args...) // #nosec G204 -- configured server command
			cmd.Env = append(os.Environ(), env...)

			client := sdk.NewClient(&sdk.Implementation{Name: "golem", Version: "0.1.0"}, nil)
			session, err := client.Connect(connCtx, &sdk.CommandTransport{Command: cmd}, nil)
			if err != nil {
				errChan <- fmt.Errorf("failed to connect server %s: %w", conn.config.Name, err)
				return
			}

			toolsResult, err := session.ListTools(connCtx, &sdk.ListToolsParams{})
			if err != nil {
				session.Close() //nolint:errcheck
				errChan <- fmt.Errorf("failed to list tools for server %s: %w", conn.config.Name, err)
				return
			}

			toolsList := make([]MCPTool, 0, len(toolsResult.Tools))
			for _, t := range toolsResult.Tools {
				if len(toolsList) >= maxDiscoveredTools {
					break
				}
				schemaJSON, err := json.Marshal(t.InputSchema)
				if err != nil {
					schemaJSON = json.RawMessage(`{}`)
				}
				toolsList = append(toolsList, MCPTool{
					Name:        sanitizeToolName(t.Name),
					Description: t.Description,
					InputSchema: schemaJSON,
				})
			}

			m.mu.Lock()
			conn.session = session
			conn.tools = toolsList
			m.mu.Unlock()
		}(conn)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// Close any sessions that connected successfully before the failure
		// so their subprocesses are not leaked.
		m.mu.Lock()
		var sessions []*sdk.ClientSession
		for _, conn := range m.servers {
			if conn.session != nil {
				sessions = append(sessions, conn.session)
			}
		}
		m.mu.Unlock()
		for _, s := range sessions {
			s.Close() //nolint:errcheck
		}
		return fmt.Errorf("failed to start %d server(s): %v", len(errors), errors)
	}

	return nil
}

// sanitizeToolName restricts external tool names to a safe identifier
// charset and length, preventing registry key collisions and prompt
// pollution from hostile servers.
func sanitizeToolName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) > maxToolNameLen {
		s = s[:maxToolNameLen]
	}
	if s == "" {
		s = "unnamed"
	}
	return s
}

func (m *Manager) DiscoverTools(ctx context.Context) ([]MCPToolProxy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var proxies []MCPToolProxy
	for serverName, conn := range m.servers {
		if conn.session == nil {
			continue
		}

		for _, mcpTool := range conn.tools {
			proxy := MCPToolProxy{
				serverName: serverName,
				mcpTool:    mcpTool,
				manager:    m,
			}
			proxies = append(proxies, proxy)
		}
	}

	return proxies, nil
}

func (m *Manager) Close() error {
	// Collect sessions under the lock, close outside it: session.Close can
	// block up to the SDK terminate window (~10s) and must not hold m.mu.
	m.mu.Lock()
	sessions := make([]*sdk.ClientSession, 0, len(m.servers))
	for _, conn := range m.servers {
		if conn.session != nil {
			sessions = append(sessions, conn.session)
		}
	}
	m.mu.Unlock()

	var errs []error
	for _, s := range sessions {
		if err := s.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d connection(s): %v", len(errs), errs)
	}

	return nil
}

func (m *Manager) callTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (*sdk.CallToolResult, error) {
	m.mu.RLock()
	conn, exists := m.servers[serverName]
	var session *sdk.ClientSession
	if exists {
		session = conn.session
	}
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	if session == nil {
		return nil, fmt.Errorf("server %s not initialized", serverName)
	}

	return session.CallTool(ctx, &sdk.CallToolParams{Name: toolName, Arguments: args})
}

// MCPToolProxy adapts an external MCP tool to the golem tools.Tool interface.
type MCPToolProxy struct {
	serverName string
	mcpTool    MCPTool
	manager    *Manager
}

func (p MCPToolProxy) Name() string {
	return fmt.Sprintf("mcp_%s_%s", p.serverName, p.mcpTool.Name)
}

func (p MCPToolProxy) Description() string {
	return p.mcpTool.Description
}

func (p MCPToolProxy) Parameters() []tools.ToolParameter {
	var schema struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}

	if err := json.Unmarshal(p.mcpTool.InputSchema, &schema); err != nil {
		return nil
	}

	params := make([]tools.ToolParameter, 0, len(schema.Properties))
	requiredMap := make(map[string]bool)
	for _, req := range schema.Required {
		requiredMap[req] = true
	}

	for name, propData := range schema.Properties {
		var prop struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(propData, &prop); err != nil {
			continue
		}

		params = append(params, tools.ToolParameter{
			Name:        name,
			Type:        prop.Type,
			Description: prop.Description,
			Required:    requiredMap[name],
		})
	}

	return params
}

func (p MCPToolProxy) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	result, err := p.manager.callTool(ctx, p.serverName, p.mcpTool.Name, args)
	if err != nil {
		// Return only the error: the agent loop formats it for the LLM.
		// Returning both an error and an error-ToolResult would drop the
		// remote detail (loop_helpers takes the err branch).
		return nil, fmt.Errorf("calling MCP tool %s: %w", p.mcpTool.Name, err)
	}

	var output string
	for _, block := range result.Content {
		if tc, ok := block.(*sdk.TextContent); ok {
			output += tc.Text
		}
	}
	if len(output) > maxProxyOutputBytes {
		output = output[:maxProxyOutputBytes] + "\n...[truncated]"
	}

	return &tools.ToolResult{
		ForLLM:  output,
		ForUser: output,
		IsError: result.IsError,
	}, nil
}
