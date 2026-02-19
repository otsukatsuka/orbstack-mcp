package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type composeUpArgs struct {
	Project  string   `json:"project" jsonschema:"description=Compose project name"`
	Services []string `json:"services,omitempty" jsonschema:"description=specific services to start (default: all)"`
}

type composeDownArgs struct {
	Project       string `json:"project" jsonschema:"description=Compose project name"`
	RemoveVolumes bool   `json:"remove_volumes,omitempty" jsonschema:"description=remove named volumes declared in the volumes section,default=false"`
}

// composeProjectContainer holds the minimal fields we need from docker ps JSON output
// when discovering a Compose project's working directory.
type composeProjectContainer struct {
	ID string `json:"ID"`
}

// discoverWorkDir finds the working directory of a Compose project by
// inspecting one of its containers.
func discoverWorkDir(ctx context.Context, exec docker.Executor, project string) (string, error) {
	// List any container (including stopped) belonging to the project.
	psOutput, err := exec.Exec(ctx, "ps", "-a", "--format", "{{json .}}", "--filter", "label=com.docker.compose.project="+project)
	if err != nil {
		return "", fmt.Errorf("failed to list containers for project %q: %w", project, err)
	}

	psOutput = strings.TrimSpace(psOutput)
	if psOutput == "" {
		return "", fmt.Errorf("no containers found for project %q: cannot determine working directory. Start the project manually first or specify the compose file path", project)
	}

	// Parse the first container to get its ID.
	firstLine := strings.SplitN(psOutput, "\n", 2)[0]
	var c composeProjectContainer
	if err := json.Unmarshal([]byte(firstLine), &c); err != nil {
		return "", fmt.Errorf("failed to parse container JSON: %w", err)
	}

	// Inspect to find the working directory label.
	workDir, err := exec.Exec(ctx, "inspect", "--format", `{{index .Config.Labels "com.docker.compose.project.working_dir"}}`, c.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", c.ID, err)
	}

	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return "", fmt.Errorf("container %s has no com.docker.compose.project.working_dir label", c.ID)
	}

	return workDir, nil
}

func handleComposeUp(ctx context.Context, exec docker.Executor, args composeUpArgs) (string, error) {
	workDir, err := discoverWorkDir(ctx, exec, args.Project)
	if err != nil {
		return "", err
	}

	cmdArgs := []string{"compose", "--project-directory", workDir, "-p", args.Project, "up", "-d"}
	cmdArgs = append(cmdArgs, args.Services...)

	output, err := exec.ExecCombined(ctx, cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("compose up failed: %w", err)
	}

	return fmt.Sprintf("Compose project %q started (workdir: %s)\n%s", args.Project, workDir, output), nil
}

func handleComposeDown(ctx context.Context, exec docker.Executor, args composeDownArgs) (string, error) {
	workDir, err := discoverWorkDir(ctx, exec, args.Project)
	if err != nil {
		return "", err
	}

	cmdArgs := []string{"compose", "--project-directory", workDir, "-p", args.Project, "down"}
	if args.RemoveVolumes {
		cmdArgs = append(cmdArgs, "--volumes")
	}

	output, err := exec.ExecCombined(ctx, cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("compose down failed: %w", err)
	}

	return fmt.Sprintf("Compose project %q stopped (workdir: %s)\n%s", args.Project, workDir, output), nil
}

func registerComposeUpDown(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "compose_up",
		Description: "Start a Docker Compose project. Discovers the project's working directory from existing containers and runs docker compose up -d.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args composeUpArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleComposeUp(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "compose_down",
		Description: "Stop a Docker Compose project. Discovers the project's working directory from existing containers and runs docker compose down.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args composeDownArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleComposeDown(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
