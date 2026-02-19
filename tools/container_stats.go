package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type containerStatsArgs struct {
	Container string `json:"container" jsonschema:"description=container name or ID (if empty returns all containers)"`
}

type statsEntry struct {
	Container string `json:"Container"`
	Name      string `json:"Name"`
	ID        string `json:"ID"`
	CPUPerc   string `json:"CPUPerc"`
	MemUsage  string `json:"MemUsage"`
	MemPerc   string `json:"MemPerc"`
	NetIO     string `json:"NetIO"`
	BlockIO   string `json:"BlockIO"`
	PIDs      string `json:"PIDs"`
}

func handleContainerStats(ctx context.Context, exec docker.Executor, args containerStatsArgs) (string, error) {
	cmdArgs := []string{"stats", "--no-stream", "--format", "{{json .}}"}
	if args.Container != "" {
		cmdArgs = append(cmdArgs, args.Container)
	}

	out, err := exec.Exec(ctx, cmdArgs...)
	if err != nil {
		return "", err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return "No running containers found.", nil
	}

	lines := strings.Split(out, "\n")
	var entries []statsEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e statsEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return "", fmt.Errorf("failed to parse stats JSON: %w", err)
		}
		entries = append(entries, e)
	}

	if len(entries) == 0 {
		return "No running containers found.", nil
	}

	return formatStatsTable(entries), nil
}

func formatStatsTable(entries []statsEntry) string {
	header := fmt.Sprintf("%-20s %-10s %-25s %-10s %-25s %-25s %-6s",
		"CONTAINER", "CPU %", "MEM USAGE", "MEM %", "NET I/O", "BLOCK I/O", "PIDS")
	sep := strings.Repeat("-", len(header))

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n")

	for _, e := range entries {
		name := e.Name
		if name == "" {
			name = e.Container
		}
		b.WriteString(fmt.Sprintf("%-20s %-10s %-25s %-10s %-25s %-25s %-6s\n",
			name, e.CPUPerc, e.MemUsage, e.MemPerc, e.NetIO, e.BlockIO, e.PIDs))
	}

	return b.String()
}

func registerContainerStats(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "container_stats",
		Description: "Get resource usage statistics for containers (CPU, memory, network, block I/O, PIDs). Optionally specify a container name/ID or leave empty for all running containers.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args containerStatsArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleContainerStats(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
