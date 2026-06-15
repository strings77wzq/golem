// Package main — agent execution modes extracted from main.go.
// Handles one-shot, interactive, and gateway run modes.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"

	"github.com/strings77wzq/golem/core/agent"
	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/internal/channels/tui"
)

// runAgentOneShot sends a single message and prints the response.
func runAgentOneShot(ag *agent.Agent, b bus.Bus, message string, existingSessionID string) error {
	sessionID := existingSessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	response, err := ag.HandleMessage(context.Background(), sessionID, message)
	if err != nil {
		return err
	}
	fmt.Print(response)
	fmt.Println()
	return nil
}

// runAgentInteractive runs the agent in plain interactive mode (no TUI).
func runAgentInteractive(ag *agent.Agent, b bus.Bus, existingSessionID string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go ag.Start(ctx)

	sessionID := existingSessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	outCh := b.Subscribe(agent.TopicOutbound)
	defer b.Unsubscribe(agent.TopicOutbound, outCh)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case raw := <-outCh:
				outMsg, ok := raw.(bus.OutboundMessage)
				if !ok {
					continue
				}
				if outMsg.SessionID != sessionID {
					continue
				}
				fmt.Print(outMsg.Content)
				if outMsg.Done {
					printUsage(outMsg.Usage)
					fmt.Printf("\n> ")
				}
			}
		}
	}()

	fmt.Println("Interactive mode: type messages, Ctrl+C to quit")
	fmt.Printf("> ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			fmt.Printf("> ")
			continue
		}

		b.Publish(agent.TopicInbound, bus.InboundMessage{
			SessionID: sessionID,
			Content:   line,
			Role:      bus.RoleUser,
		})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

// runAgentTUI starts the Bubble Tea TUI agent.
func runAgentTUI(ag *agent.Agent, sessionID string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go ag.Start(ctx)

	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	return tui.Run(ctx, sessionID, ag)
}

func printUsage(u *bus.TokenUsage) {
	if u == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "\n[tokens: %d prompt + %d completion = %d total]",
		u.PromptTokens, u.CompletionTokens, u.TotalTokens)
}
