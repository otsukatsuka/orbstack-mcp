package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type composeLogsArgs struct {
	Project    string `json:"project" jsonschema:"description=Compose project name"`
	Tail       int    `json:"tail,omitempty" jsonschema:"description=Number of lines to show from the end of the logs (default 100)"`
	Since      string `json:"since,omitempty" jsonschema:"description=Show logs since timestamp (e.g. 2021-01-01T00:00:00Z) or relative (e.g. 42m for 42 minutes)"`
	Timestamps bool   `json:"timestamps,omitempty" jsonschema:"description=Show timestamps in log output"`
}

type composeLogContainer struct {
	ID     string            `json:"ID"`
	Names  string            `json:"Names"`
	Labels map[string]string `json:"Labels"`
}

// parseComposeLogContainers parses docker ps JSON output into composeLogContainer structs.
// Docker ps --format '{{json .}}' outputs one JSON object per line, but the Labels
// field is a comma-separated key=value string, not a JSON map.
func parseComposeLogContainers(output string) ([]composeLogContainer, error) {
	var containers []composeLogContainer
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Docker outputs Labels as a comma-separated string like "key1=val1,key2=val2".
		// We first unmarshal into a raw struct to handle that.
		var raw struct {
			ID     string `json:"ID"`
			Names  string `json:"Names"`
			Labels string `json:"Labels"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("failed to parse container JSON: %w", err)
		}
		labels := make(map[string]string)
		if raw.Labels != "" {
			for _, pair := range strings.Split(raw.Labels, ",") {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					labels[kv[0]] = kv[1]
				}
			}
		}
		containers = append(containers, composeLogContainer{
			ID:     raw.ID,
			Names:  raw.Names,
			Labels: labels,
		})
	}
	return containers, nil
}

func handleComposeLogs(ctx context.Context, exec docker.Executor, args composeLogsArgs) (string, error) {
	// Find containers belonging to the Compose project.
	psOutput, err := exec.Exec(ctx, "ps", "-a", "--format", "{{json .}}", "--filter", "label=com.docker.compose.project="+args.Project)
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	if strings.TrimSpace(psOutput) == "" {
		return "", fmt.Errorf("no containers found for Compose project %q", args.Project)
	}

	containers, err := parseComposeLogContainers(psOutput)
	if err != nil {
		return "", err
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("no containers found for Compose project %q", args.Project)
	}

	tail := args.Tail
	if tail <= 0 {
		tail = 100
	}

	// Collect logs from each container.
	var result strings.Builder
	for _, c := range containers {
		serviceName := c.Labels["com.docker.compose.service"]
		if serviceName == "" {
			serviceName = c.Names
		}

		logArgs := []string{"logs", "--tail", fmt.Sprintf("%d", tail)}
		if args.Since != "" {
			logArgs = append(logArgs, "--since", args.Since)
		}
		if args.Timestamps {
			logArgs = append(logArgs, "--timestamps")
		}
		logArgs = append(logArgs, c.ID)

		logOutput, err := exec.ExecCombined(ctx, logArgs...)
		if err != nil {
			result.WriteString(fmt.Sprintf("[%s] error fetching logs: %s\n", serviceName, err))
			continue
		}

		for _, line := range strings.Split(logOutput, "\n") {
			if line == "" {
				continue
			}
			result.WriteString(fmt.Sprintf("[%s] %s\n", serviceName, line))
		}
	}

	return result.String(), nil
}

func registerComposeLogs(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "compose_logs",
		Description: "Get logs for all containers in a Docker Compose project, merged and prefixed with service names.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args composeLogsArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleComposeLogs(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
