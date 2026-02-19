package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type logDiffArgs struct {
	Container    string `json:"container" jsonschema:"description=container name or ID"`
	Period1Start string `json:"period1_start" jsonschema:"description=start of period 1 (RFC3339 or relative e.g. 2h)"`
	Period1End   string `json:"period1_end" jsonschema:"description=end of period 1 (RFC3339 or relative e.g. 1h)"`
	Period2Start string `json:"period2_start" jsonschema:"description=start of period 2 (RFC3339 or relative e.g. 1h)"`
	Period2End   string `json:"period2_end" jsonschema:"description=end of period 2 (RFC3339 or relative e.g. now)"`
}

func fetchLogs(ctx context.Context, exec docker.Executor, container, since, until string) (string, error) {
	return exec.ExecCombined(ctx, "logs", "--since", since, "--until", until, container)
}

func countLines(text string) map[string]int {
	counts := make(map[string]int)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		counts[line]++
	}
	return counts
}

func handleLogDiff(ctx context.Context, exec docker.Executor, args logDiffArgs) (string, error) {
	logs1, err := fetchLogs(ctx, exec, args.Container, args.Period1Start, args.Period1End)
	if err != nil {
		return "", fmt.Errorf("failed to fetch period 1 logs: %w", err)
	}

	logs2, err := fetchLogs(ctx, exec, args.Container, args.Period2Start, args.Period2End)
	if err != nil {
		return "", fmt.Errorf("failed to fetch period 2 logs: %w", err)
	}

	counts1 := countLines(logs1)
	counts2 := countLines(logs2)

	// Classify lines
	var onlyIn1 []string
	var onlyIn2 []string
	var changed []string
	var common []string

	// Collect all unique lines
	allLines := make(map[string]struct{})
	for line := range counts1 {
		allLines[line] = struct{}{}
	}
	for line := range counts2 {
		allLines[line] = struct{}{}
	}

	for line := range allLines {
		c1 := counts1[line]
		c2 := counts2[line]
		switch {
		case c1 > 0 && c2 == 0:
			onlyIn1 = append(onlyIn1, line)
		case c1 == 0 && c2 > 0:
			onlyIn2 = append(onlyIn2, line)
		case c1 != c2:
			changed = append(changed, fmt.Sprintf("  %s (period1: %dx, period2: %dx)", line, c1, c2))
		default:
			common = append(common, line)
		}
	}

	sort.Strings(onlyIn1)
	sort.Strings(onlyIn2)
	sort.Strings(changed)
	sort.Strings(common)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Log Diff: %s ===\n", args.Container))
	sb.WriteString(fmt.Sprintf("Period 1: %s to %s\n", args.Period1Start, args.Period1End))
	sb.WriteString(fmt.Sprintf("Period 2: %s to %s\n\n", args.Period2Start, args.Period2End))

	sb.WriteString("--- Only in Period 1 ---\n")
	if len(onlyIn1) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, line := range onlyIn1 {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	sb.WriteString("\n--- Only in Period 2 ---\n")
	if len(onlyIn2) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, line := range onlyIn2 {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	sb.WriteString("\n--- Count Changes ---\n")
	if len(changed) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, line := range changed {
			sb.WriteString(line + "\n")
		}
	}

	sb.WriteString("\n--- Common (unchanged) ---\n")
	if len(common) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, line := range common {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	return sb.String(), nil
}

func registerLogDiff(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "log_diff",
		Description: "Compare container logs between two time periods. Useful for debugging regressions by identifying what changed in log output.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args logDiffArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleLogDiff(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
