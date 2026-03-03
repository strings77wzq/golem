// unlimitedClaw - Progressive Go AI Assistant
// Learning by building, inspired by PicoClaw
// License: MIT

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version info injected at build time via ldflags
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlimitedclaw",
		Short: "unlimitedClaw - Progressive AI Assistant",
		Long:  "A progressive Go AI assistant — learning by building, inspired by PicoClaw",
		Example: `  unlimitedclaw agent
  unlimitedclaw gateway
  unlimitedclaw version`,
	}

	cmd.AddCommand(
		newVersionCommand(),
		newAgentCommand(),
		newGatewayCommand(),
	)

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("unlimitedclaw version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("date: %s\n", date)
			return nil
		},
	}
}

func newAgentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "agent",
		Short: "Start the AI agent",
		Long:  "Start the unlimitedClaw AI agent process",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Starting AI agent...")
			// TODO: Implement agent startup
			return nil
		},
	}
}

func newGatewayCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gateway",
		Short: "Start the HTTP gateway server",
		Long:  "Start the HTTP gateway server for agent communication",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Starting HTTP gateway server...")
			// TODO: Implement gateway startup
			return nil
		},
	}
}

func main() {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
