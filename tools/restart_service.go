package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type restartServiceArgs struct {
	Container string `json:"container" jsonschema:"description=Container name or ID to restart"`
	Timeout   int    `json:"timeout,omitempty" jsonschema:"description=Seconds to wait before killing the container (default 10)"`
}

func handleRestartService(ctx context.Context, exec docker.Executor, args restartServiceArgs) (string, error) {
	timeout := args.Timeout
	if timeout <= 0 {
		timeout = 10
	}

	dockerArgs := []string{"restart", "--time", fmt.Sprintf("%d", timeout), args.Container}

	_, err := exec.Exec(ctx, dockerArgs...)
	if err != nil {
		return "", fmt.Errorf("restart failed: %w", err)
	}

	return fmt.Sprintf("Successfully restarted container %s", args.Container), nil
}

func registerRestartService(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "restart_service",
		Description: "Restart a container or all containers in a Compose service.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args restartServiceArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleRestartService(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
