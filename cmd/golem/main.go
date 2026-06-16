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
	"github.com/strings77wzq/golem/core/session"
	featureconfig "github.com/strings77wzq/golem/feature/config"
	"github.com/strings77wzq/golem/foundation/logger"
	"github.com/strings77wzq/golem/foundation/term"
	"github.com/strings77wzq/golem/internal/channels/telegram"
	"github.com/strings77wzq/golem/internal/gateway"
	"github.com/strings77wzq/golem/internal/wiring"
)

var (
	version = "dev"
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
	skillRegistry := wiring.LoadSkills(log, skillsDir, skillsFilter)
	b := bus.New()
	workspace, _ := os.Getwd()
	registry := wiring.BuildToolRegistry(workspace)

	// Load database tools
	if dbFlag != "" {
		_, dbTools := wiring.BuildDBTools(dbFlag)
		if dbTools != nil {
			for _, t := range dbTools.ListTools() {
				registry.Register(t)
			}
			log.Info("loaded database tools", "db", dbFlag, "tools", dbTools.Count())
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
	if err := loadMCPTools(cmd.Context(), mustGetString(cmd, "mcp"), registry, log); err != nil {
		return err
	}
	if err := loadMemoryTools(cmd.Context(), mustGetString(cmd, "memory"), registry, log); err != nil {
		return err
	}

	factory := wiring.RegisterProviders(cfg)

	store, err := openAgentSessionStore(cmd)
	if err != nil {
		log.Warn("SQLite session store unavailable, using in-memory", "err", err)
		store = nil
	}
	var sessionStore session.SessionStore
	if store != nil {
		defer store.Close()
		sessionStore = store
	} else {
		sessionStore = session.NewMemoryStore()
	}

	systemPrompt := wiring.BuildSystemPrompt(cfg.Agents.Defaults.SystemPrompt, skillRegistry)

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
		telegramAdapter.Start(tgCtx)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			b := bus.New()
			workspace, _ := os.Getwd()
			registry := wiring.BuildToolRegistry(workspace)
			factory := wiring.RegisterProviders(cfg)
			log := logger.New(logger.DefaultOptions())

			store, err := openAgentSessionStore(cmd)
			if err != nil {
				log.Warn("SQLite session store unavailable, using in-memory", "err", err)
				store = nil
			}
			var sessionStore session.SessionStore
			if store != nil {
				defer store.Close()
				sessionStore = store
			} else {
				sessionStore = session.NewMemoryStore()
			}

			ag := agent.New(b, registry, factory, sessionStore, log, cfg)

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			go ag.Start(ctx)

			serverCfg := gateway.DefaultServerConfig()
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

			if cfg.Telegram.Token != "" && cfg.Telegram.Mode == "webhook" {
				tgCfg := cfg.Telegram
				tgCtx, tgCancel := context.WithCancel(context.Background())
				defer tgCancel()
				tgAdapter, tgErr := StartTelegramAdapter(tgCtx, tgCfg, b, log)
				if tgErr != nil {
					return fmt.Errorf("starting telegram adapter: %w", tgErr)
				}
				tgAdapter.Start(tgCtx)
				defer tgAdapter.Stop()
				server.MountHandler("/telegram/webhook", tgAdapter.WebhookHandler(tgCfg.WebhookSecret))
				log.Info("telegram webhook mounted", "path", "/telegram/webhook")
			}

			log.Info("starting gateway server",
				"addr", serverCfg.Addr,
				"auth_enabled", secCfg.EnableAuth,
				"rate_limit_enabled", secCfg.EnableRateLimit,
			)
			return server.Start()
		},
	}

	cmd.Flags().String("auth-token", "", "API token for authentication (can also set GOLEM_AUTH_TOKEN env)")
	cmd.Flags().Int("rate-limit", 100, "Rate limit requests per second")

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
