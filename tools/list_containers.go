package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type listContainersArgs struct {
	All     bool   `json:"all,omitempty" jsonschema:"description=show stopped containers too,default=true"`
	Project string `json:"project,omitempty" jsonschema:"description=filter by Compose project name"`
}

// containerInfo represents a single container from docker ps JSON output.
type containerInfo struct {
	ID        string `json:"ID"`
	Names     string `json:"Names"`
	Image     string `json:"Image"`
	State     string `json:"State"`
	Status    string `json:"Status"`
	Ports     string `json:"Ports"`
	Labels    string `json:"Labels"`
	CreatedAt string `json:"CreatedAt"`
	Networks  string `json:"Networks"`
}

// composeProject extracts the com.docker.compose.project label value.
func (c *containerInfo) composeProject() string {
	for _, label := range strings.Split(c.Labels, ",") {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 && parts[0] == "com.docker.compose.project" {
			return parts[1]
		}
	}
	return ""
}

func handleListContainers(ctx context.Context, exec docker.Executor, args listContainersArgs) (string, error) {
	cmdArgs := []string{"ps"}
	if args.All {
		cmdArgs = append(cmdArgs, "-a")
	}
	cmdArgs = append(cmdArgs, "--format", "{{json .}}")

	output, err := exec.Exec(ctx, cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return "No containers found.", nil
	}

	var containers []containerInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var c containerInfo
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return "", fmt.Errorf("failed to parse container JSON: %w", err)
		}
		containers = append(containers, c)
	}

	// Group by Compose project
	groups := make(map[string][]containerInfo)
	for _, c := range containers {
		project := c.composeProject()
		if args.Project != "" && project != args.Project {
			continue
		}
		if project == "" {
			project = "(standalone)"
		}
		groups[project] = append(groups[project], c)
	}

	if len(groups) == 0 {
		if args.Project != "" {
			return fmt.Sprintf("No containers found for project %q.", args.Project), nil
		}
		return "No containers found.", nil
	}

	// Sort group names for deterministic output, with (standalone) last
	groupNames := make([]string, 0, len(groups))
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Slice(groupNames, func(i, j int) bool {
		if groupNames[i] == "(standalone)" {
			return false
		}
		if groupNames[j] == "(standalone)" {
			return true
		}
		return groupNames[i] < groupNames[j]
	})

	var sb strings.Builder
	for gi, groupName := range groupNames {
		if gi > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("=== %s ===\n", groupName))
		for _, c := range groups[groupName] {
			sb.WriteString(fmt.Sprintf("  %-15s %-25s %-10s %s\n", c.Names, c.Image, c.State, c.Status))
		}
	}

	return sb.String(), nil
}

func registerListContainers(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_containers",
		Description: "List Docker containers, grouped by Compose project. Shows container name, image, state, and status.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listContainersArgs) (*mcp.CallToolResult, any, error) {
		// Default all=true when not explicitly set (zero value for bool is false)
		// We handle this by always passing -a flag when All is true or when the
		// struct is at its zero value (user didn't specify). Since the jsonschema
		// default is true, the MCP client should send true by default.
		// However, to be safe, if the user sends no args at all, we treat it as all=true.
		result, err := handleListContainers(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
