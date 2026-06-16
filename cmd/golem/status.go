package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/internal/wiring"
)

// FeatureStatus tracks which optional features are enabled.
type FeatureStatus struct {
	MCPEnabled    bool
	MCPServers    int
	RAGEnabled    bool
	MemoryEnabled bool
	SkillsEnabled bool
	SkillsCount   int
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		Long:  "Display version, configuration, tools, features, and service health information",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "Golem Status")
			fmt.Fprintln(out, "====================")
			fmt.Fprintln(out)

			fmt.Fprintf(out, "Version:    %s\n", version)
			fmt.Fprintf(out, "Commit:     %s\n", commit)
			fmt.Fprintf(out, "Build Date: %s\n\n", date)

			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			fmt.Fprintf(out, "Config:     %s\n", configPath)

			if _, err := os.Stat(configPath); err == nil {
				fmt.Fprintln(out, "            (exists)")
				if err := showConfigModels(out, configPath); err != nil {
					fmt.Fprintf(out, "            error reading config: %v\n", err)
				}
			} else {
				fmt.Fprintln(out, "            (not found)")
			}

			fmt.Fprintln(out)

			// Tools
			workspace, _ := os.Getwd()
			registry := wiring.BuildToolRegistry(workspace)
			showStatusTools(out, registry)

			// Features (from config flags — best-effort detection)
			features := detectFeatures(configPath)
			showStatusFeatures(out, features)

			fmt.Fprintln(out)
			checkGatewayHealth(out)

			return nil
		},
	}
}

func showStatusTools(out io.Writer, registry *tools.Registry) {
	count := registry.Count()
	if count == 0 {
		fmt.Fprintf(out, "Tools:      0 tools registered\n")
		return
	}

	fmt.Fprintf(out, "Tools:      %d tools registered\n", count)
	for _, t := range registry.ListTools() {
		fmt.Fprintf(out, "              - %s\n", t.Name())
	}
	fmt.Fprintln(out)
}

func showStatusFeatures(out io.Writer, features FeatureStatus) {
	mcpStatus := "disabled"
	if features.MCPEnabled {
		mcpStatus = fmt.Sprintf("enabled (%d servers)", features.MCPServers)
	}

	ragStatus := "disabled"
	if features.RAGEnabled {
		ragStatus = "enabled"
	}

	memStatus := "disabled"
	if features.MemoryEnabled {
		memStatus = "enabled"
	}

	skillsStatus := "disabled"
	if features.SkillsEnabled {
		skillsStatus = fmt.Sprintf("enabled (%d)", features.SkillsCount)
	}

	fmt.Fprintf(out, "Features:\n")
	fmt.Fprintf(out, "  MCP:        %s\n", mcpStatus)
	fmt.Fprintf(out, "  RAG:        %s\n", ragStatus)
	fmt.Fprintf(out, "  Memory:     %s\n", memStatus)
	fmt.Fprintf(out, "  Skills:     %s\n", skillsStatus)
}

func detectFeatures(configPath string) FeatureStatus {
	var features FeatureStatus

	data, err := os.ReadFile(configPath)
	if err != nil {
		return features
	}

	var cfg struct {
		MCP    interface{} `json:"mcp"`
		RAG    interface{} `json:"rag"`
		Memory interface{} `json:"memory"`
		Skills interface{} `json:"skills"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return features
	}

	features.MCPEnabled = cfg.MCP != nil
	features.RAGEnabled = cfg.RAG != nil
	features.MemoryEnabled = cfg.Memory != nil
	features.SkillsEnabled = cfg.Skills != nil

	return features
}

func showConfigModels(out io.Writer, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var cfg struct {
		Agents struct {
			Defaults struct {
				ModelName string `json:"model_name"`
			} `json:"defaults"`
		} `json:"agents"`
		ModelList []struct {
			ModelName string `json:"model_name"`
			Model     string `json:"model"`
		} `json:"model_list"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	if cfg.Agents.Defaults.ModelName != "" {
		fmt.Fprintf(out, "            default model: %s\n", cfg.Agents.Defaults.ModelName)
	}

	if len(cfg.ModelList) > 0 {
		fmt.Fprintf(out, "            configured models: %d\n", len(cfg.ModelList))
		for _, m := range cfg.ModelList {
			fmt.Fprintf(out, "              - %s (%s)\n", m.ModelName, m.Model)
		}
	}

	return nil
}

func checkGatewayHealth(out io.Writer) {
	fmt.Fprintf(out, "Gateway:    ")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:18790/api/health")
	if err != nil {
		fmt.Fprintln(out, "stopped (not reachable)")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Fprintln(out, "running (http://localhost:18790)")
	} else {
		fmt.Fprintf(out, "unhealthy (status: %d)\n", resp.StatusCode)
	}
}
