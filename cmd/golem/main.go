// Golem — Go-native AI agent runtime.
// This file is the composition root: it defines CLI commands and delegates
// all wiring and execution to adapter.go and run.go.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/strings77wzq/golem/core/agent"
	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/security"
	"github.com/strings77wzq/golem/core/session"
	toolexec "github.com/strings77wzq/golem/core/tools/exec"
	featureconfig "github.com/strings77wzq/golem/feature/config"
	"github.com/strings77wzq/golem/feature/health"
	"github.com/strings77wzq/golem/foundation/logger"
	"github.com/strings77wzq/golem/foundation/term"
	"github.com/strings77wzq/golem/internal/channels/telegram"
	"github.com/strings77wzq/golem/internal/gateway"
	"github.com/strings77wzq/golem/internal/wiring"
)

var (
	version = "0.10.0"
	commit  = "unknown"
	date    = "unknown"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "golem",
		Short:   "Golem — Go-native AI agent runtime",
		Long:    "A pure-Go AI agent runtime — build, embed, deploy AI agents with a single binary.",
		Example: "  golem agent\n  golem gateway\n  golem version",
	}

	cmd.PersistentFlags().StringP("config", "c", "", "config file path (default: ~/.golem/config.json)")

	cmd.AddCommand(
		newVersionCommand(),
		newAgentCommand(),
		newGatewayCommand(),
		newConfigCommand(),
		newStatusCommand(),
		newSessionCommand(),
		newInitCommand(),
		newMCPServerCommand(),
		newDemoCommand(),
		newDebugCommand(nil), // lazy init: creates registry on first use
	)

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Printf("golem version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("date: %s\n", date)
			return nil
		},
	}
}

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Start the AI agent",
		Long:  "Start the Golem AI agent process",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgent(cmd)
		},
	}
	cmd.Flags().StringP("message", "m", "", "Send a single message and exit")
	cmd.Flags().StringP("model", "M", "", "Model to use (overrides config default)")
	cmd.Flags().StringP("continue", "C", "", `Resume a session ("last" or session-id)`)
	cmd.Flags().Bool("no-tui", false, "Use plain interactive mode instead of TUI")
	cmd.Flags().String("skills-dir", "", "Directory containing skills (with skill.json files)")
	cmd.Flags().String("skills", "", "Comma-separated skill names to enable (e.g., 'summarize,code-review')")
	cmd.Flags().String("rag", "", "RAG configuration: directory path or JSON config for document index")
	cmd.Flags().String("mcp", "", "MCP servers configuration: JSON array of server configs")
	cmd.Flags().String("memory", "", "Memory file path or JSON config for long-term memory")
	cmd.Flags().String("telegram", "", "Telegram bot token or JSON config for Telegram channel")
	cmd.Flags().String("db", "", "Database path (SQLite) for SQL tools")
	cmd.Flags().Bool("infra", false, "Enable infrastructure tools (kubectl, docker, helm)")
	cmd.Flags().String("routing", "", "Routing config: JSON with model-to-provider fallback chains")
	cmd.Flags().String("health", "", "Health check config: 'true' or JSON with interval")
	cmd.Flags().String("sandbox", "", "Sandbox config: JSON with allowed/denied paths and commands")
	cmd.Flags().Bool("json-events", false, "Emit structured JSON events to stderr for each tool call (E2E observability)")
	return cmd
}

// runAgent is the main agent command handler — thin orchestrator.
func runAgent(cmd *cobra.Command) error {
	message, _ := cmd.Flags().GetString("message")
	modelFlag, _ := cmd.Flags().GetString("model")
	continueFlag, _ := cmd.Flags().GetString("continue")
	noTUI, _ := cmd.Flags().GetBool("no-tui")
	skillsDir, _ := cmd.Flags().GetString("skills-dir")
	skillsFilter, _ := cmd.Flags().GetString("skills")
	dbFlag, _ := cmd.Flags().GetString("db")
	infraFlag, _ := cmd.Flags().GetBool("infra")
	jsonEvents, _ := cmd.Flags().GetBool("json-events")

	// Cancellable context for feature-tool lifetimes: external MCP server
	// subprocesses are closed when golem shuts down (LoadMCPTools registers
	// ctx-based cleanup). cobra's cmd.Context() is context.Background() and
	// cannot trigger that cleanup.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	// If db flag is empty, check if YAML config provided a database path
	if dbFlag == "" {
		configPath, _ := getConfigPath(cmd)
		if featureconfig.IsYAMLConfig(configPath) {
			if agentCfg, loadErr := featureconfig.LoadYAML(configPath); loadErr == nil && agentCfg.Database != nil {
				dbFlag = agentCfg.Database.Path
			}
		}
	}

	log := logger.New(logger.DefaultOptions())

	if modelFlag != "" {
		if _, findErr := cfg.FindModel(modelFlag); findErr != nil {
			return fmt.Errorf("model %q not found in config; available: %s", modelFlag, listModelNames(cfg))
		}
		cfg.Agents.Defaults.ModelName = modelFlag
	}

	// Build core registries
	skillRegistry := LoadSkills(log, skillsDir, skillsFilter)
	b := bus.New()
	b.SetOnDropped(func(topic string, count int) {
		log.Warn("bus message dropped", "topic", topic, "count", count)
	})
	workspace, _ := os.Getwd()

	// Optional: exec sandbox
	var execOpts []toolexec.Option
	if sandboxFlag := mustGetString(cmd, "sandbox"); sandboxFlag != "" {
		sandboxCfg, sandboxErr := ParseSandboxConfig(sandboxFlag)
		if sandboxErr != nil {
			return fmt.Errorf("parsing sandbox config: %w", sandboxErr)
		}
		validator := NewSandboxValidator(*sandboxCfg)
		execOpts = append(execOpts, toolexec.WithValidator(validator))
		log.Info("exec sandbox enabled")
	}

	registry := wiring.BuildToolRegistry(workspace, execOpts...)

	// Load database tools. Audit every allowed and denied operation as
	// structured log output (component=audit, trace_id when available).
	auditFn := func(entry security.AuditEntry) {
		log.WithComponent(logger.ComponentAgent).Info("db audit",
			"operation", entry.Operation,
			"database", entry.Database,
			"table", entry.Table,
			"sql", entry.SQL,
			"status", entry.Status,
			"affected_rows", entry.AffectedRows,
			"rollback_sql", entry.RollbackSQL,
			"trace_id", entry.TraceID,
		)
	}
	if dbFlag != "" {
		_, dbTools := wiring.BuildDBTools(dbFlag, auditFn, nil)
		if dbTools != nil {
			for _, t := range dbTools.ListTools() {
				if err := registry.Register(t); err != nil {
					log.Warn("failed to register DB tool", "tool", t.Name(), "error", err)
				}
			}
			log.Info("loaded database tools", "db", dbFlag, "tools", dbTools.Count())
		}
		// When database tools are available, remove web_search to prevent
		// the LLM from using external data instead of database data
		if registry.Has("web_search") {
			registry.Remove("web_search")
			log.Info("removed web_search tool (database mode)")
		}
	}

	// Load infrastructure tools
	if infraFlag {
		wiring.RegisterInfraTools(registry)
		log.Info("loaded infrastructure tools", "tools", 3)
	}

	// Load feature tools
	if err := loadRAGTools(cmd.Context(), cfg, mustGetString(cmd, "rag"), registry); err != nil {
		return err
	}
	if err := loadMCPTools(ctx, mustGetString(cmd, "mcp"), registry, log); err != nil {
		return err
	}
	if err := loadMemoryTools(cmd.Context(), mustGetString(cmd, "memory"), registry, log); err != nil {
		return err
	}

	factory := wiring.RegisterProviders(cfg)

	// Optional: routing-aware provider resolution
	var routerOpts []agent.Option
	if routingFlag := mustGetString(cmd, "routing"); routingFlag != "" {
		router, routerErr := LoadRoutingFactory(routingFlag, factory)
		if routerErr != nil {
			return fmt.Errorf("loading routing config: %w", routerErr)
		}
		if router != nil {
			routerOpts = append(routerOpts, agent.WithRouter(router))
			log.Info("routing enabled")
		}
	}

	// Optional: health checks
	var healthManager *health.Manager
	if healthFlag := mustGetString(cmd, "health"); healthFlag != "" {
		mgr, healthErr := LoadHealthManager(healthFlag, log)
		if healthErr != nil {
			return fmt.Errorf("loading health config: %w", healthErr)
		}
		if mgr != nil {
			healthManager = mgr
			log.Info("health checks enabled")
		}
	}

	store, err := openAgentSessionStore(cmd)
	if err != nil {
		log.Warn("SQLite session store unavailable, using in-memory (sessions will not persist across restarts)", "err", err)
		store = nil
	}
	var sessionStore session.SessionStore
	if store != nil {
		defer store.Close() //nolint:errcheck
		sessionStore = store
		log.Info("session store ready", "type", "sqlite")
	} else {
		sessionStore = session.NewMemoryStore()
		log.Info("session store ready", "type", "in-memory", "warning", "sessions will not persist across restarts")
	}

	systemPrompt := BuildSystemPrompt(cfg.Agents.Defaults.SystemPrompt, skillRegistry)

	// Add data integrity instruction when database tools are available
	if dbFlag != "" {
		systemPrompt += "\n\n## CRITICAL: Data Integrity Rules\n" +
			"- You are connected to a database. ALL data answers MUST come from the database.\n" +
			"- IGNORE your training data for any data that exists in the database.\n" +
			"- When the database returns specific numbers, use those EXACT numbers.\n" +
			"- NEVER substitute database values with values from your training data.\n" +
			"- If you need data, query the database first. Do NOT use your training knowledge.\n" +
			"- The database is the single source of truth. Your training data may be outdated or wrong."
	}

	// Create LLM-driven compactor for session compression
	compactor := agent.NewCompactor(
		providers.NewMockProvider("compact"),
		cfg.Agents.Defaults.ModelName,
	)
	// Try to get the actual provider for compaction
	if compactorProvider, _, err := factory.GetProviderForModel(cfg.Agents.Defaults.ModelName); err == nil {
		compactor = agent.NewCompactor(compactorProvider, cfg.Agents.Defaults.ModelName)
	}

	ag := agent.New(b, registry, factory, sessionStore, log, cfg,
		agent.WithSystemPrompt(systemPrompt),
		agent.WithCompactor(compactor),
	)
	// Apply routing options
	for _, opt := range routerOpts {
		opt(ag)
	}

	// Start health manager if enabled
	if healthManager != nil {
		// Register all providers that implement HealthChecker
		for _, vendor := range factory.ListVendors() {
			if p, ok := factory.GetProviderByVendor(vendor); ok {
				if hc, ok := p.(providers.HealthChecker); ok {
					healthManager.Register(hc)
					log.Info("registered provider for health checks", "vendor", vendor)
				}
			}
		}
		healthManager.Start(cmd.Context())
		defer healthManager.Stop()
	}

	// Telegram adapter
	telegramFlag, _ := cmd.Flags().GetString("telegram")
	var telegramAdapter *telegram.Adapter
	if telegramFlag != "" {
		tgCfg, tgErr := ParseTelegramConfig(telegramFlag)
		if tgErr != nil {
			return fmt.Errorf("parsing telegram config: %w", tgErr)
		}
		tgCtx, tgCancel := context.WithCancel(context.Background())
		defer tgCancel()
		telegramAdapter, tgErr = StartTelegramAdapter(tgCtx, tgCfg, b, log)
		if tgErr != nil {
			return fmt.Errorf("starting telegram adapter: %w", tgErr)
		}
		telegramAdapter.Start(tgCtx) //nolint:errcheck
		defer telegramAdapter.Stop()
	}

	// Session resume
	var sessionID string
	if continueFlag != "" {
		sessionID, err = resolveSessionID(sessionStore, continueFlag)
		if err != nil {
			return err
		}
		log.Info("resuming session", "id", sessionID)
	}

	// Stdin merge
	if !term.IsInputTTY() {
		stdinContent, readErr := term.ReadStdin()
		if readErr != nil {
			return fmt.Errorf("reading stdin: %w", readErr)
		}
		stdinContent = strings.TrimSpace(stdinContent)
		if stdinContent != "" {
			if message != "" {
				message = message + "\n\n" + stdinContent
			} else {
				message = stdinContent
			}
		}
	}

	// Dispatch to run mode
	if message != "" {
		if jsonEvents {
			return runAgentOneShotWithEvents(ag, b, message, sessionID)
		}
		return runAgentOneShot(ag, b, message, sessionID)
	}

	if !term.IsInputTTY() {
		return fmt.Errorf("no input: use -m flag or pipe content via stdin")
	}

	if !noTUI {
		return runAgentTUI(ag, sessionID)
	}

	return runAgentInteractive(ag, b, sessionID)
}

func newGatewayCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "Start the HTTP gateway server",
		Long:  "Start the HTTP gateway server for agent communication",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			b := bus.New()
			workspace, _ := os.Getwd()
			registry := wiring.BuildToolRegistry(workspace)
			factory := wiring.RegisterProviders(cfg)
			log := logger.New(logger.DefaultOptions())
			b.SetOnDropped(func(topic string, count int) {
				log.Warn("bus message dropped", "topic", topic, "count", count)
			})

			store, err := openAgentSessionStore(cmd)
			if err != nil {
				log.Warn("SQLite session store unavailable, using in-memory", "err", err)
				store = nil
			}
			var sessionStore session.SessionStore
			if store != nil {
				defer store.Close() //nolint:errcheck
				sessionStore = store
			} else {
				sessionStore = session.NewMemoryStore()
			}

			ag := agent.New(b, registry, factory, sessionStore, log, cfg)

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			go ag.Start(ctx)

			serverCfg := gateway.DefaultServerConfig()
			serverCfg.Version = version
			secCfg := gateway.DefaultSecurityConfig()

			if cfg.Gateway.Addr != "" {
				serverCfg.Addr = cfg.Gateway.Addr
			}

			authToken := os.Getenv("GOLEM_AUTH_TOKEN")
			if authToken != "" {
				secCfg.EnableAuth = true
				secCfg.AuthToken = authToken
			} else if cfg.Gateway.AuthToken != "" {
				secCfg.EnableAuth = true
				secCfg.AuthToken = cfg.Gateway.AuthToken
			}

			if cfg.Gateway.RateLimitRPS > 0 {
				secCfg.EnableRateLimit = true
				secCfg.RateLimitRPS = float64(cfg.Gateway.RateLimitRPS)
			}
			if cfg.Gateway.RateLimitBurst > 0 {
				secCfg.RateLimitBurst = cfg.Gateway.RateLimitBurst
			}

			if len(cfg.Gateway.AllowedOrigins) > 0 {
				secCfg.CORS = gateway.CORSConfig{
					AllowedOrigins: cfg.Gateway.AllowedOrigins,
					AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type", "Authorization", "X-Request-ID"},
				}
			}

			server := gateway.NewServerWithSecurity(serverCfg, secCfg, ag, log)
			server.SetSessionStore(sessionStore)

			// Wire model list from config for /v1/models endpoint
			if len(cfg.ModelList) > 0 {
				models := make([]gateway.ModelListItem, 0, len(cfg.ModelList))
				for _, m := range cfg.ModelList {
					vendor := ""
					parts := strings.SplitN(m.Model, "/", 2)
					if len(parts) == 2 {
						vendor = parts[0]
					}
					models = append(models, gateway.ModelListItem{
						ID:     m.Model,
						Vendor: vendor,
					})
				}
				server.SetModelList(models)
			}

			// Optional: health checks for gateway
			if healthFlag, _ := cmd.Flags().GetString("health"); healthFlag != "" {
				mgr, healthErr := LoadHealthManager(healthFlag, log)
				if healthErr != nil {
					return fmt.Errorf("loading health config: %w", healthErr)
				}
				if mgr != nil {
					for _, vendor := range factory.ListVendors() {
						if p, ok := factory.GetProviderByVendor(vendor); ok {
							if hc, ok := p.(providers.HealthChecker); ok {
								mgr.Register(hc)
							}
						}
					}
					mgr.Start(cmd.Context())
					defer mgr.Stop()
					server.SetHealthChecker(mgr)
					log.Info("health checks enabled for gateway")
				}
			}

			if cfg.Telegram.Token != "" && cfg.Telegram.Mode == "webhook" {
				tgCfg := cfg.Telegram
				tgCtx, tgCancel := context.WithCancel(context.Background())
				defer tgCancel()
				tgAdapter, tgErr := StartTelegramAdapter(tgCtx, tgCfg, b, log)
				if tgErr != nil {
					return fmt.Errorf("starting telegram adapter: %w", tgErr)
				}
				tgAdapter.Start(tgCtx) //nolint:errcheck
				defer tgAdapter.Stop()
				server.MountHandler("/telegram/webhook", tgAdapter.WebhookHandler(tgCfg.WebhookSecret))
				log.Info("telegram webhook mounted", "path", "/telegram/webhook")
			}

			log.Info("starting gateway server",
				"addr", serverCfg.Addr,
				"auth_enabled", secCfg.EnableAuth,
				"rate_limit_enabled", secCfg.EnableRateLimit,
			)
			if !secCfg.EnableAuth {
				log.Warn("⚠️  Gateway running WITHOUT authentication — any client can execute SQL queries")
				log.Warn("   Set GOLEM_AUTH_TOKEN environment variable or configure auth in config file")
			}
			return server.Start()
		},
	}

	cmd.Flags().String("auth-token", "", "API token for authentication (can also set GOLEM_AUTH_TOKEN env)")
	cmd.Flags().Int("rate-limit", 100, "Rate limit requests per second")
	cmd.Flags().String("health", "", "Health check config: 'true' or JSON with interval")

	return cmd
}

// --- helpers ---

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, err := getConfigPath(cmd)
	if err != nil {
		return nil, err
	}

	// YAML config path — use feature/config loader
	if featureconfig.IsYAMLConfig(configPath) {
		agentCfg, loadErr := featureconfig.LoadYAML(configPath)
		if loadErr != nil {
			if errors.Is(loadErr, os.ErrNotExist) {
				return config.DefaultConfig(), nil
			}
			return nil, loadErr
		}
		return agentCfg.ToCoreConfig(), nil
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config.DefaultConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

func openAgentSessionStore(cmd *cobra.Command) (*session.SQLiteAdapter, error) {
	configPath, err := getConfigPath(cmd)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(configPath)
	if err := ensureConfigDir(configPath); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, "sessions.db")
	return session.NewSQLiteAdapter(dbPath)
}

func resolveSessionID(store session.SessionStore, flag string) (string, error) {
	if flag != "last" {
		if _, ok := store.Get(flag); !ok {
			return "", fmt.Errorf("session %q not found", flag)
		}
		return flag, nil
	}
	sessions := store.List()
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions to resume")
	}
	latest := sessions[0]
	for _, s := range sessions[1:] {
		if s.UpdatedAt.After(latest.UpdatedAt) {
			latest = s
		}
	}
	return latest.ID, nil
}

func listModelNames(cfg *config.Config) string {
	names := make([]string, 0, len(cfg.ModelList))
	for _, entry := range cfg.ModelList {
		names = append(names, entry.ModelName)
	}
	return strings.Join(names, ", ")
}

func mustGetString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func main() {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
