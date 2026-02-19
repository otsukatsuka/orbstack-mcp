package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type containerExecArgs struct {
	Container string `json:"container" jsonschema:"description=Container name or ID"`
	Command   string `json:"command" jsonschema:"description=Command to execute inside the container (supports pipes and redirects via sh -c)"`
	User      string `json:"user,omitempty" jsonschema:"description=Run command as a specific user"`
	Workdir   string `json:"workdir,omitempty" jsonschema:"description=Working directory inside the container"`
}

func handleContainerExec(ctx context.Context, exec docker.Executor, args containerExecArgs) (string, error) {
	dockerArgs := []string{"exec"}

	if args.User != "" {
		dockerArgs = append(dockerArgs, "--user", args.User)
	}
	if args.Workdir != "" {
		dockerArgs = append(dockerArgs, "--workdir", args.Workdir)
	}

	dockerArgs = append(dockerArgs, args.Container, "sh", "-c", args.Command)

	output, err := exec.ExecCombined(ctx, dockerArgs...)
	if err != nil {
		return "", fmt.Errorf("exec failed: %w", err)
	}

	return output, nil
}

func registerContainerExec(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "container_exec",
		Description: "Execute a command inside a running container. The command is run via sh -c, so pipes and redirects are supported.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args containerExecArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleContainerExec(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
