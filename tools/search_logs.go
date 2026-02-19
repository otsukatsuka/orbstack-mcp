package tools

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type searchLogsArgs struct {
	Container    string `json:"container" jsonschema:"description=container name or ID,required"`
	Pattern      string `json:"pattern" jsonschema:"description=regex pattern to search for in logs,required"`
	Tail         int    `json:"tail,omitempty" jsonschema:"description=number of log lines to fetch before filtering,default=1000"`
	Since        string `json:"since,omitempty" jsonschema:"description=show logs since timestamp (e.g. 2024-01-01T00:00:00) or relative (e.g. 1h)"`
	Timestamps   bool   `json:"timestamps,omitempty" jsonschema:"description=show timestamps in log output"`
	ContextLines int    `json:"context_lines,omitempty" jsonschema:"description=number of lines of context around each match (like grep -C),default=0"`
}

func handleSearchLogs(ctx context.Context, exec docker.Executor, args searchLogsArgs) (string, error) {
	if args.Container == "" {
		return "", fmt.Errorf("container name or ID is required")
	}
	if args.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	re, err := regexp.Compile(args.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern %q: %w", args.Pattern, err)
	}

	tail := args.Tail
	if tail <= 0 {
		tail = 1000
	}

	cmdArgs := []string{"logs", "--tail", strconv.Itoa(tail)}

	if args.Since != "" {
		cmdArgs = append(cmdArgs, "--since", args.Since)
	}
	if args.Timestamps {
		cmdArgs = append(cmdArgs, "--timestamps")
	}

	cmdArgs = append(cmdArgs, args.Container)

	output, err := exec.ExecCombined(ctx, cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for container %q: %w", args.Container, err)
	}

	if output == "" {
		return "No log output.", nil
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Find matching line indices
	var matchIndices []int
	for i, line := range lines {
		if re.MatchString(line) {
			matchIndices = append(matchIndices, i)
		}
	}

	if len(matchIndices) == 0 {
		return fmt.Sprintf("No matches found for pattern %q in %d log lines.", args.Pattern, len(lines)), nil
	}

	var result string
	if args.ContextLines > 0 {
		result = formatWithContext(lines, matchIndices, args.ContextLines)
	} else {
		var matchedLines []string
		for _, idx := range matchIndices {
			matchedLines = append(matchedLines, lines[idx])
		}
		result = strings.Join(matchedLines, "\n")
	}

	header := fmt.Sprintf("Found %d matches for pattern %q:\n\n", len(matchIndices), args.Pattern)
	return header + result, nil
}

// formatWithContext formats matched lines with surrounding context lines,
// similar to grep -C behavior. Overlapping context regions are merged.
func formatWithContext(lines []string, matchIndices []int, contextLines int) string {
	// Build a set of line ranges to include
	type lineRange struct {
		start int
		end   int // inclusive
	}

	var ranges []lineRange
	for _, idx := range matchIndices {
		start := idx - contextLines
		if start < 0 {
			start = 0
		}
		end := idx + contextLines
		if end >= len(lines) {
			end = len(lines) - 1
		}
		ranges = append(ranges, lineRange{start, end})
	}

	// Merge overlapping ranges
	merged := []lineRange{ranges[0]}
	for i := 1; i < len(ranges); i++ {
		last := &merged[len(merged)-1]
		if ranges[i].start <= last.end+1 {
			// Overlapping or adjacent, extend
			if ranges[i].end > last.end {
				last.end = ranges[i].end
			}
		} else {
			merged = append(merged, ranges[i])
		}
	}

	// Build a set of match indices for quick lookup
	matchSet := make(map[int]bool, len(matchIndices))
	for _, idx := range matchIndices {
		matchSet[idx] = true
	}

	var sb strings.Builder
	for ri, r := range merged {
		if ri > 0 {
			sb.WriteString("--\n")
		}
		for i := r.start; i <= r.end; i++ {
			if matchSet[i] {
				sb.WriteString("> " + lines[i] + "\n")
			} else {
				sb.WriteString("  " + lines[i] + "\n")
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

func registerSearchLogs(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_logs",
		Description: "Search Docker container logs using a regex pattern. Fetches logs then filters matching lines, with optional context lines around matches.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args searchLogsArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleSearchLogs(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
