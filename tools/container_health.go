package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type containerHealthArgs struct {
	Container string `json:"container" jsonschema:"required,description=container name or ID to check health for"`
}

type healthLog struct {
	Start    string `json:"Start"`
	End      string `json:"End"`
	ExitCode int    `json:"ExitCode"`
	Output   string `json:"Output"`
}

type healthState struct {
	Status        string      `json:"Status"`
	FailingStreak int         `json:"FailingStreak"`
	Log           []healthLog `json:"Log"`
}

type healthConfig struct {
	Test     []string `json:"Test"`
	Interval int64    `json:"Interval"`
	Timeout  int64    `json:"Timeout"`
	Retries  int      `json:"Retries"`
}

func handleContainerHealth(ctx context.Context, exec docker.Executor, args containerHealthArgs) (string, error) {
	if args.Container == "" {
		return "", fmt.Errorf("container name or ID is required")
	}

	// Get health state
	healthOut, err := exec.Exec(ctx, "inspect", "--format", "{{json .State.Health}}", args.Container)
	if err != nil {
		return "", err
	}

	// Get health config
	configOut, err := exec.Exec(ctx, "inspect", "--format", "{{json .Config.Healthcheck}}", args.Container)
	if err != nil {
		return "", err
	}

	healthOut = strings.TrimSpace(healthOut)
	configOut = strings.TrimSpace(configOut)

	// Check if no healthcheck is configured
	if (healthOut == "" || healthOut == "null" || healthOut == "<nil>" || healthOut == "<no value>") &&
		(configOut == "" || configOut == "null" || configOut == "<nil>" || configOut == "<no value>") {
		return "No healthcheck configured for this container.", nil
	}

	var b strings.Builder

	// Parse and display health config
	if configOut != "" && configOut != "null" && configOut != "<nil>" && configOut != "<no value>" {
		var cfg healthConfig
		if err := json.Unmarshal([]byte(configOut), &cfg); err == nil {
			b.WriteString("Health Check Configuration:\n")
			if len(cfg.Test) > 0 {
				b.WriteString(fmt.Sprintf("  Test:     %s\n", strings.Join(cfg.Test, " ")))
			}
			if cfg.Interval > 0 {
				b.WriteString(fmt.Sprintf("  Interval: %s\n", formatNanoseconds(cfg.Interval)))
			}
			if cfg.Timeout > 0 {
				b.WriteString(fmt.Sprintf("  Timeout:  %s\n", formatNanoseconds(cfg.Timeout)))
			}
			if cfg.Retries > 0 {
				b.WriteString(fmt.Sprintf("  Retries:  %d\n", cfg.Retries))
			}
			b.WriteString("\n")
		}
	}

	// Parse and display health state
	if healthOut != "" && healthOut != "null" && healthOut != "<nil>" && healthOut != "<no value>" {
		var state healthState
		if err := json.Unmarshal([]byte(healthOut), &state); err == nil {
			b.WriteString(fmt.Sprintf("Current Status: %s\n", state.Status))
			b.WriteString(fmt.Sprintf("Failing Streak: %d\n", state.FailingStreak))

			if len(state.Log) > 0 {
				b.WriteString(fmt.Sprintf("\nRecent Health Check Results (%d entries):\n", len(state.Log)))
				for i, entry := range state.Log {
					b.WriteString(fmt.Sprintf("  [%d] Exit Code: %d\n", i+1, entry.ExitCode))
					b.WriteString(fmt.Sprintf("      Start:     %s\n", entry.Start))
					b.WriteString(fmt.Sprintf("      End:       %s\n", entry.End))
					if entry.Output != "" {
						b.WriteString(fmt.Sprintf("      Output:    %s\n", strings.TrimSpace(entry.Output)))
					}
				}
			}
		}
	}

	result := b.String()
	if result == "" {
		return "No healthcheck configured for this container.", nil
	}

	return result, nil
}

// formatNanoseconds converts nanoseconds to a human-readable duration string.
func formatNanoseconds(ns int64) string {
	seconds := ns / 1_000_000_000
	if seconds >= 60 {
		minutes := seconds / 60
		remainSec := seconds % 60
		if remainSec == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%ds", minutes, remainSec)
	}
	return fmt.Sprintf("%ds", seconds)
}

func registerContainerHealth(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "container_health",
		Description: "Get health check configuration and status for a container, including recent check results and failing streak.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args containerHealthArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleContainerHealth(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
