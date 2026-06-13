package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

func executeRootCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	cmd := NewRootCommand()
	cmd.SetArgs(args)
	var execErr error
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW

	stdoutDone := make(chan string, 1)
	stderrDone := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stdoutR)
		stdoutDone <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stderrR)
		stderrDone <- buf.String()
	}()

	execErr = cmd.Execute()
	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = originalStdout
	os.Stderr = originalStderr

	stdout := <-stdoutDone
	stderr := <-stderrDone
	return stdout, stderr, execErr
}

func newTempConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.json")
}

func writeConfigFile(t *testing.T, path string, cfg *config.Config) {
	t.Helper()

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func newTestConfig() *config.Config {
	return &config.Config{
		Agents: config.AgentConfig{Defaults: config.AgentDefaults{ModelName: "mock", MaxTokens: 256, SystemPrompt: "test"}},
		ModelList: []config.ModelEntry{
			{ModelName: "mock", Model: "mock/echo", APIKey: ""},
			{ModelName: "gpt4", Model: "openai/gpt-4o", APIKey: "sk-openai", APIBase: "https://api.openai.example/v1"},
			{ModelName: "claude", Model: "anthropic/claude-sonnet-4", APIKey: "sk-anthropic", APIBase: "https://api.anthropic.example"},
			{ModelName: "deepseek", Model: "deepseek/deepseek-chat", APIKey: "sk-deepseek"},
		},
	}
}

func newTestSession(id string, updatedAt time.Time, contents ...string) *session.Session {
	s := session.NewSession(id)
	s.CreatedAt = updatedAt.Add(-time.Hour)
	s.UpdatedAt = updatedAt
	for _, content := range contents {
		s.AddMessage(providers.Message{Role: providers.RoleUser, Content: content})
	}
	s.UpdatedAt = updatedAt
	return s
}

func TestVersionCommand(t *testing.T) {
	stdout, _, err := executeRootCommand(t, "version")
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if !strings.Contains(stdout, "golem version") {
		t.Fatalf("expected version output, got %q", stdout)
	}
}

func TestUnknownCommand(t *testing.T) {
	_, stderr, err := executeRootCommand(t, "definitely-not-a-command")
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("expected unknown command in stderr, got %q", stderr)
	}
}

func TestListModelNames(t *testing.T) {
	got := listModelNames(newTestConfig())
	want := "mock, gpt4, claude, deepseek"
	if got != want {
		t.Fatalf("listModelNames() = %q, want %q", got, want)
	}
}

func TestBuildToolRegistry(t *testing.T) {
	registry := buildToolRegistry(t.TempDir())
	defs := registry.ListDefinitions()
	if len(defs) != 5 {
		t.Fatalf("expected 5 tool definitions, got %d", len(defs))
	}
	gotNames := []string{defs[0].Name, defs[1].Name, defs[2].Name, defs[3].Name, defs[4].Name}
	wantNames := []string{"exec", "file_list", "file_read", "file_write", "web_search"}
	for i := range wantNames {
		if gotNames[i] != wantNames[i] {
			t.Fatalf("tool[%d] = %q, want %q", i, gotNames[i], wantNames[i])
		}
	}
}

func TestRegisterProviders(t *testing.T) {
	factory := registerProviders(newTestConfig())

	for _, vendor := range []string{"mock", "openai", "anthropic", "deepseek"} {
		if _, err := factory.GetProvider(vendor); err != nil {
			t.Fatalf("expected provider %q to be registered: %v", vendor, err)
		}
	}

	provider, modelID, err := factory.GetProviderForModel("openai/gpt-4o")
	if err != nil {
		t.Fatalf("GetProviderForModel failed: %v", err)
	}
	if modelID != "gpt-4o" {
		t.Fatalf("modelID = %q, want gpt-4o", modelID)
	}
	if provider.Name() == "" {
		t.Fatal("expected provider name to be set")
	}
}

func TestLoadConfigReturnsDefaultWhenMissing(t *testing.T) {
	cmd := NewRootCommand()
	configPath := newTempConfigPath(t)
	if err := cmd.PersistentFlags().Set("config", configPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := loadConfig(cmd)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if cfg.Agents.Defaults.ModelName == "" {
		t.Fatal("expected default config to have model name")
	}
}

func TestLoadConfigReadsCustomFile(t *testing.T) {
	cmd := NewRootCommand()
	configPath := newTempConfigPath(t)
	writeConfigFile(t, configPath, newTestConfig())
	if err := cmd.PersistentFlags().Set("config", configPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := loadConfig(cmd)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if cfg.Agents.Defaults.ModelName != "mock" {
		t.Fatalf("default model = %q, want mock", cfg.Agents.Defaults.ModelName)
	}
}

func TestResolveSessionID(t *testing.T) {
	store := session.NewMemoryStore()
	older := newTestSession("older", time.Now().Add(-2*time.Hour), "one")
	latest := newTestSession("latest", time.Now().Add(-time.Hour), "two")
	if err := store.Save(older); err != nil {
		t.Fatalf("save older session: %v", err)
	}
	if err := store.Save(latest); err != nil {
		t.Fatalf("save latest session: %v", err)
	}

	resolved, err := resolveSessionID(store, "older")
	if err != nil {
		t.Fatalf("resolve explicit session: %v", err)
	}
	if resolved != "older" {
		t.Fatalf("resolved explicit = %q, want older", resolved)
	}

	resolved, err = resolveSessionID(store, "last")
	if err != nil {
		t.Fatalf("resolve last session: %v", err)
	}
	if resolved != "latest" {
		t.Fatalf("resolved last = %q, want latest", resolved)
	}

	if _, err := resolveSessionID(store, "missing"); err == nil {
		t.Fatal("expected missing session error")
	}
}

func TestOpenAgentSessionStoreCreatesDatabase(t *testing.T) {
	cmd := NewRootCommand()
	configPath := newTempConfigPath(t)
	if err := cmd.PersistentFlags().Set("config", configPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	store, err := openAgentSessionStore(cmd)
	if err != nil {
		t.Fatalf("openAgentSessionStore failed: %v", err)
	}
	defer store.Close()
	saved := newTestSession("persisted", time.Now(), "hello")
	if err := store.Save(saved); err != nil {
		t.Fatalf("save session: %v", err)
	}
	loaded, ok := store.Get("persisted")
	if !ok {
		t.Fatal("expected saved session to be retrievable")
	}
	if loaded.ID != saved.ID {
		t.Fatalf("loaded session ID = %q, want %q", loaded.ID, saved.ID)
	}
}

func TestChoosePreset(t *testing.T) {
	t.Run("valid choice", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("2\n"))
		preset, err := choosePreset(reader)
		if err != nil {
			t.Fatalf("choosePreset failed: %v", err)
		}
		if preset.vendor != "openai" {
			t.Errorf("expected vendor 'openai', got %q", preset.vendor)
		}
	})

	t.Run("second choice", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("3\n"))
		preset, err := choosePreset(reader)
		if err != nil {
			t.Fatalf("choosePreset failed: %v", err)
		}
		if preset.vendor != "anthropic" {
			t.Errorf("expected vendor 'anthropic', got %q", preset.vendor)
		}
	})

	t.Run("invalid then valid", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("0\n99\n4\n"))
		preset, err := choosePreset(reader)
		if err != nil {
			t.Fatalf("choosePreset failed: %v", err)
		}
		if preset.vendor != "deepseek" {
			t.Errorf("expected vendor 'deepseek', got %q", preset.vendor)
		}
	})
}

func TestReadLine(t *testing.T) {
	t.Run("simple line", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("hello\n"))
		line, err := readLine(reader)
		if err != nil {
			t.Fatalf("readLine failed: %v", err)
		}
		if line != "hello" {
			t.Errorf("expected 'hello', got %q", line)
		}
	})

	t.Run("line with spaces", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("hello world\n"))
		line, err := readLine(reader)
		if err != nil {
			t.Fatalf("readLine failed: %v", err)
		}
		if line != "hello world" {
			t.Errorf("expected 'hello world', got %q", line)
		}
	})

	t.Run("empty line", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("\n"))
		line, err := readLine(reader)
		if err != nil {
			t.Fatalf("readLine failed: %v", err)
		}
		if line != "" {
			t.Errorf("expected empty string, got %q", line)
		}
	})
}

func TestProviderPresets(t *testing.T) {
	if len(providerPresets) != 8 {
		t.Errorf("expected 9 provider presets, got %d", len(providerPresets))
	}

	for i, preset := range providerPresets {
		if preset.label == "" {
			t.Errorf("preset %d has empty label", i)
		}
		if preset.vendor == "" {
			t.Errorf("preset %d has empty vendor", i)
		}
		if preset.model == "" {
			t.Errorf("preset %d has empty model", i)
		}
	}
}

func TestEnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.json")

	if err := ensureConfigDir(configPath); err != nil {
		t.Fatalf("ensureConfigDir failed: %v", err)
	}

	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("expected directory %q to exist", dir)
	}
}
