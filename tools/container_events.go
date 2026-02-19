package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type containerEventsArgs struct {
	Container string `json:"container,omitempty" jsonschema:"description=filter events by container name or ID"`
	Since     string `json:"since,omitempty" jsonschema:"description=show events since this time (default: 1h),default=1h"`
	Until     string `json:"until,omitempty" jsonschema:"description=show events until this time (default: now),default=now"`
	EventType string `json:"event_type,omitempty" jsonschema:"description=filter by event type: start stop die restart oom kill pause unpause etc."`
}

// dockerEvent represents the JSON output of docker events.
type dockerEvent struct {
	Status   string            `json:"status"`
	Action   string            `json:"Action"`
	Type     string            `json:"Type"`
	Actor    dockerEventActor  `json:"Actor"`
	Time     int64             `json:"time"`
	TimeNano string            `json:"timeNano"`
}

type dockerEventActor struct {
	ID         string            `json:"ID"`
	Attributes map[string]string `json:"Attributes"`
}

func handleContainerEvents(ctx context.Context, exec docker.Executor, args containerEventsArgs) (string, error) {
	since := args.Since
	if since == "" {
		since = "1h"
	}
	until := args.Until
	if until == "" {
		until = "now"
	}

	cmdArgs := []string{"events", "--filter", "type=container"}
	if args.Container != "" {
		cmdArgs = append(cmdArgs, "--filter", "container="+args.Container)
	}
	if args.EventType != "" {
		cmdArgs = append(cmdArgs, "--filter", "event="+args.EventType)
	}
	cmdArgs = append(cmdArgs, "--since", since, "--until", until, "--format", "{{json .}}")

	output, err := exec.Exec(ctx, cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to get events: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return "No events found in the specified time range.", nil
	}

	var sb strings.Builder
	sb.WriteString("=== Container Events ===\n")
	if args.Container != "" {
		sb.WriteString(fmt.Sprintf("Container: %s\n", args.Container))
	}
	sb.WriteString(fmt.Sprintf("Time range: %s to %s\n", since, until))
	if args.EventType != "" {
		sb.WriteString(fmt.Sprintf("Event filter: %s\n", args.EventType))
	}
	sb.WriteString("\n")

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event dockerEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// If parsing fails, include the raw line.
			sb.WriteString(fmt.Sprintf("  (unparsed) %s\n", line))
			continue
		}

		name := event.Actor.Attributes["name"]
		if name == "" {
			name = event.Actor.ID
			if len(name) > 12 {
				name = name[:12]
			}
		}

		action := event.Action
		if action == "" {
			action = event.Status
		}

		// Build attributes summary (skip name since we already show it).
		var attrs []string
		for k, v := range event.Actor.Attributes {
			if k == "name" {
				continue
			}
			attrs = append(attrs, fmt.Sprintf("%s=%s", k, v))
		}

		attrStr := ""
		if len(attrs) > 0 {
			attrStr = fmt.Sprintf(" (%s)", strings.Join(attrs, ", "))
		}

		sb.WriteString(fmt.Sprintf("  [%d] %s: %s%s\n", event.Time, name, action, attrStr))
	}

	return sb.String(), nil
}

func registerContainerEvents(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "container_events",
		Description: "Get container event history (start/stop/die/restart/OOM etc). Always uses --until to prevent streaming forever.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args containerEventsArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleContainerEvents(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
