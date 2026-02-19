package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type getLogsArgs struct {
	Container  string `json:"container" jsonschema:"description=container name or ID,required"`
	Tail       int    `json:"tail,omitempty" jsonschema:"description=number of lines to show from the end of the logs,default=100"`
	Since      string `json:"since,omitempty" jsonschema:"description=show logs since timestamp (e.g. 2024-01-01T00:00:00) or relative (e.g. 1h)"`
	Until      string `json:"until,omitempty" jsonschema:"description=show logs until timestamp (e.g. 2024-01-01T00:00:00) or relative (e.g. 1h)"`
	Timestamps bool   `json:"timestamps,omitempty" jsonschema:"description=show timestamps in log output"`
}

func handleGetLogs(ctx context.Context, exec docker.Executor, args getLogsArgs) (string, error) {
	if args.Container == "" {
		return "", fmt.Errorf("container name or ID is required")
	}

	tail := args.Tail
	if tail <= 0 {
		tail = 100
	}

	cmdArgs := []string{"logs", "--tail", strconv.Itoa(tail)}

	if args.Since != "" {
		cmdArgs = append(cmdArgs, "--since", args.Since)
	}
	if args.Until != "" {
		cmdArgs = append(cmdArgs, "--until", args.Until)
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

	return output, nil
}

func registerGetLogs(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_logs",
		Description: "Get logs from a Docker container. Uses combined stdout and stderr output.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getLogsArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleGetLogs(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
